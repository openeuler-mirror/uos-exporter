package metrics

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"openvpn_exporter/config"
	"openvpn_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// RegisterOpenVPNCollector 注册OpenVPN收集器
func RegisterOpenVPNCollector() {
	collector := NewOpenVPNCollector()
	exporter.Register(collector)
	logrus.Info("OpenVPN collector registered successfully")
}

// OpenVPNServerHeader 定义服务器指标头
type OpenVPNServerHeader struct {
	LabelColumns []string
	Metrics      []OpenVPNServerHeaderField
}

// OpenVPNServerHeaderField 定义服务器指标字段
type OpenVPNServerHeaderField struct {
	Column    string
	Desc      *prometheus.Desc
	ValueType prometheus.ValueType
}

// OpenVPNCollector 收集OpenVPN指标
type OpenVPNCollector struct {
	statusPaths                 []string
	ignoreIndividuals           bool
	openvpnUpDesc               *prometheus.Desc
	openvpnStatusUpdateTimeDesc *prometheus.Desc
	openvpnConnectedClientsDesc *prometheus.Desc
	openvpnClientDescs          map[string]*prometheus.Desc
	openvpnServerHeaders        map[string]OpenVPNServerHeader
}

// NewOpenVPNCollector 创建OpenVPN收集器
func NewOpenVPNCollector() *OpenVPNCollector {
	// 从配置中获取状态文件路径和选项
	statusPaths := config.GetStatusPaths()
	ignoreIndividuals := config.GetIgnoreIndividuals()

	logrus.Infof("OpenVPN status paths: %v", statusPaths)
	logrus.Infof("Ignore individuals: %v", ignoreIndividuals)

	// 服务器和客户端通用指标
	openvpnUpDesc := prometheus.NewDesc(
		prometheus.BuildFQName("openvpn", "", "up"),
		"Whether scraping OpenVPN's metrics was successful.",
		[]string{"status_path"}, nil)
	openvpnStatusUpdateTimeDesc := prometheus.NewDesc(
		prometheus.BuildFQName("openvpn", "", "status_update_time_seconds"),
		"UNIX timestamp at which the OpenVPN statistics were updated.",
		[]string{"status_path"}, nil)

	// 服务器特有指标
	openvpnConnectedClientsDesc := prometheus.NewDesc(
		prometheus.BuildFQName("openvpn", "", "server_connected_clients"),
		"Number Of Connected Clients",
		[]string{"status_path"}, nil)

	// 客户端特有指标
	openvpnClientDescs := map[string]*prometheus.Desc{
		"TUN/TAP read bytes": prometheus.NewDesc(
			prometheus.BuildFQName("openvpn", "client", "tun_tap_read_bytes_total"),
			"Total amount of TUN/TAP traffic read, in bytes.",
			[]string{"status_path"}, nil),
		"TUN/TAP write bytes": prometheus.NewDesc(
			prometheus.BuildFQName("openvpn", "client", "tun_tap_write_bytes_total"),
			"Total amount of TUN/TAP traffic written, in bytes.",
			[]string{"status_path"}, nil),
		"TCP/UDP read bytes": prometheus.NewDesc(
			prometheus.BuildFQName("openvpn", "client", "tcp_udp_read_bytes_total"),
			"Total amount of TCP/UDP traffic read, in bytes.",
			[]string{"status_path"}, nil),
		"TCP/UDP write bytes": prometheus.NewDesc(
			prometheus.BuildFQName("openvpn", "client", "tcp_udp_write_bytes_total"),
			"Total amount of TCP/UDP traffic written, in bytes.",
			[]string{"status_path"}, nil),
		"Auth read bytes": prometheus.NewDesc(
			prometheus.BuildFQName("openvpn", "client", "auth_read_bytes_total"),
			"Total amount of authentication traffic read, in bytes.",
			[]string{"status_path"}, nil),
		"pre-compress bytes": prometheus.NewDesc(
			prometheus.BuildFQName("openvpn", "client", "pre_compress_bytes_total"),
			"Total amount of data before compression, in bytes.",
			[]string{"status_path"}, nil),
		"post-compress bytes": prometheus.NewDesc(
			prometheus.BuildFQName("openvpn", "client", "post_compress_bytes_total"),
			"Total amount of data after compression, in bytes.",
			[]string{"status_path"}, nil),
		"pre-decompress bytes": prometheus.NewDesc(
			prometheus.BuildFQName("openvpn", "client", "pre_decompress_bytes_total"),
			"Total amount of data before decompression, in bytes.",
			[]string{"status_path"}, nil),
		"post-decompress bytes": prometheus.NewDesc(
			prometheus.BuildFQName("openvpn", "client", "post_decompress_bytes_total"),
			"Total amount of data after decompression, in bytes.",
			[]string{"status_path"}, nil),
	}

	var serverHeaderClientLabels []string
	var serverHeaderClientLabelColumns []string
	var serverHeaderRoutingLabels []string
	var serverHeaderRoutingLabelColumns []string
	if ignoreIndividuals {
		serverHeaderClientLabels = []string{"status_path", "common_name"}
		serverHeaderClientLabelColumns = []string{"Common Name"}
		serverHeaderRoutingLabels = []string{"status_path", "common_name"}
		serverHeaderRoutingLabelColumns = []string{"Common Name"}
	} else {
		serverHeaderClientLabels = []string{"status_path", "common_name", "connection_time", "real_address", "virtual_address", "username"}
		serverHeaderClientLabelColumns = []string{"Common Name", "Connected Since (time_t)", "Real Address", "Virtual Address", "Username"}
		serverHeaderRoutingLabels = []string{"status_path", "common_name", "real_address", "virtual_address"}
		serverHeaderRoutingLabelColumns = []string{"Common Name", "Real Address", "Virtual Address"}
	}

	openvpnServerHeaders := map[string]OpenVPNServerHeader{
		"CLIENT_LIST": {
			LabelColumns: serverHeaderClientLabelColumns,
			Metrics: []OpenVPNServerHeaderField{
				{
					Column: "Bytes Received",
					Desc: prometheus.NewDesc(
						prometheus.BuildFQName("openvpn", "server", "client_received_bytes_total"),
						"Amount of data received over a connection on the VPN server, in bytes.",
						serverHeaderClientLabels, nil),
					ValueType: prometheus.CounterValue,
				},
				{
					Column: "Bytes Sent",
					Desc: prometheus.NewDesc(
						prometheus.BuildFQName("openvpn", "server", "client_sent_bytes_total"),
						"Amount of data sent over a connection on the VPN server, in bytes.",
						serverHeaderClientLabels, nil),
					ValueType: prometheus.CounterValue,
				},
			},
		},
		"ROUTING_TABLE": {
			LabelColumns: serverHeaderRoutingLabelColumns,
			Metrics: []OpenVPNServerHeaderField{
				{
					Column: "Last Ref (time_t)",
					Desc: prometheus.NewDesc(
						prometheus.BuildFQName("openvpn", "server", "route_last_reference_time_seconds"),
						"Time at which a route was last referenced, in seconds.",
						serverHeaderRoutingLabels, nil),
					ValueType: prometheus.GaugeValue,
				},
			},
		},
	}

	return &OpenVPNCollector{
		statusPaths:                 statusPaths,
		ignoreIndividuals:           ignoreIndividuals,
		openvpnUpDesc:               openvpnUpDesc,
		openvpnStatusUpdateTimeDesc: openvpnStatusUpdateTimeDesc,
		openvpnConnectedClientsDesc: openvpnConnectedClientsDesc,
		openvpnClientDescs:          openvpnClientDescs,
		openvpnServerHeaders:        openvpnServerHeaders,
	}
}

// Describe 实现Collector接口，描述OpenVPN指标
func (e *OpenVPNCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.openvpnUpDesc
	ch <- e.openvpnStatusUpdateTimeDesc
	ch <- e.openvpnConnectedClientsDesc

	// 描述客户端指标
	for _, desc := range e.openvpnClientDescs {
		ch <- desc
	}

	// 描述服务器指标
	for _, header := range e.openvpnServerHeaders {
		for _, metric := range header.Metrics {
			ch <- metric.Desc
		}
	}
}

// Collect 实现Collector接口，收集OpenVPN指标
func (e *OpenVPNCollector) Collect(ch chan<- prometheus.Metric) {
	for _, statusPath := range e.statusPaths {
		err := e.collectStatusFromFile(statusPath, ch)
		if err == nil {
			ch <- prometheus.MustNewConstMetric(
				e.openvpnUpDesc,
				prometheus.GaugeValue,
				1.0,
				statusPath)
		} else {
			logrus.Errorf("Failed to scrape OpenVPN status: %s", err)
			ch <- prometheus.MustNewConstMetric(
				e.openvpnUpDesc,
				prometheus.GaugeValue,
				0.0,
				statusPath)
		}
	}
}

// collectStatusFromFile 从文件中收集状态信息
func (e *OpenVPNCollector) collectStatusFromFile(statusPath string, ch chan<- prometheus.Metric) error {
	cleanStatusPath := filepath.Clean(statusPath)
	conn, err := os.Open(cleanStatusPath)
	if err != nil {
		return err
	}
	defer conn.Close()
	return e.collectStatusFromReader(statusPath, conn, ch)
}

// collectStatusFromReader 从Reader中收集状态信息
func (e *OpenVPNCollector) collectStatusFromReader(statusPath string, file io.Reader, ch chan<- prometheus.Metric) error {
	reader := bufio.NewReader(file)
	buf, err := reader.Peek(18)
	if err != nil {
		return fmt.Errorf("error peeking at OpenVPN status file: %v", err)
	}

	if bytes.HasPrefix(buf, []byte("TITLE,")) {
		// 服务器统计，使用格式版本2
		return e.collectServerStatusFromReader(statusPath, reader, ch, ",")
	} else if bytes.HasPrefix(buf, []byte("TITLE\t")) {
		// 服务器统计，使用格式版本3
		return e.collectServerStatusFromReader(statusPath, reader, ch, "\t")
	} else if bytes.HasPrefix(buf, []byte("OpenVPN STATISTICS")) {
		// 客户端统计
		return e.collectClientStatusFromReader(statusPath, reader, ch)
	} else {
		return fmt.Errorf("unexpected file contents: %q", buf)
	}
}

// collectServerStatusFromReader 从Reader中收集服务器状态信息
func (e *OpenVPNCollector) collectServerStatusFromReader(statusPath string, file io.Reader, ch chan<- prometheus.Metric, separator string) error {
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	headersFound := map[string][]string{}
	// 已连接客户端计数器
	numberConnectedClient := 0

	recordedMetrics := map[OpenVPNServerHeaderField][]string{}

	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), separator)
		if len(fields) == 0 {
			continue
		}

		if fields[0] == "END" && len(fields) == 1 {
			// 统计页脚
		} else if fields[0] == "GLOBAL_STATS" {
			// 全局服务器统计
		} else if fields[0] == "HEADER" && len(fields) > 2 {
			// CLIENT_LIST 和 ROUTING_TABLE 的列名
			headersFound[fields[1]] = fields[2:]
		} else if fields[0] == "TIME" && len(fields) == 3 {
			// 统计更新时间
			timeStartStats, err := strconv.ParseFloat(fields[2], 64)
			if err != nil {
				return fmt.Errorf("error parsing TIME field: %v", err)
			}
			ch <- prometheus.MustNewConstMetric(
				e.openvpnStatusUpdateTimeDesc,
				prometheus.GaugeValue,
				timeStartStats,
				statusPath)
		} else if fields[0] == "TITLE" && len(fields) == 2 {
			// OpenVPN 版本号
		} else if header, ok := e.openvpnServerHeaders[fields[0]]; ok {
			if fields[0] == "CLIENT_LIST" {
				numberConnectedClient++
			}
			// 依赖于前面的 HEADERS 指令的条目
			columnNames, ok := headersFound[fields[0]]
			if !ok {
				return fmt.Errorf("%s should be preceded by HEADERS", fields[0])
			}
			if len(fields) != len(columnNames)+1 {
				return fmt.Errorf("HEADER for %s describes a different number of columns", fields[0])
			}

			// 将条目值存储在按列名索引的映射中
			columnValues := map[string]string{}
			for _, column := range header.LabelColumns {
				columnValues[column] = ""
			}
			for i, column := range columnNames {
				if i+1 < len(fields) {
					columnValues[column] = fields[i+1]
				}
			}

			// 提取应作为条目标签的列
			labels := []string{statusPath}
			for _, column := range header.LabelColumns {
				labels = append(labels, columnValues[column])
			}

			// 将相关列导出为单独的指标
			for _, metric := range header.Metrics {
				if columnValue, ok := columnValues[metric.Column]; ok {
					if l, _ := recordedMetrics[metric]; !e.subslice(labels, l) {
						value, err := strconv.ParseFloat(columnValue, 64)
						if err != nil {
							return fmt.Errorf("error parsing metric value: %v", err)
						}
						ch <- prometheus.MustNewConstMetric(
							metric.Desc,
							metric.ValueType,
							value,
							labels...)
						recordedMetrics[metric] = append(recordedMetrics[metric], labels...)
					} else {
						logrus.Warnf("Metric entry with same labels: %s, %s", metric.Column, labels)
					}
				}
			}
		} else if len(fields) > 0 && fields[0] != "" {
			return fmt.Errorf("unsupported key: %q", fields[0])
		}
	}
	// 添加已连接的客户端数量
	ch <- prometheus.MustNewConstMetric(
		e.openvpnConnectedClientsDesc,
		prometheus.GaugeValue,
		float64(numberConnectedClient),
		statusPath)

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error scanning OpenVPN status file: %v", err)
	}

	return nil
}

// collectClientStatusFromReader 从Reader中收集客户端状态信息
func (e *OpenVPNCollector) collectClientStatusFromReader(statusPath string, file io.Reader, ch chan<- prometheus.Metric) error {
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), ",")
		if len(fields) == 0 {
			continue
		}

		if fields[0] == "END" && len(fields) == 1 {
			// 统计页脚
		} else if fields[0] == "OpenVPN STATISTICS" && len(fields) == 1 {
			// 统计标头
		} else if fields[0] == "Updated" && len(fields) == 2 {
			// 统计更新时间
			location, _ := time.LoadLocation("Local")
			timeParser, err := time.ParseInLocation("Mon Jan 2 15:04:05 2006", fields[1], location)
			if err != nil {
				return fmt.Errorf("error parsing Updated field: %v", err)
			}
			ch <- prometheus.MustNewConstMetric(
				e.openvpnStatusUpdateTimeDesc,
				prometheus.GaugeValue,
				float64(timeParser.Unix()),
				statusPath)
		} else if desc, ok := e.openvpnClientDescs[fields[0]]; ok && len(fields) == 2 {
			// 流量计数器
			value, err := strconv.ParseFloat(fields[1], 64)
			if err != nil {
				return fmt.Errorf("error parsing client metric value: %v", err)
			}
			ch <- prometheus.MustNewConstMetric(
				desc,
				prometheus.CounterValue,
				value,
				statusPath)
		} else if len(fields) > 0 && fields[0] != "" {
			return fmt.Errorf("unsupported key: %q", fields[0])
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error scanning OpenVPN client status file: %v", err)
	}

	return nil
}

// contains 判断slice是否包含字符串
func (e *OpenVPNCollector) contains(s []string, item string) bool {
	for _, element := range s {
		if element == item {
			return true
		}
	}
	return false
}

// subslice 判断是否是子slice
func (e *OpenVPNCollector) subslice(sub []string, main []string) bool {
	if len(sub) > len(main) {
		return false
	}
	for _, s := range sub {
		if !e.contains(main, s) {
			return false
		}
	}
	return true
}

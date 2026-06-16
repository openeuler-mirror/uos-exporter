package metrics

import (
	"bmc_exporter/internal/ipmi"
	"context"
	"fmt"
	"log"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type BMCStatusMetric struct {
	Info          *prometheus.GaugeVec // BMC基础信息
	ServiceStatus prometheus.Gauge     // 服务健康状态
	NetworkHealth *prometheus.GaugeVec // 网络连接状态
	LastHeartbeat prometheus.Gauge     // 最后心跳时间
	IPMIErrors    prometheus.Counter   // IPMI错误计数
}

func NewBMCStatusMetric() *BMCStatusMetric {
	return &BMCStatusMetric{
		Info: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "bmc_info",
				Help: "bmc basic information",
			},
			[]string{"manufacturer", "model", "firmware", "ipmi_version"},
		),
		ServiceStatus: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "bmc_service_status",
				Help: "bmc service status (1=normal, 0=abnormal)",
			},
		),
		NetworkHealth: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "bmc_network_health",
				Help: "bmc network connection status",
			},
			[]string{"interface"},
		),
		LastHeartbeat: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "last_heartbeat_timestamp",
				Help: "bmc last heartbeat timestamp",
			},
		),
		IPMIErrors: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "ipmi_errors_total",
				Help: "ipmi communication error count",
			},
		),
	}
}

type BMCInfocollector struct {
	client *ipmi.Client
	metric *BMCStatusMetric
}

func NewBMCInfocollector(client *ipmi.Client) *BMCInfocollector {
	return &BMCInfocollector{
		client: client,
		metric: NewBMCStatusMetric(),
	}
}

func (c *BMCInfocollector) Scrape() error {
	// 采集BMC基础信息
	if err := c.collectBMCInfo(); err != nil {
		c.metric.IPMIErrors.Inc()
		return fmt.Errorf("采集BMC信息失败: %w", err)
	}

	// 采集网络状态
	if err := c.collectNetworkStatus(); err != nil {
		c.metric.IPMIErrors.Inc()
		return fmt.Errorf("采集网络状态失败: %w", err)
	}

	// 更新心跳时间戳
	c.metric.LastHeartbeat.Set(float64(time.Now().Unix()))
	return nil
}

func (c *BMCInfocollector) Describe(ch chan<- *prometheus.Desc) {
	c.metric.Info.Describe(ch)
	c.metric.ServiceStatus.Describe(ch)
	c.metric.NetworkHealth.Describe(ch)
	c.metric.LastHeartbeat.Describe(ch)
	c.metric.IPMIErrors.Describe(ch)
}

func (c *BMCInfocollector) Collect(ch chan<- prometheus.Metric) {
	if err := c.Scrape(); err != nil {
		log.Printf("BMC状态采集失败: %v", err)
	}
	c.metric.Info.Collect(ch)
	c.metric.ServiceStatus.Collect(ch)
	c.metric.NetworkHealth.Collect(ch)
	c.metric.LastHeartbeat.Collect(ch)
	c.metric.IPMIErrors.Collect(ch)
}

func (c *BMCInfocollector) collectBMCInfo() error {
	output, err := c.client.GetBMCInfo(context.Background())
	if err != nil {
		return err
	}

	// 解析示例：
	// Manufacturer ID   : 19046 (0x4a66)
	// Product Name      : Unknown (0x0000)
	// Firmware Revision : 1.23
	info := parseIPMIInfo(output)
	c.metric.Info.WithLabelValues(
		info["manufacturer"],
		info["product"],
		info["firmware"],
		info["ipmi_version"],
	).Set(1)

	if status, err := strconv.ParseFloat(info["service_status"], 64); err == nil {
		c.metric.ServiceStatus.Set(status)
	} else {
		c.metric.ServiceStatus.Set(0) // 解析失败时设为故障状态
	}

	return nil
}

func (c *BMCInfocollector) collectNetworkStatus() error {
	output, err := c.client.GetNetworkStatus(context.Background())
	if err != nil {
		return err
	}

	// 解析示例：
	// MAC Address             : a4:bf:01:12:34:56
	// IP Address Source       : Static
	// Link Status             : Up
	status := parseNetworkStatus(output)
	c.metric.NetworkHealth.WithLabelValues("bmc").Set(status)

	return nil
}

// ---------------------- 工具函数 ----------------------
func parseIPMIInfo(raw string) map[string]string {
	info := make(map[string]string)
	serviceStatus := 0.0

	// 合并正则表达式，同时匹配所有字段
	re := regexp.MustCompile(`(?mi)^([\w\s]+)\s+: (.+)$`)
	ipmiRe := regexp.MustCompile(`(?mi)^IPMI\s+Version\s+: (.+)$`)

	// 先处理IPMI版本
	if match := ipmiRe.FindStringSubmatch(raw); len(match) > 1 {
		info["ipmi_version"] = strings.TrimSpace(match[1])
	}

	// 处理其他字段
	for _, match := range re.FindAllStringSubmatch(raw, -1) {
		key := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(match[1]), " ", "_"))
		value := strings.TrimSpace(match[2])

		switch key {
		case "manufacturer_id":
			info["manufacturer"] = decodeManufacturer(value)
		case "product_name":
			info["product"] = value
		case "firmware_revision":
			info["firmware"] = value
		case "device_available":
			if strings.EqualFold(value, "yes") {
				serviceStatus = 1.0
			}
		case "provides_device_sdrs":
			if !strings.EqualFold(value, "yes") {
				serviceStatus = 0.5
			}
		}
	}

	// 设置默认值
	if _, exists := info["ipmi_version"]; !exists {
		info["ipmi_version"] = "unknown"
	}
	if info["manufacturer"] == "" {
		info["manufacturer"] = "unknown"
	}
	if info["product"] == "" {
		info["product"] = "unknown"
	}
	if info["firmware"] == "" {
		info["firmware"] = "unknown"
	}

	info["service_status"] = fmt.Sprintf("%.1f", serviceStatus)
	return info
}

var (
	networkCheckRegexes = map[string]*regexp.Regexp{
		"ip":        regexp.MustCompile(`IP Address\s+: (\S+)`),
		"subnet":    regexp.MustCompile(`Subnet Mask\s+: (\S+)`),
		"gateway":   regexp.MustCompile(`Default Gateway IP\s+: (\S+)`),
		"set_state": regexp.MustCompile(`Set in Progress\s+: (.+)`),
	}
)

func parseNetworkStatus(raw string) float64 {
	results := make(map[string]string)

	// 提取关键字段
	for key, re := range networkCheckRegexes {
		if match := re.FindStringSubmatch(raw); len(match) > 1 {
			results[key] = strings.TrimSpace(match[1])
		}
	}

	// 状态判断逻辑
	if results["set_state"] != "Set Complete" {
		return 0.0
	}
	if net.ParseIP(results["ip"]).IsUnspecified() {
		return 0.0
	}
	if results["gateway"] == "0.0.0.0" {
		return 0.5
	}

	return 1.0
}

func decodeManufacturer(id string) string {
	// 示例厂商ID解码逻辑
	mfgIDs := map[string]string{
		"19046":   "Dell Inc.",
		"20301":   "HPE",
		"10876":   "Supermicro",
		"default": "Unknown",
	}
	if name, ok := mfgIDs[strings.Fields(id)[0]]; ok {
		return name
	}
	return mfgIDs["default"]
}

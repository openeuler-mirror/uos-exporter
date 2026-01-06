package metrics

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"tencentcloud_exporter/internal/exporter"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	cdb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cdb/v20170320"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	monitor "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/monitor/v20180724"
)

const (
	CdbNamespace     = "QCE/CDB"
	CdbInstanceidKey = "InstanceId"
)

var excludeMetricName = map[string]string{
	"LogVolume":           "LogVolume",
	"CurrentBackupVolume": "CurrentBackupVolume",
	"DataVolume":          "DataVolume",
	"FreeBackupVolume":    "FreeBackupVolume",
	"BillingBackupVolume": "BillingBackupVolume",
}

// 注册CDB指标收集器
func init() {
	// 不在这里直接注册，改为在Collect中动态创建指标
	exporter.Register(NewCdbMonitor())
}

// CdbMonitor 定义CDB指标收集器结构
type CdbMonitor struct {
	// 无需使用baseMetrics
	metricDescriptors map[string]*prometheus.Desc
}

// NewCdbMonitor 创建新的CDB指标收集器
func NewCdbMonitor() *CdbMonitor {
	return &CdbMonitor{
		metricDescriptors: make(map[string]*prometheus.Desc),
	}
}

// Namespace 实现NamespaceCollector接口
func (cm *CdbMonitor) Namespace() string {
	return CdbNamespace
}

// Describe 实现Metric接口
func (cm *CdbMonitor) Describe(ch chan<- *prometheus.Desc) {
	// 描述方法会在收集时动态创建
}

// 创建与原项目一致的指标名称
func createMetricName(namespace, metricName, statType string) string {
	// 按照旧项目的格式: qce_cdb_metricname_stattype
	prefix := "qce"
	productName := ""

	// 处理命名空间
	if namespace != "" {
		nameParts := strings.Split(namespace, "/")
		if len(nameParts) > 0 && strings.Contains(namespace, "/") {
			// 只有当命名空间包含分隔符时，才更新prefix
			prefix = strings.ToLower(nameParts[0])
		}
		if len(nameParts) > 1 {
			productName = strings.ToLower(nameParts[1])
		} else {
			// 如果命名空间不包含分隔符，使用整个命名空间作为产品名
			if namespace != "" {
				productName = strings.ToLower(namespace)
			}
		}
	}

	// 处理指标名称和统计类型
	metricNameLower := strings.ToLower(metricName)
	statTypeLower := strings.ToLower(statType)

	return fmt.Sprintf("%s_%s_%s_%s", prefix, productName, metricNameLower, statTypeLower)
}

// 创建与原项目一致的帮助信息
func createHelpInfo(namespace, metricName, unit, statType, meaning string) string {
	// 按照旧项目的格式: Metric from QCE/CDB.MetricName unit=Unit stat=StatType Desc=中文描述
	return fmt.Sprintf("Metric from %s.%s unit=%s stat=%s Desc=%s",
		namespace, metricName, unit, statType, meaning)
}

// 获取指标的中文描述
func getMetricMeaning(metricName string) string {
	// 完全匹配旧项目中的指标描述
	metricMeanings := map[string]string{
		"Capacity":                         "磁盘占用空间",
		"CapacityUtilization":              "磁盘使用率",
		"CPUUseRate":                       "CPU利用率",
		"MemoryUseRate":                    "内存利用率",
		"BytesReceived":                    "接受数据量",
		"BytesSent":                        "发送数据量",
		"QPS":                              "每秒执行操作数",
		"TPS":                              "每秒执行事务数",
		"Connections":                      "连接数",
		"ConnectionsUseRate":               "连接数使用率",
		"ThreadsConnected":                 "当前打开连接数",
		"ThreadsRunning":                   "当前运行的连接数",
		"MaxConnections":                   "最大连接数",
		"SlaveDelay":                       "主从延迟",
		"MasterSlaveSyncDistance":          "主从延迟",
		"SecondsBehindMaster":              "主从延迟时间",
		"SlowQueries":                      "慢查询数",
		"TableLocksWaited":                 "等待表锁次数",
		"InnoDB_buffer_pool_reads":         "innodb磁盘读次数",
		"InnoDB_buffer_pool_read_requests": "innodb读请求次数",
		"InnoDB_rows_inserted":             "innodb执行INSERT的行数",
		"InnoDB_rows_deleted":              "innodb执行DELETE的行数",
		"InnoDB_rows_updated":              "innodb执行UPDATE的行数",
		"InnoDB_rows_read":                 "innodb执行READ的行数",
		"InnoDB_buffer_pool_pages_free":    "innodb空闲页数",
		"InnoDB_buffer_pool_pages_total":   "innodb总页数",
		"InnoDB_buffer_pool_pages_data":    "innodb数据页数",
		"InnoDB_buffer_pool_pages_dirty":   "innodb脏页数",
		"InnoDB_buffer_pool_pages_flushed": "innodb刷新页数",
		"Innodb_data_reads":                "innodb总读取量",
		"Innodb_data_writes":               "innodb总写入量",
		"Innodb_data_read":                 "innodb总读取字节数",
		"Innodb_data_written":              "innodb总写入字节数",
		"Innodb_os_log_fsyncs":             "innodb日志刷新次数",
		"Innodb_os_log_written":            "innodb日志写入量",
		"Key_read_requests":                "索引读取请求次数",
		"Key_reads":                        "索引物理读次数",
		"Key_write_requests":               "索引写入请求次数",
		"Key_writes":                       "索引物理写次数",
		"Queries":                          "总请求数",
		"Questions":                        "总查询数",
		"Threads_created":                  "已创建的线程数",
		"Created_tmp_disk_tables":          "临时表创建次数",
		"Created_tmp_tables":               "临时表创建次数",
		"Select_full_join":                 "全表联合查询次数",
		"Select_full_range_join":           "全表范围联合查询次数",
		"Select_range":                     "范围查询次数",
		"Select_range_check":               "范围检查查询次数",
		"Select_scan":                      "全表扫描查询次数",
		"Opened_tables":                    "已打开的表数",
		"Opened_files":                     "已打开的文件数",
		"Table_locks_immediate":            "立即锁表次数",
		"Table_locks_waited":               "表锁等待次数",
		"Threads_cached":                   "线程缓存数量",
		"RealCapacity":                     "实际空间",
		"VolumeRate":                       "容量使用率",
		"InnodbVolumeRate":                 "innodb容量使用率",
		"CreatedTmpDiskTablesRate":         "临时表创建磁盘使用率",
		"CpuUsageRate":                     "CPU使用率",
		"MemUsageRate":                     "内存使用率",
		"MemAvailable":                     "可用内存",
		"DiskUsageRate":                    "磁盘使用率",
		"DiskIoRate":                       "磁盘IO利用率",
		"QpsRate":                          "QPS",
		"TpsRate":                          "TPS",
		"InnodbCacheUseRate":               "Innodb缓冲池命中率",
		"KeyCacheUseRate":                  "MyISAM缓冲池命中率",
		"QueryCacheHitRate":                "查询缓存命中率",
		"ThreadsConnectedRate":             "线程连接数利用率",
		"ThreadsUsedRate":                  "线程使用率",
		"InnodbCurRowLockNum":              "Innodb当前行锁数量",
		"InnodbRwlockOsWaitTime":           "Innodb等待操作系统锁时间",
		"InnodbRwlockOsWaits":              "Innodb等待操作系统锁次数",
		"InnodbRowLockWaits":               "Innodb等待行锁次数",
		"InnodbRowLockTimeAvg":             "Innodb平均获取行锁时间",
		"InnodbRowLockTime":                "Innodb获取行锁总时长",
		"InnodbLogWaits":                   "Innodb日志等待次数",
		"SendTotalTimes":                   "发送次数",
		"DiskReadTraffic":                  "磁盘读流量",
		"DiskWriteTraffic":                 "磁盘写流量",
		"DiskReadIops":                     "磁盘读IOPS",
		"DiskWriteIops":                    "磁盘写IOPS",
		"DiskIoWait":                       "磁盘IO等待时间",
		"DiskReadDelay":                    "磁盘读延迟",
		"DiskWriteDelay":                   "磁盘写延迟",
		"VipPackage":                       "包量",
		"VipBandwidth":                     "带宽",
		"VipConns":                         "连接数",
		"VipPkts":                          "新建连接数",
		"VipRate":                          "网络入包量",
	}

	if meaning, ok := metricMeanings[metricName]; ok {
		return meaning
	}
	return "MySQL实例监控指标" // 默认描述
}

// Collect 实现指标收集方法
func (cm *CdbMonitor) Collect(ch chan<- prometheus.Metric) {
	logrus.Info("Start collect CDB metrics...")

	// 获取腾讯云凭证
	credential := common.NewCredential(
		exporter.GetConfig().Credential.AccessKey,
		exporter.GetConfig().Credential.SecretKey,
	)

	// 创建监控API客户端
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "monitor.tencentcloudapi.com"
	monitorClient, err := monitor.NewClient(credential, exporter.GetConfig().Credential.Region, cpf)
	if err != nil {
		logrus.Errorf("Failed to create monitor client: %v", err)
		return
	}

	// 获取CDB实例列表
	instances, err := getCdbInstances(credential, exporter.GetConfig().Credential.Region)
	if err != nil {
		logrus.Errorf("Failed to get CDB instances: %v", err)
		return
	}

	if len(instances) == 0 {
		logrus.Warn("No CDB instances found")
		return
	}

	// 获取指标列表
	metricNames, err := getMetricsByNamespace(monitorClient, CdbNamespace)
	if err != nil {
		logrus.Errorf("Failed to get metrics for namespace %s: %v", CdbNamespace, err)
		return
	}

	// 对每个实例和指标进行查询
	for _, instance := range instances {
		for _, metricInfo := range metricNames {
			// 跳过排除的指标
			if _, ok := excludeMetricName[metricInfo.MetricName]; ok {
				continue
			}

			// 创建查询请求
			request := monitor.NewGetMonitorDataRequest()
			request.Namespace = common.StringPtr(CdbNamespace)
			request.MetricName = common.StringPtr(metricInfo.MetricName)

			// 设置查询时间范围（最近5分钟）
			endTime := time.Now()
			startTime := endTime.Add(-5 * time.Minute)
			request.StartTime = common.StringPtr(startTime.Format("2006-01-02T15:04:05+08:00"))
			request.EndTime = common.StringPtr(endTime.Format("2006-01-02T15:04:05+08:00"))

			// 设置查询周期
			period := int64(60)
			request.Period = common.Uint64Ptr(safeInt64ToUint64(period))

			// 设置查询实例
			request.Instances = []*monitor.Instance{
				{
					Dimensions: []*monitor.Dimension{
						{
							Name:  common.StringPtr(CdbInstanceidKey),
							Value: common.StringPtr(instance.InstanceId),
						},
					},
				},
			}

			// 发送查询请求
			response, err := monitorClient.GetMonitorData(request)
			if err != nil {
				if _, ok := err.(*errors.TencentCloudSDKError); ok {
					logrus.Errorf("Failed to get monitor data for instance %s, metric %s: %v",
						instance.InstanceId, metricInfo.MetricName, err)
					continue
				}
				logrus.Errorf("Unknown error when getting monitor data: %v", err)
				continue
			}

			// 处理查询结果
			if response == nil || response.Response == nil || response.Response.DataPoints == nil || len(response.Response.DataPoints) == 0 {
				logrus.Debugf("No data points returned for instance %s, metric %s", instance.InstanceId, metricInfo.MetricName)
				continue
			}

			// 提取最新的数据点
			dataPoints := response.Response.DataPoints[0]
			if dataPoints == nil || dataPoints.Values == nil || len(dataPoints.Values) == 0 {
				continue
			}

			// 获取最新的指标值
			lastValue := dataPoints.Values[len(dataPoints.Values)-1]
			if lastValue == nil {
				continue
			}

			value, err := strconv.ParseFloat(fmt.Sprintf("%v", *lastValue), 64)
			if err != nil {
				logrus.Errorf("Failed to parse metric value: %v", err)
				continue
			}

			// 处理统计类型
			statTypes := []string{"max"} // 默认使用 max 统计类型，与旧项目保持一致

			for _, statType := range statTypes {
				// 构造符合原项目格式的指标名称和帮助信息
				metricName := createMetricName(CdbNamespace, metricInfo.MetricName, statType)
				meaning := getMetricMeaning(metricInfo.MetricName)
				unit := "unknown"
				if metricInfo.Unit != nil {
					unit = *metricInfo.Unit
				}
				help := createHelpInfo(CdbNamespace, metricInfo.MetricName, unit, statType, meaning)

				// 创建或获取指标描述符
				desc, ok := cm.metricDescriptors[metricName]
				if !ok {
					// 定义标签 - 使用与旧项目完全一致的标签名称
					labelNames := []string{"instance_id", "instanceid"}

					// 创建指标描述符
					desc = prometheus.NewDesc(
						metricName,
						help,
						labelNames,
						nil,
					)
					cm.metricDescriptors[metricName] = desc
				}

				// 构造标签值 - 注意这里使用两个相同的值，与原项目保持一致
				labelValues := []string{
					instance.InstanceId, // instance_id
					instance.InstanceId, // instanceid
				}

				// 发送指标到Prometheus
				metric := prometheus.MustNewConstMetric(
					desc,
					prometheus.GaugeValue,
					value,
					labelValues...,
				)
				ch <- metric

				logrus.Debugf("Collected metric: %s, instance=%s, value=%f",
					metricName, instance.InstanceId, value)
			}
		}
	}
	logrus.Info("Finished collecting CDB metrics")
}

// CdbInstance 定义CDB实例结构
type CdbInstance struct {
	InstanceId   string
	InstanceName string
}

// MetricInfo 定义指标信息结构
type MetricInfo struct {
	MetricName string
	Unit       *string
}

// 获取CDB实例列表
func getCdbInstances(credential *common.Credential, region string) ([]CdbInstance, error) {
	// 创建API请求
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "cdb.tencentcloudapi.com"

	// 使用SDK创建CDB客户端
	cdbClient, err := cdb.NewClient(credential, region, cpf)
	if err != nil {
		return nil, err
	}

	// 查询CDB实例列表
	request := cdb.NewDescribeDBInstancesRequest()
	request.Limit = common.Uint64Ptr(100) // 限制返回100个实例，如果需要更多可以分页查询

	resp, err := cdbClient.DescribeDBInstances(request)
	if err != nil {
		return nil, err
	}

	if resp == nil || resp.Response == nil || resp.Response.Items == nil {
		return nil, fmt.Errorf("empty response when querying CDB instances")
	}

	var instances []CdbInstance
	for _, inst := range resp.Response.Items {
		if inst.InstanceId == nil || inst.InstanceName == nil {
			continue
		}

		instances = append(instances, CdbInstance{
			InstanceId:   *inst.InstanceId,
			InstanceName: *inst.InstanceName,
		})
	}

	return instances, nil
}

// 获取命名空间下的指标列表
func getMetricsByNamespace(client *monitor.Client, namespace string) ([]MetricInfo, error) {
	request := monitor.NewDescribeBaseMetricsRequest()
	request.Namespace = common.StringPtr(namespace)

	response, err := client.DescribeBaseMetrics(request)
	if err != nil {
		return nil, err
	}

	if response == nil || response.Response == nil || response.Response.MetricSet == nil {
		return nil, fmt.Errorf("empty response when querying metrics for namespace %s", namespace)
	}

	var metrics []MetricInfo
	for _, metric := range response.Response.MetricSet {
		// 如果指标被排除，则跳过
		if _, ok := excludeMetricName[*metric.MetricName]; ok {
			continue
		}

		metrics = append(metrics, MetricInfo{
			MetricName: *metric.MetricName,
			Unit:       metric.Unit,
		})
	}

	return metrics, nil
}

func safeInt64ToUint64(value int64) uint64 {
	if value < 0 {
		return 0
	}
	return uint64(value)
}

func safeUint64ToInt64(value uint64) int64 {
	if value > math.MaxInt64 {
		return int64(math.MaxInt64)
	}
	return int64(value)
}

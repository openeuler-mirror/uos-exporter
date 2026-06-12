package metrics

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"elasticsearch_exporter/config"
	"elasticsearch_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// clusterSettingsResponse 是 Elasticsearch 集群设置信息的表示
type clusterSettingsResponse struct {
	Defaults   clusterSettingsSection `json:"defaults"`
	Persistent clusterSettingsSection `json:"persistent"`
	Transient  clusterSettingsSection `json:"transient"`
}

// clusterSettingsSection 是 Elasticsearch 集群设置部分的表示
type clusterSettingsSection struct {
	Cluster clusterSettingsCluster `json:"cluster"`
}

// clusterSettingsCluster 是 Elasticsearch 集群设置集群部分的表示
type clusterSettingsCluster struct {
	Routing clusterSettingsRouting `json:"routing"`
	// 这可以是JSON对象（不包含我们感兴趣的值）或字符串
	MaxShardsPerNode interface{} `json:"max_shards_per_node"`
}

// clusterSettingsRouting 是 Elasticsearch 集群分片路由配置的表示
type clusterSettingsRouting struct {
	Allocation clusterSettingsAllocation `json:"allocation"`
}

// clusterSettingsAllocation 是 Elasticsearch 集群分片路由分配设置的表示
type clusterSettingsAllocation struct {
	Enabled string              `json:"enable"`
	Disk    clusterSettingsDisk `json:"disk"`
}

// clusterSettingsDisk 是 Elasticsearch 集群分片路由磁盘分配设置的表示
type clusterSettingsDisk struct {
	ThresholdEnabled string                   `json:"threshold_enabled"`
	Watermark        clusterSettingsWatermark `json:"watermark"`
}

// clusterSettingsWatermark 是 Elasticsearch 集群分片路由磁盘分配水位设置的表示
type clusterSettingsWatermark struct {
	FloodStage string `json:"flood_stage"`
	High       string `json:"high"`
	Low        string `json:"low"`
}

// ClusterSettings 集群设置指标收集器
type ClusterSettings struct {
	esURL             string
	client            *http.Client
	insecure          bool
	jsonParseFailures prometheus.Counter

	// 分片分配设置
	shardAllocationEnabled *baseMetrics
	maxShardsPerNode       *baseMetrics

	// 磁盘阈值设置
	thresholdEnabled *baseMetrics

	// 水位设置（比率）
	floodStageRatio *baseMetrics
	highRatio       *baseMetrics
	lowRatio        *baseMetrics

	// 水位设置（字节）
	floodStageBytes *baseMetrics
	highBytes       *baseMetrics
	lowBytes        *baseMetrics
}

func init() {
	exporter.Register(NewClusterSettings())
}

// NewClusterSettings 创建集群设置指标收集器
func NewClusterSettings() *ClusterSettings {
	return &ClusterSettings{
		esURL: "http://localhost:9200",
		client: &http.Client{
			Timeout: defaultTimeout,
		},

		jsonParseFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "cluster_settings_json_parse_failures",
			Help:      "Number of errors while parsing JSON.",
		}),

		// 分片分配设置
		shardAllocationEnabled: NewMetrics(
			prometheus.BuildFQName(namespace, "clustersettings_stats", "shard_allocation_enabled"),
			"Current mode of cluster wide shard routing allocation settings.",
			nil,
		),
		maxShardsPerNode: NewMetrics(
			prometheus.BuildFQName(namespace, "clustersettings_stats", "max_shards_per_node"),
			"Current maximum number of shards per node setting.",
			nil,
		),

		// 磁盘阈值设置
		thresholdEnabled: NewMetrics(
			prometheus.BuildFQName(namespace, "clustersettings_allocation", "threshold_enabled"),
			"Is disk allocation decider enabled.",
			nil,
		),

		// 水位设置（比率）
		floodStageRatio: NewMetrics(
			prometheus.BuildFQName(namespace, "clustersettings_allocation_watermark", "flood_stage_ratio"),
			"Flood stage watermark as a ratio.",
			nil,
		),
		highRatio: NewMetrics(
			prometheus.BuildFQName(namespace, "clustersettings_allocation_watermark", "high_ratio"),
			"High watermark for disk usage as a ratio.",
			nil,
		),
		lowRatio: NewMetrics(
			prometheus.BuildFQName(namespace, "clustersettings_allocation_watermark", "low_ratio"),
			"Low watermark for disk usage as a ratio.",
			nil,
		),

		// 水位设置（字节）
		floodStageBytes: NewMetrics(
			prometheus.BuildFQName(namespace, "clustersettings_allocation_watermark", "flood_stage_bytes"),
			"Flood stage watermark as in bytes.",
			nil,
		),
		highBytes: NewMetrics(
			prometheus.BuildFQName(namespace, "clustersettings_allocation_watermark", "high_bytes"),
			"High watermark for disk usage in bytes.",
			nil,
		),
		lowBytes: NewMetrics(
			prometheus.BuildFQName(namespace, "clustersettings_allocation_watermark", "low_bytes"),
			"Low watermark for disk usage in bytes.",
			nil,
		),
	}
}

// fetchAndDecodeClusterSettings 获取并解析集群设置信息
func (c *ClusterSettings) fetchAndDecodeClusterSettings() (clusterSettingsResponse, error) {
	// 确保客户端每次获取时更新配置
	c.client = &http.Client{
		Timeout: defaultTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				// #nosec G402
				InsecureSkipVerify: c.insecure,
			},
		},
	}

	u, err := url.Parse(c.esURL)
	if err != nil {
		return clusterSettingsResponse{}, fmt.Errorf("failed to parse ES URL: %s", err)
	}

	// 构建URL路径
	u.Path = path.Join(u.Path, "/_cluster/settings")
	q := u.Query()
	q.Set("include_defaults", "true")
	u.RawQuery = q.Encode()

	logrus.Debugf("Fetching cluster settings from %s", u.String())

	res, err := c.client.Get(u.String())
	if err != nil {
		return clusterSettingsResponse{}, fmt.Errorf("failed to get cluster settings from %s: %s", u.String(), err)
	}

	defer func() {
		err = res.Body.Close()
		if err != nil {
			logrus.Warnf("failed to close http.Client: %s", err)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return clusterSettingsResponse{}, fmt.Errorf("HTTP Request failed with code %d", res.StatusCode)
	}

	var data clusterSettingsResponse
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		c.jsonParseFailures.Inc()
		return clusterSettingsResponse{}, err
	}

	return data, nil
}

// mergeSettings 合并设置
func (c *ClusterSettings) mergeSettings(data clusterSettingsResponse) clusterSettingsSection {
	// 合并所有设置到一个结构体中
	merged := data.Defaults

	// 简单合并: 持久性设置覆盖默认设置
	if data.Persistent.Cluster.MaxShardsPerNode != nil {
		merged.Cluster.MaxShardsPerNode = data.Persistent.Cluster.MaxShardsPerNode
	}
	if data.Persistent.Cluster.Routing.Allocation.Enabled != "" {
		merged.Cluster.Routing.Allocation.Enabled = data.Persistent.Cluster.Routing.Allocation.Enabled
	}
	if data.Persistent.Cluster.Routing.Allocation.Disk.ThresholdEnabled != "" {
		merged.Cluster.Routing.Allocation.Disk.ThresholdEnabled = data.Persistent.Cluster.Routing.Allocation.Disk.ThresholdEnabled
	}
	if data.Persistent.Cluster.Routing.Allocation.Disk.Watermark.FloodStage != "" {
		merged.Cluster.Routing.Allocation.Disk.Watermark.FloodStage = data.Persistent.Cluster.Routing.Allocation.Disk.Watermark.FloodStage
	}
	if data.Persistent.Cluster.Routing.Allocation.Disk.Watermark.High != "" {
		merged.Cluster.Routing.Allocation.Disk.Watermark.High = data.Persistent.Cluster.Routing.Allocation.Disk.Watermark.High
	}
	if data.Persistent.Cluster.Routing.Allocation.Disk.Watermark.Low != "" {
		merged.Cluster.Routing.Allocation.Disk.Watermark.Low = data.Persistent.Cluster.Routing.Allocation.Disk.Watermark.Low
	}

	// 临时设置覆盖持久性设置
	if data.Transient.Cluster.MaxShardsPerNode != nil {
		merged.Cluster.MaxShardsPerNode = data.Transient.Cluster.MaxShardsPerNode
	}
	if data.Transient.Cluster.Routing.Allocation.Enabled != "" {
		merged.Cluster.Routing.Allocation.Enabled = data.Transient.Cluster.Routing.Allocation.Enabled
	}
	if data.Transient.Cluster.Routing.Allocation.Disk.ThresholdEnabled != "" {
		merged.Cluster.Routing.Allocation.Disk.ThresholdEnabled = data.Transient.Cluster.Routing.Allocation.Disk.ThresholdEnabled
	}
	if data.Transient.Cluster.Routing.Allocation.Disk.Watermark.FloodStage != "" {
		merged.Cluster.Routing.Allocation.Disk.Watermark.FloodStage = data.Transient.Cluster.Routing.Allocation.Disk.Watermark.FloodStage
	}
	if data.Transient.Cluster.Routing.Allocation.Disk.Watermark.High != "" {
		merged.Cluster.Routing.Allocation.Disk.Watermark.High = data.Transient.Cluster.Routing.Allocation.Disk.Watermark.High
	}
	if data.Transient.Cluster.Routing.Allocation.Disk.Watermark.Low != "" {
		merged.Cluster.Routing.Allocation.Disk.Watermark.Low = data.Transient.Cluster.Routing.Allocation.Disk.Watermark.Low
	}

	return merged
}

// getValueInBytes 将带单位的字符串转换为字节数
func (c *ClusterSettings) getValueInBytes(value string) (float64, error) {
	type UnitValue struct {
		unit string
		val  float64
	}

	unitMap := map[string]UnitValue{
		"b":  {"", 1},
		"k":  {"", 1024},
		"kb": {"", 1024},
		"m":  {"", 1024 * 1024},
		"mb": {"", 1024 * 1024},
		"g":  {"", 1024 * 1024 * 1024},
		"gb": {"", 1024 * 1024 * 1024},
		"t":  {"", 1024 * 1024 * 1024 * 1024},
		"tb": {"", 1024 * 1024 * 1024 * 1024},
		"p":  {"", 1024 * 1024 * 1024 * 1024 * 1024},
		"pb": {"", 1024 * 1024 * 1024 * 1024 * 1024},
	}

	value = strings.ToLower(value)
	var unit string
	var val float64
	var err error

	for k := range unitMap {
		if strings.HasSuffix(value, k) {
			unit = k
			valStr := value[0 : len(value)-len(k)]
			val, err = strconv.ParseFloat(valStr, 64)
			if err != nil {
				return 0, err
			}
			break
		}
	}

	if unit == "" {
		return 0, fmt.Errorf("no unit found in %s", value)
	}

	byteVal := val * unitMap[unit].val
	return byteVal, nil
}

// getValueAsRatio 将百分比字符串转换为比率
func (c *ClusterSettings) getValueAsRatio(value string) (float64, error) {
	if strings.HasSuffix(value, "%") {
		value = value[0 : len(value)-1]
		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return 0, err
		}
		return val / 100, nil
	}
	return 0, fmt.Errorf("value is not a percentage: %s", value)
}

// Collect 实现指标收集
func (c *ClusterSettings) Collect(ch chan<- prometheus.Metric) {
	// 从配置中读取ES URL
	if config.ScrapeUrl != nil && *config.ScrapeUrl != "" {
		c.esURL = *config.ScrapeUrl
		logrus.Debugf("Using scrape_uri from command line: %s", c.esURL)
	} else {
		// 检查配置文件中的ScrapeUri
		var settings config.Settings
		if err := exporter.Unpack(&settings); err == nil && settings.ScrapeUri != "" {
			c.esURL = settings.ScrapeUri
			logrus.Debugf("Using scrape_uri from config file: %s", c.esURL)
		} else {
			logrus.Debugf("Using default scrape_uri: %s", c.esURL)
		}
	}

	// 设置是否忽略SSL证书验证
	if config.Insecure != nil && *config.Insecure {
		c.insecure = *config.Insecure
	} else {
		// 检查配置文件中的Insecure设置
		var settings config.Settings
		if err := exporter.Unpack(&settings); err == nil {
			c.insecure = settings.Insecure
		}
	}

	// 确保计数器被收集
	ch <- c.jsonParseFailures

	// 获取集群设置信息
	data, err := c.fetchAndDecodeClusterSettings()
	if err != nil {
		logrus.Warnf("Failed to fetch and decode cluster settings: %s", err)
		return
	}

	// 合并设置
	merged := c.mergeSettings(data)

	// Max shards per node
	if maxShardsPerNodeString, ok := merged.Cluster.MaxShardsPerNode.(string); ok {
		maxShardsPerNode, err := strconv.ParseInt(maxShardsPerNodeString, 10, 64)
		if err == nil {
			c.maxShardsPerNode.collect(ch, float64(maxShardsPerNode), nil)
		}
	}

	// Shard allocation enabled
	shardAllocationMap := map[string]int{
		"all":           0,
		"primaries":     1,
		"new_primaries": 2,
		"none":          3,
	}
	if val, ok := shardAllocationMap[merged.Cluster.Routing.Allocation.Enabled]; ok {
		c.shardAllocationEnabled.collect(ch, float64(val), nil)
	}

	// Disk threshold enabled
	thresholdEnabledMap := map[string]int{
		"true":  1,
		"false": 0,
	}
	if val, ok := thresholdEnabledMap[merged.Cluster.Routing.Allocation.Disk.ThresholdEnabled]; ok {
		c.thresholdEnabled.collect(ch, float64(val), nil)
	}

	// Watermark settings
	floodStage := merged.Cluster.Routing.Allocation.Disk.Watermark.FloodStage
	high := merged.Cluster.Routing.Allocation.Disk.Watermark.High
	low := merged.Cluster.Routing.Allocation.Disk.Watermark.Low

	// 尝试转换为比率
	if floodStageRatio, err := c.getValueAsRatio(floodStage); err == nil {
		c.floodStageRatio.collect(ch, floodStageRatio, nil)
	}
	if highRatio, err := c.getValueAsRatio(high); err == nil {
		c.highRatio.collect(ch, highRatio, nil)
	}
	if lowRatio, err := c.getValueAsRatio(low); err == nil {
		c.lowRatio.collect(ch, lowRatio, nil)
	}

	// 尝试转换为字节
	if floodStageBytes, err := c.getValueInBytes(floodStage); err == nil {
		c.floodStageBytes.collect(ch, floodStageBytes, nil)
	}
	if highBytes, err := c.getValueInBytes(high); err == nil {
		c.highBytes.collect(ch, highBytes, nil)
	}
	if lowBytes, err := c.getValueInBytes(low); err == nil {
		c.lowBytes.collect(ch, lowBytes, nil)
	}
} 

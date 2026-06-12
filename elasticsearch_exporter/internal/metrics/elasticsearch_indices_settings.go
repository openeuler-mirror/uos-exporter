package metrics

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"elasticsearch_exporter/config"
	"elasticsearch_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// IndicesSettingsResponse 表示每个索引的 Elasticsearch 设置
type IndicesSettingsResponse map[string]IndexSettings

// IndexSettings 定义每个索引设置的树结构
type IndexSettings struct {
	Settings Settings `json:"settings"`
}

// Settings 定义当前索引设置
type Settings struct {
	IndexInfo IndexInfo `json:"index"`
}

// IndexInfo 定义当前索引的块信息
type IndexInfo struct {
	Blocks           Blocks  `json:"blocks"`
	Mapping          Mapping `json:"mapping"`
	NumberOfReplicas string  `json:"number_of_replicas"`
	CreationDate     string  `json:"creation_date"`
}

// Blocks 定义当前索引是否启用了 read_only_allow_delete
type Blocks struct {
	ReadOnly string `json:"read_only_allow_delete"`
}

// Mapping 定义映射设置
type Mapping struct {
	TotalFields TotalFields `json:"total_fields"`
}

// TotalFields 定义映射字段数量的限制
type TotalFields struct {
	Limit string `json:"limit"`
}

// 默认值常量
const (
	defaultTotalFieldsValue = 1000 // ES 默认的字段总数配置
	defaultDateCreation     = 0    // ES 索引默认创建日期
)

// IndicesSettings 索引设置指标收集器
type IndicesSettings struct {
	esURL             string
	client            *http.Client
	insecure          bool
	jsonParseFailures prometheus.Counter

	// 只读索引数量
	readOnlyIndices prometheus.Gauge

	// 索引设置指标
	totalFields            *baseMetrics
	replicas               *baseMetrics
	creationTimestampSecs  *baseMetrics
}

func init() {
	exporter.Register(NewIndicesSettings())
}

// NewIndicesSettings 创建索引设置指标收集器
func NewIndicesSettings() *IndicesSettings {
	return &IndicesSettings{
		esURL: "http://localhost:9200",
		client: &http.Client{
			Timeout: defaultTimeout,
		},

		jsonParseFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "indices_settings_json_parse_failures",
			Help:      "Number of errors while parsing JSON.",
		}),

		// 只读索引数量
		readOnlyIndices: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "indices_settings_stats",
			Name:      "read_only_indices",
			Help:      "Current number of read only indices within cluster",
		}),

		// 索引设置指标
		totalFields: NewMetrics(
			prometheus.BuildFQName(namespace, "indices_settings", "total_fields"),
			"Index mapping setting for total_fields",
			[]string{"index"},
		),
		
		replicas: NewMetrics(
			prometheus.BuildFQName(namespace, "indices_settings", "replicas"),
			"Index setting number_of_replicas",
			[]string{"index"},
		),
		
		creationTimestampSecs: NewMetrics(
			prometheus.BuildFQName(namespace, "indices_settings", "creation_timestamp_seconds"),
			"Index setting creation_date",
			[]string{"index"},
		),
	}
}

// fetchAndDecodeIndicesSettings 获取并解析索引设置信息
func (is *IndicesSettings) fetchAndDecodeIndicesSettings() (IndicesSettingsResponse, error) {
	// 确保客户端每次获取时更新配置
	is.client = &http.Client{
		Timeout: defaultTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				// #nosec G402
				InsecureSkipVerify: is.insecure,
			},
		},
	}

	u, err := url.Parse(is.esURL)
	if err != nil {
		return IndicesSettingsResponse{}, fmt.Errorf("failed to parse ES URL: %s", err)
	}

	// 构建URL路径
	u.Path = path.Join(u.Path, "/_all/_settings")

	logrus.Debugf("Fetching indices settings from %s", u.String())

	res, err := is.client.Get(u.String())
	if err != nil {
		return IndicesSettingsResponse{}, fmt.Errorf("failed to get indices settings from %s: %s", u.String(), err)
	}

	defer func() {
		err = res.Body.Close()
		if err != nil {
			logrus.Warnf("failed to close http.Client: %s", err)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return IndicesSettingsResponse{}, fmt.Errorf("HTTP Request failed with code %d", res.StatusCode)
	}

	var data IndicesSettingsResponse
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		is.jsonParseFailures.Inc()
		return IndicesSettingsResponse{}, err
	}

	return data, nil
}

// Collect 实现指标收集
func (is *IndicesSettings) Collect(ch chan<- prometheus.Metric) {
	// 从配置中读取ES URL
	if config.ScrapeUrl != nil && *config.ScrapeUrl != "" {
		is.esURL = *config.ScrapeUrl
		logrus.Debugf("Using scrape_uri from command line: %s", is.esURL)
	} else {
		// 检查配置文件中的ScrapeUri
		var settings config.Settings
		if err := exporter.Unpack(&settings); err == nil && settings.ScrapeUri != "" {
			is.esURL = settings.ScrapeUri
			logrus.Debugf("Using scrape_uri from config file: %s", is.esURL)
		} else {
			logrus.Debugf("Using default scrape_uri: %s", is.esURL)
		}
	}

	// 设置是否忽略SSL证书验证
	if config.Insecure != nil && *config.Insecure {
		is.insecure = *config.Insecure
	} else {
		// 检查配置文件中的Insecure设置
		var settings config.Settings
		if err := exporter.Unpack(&settings); err == nil {
			is.insecure = settings.Insecure
		}
	}

	// 确保计数器被收集
	ch <- is.jsonParseFailures

	// 获取索引设置信息
	data, err := is.fetchAndDecodeIndicesSettings()
	if err != nil {
		is.readOnlyIndices.Set(0)
		logrus.Warnf("Failed to fetch and decode indices settings: %s", err)
		ch <- is.readOnlyIndices
		return
	}

	// 统计只读索引数量
	var readOnlyCount int
	for indexName, value := range data {
		if value.Settings.IndexInfo.Blocks.ReadOnly == "true" {
			readOnlyCount++
		}

		// 总字段数
		totalFieldsValue := defaultTotalFieldsValue
		if value.Settings.IndexInfo.Mapping.TotalFields.Limit != "" {
			if val, err := strconv.ParseFloat(value.Settings.IndexInfo.Mapping.TotalFields.Limit, 64); err == nil {
				totalFieldsValue = int(val)
			}
		}
		is.totalFields.collect(ch, float64(totalFieldsValue), prometheus.Labels{"index": indexName})

		// 副本数
		replicasValue := 0
		if value.Settings.IndexInfo.NumberOfReplicas != "" {
			if val, err := strconv.ParseFloat(value.Settings.IndexInfo.NumberOfReplicas, 64); err == nil {
				replicasValue = int(val)
			}
		}
		is.replicas.collect(ch, float64(replicasValue), prometheus.Labels{"index": indexName})

		// 创建时间
		creationDate := defaultDateCreation
		if value.Settings.IndexInfo.CreationDate != "" {
			if val, err := strconv.ParseFloat(value.Settings.IndexInfo.CreationDate, 64); err == nil {
				creationDate = int(val)
			}
		}
		is.creationTimestampSecs.collect(ch, float64(creationDate)/1000.0, prometheus.Labels{"index": indexName})
	}

	// 更新并收集只读索引数量
	is.readOnlyIndices.Set(float64(readOnlyCount))
	ch <- is.readOnlyIndices
} 

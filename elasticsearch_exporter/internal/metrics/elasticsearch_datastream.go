package metrics

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"elasticsearch_exporter/config"
	"elasticsearch_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// DataStreamStatsResponse 数据流统计信息的表示
type DataStreamStatsResponse struct {
	Shards              DataStreamStatsShards       `json:"_shards"`
	DataStreamCount     int64                       `json:"data_stream_count"`
	BackingIndices      int64                       `json:"backing_indices"`
	TotalStoreSizeBytes int64                       `json:"total_store_size_bytes"`
	DataStreamStats     []DataStreamStatsDataStream `json:"data_streams"`
}

// DataStreamStatsShards 定义数据流统计分片信息结构
type DataStreamStatsShards struct {
	Total      int64 `json:"total"`
	Successful int64 `json:"successful"`
	Failed     int64 `json:"failed"`
}

// DataStreamStatsDataStream 定义每个数据流统计的结构
type DataStreamStatsDataStream struct {
	DataStream       string `json:"data_stream"`
	BackingIndices   int64  `json:"backing_indices"`
	StoreSizeBytes   int64  `json:"store_size_bytes"`
	MaximumTimestamp int64  `json:"maximum_timestamp"`
}

// DataStream 数据流指标收集器
type DataStream struct {
	esURL             string
	client            *http.Client
	insecure          bool
	jsonParseFailures prometheus.Counter

	// 数据流指标
	backingIndicesTotal *baseMetrics
	storeSizeBytes      *baseMetrics
}

func init() {
	exporter.Register(NewDataStream())
}

// NewDataStream 创建数据流指标收集器
func NewDataStream() *DataStream {
	return &DataStream{
		esURL: "http://localhost:9200",
		client: &http.Client{
			Timeout: defaultTimeout,
		},

		jsonParseFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "data_stream_json_parse_failures",
			Help:      "Number of errors while parsing JSON.",
		}),

		// 数据流指标
		backingIndicesTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "data_stream", "backing_indices_total"),
			"Number of backing indices",
			[]string{"data_stream"},
		),
		
		storeSizeBytes: NewMetrics(
			prometheus.BuildFQName(namespace, "data_stream", "store_size_bytes"),
			"Store size of data stream",
			[]string{"data_stream"},
		),
	}
}

// fetchAndDecodeDataStreamStats 获取并解析数据流统计信息
func (ds *DataStream) fetchAndDecodeDataStreamStats() (DataStreamStatsResponse, error) {
	// 确保客户端每次获取时更新配置
	ds.client = &http.Client{
		Timeout: defaultTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				// #nosec G402
				InsecureSkipVerify: ds.insecure,
			},
		},
	}

	u, err := url.Parse(ds.esURL)
	if err != nil {
		return DataStreamStatsResponse{}, fmt.Errorf("failed to parse ES URL: %s", err)
	}

	// 构建URL路径
	u.Path = path.Join(u.Path, "/_data_stream/*/_stats")

	logrus.Debugf("Fetching data stream stats from %s", u.String())

	res, err := ds.client.Get(u.String())
	if err != nil {
		return DataStreamStatsResponse{}, fmt.Errorf("failed to get data stream stats from %s: %s", u.String(), err)
	}

	defer func() {
		err = res.Body.Close()
		if err != nil {
			logrus.Warnf("failed to close http.Client: %s", err)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return DataStreamStatsResponse{}, fmt.Errorf("HTTP Request failed with code %d", res.StatusCode)
	}

	var data DataStreamStatsResponse
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		ds.jsonParseFailures.Inc()
		return DataStreamStatsResponse{}, err
	}

	return data, nil
}

// Describe 实现 prometheus.Collector 接口
func (ds *DataStream) Describe(ch chan<- *prometheus.Desc) {
	ch <- ds.backingIndicesTotal.desc
	ch <- ds.storeSizeBytes.desc
	ch <- ds.jsonParseFailures.Desc()
}

// Collect 实现指标收集
func (ds *DataStream) Collect(ch chan<- prometheus.Metric) {
	// 从配置中读取ES URL
	if config.ScrapeUrl != nil && *config.ScrapeUrl != "" {
		ds.esURL = *config.ScrapeUrl
		logrus.Debugf("Using scrape_uri from command line: %s", ds.esURL)
	} else {
		// 检查配置文件中的ScrapeUri
		var settings config.Settings
		if err := exporter.Unpack(&settings); err == nil && settings.ScrapeUri != "" {
			ds.esURL = settings.ScrapeUri
			logrus.Debugf("Using scrape_uri from config file: %s", ds.esURL)
		} else {
			logrus.Debugf("Using default scrape_uri: %s", ds.esURL)
		}
	}

	// 设置是否忽略SSL证书验证
	if config.Insecure != nil && *config.Insecure {
		ds.insecure = *config.Insecure
	} else {
		// 检查配置文件中的Insecure设置
		var settings config.Settings
		if err := exporter.Unpack(&settings); err == nil {
			ds.insecure = settings.Insecure
		}
	}

	// 确保计数器被收集
	ch <- ds.jsonParseFailures

	// 获取数据流统计信息
	data, err := ds.fetchAndDecodeDataStreamStats()
	if err != nil {
		logrus.Warnf("Failed to fetch and decode data stream stats: %s", err)
		return
	}

	// 收集数据流指标
	for _, dataStream := range data.DataStreamStats {
		ds.backingIndicesTotal.collect(ch, float64(dataStream.BackingIndices), prometheus.Labels{"data_stream": dataStream.DataStream})
		ds.storeSizeBytes.collect(ch, float64(dataStream.StoreSizeBytes), prometheus.Labels{"data_stream": dataStream.DataStream})
	}
} 

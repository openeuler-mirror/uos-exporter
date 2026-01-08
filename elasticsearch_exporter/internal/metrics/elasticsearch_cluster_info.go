package metrics

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"elasticsearch_exporter/config"
	"elasticsearch_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// ClusterInfoResponse 是从 / 端点可获取的集群信息
type ClusterInfoResponse struct {
	Name        string      `json:"name"`
	ClusterName string      `json:"cluster_name"`
	ClusterUUID string      `json:"cluster_uuid"`
	Version     VersionInfo `json:"version"`
	Tagline     string      `json:"tagline"`
}

// VersionInfo 是从 / 端点可获取的版本信息，嵌入在 ClusterInfoResponse 中
type VersionInfo struct {
	Number        string `json:"number"`
	BuildHash     string `json:"build_hash"`
	BuildDate     string `json:"build_date"`
	BuildSnapshot bool   `json:"build_snapshot"`
	LuceneVersion string `json:"lucene_version"`
}

// ClusterInfo 集群信息指标收集器
type ClusterInfo struct {
	esURL             string
	client            *http.Client
	insecure          bool
	jsonParseFailures prometheus.Counter

	// 集群信息指标
	version *prometheus.Desc
	up      *prometheus.Desc
}

func init() {
	exporter.Register(NewClusterInfo())
}

// NewClusterInfo 创建集群信息指标收集器
func NewClusterInfo() *ClusterInfo {
	return &ClusterInfo{
		esURL: "http://localhost:9200",
		client: &http.Client{
			Timeout: defaultTimeout,
		},

		jsonParseFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "cluster_info_json_parse_failures",
			Help:      "Number of errors while parsing JSON.",
		}),

		// 集群信息指标
		version: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "version"),
			"Elasticsearch version information.",
			[]string{
				"cluster",
				"cluster_uuid",
				"build_date",
				"build_hash",
				"version",
				"lucene_version",
			},
			nil,
		),
		up: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "clusterinfo_up"),
			"Up metric for the cluster info collector",
			[]string{"url"},
			nil,
		),
	}
}

// fetchAndDecodeClusterInfo 获取并解析集群信息
func (c *ClusterInfo) fetchAndDecodeClusterInfo() (ClusterInfoResponse, error) {
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
		return ClusterInfoResponse{}, fmt.Errorf("failed to parse ES URL: %s", err)
	}

	logrus.Debugf("Fetching cluster info from %s", u.String())

	res, err := c.client.Get(u.String())
	if err != nil {
		return ClusterInfoResponse{}, fmt.Errorf("failed to get cluster info from %s: %s", u.String(), err)
	}

	defer func() {
		err = res.Body.Close()
		if err != nil {
			logrus.Warnf("failed to close http.Client: %s", err)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return ClusterInfoResponse{}, fmt.Errorf("HTTP Request failed with code %d", res.StatusCode)
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return ClusterInfoResponse{}, err
	}

	var info ClusterInfoResponse
	if err := json.Unmarshal(b, &info); err != nil {
		c.jsonParseFailures.Inc()
		return ClusterInfoResponse{}, err
	}

	return info, nil
}

// Describe 实现prometheus.Collector接口
func (c *ClusterInfo) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.version
	ch <- c.up
	ch <- c.jsonParseFailures.Desc()
}

// Collect 实现指标收集
func (c *ClusterInfo) Collect(ch chan<- prometheus.Metric) {
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

	// 获取集群信息
	info, err := c.fetchAndDecodeClusterInfo()
	if err != nil {
		logrus.Warnf("Failed to fetch and decode cluster info: %s", err)
		ch <- prometheus.MustNewConstMetric(
			c.up,
			prometheus.GaugeValue,
			0,
			c.esURL,
		)
		return
	}

	// 收集集群信息指标
	ch <- prometheus.MustNewConstMetric(
		c.version,
		prometheus.GaugeValue,
		1,
		info.ClusterName,
		info.ClusterUUID,
		info.Version.BuildDate,
		info.Version.BuildHash,
		info.Version.Number,
		info.Version.LuceneVersion,
	)
	
	// 收集up指标
	ch <- prometheus.MustNewConstMetric(
		c.up,
		prometheus.GaugeValue,
		1,
		c.esURL,
	)
} 

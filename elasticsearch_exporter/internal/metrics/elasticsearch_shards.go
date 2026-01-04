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

// ShardResponse 表示分片响应信息
type ShardResponse struct {
	Index string `json:"index"`
	Shard string `json:"shard"`
	State string `json:"state"`
	Node  string `json:"node"`
}

// Shards 分片信息收集器
type Shards struct {
	esURL             string
	client            *http.Client
	insecure          bool
	jsonParseFailures prometheus.Counter
	nodeShardTotal    *baseMetrics
}

func init() {
	exporter.Register(NewShards())
}

// NewShards 创建分片监控收集器
func NewShards() *Shards {
	return &Shards{
		esURL: "http://localhost:9200",
		client: &http.Client{
			Timeout: defaultTimeout,
		},
		jsonParseFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "node_shards_json_parse_failures",
			Help:      "Number of errors while parsing JSON.",
		}),
		nodeShardTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "node_shards", "total"),
			"Total shards per node",
			[]string{"node", "cluster"},
		),
	}
}

// fetchAndDecodeShards 获取并解析分片信息
func (s *Shards) fetchAndDecodeShards() ([]ShardResponse, error) {
	// 确保客户端每次获取时更新配置
	s.client = &http.Client{
		Timeout: defaultTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				// #nosec G402
				InsecureSkipVerify: s.insecure,
			},
		},
	}

	u, err := url.Parse(s.esURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ES URL: %s", err)
	}

	// 构建URL路径
	u.Path = path.Join(u.Path, "/_cat/shards")
	q := u.Query()
	q.Set("format", "json")
	u.RawQuery = q.Encode()

	logrus.Debugf("Fetching shards from %s", u.String())

	res, err := s.client.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get shards from %s: %s", u.String(), err)
	}

	defer func() {
		err = res.Body.Close()
		if err != nil {
			logrus.Warnf("failed to close http.Client: %s", err)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP Request failed with code %d", res.StatusCode)
	}

	var shards []ShardResponse
	if err := json.NewDecoder(res.Body).Decode(&shards); err != nil {
		s.jsonParseFailures.Inc()
		return nil, err
	}

	return shards, nil
}

// fetchClusterInfo 获取集群信息
func (s *Shards) fetchClusterInfo() (string, error) {
	u, err := url.Parse(s.esURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse ES URL: %s", err)
	}
	
	// 构建URL路径
	u.Path = path.Join(u.Path, "/")
	
	logrus.Debugf("Fetching cluster info from %s", u.String())
	
	res, err := s.client.Get(u.String())
	if err != nil {
		return "", fmt.Errorf("failed to get cluster info from %s: %s", u.String(), err)
	}
	
	defer func() {
		err = res.Body.Close()
		if err != nil {
			logrus.Warnf("failed to close http.Client: %s", err)
		}
	}()
	
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP Request failed with code %d", res.StatusCode)
	}
	
	var response struct {
		ClusterName string `json:"cluster_name"`
	}
	
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		s.jsonParseFailures.Inc()
		return "", err
	}
	
	return response.ClusterName, nil
}

// Describe 实现 prometheus.Collector 接口
func (s *Shards) Describe(ch chan<- *prometheus.Desc) {
	ch <- s.nodeShardTotal.desc
	ch <- s.jsonParseFailures.Desc()
}

// Collect 实现指标收集
func (s *Shards) Collect(ch chan<- prometheus.Metric) {
	// 从配置中读取ES URL
	if config.ScrapeUrl != nil && *config.ScrapeUrl != "" {
		s.esURL = *config.ScrapeUrl
		logrus.Debugf("Using scrape_uri from command line: %s", s.esURL)
	} else {
		// 检查配置文件中的ScrapeUri
		var settings config.Settings
		if err := exporter.Unpack(&settings); err == nil && settings.ScrapeUri != "" {
			s.esURL = settings.ScrapeUri
			logrus.Debugf("Using scrape_uri from config file: %s", s.esURL)
		} else {
			logrus.Debugf("Using default scrape_uri: %s", s.esURL)
		}
	}

	// 设置是否忽略SSL证书验证
	if config.Insecure != nil && *config.Insecure {
		s.insecure = *config.Insecure
	} else {
		// 检查配置文件中的Insecure设置
		var settings config.Settings
		if err := exporter.Unpack(&settings); err == nil {
			s.insecure = settings.Insecure
		}
	}

	// 确保计数器被收集
	ch <- s.jsonParseFailures

	// 获取分片信息
	shards, err := s.fetchAndDecodeShards()
	if err != nil {
		logrus.Warnf("Failed to fetch and decode shards: %s", err)
		return
	}

	// 统计每个节点上的分片数量
	nodeShards := make(map[string]float64)
	var clusterName string

	// 从配置中读取集群名称
	var settings config.Settings
	if err := exporter.Unpack(&settings); err == nil {
		// 使用静态值，config.Settings中没有ClusterName字段
		clusterName = "elasticsearch"
	} else {
		clusterName = "unknown_cluster"
	}

	// 统计每个节点上的已启动分片数量
	for _, shard := range shards {
		if shard.State == "STARTED" && shard.Node != "" {
			nodeShards[shard.Node]++
		}
	}

	// 收集每个节点的分片数量指标
	for node, count := range nodeShards {
		s.nodeShardTotal.collect(ch, count, prometheus.Labels{
			"node":    node,
			"cluster": clusterName,
		})
	}
} 

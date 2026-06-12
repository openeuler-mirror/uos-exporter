package metrics

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"elasticsearch_exporter/config"
	"elasticsearch_exporter/internal/exporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// 集群健康响应结构体
type clusterHealthResponse struct {
	ClusterName                 string  `json:"cluster_name"`
	Status                      string  `json:"status"`
	TimedOut                    bool    `json:"timed_out"`
	NumberOfNodes               int     `json:"number_of_nodes"`
	NumberOfDataNodes           int     `json:"number_of_data_nodes"`
	ActivePrimaryShards         int     `json:"active_primary_shards"`
	ActiveShards                int     `json:"active_shards"`
	RelocatingShards            int     `json:"relocating_shards"`
	InitializingShards          int     `json:"initializing_shards"`
	UnassignedShards            int     `json:"unassigned_shards"`
	DelayedUnassignedShards     int     `json:"delayed_unassigned_shards"`
	NumberOfPendingTasks        int     `json:"number_of_pending_tasks"`
	NumberOfInFlightFetch       int     `json:"number_of_in_flight_fetch"`
	TaskMaxWaitingInQueueMillis int     `json:"task_max_waiting_in_queue_millis"`
	ActiveShardsPercentAsNumber float64 `json:"active_shards_percent_as_number"`
}

var (
	colors = []string{"green", "yellow", "red"}
)

// ClusterHealth 集群健康指标收集器
type ClusterHealth struct {
	esURL             string
	client            *http.Client
	insecure          bool
	jsonParseFailures prometheus.Counter
	
	// 集群健康指标
	activePrimaryShards     *baseMetrics
	activeShards            *baseMetrics
	delayedUnassignedShards *baseMetrics
	initializingShards      *baseMetrics
	numberOfDataNodes       *baseMetrics
	numberOfInFlightFetch   *baseMetrics
	taskMaxWaitingInQueue   *baseMetrics
	numberOfNodes           *baseMetrics
	numberOfPendingTasks    *baseMetrics
	relocatingShards        *baseMetrics
	unassignedShards        *baseMetrics
	
	// 集群状态指标
	clusterStatusGreen  *baseMetrics
	clusterStatusYellow *baseMetrics
	clusterStatusRed    *baseMetrics
	
	// 集群状态指标
	clusterStatus   *baseMetrics
	timedOut        *baseMetrics
	
	// Up指标
	up              *prometheus.Desc
}

func init() {
	exporter.Register(NewClusterHealth())
}

// NewClusterHealth 创建集群健康指标收集器
func NewClusterHealth() *ClusterHealth {
	// 默认标签
	defaultClusterHealthLabels := []string{"cluster"}
	
	// 创建收集器
	clusterHealth := &ClusterHealth{
		esURL: "http://localhost:9200",
		client: &http.Client{
			Timeout: defaultTimeout,
		},
		
		jsonParseFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "cluster_health_json_parse_failures",
			Help:      "Number of errors while parsing JSON.",
		}),
		
		// 集群健康指标
		activePrimaryShards: NewMetrics(
			prometheus.BuildFQName(namespace, "cluster_health", "active_primary_shards"),
			"The number of primary shards in your cluster. This is an aggregate total across all indices.",
			defaultClusterHealthLabels,
		),
		activeShards: NewMetrics(
			prometheus.BuildFQName(namespace, "cluster_health", "active_shards"),
			"Aggregate total of all shards across all indices, which includes replica shards.",
			defaultClusterHealthLabels,
		),
		delayedUnassignedShards: NewMetrics(
			prometheus.BuildFQName(namespace, "cluster_health", "delayed_unassigned_shards"),
			"Shards delayed to reduce reallocation overhead",
			defaultClusterHealthLabels,
		),
		initializingShards: NewMetrics(
			prometheus.BuildFQName(namespace, "cluster_health", "initializing_shards"),
			"Count of shards that are being freshly created.",
			defaultClusterHealthLabels,
		),
		numberOfDataNodes: NewMetrics(
			prometheus.BuildFQName(namespace, "cluster_health", "number_of_data_nodes"),
			"Number of data nodes in the cluster.",
			defaultClusterHealthLabels,
		),
		numberOfInFlightFetch: NewMetrics(
			prometheus.BuildFQName(namespace, "cluster_health", "number_of_in_flight_fetch"),
			"The number of ongoing shard info requests.",
			defaultClusterHealthLabels,
		),
		taskMaxWaitingInQueue: NewMetrics(
			prometheus.BuildFQName(namespace, "cluster_health", "task_max_waiting_in_queue_millis"),
			"Tasks max time waiting in queue.",
			defaultClusterHealthLabels,
		),
		numberOfNodes: NewMetrics(
			prometheus.BuildFQName(namespace, "cluster_health", "number_of_nodes"),
			"Number of nodes in the cluster.",
			defaultClusterHealthLabels,
		),
		numberOfPendingTasks: NewMetrics(
			prometheus.BuildFQName(namespace, "cluster_health", "number_of_pending_tasks"),
			"Cluster level changes which have not yet been executed",
			defaultClusterHealthLabels,
		),
		relocatingShards: NewMetrics(
			prometheus.BuildFQName(namespace, "cluster_health", "relocating_shards"),
			"The number of shards that are currently moving from one node to another node.",
			defaultClusterHealthLabels,
		),
		unassignedShards: NewMetrics(
			prometheus.BuildFQName(namespace, "cluster_health", "unassigned_shards"),
			"The number of shards that exist in the cluster state, but cannot be found in the cluster itself.",
			defaultClusterHealthLabels,
		),
		
		// 集群状态指标
		clusterStatusGreen: NewMetrics(
			prometheus.BuildFQName(namespace, "cluster_health", "status"),
			"Whether all primary and replica shards are allocated.",
			[]string{"cluster", "color"},
		),
		clusterStatusYellow: NewMetrics(
			prometheus.BuildFQName(namespace, "cluster_health", "status"),
			"Whether all primary and replica shards are allocated.",
			[]string{"cluster", "color"},
		),
		clusterStatusRed: NewMetrics(
			prometheus.BuildFQName(namespace, "cluster_health", "status"),
			"Whether all primary and replica shards are allocated.",
			[]string{"cluster", "color"},
		),
		
		// 集群状态指标
		clusterStatus: NewMetrics(
			prometheus.BuildFQName(namespace, "cluster_health", "status"),
			"Whether all primary and replica shards are allocated.",
			defaultClusterHealthLabels,
		),
		timedOut: NewMetrics(
			prometheus.BuildFQName(namespace, "cluster_health", "timed_out"),
			"Cluster health timed out (0=false, 1=true)",
			defaultClusterHealthLabels,
		),
		
		// Up指标
		up: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "clusterhealth_up"),
			"Up metric for the cluster health collector",
			[]string{"url"},
			nil,
		),
	}
	
	return clusterHealth
}

// fetchAndDecodeClusterHealth 获取并解析集群健康信息
func (c *ClusterHealth) fetchAndDecodeClusterHealth() (clusterHealthResponse, error) {
	var chr clusterHealthResponse
	
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
		return chr, fmt.Errorf("failed to parse ES URL: %s", err)
	}
	
	// 构建URL路径
	u.Path = path.Join(u.Path, "/_cluster/health")
	
	logrus.Debugf("Fetching cluster health from %s", u.String())
	
	res, err := c.client.Get(u.String())
	if err != nil {
		return chr, fmt.Errorf("failed to get cluster health from %s: %s", u.String(), err)
	}
	
	defer func() {
		err = res.Body.Close()
		if err != nil {
			logrus.Warnf("failed to close http.Client: %s", err)
		}
	}()
	
	if res.StatusCode != http.StatusOK {
		return chr, fmt.Errorf("HTTP Request failed with code %d", res.StatusCode)
	}
	
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return chr, err
	}
	
	if err := json.Unmarshal(body, &chr); err != nil {
		c.jsonParseFailures.Inc()
		return chr, err
	}
	
	return chr, nil
}

// Collect 实现指标收集
func (c *ClusterHealth) Collect(ch chan<- prometheus.Metric) {
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
	
	// 获取集群健康信息
	clusterHealth, err := c.fetchAndDecodeClusterHealth()
	if err != nil {
		logrus.Warnf("Failed to fetch and decode cluster health: %s", err)
		ch <- prometheus.MustNewConstMetric(
			c.up,
			prometheus.GaugeValue,
			0,
			c.esURL,
		)
		return
	}
	
	// 构建标签值
	clusterLabels := []string{clusterHealth.ClusterName}
	
	// 收集集群健康指标
	c.activePrimaryShards.collect(ch, float64(clusterHealth.ActivePrimaryShards), clusterLabels)
	c.activeShards.collect(ch, float64(clusterHealth.ActiveShards), clusterLabels)
	c.delayedUnassignedShards.collect(ch, float64(clusterHealth.DelayedUnassignedShards), clusterLabels)
	c.initializingShards.collect(ch, float64(clusterHealth.InitializingShards), clusterLabels)
	c.numberOfDataNodes.collect(ch, float64(clusterHealth.NumberOfDataNodes), clusterLabels)
	c.numberOfInFlightFetch.collect(ch, float64(clusterHealth.NumberOfInFlightFetch), clusterLabels)
	c.taskMaxWaitingInQueue.collect(ch, float64(clusterHealth.TaskMaxWaitingInQueueMillis), clusterLabels)
	c.numberOfNodes.collect(ch, float64(clusterHealth.NumberOfNodes), clusterLabels)
	c.numberOfPendingTasks.collect(ch, float64(clusterHealth.NumberOfPendingTasks), clusterLabels)
	c.relocatingShards.collect(ch, float64(clusterHealth.RelocatingShards), clusterLabels)
	c.unassignedShards.collect(ch, float64(clusterHealth.UnassignedShards), clusterLabels)
	
	// 收集集群状态指标
	for _, color := range colors {
		var status float64
		if clusterHealth.Status == color {
			status = 1
		} else {
			status = 0
		}
		
		statusLabels := []string{clusterHealth.ClusterName, color}
		
		switch color {
		case "green":
			c.clusterStatusGreen.collect(ch, status, statusLabels)
		case "yellow":
			c.clusterStatusYellow.collect(ch, status, statusLabels)
		case "red":
			c.clusterStatusRed.collect(ch, status, statusLabels)
		}
	}
	
	// 转换状态为数值
	var clusterStatus float64
	switch clusterHealth.Status {
	case "green":
		clusterStatus = 0
	case "yellow":
		clusterStatus = 1
	case "red":
		clusterStatus = 2
	}
	
	// 转换布尔值为数值
	timedOut := 0.0
	if clusterHealth.TimedOut {
		timedOut = 1.0
	}
	
	// 收集集群状态指标
	c.clusterStatus.collect(ch, clusterStatus, clusterLabels)
	c.timedOut.collect(ch, timedOut, clusterLabels)
	
	// 收集Up指标
	ch <- prometheus.MustNewConstMetric(
		c.up,
		prometheus.GaugeValue,
		1,
		c.esURL,
	)
}

// Describe 实现 prometheus.Collector 接口
func (c *ClusterHealth) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.activePrimaryShards.desc
	ch <- c.activeShards.desc
	ch <- c.delayedUnassignedShards.desc
	ch <- c.initializingShards.desc
	ch <- c.numberOfDataNodes.desc
	ch <- c.numberOfInFlightFetch.desc
	ch <- c.taskMaxWaitingInQueue.desc
	ch <- c.numberOfNodes.desc
	ch <- c.numberOfPendingTasks.desc
	ch <- c.relocatingShards.desc
	ch <- c.unassignedShards.desc
	ch <- c.clusterStatusGreen.desc
	ch <- c.clusterStatusYellow.desc
	ch <- c.clusterStatusRed.desc
	ch <- c.clusterStatus.desc
	ch <- c.timedOut.desc
	ch <- c.up
	ch <- c.jsonParseFailures.Desc()
} 

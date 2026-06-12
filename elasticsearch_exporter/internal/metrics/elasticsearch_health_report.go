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

var (
	statusColors = []string{"green", "yellow", "red"}
)

// HealthReport 健康报告收集器
type HealthReport struct {
	esURL             string
	client            *http.Client
	insecure          bool
	jsonParseFailures prometheus.Counter

	// 健康报告指标
	totalRepositories          *prometheus.Desc
	maxShardsInClusterData     *prometheus.Desc
	maxShardsInClusterFrozen   *prometheus.Desc
	restartingReplicas         *prometheus.Desc
	creatingPrimaries          *prometheus.Desc
	initializingReplicas       *prometheus.Desc
	unassignedReplicas         *prometheus.Desc
	startedPrimaries           *prometheus.Desc
	restartingPrimaries        *prometheus.Desc
	initializingPrimaries      *prometheus.Desc
	creatingReplicas           *prometheus.Desc
	startedReplicas            *prometheus.Desc
	unassignedPrimaries        *prometheus.Desc
	slmPolicies                *prometheus.Desc
	ilmPolicies                *prometheus.Desc
	ilmStagnatingIndices       *prometheus.Desc
	status                     *prometheus.Desc
	masterIsStableStatus       *prometheus.Desc
	repositoryIntegrityStatus  *prometheus.Desc
	diskStatus                 *prometheus.Desc
	shardsCapacityStatus       *prometheus.Desc
	shardsAvailabilityStatus   *prometheus.Desc
	dataStreamLifecycleStatus  *prometheus.Desc
	slmStatus                  *prometheus.Desc
	ilmStatus                  *prometheus.Desc
}

// HealthReportResponse 表示健康报告响应
type HealthReportResponse struct {
	ClusterName string                 `json:"cluster_name"`
	Status      string                 `json:"status"`
	Indicators  HealthReportIndicators `json:"indicators"`
}

// HealthReportIndicators 表示健康报告指标
type HealthReportIndicators struct {
	MasterIsStable      HealthReportMasterIsStable      `json:"master_is_stable"`
	RepositoryIntegrity HealthReportRepositoryIntegrity `json:"repository_integrity"`
	Disk                HealthReportDisk                `json:"disk"`
	ShardsCapacity      HealthReportShardsCapacity      `json:"shards_capacity"`
	ShardsAvailability  HealthReportShardsAvailability  `json:"shards_availability"`
	DataStreamLifecycle HealthReportDataStreamLifecycle `json:"data_stream_lifecycle"`
	Slm                 HealthReportSlm                 `json:"slm"`
	Ilm                 HealthReportIlm                 `json:"ilm"`
}

// HealthReportMasterIsStable 表示主节点稳定性
type HealthReportMasterIsStable struct {
	Status  string                            `json:"status"`
	Symptom string                            `json:"symptom"`
	Details HealthReportMasterIsStableDetails `json:"details"`
}

// HealthReportMasterIsStableDetails 表示主节点稳定性详情
type HealthReportMasterIsStableDetails struct {
	CurrentMaster HealthReportMasterIsStableDetailsNode   `json:"current_master"`
	RecentMasters []HealthReportMasterIsStableDetailsNode `json:"recent_masters"`
}

// HealthReportMasterIsStableDetailsNode 表示主节点详情
type HealthReportMasterIsStableDetailsNode struct {
	NodeID string `json:"node_id"`
	Name   string `json:"name"`
}

// HealthReportRepositoryIntegrity 表示存储库完整性
type HealthReportRepositoryIntegrity struct {
	Status  string                                  `json:"status"`
	Symptom string                                  `json:"symptom"`
	Details HealthReportRepositoriyIntegrityDetails `json:"details"`
}

// HealthReportRepositoriyIntegrityDetails 表示存储库完整性详情
type HealthReportRepositoriyIntegrityDetails struct {
	TotalRepositories int `json:"total_repositories"`
}

// HealthReportDisk 表示磁盘状态
type HealthReportDisk struct {
	Status  string                  `json:"status"`
	Symptom string                  `json:"symptom"`
	Details HealthReportDiskDetails `json:"details"`
}

// HealthReportDiskDetails 表示磁盘状态详情
type HealthReportDiskDetails struct {
	IndicesWithReadonlyBlock     int `json:"indices_with_readonly_block"`
	NodesWithEnoughDiskSpace     int `json:"nodes_with_enough_disk_space"`
	NodesWithUnknownDiskStatus   int `json:"nodes_with_unknown_disk_status"`
	NodesOverHighWatermark       int `json:"nodes_over_high_watermark"`
	NodesOverFloodStageWatermark int `json:"nodes_over_flood_stage_watermark"`
}

// HealthReportShardsCapacity 表示分片容量
type HealthReportShardsCapacity struct {
	Status  string                            `json:"status"`
	Symptom string                            `json:"symptom"`
	Details HealthReportShardsCapacityDetails `json:"details"`
}

// HealthReportShardsCapacityDetails 表示分片容量详情
type HealthReportShardsCapacityDetails struct {
	Data   HealthReportShardsCapacityDetailsMaxShards `json:"data"`
	Frozen HealthReportShardsCapacityDetailsMaxShards `json:"frozen"`
}

// HealthReportShardsCapacityDetailsMaxShards 表示分片容量最大分片数
type HealthReportShardsCapacityDetailsMaxShards struct {
	MaxShardsInCluster int `json:"max_shards_in_cluster"`
}

// HealthReportShardsAvailability 表示分片可用性
type HealthReportShardsAvailability struct {
	Status  string                                `json:"status"`
	Symptom string                                `json:"symptom"`
	Details HealthReportShardsAvailabilityDetails `json:"details"`
}

// HealthReportShardsAvailabilityDetails 表示分片可用性详情
type HealthReportShardsAvailabilityDetails struct {
	RestartingReplicas    int `json:"restarting_replicas"`
	CreatingPrimaries     int `json:"creating_primaries"`
	InitializingReplicas  int `json:"initializing_replicas"`
	UnassignedReplicas    int `json:"unassigned_replicas"`
	StartedPrimaries      int `json:"started_primaries"`
	RestartingPrimaries   int `json:"restarting_primaries"`
	InitializingPrimaries int `json:"initializing_primaries"`
	CreatingReplicas      int `json:"creating_replicas"`
	StartedReplicas       int `json:"started_replicas"`
	UnassignedPrimaries   int `json:"unassigned_primaries"`
}

// HealthReportDataStreamLifecycle 表示数据流生命周期
type HealthReportDataStreamLifecycle struct {
	Status  string `json:"status"`
	Symptom string `json:"symptom"`
}

// HealthReportSlm 表示SLM状态
type HealthReportSlm struct {
	Status  string                 `json:"status"`
	Symptom string                 `json:"symptom"`
	Details HealthReportSlmDetails `json:"details"`
}

// HealthReportSlmDetails 表示SLM详情
type HealthReportSlmDetails struct {
	SlmStatus string `json:"slm_status"`
	Policies  int    `json:"policies"`
}

// HealthReportIlm 表示ILM状态
type HealthReportIlm struct {
	Status  string                 `json:"status"`
	Symptom string                 `json:"symptom"`
	Details HealthReportIlmDetails `json:"details"`
}

// HealthReportIlmDetails 表示ILM详情
type HealthReportIlmDetails struct {
	Policies          int    `json:"policies"`
	StagnatingIndices int    `json:"stagnating_indices"`
	IlmStatus         string `json:"ilm_status"`
}

func init() {
	exporter.Register(NewHealthReport())
}

// statusValue 计算状态值
func statusValue(value string, color string) float64 {
	if value == color {
		return 1
	}
	return 0
}

// NewHealthReport 创建健康报告收集器
func NewHealthReport() *HealthReport {
	defaultHealthReportLabels := []string{"cluster"}
	statusLabels := []string{"cluster", "color"}

	return &HealthReport{
		esURL: "http://localhost:9200",
		client: &http.Client{
			Timeout: defaultTimeout,
		},
		jsonParseFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "health_report_json_parse_failures",
			Help:      "Number of errors while parsing JSON.",
		}),

		// 健康报告指标
		totalRepositories: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "health_report", "total_repositories"),
			"The number of total repositories",
			defaultHealthReportLabels, nil,
		),
		maxShardsInClusterData: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "health_report", "max_shards_in_cluster_data"),
			"The maximum shards in data cluster",
			defaultHealthReportLabels, nil,
		),
		maxShardsInClusterFrozen: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "health_report", "max_shards_in_cluster_frozen"),
			"The maximum shards in frozen cluster",
			defaultHealthReportLabels, nil,
		),
		restartingReplicas: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "health_report", "restarting_replicas"),
			"The number of restarting replica shards",
			defaultHealthReportLabels, nil,
		),
		creatingPrimaries: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "health_report", "creating_primaries"),
			"The number of creating primary shards",
			defaultHealthReportLabels, nil,
		),
		initializingReplicas: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "health_report", "initializing_replicas"),
			"The number of initializing replica shards",
			defaultHealthReportLabels, nil,
		),
		unassignedReplicas: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "health_report", "unassigned_replicas"),
			"The number of unassigned replica shards",
			defaultHealthReportLabels, nil,
		),
		startedPrimaries: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "health_report", "started_primaries"),
			"The number of started primary shards",
			defaultHealthReportLabels, nil,
		),
		restartingPrimaries: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "health_report", "restarting_primaries"),
			"The number of restarting primary shards",
			defaultHealthReportLabels, nil,
		),
		initializingPrimaries: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "health_report", "initializing_primaries"),
			"The number of initializing primary shards",
			defaultHealthReportLabels, nil,
		),
		creatingReplicas: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "health_report", "creating_replicas"),
			"The number of creating replica shards",
			defaultHealthReportLabels, nil,
		),
		startedReplicas: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "health_report", "started_replicas"),
			"The number of started replica shards",
			defaultHealthReportLabels, nil,
		),
		unassignedPrimaries: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "health_report", "unassigned_primaries"),
			"The number of unassigned primary shards",
			defaultHealthReportLabels, nil,
		),
		slmPolicies: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "health_report", "slm_policies"),
			"The number of SLM policies",
			defaultHealthReportLabels, nil,
		),
		ilmPolicies: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "health_report", "ilm_policies"),
			"The number of ILM Policies",
			defaultHealthReportLabels, nil,
		),
		ilmStagnatingIndices: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "health_report", "ilm_stagnating_indices"),
			"The number of stagnating indices",
			defaultHealthReportLabels, nil,
		),
		status: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "health_report", "status"),
			"Overall cluster status",
			statusLabels, nil,
		),
		masterIsStableStatus: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "health_report", "master_is_stable_status"),
			"Master is stable status",
			statusLabels, nil,
		),
		repositoryIntegrityStatus: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "health_report", "repository_integrity_status"),
			"Repository integrity status",
			statusLabels, nil,
		),
		diskStatus: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "health_report", "disk_status"),
			"Disk status",
			statusLabels, nil,
		),
		shardsCapacityStatus: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "health_report", "shards_capacity_status"),
			"Shards capacity status",
			statusLabels, nil,
		),
		shardsAvailabilityStatus: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "health_report", "shards_availability_status"),
			"Shards availability status",
			statusLabels, nil,
		),
		dataStreamLifecycleStatus: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "health_report", "data_stream_lifecycle_status"),
			"Data stream lifecycle status",
			statusLabels, nil,
		),
		slmStatus: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "health_report", "slm_status"),
			"SLM status",
			statusLabels, nil,
		),
		ilmStatus: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "health_report", "ilm_status"),
			"ILM status",
			statusLabels, nil,
		),
	}
}

// fetchAndDecodeHealthReport 获取并解析健康报告
func (h *HealthReport) fetchAndDecodeHealthReport() (HealthReportResponse, error) {
	// 确保客户端每次获取时更新配置
	h.client = &http.Client{
		Timeout: defaultTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				// #nosec G402
				InsecureSkipVerify: h.insecure,
			},
		},
	}

	u, err := url.Parse(h.esURL)
	if err != nil {
		return HealthReportResponse{}, fmt.Errorf("failed to parse ES URL: %s", err)
	}

	// 构建URL路径
	u.Path = path.Join(u.Path, "/_health_report")

	logrus.Debugf("Fetching health report from %s", u.String())

	res, err := h.client.Get(u.String())
	if err != nil {
		return HealthReportResponse{}, fmt.Errorf("failed to get health report from %s: %s", u.String(), err)
	}

	defer func() {
		err = res.Body.Close()
		if err != nil {
			logrus.Warnf("failed to close http.Client: %s", err)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return HealthReportResponse{}, fmt.Errorf("HTTP Request failed with code %d", res.StatusCode)
	}

	var healthReportResp HealthReportResponse
	if err := json.NewDecoder(res.Body).Decode(&healthReportResp); err != nil {
		h.jsonParseFailures.Inc()
		return HealthReportResponse{}, err
	}

	return healthReportResp, nil
}

// Collect 实现指标收集
func (h *HealthReport) Collect(ch chan<- prometheus.Metric) {
	// 从配置中读取ES URL
	if config.ScrapeUrl != nil && *config.ScrapeUrl != "" {
		h.esURL = *config.ScrapeUrl
		logrus.Debugf("Using scrape_uri from command line: %s", h.esURL)
	} else {
		// 检查配置文件中的ScrapeUri
		var settings config.Settings
		if err := exporter.Unpack(&settings); err == nil && settings.ScrapeUri != "" {
			h.esURL = settings.ScrapeUri
			logrus.Debugf("Using scrape_uri from config file: %s", h.esURL)
		} else {
			logrus.Debugf("Using default scrape_uri: %s", h.esURL)
		}
	}

	// 设置是否忽略SSL证书验证
	if config.Insecure != nil && *config.Insecure {
		h.insecure = *config.Insecure
	} else {
		// 检查配置文件中的Insecure设置
		var settings config.Settings
		if err := exporter.Unpack(&settings); err == nil {
			h.insecure = settings.Insecure
		}
	}

	// 确保计数器被收集
	ch <- h.jsonParseFailures

	// 获取健康报告
	healthReportResp, err := h.fetchAndDecodeHealthReport()
	if err != nil {
		logrus.Warnf("Failed to fetch and decode health report: %s", err)
		return
	}

	// 收集指标
	ch <- prometheus.MustNewConstMetric(
		h.totalRepositories,
		prometheus.GaugeValue,
		float64(healthReportResp.Indicators.RepositoryIntegrity.Details.TotalRepositories),
		healthReportResp.ClusterName,
	)
	ch <- prometheus.MustNewConstMetric(
		h.maxShardsInClusterData,
		prometheus.GaugeValue,
		float64(healthReportResp.Indicators.ShardsCapacity.Details.Data.MaxShardsInCluster),
		healthReportResp.ClusterName,
	)
	ch <- prometheus.MustNewConstMetric(
		h.maxShardsInClusterFrozen,
		prometheus.GaugeValue,
		float64(healthReportResp.Indicators.ShardsCapacity.Details.Frozen.MaxShardsInCluster),
		healthReportResp.ClusterName,
	)
	ch <- prometheus.MustNewConstMetric(
		h.restartingReplicas,
		prometheus.GaugeValue,
		float64(healthReportResp.Indicators.ShardsAvailability.Details.RestartingReplicas),
		healthReportResp.ClusterName,
	)
	ch <- prometheus.MustNewConstMetric(
		h.creatingPrimaries,
		prometheus.GaugeValue,
		float64(healthReportResp.Indicators.ShardsAvailability.Details.CreatingPrimaries),
		healthReportResp.ClusterName,
	)
	ch <- prometheus.MustNewConstMetric(
		h.initializingReplicas,
		prometheus.GaugeValue,
		float64(healthReportResp.Indicators.ShardsAvailability.Details.InitializingReplicas),
		healthReportResp.ClusterName,
	)
	ch <- prometheus.MustNewConstMetric(
		h.unassignedReplicas,
		prometheus.GaugeValue,
		float64(healthReportResp.Indicators.ShardsAvailability.Details.UnassignedReplicas),
		healthReportResp.ClusterName,
	)
	ch <- prometheus.MustNewConstMetric(
		h.startedPrimaries,
		prometheus.GaugeValue,
		float64(healthReportResp.Indicators.ShardsAvailability.Details.StartedPrimaries),
		healthReportResp.ClusterName,
	)
	ch <- prometheus.MustNewConstMetric(
		h.restartingPrimaries,
		prometheus.GaugeValue,
		float64(healthReportResp.Indicators.ShardsAvailability.Details.RestartingPrimaries),
		healthReportResp.ClusterName,
	)
	ch <- prometheus.MustNewConstMetric(
		h.initializingPrimaries,
		prometheus.GaugeValue,
		float64(healthReportResp.Indicators.ShardsAvailability.Details.InitializingPrimaries),
		healthReportResp.ClusterName,
	)
	ch <- prometheus.MustNewConstMetric(
		h.creatingReplicas,
		prometheus.GaugeValue,
		float64(healthReportResp.Indicators.ShardsAvailability.Details.CreatingReplicas),
		healthReportResp.ClusterName,
	)
	ch <- prometheus.MustNewConstMetric(
		h.startedReplicas,
		prometheus.GaugeValue,
		float64(healthReportResp.Indicators.ShardsAvailability.Details.StartedReplicas),
		healthReportResp.ClusterName,
	)
	ch <- prometheus.MustNewConstMetric(
		h.unassignedPrimaries,
		prometheus.GaugeValue,
		float64(healthReportResp.Indicators.ShardsAvailability.Details.UnassignedPrimaries),
		healthReportResp.ClusterName,
	)
	ch <- prometheus.MustNewConstMetric(
		h.slmPolicies,
		prometheus.GaugeValue,
		float64(healthReportResp.Indicators.Slm.Details.Policies),
		healthReportResp.ClusterName,
	)
	ch <- prometheus.MustNewConstMetric(
		h.ilmPolicies,
		prometheus.GaugeValue,
		float64(healthReportResp.Indicators.Ilm.Details.Policies),
		healthReportResp.ClusterName,
	)
	ch <- prometheus.MustNewConstMetric(
		h.ilmStagnatingIndices,
		prometheus.GaugeValue,
		float64(healthReportResp.Indicators.Ilm.Details.StagnatingIndices),
		healthReportResp.ClusterName,
	)

	// 收集状态指标
	for _, color := range statusColors {
		ch <- prometheus.MustNewConstMetric(
			h.status,
			prometheus.GaugeValue,
			statusValue(healthReportResp.Status, color),
			healthReportResp.ClusterName, color,
		)
		ch <- prometheus.MustNewConstMetric(
			h.masterIsStableStatus,
			prometheus.GaugeValue,
			statusValue(healthReportResp.Indicators.MasterIsStable.Status, color),
			healthReportResp.ClusterName, color,
		)
		ch <- prometheus.MustNewConstMetric(
			h.repositoryIntegrityStatus,
			prometheus.GaugeValue,
			statusValue(healthReportResp.Indicators.RepositoryIntegrity.Status, color),
			healthReportResp.ClusterName, color,
		)
		ch <- prometheus.MustNewConstMetric(
			h.diskStatus,
			prometheus.GaugeValue,
			statusValue(healthReportResp.Indicators.Disk.Status, color),
			healthReportResp.ClusterName, color,
		)
		ch <- prometheus.MustNewConstMetric(
			h.shardsCapacityStatus,
			prometheus.GaugeValue,
			statusValue(healthReportResp.Indicators.ShardsCapacity.Status, color),
			healthReportResp.ClusterName, color,
		)
		ch <- prometheus.MustNewConstMetric(
			h.shardsAvailabilityStatus,
			prometheus.GaugeValue,
			statusValue(healthReportResp.Indicators.ShardsAvailability.Status, color),
			healthReportResp.ClusterName, color,
		)
		ch <- prometheus.MustNewConstMetric(
			h.dataStreamLifecycleStatus,
			prometheus.GaugeValue,
			statusValue(healthReportResp.Indicators.DataStreamLifecycle.Status, color),
			healthReportResp.ClusterName, color,
		)
		ch <- prometheus.MustNewConstMetric(
			h.slmStatus,
			prometheus.GaugeValue,
			statusValue(healthReportResp.Indicators.Slm.Status, color),
			healthReportResp.ClusterName, color,
		)
		ch <- prometheus.MustNewConstMetric(
			h.ilmStatus,
			prometheus.GaugeValue,
			statusValue(healthReportResp.Indicators.Ilm.Status, color),
			healthReportResp.ClusterName, color,
		)
	}
}

// Describe 实现指标描述
func (hr *HealthReport) Describe(ch chan<- *prometheus.Desc) {
	ch <- hr.totalRepositories
	ch <- hr.maxShardsInClusterData
	ch <- hr.maxShardsInClusterFrozen
	ch <- hr.restartingReplicas
	ch <- hr.creatingPrimaries
	ch <- hr.initializingReplicas
	ch <- hr.unassignedReplicas
	ch <- hr.startedPrimaries
	ch <- hr.restartingPrimaries
	ch <- hr.initializingPrimaries
	ch <- hr.creatingReplicas
	ch <- hr.startedReplicas
	ch <- hr.unassignedPrimaries
	ch <- hr.slmPolicies
	ch <- hr.ilmPolicies
	ch <- hr.ilmStagnatingIndices
	ch <- hr.status
	ch <- hr.masterIsStableStatus
	ch <- hr.repositoryIntegrityStatus
	ch <- hr.diskStatus
	ch <- hr.shardsCapacityStatus
	ch <- hr.shardsAvailabilityStatus
	ch <- hr.dataStreamLifecycleStatus
	ch <- hr.slmStatus
	ch <- hr.ilmStatus
	ch <- hr.jsonParseFailures.Desc()
} 

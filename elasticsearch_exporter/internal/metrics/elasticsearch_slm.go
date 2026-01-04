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

// SLMStatsResponse 表示SLM统计信息
type SLMStatsResponse struct {
	RetentionRuns                 int64         `json:"retention_runs"`
	RetentionFailed               int64         `json:"retention_failed"`
	RetentionTimedOut             int64         `json:"retention_timed_out"`
	RetentionDeletionTime         string        `json:"retention_deletion_time"`
	RetentionDeletionTimeMillis   int64         `json:"retention_deletion_time_millis"`
	TotalSnapshotsTaken           int64         `json:"total_snapshots_taken"`
	TotalSnapshotsFailed          int64         `json:"total_snapshots_failed"`
	TotalSnapshotsDeleted         int64         `json:"total_snapshots_deleted"`
	TotalSnapshotDeletionFailures int64         `json:"total_snapshot_deletion_failures"`
	PolicyStats                   []PolicyStats `json:"policy_stats"`
}

// PolicyStats 表示特定策略的SLM统计信息
type PolicyStats struct {
	Policy                   string `json:"policy"`
	SnapshotsTaken           int64  `json:"snapshots_taken"`
	SnapshotsFailed          int64  `json:"snapshots_failed"`
	SnapshotsDeleted         int64  `json:"snapshots_deleted"`
	SnapshotDeletionFailures int64  `json:"snapshot_deletion_failures"`
}

// SLMStatusResponse 表示SLM状态信息
type SLMStatusResponse struct {
	OperationMode string `json:"operation_mode"`
}

// SLM 表示SLM指标收集器
type SLM struct {
	esURL             string
	client            *http.Client
	insecure          bool
	jsonParseFailures prometheus.Counter
	statuses          []string

	// SLM统计指标
	retentionRunsTotal            *prometheus.Desc
	retentionFailedTotal          *prometheus.Desc
	retentionTimedOutTotal        *prometheus.Desc
	retentionDeletionTimeSeconds  *prometheus.Desc
	totalSnapshotsTaken           *prometheus.Desc
	totalSnapshotsFailed          *prometheus.Desc
	totalSnapshotsDeleted         *prometheus.Desc
	totalSnapshotsDeleteFailed    *prometheus.Desc
	operationMode                 *baseMetrics
	snapshotsTaken                *baseMetrics
	snapshotsFailed               *baseMetrics
	snapshotsDeleted              *baseMetrics
	snapshotsDeletionFailure      *baseMetrics
}

func init() {
	exporter.Register(NewSLM())
}

// NewSLM 创建快照生命周期管理指标收集器
func NewSLM() *SLM {
	return &SLM{
		esURL: "http://localhost:9200",
		client: &http.Client{
			Timeout: defaultTimeout,
		},
		statuses: []string{"RUNNING", "STOPPING", "STOPPED"},

		jsonParseFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "slm_json_parse_failures",
			Help:      "Number of errors while parsing JSON.",
		}),

		// SLM统计指标 - 全局计数器
		retentionRunsTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "slm_stats", "retention_runs_total"),
			"Total retention runs",
			nil, nil,
		),
		retentionFailedTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "slm_stats", "retention_failed_total"),
			"Total failed retention runs",
			nil, nil,
		),
		retentionTimedOutTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "slm_stats", "retention_timed_out_total"),
			"Total timed out retention runs",
			nil, nil,
		),
		retentionDeletionTimeSeconds: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "slm_stats", "retention_deletion_time_seconds"),
			"Retention run deletion time",
			nil, nil,
		),
		totalSnapshotsTaken: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "slm_stats", "total_snapshots_taken_total"),
			"Total snapshots taken",
			nil, nil,
		),
		totalSnapshotsFailed: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "slm_stats", "total_snapshots_failed_total"),
			"Total snapshots failed",
			nil, nil,
		),
		totalSnapshotsDeleted: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "slm_stats", "total_snapshots_deleted_total"),
			"Total snapshots deleted",
			nil, nil,
		),
		totalSnapshotsDeleteFailed: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "slm_stats", "total_snapshot_deletion_failures_total"),
			"Total snapshot deletion failures",
			nil, nil,
		),

		// 带标签的指标
		operationMode: NewMetrics(
			prometheus.BuildFQName(namespace, "slm_stats", "operation_mode"),
			"Operating status of SLM",
			[]string{"operation_mode"},
		),
		
		snapshotsTaken: NewMetrics(
			prometheus.BuildFQName(namespace, "slm_stats", "snapshots_taken_total"),
			"Total snapshots taken",
			[]string{"policy"},
		),
		
		snapshotsFailed: NewMetrics(
			prometheus.BuildFQName(namespace, "slm_stats", "snapshots_failed_total"),
			"Total snapshots failed",
			[]string{"policy"},
		),
		
		snapshotsDeleted: NewMetrics(
			prometheus.BuildFQName(namespace, "slm_stats", "snapshots_deleted_total"),
			"Total snapshots deleted",
			[]string{"policy"},
		),
		
		snapshotsDeletionFailure: NewMetrics(
			prometheus.BuildFQName(namespace, "slm_stats", "snapshot_deletion_failures_total"),
			"Total snapshot deletion failures",
			[]string{"policy"},
		),
	}
}

// Describe 实现 prometheus.Collector 接口
func (s *SLM) Describe(ch chan<- *prometheus.Desc) {
	ch <- s.retentionRunsTotal
	ch <- s.retentionFailedTotal
	ch <- s.retentionTimedOutTotal
	ch <- s.retentionDeletionTimeSeconds
	ch <- s.totalSnapshotsTaken
	ch <- s.totalSnapshotsFailed
	ch <- s.totalSnapshotsDeleted
	ch <- s.totalSnapshotsDeleteFailed
	ch <- s.operationMode.desc
	ch <- s.snapshotsTaken.desc
	ch <- s.snapshotsFailed.desc
	ch <- s.snapshotsDeleted.desc
	ch <- s.snapshotsDeletionFailure.desc
	ch <- s.jsonParseFailures.Desc()
}

// fetchAndDecodeSLMStatus 获取并解析SLM状态
func (s *SLM) fetchAndDecodeSLMStatus() (SLMStatusResponse, error) {
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
		return SLMStatusResponse{}, fmt.Errorf("failed to parse ES URL: %s", err)
	}

	// 构建URL路径
	u.Path = path.Join(u.Path, "/_slm/status")

	logrus.Debugf("Fetching SLM status from %s", u.String())

	res, err := s.client.Get(u.String())
	if err != nil {
		return SLMStatusResponse{}, fmt.Errorf("failed to get SLM status from %s: %s", u.String(), err)
	}

	defer func() {
		err = res.Body.Close()
		if err != nil {
			logrus.Warnf("failed to close http.Client: %s", err)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return SLMStatusResponse{}, fmt.Errorf("HTTP Request failed with code %d", res.StatusCode)
	}

	var slmStatusResp SLMStatusResponse
	if err := json.NewDecoder(res.Body).Decode(&slmStatusResp); err != nil {
		s.jsonParseFailures.Inc()
		return SLMStatusResponse{}, err
	}

	return slmStatusResp, nil
}

// fetchAndDecodeSLMStats 获取并解析SLM统计信息
func (s *SLM) fetchAndDecodeSLMStats() (SLMStatsResponse, error) {
	u, err := url.Parse(s.esURL)
	if err != nil {
		return SLMStatsResponse{}, fmt.Errorf("failed to parse ES URL: %s", err)
	}

	// 构建URL路径
	u.Path = path.Join(u.Path, "/_slm/stats")

	logrus.Debugf("Fetching SLM stats from %s", u.String())

	res, err := s.client.Get(u.String())
	if err != nil {
		return SLMStatsResponse{}, fmt.Errorf("failed to get SLM stats from %s: %s", u.String(), err)
	}

	defer func() {
		err = res.Body.Close()
		if err != nil {
			logrus.Warnf("failed to close http.Client: %s", err)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return SLMStatsResponse{}, fmt.Errorf("HTTP Request failed with code %d", res.StatusCode)
	}

	var slmStatsResp SLMStatsResponse
	if err := json.NewDecoder(res.Body).Decode(&slmStatsResp); err != nil {
		s.jsonParseFailures.Inc()
		return SLMStatsResponse{}, err
	}

	return slmStatsResp, nil
}

// Collect 实现指标收集
func (s *SLM) Collect(ch chan<- prometheus.Metric) {
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

	// 获取SLM状态
	slmStatusResp, err := s.fetchAndDecodeSLMStatus()
	if err != nil {
		logrus.Warnf("Failed to fetch and decode SLM status: %s", err)
		return
	}

	// 获取SLM统计信息
	slmStatsResp, err := s.fetchAndDecodeSLMStats()
	if err != nil {
		logrus.Warnf("Failed to fetch and decode SLM stats: %s", err)
		return
	}

	// 收集操作模式指标
	for _, status := range s.statuses {
		var value float64
		if slmStatusResp.OperationMode == status {
			value = 1
		}
		s.operationMode.collect(ch, value, prometheus.Labels{
			"operation_mode": status,
		})
	}

	// 收集全局计数器指标
	ch <- prometheus.MustNewConstMetric(
		s.retentionRunsTotal,
		prometheus.CounterValue,
		float64(slmStatsResp.RetentionRuns),
	)

	ch <- prometheus.MustNewConstMetric(
		s.retentionFailedTotal,
		prometheus.CounterValue,
		float64(slmStatsResp.RetentionFailed),
	)

	ch <- prometheus.MustNewConstMetric(
		s.retentionTimedOutTotal,
		prometheus.CounterValue,
		float64(slmStatsResp.RetentionTimedOut),
	)

	ch <- prometheus.MustNewConstMetric(
		s.retentionDeletionTimeSeconds,
		prometheus.GaugeValue,
		float64(slmStatsResp.RetentionDeletionTimeMillis)/1000,
	)

	ch <- prometheus.MustNewConstMetric(
		s.totalSnapshotsTaken,
		prometheus.CounterValue,
		float64(slmStatsResp.TotalSnapshotsTaken),
	)

	ch <- prometheus.MustNewConstMetric(
		s.totalSnapshotsFailed,
		prometheus.CounterValue,
		float64(slmStatsResp.TotalSnapshotsFailed),
	)

	ch <- prometheus.MustNewConstMetric(
		s.totalSnapshotsDeleted,
		prometheus.CounterValue,
		float64(slmStatsResp.TotalSnapshotsDeleted),
	)

	ch <- prometheus.MustNewConstMetric(
		s.totalSnapshotsDeleteFailed,
		prometheus.CounterValue,
		float64(slmStatsResp.TotalSnapshotDeletionFailures),
	)

	// 收集策略级别指标
	for _, policy := range slmStatsResp.PolicyStats {
		s.snapshotsTaken.collect(ch, float64(policy.SnapshotsTaken), prometheus.Labels{
			"policy": policy.Policy,
		})
		
		s.snapshotsFailed.collect(ch, float64(policy.SnapshotsFailed), prometheus.Labels{
			"policy": policy.Policy,
		})
		
		s.snapshotsDeleted.collect(ch, float64(policy.SnapshotsDeleted), prometheus.Labels{
			"policy": policy.Policy,
		})
		
		s.snapshotsDeletionFailure.collect(ch, float64(policy.SnapshotDeletionFailures), prometheus.Labels{
			"policy": policy.Policy,
		})
	}
}
// Part 2 commit for elasticsearch_exporter/internal/metrics/elasticsearch_slm.go

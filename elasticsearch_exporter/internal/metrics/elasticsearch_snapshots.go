package metrics

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"
	"elasticsearch_exporter/config"
	"elasticsearch_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// SnapshotStatsResponse 是快照统计信息的表示
type SnapshotStatsResponse struct {
	Snapshots []SnapshotStatDataResponse `json:"snapshots"`
}

// SnapshotStatDataResponse 是单个快照统计信息的表示
type SnapshotStatDataResponse struct {
	Snapshot          string        `json:"snapshot"`
	UUID              string        `json:"uuid"`
	VersionID         int64         `json:"version_id"`
	Version           string        `json:"version"`
	Indices           []string      `json:"indices"`
	State             string        `json:"state"`
	StartTime         time.Time     `json:"start_time"`
	StartTimeInMillis int64         `json:"start_time_in_millis"`
	EndTime           time.Time     `json:"end_time"`
	EndTimeInMillis   int64         `json:"end_time_in_millis"`
	DurationInMillis  int64         `json:"duration_in_millis"`
	Failures          []interface{} `json:"failures"`
	Shards            struct {
		Total      int64 `json:"total"`
		Failed     int64 `json:"failed"`
		Successful int64 `json:"successful"`
	} `json:"shards"`
}

// SnapshotRepositoriesResponse 是快照存储库的表示
type SnapshotRepositoriesResponse map[string]struct {
	Type string `json:"type"`
}

// Snapshots 快照指标收集器
type Snapshots struct {
	esURL             string
	client            *http.Client
	insecure          bool
	jsonParseFailures prometheus.Counter

	// 快照指标
	numIndices             *baseMetrics
	snapshotStartTimestamp *baseMetrics
	snapshotEndTimestamp   *baseMetrics
	snapshotNumFailures    *baseMetrics
	snapshotNumShards      *baseMetrics
	snapshotFailedShards   *baseMetrics
	snapshotSuccessShards  *baseMetrics

	// 存储库指标
	numSnapshots            *baseMetrics
	oldestSnapshotTimestamp *baseMetrics
	latestSnapshotTimestamp *baseMetrics
}

func init() {
	exporter.Register(NewSnapshots())
}

// NewSnapshots 创建快照指标收集器
func NewSnapshots() *Snapshots {
	return &Snapshots{
		esURL: "http://localhost:9200",
		client: &http.Client{
			Timeout: defaultTimeout,
		},

		jsonParseFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "snapshots_json_parse_failures",
			Help:      "Number of errors while parsing JSON.",
		}),

		// 快照指标
		numIndices: NewMetrics(
			prometheus.BuildFQName(namespace, "snapshot_stats", "snapshot_number_of_indices"),
			"Number of indices in the last snapshot",
			[]string{"repository", "state", "version"},
		),
		snapshotStartTimestamp: NewMetrics(
			prometheus.BuildFQName(namespace, "snapshot_stats", "snapshot_start_time_timestamp"),
			"Last snapshot start timestamp",
			[]string{"repository", "state", "version"},
		),
		snapshotEndTimestamp: NewMetrics(
			prometheus.BuildFQName(namespace, "snapshot_stats", "snapshot_end_time_timestamp"),
			"Last snapshot end timestamp",
			[]string{"repository", "state", "version"},
		),
		snapshotNumFailures: NewMetrics(
			prometheus.BuildFQName(namespace, "snapshot_stats", "snapshot_number_of_failures"),
			"Last snapshot number of failures",
			[]string{"repository", "state", "version"},
		),
		snapshotNumShards: NewMetrics(
			prometheus.BuildFQName(namespace, "snapshot_stats", "snapshot_total_shards"),
			"Last snapshot total shards",
			[]string{"repository", "state", "version"},
		),
		snapshotFailedShards: NewMetrics(
			prometheus.BuildFQName(namespace, "snapshot_stats", "snapshot_failed_shards"),
			"Last snapshot failed shards",
			[]string{"repository", "state", "version"},
		),
		snapshotSuccessShards: NewMetrics(
			prometheus.BuildFQName(namespace, "snapshot_stats", "snapshot_successful_shards"),
			"Last snapshot successful shards",
			[]string{"repository", "state", "version"},
		),

		// 存储库指标
		numSnapshots: NewMetrics(
			prometheus.BuildFQName(namespace, "snapshot_stats", "number_of_snapshots"),
			"Number of snapshots in a repository",
			[]string{"repository"},
		),
		oldestSnapshotTimestamp: NewMetrics(
			prometheus.BuildFQName(namespace, "snapshot_stats", "oldest_snapshot_timestamp"),
			"Timestamp of the oldest snapshot",
			[]string{"repository"},
		),
		latestSnapshotTimestamp: NewMetrics(
			prometheus.BuildFQName(namespace, "snapshot_stats", "latest_snapshot_timestamp_seconds"),
			"Timestamp of the latest SUCCESS or PARTIAL snapshot",
			[]string{"repository"},
		),
	}
}

// Describe 实现 prometheus.Collector 接口
func (s *Snapshots) Describe(ch chan<- *prometheus.Desc) {
	ch <- s.numIndices.desc
	ch <- s.snapshotStartTimestamp.desc
	ch <- s.snapshotEndTimestamp.desc
	ch <- s.snapshotNumFailures.desc
	ch <- s.snapshotNumShards.desc
	ch <- s.snapshotFailedShards.desc
	ch <- s.snapshotSuccessShards.desc
	ch <- s.numSnapshots.desc
	ch <- s.oldestSnapshotTimestamp.desc
	ch <- s.latestSnapshotTimestamp.desc
	ch <- s.jsonParseFailures.Desc()
}

// fetchAndDecodeSnapshotRepositories 获取并解析快照存储库信息
func (s *Snapshots) fetchAndDecodeSnapshotRepositories() (SnapshotRepositoriesResponse, error) {
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
		return SnapshotRepositoriesResponse{}, fmt.Errorf("failed to parse ES URL: %s", err)
	}

	// 构建URL路径
	u.Path = path.Join(u.Path, "/_snapshot")

	logrus.Debugf("Fetching snapshot repositories from %s", u.String())

	res, err := s.client.Get(u.String())
	if err != nil {
		return SnapshotRepositoriesResponse{}, fmt.Errorf("failed to get snapshot repositories from %s: %s", u.String(), err)
	}

	defer func() {
		err = res.Body.Close()
		if err != nil {
			logrus.Warnf("failed to close http.Client: %s", err)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return SnapshotRepositoriesResponse{}, fmt.Errorf("HTTP Request failed with code %d", res.StatusCode)
	}

	var data SnapshotRepositoriesResponse
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		s.jsonParseFailures.Inc()
		return SnapshotRepositoriesResponse{}, err
	}

	return data, nil
}

// fetchAndDecodeSnapshotStats 获取并解析指定存储库的快照统计信息
func (s *Snapshots) fetchAndDecodeSnapshotStats(repository string) (SnapshotStatsResponse, error) {
	u, err := url.Parse(s.esURL)
	if err != nil {
		return SnapshotStatsResponse{}, fmt.Errorf("failed to parse ES URL: %s", err)
	}

	// 构建URL路径
	u.Path = path.Join(u.Path, "/_snapshot", repository, "/_all")

	logrus.Debugf("Fetching snapshots from %s", u.String())

	res, err := s.client.Get(u.String())
	if err != nil {
		return SnapshotStatsResponse{}, fmt.Errorf("failed to get snapshots from %s: %s", u.String(), err)
	}

	defer func() {
		err = res.Body.Close()
		if err != nil {
			logrus.Warnf("failed to close http.Client: %s", err)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return SnapshotStatsResponse{}, fmt.Errorf("HTTP Request failed with code %d", res.StatusCode)
	}

	var data SnapshotStatsResponse
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		s.jsonParseFailures.Inc()
		return SnapshotStatsResponse{}, err
	}

	return data, nil
}

// Collect 实现指标收集
func (s *Snapshots) Collect(ch chan<- prometheus.Metric) {
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

	// 获取快照存储库信息
	repositories, err := s.fetchAndDecodeSnapshotRepositories()
	if err != nil {
		logrus.Warnf("Failed to fetch and decode snapshot repositories: %s", err)
		return
	}

	// 对每个存储库获取快照统计信息
	snapshotsStatsResp := make(map[string]SnapshotStatsResponse)
	for repository := range repositories {
		snapshotStats, err := s.fetchAndDecodeSnapshotStats(repository)
		if err != nil {
			logrus.Warnf("Failed to fetch and decode snapshots for repository %s: %s", repository, err)
			continue
		}
		snapshotsStatsResp[repository] = snapshotStats
	}

	// 收集快照指标
	for repositoryName, snapshotStats := range snapshotsStatsResp {
		// 存储库指标
		s.numSnapshots.collect(ch, float64(len(snapshotStats.Snapshots)), prometheus.Labels{"repository": repositoryName})

		// 最旧快照时间戳
		oldest := float64(0)
		if len(snapshotStats.Snapshots) > 0 {
			oldest = float64(snapshotStats.Snapshots[0].StartTimeInMillis / 1000)
		}
		s.oldestSnapshotTimestamp.collect(ch, oldest, prometheus.Labels{"repository": repositoryName})

		// 最新成功或部分成功快照时间戳
		latest := float64(0)
		for i := len(snapshotStats.Snapshots) - 1; i >= 0; i-- {
			snap := snapshotStats.Snapshots[i]
			if snap.State == "SUCCESS" || snap.State == "PARTIAL" {
				latest = float64(snap.StartTimeInMillis / 1000)
				break
			}
		}
		s.latestSnapshotTimestamp.collect(ch, latest, prometheus.Labels{"repository": repositoryName})

		// 如果没有快照，跳过
		if len(snapshotStats.Snapshots) == 0 {
			continue
		}

		// 收集最后一个快照的指标
		lastSnapshot := snapshotStats.Snapshots[len(snapshotStats.Snapshots)-1]
		labels := prometheus.Labels{
			"repository": repositoryName,
			"state":      lastSnapshot.State,
			"version":    lastSnapshot.Version,
		}

		s.numIndices.collect(ch, float64(len(lastSnapshot.Indices)), labels)
		s.snapshotStartTimestamp.collect(ch, float64(lastSnapshot.StartTimeInMillis/1000), labels)
		s.snapshotEndTimestamp.collect(ch, float64(lastSnapshot.EndTimeInMillis/1000), labels)
		s.snapshotNumFailures.collect(ch, float64(len(lastSnapshot.Failures)), labels)
		s.snapshotNumShards.collect(ch, float64(lastSnapshot.Shards.Total), labels)
		s.snapshotFailedShards.collect(ch, float64(lastSnapshot.Shards.Failed), labels)
		s.snapshotSuccessShards.collect(ch, float64(lastSnapshot.Shards.Successful), labels)
	}
} 

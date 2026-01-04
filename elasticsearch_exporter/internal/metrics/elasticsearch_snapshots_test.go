package metrics

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSnapshots(t *testing.T) {
	snapshots := NewSnapshots()
	assert.NotNil(t, snapshots)
	assert.Equal(t, "http://localhost:9200", snapshots.esURL)
	assert.NotNil(t, snapshots.client)
	assert.NotNil(t, snapshots.jsonParseFailures)
	
	// 测试快照指标
	assert.NotNil(t, snapshots.numIndices)
	assert.NotNil(t, snapshots.snapshotStartTimestamp)
	assert.NotNil(t, snapshots.snapshotEndTimestamp)
	assert.NotNil(t, snapshots.snapshotNumFailures)
	assert.NotNil(t, snapshots.snapshotNumShards)
	assert.NotNil(t, snapshots.snapshotFailedShards)
	assert.NotNil(t, snapshots.snapshotSuccessShards)
	
	// 测试存储库指标
	assert.NotNil(t, snapshots.numSnapshots)
	assert.NotNil(t, snapshots.oldestSnapshotTimestamp)
	assert.NotNil(t, snapshots.latestSnapshotTimestamp)
}

func TestSnapshotsFetchAndDecodeSnapshotRepositories(t *testing.T) {
	// 创建模拟响应
	mockRepositories := map[string]struct {
		Type string `json:"type"`
	}{
		"my_backup": {
			Type: "fs",
		},
		"daily_backup": {
			Type: "s3",
		},
	}

	// 创建模拟服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/_snapshot") && !strings.Contains(r.URL.Path, "/_all") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockRepositories)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// 创建Snapshots并设置URL
	snapshots := NewSnapshots()
	snapshots.esURL = mockServer.URL

	// 测试获取和解析快照存储库信息
	repos, err := snapshots.fetchAndDecodeSnapshotRepositories()
	require.NoError(t, err)
	assert.Len(t, repos, 2)
	assert.Contains(t, repos, "my_backup")
	assert.Contains(t, repos, "daily_backup")
	assert.Equal(t, "fs", repos["my_backup"].Type)
	assert.Equal(t, "s3", repos["daily_backup"].Type)
}

func TestSnapshotsFetchAndDecodeSnapshotStats(t *testing.T) {
	now := time.Now()
	startTime := now.Add(-1 * time.Hour)
	endTime := now.Add(-30 * time.Minute)
	startTimeMillis := startTime.UnixNano() / int64(time.Millisecond)
	endTimeMillis := endTime.UnixNano() / int64(time.Millisecond)
	durationMillis := endTimeMillis - startTimeMillis

	// 创建模拟快照响应
	mockResponse := SnapshotStatsResponse{
		Snapshots: []SnapshotStatDataResponse{
			{
				Snapshot:          "snapshot_1",
				UUID:              "snap_uuid_1",
				VersionID:         7050399,
				Version:           "7.5.0",
				Indices:           []string{"index1", "index2"},
				State:             "SUCCESS",
				StartTime:         startTime,
				StartTimeInMillis: startTimeMillis,
				EndTime:           endTime,
				EndTimeInMillis:   endTimeMillis,
				DurationInMillis:  durationMillis,
				Failures:          []interface{}{},
				Shards: struct {
					Total      int64 `json:"total"`
					Failed     int64 `json:"failed"`
					Successful int64 `json:"successful"`
				}{
					Total:      10,
					Failed:     0,
					Successful: 10,
				},
			},
			{
				Snapshot:          "snapshot_2",
				UUID:              "snap_uuid_2",
				VersionID:         7050399,
				Version:           "7.5.0",
				Indices:           []string{"index1", "index2", "index3"},
				State:             "PARTIAL",
				StartTime:         now.Add(-30 * time.Minute),
				StartTimeInMillis: now.Add(-30 * time.Minute).UnixNano() / int64(time.Millisecond),
				EndTime:           now.Add(-15 * time.Minute),
				EndTimeInMillis:   now.Add(-15 * time.Minute).UnixNano() / int64(time.Millisecond),
				DurationInMillis:  15 * 60 * 1000,
				Failures:          []interface{}{"failure1"},
				Shards: struct {
					Total      int64 `json:"total"`
					Failed     int64 `json:"failed"`
					Successful int64 `json:"successful"`
				}{
					Total:      15,
					Failed:     3,
					Successful: 12,
				},
			},
		},
	}

	// 创建模拟服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/_snapshot/my_backup/_all") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// 创建Snapshots并设置URL
	snapshots := NewSnapshots()
	snapshots.esURL = mockServer.URL

	// 测试获取和解析特定存储库的快照统计信息
	stats, err := snapshots.fetchAndDecodeSnapshotStats("my_backup")
	require.NoError(t, err)
	assert.Len(t, stats.Snapshots, 2)

	// 验证第一个快照
	assert.Equal(t, "snapshot_1", stats.Snapshots[0].Snapshot)
	assert.Equal(t, "snap_uuid_1", stats.Snapshots[0].UUID)
	assert.Equal(t, int64(7050399), stats.Snapshots[0].VersionID)
	assert.Equal(t, "7.5.0", stats.Snapshots[0].Version)
	assert.Len(t, stats.Snapshots[0].Indices, 2)
	assert.Equal(t, "SUCCESS", stats.Snapshots[0].State)
	assert.Equal(t, startTimeMillis, stats.Snapshots[0].StartTimeInMillis)
	assert.Equal(t, endTimeMillis, stats.Snapshots[0].EndTimeInMillis)
	assert.Equal(t, durationMillis, stats.Snapshots[0].DurationInMillis)
	assert.Empty(t, stats.Snapshots[0].Failures)
	assert.Equal(t, int64(10), stats.Snapshots[0].Shards.Total)
	assert.Equal(t, int64(0), stats.Snapshots[0].Shards.Failed)
	assert.Equal(t, int64(10), stats.Snapshots[0].Shards.Successful)

	// 验证第二个快照
	assert.Equal(t, "snapshot_2", stats.Snapshots[1].Snapshot)
	assert.Equal(t, "PARTIAL", stats.Snapshots[1].State)
	assert.Len(t, stats.Snapshots[1].Indices, 3)
	assert.Len(t, stats.Snapshots[1].Failures, 1)
	assert.Equal(t, int64(15), stats.Snapshots[1].Shards.Total)
	assert.Equal(t, int64(3), stats.Snapshots[1].Shards.Failed)
	assert.Equal(t, int64(12), stats.Snapshots[1].Shards.Successful)
}

func TestSnapshotsCollect(t *testing.T) {
	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)
	thirtyMinutesAgo := now.Add(-30 * time.Minute)
	fifteenMinutesAgo := now.Add(-15 * time.Minute)

	// 创建模拟存储库响应
	mockRepositories := map[string]struct {
		Type string `json:"type"`
	}{
		"my_backup": {
			Type: "fs",
		},
	}

	// 创建模拟快照响应
	mockSnapshots := SnapshotStatsResponse{
		Snapshots: []SnapshotStatDataResponse{
			{
				Snapshot:          "snapshot_1",
				UUID:              "snap_uuid_1",
				VersionID:         7050399,
				Version:           "7.5.0",
				Indices:           []string{"index1", "index2"},
				State:             "SUCCESS",
				StartTime:         oneHourAgo,
				StartTimeInMillis: oneHourAgo.UnixNano() / int64(time.Millisecond),
				EndTime:           thirtyMinutesAgo,
				EndTimeInMillis:   thirtyMinutesAgo.UnixNano() / int64(time.Millisecond),
				DurationInMillis:  30 * 60 * 1000,
				Failures:          []interface{}{},
				Shards: struct {
					Total      int64 `json:"total"`
					Failed     int64 `json:"failed"`
					Successful int64 `json:"successful"`
				}{
					Total:      10,
					Failed:     0,
					Successful: 10,
				},
			},
			{
				Snapshot:          "snapshot_2",
				UUID:              "snap_uuid_2",
				VersionID:         7050399,
				Version:           "7.5.0",
				Indices:           []string{"index1", "index2", "index3"},
				State:             "PARTIAL",
				StartTime:         thirtyMinutesAgo,
				StartTimeInMillis: thirtyMinutesAgo.UnixNano() / int64(time.Millisecond),
				EndTime:           fifteenMinutesAgo,
				EndTimeInMillis:   fifteenMinutesAgo.UnixNano() / int64(time.Millisecond),
				DurationInMillis:  15 * 60 * 1000,
				Failures:          []interface{}{"failure1"},
				Shards: struct {
					Total      int64 `json:"total"`
					Failed     int64 `json:"failed"`
					Successful int64 `json:"successful"`
				}{
					Total:      15,
					Failed:     3,
					Successful: 12,
				},
			},
		},
	}

	// 创建模拟服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/_snapshot" {
			json.NewEncoder(w).Encode(mockRepositories)
		} else if strings.Contains(r.URL.Path, "/_snapshot/my_backup/_all") {
			json.NewEncoder(w).Encode(mockSnapshots)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// 创建Snapshots并设置URL
	snapshots := NewSnapshots()
	snapshots.esURL = mockServer.URL

	// 收集指标
	metrics := make(chan prometheus.Metric, 15)
	snapshots.Collect(metrics)
	close(metrics)

	// 验证收集的指标
	expectedMetrics := map[string]float64{
		"number_of_snapshots":         2.0,
		"oldest_snapshot_timestamp":   float64(oneHourAgo.UnixNano() / int64(time.Millisecond) / 1000),
		"latest_snapshot_timestamp":   float64(thirtyMinutesAgo.UnixNano() / int64(time.Millisecond) / 1000),
		"snapshot_number_of_indices":  3.0, // 最后一个快照的索引数
		"snapshot_start_time":         float64(thirtyMinutesAgo.UnixNano() / int64(time.Millisecond) / 1000),
		"snapshot_end_time":           float64(fifteenMinutesAgo.UnixNano() / int64(time.Millisecond) / 1000),
		"snapshot_number_of_failures": 1.0,
		"snapshot_total_shards":       15.0,
		"snapshot_failed_shards":      3.0,
		"snapshot_successful_shards":  12.0,
	}

	foundMetrics := make(map[string]bool)
	
	for metric := range metrics {
		var m dto.Metric
		err := metric.Write(&m)
		require.NoError(t, err)
		
		desc := metric.Desc().String()
		
		// 检查存储库级别的指标
		if strings.Contains(desc, "number_of_snapshots") {
			foundMetrics["number_of_snapshots"] = true
			assert.Equal(t, 2.0, *m.Gauge.Value)
		} else if strings.Contains(desc, "oldest_snapshot_timestamp") {
			foundMetrics["oldest_snapshot_timestamp"] = true
			assert.InDelta(t, expectedMetrics["oldest_snapshot_timestamp"], *m.Gauge.Value, 1.0)
		} else if strings.Contains(desc, "latest_snapshot_timestamp") {
			foundMetrics["latest_snapshot_timestamp"] = true
			assert.InDelta(t, expectedMetrics["latest_snapshot_timestamp"], *m.Gauge.Value, 1.0)
		}
		
		// 检查最后一个快照的指标
		if strings.Contains(desc, "snapshot_number_of_indices") {
			foundMetrics["snapshot_number_of_indices"] = true
			assert.Equal(t, 3.0, *m.Gauge.Value)
		} else if strings.Contains(desc, "snapshot_start_time_timestamp") {
			foundMetrics["snapshot_start_time"] = true
			assert.InDelta(t, expectedMetrics["snapshot_start_time"], *m.Gauge.Value, 1.0)
		} else if strings.Contains(desc, "snapshot_end_time_timestamp") {
			foundMetrics["snapshot_end_time"] = true
			assert.InDelta(t, expectedMetrics["snapshot_end_time"], *m.Gauge.Value, 1.0)
		} else if strings.Contains(desc, "snapshot_number_of_failures") {
			foundMetrics["snapshot_number_of_failures"] = true
			assert.Equal(t, 1.0, *m.Gauge.Value)
		} else if strings.Contains(desc, "snapshot_total_shards") {
			foundMetrics["snapshot_total_shards"] = true
			assert.Equal(t, 15.0, *m.Gauge.Value)
		} else if strings.Contains(desc, "snapshot_failed_shards") {
			foundMetrics["snapshot_failed_shards"] = true
			assert.Equal(t, 3.0, *m.Gauge.Value)
		} else if strings.Contains(desc, "snapshot_successful_shards") {
			foundMetrics["snapshot_successful_shards"] = true
			assert.Equal(t, 12.0, *m.Gauge.Value)
		}
	}
	
	// 验证所有预期指标都被找到
	for name := range expectedMetrics {
		assert.True(t, foundMetrics[name], "Expected to find metric: %s", name)
	}
}

func TestSnapshotsCollectWithServerError(t *testing.T) {
	// 创建返回错误的模拟服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	// 创建Snapshots并设置URL
	snapshots := NewSnapshots()
	snapshots.esURL = mockServer.URL

	// 收集指标
	metrics := make(chan prometheus.Metric, 5)
	snapshots.Collect(metrics)
	close(metrics)

	// 验证只收集到了解析失败计数器
	var foundCounter bool
	for metric := range metrics {
		var m dto.Metric
		err := metric.Write(&m)
		require.NoError(t, err)
		
		desc := metric.Desc().String()
		if strings.Contains(desc, "elasticsearch_snapshots_json_parse_failures") {
			foundCounter = true
		} else {
			// 不应有其他指标
			assert.Fail(t, "Unexpected metric: "+desc)
		}
	}
	
	assert.True(t, foundCounter, "Expected to find json parse failures counter")
}

func TestSnapshotsCollectWithEmptyRepository(t *testing.T) {
	// 创建模拟存储库响应
	mockRepositories := map[string]struct {
		Type string `json:"type"`
	}{
		"my_backup": {
			Type: "fs",
		},
	}

	// 创建空快照响应
	mockSnapshots := SnapshotStatsResponse{
		Snapshots: []SnapshotStatDataResponse{},
	}

	// 创建模拟服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/_snapshot" {
			json.NewEncoder(w).Encode(mockRepositories)
		} else if strings.Contains(r.URL.Path, "/_snapshot/my_backup/_all") {
			json.NewEncoder(w).Encode(mockSnapshots)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// 创建Snapshots并设置URL
	snapshots := NewSnapshots()
	snapshots.esURL = mockServer.URL

	// 收集指标
	metrics := make(chan prometheus.Metric, 5)
	snapshots.Collect(metrics)
	close(metrics)

	// 验证收集的指标
	var foundNumSnapshots, foundOldest, foundLatest bool
	for metric := range metrics {
		var m dto.Metric
		err := metric.Write(&m)
		require.NoError(t, err)
		
		desc := metric.Desc().String()
		
		if strings.Contains(desc, "number_of_snapshots") {
			foundNumSnapshots = true
			assert.Equal(t, 0.0, *m.Gauge.Value)
		} else if strings.Contains(desc, "oldest_snapshot_timestamp") {
			foundOldest = true
			assert.Equal(t, 0.0, *m.Gauge.Value)
		} else if strings.Contains(desc, "latest_snapshot_timestamp") {
			foundLatest = true
			assert.Equal(t, 0.0, *m.Gauge.Value)
		} else if strings.Contains(desc, "snapshot_number_of_indices") ||
			strings.Contains(desc, "snapshot_start_time_timestamp") ||
			strings.Contains(desc, "snapshot_end_time_timestamp") ||
			strings.Contains(desc, "snapshot_number_of_failures") ||
			strings.Contains(desc, "snapshot_total_shards") ||
			strings.Contains(desc, "snapshot_failed_shards") ||
			strings.Contains(desc, "snapshot_successful_shards") {
			assert.Fail(t, "Should not find snapshot metrics for empty repository")
		}
	}
	
	assert.True(t, foundNumSnapshots, "Expected to find number_of_snapshots metric")
	assert.True(t, foundOldest, "Expected to find oldest_snapshot_timestamp metric")
	assert.True(t, foundLatest, "Expected to find latest_snapshot_timestamp metric")
}

func TestSnapshotsCollectWithInvalidJSON(t *testing.T) {
	// 创建返回无效JSON的模拟服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer mockServer.Close()

	// 创建Snapshots并设置URL
	snapshots := NewSnapshots()
	snapshots.esURL = mockServer.URL

	// 收集指标
	metrics := make(chan prometheus.Metric, 5)
	snapshots.Collect(metrics)
	close(metrics)

	// 验证解析失败计数器被增加
	var foundCounter bool
	for metric := range metrics {
		var m dto.Metric
		err := metric.Write(&m)
		require.NoError(t, err)
		
		desc := metric.Desc().String()
		if strings.Contains(desc, "elasticsearch_snapshots_json_parse_failures") {
			foundCounter = true
			assert.Equal(t, 1.0, *m.Counter.Value)
		}
	}
	
	assert.True(t, foundCounter, "Expected to find json parse failures counter")
}
// Part 2 commit for elasticsearch_exporter/internal/metrics/elasticsearch_snapshots_test.go

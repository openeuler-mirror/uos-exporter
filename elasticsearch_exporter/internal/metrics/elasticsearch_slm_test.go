package metrics

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSLM(t *testing.T) {
	slm := NewSLM()
	assert.NotNil(t, slm)
	assert.Equal(t, "http://localhost:9200", slm.esURL)
	assert.NotNil(t, slm.client)
	assert.NotNil(t, slm.jsonParseFailures)
	assert.NotNil(t, slm.retentionRunsTotal)
	assert.NotNil(t, slm.retentionFailedTotal)
	assert.NotNil(t, slm.retentionTimedOutTotal)
	assert.NotNil(t, slm.retentionDeletionTimeSeconds)
	assert.NotNil(t, slm.totalSnapshotsTaken)
	assert.NotNil(t, slm.totalSnapshotsFailed)
	assert.NotNil(t, slm.totalSnapshotsDeleted)
	assert.NotNil(t, slm.totalSnapshotsDeleteFailed)
	assert.NotNil(t, slm.operationMode)
	assert.NotNil(t, slm.snapshotsTaken)
	assert.NotNil(t, slm.snapshotsFailed)
	assert.NotNil(t, slm.snapshotsDeleted)
	assert.NotNil(t, slm.snapshotsDeletionFailure)

	assert.Len(t, slm.statuses, 3)
	assert.Contains(t, slm.statuses, "RUNNING")
	assert.Contains(t, slm.statuses, "STOPPING")
	assert.Contains(t, slm.statuses, "STOPPED")
}

func TestSLMFetchAndDecodeSLMStatus(t *testing.T) {
	// 创建模拟响应
	mockResponse := SLMStatusResponse{
		OperationMode: "RUNNING",
	}

	// 创建模拟服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/_slm/status") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// 创建SLM并设置URL
	slm := NewSLM()
	slm.esURL = mockServer.URL

	// 测试获取和解析SLM状态
	response, err := slm.fetchAndDecodeSLMStatus()
	require.NoError(t, err)
	assert.Equal(t, "RUNNING", response.OperationMode)
}

func TestSLMFetchAndDecodeSLMStats(t *testing.T) {
	// 创建模拟响应
	mockResponse := SLMStatsResponse{
		RetentionRuns:                 10,
		RetentionFailed:               1,
		RetentionTimedOut:             2,
		RetentionDeletionTime:         "10s",
		RetentionDeletionTimeMillis:   10000,
		TotalSnapshotsTaken:           50,
		TotalSnapshotsFailed:          5,
		TotalSnapshotsDeleted:         20,
		TotalSnapshotDeletionFailures: 2,
		PolicyStats: []PolicyStats{
			{
				Policy:                   "daily-snapshots",
				SnapshotsTaken:           30,
				SnapshotsFailed:          3,
				SnapshotsDeleted:         15,
				SnapshotDeletionFailures: 1,
			},
			{
				Policy:                   "weekly-snapshots",
				SnapshotsTaken:           20,
				SnapshotsFailed:          2,
				SnapshotsDeleted:         5,
				SnapshotDeletionFailures: 1,
			},
		},
	}

	// 创建模拟服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/_slm/stats") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// 创建SLM并设置URL
	slm := NewSLM()
	slm.esURL = mockServer.URL

	// 测试获取和解析SLM统计信息
	response, err := slm.fetchAndDecodeSLMStats()
	require.NoError(t, err)
	
	assert.Equal(t, int64(10), response.RetentionRuns)
	assert.Equal(t, int64(1), response.RetentionFailed)
	assert.Equal(t, int64(2), response.RetentionTimedOut)
	assert.Equal(t, "10s", response.RetentionDeletionTime)
	assert.Equal(t, int64(10000), response.RetentionDeletionTimeMillis)
	assert.Equal(t, int64(50), response.TotalSnapshotsTaken)
	assert.Equal(t, int64(5), response.TotalSnapshotsFailed)
	assert.Equal(t, int64(20), response.TotalSnapshotsDeleted)
	assert.Equal(t, int64(2), response.TotalSnapshotDeletionFailures)
	
	assert.Len(t, response.PolicyStats, 2)
	
	// 验证第一个策略数据
	assert.Equal(t, "daily-snapshots", response.PolicyStats[0].Policy)
	assert.Equal(t, int64(30), response.PolicyStats[0].SnapshotsTaken)
	assert.Equal(t, int64(3), response.PolicyStats[0].SnapshotsFailed)
	assert.Equal(t, int64(15), response.PolicyStats[0].SnapshotsDeleted)
	assert.Equal(t, int64(1), response.PolicyStats[0].SnapshotDeletionFailures)
	
	// 验证第二个策略数据
	assert.Equal(t, "weekly-snapshots", response.PolicyStats[1].Policy)
	assert.Equal(t, int64(20), response.PolicyStats[1].SnapshotsTaken)
	assert.Equal(t, int64(2), response.PolicyStats[1].SnapshotsFailed)
	assert.Equal(t, int64(5), response.PolicyStats[1].SnapshotsDeleted)
	assert.Equal(t, int64(1), response.PolicyStats[1].SnapshotDeletionFailures)
}

func TestSLMCollect(t *testing.T) {
	// 创建模拟状态响应
	statusResponse := SLMStatusResponse{
		OperationMode: "RUNNING",
	}
	
	// 创建模拟统计响应
	statsResponse := SLMStatsResponse{
		RetentionRuns:                 10,
		RetentionFailed:               1,
		RetentionTimedOut:             2,
		RetentionDeletionTime:         "10s",
		RetentionDeletionTimeMillis:   10000,
		TotalSnapshotsTaken:           50,
		TotalSnapshotsFailed:          5,
		TotalSnapshotsDeleted:         20,
		TotalSnapshotDeletionFailures: 2,
		PolicyStats: []PolicyStats{
			{
				Policy:                   "daily-snapshots",
				SnapshotsTaken:           30,
				SnapshotsFailed:          3,
				SnapshotsDeleted:         15,
				SnapshotDeletionFailures: 1,
			},
			{
				Policy:                   "weekly-snapshots",
				SnapshotsTaken:           20,
				SnapshotsFailed:          2,
				SnapshotsDeleted:         5,
				SnapshotDeletionFailures: 1,
			},
		},
	}

	// 创建模拟服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/_slm/status") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(statusResponse)
		} else if strings.Contains(r.URL.Path, "/_slm/stats") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(statsResponse)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// 创建SLM并设置URL
	slm := NewSLM()
	slm.esURL = mockServer.URL

	// 收集指标
	metrics := make(chan prometheus.Metric, 30)
	slm.Collect(metrics)
	close(metrics)

	// 验证收集的指标
	foundMetrics := make(map[string]float64)
	foundLabels := make(map[string]map[string]string)
	
	for metric := range metrics {
		var m dto.Metric
		err := metric.Write(&m)
		require.NoError(t, err)
		
		desc := metric.Desc().String()
		
		// 处理计数器指标
		if strings.Contains(desc, "retention_runs_total") {
			foundMetrics["retention_runs_total"] = *m.Counter.Value
		} else if strings.Contains(desc, "retention_failed_total") {
			foundMetrics["retention_failed_total"] = *m.Counter.Value
		} else if strings.Contains(desc, "retention_timed_out_total") {
			foundMetrics["retention_timed_out_total"] = *m.Counter.Value
		} else if strings.Contains(desc, "retention_deletion_time_seconds") {
			foundMetrics["retention_deletion_time_seconds"] = *m.Gauge.Value
		} else if strings.Contains(desc, "total_snapshots_taken_total") && !strings.Contains(desc, "policy") {
			foundMetrics["total_snapshots_taken_total"] = *m.Counter.Value
		} else if strings.Contains(desc, "total_snapshots_failed_total") && !strings.Contains(desc, "policy") {
			foundMetrics["total_snapshots_failed_total"] = *m.Counter.Value
		} else if strings.Contains(desc, "total_snapshots_deleted_total") && !strings.Contains(desc, "policy") {
			foundMetrics["total_snapshots_deleted_total"] = *m.Counter.Value
		} else if strings.Contains(desc, "total_snapshot_deletion_failures_total") && !strings.Contains(desc, "policy") {
			foundMetrics["total_snapshot_deletion_failures_total"] = *m.Counter.Value
		} else if strings.Contains(desc, "operation_mode") {
			// 解析操作模式标签
			var opMode string
			var value float64
			
			for _, label := range m.Label {
				if *label.Name == "operation_mode" {
					opMode = *label.Value
					break
				}
			}
			
			value = *m.Gauge.Value
			
			if foundLabels["operation_mode"] == nil {
				foundLabels["operation_mode"] = make(map[string]string)
			}
			
			foundLabels["operation_mode"][opMode] = fmt.Sprintf("%.1f", value)
		} else if strings.Contains(desc, "snapshots_taken_total") && strings.Contains(desc, "policy") {
			// 解析策略标签
			var policy string
			var value float64
			
			for _, label := range m.Label {
				if *label.Name == "policy" {
					policy = *label.Value
					break
				}
			}
			
			value = *m.Gauge.Value
			
			if foundLabels["snapshots_taken"] == nil {
				foundLabels["snapshots_taken"] = make(map[string]string)
			}
			
			foundLabels["snapshots_taken"][policy] = fmt.Sprintf("%.1f", value)
		}
		// 其他标签指标处理类似...
	}
	
	// 验证全局计数器
	assert.Equal(t, float64(10), foundMetrics["retention_runs_total"])
	assert.Equal(t, float64(1), foundMetrics["retention_failed_total"])
	assert.Equal(t, float64(2), foundMetrics["retention_timed_out_total"])
	assert.Equal(t, float64(10), foundMetrics["retention_deletion_time_seconds"]) // 10000 / 1000
	assert.Equal(t, float64(50), foundMetrics["total_snapshots_taken_total"])
	assert.Equal(t, float64(5), foundMetrics["total_snapshots_failed_total"])
	assert.Equal(t, float64(20), foundMetrics["total_snapshots_deleted_total"])
	assert.Equal(t, float64(2), foundMetrics["total_snapshot_deletion_failures_total"])
	
	// 验证标签指标
	if op, ok := foundLabels["operation_mode"]; ok {
		assert.Equal(t, "1.0", op["RUNNING"])
		assert.Equal(t, "0.0", op["STOPPED"])
		assert.Equal(t, "0.0", op["STOPPING"])
	} else {
		assert.Fail(t, "Missing operation_mode metrics")
	}
	
	if snapsTaken, ok := foundLabels["snapshots_taken"]; ok {
		assert.Equal(t, "30.0", snapsTaken["daily-snapshots"])
		assert.Equal(t, "20.0", snapsTaken["weekly-snapshots"])
	} else {
		assert.Fail(t, "Missing snapshots_taken metrics")
	}
}

func TestSLMCollectWithServerError(t *testing.T) {
	// 创建返回错误的模拟服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	// 创建SLM并设置URL
	slm := NewSLM()
	slm.esURL = mockServer.URL

	// 收集指标
	metrics := make(chan prometheus.Metric, 5)
	slm.Collect(metrics)
	close(metrics)

	// 验证只收集到了解析失败计数器
	var foundCounter bool
	for metric := range metrics {
		var m dto.Metric
		err := metric.Write(&m)
		require.NoError(t, err)

		desc := metric.Desc().String()
		if strings.Contains(desc, "elasticsearch_slm_json_parse_failures") {
			foundCounter = true
		} else {
			// 不应有其他指标
			assert.Fail(t, "Unexpected metric: "+desc)
		}
	}

	assert.True(t, foundCounter, "Expected to find json parse failures counter")
}

func TestSLMCollectWithInvalidJSON(t *testing.T) {
	// 创建返回无效JSON的模拟服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/_slm/status") {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("invalid json"))
		} else {
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// 创建SLM并设置URL
	slm := NewSLM()
	slm.esURL = mockServer.URL

	// 收集指标
	metrics := make(chan prometheus.Metric, 5)
	slm.Collect(metrics)
	close(metrics)

	// 验证解析失败计数器被增加
	var foundCounter bool
	for metric := range metrics {
		var m dto.Metric
		err := metric.Write(&m)
		require.NoError(t, err)

		desc := metric.Desc().String()
		if strings.Contains(desc, "elasticsearch_slm_json_parse_failures") {
			foundCounter = true
			assert.Equal(t, 1.0, *m.Counter.Value)
		}
	}

	assert.True(t, foundCounter, "Expected to find json parse failures counter")
}
// Part 2 commit for elasticsearch_exporter/internal/metrics/elasticsearch_slm_test.go

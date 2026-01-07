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

func TestNewClusterHealth(t *testing.T) {
	ch := NewClusterHealth()
	assert.NotNil(t, ch)
	assert.Equal(t, "http://localhost:9200", ch.esURL)
	assert.NotNil(t, ch.client)
	assert.NotNil(t, ch.activePrimaryShards)
	assert.NotNil(t, ch.activeShards)
	assert.NotNil(t, ch.delayedUnassignedShards)
	assert.NotNil(t, ch.initializingShards)
	assert.NotNil(t, ch.numberOfDataNodes)
	assert.NotNil(t, ch.up)
}

func TestClusterHealthFetchAndDecodeClusterHealth(t *testing.T) {
	// 创建一个模拟的ES服务器
	mockResponse := clusterHealthResponse{
		ClusterName:             "test-cluster",
		Status:                  "green",
		TimedOut:                false,
		NumberOfNodes:           3,
		NumberOfDataNodes:       2,
		ActivePrimaryShards:     5,
		ActiveShards:            10,
		RelocatingShards:        0,
		InitializingShards:      0,
		UnassignedShards:        0,
		DelayedUnassignedShards: 0,
		NumberOfPendingTasks:    0,
		NumberOfInFlightFetch:   0,
		TaskMaxWaitingInQueueMillis: 0,
		ActiveShardsPercentAsNumber: 100.0,
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求路径
		if strings.Contains(r.URL.Path, "/_cluster/health") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// 创建ClusterHealth实例并设置URL为模拟服务器
	ch := NewClusterHealth()
	ch.esURL = mockServer.URL

	// 测试获取并解析集群健康信息
	health, err := ch.fetchAndDecodeClusterHealth()
	require.NoError(t, err)
	assert.Equal(t, mockResponse.ClusterName, health.ClusterName)
	assert.Equal(t, mockResponse.Status, health.Status)
	assert.Equal(t, mockResponse.TimedOut, health.TimedOut)
	assert.Equal(t, mockResponse.NumberOfNodes, health.NumberOfNodes)
	assert.Equal(t, mockResponse.NumberOfDataNodes, health.NumberOfDataNodes)
	assert.Equal(t, mockResponse.ActivePrimaryShards, health.ActivePrimaryShards)
	assert.Equal(t, mockResponse.ActiveShards, health.ActiveShards)
	assert.Equal(t, mockResponse.RelocatingShards, health.RelocatingShards)
	assert.Equal(t, mockResponse.InitializingShards, health.InitializingShards)
	assert.Equal(t, mockResponse.UnassignedShards, health.UnassignedShards)
	assert.Equal(t, mockResponse.DelayedUnassignedShards, health.DelayedUnassignedShards)
	assert.Equal(t, mockResponse.NumberOfPendingTasks, health.NumberOfPendingTasks)
	assert.Equal(t, mockResponse.NumberOfInFlightFetch, health.NumberOfInFlightFetch)
	assert.Equal(t, mockResponse.TaskMaxWaitingInQueueMillis, health.TaskMaxWaitingInQueueMillis)
	assert.Equal(t, mockResponse.ActiveShardsPercentAsNumber, health.ActiveShardsPercentAsNumber)
}

func TestClusterHealthCollect(t *testing.T) {
	// 创建一个模拟的ES服务器
	mockResponse := clusterHealthResponse{
		ClusterName:             "test-cluster",
		Status:                  "green",
		TimedOut:                false,
		NumberOfNodes:           3,
		NumberOfDataNodes:       2,
		ActivePrimaryShards:     5,
		ActiveShards:            10,
		RelocatingShards:        0,
		InitializingShards:      0,
		UnassignedShards:        0,
		DelayedUnassignedShards: 0,
		NumberOfPendingTasks:    0,
		NumberOfInFlightFetch:   0,
		TaskMaxWaitingInQueueMillis: 0,
		ActiveShardsPercentAsNumber: 100.0,
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求路径
		if strings.Contains(r.URL.Path, "/_cluster/health") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// 创建ClusterHealth实例并设置URL为模拟服务器
	ch := NewClusterHealth()
	ch.esURL = mockServer.URL

	// 创建一个通道收集指标
	metrics := make(chan prometheus.Metric, 100)
	ch.Collect(metrics)

	// 获取并验证指标，确保通道不会阻塞
	close(metrics)
	
	// 确保至少有一些指标被收集到
	var metricCount int
	for range metrics {
		metricCount++
	}
	
	assert.Greater(t, metricCount, 0, "应该收集到至少一个指标")
}

func TestClusterHealthWithVariousStatuses(t *testing.T) {
	testCases := []struct {
		name      string
		status    string
		statusVal float64
		greenVal  float64
		yellowVal float64
		redVal    float64
	}{
		{
			name:      "Green Status",
			status:    "green",
			statusVal: 0,
			greenVal:  1,
			yellowVal: 0,
			redVal:    0,
		},
		{
			name:      "Yellow Status",
			status:    "yellow",
			statusVal: 1,
			greenVal:  0,
			yellowVal: 1,
			redVal:    0,
		},
		{
			name:      "Red Status",
			status:    "red",
			statusVal: 2,
			greenVal:  0,
			yellowVal: 0,
			redVal:    1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建一个模拟的ES服务器
			mockResponse := clusterHealthResponse{
				ClusterName:             "test-cluster",
				Status:                  tc.status,
				TimedOut:                false,
				NumberOfNodes:           3,
				NumberOfDataNodes:       2,
				ActivePrimaryShards:     5,
				ActiveShards:            10,
				RelocatingShards:        0,
				InitializingShards:      0,
				UnassignedShards:        0,
				DelayedUnassignedShards: 0,
				NumberOfPendingTasks:    0,
				NumberOfInFlightFetch:   0,
				TaskMaxWaitingInQueueMillis: 0,
				ActiveShardsPercentAsNumber: 100.0,
			}

			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, "/_cluster/health") {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(mockResponse)
				} else {
					http.NotFound(w, r)
				}
			}))
			defer mockServer.Close()

			// 创建ClusterHealth实例并设置URL为模拟服务器
			ch := NewClusterHealth()
			ch.esURL = mockServer.URL

			// 为每个测试用例创建单独的收集器
			// 使用通道直接收集指标而不是注册到registry
			metrics := make(chan prometheus.Metric, 100)
			
			// 手动收集指标
			ch.Collect(metrics)
			
			// 关闭通道
			close(metrics)
			
			// 验证指标
			var foundStatus, foundColorStatus bool
			var statusValue, colorStatusValue float64
			var statusLabels, colorStatusLabels []string
			
			// 检查收集的所有指标
			for metric := range metrics {
				var metricPb dto.Metric
				err := metric.Write(&metricPb)
				assert.NoError(t, err)
				
				desc := metric.Desc().String()
				
				// 检查status指标
				if strings.Contains(desc, "elasticsearch_cluster_health_status") {
					// 检查标签
					if len(metricPb.Label) == 1 {
						// 这是普通status指标
						foundStatus = true
						statusValue = *metricPb.Gauge.Value
						statusLabels = make([]string, len(metricPb.Label))
						for i, label := range metricPb.Label {
							statusLabels[i] = *label.Value
						}
						
						// 验证值
						assert.Equal(t, tc.statusVal, statusValue)
					} else if len(metricPb.Label) == 2 {
						// 这是带颜色标签的status指标
						colorName := ""
						for _, label := range metricPb.Label {
							if *label.Name == "color" {
								colorName = *label.Value
								break
							}
						}
						
						// 根据颜色验证值
						switch colorName {
						case "green":
							assert.Equal(t, tc.greenVal, *metricPb.Gauge.Value)
							if colorName == tc.status {
								foundColorStatus = true
								colorStatusValue = *metricPb.Gauge.Value
								colorStatusLabels = make([]string, len(metricPb.Label))
								for i, label := range metricPb.Label {
									colorStatusLabels[i] = *label.Value
								}
							}
						case "yellow":
							assert.Equal(t, tc.yellowVal, *metricPb.Gauge.Value)
							if colorName == tc.status {
								foundColorStatus = true
								colorStatusValue = *metricPb.Gauge.Value
								colorStatusLabels = make([]string, len(metricPb.Label))
								for i, label := range metricPb.Label {
									colorStatusLabels[i] = *label.Value
								}
							}
						case "red":
							assert.Equal(t, tc.redVal, *metricPb.Gauge.Value)
							if colorName == tc.status {
								foundColorStatus = true
								colorStatusValue = *metricPb.Gauge.Value
								colorStatusLabels = make([]string, len(metricPb.Label))
								for i, label := range metricPb.Label {
									colorStatusLabels[i] = *label.Value
								}
							}
						}
					}
				}
				
				// 验证up指标
				if strings.Contains(desc, "elasticsearch_clusterhealth_up") {
					assert.Equal(t, 1.0, *metricPb.Gauge.Value)
				}
			}
			
			// 确保找到了所有需要验证的指标
			assert.True(t, foundStatus, "Status metric not found")
			assert.True(t, foundColorStatus, "Status color metric not found")
			
			// 验证值是否符合预期
			assert.Equal(t, tc.statusVal, statusValue)
			assert.Equal(t, 1.0, colorStatusValue)
		})
	}
}

func TestClusterHealthTimedOut(t *testing.T) {
	// 创建一个模拟的ES服务器
	mockResponse := clusterHealthResponse{
		ClusterName:             "test-cluster",
		Status:                  "green",
		TimedOut:                true, // 设置超时为true
		NumberOfNodes:           3,
		NumberOfDataNodes:       2,
		ActivePrimaryShards:     5,
		ActiveShards:            10,
		RelocatingShards:        0,
		InitializingShards:      0,
		UnassignedShards:        0,
		DelayedUnassignedShards: 0,
		NumberOfPendingTasks:    0,
		NumberOfInFlightFetch:   0,
		TaskMaxWaitingInQueueMillis: 0,
		ActiveShardsPercentAsNumber: 100.0,
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/_cluster/health") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// 创建ClusterHealth实例并设置URL为模拟服务器
	ch := NewClusterHealth()
	ch.esURL = mockServer.URL

	// 收集指标到通道
	metrics := make(chan prometheus.Metric, 100)
	ch.Collect(metrics)
	close(metrics)

	// 验证超时指标
	var foundTimedOut bool
	for metric := range metrics {
		var metricPb dto.Metric
		err := metric.Write(&metricPb)
		assert.NoError(t, err)
		
		desc := metric.Desc().String()
		if strings.Contains(desc, "elasticsearch_cluster_health_timed_out") {
			foundTimedOut = true
			assert.Equal(t, 1.0, *metricPb.Gauge.Value) // 期望值为1（true）
		}
	}
	
	assert.True(t, foundTimedOut, "Timed out metric not found")
}

func TestClusterHealthCollectWithServerError(t *testing.T) {
	// 创建一个返回错误的服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	// 创建ClusterHealth实例并设置URL为模拟服务器
	ch := NewClusterHealth()
	ch.esURL = mockServer.URL

	// 收集指标到通道
	metrics := make(chan prometheus.Metric, 100)
	ch.Collect(metrics)
	close(metrics)

	// 验证up指标
	var foundUp bool
	for metric := range metrics {
		var metricPb dto.Metric
		err := metric.Write(&metricPb)
		assert.NoError(t, err)
		
		desc := metric.Desc().String()
		if strings.Contains(desc, "elasticsearch_clusterhealth_up") {
			foundUp = true
			assert.Equal(t, 0.0, *metricPb.Gauge.Value) // 期望值为0（down）
			
			// 验证URL标签
			for _, label := range metricPb.Label {
				if *label.Name == "url" {
					assert.Equal(t, mockServer.URL, *label.Value)
				}
			}
		}
	}
	
	assert.True(t, foundUp, "Up metric not found")
}

func TestClusterHealthCollectWithInvalidJSON(t *testing.T) {
	// 创建一个返回无效JSON的服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer mockServer.Close()

	// 创建ClusterHealth实例并设置URL为模拟服务器
	ch := NewClusterHealth()
	ch.esURL = mockServer.URL

	// 收集指标到通道
	metrics := make(chan prometheus.Metric, 100)
	ch.Collect(metrics)
	close(metrics)

	// 验证解析失败计数器和up指标
	var foundParseFailures, foundUp bool
	for metric := range metrics {
		var metricPb dto.Metric
		err := metric.Write(&metricPb)
		assert.NoError(t, err)
		
		desc := metric.Desc().String()
		if strings.Contains(desc, "elasticsearch_cluster_health_json_parse_failures") {
			foundParseFailures = true
			assert.Equal(t, 1.0, *metricPb.Counter.Value) // 期望解析失败计数为1
		} else if strings.Contains(desc, "elasticsearch_clusterhealth_up") {
			foundUp = true
			assert.Equal(t, 0.0, *metricPb.Gauge.Value) // 期望up为0（down）
		}
	}
	
	assert.True(t, foundParseFailures, "Parse failures metric not found")
	assert.True(t, foundUp, "Up metric not found")
}

func TestClusterHealthWithZeroShards(t *testing.T) {
	// 创建一个模拟的ES服务器，所有分片数为0
	mockResponse := clusterHealthResponse{
		ClusterName:             "test-cluster",
		Status:                  "green",
		TimedOut:                false,
		NumberOfNodes:           1,
		NumberOfDataNodes:       1,
		ActivePrimaryShards:     0,
		ActiveShards:            0,
		RelocatingShards:        0,
		InitializingShards:      0,
		UnassignedShards:        0,
		DelayedUnassignedShards: 0,
		NumberOfPendingTasks:    0,
		NumberOfInFlightFetch:   0,
		TaskMaxWaitingInQueueMillis: 0,
		ActiveShardsPercentAsNumber: 100.0,
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/_cluster/health") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// 创建ClusterHealth实例并设置URL为模拟服务器
	ch := NewClusterHealth()
	ch.esURL = mockServer.URL

	// 收集指标到通道
	metrics := make(chan prometheus.Metric, 100)
	ch.Collect(metrics)
	close(metrics)

	// 验证分片相关指标
	var foundActivePrimaryShards, foundActiveShards bool
	for metric := range metrics {
		var metricPb dto.Metric
		err := metric.Write(&metricPb)
		assert.NoError(t, err)
		
		desc := metric.Desc().String()
		if strings.Contains(desc, "elasticsearch_cluster_health_active_primary_shards") {
			foundActivePrimaryShards = true
			assert.Equal(t, 0.0, *metricPb.Gauge.Value) // 期望值为0
		} else if strings.Contains(desc, "elasticsearch_cluster_health_active_shards") && !strings.Contains(desc, "active_primary_shards") {
			foundActiveShards = true
			assert.Equal(t, 0.0, *metricPb.Gauge.Value) // 期望值为0
		}
	}
	
	assert.True(t, foundActivePrimaryShards, "Active primary shards metric not found")
	assert.True(t, foundActiveShards, "Active shards metric not found")
}

func TestClusterHealthWithHighLoadValues(t *testing.T) {
	// 创建一个模拟的ES服务器，模拟高负载情况
	mockResponse := clusterHealthResponse{
		ClusterName:             "large-cluster",
		Status:                  "yellow",
		TimedOut:                false,
		NumberOfNodes:           50,
		NumberOfDataNodes:       45,
		ActivePrimaryShards:     5000,
		ActiveShards:            10000,
		RelocatingShards:        25,
		InitializingShards:      15,
		UnassignedShards:        30,
		DelayedUnassignedShards: 10,
		NumberOfPendingTasks:    100,
		NumberOfInFlightFetch:   20,
		TaskMaxWaitingInQueueMillis: 5000,
		ActiveShardsPercentAsNumber: 95.5,
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/_cluster/health") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// 创建ClusterHealth实例并设置URL为模拟服务器
	ch := NewClusterHealth()
	ch.esURL = mockServer.URL

	// 收集指标到通道
	metrics := make(chan prometheus.Metric, 100)
	ch.Collect(metrics)
	close(metrics)

	// 验证高负载指标值
	expectedValues := map[string]float64{
		"elasticsearch_cluster_health_number_of_nodes":                    50,
		"elasticsearch_cluster_health_number_of_data_nodes":               45,
		"elasticsearch_cluster_health_active_primary_shards":              5000,
		"elasticsearch_cluster_health_active_shards":                      10000,
		"elasticsearch_cluster_health_relocating_shards":                  25,
		"elasticsearch_cluster_health_initializing_shards":                15,
		"elasticsearch_cluster_health_unassigned_shards":                  30,
		"elasticsearch_cluster_health_delayed_unassigned_shards":          10,
		"elasticsearch_cluster_health_number_of_pending_tasks":            100,
		"elasticsearch_cluster_health_number_of_in_flight_fetch":          20,
		"elasticsearch_cluster_health_task_max_waiting_in_queue_millis":   5000,
	}
	
	// 统计找到的指标
	foundMetrics := make(map[string]bool)
	
	for metric := range metrics {
		var metricPb dto.Metric
		err := metric.Write(&metricPb)
		assert.NoError(t, err)
		
		desc := metric.Desc().String()
		
		// 检查指标是否在预期列表中
		for name, expectedValue := range expectedValues {
			if strings.Contains(desc, name) {
				foundMetrics[name] = true
				assert.Equal(t, expectedValue, *metricPb.Gauge.Value)
			}
		}
	}
	
	// 确保所有预期的指标都找到了
	for name := range expectedValues {
		assert.True(t, foundMetrics[name], fmt.Sprintf("Expected metric %s not found", name))
	}
} 
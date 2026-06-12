package metrics

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClusterSettings(t *testing.T) {
	cs := NewClusterSettings()
	
	assert.NotNil(t, cs)
	assert.Equal(t, "http://localhost:9200", cs.esURL)
	assert.NotNil(t, cs.client)
	assert.NotNil(t, cs.jsonParseFailures)
	assert.NotNil(t, cs.shardAllocationEnabled)
	assert.NotNil(t, cs.maxShardsPerNode)
	assert.NotNil(t, cs.thresholdEnabled)
	assert.NotNil(t, cs.floodStageRatio)
	assert.NotNil(t, cs.highRatio)
	assert.NotNil(t, cs.lowRatio)
	assert.NotNil(t, cs.floodStageBytes)
	assert.NotNil(t, cs.highBytes)
	assert.NotNil(t, cs.lowBytes)
}

func TestClusterSettingsFetchAndDecodeClusterSettings(t *testing.T) {
	// 创建模拟响应数据
	mockResponse := map[string]interface{}{
		"persistent": map[string]interface{}{
			"cluster": map[string]interface{}{
				"max_shards_per_node": "1000",
				"routing": map[string]interface{}{
					"allocation": map[string]interface{}{
						"enable": "all",
						"disk": map[string]interface{}{
							"threshold_enabled": "true",
							"watermark": map[string]interface{}{
								"flood_stage": "95%",
								"high": "90%",
								"low": "85%",
							},
						},
					},
				},
			},
		},
		"transient": map[string]interface{}{
			"cluster": map[string]interface{}{
				"routing": map[string]interface{}{
					"allocation": map[string]interface{}{
						"disk": map[string]interface{}{
							"threshold_enabled": "false",
						},
					},
				},
			},
		},
		"defaults": map[string]interface{}{
			"cluster": map[string]interface{}{
				"max_shards_per_node": "1000",
				"routing": map[string]interface{}{
					"allocation": map[string]interface{}{
						"enable": "all",
						"disk": map[string]interface{}{
							"threshold_enabled": "true",
							"watermark": map[string]interface{}{
								"flood_stage": "95%",
								"high": "90%",
								"low": "85%",
							},
						},
					},
				},
			},
		},
	}

	// 设置模拟服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/_cluster/settings") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// 创建 ClusterSettings 实例
	cs := NewClusterSettings()
	cs.esURL = mockServer.URL

	// 调用被测试函数
	settings, err := cs.fetchAndDecodeClusterSettings()
	
	// 验证结果
	require.NoError(t, err)
	assert.NotNil(t, settings)
	
	// 验证持久设置
	assert.NotNil(t, settings.Persistent)
	assert.NotNil(t, settings.Persistent.Cluster)
	assert.Equal(t, "1000", settings.Persistent.Cluster.MaxShardsPerNode)
	assert.NotNil(t, settings.Persistent.Cluster.Routing)
	assert.NotNil(t, settings.Persistent.Cluster.Routing.Allocation)
	assert.Equal(t, "all", settings.Persistent.Cluster.Routing.Allocation.Enabled)
	
	// 验证临时设置
	assert.NotNil(t, settings.Transient)
	assert.NotNil(t, settings.Transient.Cluster)
	assert.NotNil(t, settings.Transient.Cluster.Routing)
	assert.NotNil(t, settings.Transient.Cluster.Routing.Allocation)
	assert.Equal(t, "false", settings.Transient.Cluster.Routing.Allocation.Disk.ThresholdEnabled)
}

func TestClusterSettingsCollect(t *testing.T) {
	// 创建模拟响应数据
	mockResponse := map[string]interface{}{
		"persistent": map[string]interface{}{
			"cluster": map[string]interface{}{
				"max_shards_per_node": "1000",
				"routing": map[string]interface{}{
					"allocation": map[string]interface{}{
						"enable": "all",
						"disk": map[string]interface{}{
							"threshold_enabled": "true",
							"watermark": map[string]interface{}{
								"flood_stage": "95%",
								"high": "90%",
								"low": "85%",
							},
						},
					},
				},
			},
		},
		"transient": map[string]interface{}{
			"cluster": map[string]interface{}{
				"routing": map[string]interface{}{
					"allocation": map[string]interface{}{
						"disk": map[string]interface{}{
							"threshold_enabled": "false",
						},
					},
				},
			},
		},
		"defaults": map[string]interface{}{
			"cluster": map[string]interface{}{
				"max_shards_per_node": "1000",
				"routing": map[string]interface{}{
					"allocation": map[string]interface{}{
						"enable": "all",
						"disk": map[string]interface{}{
							"threshold_enabled": "true",
							"watermark": map[string]interface{}{
								"flood_stage": "95%",
								"high": "90%",
								"low": "85%",
							},
						},
					},
				},
			},
		},
	}

	// 设置模拟服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/_cluster/settings") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// 创建 ClusterSettings 实例
	cs := NewClusterSettings()
	cs.esURL = mockServer.URL

	// 收集指标
	metrics := make(chan prometheus.Metric, 20)
	cs.Collect(metrics)
	close(metrics)

	// 验证收集到的指标
	expectedMetrics := map[string]float64{
		"elasticsearch_clustersettings_stats_max_shards_per_node": 1000.0,
		"elasticsearch_clustersettings_stats_shard_allocation_enabled": 0.0, // "all"
		"elasticsearch_clustersettings_allocation_threshold_enabled": 0.0, // "false" 从transient优先
		"elasticsearch_clustersettings_allocation_watermark_flood_stage_ratio": 0.95,
		"elasticsearch_clustersettings_allocation_watermark_high_ratio": 0.90,
		"elasticsearch_clustersettings_allocation_watermark_low_ratio": 0.85,
	}
	
	foundMetrics := make(map[string]bool)
	
	for metric := range metrics {
		var metricPb dto.Metric
		err := metric.Write(&metricPb)
		assert.NoError(t, err)
		
		desc := metric.Desc().String()
		
		// 检查是否在预期指标列表中
		for name, expectedValue := range expectedMetrics {
			if strings.Contains(desc, name) {
				foundMetrics[name] = true
				assert.Equal(t, expectedValue, *metricPb.Gauge.Value, "指标 %s 的值不符合预期", name)
			}
		}
	}
	
	// 确保所有预期的指标都被找到
	for name := range expectedMetrics {
		assert.True(t, foundMetrics[name], "未找到指标 %s", name)
	}
}

func TestClusterSettingsCollectWithServerError(t *testing.T) {
	// 设置模拟服务器返回错误
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	// 创建 ClusterSettings 实例
	cs := NewClusterSettings()
	cs.esURL = mockServer.URL

	// 收集指标
	metrics := make(chan prometheus.Metric, 10)
	cs.Collect(metrics)
	close(metrics)

	// 由于服务器错误，不应收集到任何指标，只有解析失败计数器
	for metric := range metrics {
		var metricPb dto.Metric
		err := metric.Write(&metricPb)
		assert.NoError(t, err)
		
		desc := metric.Desc().String()
		assert.True(t, strings.Contains(desc, "elasticsearch_cluster_settings_json_parse_failures"))
	}
}

func TestClusterSettingsCollectWithInvalidJSON(t *testing.T) {
	// 设置模拟服务器返回无效JSON
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer mockServer.Close()

	// 创建 ClusterSettings 实例
	cs := NewClusterSettings()
	cs.esURL = mockServer.URL

	// 收集指标
	metrics := make(chan prometheus.Metric, 10)
	cs.Collect(metrics)
	close(metrics)

	// 验证解析失败指标
	var foundParseFailures bool
	for metric := range metrics {
		var metricPb dto.Metric
		err := metric.Write(&metricPb)
		assert.NoError(t, err)
		
		desc := metric.Desc().String()
		if strings.Contains(desc, "elasticsearch_cluster_settings_json_parse_failures") {
			foundParseFailures = true
			assert.Equal(t, 1.0, *metricPb.Counter.Value)
		}
	}
	
	assert.True(t, foundParseFailures, "解析失败指标未找到")
}

func TestClusterSettingsHandleEmptySetting(t *testing.T) {
	// 创建模拟响应数据 - 缺少某些设置
	mockResponse := map[string]interface{}{
		"persistent": map[string]interface{}{},
		"transient": map[string]interface{}{},
		"defaults": map[string]interface{}{
			"cluster": map[string]interface{}{
				"max_shards_per_node": "1000",
				"routing": map[string]interface{}{
					"allocation": map[string]interface{}{
						"enable": "all",
						"disk": map[string]interface{}{
							"threshold_enabled": "true",
							"watermark": map[string]interface{}{
								"flood_stage": "95%",
								"high": "90%",
								"low": "85%",
							},
						},
					},
				},
			},
		},
	}

	// 设置模拟服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/_cluster/settings") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// 创建 ClusterSettings 实例
	cs := NewClusterSettings()
	cs.esURL = mockServer.URL

	// 收集指标
	metrics := make(chan prometheus.Metric, 10)
	cs.Collect(metrics)
	close(metrics)

	// 验证默认设置被收集
	var foundMaxShardsPerNode bool
	for metric := range metrics {
		var metricPb dto.Metric
		err := metric.Write(&metricPb)
		assert.NoError(t, err)
		
		desc := metric.Desc().String()
		if strings.Contains(desc, "elasticsearch_clustersettings_stats_max_shards_per_node") {
			foundMaxShardsPerNode = true
			assert.Equal(t, 1000.0, *metricPb.Gauge.Value)
		}
	}
	
	assert.True(t, foundMaxShardsPerNode, "max_shards_per_node 指标未找到")
} 
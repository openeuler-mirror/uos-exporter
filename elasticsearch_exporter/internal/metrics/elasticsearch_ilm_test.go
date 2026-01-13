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

func TestNewILM(t *testing.T) {
	ilm := NewILM()
	
	assert.NotNil(t, ilm)
	assert.Equal(t, "http://localhost:9200", ilm.esURL)
	assert.NotNil(t, ilm.client)
	assert.NotNil(t, ilm.jsonParseFailures)
	assert.NotNil(t, ilm.ilmStatus)
	assert.NotNil(t, ilm.ilmIndexStatus)
	assert.NotNil(t, ilm.ilmStatusOptions)
}

func TestILMFetchAndDecodeIlmStatus(t *testing.T) {
	// 创建模拟响应数据
	mockResponse := map[string]interface{}{
		"operation_mode": "RUNNING",
	}

	// 设置模拟服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/_ilm/status") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// 创建ILM实例
	ilm := NewILM()
	ilm.esURL = mockServer.URL

	// 调用被测试函数
	status, err := ilm.fetchAndDecodeIlmStatus()
	
	// 验证结果
	require.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, "RUNNING", status.OperationMode)
}

func TestILMFetchAndDecodeIlmIndexStatus(t *testing.T) {
	// 创建模拟响应数据
	mockResponse := map[string]interface{}{
		"indices": map[string]interface{}{
			"test-index-1": map[string]interface{}{
				"index": "test-index-1",
				"managed": true,
				"phase": "hot",
				"action": "rollover",
				"step": "check-rollover-ready",
				"step_time_millis": 1640995200000,
			},
			"test-index-2": map[string]interface{}{
				"index": "test-index-2",
				"managed": true,
				"phase": "warm",
				"action": "allocate",
				"step": "complete",
				"step_time_millis": 1640995200000,
			},
		},
	}

	// 设置模拟服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/_all/_ilm/explain") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// 创建ILM实例
	ilm := NewILM()
	ilm.esURL = mockServer.URL

	// 调用被测试函数
	indexStatus, err := ilm.fetchAndDecodeIlmIndexStatus()
	
	// 验证结果
	require.NoError(t, err)
	assert.NotNil(t, indexStatus)
	assert.Len(t, indexStatus.Indices, 2)
	
	// 验证索引状态
	index1, ok := indexStatus.Indices["test-index-1"]
	assert.True(t, ok)
	assert.Equal(t, "test-index-1", index1.Index)
	assert.True(t, index1.Managed)
	assert.Equal(t, "hot", index1.Phase)
	assert.Equal(t, "rollover", index1.Action)
	assert.Equal(t, "check-rollover-ready", index1.Step)
	assert.Equal(t, float64(1640995200000), index1.StepTimeMillis)
	
	index2, ok := indexStatus.Indices["test-index-2"]
	assert.True(t, ok)
	assert.Equal(t, "test-index-2", index2.Index)
	assert.True(t, index2.Managed)
	assert.Equal(t, "warm", index2.Phase)
	assert.Equal(t, "allocate", index2.Action)
	assert.Equal(t, "complete", index2.Step)
}

func TestILMCollect(t *testing.T) {
	// 设置模拟服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		if strings.Contains(r.URL.Path, "/_ilm/status") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"operation_mode": "RUNNING",
			})
		} else if strings.Contains(r.URL.Path, "/_all/_ilm/explain") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"indices": map[string]interface{}{
					"test-index-1": map[string]interface{}{
						"index": "test-index-1",
						"managed": true,
						"phase": "hot",
						"action": "rollover",
						"step": "check-rollover-ready",
						"step_time_millis": time.Now().UnixNano() / int64(time.Millisecond),
					},
					"test-index-2": map[string]interface{}{
						"index": "test-index-2",
						"managed": true,
						"phase": "warm",
						"action": "allocate",
						"step": "complete",
						"step_time_millis": time.Now().UnixNano() / int64(time.Millisecond),
					},
				},
			})
		} else {
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// 创建ILM实例
	ilm := NewILM()
	ilm.esURL = mockServer.URL

	// 收集指标
	metrics := make(chan prometheus.Metric, 10)
	ilm.Collect(metrics)
	close(metrics)

	// 验证收集到的指标
	expectedIndexStatusLabels := map[string][]string{
		"test-index-1": {"hot", "rollover", "check-rollover-ready"},
		"test-index-2": {"warm", "allocate", "complete"},
	}
	
	expectedIlmStatusLabels := []string{"STOPPED", "RUNNING", "STOPPING"}
	foundIndexLabels := make(map[string]bool)
	foundIlmStatusLabels := make(map[string]bool)
	
	for metric := range metrics {
		var metricPb dto.Metric
		err := metric.Write(&metricPb)
		assert.NoError(t, err)
		
		desc := metric.Desc().String()
		
		// 检查索引状态指标
		if strings.Contains(desc, "elasticsearch_ilm_index_status") {
			var indexName, phase, action, step string
			for _, label := range metricPb.Label {
				switch *label.Name {
				case "index":
					indexName = *label.Value
				case "phase":
					phase = *label.Value
				case "action":
					action = *label.Value
				case "step":
					step = *label.Value
				}
			}
			
			// 验证标签值
			expectedLabels, ok := expectedIndexStatusLabels[indexName]
			if ok {
				assert.Equal(t, expectedLabels[0], phase)
				assert.Equal(t, expectedLabels[1], action)
				assert.Equal(t, expectedLabels[2], step)
				assert.Equal(t, 1.0, *metricPb.Gauge.Value)
				foundIndexLabels[indexName] = true
			}
		}
		
		// 检查ILM状态指标
		if strings.Contains(desc, "elasticsearch_ilm_status") {
			var operationMode string
			for _, label := range metricPb.Label {
				if *label.Name == "operation_mode" {
					operationMode = *label.Value
					foundIlmStatusLabels[operationMode] = true
				}
			}
			
			// 验证值
			if operationMode == "RUNNING" {
				assert.Equal(t, 1.0, *metricPb.Gauge.Value)
			} else {
				assert.Equal(t, 0.0, *metricPb.Gauge.Value)
			}
		}
	}
	
	// 确保所有预期的索引标签都被找到
	for indexName := range expectedIndexStatusLabels {
		assert.True(t, foundIndexLabels[indexName], "未找到索引 %s 的指标", indexName)
	}
	
	// 确保所有预期的ILM状态标签都被找到
	for _, status := range expectedIlmStatusLabels {
		assert.True(t, foundIlmStatusLabels[status], "未找到操作模式 %s 的指标", status)
	}
}

func TestILMCollectWithServerError(t *testing.T) {
	// 设置模拟服务器返回错误
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	// 创建ILM实例
	ilm := NewILM()
	ilm.esURL = mockServer.URL

	// 收集指标
	metrics := make(chan prometheus.Metric, 5)
	ilm.Collect(metrics)
	close(metrics)

	// 由于服务器错误，不应收集到任何ILM指标，只有解析失败计数器
	onlyParseFailures := true
	for metric := range metrics {
		if !strings.Contains(metric.Desc().String(), "json_parse_failures") {
			onlyParseFailures = false
		}
	}
	
	assert.True(t, onlyParseFailures, "应该只有解析失败计数器被收集")
}

func TestILMCollectWithInvalidJSON(t *testing.T) {
	// 设置模拟服务器返回无效JSON
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer mockServer.Close()

	// 创建ILM实例
	ilm := NewILM()
	ilm.esURL = mockServer.URL

	// 收集指标
	metrics := make(chan prometheus.Metric, 5)
	ilm.Collect(metrics)
	close(metrics)

	// 验证解析失败指标
	var foundParseFailures bool
	for metric := range metrics {
		var metricPb dto.Metric
		err := metric.Write(&metricPb)
		assert.NoError(t, err)
		
		desc := metric.Desc().String()
		if strings.Contains(desc, "elasticsearch_ilm_json_parse_failures") {
			foundParseFailures = true
			assert.Equal(t, 1.0, *metricPb.Counter.Value)
		}
	}
	
	assert.True(t, foundParseFailures, "解析失败指标未找到")
} 
package metrics

import (
	"crypto/tls"
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

func TestNewIndices(t *testing.T) {
	i := NewIndices()
	
	assert.NotNil(t, i)
	assert.Equal(t, "http://localhost:9200", i.esURL)
	assert.NotNil(t, i.client)
	assert.NotNil(t, i.jsonParseFailures)
	assert.NotNil(t, i.storeSizeBytesTotal)
	assert.NotNil(t, i.docsTotal)
	assert.NotNil(t, i.segmentCountTotal)
}

func TestIndicesFetchAndDecodeIndexStats(t *testing.T) {
	// 创建模拟响应数据
	mockResponse := map[string]interface{}{
		"_all": map[string]interface{}{
			"primaries": map[string]interface{}{
				"docs": map[string]interface{}{
					"count": 100,
				},
				"store": map[string]interface{}{
					"size_in_bytes": 1024000,
				},
			},
			"total": map[string]interface{}{
				"docs": map[string]interface{}{
					"count": 200,
				},
				"store": map[string]interface{}{
					"size_in_bytes": 2048000,
				},
			},
		},
		"indices": map[string]interface{}{
			"test-index": map[string]interface{}{
				"primaries": map[string]interface{}{
					"docs": map[string]interface{}{
						"count": 50,
					},
					"store": map[string]interface{}{
						"size_in_bytes": 512000,
					},
				},
				"total": map[string]interface{}{
					"docs": map[string]interface{}{
						"count": 100,
					},
					"store": map[string]interface{}{
						"size_in_bytes": 1024000,
					},
				},
			},
		},
	}

	// 设置模拟服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/_stats") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// 创建 Indices 实例
	i := NewIndices()
	i.esURL = mockServer.URL

	// 调用被测试函数
	stats, err := i.fetchAndDecodeIndexStats()
	
	// 验证结果
	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.NotNil(t, stats.All)
	assert.NotNil(t, stats.Indices)
	
	// 验证数据
	assert.Equal(t, int64(100), stats.All.Primaries.Docs.Count)
	assert.Equal(t, int64(1024000), stats.All.Primaries.Store.SizeInBytes)
	assert.Equal(t, int64(200), stats.All.Total.Docs.Count)
	assert.Equal(t, int64(2048000), stats.All.Total.Store.SizeInBytes)
	
	// 验证索引数据
	assert.Contains(t, stats.Indices, "test-index")
	assert.Equal(t, int64(50), stats.Indices["test-index"].Primaries.Docs.Count)
	assert.Equal(t, int64(512000), stats.Indices["test-index"].Primaries.Store.SizeInBytes)
}

func TestIndicesFetchClusterInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "" {
			fmt.Fprint(w, `{"cluster_name":"test-cluster"}`)
			return
		}
		fmt.Fprint(w, "")
	}))
	defer server.Close()

	indices := NewIndices()
	indices.esURL = server.URL
	indices.client = &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}}

	clusterName, err := indices.fetchClusterInfo()
	assert.NoError(t, err)
	assert.Equal(t, "test-cluster", clusterName)
}

func TestIndicesCollect(t *testing.T) {
	// 创建模拟索引统计响应
	statsResponse := map[string]interface{}{
		"indices": map[string]interface{}{
			"test-index": map[string]interface{}{
				"primaries": map[string]interface{}{
					"docs": map[string]interface{}{
						"count": 50,
					},
					"store": map[string]interface{}{
						"size_in_bytes": 512000,
					},
				},
				"total": map[string]interface{}{
					"docs": map[string]interface{}{
						"count": 100,
					},
					"store": map[string]interface{}{
						"size_in_bytes": 1024000,
					},
				},
			},
		},
	}

	// 创建模拟集群信息响应
	clusterResponse := map[string]interface{}{
		"cluster_name": "test-cluster",
		"status": "green",
	}

	// 设置模拟服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/_stats") {
			json.NewEncoder(w).Encode(statsResponse)
		} else if strings.Contains(r.URL.Path, "/_cluster/health") {
			json.NewEncoder(w).Encode(clusterResponse)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// 创建 Indices 实例
	i := NewIndices()
	i.esURL = mockServer.URL

	// 收集指标
	metrics := make(chan prometheus.Metric, 100)
	i.Collect(metrics)
	close(metrics)

	// 验证收集到的指标
	var foundStoreSizeMetric, foundDocsCountMetric bool
	
	for metric := range metrics {
		var metricPb dto.Metric
		err := metric.Write(&metricPb)
		assert.NoError(t, err)
		
		desc := metric.Desc().String()
		
		if strings.Contains(desc, "elasticsearch_indices_store_size_bytes_total") {
			foundStoreSizeMetric = true
			// 验证标签
			for _, label := range metricPb.Label {
				if *label.Name == "index" && *label.Value == "test-index" {
					assert.Equal(t, 1024000.0, *metricPb.Gauge.Value) // total.store.size_in_bytes
				}
			}
		} else if strings.Contains(desc, "elasticsearch_indices_docs_total") {
			foundDocsCountMetric = true
			// 验证标签
			for _, label := range metricPb.Label {
				if *label.Name == "index" && *label.Value == "test-index" {
					assert.Equal(t, 100.0, *metricPb.Gauge.Value) // total.docs.count
				}
			}
		}
	}
	
	assert.True(t, foundStoreSizeMetric, "未找到存储大小指标")
	assert.True(t, foundDocsCountMetric, "未找到文档数量指标")
}

func TestIndicesCollectWithServerError(t *testing.T) {
	// 设置模拟服务器返回错误
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	// 创建 Indices 实例
	i := NewIndices()
	i.esURL = mockServer.URL

	// 收集指标
	metrics := make(chan prometheus.Metric, 10)
	i.Collect(metrics)
	close(metrics)

	// 由于服务器错误，不应收集到任何索引指标，只有解析失败计数器
	onlyParseFailures := true
	for metric := range metrics {
		if !strings.Contains(metric.Desc().String(), "parse_failures") {
			onlyParseFailures = false
		}
	}
	
	assert.True(t, onlyParseFailures, "应该只有解析失败计数器被收集")
}

func TestIndicesCollectWithInvalidJSON(t *testing.T) {
	// 设置模拟服务器返回无效JSON
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer mockServer.Close()

	// 创建 Indices 实例
	i := NewIndices()
	i.esURL = mockServer.URL

	// 收集指标
	metrics := make(chan prometheus.Metric, 10)
	i.Collect(metrics)
	close(metrics)

	// 验证解析失败指标
	var foundParseFailures bool
	var parseFailureCount float64
	for metric := range metrics {
		var metricPb dto.Metric
		err := metric.Write(&metricPb)
		assert.NoError(t, err)
		
		desc := metric.Desc().String()
		if strings.Contains(desc, "json_parse_failures") {
			foundParseFailures = true
			parseFailureCount = *metricPb.Counter.Value
		}
	}
	
	assert.True(t, foundParseFailures, "解析失败指标未找到")
	// 由于测试用例内有两个请求会尝试解析JSON，所以累计值很可能是2
	assert.GreaterOrEqual(t, parseFailureCount, float64(1), "解析失败计数应至少为1")
} 
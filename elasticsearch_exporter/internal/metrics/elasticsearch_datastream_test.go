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

func TestNewDataStream(t *testing.T) {
	ds := NewDataStream()
	assert.NotNil(t, ds)
	assert.Equal(t, "http://localhost:9200", ds.esURL)
	assert.NotNil(t, ds.client)
	assert.NotNil(t, ds.jsonParseFailures)
	assert.NotNil(t, ds.backingIndicesTotal)
	assert.NotNil(t, ds.storeSizeBytes)
}

func TestDataStreamFetchAndDecodeDataStreamStats(t *testing.T) {
	// 创建一个模拟的ES服务器
	mockResponse := DataStreamStatsResponse{
		Shards: DataStreamStatsShards{
			Total:      5,
			Successful: 5,
			Failed:     0,
		},
		DataStreamCount:     2,
		BackingIndices:      4,
		TotalStoreSizeBytes: 10240,
		DataStreamStats: []DataStreamStatsDataStream{
			{
				DataStream:       "logs-app-dev",
				BackingIndices:   2,
				StoreSizeBytes:   5120,
				MaximumTimestamp: 1609459200000,
			},
			{
				DataStream:       "metrics-system-prod",
				BackingIndices:   2,
				StoreSizeBytes:   5120,
				MaximumTimestamp: 1609459200000,
			},
		},
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/_data_stream") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// 创建DataStream实例并设置URL为模拟服务器
	ds := NewDataStream()
	ds.esURL = mockServer.URL

	// 测试获取并解析数据流统计信息
	data, err := ds.fetchAndDecodeDataStreamStats()
	require.NoError(t, err)
	assert.Equal(t, int64(2), data.DataStreamCount)
	assert.Equal(t, int64(4), data.BackingIndices)
	assert.Equal(t, int64(10240), data.TotalStoreSizeBytes)
	assert.Len(t, data.DataStreamStats, 2)
	
	// 检查第一个数据流
	assert.Equal(t, "logs-app-dev", data.DataStreamStats[0].DataStream)
	assert.Equal(t, int64(2), data.DataStreamStats[0].BackingIndices)
	assert.Equal(t, int64(5120), data.DataStreamStats[0].StoreSizeBytes)
	
	// 检查第二个数据流
	assert.Equal(t, "metrics-system-prod", data.DataStreamStats[1].DataStream)
	assert.Equal(t, int64(2), data.DataStreamStats[1].BackingIndices)
	assert.Equal(t, int64(5120), data.DataStreamStats[1].StoreSizeBytes)
}

func TestDataStreamCollect(t *testing.T) {
	// 创建一个模拟的ES服务器
	mockResponse := DataStreamStatsResponse{
		Shards: DataStreamStatsShards{
			Total:      5,
			Successful: 5,
			Failed:     0,
		},
		DataStreamCount:     2,
		BackingIndices:      4,
		TotalStoreSizeBytes: 10240,
		DataStreamStats: []DataStreamStatsDataStream{
			{
				DataStream:       "logs-app-dev",
				BackingIndices:   2,
				StoreSizeBytes:   5120,
				MaximumTimestamp: 1609459200000,
			},
			{
				DataStream:       "metrics-system-prod",
				BackingIndices:   2,
				StoreSizeBytes:   5120,
				MaximumTimestamp: 1609459200000,
			},
		},
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/_data_stream") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// 创建DataStream实例并设置URL为模拟服务器
	ds := NewDataStream()
	ds.esURL = mockServer.URL

	// 收集指标到通道
	metrics := make(chan prometheus.Metric, 10)
	ds.Collect(metrics)
	close(metrics)

	// 验证收集的指标
	// 定义期望的指标值
	expectedMetrics := map[string]map[string]float64{
		"elasticsearch_data_stream_backing_indices_total": {
			"logs-app-dev":       2.0,
			"metrics-system-prod": 2.0,
		},
		"elasticsearch_data_stream_store_size_bytes": {
			"logs-app-dev":       5120.0,
			"metrics-system-prod": 5120.0,
		},
	}

	// 验证指标
	for metric := range metrics {
		var metricPb dto.Metric
		err := metric.Write(&metricPb)
		assert.NoError(t, err)
		
		desc := metric.Desc().String()
		
		// 跳过计数器指标
		if strings.Contains(desc, "failures") {
			continue
		}
		
		// 获取数据流名称
		var dataStreamName string
		for _, label := range metricPb.Label {
			if *label.Name == "data_stream" {
				dataStreamName = *label.Value
				break
			}
		}
		
		// 验证指标值
		for metricName, expectedValues := range expectedMetrics {
			if strings.Contains(desc, metricName) {
				assert.Equal(t, expectedValues[dataStreamName], *metricPb.Gauge.Value)
			}
		}
	}
}

func TestDataStreamCollectWithServerError(t *testing.T) {
	// 创建一个返回错误的服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	// 创建DataStream实例并设置URL为模拟服务器
	ds := NewDataStream()
	ds.esURL = mockServer.URL

	// 收集指标到通道
	metrics := make(chan prometheus.Metric, 10)
	ds.Collect(metrics)
	close(metrics)

	// 只应该有失败计数器
	var foundCounter bool
	for metric := range metrics {
		var metricPb dto.Metric
		err := metric.Write(&metricPb)
		assert.NoError(t, err)
		
		desc := metric.Desc().String()
		if strings.Contains(desc, "elasticsearch_data_stream_json_parse_failures") {
			foundCounter = true
		} else {
			// 不应有其他指标
			assert.Fail(t, "Unexpected metric: "+desc)
		}
	}
	
	assert.True(t, foundCounter, "Expected to find json parse failures counter")
}

func TestDataStreamCollectWithInvalidJSON(t *testing.T) {
	// 创建一个返回无效JSON的服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/_data_stream") {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("invalid json"))
		} else {
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// 创建DataStream实例并设置URL为模拟服务器
	ds := NewDataStream()
	ds.esURL = mockServer.URL

	// 收集指标到通道
	metrics := make(chan prometheus.Metric, 10)
	ds.Collect(metrics)
	close(metrics)

	// 验证解析失败计数器被增加
	var foundCounter bool
	for metric := range metrics {
		var metricPb dto.Metric
		err := metric.Write(&metricPb)
		assert.NoError(t, err)
		
		desc := metric.Desc().String()
		if strings.Contains(desc, "elasticsearch_data_stream_json_parse_failures") {
			foundCounter = true
			// 解析失败计数器应该是1
			assert.Equal(t, 1.0, *metricPb.Counter.Value)
		}
	}
	
	assert.True(t, foundCounter, "Expected to find json parse failures counter")
} 
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

func TestNewShards(t *testing.T) {
	shards := NewShards()
	assert.NotNil(t, shards)
	assert.Equal(t, "http://localhost:9200", shards.esURL)
	assert.NotNil(t, shards.client)
	assert.NotNil(t, shards.jsonParseFailures)
	assert.NotNil(t, shards.nodeShardTotal)
}

func TestShardsFetchAndDecodeShards(t *testing.T) {
	// 创建一个模拟的ES服务器
	mockResponse := []ShardResponse{
		{
			Index: "test-index",
			Shard: "0",
			State: "STARTED",
			Node:  "node1",
		},
		{
			Index: "test-index",
			Shard: "1",
			State: "STARTED",
			Node:  "node2",
		},
		{
			Index: "test-index-2",
			Shard: "0",
			State: "RELOCATING",
			Node:  "node1",
		},
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求路径
		if strings.Contains(r.URL.Path, "/_cat/shards") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
		} else if r.URL.Path == "/" {
			// 模拟根路径返回集群信息
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"cluster_name": "test-cluster"}`))
		} else {
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// 创建Shards实例并设置URL为模拟服务器
	shards := NewShards()
	shards.esURL = mockServer.URL

	// 测试获取并解析分片信息
	shardsData, err := shards.fetchAndDecodeShards()
	require.NoError(t, err)
	assert.Len(t, shardsData, 3)
	assert.Equal(t, mockResponse[0].Index, shardsData[0].Index)
	assert.Equal(t, mockResponse[0].Shard, shardsData[0].Shard)
	assert.Equal(t, mockResponse[0].State, shardsData[0].State)
	assert.Equal(t, mockResponse[0].Node, shardsData[0].Node)
}

func TestShardsFetchClusterInfo(t *testing.T) {
	// 创建一个模拟的ES服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"cluster_name": "test-cluster"}`))
		} else {
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// 创建Shards实例并设置URL为模拟服务器
	shards := NewShards()
	shards.esURL = mockServer.URL

	// 测试获取集群信息
	clusterName, err := shards.fetchClusterInfo()
	require.NoError(t, err)
	assert.Equal(t, "test-cluster", clusterName)
}

func TestShardsCollect(t *testing.T) {
	// 创建一个模拟的ES服务器
	mockShardResponse := []ShardResponse{
		{
			Index: "test-index",
			Shard: "0",
			State: "STARTED",
			Node:  "node1",
		},
		{
			Index: "test-index",
			Shard: "1",
			State: "STARTED",
			Node:  "node1",
		},
		{
			Index: "test-index-2",
			Shard: "0",
			State: "STARTED",
			Node:  "node2",
		},
		{
			Index: "test-index-2",
			Shard: "1",
			State: "RELOCATING", // 非STARTED状态，不应计入
			Node:  "node2",
		},
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求路径
		if strings.Contains(r.URL.Path, "/_cat/shards") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockShardResponse)
		} else if r.URL.Path == "/" {
			// 模拟根路径返回集群信息
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"cluster_name": "test-cluster"}`))
		} else {
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// 创建Shards实例并设置URL为模拟服务器
	shards := NewShards()
	shards.esURL = mockServer.URL

	// 收集指标到通道
	metrics := make(chan prometheus.Metric, 10)
	shards.Collect(metrics)
	close(metrics)

	// 验证收集的指标
	expectedNodeShards := map[string]float64{
		"node1": 2.0, // 有两个STARTED状态的分片
		"node2": 1.0, // 有一个STARTED状态的分片
	}

	// 验证指标
	foundNodes := make(map[string]bool)
	for metric := range metrics {
		var metricPb dto.Metric
		err := metric.Write(&metricPb)
		assert.NoError(t, err)
		
		desc := metric.Desc().String()
		if strings.Contains(desc, "elasticsearch_node_shards_total") {
			// 获取节点名称
			var nodeName string
			for _, label := range metricPb.Label {
				if *label.Name == "node" {
					nodeName = *label.Value
					break
				}
			}
			
			foundNodes[nodeName] = true
			assert.Equal(t, expectedNodeShards[nodeName], *metricPb.Gauge.Value)
		}
	}
	
	assert.Len(t, foundNodes, 2)
	assert.True(t, foundNodes["node1"])
	assert.True(t, foundNodes["node2"])
}

func TestShardsCollectWithServerError(t *testing.T) {
	// 创建一个返回错误的服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	// 创建Shards实例并设置URL为模拟服务器
	shards := NewShards()
	shards.esURL = mockServer.URL

	// 收集指标到通道
	metrics := make(chan prometheus.Metric, 10)
	shards.Collect(metrics)
	close(metrics)

	// 只应该有失败计数器
	var foundCounter bool
	for metric := range metrics {
		var metricPb dto.Metric
		err := metric.Write(&metricPb)
		assert.NoError(t, err)
		
		desc := metric.Desc().String()
		if strings.Contains(desc, "elasticsearch_node_shards_json_parse_failures") {
			foundCounter = true
		} else {
			// 不应有其他指标
			assert.Fail(t, "Unexpected metric: "+desc)
		}
	}
	
	assert.True(t, foundCounter, "Expected to find json parse failures counter")
}

func TestShardsCollectWithInvalidJSON(t *testing.T) {
	// 创建一个返回无效JSON的服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/_cat/shards") {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("invalid json"))
		} else {
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// 创建Shards实例并设置URL为模拟服务器
	shards := NewShards()
	shards.esURL = mockServer.URL

	// 收集指标到通道
	metrics := make(chan prometheus.Metric, 10)
	shards.Collect(metrics)
	close(metrics)

	// 验证解析失败计数器被增加
	var foundCounter bool
	for metric := range metrics {
		var metricPb dto.Metric
		err := metric.Write(&metricPb)
		assert.NoError(t, err)
		
		desc := metric.Desc().String()
		if strings.Contains(desc, "elasticsearch_node_shards_json_parse_failures") {
			foundCounter = true
			// 解析失败计数器应该是1
			assert.Equal(t, 1.0, *metricPb.Counter.Value)
		}
	}
	
	assert.True(t, foundCounter, "Expected to find json parse failures counter")
}
// Part 2 commit for elasticsearch_exporter/internal/metrics/elasticsearch_shards_test.go

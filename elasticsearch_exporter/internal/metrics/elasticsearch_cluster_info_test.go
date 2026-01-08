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

func TestNewClusterInfo(t *testing.T) {
	ci := NewClusterInfo()
	assert.NotNil(t, ci)
	assert.Equal(t, "http://localhost:9200", ci.esURL)
	assert.NotNil(t, ci.client)
	assert.NotNil(t, ci.version)
	assert.NotNil(t, ci.up)
}

func TestClusterInfoFetchAndDecodeClusterInfo(t *testing.T) {
	// 创建一个模拟的ES服务器
	mockResponse := ClusterInfoResponse{
		Name:        "test-node",
		ClusterName: "test-cluster",
		ClusterUUID: "test-uuid",
		Version: VersionInfo{
			Number:        "7.10.0",
			BuildHash:     "abc123",
			BuildDate:     "2021-01-01T00:00:00Z",
			BuildSnapshot: false,
			LuceneVersion: "8.7.0",
		},
		Tagline: "You Know, for Search",
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer mockServer.Close()

	// 创建ClusterInfo实例并设置URL为模拟服务器
	ci := NewClusterInfo()
	ci.esURL = mockServer.URL

	// 测试获取并解析集群信息
	info, err := ci.fetchAndDecodeClusterInfo()
	require.NoError(t, err)
	assert.Equal(t, mockResponse.Name, info.Name)
	assert.Equal(t, mockResponse.ClusterName, info.ClusterName)
	assert.Equal(t, mockResponse.ClusterUUID, info.ClusterUUID)
	assert.Equal(t, mockResponse.Version.Number, info.Version.Number)
	assert.Equal(t, mockResponse.Version.BuildHash, info.Version.BuildHash)
	assert.Equal(t, mockResponse.Version.BuildDate, info.Version.BuildDate)
	assert.Equal(t, mockResponse.Version.BuildSnapshot, info.Version.BuildSnapshot)
	assert.Equal(t, mockResponse.Version.LuceneVersion, info.Version.LuceneVersion)
}

func TestClusterInfoCollect(t *testing.T) {
	// 创建一个模拟的ES服务器
	mockResponse := ClusterInfoResponse{
		Name:        "test-node",
		ClusterName: "test-cluster",
		ClusterUUID: "test-uuid",
		Version: VersionInfo{
			Number:        "7.10.0",
			BuildHash:     "abc123",
			BuildDate:     "2021-01-01T00:00:00Z",
			BuildSnapshot: false,
			LuceneVersion: "8.7.0",
		},
		Tagline: "You Know, for Search",
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer mockServer.Close()

	// 创建ClusterInfo实例并设置URL为模拟服务器
	ci := NewClusterInfo()
	ci.esURL = mockServer.URL

	// 收集指标到通道
	metrics := make(chan prometheus.Metric, 10)
	ci.Collect(metrics)
	close(metrics)

	// 验证收集的指标
	var foundVersion, foundUp bool
	for metric := range metrics {
		var metricPb dto.Metric
		err := metric.Write(&metricPb)
		assert.NoError(t, err)
		
		desc := metric.Desc().String()
		if strings.Contains(desc, "elasticsearch_version") {
			foundVersion = true
			assert.Equal(t, 1.0, *metricPb.Gauge.Value)
			
			// 验证标签
			labelMap := make(map[string]string)
			for _, label := range metricPb.Label {
				labelMap[*label.Name] = *label.Value
			}
			
			assert.Equal(t, "test-cluster", labelMap["cluster"])
			assert.Equal(t, "test-uuid", labelMap["cluster_uuid"])
			assert.Equal(t, "2021-01-01T00:00:00Z", labelMap["build_date"])
			assert.Equal(t, "abc123", labelMap["build_hash"])
			assert.Equal(t, "7.10.0", labelMap["version"])
			assert.Equal(t, "8.7.0", labelMap["lucene_version"])
		} else if strings.Contains(desc, "elasticsearch_clusterinfo_up") {
			foundUp = true
			assert.Equal(t, 1.0, *metricPb.Gauge.Value)
			
			// 验证URL标签
			for _, label := range metricPb.Label {
				if *label.Name == "url" {
					assert.Equal(t, mockServer.URL, *label.Value)
				}
			}
		}
	}
	
	assert.True(t, foundVersion, "Version metric not found")
	assert.True(t, foundUp, "Up metric not found")
}

func TestClusterInfoCollectError(t *testing.T) {
	// 创建一个返回错误的服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	// 创建ClusterInfo实例并设置URL为模拟服务器
	ci := NewClusterInfo()
	ci.esURL = mockServer.URL

	// 收集指标到通道
	metrics := make(chan prometheus.Metric, 10)
	ci.Collect(metrics)
	close(metrics)

	// 验证错误情况下的指标
	var foundUp bool
	for metric := range metrics {
		var metricPb dto.Metric
		err := metric.Write(&metricPb)
		assert.NoError(t, err)
		
		desc := metric.Desc().String()
		if strings.Contains(desc, "elasticsearch_clusterinfo_up") {
			foundUp = true
			assert.Equal(t, 0.0, *metricPb.Gauge.Value) // 期望up为0（down）
			
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

func TestClusterInfoCollectInvalidJSON(t *testing.T) {
	// 创建一个返回无效JSON的服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer mockServer.Close()

	// 创建ClusterInfo实例并设置URL为模拟服务器
	ci := NewClusterInfo()
	ci.esURL = mockServer.URL

	// 收集指标到通道
	metrics := make(chan prometheus.Metric, 10)
	ci.Collect(metrics)
	close(metrics)

	// 验证解析失败情况下的指标
	var foundParseFailures, foundUp bool
	for metric := range metrics {
		var metricPb dto.Metric
		err := metric.Write(&metricPb)
		assert.NoError(t, err)
		
		desc := metric.Desc().String()
		if strings.Contains(desc, "elasticsearch_cluster_info_json_parse_failures") {
			foundParseFailures = true
			assert.Equal(t, 1.0, *metricPb.Counter.Value) // 期望解析失败计数为1
		} else if strings.Contains(desc, "elasticsearch_clusterinfo_up") {
			foundUp = true
			assert.Equal(t, 0.0, *metricPb.Gauge.Value) // 期望up为0（down）
		}
	}
	
	assert.True(t, foundParseFailures, "Parse failures metric not found")
	assert.True(t, foundUp, "Up metric not found")
}

func TestClusterInfoCollectConnectionTimeout(t *testing.T) {
	// 创建一个不存在的URL
	nonExistentURL := "http://non-existent-host:9200"

	// 创建ClusterInfo实例并设置URL为不存在的URL
	ci := NewClusterInfo()
	ci.esURL = nonExistentURL

	// 减少超时时间以加快测试
	ci.client.Timeout = 100 * time.Millisecond

	// 使用通道收集指标
	ch := make(chan prometheus.Metric, 10)
	ci.Collect(ch)

	// 验证up指标为0
	found := false
	for i := 0; i < 10; i++ {
		select {
		case metric := <-ch:
			// 提取指标
			desc := metric.Desc().String()
			if strings.Contains(desc, "elasticsearch_clusterinfo_up") {
				found = true
				var dtoMetric dto.Metric
				metric.Write(&dtoMetric)
				assert.Equal(t, 0.0, *dtoMetric.Gauge.Value)
			}
		default:
			if i == 9 {
				t.Fatal("up metric not found")
			}
			time.Sleep(10 * time.Millisecond)
		}
		
		if found {
			break
		}
	}
}

func TestClusterInfoWithVariousVersions(t *testing.T) {
	testCases := []struct {
		name          string
		version       string
		buildHash     string
		luceneVersion string
	}{
		{
			name:          "ES 7.10.0",
			version:       "7.10.0",
			buildHash:     "hash1",
			luceneVersion: "8.7.0",
		},
		{
			name:          "ES 6.8.0",
			version:       "6.8.0",
			buildHash:     "hash2",
			luceneVersion: "7.7.0",
		},
		{
			name:          "ES 5.6.0",
			version:       "5.6.0",
			buildHash:     "hash3",
			luceneVersion: "6.6.0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建一个模拟的ES服务器
			mockResponse := ClusterInfoResponse{
				Name:        "test-node",
				ClusterName: "test-cluster",
				ClusterUUID: "test-uuid",
				Version: VersionInfo{
					Number:        tc.version,
					BuildHash:     tc.buildHash,
					BuildDate:     "2021-01-01T00:00:00Z",
					BuildSnapshot: false,
					LuceneVersion: tc.luceneVersion,
				},
				Tagline: "You Know, for Search",
			}

			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(mockResponse)
			}))
			defer mockServer.Close()

			// 创建ClusterInfo实例并设置URL为模拟服务器
			ci := NewClusterInfo()
			ci.esURL = mockServer.URL

			// 收集指标到通道
			metrics := make(chan prometheus.Metric, 10)
			ci.Collect(metrics)
			close(metrics)

			// 验证收集的指标
			var foundVersion, foundUp bool
			for metric := range metrics {
				var metricPb dto.Metric
				err := metric.Write(&metricPb)
				assert.NoError(t, err)
				
				desc := metric.Desc().String()
				if strings.Contains(desc, "elasticsearch_version") {
					foundVersion = true
					assert.Equal(t, 1.0, *metricPb.Gauge.Value)
					
					// 验证标签
					labelMap := make(map[string]string)
					for _, label := range metricPb.Label {
						labelMap[*label.Name] = *label.Value
					}
					
					assert.Equal(t, "test-cluster", labelMap["cluster"])
					assert.Equal(t, "test-uuid", labelMap["cluster_uuid"])
					assert.Equal(t, "2021-01-01T00:00:00Z", labelMap["build_date"])
					assert.Equal(t, tc.buildHash, labelMap["build_hash"])
					assert.Equal(t, tc.version, labelMap["version"])
					assert.Equal(t, tc.luceneVersion, labelMap["lucene_version"])
				} else if strings.Contains(desc, "elasticsearch_clusterinfo_up") {
					foundUp = true
					assert.Equal(t, 1.0, *metricPb.Gauge.Value)
				}
			}
			
			assert.True(t, foundVersion, "Version metric not found")
			assert.True(t, foundUp, "Up metric not found")
		})
	}
}

// 测试多种场景下的集群UUID
func TestClusterInfoWithVariousClusterUUIDs(t *testing.T) {
	testCases := []struct {
		name        string
		clusterUUID string
	}{
		{
			name:        "常规UUID",
			clusterUUID: "normal-uuid-12345",
		},
		{
			name:        "带连字符UUID",
			clusterUUID: "uuid-with-hyphens-123-456-789",
		},
		{
			name:        "带特殊字符的UUID",
			clusterUUID: "special_uuid@with#characters",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建一个模拟的ES服务器
			mockResponse := ClusterInfoResponse{
				Name:        "test-node",
				ClusterName: "test-cluster",
				ClusterUUID: tc.clusterUUID,
				Version: VersionInfo{
					Number:        "7.10.0",
					BuildHash:     "abc123",
					BuildDate:     "2021-01-01T00:00:00Z",
					BuildSnapshot: false,
					LuceneVersion: "8.7.0",
				},
				Tagline: "You Know, for Search",
			}

			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(mockResponse)
			}))
			defer mockServer.Close()

			// 创建ClusterInfo实例并设置URL为模拟服务器
			ci := NewClusterInfo()
			ci.esURL = mockServer.URL

			// 收集指标到通道
			metrics := make(chan prometheus.Metric, 10)
			ci.Collect(metrics)
			close(metrics)

			// 验证收集的指标
			var foundVersion bool
			for metric := range metrics {
				var metricPb dto.Metric
				err := metric.Write(&metricPb)
				assert.NoError(t, err)
				
				desc := metric.Desc().String()
				if strings.Contains(desc, "elasticsearch_version") {
					foundVersion = true
					
					// 验证cluster_uuid标签
					for _, label := range metricPb.Label {
						if *label.Name == "cluster_uuid" {
							assert.Equal(t, tc.clusterUUID, *label.Value)
						}
					}
				}
			}
			
			assert.True(t, foundVersion, "Version metric not found")
		})
	}
} 
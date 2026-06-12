package metrics

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewIndicesMappings(t *testing.T) {
	im := NewIndicesMappings()
	assert.NotNil(t, im)
	assert.Equal(t, "http://localhost:9200", im.esURL)
	assert.False(t, im.insecure)
	assert.NotNil(t, im.jsonParseFailures)
	assert.NotNil(t, im.fields)
}

func TestIndicesMappingsFetchAndDecodeIndicesMappings(t *testing.T) {
	// 创建模拟的索引映射响应
	mockMappings := map[string]IndexMapping{
		"test-index-1": {
			Mappings: IndexMappings{
				Properties: IndexMappingProperties{
					"field1": &IndexMappingProperty{
						Type: strPtr("text"),
					},
					"field2": &IndexMappingProperty{
						Type: strPtr("keyword"),
					},
					"field3": &IndexMappingProperty{
						Type: strPtr("integer"),
					},
					"nested_field": &IndexMappingProperty{
						Type: strPtr("nested"),
						Properties: IndexMappingProperties{
							"nested_field1": &IndexMappingProperty{
								Type: strPtr("text"),
							},
							"nested_field2": &IndexMappingProperty{
								Type: strPtr("keyword"),
							},
						},
					},
				},
			},
		},
		"test-index-2": {
			Mappings: IndexMappings{
				Properties: IndexMappingProperties{
					"field1": &IndexMappingProperty{
						Type: strPtr("keyword"),
					},
					"field2": &IndexMappingProperty{
						Type: strPtr("date"),
					},
				},
			},
		},
	}

	// 创建模拟服务器
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/_all/_mappings") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			data, _ := json.Marshal(mockMappings)
			w.Write(data)
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "Not found")
		}
	}))
	defer ts.Close()

	// 测试成功的响应
	im := NewIndicesMappings()
	im.esURL = ts.URL
	resp, err := im.fetchAndDecodeIndicesMappings()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(*resp))
	assert.Contains(t, *resp, "test-index-1")
	assert.Contains(t, *resp, "test-index-2")
	assert.Equal(t, 4, len((*resp)["test-index-1"].Mappings.Properties))
	assert.Equal(t, 2, len((*resp)["test-index-2"].Mappings.Properties))

	// 测试服务器错误
	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Internal Server Error")
	}))
	defer ts2.Close()

	im.esURL = ts2.URL
	_, err = im.fetchAndDecodeIndicesMappings()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP Request failed with code 500")

	// 测试无效的JSON响应
	ts3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "invalid json data")
	}))
	defer ts3.Close()

	im.esURL = ts3.URL
	_, err = im.fetchAndDecodeIndicesMappings()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

func TestIndicesMappingsCollect(t *testing.T) {
	// 创建模拟的索引映射响应
	mockMappings := map[string]IndexMapping{
		"test-index-1": {
			Mappings: IndexMappings{
				Properties: IndexMappingProperties{
					"field1": &IndexMappingProperty{
						Type: strPtr("text"),
					},
					"field2": &IndexMappingProperty{
						Type: strPtr("keyword"),
					},
					"field3": &IndexMappingProperty{
						Type: strPtr("integer"),
					},
					"nested_field": &IndexMappingProperty{
						Type: strPtr("nested"),
						Properties: IndexMappingProperties{
							"nested_field1": &IndexMappingProperty{
								Type: strPtr("text"),
							},
							"nested_field2": &IndexMappingProperty{
								Type: strPtr("keyword"),
							},
						},
					},
				},
			},
		},
		"test-index-2": {
			Mappings: IndexMappings{
				Properties: IndexMappingProperties{
					"field1": &IndexMappingProperty{
						Type: strPtr("keyword"),
					},
					"field2": &IndexMappingProperty{
						Type: strPtr("date"),
					},
				},
			},
		},
	}

	// 创建模拟服务器
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/_all/_mappings") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			data, _ := json.Marshal(mockMappings)
			w.Write(data)
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "Not found")
		}
	}))
	defer ts.Close()

	// 设置日志级别
	logrus.SetLevel(logrus.DebugLevel)

	// 测试 Collect 方法
	im := NewIndicesMappings()
	im.esURL = ts.URL

	// 创建注册表
	registry := prometheus.NewRegistry()
	registry.MustRegister(im)

	// 验证指标
	expected := `
# HELP elasticsearch_indices_mappings_json_parse_failures Number of errors while parsing JSON.
# TYPE elasticsearch_indices_mappings_json_parse_failures counter
elasticsearch_indices_mappings_json_parse_failures 0
# HELP elasticsearch_indices_mappings_stats_fields Current number fields within cluster.
# TYPE elasticsearch_indices_mappings_stats_fields gauge
elasticsearch_indices_mappings_stats_fields{index="test-index-1"} 7
elasticsearch_indices_mappings_stats_fields{index="test-index-2"} 2
`

	err := testutil.GatherAndCompare(registry, strings.NewReader(expected))
	assert.NoError(t, err)

	// 测试服务器错误时的 Collect
	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Internal Server Error")
	}))
	defer ts2.Close()

	im2 := NewIndicesMappings()
	im2.esURL = ts2.URL

	// 捕获日志输出
	var logOutput strings.Builder
	logrus.SetOutput(io.MultiWriter(&logOutput))
	defer func() {
		logrus.SetOutput(io.Discard)
	}()

	registry2 := prometheus.NewRegistry()
	registry2.MustRegister(im2)

	// 测试日志输出
	_, err = testutil.GatherAndCount(registry2)
	assert.NoError(t, err)
	assert.Contains(t, logOutput.String(), "Failed to fetch and decode indices mappings")
}

// 辅助函数，用于创建字符串指针
func strPtr(s string) *string {
	return &s
}

// 测试 countFieldsRecursive 函数
func TestCountFieldsRecursive(t *testing.T) {
	tests := []struct {
		name       string
		properties IndexMappingProperties
		expected   float64
	}{
		{
			name: "简单字段",
			properties: IndexMappingProperties{
				"field1": &IndexMappingProperty{
					Type: strPtr("text"),
				},
				"field2": &IndexMappingProperty{
					Type: strPtr("keyword"),
				},
			},
			expected: 2,
		},
		{
			name: "嵌套字段",
			properties: IndexMappingProperties{
				"field1": &IndexMappingProperty{
					Type: strPtr("text"),
				},
				"nested_field": &IndexMappingProperty{
					Type: strPtr("nested"),
					Properties: IndexMappingProperties{
						"nested_field1": &IndexMappingProperty{
							Type: strPtr("text"),
						},
						"nested_field2": &IndexMappingProperty{
							Type: strPtr("keyword"),
						},
					},
				},
			},
			expected: 5,  // 1 (field1) + 1 (nested_field) + 1 (递归计数加1) + 2 (子字段)
		},
		{
			name: "多层嵌套字段",
			properties: IndexMappingProperties{
				"field1": &IndexMappingProperty{
					Type: strPtr("text"),
				},
				"nested_field": &IndexMappingProperty{
					Type: strPtr("nested"),
					Properties: IndexMappingProperties{
						"nested_field1": &IndexMappingProperty{
							Type: strPtr("text"),
						},
						"deep_nested": &IndexMappingProperty{
							Type: strPtr("nested"),
							Properties: IndexMappingProperties{
								"deep_field1": &IndexMappingProperty{
									Type: strPtr("text"),
								},
								"deep_field2": &IndexMappingProperty{
									Type: strPtr("keyword"),
								},
							},
						},
					},
				},
			},
			expected: 8,  // 1 (field1) + 1 (nested_field) + 1 (nested_field1) + 1 (deep_nested) + 2 (子字段) + 2 (递归计数加1，嵌套两层)
		},
		{
			name:       "空字段",
			properties: IndexMappingProperties{},
			expected:   0,
		},
		{
			name: "对象类型字段",
			properties: IndexMappingProperties{
				"object_field": &IndexMappingProperty{
					Type: strPtr("object"),
					Properties: IndexMappingProperties{
						"sub_field1": &IndexMappingProperty{
							Type: strPtr("text"),
						},
						"sub_field2": &IndexMappingProperty{
							Type: strPtr("integer"),
						},
					},
				},
			},
			expected: 3,  // object类型本身不计数，但有两个子字段 + 递归计数的添加1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := countFieldsRecursive(tt.properties, 0)
			assert.Equal(t, tt.expected, count, "字段数量应该匹配")
		})
	}
} 
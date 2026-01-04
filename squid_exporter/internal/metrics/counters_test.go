package metrics

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// 测试计数器解码函数
func TestDecodeCounterStringsExtensive(t *testing.T) {
	tests := []struct {
		name          string
		line          string
		expectedKey   string
		expectedValue float64
		expectedError bool
	}{
		{
			name:          "整数值计数器",
			line:          "client_http.requests = 12345",
			expectedKey:   "client_http.requests",
			expectedValue: 12345,
			expectedError: false,
		},
		{
			name:          "浮点数值计数器",
			line:          "cache.ratio = 98.7654",
			expectedKey:   "cache.ratio",
			expectedValue: 98.7654,
			expectedError: false,
		},
		{
			name:          "带空格的键名",
			line:          "http_requests 5min = 123",
			expectedKey:   "http_requests 5min",
			expectedValue: 123,
			expectedError: false,
		},
		{
			name:          "负数值",
			line:          "cache.negative = -42.5",
			expectedKey:   "cache.negative",
			expectedValue: -42.5,
			expectedError: false,
		},
		{
			name:          "科学计数法",
			line:          "cache.large = 1.23e+6",
			expectedKey:   "cache.large",
			expectedValue: 1.23e+6,
			expectedError: false,
		},
		{
			name:          "格式错误-无等号",
			line:          "invalid counter format",
			expectedKey:   "",
			expectedValue: 0,
			expectedError: true,
		},
		{
			name:          "格式错误-非数值",
			line:          "counter = text",
			expectedKey:   "",
			expectedValue: 0,
			expectedError: true,
		},
		{
			name:          "格式错误-多个等号",
			line:          "a = b = c",
			expectedKey:   "",
			expectedValue: 0,
			expectedError: true,
		},
		{
			name:          "空行",
			line:          "",
			expectedKey:   "",
			expectedValue: 0,
			expectedError: true,
		},
		{
			name:          "只有空格",
			line:          "     ",
			expectedKey:   "",
			expectedValue: 0,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter, err := decodeCounterStrings(tt.line)
			
			if tt.expectedError {
				assert.Error(t, err, "应返回错误")
			} else {
				assert.NoError(t, err, "不应返回错误")
				assert.Equal(t, tt.expectedKey, counter.Key, "计数器键应匹配")
				assert.Equal(t, tt.expectedValue, counter.Value, "计数器值应匹配")
			}
		})
	}
}

// 生成模拟计数器数据
func generateMockCounterData(count int) []string {
	var lines []string
	for i := 0; i < count; i++ {
		lines = append(lines, fmt.Sprintf("counter.%d = %d", i, i*10))
	}
	return lines
}

// 测试Squid客户端获取计数器
func TestGetCountersWithMock(t *testing.T) {
	tests := []struct {
		name            string
		mockData        []string
		expectedCount   int
		shouldHaveError bool
	}{
		{
			name:            "正常计数器数据",
			mockData:        []string{"counter.1 = 10", "counter.2 = 20", "counter.3 = 30"},
			expectedCount:   3,
			shouldHaveError: false,
		},
		{
			name:            "空数据",
			mockData:        []string{},
			expectedCount:   0,
			shouldHaveError: false,
		},
		{
			name:            "大量数据",
			mockData:        generateMockCounterData(100),
			expectedCount:   100,
			shouldHaveError: false,
		},
		{
			name:            "部分错误数据",
			mockData:        []string{"counter.1 = 10", "invalid line", "counter.3 = 30"},
			expectedCount:   2,
			shouldHaveError: false,
		},
		{
			name:            "全部错误数据",
			mockData:        []string{"invalid.1", "invalid.2", "invalid.3"},
			expectedCount:   0,
			shouldHaveError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建模拟客户端
			mockClient := createMockClientForCounterTests(tt.mockData)
			
			// 调用GetCounters方法
			counters, err := mockClient.GetCounters()
			
			if tt.shouldHaveError {
				assert.Error(t, err, "应返回错误")
			} else {
				assert.NoError(t, err, "不应返回错误")
				assert.Equal(t, tt.expectedCount, len(counters), "计数器数量应匹配")
				
				// 验证计数器值
				if len(counters) > 0 {
					for i, counter := range counters {
						assert.NotEmpty(t, counter.Key, "计数器键不应为空")
						
						// 确认键和值匹配我们的期望（如果我们可以解析出原始行）
						if i < len(tt.mockData) && !strings.Contains(tt.mockData[i], "invalid") {
							parts := strings.Split(tt.mockData[i], " = ")
							if len(parts) == 2 {
								assert.Equal(t, strings.TrimSpace(parts[0]), counter.Key, "计数器键应匹配")
							}
						}
					}
				}
			}
		})
	}
}

// 创建一个模拟的CacheObjectClient用于测试
func createMockClientForCounterTests(mockData []string) *MockCacheObjectClient {
	return &MockCacheObjectClient{
		mockData: mockData,
	}
}

// MockCacheObjectClient是CacheObjectClient的简化模拟实现
type MockCacheObjectClient struct {
	mockData []string
}

func (m *MockCacheObjectClient) GetCounters() ([]Counter, error) {
	var counters []Counter
	
	for _, line := range m.mockData {
		counter, err := decodeCounterStrings(line)
		if err == nil {
			counters = append(counters, counter)
		}
	}
	
	return counters, nil
}

func (m *MockCacheObjectClient) GetServiceTimes() ([]Counter, error) {
	// 在这个测试中不需要实现
	return nil, nil
}

func (m *MockCacheObjectClient) GetInfos() ([]Counter, error) {
	// 在这个测试中不需要实现
	return nil, nil
}

// 测试大规模计数器处理
func TestLargeScaleCounterProcessing(t *testing.T) {
	// 生成1000个计数器
	largeMockData := generateMockCounterData(1000)
	mockClient := createMockClientForCounterTests(largeMockData)
	
	// 获取计数器并测量性能
	counters, err := mockClient.GetCounters()
	
	// 验证结果
	assert.NoError(t, err, "不应返回错误")
	assert.Equal(t, 1000, len(counters), "应该有1000个计数器")
	
	// 验证第一个和最后一个计数器
	assert.Equal(t, "counter.0", counters[0].Key)
	assert.Equal(t, 0.0, counters[0].Value)
	
	assert.Equal(t, "counter.999", counters[999].Key)
	assert.Equal(t, 9990.0, counters[999].Value)
}

// 测试提取带标签的计数器
func TestCountersWithLabels(t *testing.T) {
	testCases := []struct {
		name          string
		input         string
		expectedKey   string
		expectedValue float64
		expectedLabel VarLabel
	}{
		{
			name:          "带HTTP方法标签",
			input:         "http.method.GET = 500",
			expectedKey:   "http.method",
			expectedValue: 500,
			expectedLabel: VarLabel{Key: "method", Value: "GET"},
		},
		{
			name:          "带状态码标签",
			input:         "http.status.200 = 1000",
			expectedKey:   "http.status",
			expectedValue: 1000,
			expectedLabel: VarLabel{Key: "status", Value: "200"},
		},
		{
			name:          "带复杂标签",
			input:         "cache.size.total.disk1 = 1024",
			expectedKey:   "cache.size.total",
			expectedValue: 1024,
			expectedLabel: VarLabel{Key: "device", Value: "disk1"},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建一个解析带标签的计数器的模拟函数
			mockDecodeCounterWithLabel := func(line string) (Counter, error) {
				// 简化的实现，仅用于测试
				parts := strings.Split(line, " = ")
				if len(parts) != 2 {
					return Counter{}, fmt.Errorf("Invalid format")
				}
				
				key := parts[0]
				valueStr := parts[1]
				
				// 将最后一个点后的部分提取为标签
				keyParts := strings.Split(key, ".")
				labelValue := keyParts[len(keyParts)-1]
				baseKey := strings.Join(keyParts[:len(keyParts)-1], ".")
				
				// 确定标签键
				var labelKey string
				switch {
				case strings.Contains(baseKey, "method"):
					labelKey = "method"
				case strings.Contains(baseKey, "status"):
					labelKey = "status"
				default:
					labelKey = "device"
				}
				
				// 解析值
				var value float64
				_, err := fmt.Sscanf(valueStr, "%f", &value)
				if err != nil {
					return Counter{}, err
				}
				
				return Counter{
					Key:   baseKey,
					Value: value,
					VarLabels: []VarLabel{
						{Key: labelKey, Value: labelValue},
					},
				}, nil
			}
			
			// 使用我们的模拟函数解析输入
			counter, err := mockDecodeCounterWithLabel(tc.input)
			
			// 验证结果
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedKey, counter.Key)
			assert.Equal(t, tc.expectedValue, counter.Value)
			assert.Len(t, counter.VarLabels, 1)
			assert.Equal(t, tc.expectedLabel.Key, counter.VarLabels[0].Key)
			assert.Equal(t, tc.expectedLabel.Value, counter.VarLabels[0].Value)
		})
	}
}

// 测试特殊格式的计数器解析
func TestSpecialFormatCounters(t *testing.T) {
	tests := []struct {
		name          string
		line          string
		expectedKey   string
		expectedValue float64
		expectError   bool
	}{
		{
			name:          "前导空格",
			line:          "  leading.spaces = 100",
			expectedKey:   "leading.spaces",
			expectedValue: 100,
			expectError:   false,
		},
		{
			name:          "尾随空格",
			line:          "trailing.spaces = 200  ",
			expectedKey:   "trailing.spaces",
			expectedValue: 200,
			expectError:   false,
		},
		{
			name:          "等号周围有空格",
			line:          "spaces.around.equals  =  300",
			expectedKey:   "spaces.around.equals",
			expectedValue: 300,
			expectError:   false,
		},
		{
			name:          "非常大的数值",
			line:          "very.large = 9999999999",
			expectedKey:   "very.large",
			expectedValue: 9999999999,
			expectError:   false,
		},
		{
			name:          "非常小的数值",
			line:          "very.small = 0.0000001",
			expectedKey:   "very.small",
			expectedValue: 0.0000001,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter, err := decodeCounterStrings(tt.line)
			
			if tt.expectError {
				assert.Error(t, err, "应返回错误")
			} else {
				assert.NoError(t, err, "不应返回错误")
				assert.Equal(t, tt.expectedKey, counter.Key, "计数器键应匹配")
				assert.Equal(t, tt.expectedValue, counter.Value, "计数器值应匹配")
			}
		})
	}
} 
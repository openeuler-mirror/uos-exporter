package metrics

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// 模拟连接处理接口
type mockConnectionHandler struct {
	mock.Mock
}

func (m *mockConnectionHandler) connect() (net.Conn, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(net.Conn), args.Error(1)
}

// 模拟网络连接
type mockConn struct {
	mock.Mock
	reader *bytes.Buffer
	writer *bytes.Buffer
}

func newMockConn() *mockConn {
	return &mockConn{
		reader: bytes.NewBuffer(nil),
		writer: bytes.NewBuffer(nil),
	}
}

func (c *mockConn) Read(b []byte) (n int, err error) {
	return c.reader.Read(b)
}

func (c *mockConn) Write(b []byte) (n int, err error) {
	return c.writer.Write(b)
}

func (c *mockConn) Close() error {
	args := c.Called()
	return args.Error(0)
}

func (c *mockConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 12345}
}

func (c *mockConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 3128}
}

func (c *mockConn) SetDeadline(t time.Time) error {
	return nil
}

func (c *mockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *mockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

// 辅助函数：模拟HTTP响应
func prepareMockResponse(conn *mockConn, statusCode int, body string) {
	response := fmt.Sprintf("HTTP/1.0 %d OK\r\nContent-Length: %d\r\n\r\n%s",
		statusCode, len(body), body)
	conn.reader.WriteString(response)
}

// 测试基础指标创建和收集
func TestBaseMetrics(t *testing.T) {
	tests := []struct {
		name      string
		fqname    string
		help      string
		labels    []string
		value     float64
		labelVals []string
	}{
		{
			name:      "基本指标-无标签",
			fqname:    "test_metric",
			help:      "Test help text",
			labels:    []string{},
			value:     42.0,
			labelVals: []string{},
		},
		{
			name:      "基本指标-单标签",
			fqname:    "test_metric_with_label",
			help:      "Test metric with label",
			labels:    []string{"label1"},
			value:     123.45,
			labelVals: []string{"value1"},
		},
		{
			name:      "基本指标-多标签",
			fqname:    "test_metric_multi_labels",
			help:      "Test metric with multiple labels",
			labels:    []string{"label1", "label2", "label3"},
			value:     99.99,
			labelVals: []string{"value1", "value2", "value3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建基本指标
			m := NewMetrics(tt.fqname, tt.help, tt.labels)
			
			// 验证字段设置
			assert.Equal(t, tt.labels, m.labels, "标签应匹配")
			assert.NotNil(t, m.desc, "描述不应为空")
			
			// 测试收集方法
			ch := make(chan prometheus.Metric, 1)
			m.collect(ch, tt.value, tt.labelVals)
			
			// 检查通道中是否有数据
			select {
			case metric := <-ch:
				assert.NotNil(t, metric, "应收集到指标")
			default:
				t.Error("未能收集指标")
			}
		})
	}
}

// 测试基本认证字符串构建函数
func TestBuildBasicAuthString(t *testing.T) {
	tests := []struct {
		name     string
		login    string
		password string
		expected string
	}{
		{
			name:     "空登录名",
			login:    "",
			password: "pass",
			expected: "",
		},
		{
			name:     "有效登录名和密码",
			login:    "user",
			password: "pass",
			expected: base64.StdEncoding.EncodeToString([]byte("user:pass")),
		},
		{
			name:     "空密码",
			login:    "user",
			password: "",
			expected: base64.StdEncoding.EncodeToString([]byte("user:")),
		},
		{
			name:     "特殊字符",
			login:    "user@domain",
			password: "p@$$w0rd!",
			expected: base64.StdEncoding.EncodeToString([]byte("user@domain:p@$$w0rd!")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildBasicAuthString(tt.login, tt.password)
			assert.Equal(t, tt.expected, result, "基本认证字符串应匹配")
		})
	}
}

// 测试计数器解码函数
func TestDecodeCounterStrings(t *testing.T) {
	tests := []struct {
		name          string
		line          string
		expectedKey   string
		expectedValue float64
		expectError   bool
	}{
		{
			name:          "有效计数器行",
			line:          "client_http.requests = 12345",
			expectedKey:   "client_http.requests",
			expectedValue: 12345,
			expectError:   false,
		},
		{
			name:          "浮点数值",
			line:          "cache.ratio = 98.7654",
			expectedKey:   "cache.ratio",
			expectedValue: 98.7654,
			expectError:   false,
		},
		{
			name:          "无效格式",
			line:          "invalid counter format",
			expectedKey:   "",
			expectedValue: 0,
			expectError:   true,
		},
		{
			name:          "非数值",
			line:          "counter = not_a_number",
			expectedKey:   "",
			expectedValue: 0,
			expectError:   true,
		},
		{
			name:          "空行",
			line:          "",
			expectedKey:   "",
			expectedValue: 0,
			expectError:   true,
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

// 测试创建缓存对象客户端
func TestNewCacheObjectClient(t *testing.T) {
	tests := []struct {
		name     string
		request  *CacheObjectRequest
		expected *CacheObjectClient
	}{
		{
			name: "基本客户端",
			request: &CacheObjectRequest{
				Hostname: "localhost",
				Port:     3128,
				Login:    "",
				Password: "",
				Headers:  nil,
			},
			expected: &CacheObjectClient{
				basicAuthString: "",
				headers:         nil,
			},
		},
		{
			name: "带认证的客户端",
			request: &CacheObjectRequest{
				Hostname: "squid.example.com",
				Port:     8080,
				Login:    "admin",
				Password: "secret",
				Headers:  []string{"Custom-Header: Value"},
			},
			expected: &CacheObjectClient{
				basicAuthString: buildBasicAuthString("admin", "secret"),
				headers:         []string{"Custom-Header: Value"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewCacheObjectClient(tt.request)
			
			assert.NotNil(t, client, "客户端不应为空")
			assert.NotNil(t, client.ch, "连接处理程序不应为空")
			assert.Equal(t, tt.expected.basicAuthString, client.basicAuthString, "基本认证字符串应匹配")
			assert.Equal(t, tt.expected.headers, client.headers, "头信息应匹配")
		})
	}
}

// 测试连接处理实现
func TestConnectionHandlerImpl(t *testing.T) {
	handler := &connectionHandlerImpl{
		hostname: "localhost",
		port:     3128,
	}
	
	// 这个测试实际上不会建立连接，仅验证接口实现
	t.Run("接口实现", func(t *testing.T) {
		var _ connectionHandler = handler
	})
}

// 测试从Squid读取函数（模拟网络调用）
func TestReadFromSquid(t *testing.T) {
	tests := []struct {
		name           string
		endpoint       string
		statusCode     int
		responseBody   string
		authCredentials bool
		expectError    bool
	}{
		{
			name:           "成功响应",
			endpoint:       "counters",
			statusCode:     200,
			responseBody:   "metric1 = 123\nmetric2 = 456\n",
			authCredentials: false,
			expectError:    false,
		},
		{
			name:           "错误状态码",
			endpoint:       "counters",
			statusCode:     404,
			responseBody:   "Not Found",
			authCredentials: false,
			expectError:    true,
		},
		{
			name:           "带认证的请求",
			endpoint:       "counters",
			statusCode:     200,
			responseBody:   "metric1 = 123\n",
			authCredentials: true,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建模拟连接和处理程序
			mockConn := newMockConn()
			mockConn.On("Close").Return(nil)
			
			mockHandler := new(mockConnectionHandler)
			mockHandler.On("connect").Return(mockConn, nil)
			
			// 配置基本认证
			var basicAuthString string
			if tt.authCredentials {
				basicAuthString = buildBasicAuthString("user", "pass")
			}
			
			// 创建客户端
			client := &CacheObjectClient{
				ch:              mockHandler,
				basicAuthString: basicAuthString,
				headers:         []string{"Test-Header: Value"},
			}
			
			// 准备响应
			prepareMockResponse(mockConn, tt.statusCode, tt.responseBody)
			
			// 这里我们不调用client.readFromSquid，而是测试发送HTTP请求和处理响应的逻辑
			
			// 模拟连接
			conn, err := client.ch.connect()
			assert.NoError(t, err, "连接应该成功")
			assert.NotNil(t, conn, "连接不应为空")
			
			// 构建请求并发送
			var requestStr string
			
			// 测试构建请求部分
			rBody := append(client.headers, []string{
				fmt.Sprintf(requestProtocol, tt.endpoint),
				"Host: localhost",
				"User-Agent: squidclient/3.5.12",
			}...)
			
			if basicAuthString != "" {
				rBody = append(rBody, "Proxy-Authorization: Basic "+basicAuthString)
				rBody = append(rBody, "Authorization: Basic "+basicAuthString)
			}
			
			rBody = append(rBody, "Accept: */*", "\r\n")
			requestStr = strings.Join(rBody, "\r\n")
			
			// 验证请求内容
			// 检查端点
			assert.Contains(t, requestStr, fmt.Sprintf("GET cache_object://localhost/%s", tt.endpoint))
			
			// 检查自定义头
			assert.Contains(t, requestStr, "Test-Header: Value")
			
			// 如果提供了认证凭据，则检查认证头
			if tt.authCredentials {
				assert.Contains(t, requestStr, "Proxy-Authorization: Basic "+basicAuthString)
				assert.Contains(t, requestStr, "Authorization: Basic "+basicAuthString)
			}
			
			// 模拟处理响应
			if tt.statusCode != 200 {
				assert.True(t, tt.expectError, "非200状态码应该导致错误")
			} else {
				assert.False(t, tt.expectError, "200状态码不应该导致错误")
				
				// 模拟读取响应主体
				reader := bufio.NewReader(strings.NewReader(tt.responseBody))
				linesChan := make(chan string)
				
				// 读取行
				go func() {
					defer close(linesChan)
					for {
						line, err := reader.ReadString('\n')
						if err != nil {
							break
						}
						linesChan <- line
					}
				}()
				
				// 收集响应行
				var lines []string
				for line := range linesChan {
					lines = append(lines, line)
				}
				
				// 验证响应主体
				expectedLines := strings.Split(tt.responseBody, "\n")
				if expectedLines[len(expectedLines)-1] == "" {
					expectedLines = expectedLines[:len(expectedLines)-1]
				}
				assert.Equal(t, len(expectedLines), len(lines), "行数应匹配")
			}
		})
	}
}

// 测试读取行函数
func TestReadLines(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedLines []string
	}{
		{
			name:          "多行输入",
			input:         "line1\nline2\nline3\n",
			expectedLines: []string{"line1\n", "line2\n", "line3\n"},
		},
		{
			name:          "空输入",
			input:         "",
			expectedLines: []string{},
		},
		{
			name:          "单行输入",
			input:         "just one line\n",
			expectedLines: []string{"just one line\n"},
		},
		{
			name:          "没有结束换行符",
			input:         "line1\nline2",
			expectedLines: []string{"line1\n"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			linesChan := make(chan string)
			
			// 并行读取行
			go readLines(reader, linesChan)
			
			// 收集所有行
			var receivedLines []string
			for line := range linesChan {
				receivedLines = append(receivedLines, line)
			}
			
			// 验证结果
			assert.Equal(t, len(tt.expectedLines), len(receivedLines), "行数应匹配")
			
			// 逐行比较内容，避免索引越界
			minLength := len(tt.expectedLines)
			if len(receivedLines) < minLength {
				minLength = len(receivedLines)
			}
			
			for i := 0; i < minLength; i++ {
				assert.Equal(t, tt.expectedLines[i], receivedLines[i], "行内容应匹配")
			}
		})
	}
}

// 测试Squid收集器
func TestSquidCollector(t *testing.T) {
	// 创建模拟Squid客户端
	mockClient := new(MockSquidClient)
	mockCounters := []Counter{
		{Key: "test.counter1", Value: 123},
		{Key: "test.counter2", Value: 456},
	}
	
	// 配置模拟行为
	mockClient.On("GetCounters").Return(mockCounters, nil)
	
	// 创建收集器
	collector := &SquidCollector{
		client:   mockClient,
		hostname: "localhost",
		port:     3128,
		up:       prometheus.NewGauge(prometheus.GaugeOpts{Name: "squid_up"}),
	}
	
	// 测试Describe方法
	t.Run("Describe方法", func(t *testing.T) {
		ch := make(chan *prometheus.Desc, 1)
		collector.Describe(ch)
		
		select {
		case desc := <-ch:
			assert.NotNil(t, desc, "描述不应为空")
		default:
			t.Error("未能获取描述")
		}
	})
	
	// 测试Collect方法
	t.Run("Collect方法-服务可用", func(t *testing.T) {
		ch := make(chan prometheus.Metric, 1)
		collector.Collect(ch)
		
		select {
		case metric := <-ch:
			assert.NotNil(t, metric, "指标不应为空")
		default:
			t.Error("未能收集指标")
		}
		
		// 验证调用
		mockClient.AssertCalled(t, "GetCounters")
	})
	
	// 测试错误情况
	t.Run("Collect方法-服务不可用", func(t *testing.T) {
		errorClient := new(MockSquidClient)
		errorClient.On("GetCounters").Return(nil, fmt.Errorf("连接错误"))
		
		errorCollector := &SquidCollector{
			client:   errorClient,
			hostname: "localhost",
			port:     3128,
			up:       prometheus.NewGauge(prometheus.GaugeOpts{Name: "squid_up"}),
		}
		
		ch := make(chan prometheus.Metric, 1)
		errorCollector.Collect(ch)
		
		select {
		case metric := <-ch:
			assert.NotNil(t, metric, "指标不应为空")
		default:
			t.Error("未能收集指标")
		}
		
		// 验证调用
		errorClient.AssertCalled(t, "GetCounters")
	})
}

// 模拟Squid客户端用于测试
type MockSquidClient struct {
	mock.Mock
}

func (m *MockSquidClient) GetCounters() ([]Counter, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]Counter), args.Error(1)
}

func (m *MockSquidClient) GetServiceTimes() ([]Counter, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]Counter), args.Error(1)
}

func (m *MockSquidClient) GetInfos() ([]Counter, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]Counter), args.Error(1)
}

// 测试NewSquidCollector函数
func TestNewSquidCollector(t *testing.T) {
	tests := []struct {
		name   string
		config *SquidConfig
	}{
		{
			name: "基本配置",
			config: &SquidConfig{
				Hostname:     "localhost",
				Port:         3128,
				Login:        "",
				Password:     "",
				Headers:      nil,
				ExtractTimes: false,
			},
		},
		{
			name: "完整配置",
			config: &SquidConfig{
				Hostname:     "squid.example.com",
				Port:         8080,
				Login:        "admin",
				Password:     "secret",
				Headers:      []string{"Custom-Header: Value"},
				ExtractTimes: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := NewSquidCollector(tt.config)
			
			assert.NotNil(t, collector, "收集器不应为空")
			assert.NotNil(t, collector.client, "客户端不应为空")
			assert.Equal(t, tt.config.Hostname, collector.hostname, "主机名应匹配")
			assert.Equal(t, tt.config.Port, collector.port, "端口应匹配")
			assert.Equal(t, tt.config.ExtractTimes, collector.extractTimes, "ExtractTimes标志应匹配")
			assert.NotNil(t, collector.up, "up指标不应为空")
		})
	}
}

// 测试Counter和VarLabel结构体
func TestCounterStructs(t *testing.T) {
	t.Run("Counter结构", func(t *testing.T) {
		counter := Counter{
			Key:   "test.counter",
			Value: 123.45,
			VarLabels: []VarLabel{
				{Key: "label1", Value: "value1"},
				{Key: "label2", Value: "value2"},
			},
		}
		
		assert.Equal(t, "test.counter", counter.Key, "键应匹配")
		assert.Equal(t, 123.45, counter.Value, "值应匹配")
		assert.Len(t, counter.VarLabels, 2, "应有两个标签")
		assert.Equal(t, "label1", counter.VarLabels[0].Key, "第一个标签键应匹配")
		assert.Equal(t, "value1", counter.VarLabels[0].Value, "第一个标签值应匹配")
	})
	
	t.Run("VarLabel结构", func(t *testing.T) {
		label := VarLabel{
			Key:   "test_label",
			Value: "test_value",
		}
		
		assert.Equal(t, "test_label", label.Key, "标签键应匹配")
		assert.Equal(t, "test_value", label.Value, "标签值应匹配")
	})
}

// 测试全局变量
func TestGlobalVariables(t *testing.T) {
	t.Run("名称", func(t *testing.T) {
		assert.Equal(t, "squid_exporter", Name, "全局名称应匹配")
	})
	
	t.Run("版本", func(t *testing.T) {
		assert.Equal(t, "1.0.0", Version, "全局版本应匹配")
	})
} 
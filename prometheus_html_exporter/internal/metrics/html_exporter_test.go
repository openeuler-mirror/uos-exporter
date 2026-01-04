package metrics

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/antchfx/htmlquery"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

// 模拟HTTP客户端的响应
type mockReadCloser struct {
	io.Reader
}

func (m mockReadCloser) Close() error {
	return nil
}

func newMockReadCloser(s string) io.ReadCloser {
	return mockReadCloser{bytes.NewBufferString(s)}
}

// TestNewHTMLExporter 测试创建新的HTMLExporter实例
func TestNewHTMLExporter(t *testing.T) {
	// 准备测试数据
	metricConfig := MetricConfig{
		Name:   "test_metric",
		Help:   "Test metric help text",
		Type:   "gauge",
		Labels: map[string]string{"label1": "value1", "label2": "value2"},
	}
	metricPrefix := "test_prefix_"
	address := "http://example.com"
	selector := "//div[@id='value']"
	decimalPointSeparator := "."
	thousandsSeparator := ","

	// 创建HTMLExporter
	exporter := NewHTMLExporter(
		metricPrefix,
		metricConfig,
		address,
		selector,
		decimalPointSeparator,
		thousandsSeparator,
	)

	// 验证对象属性
	assert.Equal(t, address, exporter.address)
	assert.Equal(t, selector, exporter.selector)
	assert.Equal(t, decimalPointSeparator, exporter.decimalPointSeparator)
	assert.Equal(t, thousandsSeparator, exporter.thousandsSeparator)
	assert.Equal(t, metricConfig, exporter.metricConfig)
	assert.Equal(t, metricPrefix, exporter.metricPrefix)
	assert.NotNil(t, exporter.baseMetrics)
	assert.NotNil(t, exporter.desc)
}

// TestGetPrometheusValueType 测试获取Prometheus值类型
func TestGetPrometheusValueType(t *testing.T) {
	// 创建一个简单的HTMLExporter以便调用方法
	exporter := &HTMLExporter{}

	// 测试不同的指标类型
	testCases := []struct {
		metricType string
		expected   prometheus.ValueType
	}{
		{"gauge", prometheus.GaugeValue},
		{"counter", prometheus.CounterValue},
		{"unknown", prometheus.UntypedValue},
		{"", prometheus.UntypedValue},
	}

	for _, tc := range testCases {
		t.Run(tc.metricType, func(t *testing.T) {
			result := exporter.getPrometheusValueType(tc.metricType)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestNormalizeNumericValue 测试数值标准化
func TestNormalizeNumericValue(t *testing.T) {
	// 创建一个简单的HTMLExporter以便调用方法
	exporter := &HTMLExporter{}

	// 测试不同的数值格式
	testCases := []struct {
		name                 string
		value                string
		thousandsSeparator   string
		decimalPointSeparator string
		expected             float64
		expectError          bool
	}{
		{"Simple integer", "123", ",", ".", 123.0, false},
		{"Decimal", "123.45", ",", ".", 123.45, false},
		{"With thousands separator", "1,234", ",", ".", 1234.0, false},
		{"European format", "1.234,56", ".", ",", 1234.56, false},
		{"Trailing space", "123.45 ", ",", ".", 123.45, false},
		{"Leading space", " 123.45", ",", ".", 123.45, false},
		{"Non-numeric", "abc", ",", ".", 0, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := exporter.normalizeNumericValue(
				tc.value, 
				tc.thousandsSeparator, 
				tc.decimalPointSeparator,
			)
			
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

// TestParseSelector 测试XPath选择器解析
func TestParseSelector(t *testing.T) {
	// 创建一个简单的HTMLExporter以便调用方法
	exporter := &HTMLExporter{}

	// 测试HTML解析
	testCases := []struct {
		name          string
		html          string
		selector      string
		expected      string
		expectError   bool
	}{
		{
			name:        "Valid selector",
			html:        "<html><body><div id='value'>123.45</div></body></html>",
			selector:    "//div[@id='value']",
			expected:    "123.45",
			expectError: false,
		},
		{
			name:        "No matching elements",
			html:        "<html><body><div>123.45</div></body></html>",
			selector:    "//div[@id='value']",
			expected:    "",
			expectError: true,
		},
		{
			name:        "Multiple matching elements",
			html:        "<html><body><div class='value'>123.45</div><div class='value'>678.90</div></body></html>",
			selector:    "//div[@class='value']",
			expected:    "123.45", // 应该返回第一个匹配的元素
			expectError: false,
		},
		{
			name:        "Invalid XPath selector",
			html:        "<html><body><div>123.45</div></body></html>",
			selector:    "//div[", // 不完整的XPath表达式
			expected:    "",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body := newMockReadCloser(tc.html)
			result, err := exporter.parseSelector(body, tc.selector)
			
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

// TestCollect 测试指标收集功能
func TestCollect(t *testing.T) {
	// 准备测试数据
	metricConfig := MetricConfig{
		Name:   "test_metric",
		Help:   "Test metric help text",
		Type:   "gauge",
		Labels: map[string]string{"label1": "value1", "label2": "value2"},
	}
	metricPrefix := "test_prefix_"
	address := "http://example.com"
	selector := "//div[@id='value']"
	decimalPointSeparator := "."
	thousandsSeparator := ","

	// 创建一个测试用的HTMLExporter
	exporter := NewHTMLExporter(
		metricPrefix,
		metricConfig,
		address,
		selector,
		decimalPointSeparator,
		thousandsSeparator,
	)

	// 使用httptest创建一个测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body><div id='value'>123.45</div></body></html>"))
	}))
	defer server.Close()
	exporter.address = server.URL

	// 创建一个通道来接收指标
	ch := make(chan prometheus.Metric, 1)

	// 收集指标
	go exporter.Collect(ch)

	// 从通道接收指标
	metric := <-ch

	// 验证指标
	assert.NotNil(t, metric)
}

// TestCollectEmptyAddress 测试空地址情况
func TestCollectEmptyAddress(t *testing.T) {
	// 准备测试数据
	metricConfig := MetricConfig{
		Name:   "test_metric",
		Help:   "Test metric help text",
		Type:   "gauge",
		Labels: map[string]string{"label1": "value1", "label2": "value2"},
	}
	metricPrefix := "test_prefix_"
	address := "" // 空地址
	selector := "//div[@id='value']"
	decimalPointSeparator := "."
	thousandsSeparator := ","

	// 创建一个测试用的HTMLExporter
	exporter := NewHTMLExporter(
		metricPrefix,
		metricConfig,
		address,
		selector,
		decimalPointSeparator,
		thousandsSeparator,
	)

	// 创建一个通道来接收指标
	ch := make(chan prometheus.Metric, 1)

	// 收集指标（应该跳过）
	exporter.Collect(ch)

	// 验证通道是空的
	select {
	case <-ch:
		t.Error("Expected no metrics to be collected due to empty address")
	default:
		// 通道是空的，符合预期
	}
}

// 模拟HTML解析，用于测试
func mockHTMLParse(rc io.ReadCloser) (*html.Node, error) {
	if rc == nil {
		return nil, io.ErrUnexpectedEOF
	}
	html, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}
	// 简单解析HTML
	doc, err := htmlquery.Parse(strings.NewReader(string(html)))
	return doc, err
}

// TestDoRequest 测试HTTP请求功能
func TestDoRequest(t *testing.T) {
	// 创建一个简单的HTMLExporter
	exporter := &HTMLExporter{}

	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求头
		userAgent := r.Header.Get("User-Agent")
		assert.Contains(t, userAgent, "prometheus-html-exporter")
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body>Test Response</body></html>"))
	}))
	defer server.Close()

	// 执行请求
	body, err := exporter.doRequest(server.URL)
	require.NoError(t, err)
	defer body.Close()

	// 读取响应
	bodyBytes, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Contains(t, string(bodyBytes), "Test Response")
}

// TestDoRequestError 测试HTTP请求错误情况
func TestDoRequestError(t *testing.T) {
	// 创建一个简单的HTMLExporter
	exporter := &HTMLExporter{}

	// 测试无效URL
	_, err := exporter.doRequest("invalid-url")
	assert.Error(t, err)

	// 测试服务器返回错误状态码
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err = exporter.doRequest(server.URL)
	assert.Error(t, err)
}

// TestDescribe 测试描述方法
func TestDescribe(t *testing.T) {
	// 准备测试数据
	metricConfig := MetricConfig{
		Name:   "test_metric",
		Help:   "Test metric help text",
		Type:   "gauge",
		Labels: map[string]string{"label1": "value1", "label2": "value2"},
	}
	metricPrefix := "test_prefix_"
	address := "http://example.com"
	selector := "//div[@id='value']"
	decimalPointSeparator := "."
	thousandsSeparator := ","

	// 创建一个测试用的HTMLExporter
	exporter := NewHTMLExporter(
		metricPrefix,
		metricConfig,
		address,
		selector,
		decimalPointSeparator,
		thousandsSeparator,
	)

	// 创建一个通道来接收描述
	ch := make(chan *prometheus.Desc, 1)

	// 描述指标
	exporter.Describe(ch)

	// 从通道接收描述
	desc := <-ch

	// 验证描述
	assert.NotNil(t, desc)
} 
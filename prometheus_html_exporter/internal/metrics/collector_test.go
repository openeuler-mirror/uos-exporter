package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 创建一个简单的指标收集器供测试使用
type TestCollector struct {
	*baseMetrics
	value float64
	labels []string
}

func NewTestCollector(value float64, labels []string) *TestCollector {
	return &TestCollector{
		baseMetrics: NewMetrics(
			"test_collector_metric",
			"Test collector metric help text",
			[]string{"label1", "label2"},
		),
		value: value,
		labels: labels,
	}
}

func (c *TestCollector) Collect(ch chan<- prometheus.Metric) {
	c.baseMetrics.collect(ch, c.value, c.labels)
}

func (c *TestCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

// TestRegisterCollector 测试注册收集器
func TestRegisterCollector(t *testing.T) {
	// 创建一个测试收集器
	collector := NewTestCollector(123.45, []string{"test1", "test2"})

	// 注册到新的Registry
	registry := prometheus.NewRegistry()
	err := registry.Register(collector)
	require.NoError(t, err, "收集器应该成功注册")

	// 尝试注册重复名称的收集器（应该失败）
	dupCollector := NewTestCollector(456.78, []string{"test3", "test4"})
	err = registry.Register(dupCollector)
	assert.Error(t, err, "注册重复的收集器应该失败")
}

// TestCollectMetric 测试收集指标
func TestCollectMetric(t *testing.T) {
	// 创建一个测试收集器
	value := 123.45
	labels := []string{"test1", "test2"}
	collector := NewTestCollector(value, labels)

	// 创建一个通道来接收指标
	ch := make(chan prometheus.Metric, 1)

	// 收集指标
	collector.Collect(ch)

	// 从通道接收指标
	metric := <-ch

	// 验证指标
	assert.NotNil(t, metric)

	// 注册到一个registry并测试
	registry := prometheus.NewRegistry()
	err := registry.Register(collector)
	require.NoError(t, err)

	// 使用testutil验证指标值和标签
	expected := `
# HELP test_collector_metric Test collector metric help text
# TYPE test_collector_metric gauge
test_collector_metric{label1="test1",label2="test2"} 123.45
`
	err = testutil.GatherAndCompare(registry, strings.NewReader(expected))
	assert.NoError(t, err)
}

// TestHTMLExporterCollector 测试HTML导出器收集器
func TestHTMLExporterCollector(t *testing.T) {
	// 创建测试服务器返回HTML响应
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `<html><body><div id="value">123.45</div></body></html>`
		w.Write([]byte(html))
	}))
	defer server.Close()

	// 创建配置
	metricConfig := MetricConfig{
		Name:   "html_test_metric",
		Help:   "HTML test metric help text",
		Type:   "gauge",
		Labels: map[string]string{"label1": "value1", "label2": "value2"},
	}

	// 创建HTML收集器
	exporter := NewHTMLExporter(
		"test_prefix_",
		metricConfig,
		server.URL,
		"//div[@id='value']",
		".",
		",",
	)

	// 注册到一个新的registry
	registry := prometheus.NewRegistry()
	err := registry.Register(exporter)
	require.NoError(t, err)

	// 使用testutil验证指标名称格式
	metrics, err := registry.Gather()
	require.NoError(t, err)
	require.NotEmpty(t, metrics)

	// 验证收集的指标
	assert.Equal(t, "test_prefix_html_test_metric", metrics[0].GetName())
	assert.Equal(t, "HTML test metric help text", metrics[0].GetHelp())
	assert.Equal(t, dto.MetricType_GAUGE, metrics[0].GetType())

	// 验证标签
	for _, m := range metrics[0].GetMetric() {
		assert.Equal(t, 2, len(m.GetLabel()))
		labelMap := make(map[string]string)
		for _, label := range m.GetLabel() {
			labelMap[label.GetName()] = label.GetValue()
		}
		assert.Equal(t, "value1", labelMap["label1"])
		assert.Equal(t, "value2", labelMap["label2"])
	}
}

// TestMultipleCollectors 测试多个收集器
func TestMultipleCollectors(t *testing.T) {
	// 创建多个测试收集器
	collector1 := NewTestCollector(123.45, []string{"test1", "test2"})
	
	// 创建一个带不同名称的收集器
	collector2 := &TestCollector{
		baseMetrics: NewMetrics(
			"another_test_metric",
			"Another test metric help text",
			[]string{"label1", "label2"},
		),
		value: 456.78,
		labels: []string{"test3", "test4"},
	}

	// 注册到一个registry
	registry := prometheus.NewRegistry()
	err := registry.Register(collector1)
	require.NoError(t, err)
	
	err = registry.Register(collector2)
	require.NoError(t, err)

	// 收集指标
	metrics, err := registry.Gather()
	require.NoError(t, err)
	
	// 应该有两个指标系列
	assert.Equal(t, 2, len(metrics))
	
	// 验证指标名称
	metricNames := []string{metrics[0].GetName(), metrics[1].GetName()}
	assert.Contains(t, metricNames, "test_collector_metric")
	assert.Contains(t, metricNames, "another_test_metric")
}

// TestBaseMetricsCollect 测试基础指标收集功能
func TestBaseMetricsCollect(t *testing.T) {
	// 创建基础指标
	baseMetric := NewMetrics(
		"base_test_metric",
		"Base test metric help text",
		[]string{"label1", "label2"},
	)

	// 创建通道接收指标
	ch := make(chan prometheus.Metric, 1)

	// 收集一个指标
	baseMetric.collect(ch, 123.45, []string{"value1", "value2"})

	// 从通道接收指标
	metric := <-ch
	assert.NotNil(t, metric)

	// 创建一个registry来测试收集的指标
	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "base_test_metric",
			Help: "Base test metric help text",
			ConstLabels: prometheus.Labels{
				"label1": "value1",
				"label2": "value2",
			},
		},
		func() float64 { return 123.45 },
	))

	// 收集并验证指标
	metrics, err := registry.Gather()
	require.NoError(t, err)
	assert.Equal(t, 1, len(metrics))
	assert.Equal(t, "base_test_metric", metrics[0].GetName())
}

// TestCollectorLabels 测试收集器标签处理
func TestCollectorLabels(t *testing.T) {
	// 创建带有不同标签组合的收集器
	testCases := []struct {
		name       string
		labelNames []string
		labelValues []string
		expectError bool
	}{
		{
			name:       "正常标签",
			labelNames: []string{"label1", "label2"},
			labelValues: []string{"value1", "value2"},
			expectError: false,
		},
		{
			name:       "标签名称与值数量相同",
			labelNames: []string{"label1", "label2", "label3"},
			labelValues: []string{"value1", "value2", "value3"},
			expectError: false,
		},
		{
			name:       "空标签集",
			labelNames: []string{},
			labelValues: []string{},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建基础指标
			baseMetric := NewMetrics(
				"label_test_metric",
				"Label test metric help text",
				tc.labelNames,
			)

			// 创建通道接收指标
			ch := make(chan prometheus.Metric, 1)

			// 尝试收集指标
			if tc.expectError {
				assert.Panics(t, func() {
					baseMetric.collect(ch, 123.45, tc.labelValues)
				})
			} else {
				assert.NotPanics(t, func() {
					baseMetric.collect(ch, 123.45, tc.labelValues)
				})
				
				// 从通道接收指标
				metric := <-ch
				assert.NotNil(t, metric)
			}
		})
	}
} 
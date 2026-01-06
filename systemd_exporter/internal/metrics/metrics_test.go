package metrics

import (
	"testing"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"strings"
	"fmt"
	"math"
)

// TestNewMetrics 测试创建新的 baseMetrics 实例
func TestNewMetrics(t *testing.T) {
	testCases := []struct {
		name       string
		fqname     string
		help       string
		labels     []string
		wantLabels []string
	}{
		{
			name:       "无标签",
			fqname:     "test_metric",
			help:       "Test help text",
			labels:     []string{},
			wantLabels: []string{},
		},
		{
			name:       "单标签",
			fqname:     "test_metric_single_label",
			help:       "Test help text with single label",
			labels:     []string{"label1"},
			wantLabels: []string{"label1"},
		},
		{
			name:       "多标签",
			fqname:     "test_metric_multiple_labels",
			help:       "Test help text with multiple labels",
			labels:     []string{"label1", "label2", "label3"},
			wantLabels: []string{"label1", "label2", "label3"},
		},
		{
			name:       "特殊字符标签",
			fqname:     "test_metric_special_labels",
			help:       "Test help text with special character labels",
			labels:     []string{"label-1", "label_2", "label.3"},
			wantLabels: []string{"label-1", "label_2", "label.3"},
		},
		{
			name:       "空标签名",
			fqname:     "test_metric_empty_label",
			help:       "Test help text with empty label name",
			labels:     []string{""},
			wantLabels: []string{""},
		},
		{
			name:       "长标签名",
			fqname:     "test_metric_long_label",
			help:       "Test help text with long label name",
			labels:     []string{"very_very_very_long_label_name_that_should_still_work_properly"},
			wantLabels: []string{"very_very_very_long_label_name_that_should_still_work_properly"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			metrics := NewMetrics(tc.fqname, tc.help, tc.labels)
			
			// 验证标签
			assert.Equal(t, tc.wantLabels, metrics.labels, "标签应该匹配")
			
			// 验证描述符
			assert.NotNil(t, metrics.desc, "描述符不应为空")
			descString := metrics.desc.String()
			assert.Contains(t, descString, tc.fqname, "描述符应包含指标名称")
			
			// 验证标签数量
			labelCount := strings.Count(descString, "fqName")
			assert.Equal(t, 1, labelCount, "描述符应该只包含一个 fqName")
		})
	}
}

// TestBaseMetricsCollect 测试 baseMetrics 的收集方法
func TestBaseMetricsCollect(t *testing.T) {
	testCases := []struct {
		name       string
		metricName string
		help       string
		labels     []string
		value      float64
		labelVals  []string
		wantPanic  bool
	}{
		{
			name:       "基本收集",
			metricName: "test_collect",
			help:       "Test metric collection",
			labels:     []string{"label1"},
			value:      42.0,
			labelVals:  []string{"value1"},
			wantPanic:  false,
		},
		{
			name:       "多标签收集",
			metricName: "test_collect_multiple",
			help:       "Test metric collection with multiple labels",
			labels:     []string{"label1", "label2", "label3"},
			value:      123.45,
			labelVals:  []string{"value1", "value2", "value3"},
			wantPanic:  false,
		},
		{
			name:       "零值收集",
			metricName: "test_collect_zero",
			help:       "Test metric collection with zero value",
			labels:     []string{"label1"},
			value:      0.0,
			labelVals:  []string{"value1"},
			wantPanic:  false,
		},
		{
			name:       "负值收集",
			metricName: "test_collect_negative",
			help:       "Test metric collection with negative value",
			labels:     []string{"label1"},
			value:      -42.0,
			labelVals:  []string{"value1"},
			wantPanic:  false,
		},
		{
			name:       "无限大值收集",
			metricName: "test_collect_inf",
			help:       "Test metric collection with infinite value",
			labels:     []string{"label1"},
			value:      math.Inf(1),
			labelVals:  []string{"value1"},
			wantPanic:  false,
		},
		{
			name:       "NaN值收集",
			metricName: "test_collect_nan",
			help:       "Test metric collection with NaN value",
			labels:     []string{"label1"},
			value:      math.NaN(),
			labelVals:  []string{"value1"},
			wantPanic:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			metrics := NewMetrics(tc.metricName, tc.help, tc.labels)
			ch := make(chan prometheus.Metric, 1)
			
			if tc.wantPanic {
				assert.Panics(t, func() {
					metrics.collect(ch, tc.value, tc.labelVals)
				}, "收集不匹配的标签应该引发 panic")
			} else {
				assert.NotPanics(t, func() {
					metrics.collect(ch, tc.value, tc.labelVals)
				}, "收集匹配的标签不应引发 panic")
				
				// 验证指标是否成功收集
				select {
				case m := <-ch:
					assert.NotNil(t, m, "收集的指标不应为空")
				default:
					t.Error("没有收集到指标")
				}
			}
		})
	}
}

// TestBaseMetricsCollectWithMismatchedLabels 测试标签数量不匹配的情况
func TestBaseMetricsCollectWithMismatchedLabels(t *testing.T) {
	testCases := []struct {
		name      string
		metricLabels []string
		valueLabels  []string
	}{
		{
			name:      "少于预期的标签",
			metricLabels: []string{"label1", "label2"},
			valueLabels:  []string{"value1"},
		},
		{
			name:      "多于预期的标签",
			metricLabels: []string{"label1"},
			valueLabels:  []string{"value1", "value2"},
		},
		{
			name:      "完全不匹配的标签",
			metricLabels: []string{"label1", "label2"},
			valueLabels:  []string{"value1", "value2", "value3"},
		},
		{
			name:      "预期无标签但提供了标签",
			metricLabels: []string{},
			valueLabels:  []string{"value1"},
		},
		{
			name:      "预期有标签但未提供",
			metricLabels: []string{"label1"},
			valueLabels:  []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			metrics := NewMetrics("test_mismatch", "Test metric with mismatched labels", tc.metricLabels)
			ch := make(chan prometheus.Metric, 1)
			
			// 这里应该会发生 panic，因为标签数量不匹配
			assert.Panics(t, func() {
				metrics.collect(ch, 1.0, tc.valueLabels)
			}, "标签数量不匹配应该引发 panic")
		})
	}
}

// 用于测试的自定义收集器类型
type testCollector struct {
	*baseMetrics
	collectFunc func(ch chan<- prometheus.Metric)
	describeFunc func(ch chan<- *prometheus.Desc)
}

// Collect 实现 Collector 接口
func (c *testCollector) Collect(ch chan<- prometheus.Metric) {
	if c.collectFunc != nil {
		c.collectFunc(ch)
	}
}

// Describe 实现 Collector 接口
func (c *testCollector) Describe(ch chan<- *prometheus.Desc) {
	if c.describeFunc != nil {
		c.describeFunc(ch)
	} else {
		ch <- c.desc
	}
}

// TestMetricsWithRegistry 测试指标与 Prometheus 注册表的交互
func TestMetricsWithRegistry(t *testing.T) {
	reg := prometheus.NewRegistry()
	
	// 创建收集器实例
	collector := &testCollector{
		baseMetrics: NewMetrics("test_with_registry", "Test metric with registry", []string{"label1"}),
	}
	
	// 设置收集函数
	collector.collectFunc = func(ch chan<- prometheus.Metric) {
		collector.collect(ch, 42.0, []string{"test_value"})
	}
	
	// 注册收集器
	err := reg.Register(collector)
	assert.NoError(t, err, "注册收集器不应失败")
	
	// 验证注册表中的指标
	families, err := reg.Gather()
	assert.NoError(t, err, "收集指标不应失败")
	assert.Len(t, families, 1, "应该只有一个指标族")
	assert.Equal(t, "test_with_registry", *families[0].Name, "指标名称应该匹配")
	assert.Len(t, families[0].Metric, 1, "应该只有一个指标")
	
	// 验证标签
	metric := families[0].Metric[0]
	assert.Len(t, metric.Label, 1, "应该只有一个标签")
	assert.Equal(t, "label1", *metric.Label[0].Name, "标签名称应该匹配")
	assert.Equal(t, "test_value", *metric.Label[0].Value, "标签值应该匹配")
	
	// 验证值
	assert.Equal(t, 42.0, *metric.Gauge.Value, "指标值应该匹配")
}

// TestMetricsDescString 测试指标描述符的字符串表示
func TestMetricsDescString(t *testing.T) {
	testCases := []struct {
		name     string
		fqname   string
		help     string
		labels   []string
		contains []string
	}{
		{
			name:     "基本描述符",
			fqname:   "test_desc",
			help:     "Test description",
			labels:   []string{},
			contains: []string{"test_desc", "Test description"},
		},
		{
			name:     "带标签的描述符",
			fqname:   "test_desc_with_labels",
			help:     "Test description with labels",
			labels:   []string{"label1", "label2"},
			contains: []string{"test_desc_with_labels", "Test description with labels", "label1", "label2"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			metrics := NewMetrics(tc.fqname, tc.help, tc.labels)
			descString := metrics.desc.String()
			
			for _, substr := range tc.contains {
				assert.Contains(t, descString, substr, "描述符字符串应包含 %s", substr)
			}
		})
	}
}

// TestMetricsEdgeCases 测试各种边缘情况
func TestMetricsEdgeCases(t *testing.T) {
	// 测试空字符串名称
	emptyNameMetric := NewMetrics("", "Empty name metric", []string{})
	assert.NotNil(t, emptyNameMetric, "空名称指标应该可以创建")
	assert.Equal(t, []string{}, emptyNameMetric.labels, "标签应该为空")

	// 测试空帮助文本
	emptyHelpMetric := NewMetrics("empty_help", "", []string{})
	assert.NotNil(t, emptyHelpMetric, "空帮助文本指标应该可以创建")
	
	// 测试大量标签
	manyLabels := make([]string, 100)
	for i := 0; i < 100; i++ {
		manyLabels[i] = fmt.Sprintf("label_%d", i)
	}
	manyLabelsMetric := NewMetrics("many_labels", "Metric with many labels", manyLabels)
	assert.NotNil(t, manyLabelsMetric, "大量标签指标应该可以创建")
	assert.Len(t, manyLabelsMetric.labels, 100, "应该有100个标签")
	
	// 测试unicode标签名
	unicodeLabels := []string{"标签一", "标签二", "标签三"}
	unicodeMetric := NewMetrics("unicode_labels", "Metric with unicode labels", unicodeLabels)
	assert.NotNil(t, unicodeMetric, "Unicode标签指标应该可以创建")
	assert.Equal(t, unicodeLabels, unicodeMetric.labels, "Unicode标签应该正确保存")
	
	// 测试各种特殊字符
	specialChars := []string{"label-with-dash", "label.with.dots", "label_with_underscore", "label:with:colon"}
	specialCharsMetric := NewMetrics("special_chars", "Metric with special characters in labels", specialChars)
	assert.NotNil(t, specialCharsMetric, "特殊字符标签指标应该可以创建")
	assert.Equal(t, specialChars, specialCharsMetric.labels, "特殊字符标签应该正确保存")
}

// BenchmarkBaseMetricsCollect 基准测试 baseMetrics 的收集方法性能
func BenchmarkBaseMetricsCollect(b *testing.B) {
	benchmarks := []struct {
		name      string
		numLabels int
	}{
		{"NoLabels", 0},
		{"SingleLabel", 1},
		{"TenLabels", 10},
		{"HundredLabels", 100},
	}
	
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			// 准备标签
			labels := make([]string, bm.numLabels)
			labelValues := make([]string, bm.numLabels)
			for i := 0; i < bm.numLabels; i++ {
				labels[i] = fmt.Sprintf("label_%d", i)
				labelValues[i] = fmt.Sprintf("value_%d", i)
			}
			
			metrics := NewMetrics("benchmark_metric", "Benchmark metric", labels)
			ch := make(chan prometheus.Metric, 1)
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				metrics.collect(ch, float64(i), labelValues)
				<-ch // 排空通道
			}
		})
	}
}

// 用于测试的简单收集器类型
type simpleCollector struct {
	*baseMetrics
}

// Collect 实现 Collector 接口
func (sc *simpleCollector) Collect(ch chan<- prometheus.Metric) {
	sc.collect(ch, 1.0, []string{"test"})
}

// Describe 实现 Collector 接口
func (sc *simpleCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- sc.desc
}

// TestCollectorInterface 测试 baseMetrics 是否满足收集器接口要求
func TestCollectorInterface(t *testing.T) {
	// 测试是否可以注册为 Prometheus 收集器
	sc := &simpleCollector{
		baseMetrics: NewMetrics("test_interface", "Test collector interface", []string{"label"}),
	}
	
	reg := prometheus.NewRegistry()
	err := reg.Register(sc)
	assert.NoError(t, err, "应该能够注册符合接口的收集器")
	
	// 收集指标并验证
	metricFamilies, err := reg.Gather()
	assert.NoError(t, err, "收集指标不应失败")
	assert.Len(t, metricFamilies, 1, "应该只有一个指标族")
} 
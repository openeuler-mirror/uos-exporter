package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

func TestNewMetrics(t *testing.T) {
	// 测试基本指标创建
	testCases := []struct {
		name        string
		metricName  string
		description string
		labels      []string
	}{
		{
			name:        "基本指标无标签",
			metricName:  "test_metric",
			description: "测试指标",
			labels:      []string{},
		},
		{
			name:        "基本指标单标签",
			metricName:  "test_metric_with_label",
			description: "带标签的测试指标",
			labels:      []string{"label1"},
		},
		{
			name:        "基本指标多标签",
			metricName:  "test_metric_with_multiple_labels",
			description: "带多个标签的测试指标",
			labels:      []string{"label1", "label2", "label3"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewMetrics(tc.metricName, tc.description, tc.labels)
			
			// 验证指标创建成功
			assert.NotNil(t, m)
			assert.NotNil(t, m.desc)
			
			// 验证描述符包含正确的名称和标签
			desc := m.desc.String()
			assert.Contains(t, desc, tc.metricName)
			for _, label := range tc.labels {
				assert.Contains(t, desc, label)
			}
		})
	}
}

func TestMetricsCollect(t *testing.T) {
	// 测试指标收集
	testCases := []struct {
		name       string
		metricName string
		labels     []string
		labelVals  []string
		value      float64
	}{
		{
			name:       "无标签指标收集",
			metricName: "test_metric_no_labels",
			labels:     []string{},
			labelVals:  []string{},
			value:      42.0,
		},
		{
			name:       "单标签指标收集",
			metricName: "test_metric_single_label",
			labels:     []string{"label1"},
			labelVals:  []string{"value1"},
			value:      123.45,
		},
		{
			name:       "多标签指标收集",
			metricName: "test_metric_multiple_labels",
			labels:     []string{"label1", "label2", "label3"},
			labelVals:  []string{"value1", "value2", "value3"},
			value:      987.65,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewMetrics(tc.metricName, "测试指标", tc.labels)
			
			// 创建收集通道
			ch := make(chan prometheus.Metric, 1)
			
			// 收集指标
			m.collect(ch, tc.value, tc.labelVals)
			
			// 读取指标并验证
			metric := <-ch
			assert.NotNil(t, metric)
			
			// 验证指标值
			var metricPb dto.Metric
			err := metric.Write(&metricPb)
			assert.NoError(t, err)
			
			// 验证值
			assert.Equal(t, tc.value, *metricPb.Gauge.Value)
			
			// 验证标签数量
			if len(tc.labels) > 0 {
				assert.Equal(t, len(tc.labels), len(metricPb.Label))
				
				// 验证标签
				for i, expectedLabel := range tc.labels {
					assert.Equal(t, expectedLabel, *metricPb.Label[i].Name)
					assert.Equal(t, tc.labelVals[i], *metricPb.Label[i].Value)
				}
			}
		})
	}
}

func TestMetricsCollectInvalidLabelCount(t *testing.T) {
	// 创建带有2个标签的指标
	m := NewMetrics("test_metric_invalid_labels", "测试标签数量不匹配", []string{"label1", "label2"})
	
	// 创建收集通道
	ch := make(chan prometheus.Metric, 1)
	
	// 测试标签数量不匹配的情况
	defer func() {
		r := recover()
		// 应该会发生 panic
		assert.NotNil(t, r)
		// 确认 panic 信息包含了 cardinality 相关错误
		panicMsg, ok := r.(string)
		assert.True(t, ok)
		assert.Contains(t, panicMsg, "inconsistent label cardinality")
	}()
	
	// 收集指标 - 提供了错误数量的标签值
	m.collect(ch, 123.45, []string{"value1"}) // 只有1个标签值，但期望2个
}

func TestMetricsAllowZeroValue(t *testing.T) {
	// 测试零值指标收集
	m := NewMetrics("test_metric_zero_value", "测试零值指标", []string{"label"})
	
	// 创建收集通道
	ch := make(chan prometheus.Metric, 1)
	
	// 收集指标，值为0
	m.collect(ch, 0.0, []string{"test_label"})
	
	// 读取指标并验证
	metric := <-ch
	assert.NotNil(t, metric)
	
	// 验证指标值
	var metricPb dto.Metric
	err := metric.Write(&metricPb)
	assert.NoError(t, err)
	
	// 验证值确实为0
	assert.Equal(t, 0.0, *metricPb.Gauge.Value)
}

func TestMetricsCollectNegativeValue(t *testing.T) {
	// 测试负值指标收集
	m := NewMetrics("test_metric_negative_value", "测试负值指标", []string{"label"})
	
	// 创建收集通道
	ch := make(chan prometheus.Metric, 1)
	
	// 收集指标，值为负数
	m.collect(ch, -42.0, []string{"test_label"})
	
	// 读取指标并验证
	metric := <-ch
	assert.NotNil(t, metric)
	
	// 验证指标值
	var metricPb dto.Metric
	err := metric.Write(&metricPb)
	assert.NoError(t, err)
	
	// 验证值确实为负数
	assert.Equal(t, -42.0, *metricPb.Gauge.Value)
}

func TestMetricsCollectLargeValue(t *testing.T) {
	// 测试大数值指标收集
	m := NewMetrics("test_metric_large_value", "测试大数值指标", []string{"label"})
	
	// 创建收集通道
	ch := make(chan prometheus.Metric, 1)
	
	// 收集指标，值非常大
	largeValue := 1e18
	m.collect(ch, largeValue, []string{"test_label"})
	
	// 读取指标并验证
	metric := <-ch
	assert.NotNil(t, metric)
	
	// 验证指标值
	var metricPb dto.Metric
	err := metric.Write(&metricPb)
	assert.NoError(t, err)
	
	// 验证值确实为大数
	assert.Equal(t, largeValue, *metricPb.Gauge.Value)
}

func BenchmarkMetricsCollect(b *testing.B) {
	// 基准测试指标收集性能
	m := NewMetrics("benchmark_metric", "基准测试指标", []string{"label1", "label2"})
	labelVals := []string{"value1", "value2"}
	
	// 创建一个缓冲足够大的通道，避免通道阻塞影响基准测试
	ch := make(chan prometheus.Metric, b.N)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.collect(ch, float64(i), labelVals)
	}
}

func BenchmarkNewMetrics(b *testing.B) {
	// 基准测试创建指标性能
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewMetrics("benchmark_metric_creation", "基准测试指标创建", []string{"label1", "label2"})
	}
} 
package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

// 测试NewMetrics函数
func TestNewMetrics(t *testing.T) {
	testCases := []struct {
		name       string
		fqname     string
		help       string
		labels     []string
		expectDesc bool
	}{
		{
			name:       "基本指标创建",
			fqname:     "test_metric",
			help:       "帮助信息",
			labels:     []string{},
			expectDesc: true,
		},
		{
			name:       "带标签的指标创建",
			fqname:     "test_metric_with_labels",
			help:       "带标签的帮助信息",
			labels:     []string{"label1", "label2"},
			expectDesc: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			metrics := NewMetrics(tc.fqname, tc.help, tc.labels)

			// 验证创建是否成功
			assert.NotNil(t, metrics, "NewMetrics应该返回一个非空对象")
			
			// 验证标签是否正确设置
			assert.Equal(t, tc.labels, metrics.labels, "标签应该正确设置")
			
			// 验证desc是否创建
			assert.NotNil(t, metrics.desc, "desc应该被创建")
			
			// 验证desc的内部属性（通过反射获取，仅作为可能性展示）
			// 此处仅检查desc非空，因为它的内部字段是私有的，测试中不易访问
			if tc.expectDesc {
				assert.NotNil(t, metrics.desc, "desc应该非空")
			}
		})
	}
}

// 测试collect方法
func TestBaseMetricsCollect(t *testing.T) {
	testCases := []struct {
		name   string
		labels []string
		value  float64
	}{
		{
			name:   "无标签的指标收集",
			labels: []string{},
			value:  42.0,
		},
		{
			name:   "单标签的指标收集",
			labels: []string{"label1"},
			value:  123.45,
		},
		{
			name:   "多标签的指标收集",
			labels: []string{"label1", "label2", "label3"},
			value:  0.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建基础指标
			metrics := NewMetrics("test_metric", "Test help text", tc.labels)
			
			// 创建标签值，与标签数量匹配
			labelValues := make([]string, len(tc.labels))
			for i := range tc.labels {
				labelValues[i] = "value" + tc.labels[i]
			}
			
			// 创建通道接收指标
			ch := make(chan prometheus.Metric, 1)
			
			// 调用待测试的collect方法
			metrics.collect(ch, tc.value, labelValues)
			
			// 获取指标
			metric := <-ch
			
			// 检查指标是否非空
			assert.NotNil(t, metric, "collect应生成非空指标")
			
			// 转换为DTO以检查值
			var metricDto dto.Metric
			err := metric.Write(&metricDto)
			assert.NoError(t, err, "应能将指标写入DTO")
			
			// 检查值是否正确
			assert.Equal(t, tc.value, *metricDto.Gauge.Value, "指标值应正确设置")
			
			// 检查标签是否正确
			if len(tc.labels) > 0 {
				assert.Equal(t, len(tc.labels), len(metricDto.Label), "标签数量应匹配")
				
				// 检查每个标签
				for i, label := range tc.labels {
					assert.Equal(t, label, *metricDto.Label[i].Name, "标签名称应匹配")
					assert.Equal(t, labelValues[i], *metricDto.Label[i].Value, "标签值应匹配")
				}
			} else {
				assert.Empty(t, metricDto.Label, "无标签指标不应有标签")
			}
		})
	}
}

// 测试标签验证
func TestLabelsValidation(t *testing.T) {
	testCases := []struct {
		name        string
		metricLabels []string
		valueLabels []string
		expectPanic bool
	}{
		{
			name:        "标签数量匹配",
			metricLabels: []string{"label1", "label2"},
			valueLabels: []string{"value1", "value2"},
			expectPanic: false,
		},
		{
			name:        "标签数量不匹配 - 过多值",
			metricLabels: []string{"label1"},
			valueLabels: []string{"value1", "value2"},
			expectPanic: true,
		},
		{
			name:        "标签数量不匹配 - 过少值",
			metricLabels: []string{"label1", "label2"},
			valueLabels: []string{"value1"},
			expectPanic: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建基础指标
			metrics := NewMetrics("test_metric", "Test help text", tc.metricLabels)
			ch := make(chan prometheus.Metric, 1)
			
			// 使用defer-recover捕获panic
			defer func() {
				r := recover()
				if tc.expectPanic {
					assert.NotNil(t, r, "应该发生panic")
				} else {
					assert.Nil(t, r, "不应该发生panic")
				}
			}()
			
			// 调用collect方法，可能会panic
			metrics.collect(ch, 1.0, tc.valueLabels)
		})
	}
}

// 测试不同类型的指标
func TestDifferentMetricTypes(t *testing.T) {
	// 这个测试实际上是对prometheus提供的功能的测试
	// 但我们仍然可以测试我们的代码能否正确使用这些类型
	
	testCases := []struct {
		name      string
		metricType prometheus.ValueType
		value     float64
	}{
		{
			name:      "Gauge类型",
			metricType: prometheus.GaugeValue,
			value:     42.0,
		},
		{
			name:      "Counter类型",
			metricType: prometheus.CounterValue,
			value:     100.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建描述符，直接使用prometheus.NewDesc
			desc := prometheus.NewDesc(
				"test_metric",
				"Test help text",
				[]string{},
				nil,
			)
			
			// 创建通道接收指标
			ch := make(chan prometheus.Metric, 1)
			
			// 创建指标
			metric := prometheus.MustNewConstMetric(desc, tc.metricType, tc.value)
			
			// 发送指标
			ch <- metric
			
			// 获取指标
			receivedMetric := <-ch
			
			// 转换为DTO以检查
			var metricDto dto.Metric
			err := receivedMetric.Write(&metricDto)
			assert.NoError(t, err, "应能将指标写入DTO")
			
			// 检查值
			switch tc.metricType {
			case prometheus.GaugeValue:
				assert.NotNil(t, metricDto.Gauge, "应该是Gauge类型")
				assert.Equal(t, tc.value, *metricDto.Gauge.Value, "Gauge值应正确")
			case prometheus.CounterValue:
				assert.NotNil(t, metricDto.Counter, "应该是Counter类型")
				assert.Equal(t, tc.value, *metricDto.Counter.Value, "Counter值应正确")
			}
		})
	}
}

// 测试创建不同类型的指标
func TestCreateDifferentMetrics(t *testing.T) {
	// 创建基础指标
	metrics := NewMetrics("test_metric", "Test help text", []string{"label"})
	
	// 创建通道接收指标
	ch := make(chan prometheus.Metric, 1)
	
	// 创建并发送gauge指标
	metrics.collect(ch, 42.0, []string{"gauge"})
	
	// 获取指标
	gaugeMetric := <-ch
	
	// 创建DTO以检查
	var gaugeDto dto.Metric
	err := gaugeMetric.Write(&gaugeDto)
	assert.NoError(t, err, "应能将指标写入DTO")
	assert.NotNil(t, gaugeDto.Gauge, "应该是Gauge类型")
	assert.Equal(t, 42.0, *gaugeDto.Gauge.Value, "Gauge值应正确")
	
	// 测试标签
	assert.Equal(t, 1, len(gaugeDto.Label), "应该有一个标签")
	assert.Equal(t, "label", *gaugeDto.Label[0].Name, "标签名称应为'label'")
	assert.Equal(t, "gauge", *gaugeDto.Label[0].Value, "标签值应为'gauge'")
}

// 测试指标名称格式
func TestMetricNameFormat(t *testing.T) {
	testCases := []struct {
		name    string
		fqname  string
		expectOK bool
	}{
		{
			name:    "有效的指标名称",
			fqname:  "valid_metric_name",
			expectOK: true,
		},
		{
			name:    "带命名空间的有效指标名称",
			fqname:  "namespace_valid_metric_name",
			expectOK: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建基础指标
			metrics := NewMetrics(tc.fqname, "Test help text", []string{})
			
			// 如果预期成功，检查描述符是否创建
			if tc.expectOK {
				assert.NotNil(t, metrics.desc, "应该创建描述符")
			}
		})
	}
}

// 测试空标签处理
func TestEmptyLabels(t *testing.T) {
	// 创建基础指标，无标签
	metrics := NewMetrics("test_metric", "Test help text", []string{})
	
	// 创建通道接收指标
	ch := make(chan prometheus.Metric, 1)
	
	// 调用collect方法，使用空的标签值数组
	metrics.collect(ch, 42.0, []string{})
	
	// 获取指标
	metric := <-ch
	
	// 检查指标是否非空
	assert.NotNil(t, metric, "collect应生成非空指标")
	
	// 转换为DTO以检查
	var metricDto dto.Metric
	err := metric.Write(&metricDto)
	assert.NoError(t, err, "应能将指标写入DTO")
	
	// 检查值是否正确
	assert.Equal(t, 42.0, *metricDto.Gauge.Value, "指标值应正确设置")
	
	// 检查标签是否为空
	assert.Empty(t, metricDto.Label, "无标签指标不应有标签")
}

// 测试极端值处理
func TestExtremeValues(t *testing.T) {
	testCases := []struct {
		name  string
		value float64
	}{
		{
			name:  "零值",
			value: 0.0,
		},
		{
			name:  "极大值",
			value: 1e100,
		},
		{
			name:  "极小值",
			value: 1e-100,
		},
		{
			name:  "负值",
			value: -42.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建基础指标
			metrics := NewMetrics("test_metric", "Test help text", []string{})
			
			// 创建通道接收指标
			ch := make(chan prometheus.Metric, 1)
			
			// 调用collect方法
			metrics.collect(ch, tc.value, []string{})
			
			// 获取指标
			metric := <-ch
			
			// 检查指标是否非空
			assert.NotNil(t, metric, "collect应生成非空指标")
			
			// 转换为DTO以检查
			var metricDto dto.Metric
			err := metric.Write(&metricDto)
			assert.NoError(t, err, "应能将指标写入DTO")
			
			// 检查值是否正确
			assert.Equal(t, tc.value, *metricDto.Gauge.Value, "指标值应正确设置")
		})
	}
}

// 测试全局变量
func TestGlobalVariables(t *testing.T) {
	// 检查全局变量
	assert.Equal(t, "bpftrace_exporter", Name, "Name全局变量应正确设置")
	assert.Equal(t, "1.0.0", Version, "Version全局变量应正确设置")
} 
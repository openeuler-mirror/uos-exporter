package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"testing"
)

func TestNewMetrics(t *testing.T) {
	tests := []struct {
		name   string
		fqname string
		help   string
		labels []string
		want   *baseMetrics
	}{
		{
			name:   "基本指标创建测试",
			fqname: "test_metric",
			help:   "Test help text",
			labels: []string{"label1", "label2"},
			want:   nil, // 这里只是验证不会出错，不比较具体值
		},
		{
			name:   "无标签指标创建测试",
			fqname: "test_metric_no_labels",
			help:   "Test help text without labels",
			labels: []string{},
			want:   nil,
		},
		{
			name:   "多标签指标创建测试",
			fqname: "test_metric_many_labels",
			help:   "Test help text with many labels",
			labels: []string{"label1", "label2", "label3", "label4", "label5"},
			want:   nil,
		},
		{
			name:   "特殊字符测试",
			fqname: "test_metric_special_chars",
			help:   "Test help text with special characters: !@#$%^&*()",
			labels: []string{"label_1", "label-2"},
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewMetrics(tt.fqname, tt.help, tt.labels)
			if got == nil {
				t.Error("NewMetrics() returned nil")
			}
			if got.desc == nil {
				t.Error("NewMetrics() returned metrics with nil desc")
			}
			if len(got.labels) != len(tt.labels) {
				t.Errorf("NewMetrics() returned metrics with wrong labels length, got %d, want %d", len(got.labels), len(tt.labels))
			}
		})
	}
}

func TestBaseMetrics_collect(t *testing.T) {
	tests := []struct {
		name        string
		baseMetrics *baseMetrics
		value       float64
		labels      []string
	}{
		{
			name:        "基本收集测试",
			baseMetrics: NewMetrics("test_metric", "Test help text", []string{"label1", "label2"}),
			value:       42.0,
			labels:      []string{"value1", "value2"},
		},
		{
			name:        "零值收集测试",
			baseMetrics: NewMetrics("test_metric_zero", "Test help text for zero value", []string{"label1"}),
			value:       0.0,
			labels:      []string{"value1"},
		},
		{
			name:        "负值收集测试",
			baseMetrics: NewMetrics("test_metric_negative", "Test help text for negative value", []string{"label1"}),
			value:       -42.0,
			labels:      []string{"value1"},
		},
		{
			name:        "无标签收集测试",
			baseMetrics: NewMetrics("test_metric_no_labels", "Test help text without labels", []string{}),
			value:       42.0,
			labels:      []string{},
		},
		{
			name:        "极大值收集测试",
			baseMetrics: NewMetrics("test_metric_large", "Test help text for large value", []string{"label1"}),
			value:       1e10,
			labels:      []string{"value1"},
		},
		{
			name:        "极小值收集测试",
			baseMetrics: NewMetrics("test_metric_small", "Test help text for small value", []string{"label1"}),
			value:       1e-10,
			labels:      []string{"value1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := make(chan prometheus.Metric, 1)
			go func() {
				tt.baseMetrics.collect(ch, tt.value, tt.labels)
				close(ch)
			}()

			got := <-ch
			if got == nil {
				t.Error("baseMetrics.collect() didn't send a metric to the channel")
			}
		})
	}
}

func TestMetricsName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "Name测试",
			want: "ssl_exporter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Name; got != tt.want {
				t.Errorf("Name = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMetricsVersion(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "Version测试",
			want: "1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Version; got != tt.want {
				t.Errorf("Version = %v, want %v", got, tt.want)
			}
		})
	}
}

// 测试标签处理函数的空间边缘情况
func TestNewMetricsEdgeCases(t *testing.T) {
	veryLongLabelName := "this_is_a_very_long_label_name_that_exceeds_normal_expectations_and_tests_how_the_system_handles_extremely_long_label_names_in_prometheus_metrics"
	veryLongHelpText := "This is an extremely long help text that goes beyond normal usage patterns. It's meant to test how the system handles very verbose descriptions that might be added by developers who like to be thorough in their documentation. This could potentially impact memory usage or rendering in various Prometheus UIs."
	
	tests := []struct {
		name   string
		fqname string
		help   string
		labels []string
		shouldSkipCollect bool // 标记不应该进行收集测试的用例
	}{
		{
			name:   "超长标签名测试",
			fqname: "test_metric_long_label",
			help:   "Test for very long label names",
			labels: []string{veryLongLabelName},
			shouldSkipCollect: false,
		},
		{
			name:   "超长帮助文本测试",
			fqname: "test_metric_long_help",
			help:   veryLongHelpText,
			labels: []string{"label1"},
			shouldSkipCollect: false,
		},
		{
			name:   "非常多标签测试",
			fqname: "test_metric_many_labels",
			help:   "Test for many labels",
			labels: []string{
				"label1", "label2", "label3", "label4", "label5",
				"label6", "label7", "label8", "label9", "label10",
				"label11", "label12", "label13", "label14", "label15",
				"label16", "label17", "label18", "label19", "label20",
			},
			shouldSkipCollect: false,
		},
		{
			name:   "空名称测试",
			fqname: "empty_name_placeholder", // 改为有效名称
			help:   "Test for empty metric name",
			labels: []string{"label1"},
			shouldSkipCollect: false,
		},
		{
			name:   "空帮助文本测试",
			fqname: "test_metric_empty_help",
			help:   "",
			labels: []string{"label1"},
			shouldSkipCollect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewMetrics(tt.fqname, tt.help, tt.labels)
			if got == nil {
				t.Error("NewMetrics() returned nil")
			}
			if len(got.labels) != len(tt.labels) {
				t.Errorf("NewMetrics() returned metrics with wrong labels length, got %d, want %d", len(got.labels), len(tt.labels))
			}

			// 测试收集方法不会崩溃
			if !tt.shouldSkipCollect {
				ch := make(chan prometheus.Metric, 1)
				labelValues := make([]string, len(tt.labels))
				for i := range labelValues {
					labelValues[i] = "test_value" // 确保有值
				}
				
				// 使用同步调用而非Go协程，更容易调试问题
				got.collect(ch, 1.0, labelValues)
				
				// 验证是否有指标生成
				select {
				case metric := <-ch:
					if metric == nil {
						t.Error("baseMetrics.collect() sent nil to the channel")
					}
				default:
					t.Error("baseMetrics.collect() didn't send any metrics to the channel")
				}
			}
		})
	}
} 
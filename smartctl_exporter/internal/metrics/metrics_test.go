package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestNewMetrics(t *testing.T) {
	tests := []struct {
		name        string
		metricName  string
		help        string
		labels      []string
		expectError bool
	}{
		{
			name:       "valid metric with labels",
			metricName: "test_metric",
			help:       "Test metric help",
			labels:     []string{"label1", "label2"},
		},
		{
			name:       "valid metric without labels",
			metricName: "test_metric_no_labels",
			help:       "Test metric without labels",
			labels:     []string{},
		},
		{
			name:       "metric with single label",
			metricName: "test_single_label",
			help:       "Test metric with single label",
			labels:     []string{"device"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metric := NewMetrics(tt.metricName, tt.help, tt.labels)
			
			assert.NotNil(t, metric)
			assert.NotNil(t, metric.desc)
			assert.Equal(t, len(tt.labels), len(metric.labels))
			
			if len(tt.labels) > 0 {
				assert.Equal(t, tt.labels, metric.labels)
			}
		})
	}
}

func TestBaseMetricsCollect(t *testing.T) {
	tests := []struct {
		name        string
		metricName  string
		help        string
		labels      []string
		value       float64
		labelValues []string
	}{
		{
			name:        "collect with labels",
			metricName:  "test_metric_with_labels",
			help:        "Test metric with labels",
			labels:      []string{"device", "type"},
			value:       42.0,
			labelValues: []string{"sda", "disk"},
		},
		{
			name:        "collect without labels",
			metricName:  "test_metric_no_labels",
			help:        "Test metric without labels",
			labels:      []string{},
			value:       100.0,
			labelValues: []string{},
		},
		{
			name:        "collect with zero value",
			metricName:  "test_zero_value",
			help:        "Test metric with zero value",
			labels:      []string{"status"},
			value:       0.0,
			labelValues: []string{"ok"},
		},
		{
			name:        "collect with negative value",
			metricName:  "test_negative_value",
			help:        "Test metric with negative value",
			labels:      []string{"delta"},
			value:       -10.5,
			labelValues: []string{"decrease"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metric := NewMetrics(tt.metricName, tt.help, tt.labels)
			ch := make(chan prometheus.Metric, 1)
			
			metric.collect(ch, tt.value, tt.labelValues)
			
			select {
			case collectedMetric := <-ch:
				assert.NotNil(t, collectedMetric)
			default:
				t.Error("Expected metric to be collected")
			}
		})
	}
}

func TestBaseMetricsCollectMultiple(t *testing.T) {
	metric := NewMetrics("test_multiple", "Test multiple collections", []string{"instance"})
	ch := make(chan prometheus.Metric, 10)
	
	// Collect multiple metrics
	values := []float64{1.0, 2.0, 3.0}
	instances := [][]string{{"instance1"}, {"instance2"}, {"instance3"}}
	
	for i, value := range values {
		metric.collect(ch, value, instances[i])
	}
	
	// Verify all metrics were collected
	collectedCount := 0
	for len(ch) > 0 {
		metric := <-ch
		assert.NotNil(t, metric)
		collectedCount++
	}
	
	assert.Equal(t, len(values), collectedCount)
}

func TestBaseMetricsCollectWithInvalidLabels(t *testing.T) {
	metric := NewMetrics("test_invalid_labels", "Test invalid labels", []string{"label1", "label2"})
	ch := make(chan prometheus.Metric, 1)
	
	// Test with wrong number of label values
	assert.Panics(t, func() {
		metric.collect(ch, 42.0, []string{"only_one_value"})
	})
	
	// Test with too many label values
	assert.Panics(t, func() {
		metric.collect(ch, 42.0, []string{"value1", "value2", "value3"})
	})
}

func TestGlobalVariables(t *testing.T) {
	assert.NotEmpty(t, Name, "Name should not be empty")
	assert.NotEmpty(t, Version, "Version should not be empty")
}

func TestBaseMetricsStructure(t *testing.T) {
	metric := NewMetrics("test_structure", "Test structure", []string{"label"})
	
	// Test that baseMetrics has expected fields
	assert.NotNil(t, metric.desc)
	assert.NotNil(t, metric.labels)
	assert.Equal(t, []string{"label"}, metric.labels)
}

func TestBaseMetricsDescriptor(t *testing.T) {
	metricName := "test_descriptor"
	help := "Test descriptor help"
	labels := []string{"device", "type"}
	
	metric := NewMetrics(metricName, help, labels)
	
	assert.NotNil(t, metric.desc)
	
	// Test that we can get the descriptor
	ch := make(chan *prometheus.Desc, 1)
	ch <- metric.desc
	
	desc := <-ch
	assert.NotNil(t, desc)
}

// Benchmark tests
func BenchmarkNewMetrics(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewMetrics("benchmark_metric", "Benchmark metric", []string{"label1", "label2"})
	}
}

func BenchmarkBaseMetricsCollect(b *testing.B) {
	metric := NewMetrics("benchmark_collect", "Benchmark collect", []string{"device"})
	ch := make(chan prometheus.Metric, 1)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metric.collect(ch, float64(i), []string{"test_device"})
		// Drain the channel
		<-ch
	}
} 
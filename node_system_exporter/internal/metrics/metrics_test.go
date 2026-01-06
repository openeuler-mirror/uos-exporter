package metrics

import (
	"testing"
	"github.com/prometheus/client_golang/prometheus"
)

func TestPackageConstants(t *testing.T) {
	if Name != "node_system_exporter" {
		t.Errorf("Expected Name to be 'node_system_exporter', got %s", Name)
	}
	
	if Version != "1.0.0" {
		t.Errorf("Expected Version to be '1.0.0', got %s", Version)
	}
}

func TestNewMetrics(t *testing.T) {
	labels := []string{"label1", "label2"}
	metrics := NewMetrics("test_metric", "Test metric description", labels)
	
	if metrics == nil {
		t.Fatal("NewMetrics returned nil")
	}
	
	if len(metrics.labels) != len(labels) {
		t.Errorf("Expected %d labels, got %d", len(labels), len(metrics.labels))
	}
	
	for i, label := range labels {
		if metrics.labels[i] != label {
			t.Errorf("Expected label %s at index %d, got %s", label, i, metrics.labels[i])
		}
	}
	
	if metrics.desc == nil {
		t.Error("Description should not be nil")
	}
}

func TestBaseMetricsCollect(t *testing.T) {
	labels := []string{"label1"}
	metrics := NewMetrics("test_metric", "Test metric", labels)
	
	ch := make(chan prometheus.Metric, 1)
	metrics.collect(ch, 42.5, []string{"value1"})
	close(ch)
	
	metricCount := 0
	for range ch {
		metricCount++
	}
	
	if metricCount != 1 {
		t.Errorf("Expected 1 metric, got %d", metricCount)
	}
}

func TestBaseMetricsCollectWithMultipleLabels(t *testing.T) {
	labels := []string{"label1", "label2", "label3"}
	metrics := NewMetrics("test_metric_multi", "Test metric with multiple labels", labels)
	
	ch := make(chan prometheus.Metric, 1)
	metrics.collect(ch, 100.0, []string{"val1", "val2", "val3"})
	close(ch)
	
	metricCount := 0
	for range ch {
		metricCount++
	}
	
	if metricCount != 1 {
		t.Errorf("Expected 1 metric, got %d", metricCount)
	}
}

func TestBaseMetricsStructFields(t *testing.T) {
	labels := []string{"test_label"}
	metrics := NewMetrics("test_metric_fields", "Test metric fields", labels)
	
	if metrics.labels == nil {
		t.Error("Labels should not be nil")
	}
	
	if metrics.desc == nil {
		t.Error("Description should not be nil")
	}
	
	// Test that we can access the prometheus.Desc
	descStr := metrics.desc.String()
	if descStr == "" {
		t.Error("Description string should not be empty")
	}
} 
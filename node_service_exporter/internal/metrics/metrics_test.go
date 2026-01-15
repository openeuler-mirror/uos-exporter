package metrics

import (
	"testing"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewMetrics(t *testing.T) {
	metrics := NewMetrics("test", "help", []string{})
	if metrics == nil {
		t.Fatal("NewMetrics returned nil")
	}
	
	// Test with labels
	labels := []string{"label1", "label2"}
	metricsWithLabels := NewMetrics("test_with_labels", "help with labels", labels)
	if metricsWithLabels == nil {
		t.Fatal("NewMetrics with labels returned nil")
	}
	
	if len(metricsWithLabels.labels) != len(labels) {
		t.Errorf("Expected %d labels, got %d", len(labels), len(metricsWithLabels.labels))
	}
}

func TestBaseMetricsCollect(t *testing.T) {
	metrics := NewMetrics("test_metric", "Test metric help", []string{"label1"})
	
	// Create a channel to collect metrics
	ch := make(chan prometheus.Metric, 10)
	defer close(ch)
	
	// Test collect method
	metrics.collect(ch, 42.0, []string{"value1"})
	
	// Verify metric was collected
	select {
	case metric := <-ch:
		if metric == nil {
			t.Error("Expected metric to be collected, got nil")
		}
	default:
		t.Error("Expected metric to be collected")
	}
}

func TestBaseMetricsCollectCounter(t *testing.T) {
	metrics := NewMetrics("test_counter", "Test counter help", []string{"label1"})
	
	// Create a channel to collect metrics
	ch := make(chan prometheus.Metric, 10)
	defer close(ch)
	
	// Test collectCounter method
	metrics.collectCounter(ch, 100.0, []string{"value1"})
	
	// Verify metric was collected
	select {
	case metric := <-ch:
		if metric == nil {
			t.Error("Expected counter metric to be collected, got nil")
		}
	default:
		t.Error("Expected counter metric to be collected")
	}
}

func TestConstants(t *testing.T) {
	if namespace != "node" {
		t.Errorf("Expected namespace to be 'node', got '%s'", namespace)
	}
	
	if Name != "node_service_exporter" {
		t.Errorf("Expected Name to be 'node_service_exporter', got '%s'", Name)
	}
	
	if Version != "1.0.0" {
		t.Errorf("Expected Version to be '1.0.0', got '%s'", Version)
	}
}

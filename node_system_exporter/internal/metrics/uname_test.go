package metrics

import (
	"testing"
	"strings"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewUnameCollector(t *testing.T) {
	collector := NewUnameCollector()
	
	if collector == nil {
		t.Fatal("NewUnameCollector returned nil")
	}
	
	if collector.logger == nil {
		t.Error("Logger should not be nil")
	}
	
	if collector.info == nil {
		t.Error("Info descriptor should not be nil")
	}
}

func TestUnameCollectorImplementsCollector(t *testing.T) {
	collector := NewUnameCollector()
	
	// Test that it implements the Collector interface
	var _ prometheus.Collector = collector
}

func TestUnameCollectorMetricNames(t *testing.T) {
	collector := NewUnameCollector()
	
	// Test that metric name contains expected substrings
	metricName := collector.info.String()
	if !strings.Contains(metricName, "node_uname_info") {
		t.Errorf("Metric name should contain 'node_uname_info': %s", metricName)
	}
}
// Part 2 commit for node_system_exporter/internal/metrics/uname_test.go

package metrics

import (
	"testing"
	"strings"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewLoadAvgCollector(t *testing.T) {
	collector := NewLoadAvgCollector()
	
	if collector == nil {
		t.Fatal("NewLoadAvgCollector returned nil")
	}
	
	if collector.logger == nil {
		t.Error("Logger should not be nil")
	}
	
	if collector.load1 == nil {
		t.Error("Load1 descriptor should not be nil")
	}
	
	if collector.load5 == nil {
		t.Error("Load5 descriptor should not be nil")
	}
	
	if collector.load15 == nil {
		t.Error("Load15 descriptor should not be nil")
	}
}

func TestLoadAvgCollectorImplementsCollector(t *testing.T) {
	collector := NewLoadAvgCollector()
	
	// Test that it implements the Collector interface
	var _ prometheus.Collector = collector
}

func TestLoadAvgCollectorMetricNames(t *testing.T) {
	collector := NewLoadAvgCollector()
	
	// Test that metric names contain expected substrings
	descriptors := map[string]*prometheus.Desc{
		"load1":  collector.load1,
		"load5":  collector.load5,
		"load15": collector.load15,
	}
	
	for expectedName, desc := range descriptors {
		metricName := desc.String()
		if !strings.Contains(metricName, "node_"+expectedName) {
			t.Errorf("Metric name should contain 'node_%s': %s", expectedName, metricName)
		}
	}
} 
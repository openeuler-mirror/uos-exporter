package metrics

import (
	"testing"
	"strings"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewStatCollector(t *testing.T) {
	collector := NewStatCollector()
	
	if collector == nil {
		t.Fatal("NewStatCollector returned nil")
	}
	
	if collector.logger == nil {
		t.Error("Logger should not be nil")
	}
	
	// Note: fs field is a struct, not a pointer, so we can't check for nil
	
	// Test that descriptors are initialized
	descriptors := []*prometheus.Desc{
		collector.intr,
		collector.ctxt,
		collector.forks,
		collector.btime,
		collector.procsRunning,
		collector.procsBlocked,
		collector.softIRQ,
	}
	
	for i, desc := range descriptors {
		if desc == nil {
			t.Errorf("Descriptor %d should not be nil", i)
		}
	}
}

func TestStatCollectorImplementsCollector(t *testing.T) {
	collector := NewStatCollector()
	if collector == nil {
		t.Skip("StatCollector creation failed, skipping interface test")
	}
	
	// Test that it implements the Collector interface
	var _ prometheus.Collector = collector
}

func TestStatCollectorMetricNames(t *testing.T) {
	collector := NewStatCollector()
	if collector == nil {
		t.Skip("StatCollector creation failed, skipping metric names test")
	}
	
	// Test that metric names contain expected substrings
	descriptors := map[string]*prometheus.Desc{
		"intr_total":              collector.intr,
		"context_switches_total":  collector.ctxt,
		"forks_total":             collector.forks,
		"boot_time_seconds":       collector.btime,
		"procs_running":           collector.procsRunning,
		"procs_blocked":           collector.procsBlocked,
		"softirqs_total":          collector.softIRQ,
	}
	
	for expectedName, desc := range descriptors {
		metricName := desc.String()
		if !strings.Contains(metricName, "node_"+expectedName) {
			t.Errorf("Metric name should contain 'node_%s': %s", expectedName, metricName)
		}
	}
} 
package metrics

import (
	"testing"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewVMStatCollector(t *testing.T) {
	collector := NewVMStatCollector()
	
	if collector == nil {
		t.Fatal("NewVMStatCollector returned nil")
	}
	
	if collector.logger == nil {
		t.Error("Logger should not be nil")
	}
	
	if collector.fieldPattern == nil {
		t.Error("Field pattern should not be nil")
	}
}

func TestVMStatCollectorImplementsCollector(t *testing.T) {
	collector := NewVMStatCollector()
	
	// Test that it implements the Collector interface
	var _ prometheus.Collector = collector
}
// Part 2 commit for node_system_exporter/internal/metrics/vmstat_test.go

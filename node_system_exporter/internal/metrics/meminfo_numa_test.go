package metrics

import (
	"testing"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewMeminfoNumaCollector(t *testing.T) {
	collector := NewMeminfoNumaCollector()
	
	if collector == nil {
		t.Fatal("NewMeminfoNumaCollector returned nil")
	}
	
	if collector.logger == nil {
		t.Error("Logger should not be nil")
	}
}

func TestMeminfoNumaCollectorImplementsCollector(t *testing.T) {
	collector := NewMeminfoNumaCollector()
	
	// Test that it implements the Collector interface
	var _ prometheus.Collector = collector
}





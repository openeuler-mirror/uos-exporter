package metrics

import (
	"testing"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewKSMDCollector(t *testing.T) {
	collector := NewKSMDCollector()
	
	if collector == nil {
		t.Fatal("NewKSMDCollector returned nil")
	}
	
	if collector.logger == nil {
		t.Error("Logger should not be nil")
	}
}

func TestKSMDCollectorImplementsCollector(t *testing.T) {
	collector := NewKSMDCollector()
	
	// Test that it implements the Collector interface
	var _ prometheus.Collector = collector
}





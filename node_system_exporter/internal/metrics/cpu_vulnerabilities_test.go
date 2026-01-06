package metrics

import (
	"testing"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewCPUVulnerabilitiesCollector(t *testing.T) {
	collector := NewCPUVulnerabilitiesCollector()
	
	if collector == nil {
		t.Fatal("NewCPUVulnerabilitiesCollector returned nil")
	}
	
	if collector.logger == nil {
		t.Error("Logger should not be nil")
	}
}

func TestCPUVulnerabilitiesCollectorImplementsCollector(t *testing.T) {
	collector := NewCPUVulnerabilitiesCollector()
	
	// Test that it implements the Collector interface
	var _ prometheus.Collector = collector
}





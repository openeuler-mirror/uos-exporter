package metrics

import (
	"testing"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewBuddyInfoCollector(t *testing.T) {
	collector := NewBuddyInfoCollector()
	
	if collector == nil {
		t.Fatal("NewBuddyInfoCollector returned nil")
	}
	
	if collector.logger == nil {
		t.Error("Logger should not be nil")
	}
}

func TestBuddyInfoCollectorImplementsCollector(t *testing.T) {
	collector := NewBuddyInfoCollector()
	
	// Test that it implements the Collector interface
	var _ prometheus.Collector = collector
} 
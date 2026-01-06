package metrics

import (
	"testing"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewMemoryCollector(t *testing.T) {
	collector := NewMemoryCollector()
	
	if collector == nil {
		t.Fatal("NewMemoryCollector returned nil")
	}
	
	if collector.logger == nil {
		t.Error("Logger should not be nil")
	}
	
	// Note: fs field is a struct, not a pointer, so we can't check for nil
}

func TestMemoryCollectorImplementsCollector(t *testing.T) {
	collector := NewMemoryCollector()
	if collector == nil {
		t.Skip("MemoryCollector creation failed, skipping interface test")
	}
	
	// Test that it implements the Collector interface
	var _ prometheus.Collector = collector
} 
package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewEdacCollector(t *testing.T) {
	collector := NewEdacCollector()
	if collector == nil {
		t.Fatal("NewEdacCollector returned nil")
	}
	if collector.baseMetrics == nil {
		t.Error("baseMetrics should not be nil")
	}
}

func TestEdacCollectorCollect(t *testing.T) {
	collector := NewEdacCollector()
	if collector == nil {
		t.Fatal("NewEdacCollector returned nil")
	}
	
	// Create a channel to collect metrics
	ch := make(chan prometheus.Metric, 100)
	
	// This should not panic even if EDAC info is not available
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Collect should not panic: %v", r)
			}
		}()
		collector.Collect(ch)
	}()
	
	close(ch)
	
	// Count collected metrics
	count := 0
	for range ch {
		count++
	}
	
	// We don't assert on the exact count because it depends on system capabilities
	// Just ensure it doesn't crash
	t.Logf("Collected %d EDAC metrics", count)
} 
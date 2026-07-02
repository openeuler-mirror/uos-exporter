package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewRaplCollector(t *testing.T) {
	collector := NewRaplCollector()
	if collector == nil {
		t.Fatal("NewRaplCollector returned nil")
	}
	if collector.baseMetrics == nil {
		t.Error("baseMetrics should not be nil")
	}
}

func TestRaplCollectorCollect(t *testing.T) {
	collector := NewRaplCollector()
	if collector == nil {
		t.Fatal("NewRaplCollector returned nil")
	}
	
	// Create a channel to collect metrics
	ch := make(chan prometheus.Metric, 100)
	
	// This should not panic even if RAPL info is not available
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
	t.Logf("Collected %d RAPL metrics", count)
}
// Part 2 commit for node_hardware_exporter/internal/metrics/rapl_test.go

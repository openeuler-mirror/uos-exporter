package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewWatchdogCollector(t *testing.T) {
	collector := NewWatchdogCollector()
	if collector == nil {
		t.Fatal("NewWatchdogCollector returned nil")
	}
	if collector.baseMetrics == nil {
		t.Error("baseMetrics should not be nil")
	}
}

func TestWatchdogCollectorCollect(t *testing.T) {
	collector := NewWatchdogCollector()
	if collector == nil {
		t.Fatal("NewWatchdogCollector returned nil")
	}
	
	// Create a channel to collect metrics
	ch := make(chan prometheus.Metric, 100)
	
	// This should not panic even if watchdog info is not available
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
	t.Logf("Collected %d watchdog metrics", count)
}
// Part 2 commit for node_hardware_exporter/internal/metrics/watchdog_test.go

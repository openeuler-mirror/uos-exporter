package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewNVMeCollector(t *testing.T) {
	collector := NewNVMeCollector()
	if collector == nil {
		t.Fatal("NewNVMeCollector returned nil")
	}
	if collector.baseMetrics == nil {
		t.Error("baseMetrics should not be nil")
	}
}

func TestNVMeCollectorCollect(t *testing.T) {
	collector := NewNVMeCollector()
	if collector == nil {
		t.Fatal("NewNVMeCollector returned nil")
	}
	
	// Create a channel to collect metrics
	ch := make(chan prometheus.Metric, 100)
	
	// This should not panic even if NVMe info is not available
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
	t.Logf("Collected %d NVMe metrics", count)
}
// Part 2 commit for node_hardware_exporter/internal/metrics/nvme_test.go

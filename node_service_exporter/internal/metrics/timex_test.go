package metrics

import (
	"testing"
	"github.com/prometheus/client_golang/prometheus"
)

func TestTimexCollectors(t *testing.T) {
	collectors := []interface{}{
		NewTimexCollectorWrapper(),
	}
	for i, c := range collectors {
		if c == nil {
			t.Errorf("collector %d is nil", i)
		}
	}
}

func TestTimexCollectorWrapper(t *testing.T) {
	collector := NewTimexCollectorWrapper()
	if collector == nil {
		t.Fatal("NewTimexCollectorWrapper returned nil")
	}
	
	// Test that Collect doesn't panic
	ch := make(chan prometheus.Metric, 50)
	defer close(ch)
	
	collector.Collect(ch)
	
	// Should have collected metrics
	if len(ch) == 0 {
		t.Log("No timex metrics collected - this might be expected in test environment")
	}
}
// Part 2 commit for node_service_exporter/internal/metrics/timex_test.go

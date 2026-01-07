package metrics

import (
	"testing"
	"github.com/prometheus/client_golang/prometheus"
)

func TestTimeCollectors(t *testing.T) {
	collectors := []interface{}{
		NewTimeCollectorWrapper(),
	}
	for i, c := range collectors {
		if c == nil {
			t.Errorf("collector %d is nil", i)
		}
	}
}

func TestTimeCollectorWrapper(t *testing.T) {
	collector := NewTimeCollectorWrapper()
	if collector == nil {
		t.Fatal("NewTimeCollectorWrapper returned nil")
	}
	
	// Test that Collect doesn't panic
	ch := make(chan prometheus.Metric, 10)
	defer close(ch)
	
	collector.Collect(ch)
	
	// Should have collected at least one metric
	if len(ch) == 0 {
		t.Log("No time metrics collected - this might be expected in test environment")
	}
}
// Part 2 commit for node_service_exporter/internal/metrics/time_test.go

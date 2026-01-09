package metrics

import (
	"testing"
	"github.com/prometheus/client_golang/prometheus"
)

func TestDRBDCollectors(t *testing.T) {
	collectors := []interface{}{
		NewDRBDCollectorWrapper(),
	}
	for i, c := range collectors {
		if c == nil {
			t.Errorf("collector %d is nil", i)
		}
	}
}

func TestDRBDCollectorWrapper(t *testing.T) {
	collector := NewDRBDCollectorWrapper()
	if collector == nil {
		t.Fatal("NewDRBDCollectorWrapper returned nil")
	}
	
	// Test that Collect doesn't panic
	ch := make(chan prometheus.Metric, 50)
	defer close(ch)
	
	collector.Collect(ch)
	
	// DRBD might not be available in test environment
	t.Log("DRBD test completed - metrics availability depends on system configuration")
}

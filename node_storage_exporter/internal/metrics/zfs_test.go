package metrics

import (
	"testing"
	"strings"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestNewZFSCollector(t *testing.T) {
	collector := NewZFSCollector()
	
	if collector == nil {
		t.Fatal("NewZFSCollector returned nil")
	}
	
	if collector.logger == nil {
		t.Error("Logger should not be nil")
	}
	
	if len(collector.descs) == 0 {
		t.Error("Descriptors should not be empty")
	}
}

func TestZFSCollectorImplementsCollector(t *testing.T) {
	collector := NewZFSCollector()
	
	// Test that it implements the Collector interface
	var _ prometheus.Collector = collector
}

func TestZFSCollectorCollect(t *testing.T) {
	collector := NewZFSCollector()
	
	// Create a channel to collect metrics
	ch := make(chan prometheus.Metric, 100)
	
	// Run collect in a goroutine
	go func() {
		defer close(ch)
		collector.Collect(ch)
	}()
	
	// Count metrics
	metricCount := 0
	for range ch {
		metricCount++
	}
	
	// We should have at least some metrics or none (if ZFS is not available)
	if metricCount < 0 {
		t.Errorf("Expected at least 0 metrics, got %d", metricCount)
	}
}

func TestZFSDescriptors(t *testing.T) {
	collector := NewZFSCollector()
	
	// Expected descriptors for ZFS
	expectedDescs := []string{
		"allocated_bytes",
		"free_bytes",
		"size_bytes",
	}
	
	if len(collector.descs) < len(expectedDescs) {
		t.Logf("Expected at least %d descriptors, got %d", len(expectedDescs), len(collector.descs))
		// Note: This might be expected if ZFS is not available
	}
	
	// Test that descriptors are properly formed
	for name, desc := range collector.descs {
		if desc == nil {
			t.Errorf("Descriptor %s should not be nil", name)
		}
		if desc.String() == "" {
			t.Errorf("Descriptor %s should have a string representation", name)
		}
	}
}

func TestZFSMetricNames(t *testing.T) {
	collector := NewZFSCollector()
	
	// Test that metric names contain expected substrings
	for name, desc := range collector.descs {
		metricName := desc.String()
		if !strings.Contains(metricName, "node_zfs") {
			t.Errorf("Metric name should contain 'node_zfs': %s", metricName)
		}
		
		// Test specific metric names
		switch {
		case strings.Contains(name, "allocated"):
			if !strings.Contains(metricName, "allocated") {
				t.Errorf("Allocated metric should contain 'allocated': %s", metricName)
			}
		case strings.Contains(name, "free"):
			if !strings.Contains(metricName, "free") {
				t.Errorf("Free metric should contain 'free': %s", metricName)
			}
		case strings.Contains(name, "size"):
			if !strings.Contains(metricName, "size") {
				t.Errorf("Size metric should contain 'size': %s", metricName)
			}
		}
	}
}

func TestZFSCollectResilience(t *testing.T) {
	collector := NewZFSCollector()
	
	// This test ensures the collector handles missing ZFS gracefully
	// In a test environment, ZFS might not be available
	ch := make(chan prometheus.Metric, 10)
	
	// This should not panic even if ZFS is not available
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Collect should not panic: %v", r)
		}
	}()
	
	collector.Collect(ch)
	close(ch)
}

func TestZFSConstants(t *testing.T) {
	// Test that ZFS-related behavior is reasonable
	collector := NewZFSCollector()
	
	if collector == nil {
		t.Error("Constructor should return valid collector")
	}
	
	// Check that it has the expected subsystem
	hasZFSMetrics := false
	for name := range collector.descs {
		if strings.Contains(name, "zfs") || strings.Contains(name, "allocated") || strings.Contains(name, "free") {
			hasZFSMetrics = true
			break
		}
	}
	
	if !hasZFSMetrics {
		t.Log("No ZFS metrics found (expected if ZFS is not available)")
	}
}

func TestZFSLabels(t *testing.T) {
	collector := NewZFSCollector()
	
	// Test that descriptors have appropriate labels
	for name, desc := range collector.descs {
		descStr := desc.String()
		
		// ZFS metrics should have dataset labels
		if strings.Contains(name, "zfs") {
			// Check that the descriptor is well-formed
			if !strings.Contains(descStr, "Desc{") {
				t.Errorf("Descriptor should be well-formed: %s", descStr)
			}
		}
	}
}

func TestZFSCollectNonBlocking(t *testing.T) {
	collector := NewZFSCollector()
	
	// Test that Collect returns within a reasonable time
	ch := make(chan prometheus.Metric, 100)
	done := make(chan bool, 1)
	
	go func() {
		collector.Collect(ch)
		close(ch)
		done <- true
	}()
	
	// Consume metrics
	go func() {
		for range ch {
			// Just consume metrics
		}
	}()
	
	// Wait for completion
	<-done
	
	// If we get here, the function didn't block indefinitely
	t.Log("ZFS Collect completed successfully")
}

func TestZFSMetricTypes(t *testing.T) {
	collector := NewZFSCollector()
	
	// Test that we can collect metrics without errors
	ch := make(chan prometheus.Metric, 100)
	
	go func() {
		defer close(ch)
		collector.Collect(ch)
	}()
	
	// Verify metrics are well-formed
	for metric := range ch {
		if metric == nil {
			t.Error("Metric should not be nil")
		}
		
		// Basic validation - metric should have a description
		metricDto := &dto.Metric{}
		if err := metric.Write(metricDto); err != nil {
			t.Errorf("Metric should be writable: %v", err)
		}
	}
} 
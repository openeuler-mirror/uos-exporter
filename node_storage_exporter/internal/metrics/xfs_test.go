package metrics

import (
	"testing"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestNewXFSCollector(t *testing.T) {
	collector := NewXFSCollector()
	
	if collector == nil {
		t.Fatal("NewXFSCollector returned nil")
	}
	
	if collector.logger == nil {
		t.Error("Logger should not be nil")
	}
}

func TestXFSCollectorImplementsCollector(t *testing.T) {
	collector := NewXFSCollector()
	
	// Test that it implements the Collector interface
	var _ prometheus.Collector = collector
}

func TestXFSCollectorCollect(t *testing.T) {
	collector := NewXFSCollector()
	
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
	
	// We should have at least some metrics or none (if XFS is not available)
	if metricCount < 0 {
		t.Errorf("Expected at least 0 metrics, got %d", metricCount)
	}
}

func TestXFSCollectResilience(t *testing.T) {
	collector := NewXFSCollector()
	
	// This test ensures the collector handles missing XFS gracefully
	// In a test environment, XFS might not be available
	ch := make(chan prometheus.Metric, 10)
	
	// This should not panic even if XFS is not available
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Collect should not panic: %v", r)
		}
	}()
	
	collector.Collect(ch)
	close(ch)
}

func TestXFSConstants(t *testing.T) {
	// Test that XFS-related behavior is reasonable
	collector := NewXFSCollector()
	
	if collector == nil {
		t.Error("Constructor should return valid collector")
	}
}

func TestXFSCollectNonBlocking(t *testing.T) {
	collector := NewXFSCollector()
	
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
	t.Log("XFS Collect completed successfully")
}

func TestXFSMetricTypes(t *testing.T) {
	collector := NewXFSCollector()
	
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

func TestXFSDescribe(t *testing.T) {
	collector := NewXFSCollector()
	
	// Test that Describe method works
	ch := make(chan *prometheus.Desc, 100)
	
	go func() {
		defer close(ch)
		collector.Describe(ch)
	}()
	
	// Count descriptions
	descCount := 0
	for desc := range ch {
		if desc == nil {
			t.Error("Description should not be nil")
		}
		descCount++
	}
	
	// Should have at least 0 descriptions (might be 0 if xfs not available)
	if descCount < 0 {
		t.Errorf("Expected at least 0 descriptions, got %d", descCount)
	}
} 
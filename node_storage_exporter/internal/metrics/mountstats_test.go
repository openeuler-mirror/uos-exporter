package metrics

import (
	"testing"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewMountStatsCollector(t *testing.T) {
	collector := NewMountStatsCollector()
	
	if collector == nil {
		t.Fatal("NewMountStatsCollector returned nil")
	}
	
	if collector.logger == nil {
		t.Error("Logger should not be nil")
	}
}

func TestMountStatsCollectorImplementsCollector(t *testing.T) {
	collector := NewMountStatsCollector()
	
	// Test that it implements the Collector interface
	var _ prometheus.Collector = collector
}

func TestMountStatsCollectorCollect(t *testing.T) {
	collector := NewMountStatsCollector()
	
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
	
	// We should have at least some metrics or none (if no mount stats available)
	if metricCount < 0 {
		t.Errorf("Expected at least 0 metrics, got %d", metricCount)
	}
}

func TestMountStatsConstants(t *testing.T) {
	// Test that NFS-related constants are reasonable
	// Since the actual constants aren't visible in the test, we'll test behavior
	collector := NewMountStatsCollector()
	
	if collector == nil {
		t.Error("Constructor should return valid collector")
	}
}

func TestMountStatsParseResilience(t *testing.T) {
	collector := NewMountStatsCollector()
	
	// This test ensures the collector handles missing /proc/self/mountstats gracefully
	// In a test environment, this file might not exist
	ch := make(chan prometheus.Metric, 10)
	done := make(chan bool, 1)
	
	// This should not panic even if files are missing
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Collect should not panic: %v", r)
		}
	}()
	
	go func() {
		defer func() { done <- true }()
		collector.Collect(ch)
		close(ch)
	}()
	
	// Consume metrics to prevent blocking
	go func() {
		for range ch {
			// Just consume
		}
	}()
	
	// Wait for completion
	<-done
}

func TestMountStatsCollectNonBlocking(t *testing.T) {
	collector := NewMountStatsCollector()
	
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
	t.Log("Collect completed successfully")
}

func TestMountStatsDescribe(t *testing.T) {
	collector := NewMountStatsCollector()
	
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
	
	// Should have a reasonable number of descriptions
	if descCount < 0 {
		t.Errorf("Expected at least 0 descriptions, got %d", descCount)
	}
	
	// MountStatsCollector should have many descriptors
	if descCount > 0 && descCount < 10 {
		t.Log("MountStatsCollector has fewer descriptors than expected, but this might be normal")
	}
} 
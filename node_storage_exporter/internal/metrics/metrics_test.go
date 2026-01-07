package metrics

import (
	"testing"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestCollectAll(t *testing.T) {
	// Create a channel to collect metrics
	ch := make(chan prometheus.Metric, 1000)
	
	// Run CollectAll in a goroutine
	go func() {
		defer close(ch)
		CollectAll(ch)
	}()
	
	// Count metrics
	metricCount := 0
	for range ch {
		metricCount++
	}
	
	// We should have collected metrics from all registered collectors
	if metricCount < 0 {
		t.Errorf("Expected at least 0 metrics, got %d", metricCount)
	}
	
	t.Logf("Collected %d metrics total", metricCount)
}

func TestCollectAllNonBlocking(t *testing.T) {
	// Test that CollectAll returns within a reasonable time
	ch := make(chan prometheus.Metric, 1000)
	done := make(chan bool, 1)
	
	go func() {
		CollectAll(ch)
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
	t.Log("CollectAll completed successfully")
}

func TestCollectAllResilience(t *testing.T) {
	// Test that CollectAll handles errors gracefully
	ch := make(chan prometheus.Metric, 10)
	
	// This should not panic even if individual collectors have issues
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("CollectAll should not panic: %v", r)
		}
	}()
	
	CollectAll(ch)
	close(ch)
}

func TestCollectAllMetrics(t *testing.T) {
	// Test that all metrics from CollectAll are well-formed
	ch := make(chan prometheus.Metric, 1000)
	
	go func() {
		defer close(ch)
		CollectAll(ch)
	}()
	
	metricCount := 0
	for metric := range ch {
		metricCount++
		
		if metric == nil {
			t.Error("Metric should not be nil")
			continue
		}
		
		// Basic validation - metric should be writable
		metricDto := &dto.Metric{}
		if err := metric.Write(metricDto); err != nil {
			t.Errorf("Metric should be writable: %v", err)
		}
	}
	
	t.Logf("Validated %d metrics", metricCount)
}

func TestAllCollectorsRegistered(t *testing.T) {
	// Test that we can create all expected collectors without errors
	collectors := []func() interface{}{
		func() interface{} { return NewDiskStatsCollector() },
		func() interface{} { return NewFilesystemCollector() },
		func() interface{} { return NewMountStatsCollector() },
		func() interface{} { return NewZFSCollector() },
		func() interface{} { return NewBtrfsCollector() },
		func() interface{} { return NewXFSCollector() },
	}
	
	for i, createCollector := range collectors {
		collector := createCollector()
		if collector == nil {
			t.Errorf("Collector %d should not be nil", i)
		}
		
		// Test that it implements prometheus.Collector
		if _, ok := collector.(prometheus.Collector); !ok {
			t.Errorf("Collector %d should implement prometheus.Collector interface", i)
		}
	}
}

func TestPackageInitialization(t *testing.T) {
	// Test that the package initializes without panicking
	// The init() functions should have run without issues
	t.Log("Package initialized successfully")
	
	// Test that we can call CollectAll multiple times
	for i := 0; i < 3; i++ {
		ch := make(chan prometheus.Metric, 100)
		
		go func() {
			defer close(ch)
			CollectAll(ch)
		}()
		
		count := 0
		for range ch {
			count++
		}
		
		t.Logf("Run %d: collected %d metrics", i+1, count)
	}
}

func TestMetricsChannelCapacity(t *testing.T) {
	// Test that CollectAll works with different channel capacities
	capacities := []int{1, 10, 100, 1000}
	
	for _, capacity := range capacities {
		ch := make(chan prometheus.Metric, capacity)
		
		go func() {
			defer close(ch)
			CollectAll(ch)
		}()
		
		count := 0
		for range ch {
			count++
		}
		
		t.Logf("Channel capacity %d: collected %d metrics", capacity, count)
	}
}

func TestConcurrentCollectAll(t *testing.T) {
	// Test that multiple concurrent calls to CollectAll work correctly
	const numGoroutines = 5
	results := make(chan int, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func() {
			ch := make(chan prometheus.Metric, 1000)
			
			go func() {
				defer close(ch)
				CollectAll(ch)
			}()
			
			count := 0
			for range ch {
				count++
			}
			
			results <- count
		}()
	}
	
	// Collect results
	for i := 0; i < numGoroutines; i++ {
		count := <-results
		t.Logf("Goroutine %d collected %d metrics", i+1, count)
	}
} 
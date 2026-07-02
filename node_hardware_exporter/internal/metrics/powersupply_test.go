package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewPowerSupplyCollector(t *testing.T) {
	collector := NewPowerSupplyCollector()
	if collector == nil {
		t.Fatal("NewPowerSupplyCollector returned nil")
	}
	if collector.baseMetrics == nil {
		t.Error("baseMetrics should not be nil")
	}
	if collector.subsystem != "power_supply" {
		t.Errorf("expected subsystem to be 'power_supply', got %s", collector.subsystem)
	}
	if collector.ignoredPattern == nil {
		t.Error("ignoredPattern should not be nil")
	}
}

func TestPowerSupplyCollectorCollect(t *testing.T) {
	collector := NewPowerSupplyCollector()
	if collector == nil {
		t.Fatal("NewPowerSupplyCollector returned nil")
	}
	
	// Create a channel to collect metrics
	ch := make(chan prometheus.Metric, 100)
	
	// This should not panic even if power supply info is not available
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
	t.Logf("Collected %d power supply metrics", count)
}

func TestPowerSupplyPushMetric(t *testing.T) {
	collector := NewPowerSupplyCollector()
	if collector == nil {
		t.Fatal("NewPowerSupplyCollector returned nil")
	}
	
	// Create a channel to collect metrics
	ch := make(chan prometheus.Metric, 10)
	
	// Test pushing a metric
	collector.pushPowerSupplyMetric(ch, "capacity", 85.0, "BAT0", prometheus.GaugeValue)
	
	close(ch)
	
	// Should have exactly one metric
	count := 0
	for metric := range ch {
		if metric == nil {
			t.Error("metric should not be nil")
		}
		count++
	}
	
	if count != 1 {
		t.Errorf("expected 1 metric, got %d", count)
	}
}

func TestPowerSupplyIgnoredPattern(t *testing.T) {
	collector := NewPowerSupplyCollector()
	if collector == nil {
		t.Fatal("NewPowerSupplyCollector returned nil")
	}
	
	// Test that the default pattern ignores nothing (empty pattern)
	testStrings := []string{"BAT0", "ADP1", "HID-compliant-battery"}
	
	for _, str := range testStrings {
		if collector.ignoredPattern.MatchString(str) {
			t.Errorf("default pattern should not match %q (should ignore nothing)", str)
		}
	}
	
	// The pattern should only match empty strings
	if !collector.ignoredPattern.MatchString("") {
		t.Error("pattern should match empty string")
	}
}

func TestPowerSupplyGetPowerSupplyClassInfo(t *testing.T) {
	collector := NewPowerSupplyCollector()
	if collector == nil {
		t.Fatal("NewPowerSupplyCollector returned nil")
	}
	
	// This should not panic even if power supply class info is not available
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("getPowerSupplyClassInfo should not panic: %v", r)
			}
		}()
		
		_, err := collector.getPowerSupplyClassInfo()
		// Error is expected if /sys is not available or no power supplies exist
		if err != nil {
			t.Logf("getPowerSupplyClassInfo returned error (expected in test environment): %v", err)
		}
	}()
}
// Part 2 commit for node_hardware_exporter/internal/metrics/powersupply_test.go

package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewThermalZoneCollector(t *testing.T) {
	collector := NewThermalZoneCollector()
	if collector == nil {
		t.Fatal("NewThermalZoneCollector returned nil")
	}
	if collector.baseMetrics == nil {
		t.Error("baseMetrics should not be nil")
	}
}

func TestThermalZoneConstants(t *testing.T) {
	expectedCoolingDevice := "cooling_device"
	if coolingDevice != expectedCoolingDevice {
		t.Errorf("expected coolingDevice to be %q, got %q", expectedCoolingDevice, coolingDevice)
	}
	
	expectedThermalZone := "thermal_zone"
	if thermalZone != expectedThermalZone {
		t.Errorf("expected thermalZone to be %q, got %q", expectedThermalZone, thermalZone)
	}
}

func TestThermalZoneCollectorCollect(t *testing.T) {
	collector := NewThermalZoneCollector()
	if collector == nil {
		t.Fatal("NewThermalZoneCollector returned nil")
	}
	
	// Create a channel to collect metrics
	ch := make(chan prometheus.Metric, 100)
	
	// This should not panic even if thermal zones are not available
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
	t.Logf("Collected %d thermal zone metrics", count)
}

func TestThermalZoneCollectorFields(t *testing.T) {
	collector := NewThermalZoneCollector()
	if collector == nil {
		t.Fatal("NewThermalZoneCollector returned nil")
	}
	
	// Just test that the collector was created properly
	// Don't actually collect metrics in this test to avoid blocking
	if collector.baseMetrics == nil {
		t.Error("baseMetrics should not be nil")
	}
}
// Part 2 commit for node_hardware_exporter/internal/metrics/thermal_zone_test.go

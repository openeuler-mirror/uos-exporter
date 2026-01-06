package metrics

import (
	"testing"
	"strings"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewCPUCollector(t *testing.T) {
	collector := NewCPUCollector()
	
	if collector == nil {
		t.Fatal("NewCPUCollector returned nil")
	}
	
	if collector.logger == nil {
		t.Error("Logger should not be nil")
	}
	
	if collector.cpu == nil {
		t.Error("CPU descriptor should not be nil")
	}
	
	if collector.cpuInfo == nil {
		t.Error("CPU info descriptor should not be nil")
	}
	
	if collector.cpuFrequencyHz == nil {
		t.Error("CPU frequency descriptor should not be nil")
	}
}

func TestCPUCollectorImplementsCollector(t *testing.T) {
	collector := NewCPUCollector()
	if collector == nil {
		t.Skip("CPUCollector creation failed, skipping interface test")
	}
	
	// Test that it implements the Collector interface
	var _ prometheus.Collector = collector
}

func TestCPUCollectorDescriptors(t *testing.T) {
	collector := NewCPUCollector()
	if collector == nil {
		t.Skip("CPUCollector creation failed, skipping descriptor test")
	}
	
	// Test that descriptors are properly formed
	descriptors := []*prometheus.Desc{
		collector.cpu,
		collector.cpuInfo,
		collector.cpuFrequencyHz,
		collector.cpuFlagsInfo,
		collector.cpuBugsInfo,
		collector.cpuGuest,
		collector.cpuCoreThrottle,
		collector.cpuPackageThrottle,
		collector.cpuIsolated,
		collector.cpuOnline,
	}
	
	for i, desc := range descriptors {
		if desc == nil {
			t.Errorf("Descriptor %d should not be nil", i)
		} else {
			descStr := desc.String()
			if descStr == "" {
				t.Errorf("Descriptor %d should have a string representation", i)
			}
		}
	}
}

func TestCPUCollectorMetricNames(t *testing.T) {
	collector := NewCPUCollector()
	if collector == nil {
		t.Skip("CPUCollector creation failed, skipping metric names test")
	}
	
	// Test that metric names contain expected substrings
	descriptors := map[string]*prometheus.Desc{
		"cpu_seconds_total":      collector.cpu,
		"cpu_info":               collector.cpuInfo,
		"cpu_frequency_hertz":    collector.cpuFrequencyHz,
		"cpu_flag_info":          collector.cpuFlagsInfo,
		"cpu_bug_info":           collector.cpuBugsInfo,
		"cpu_guest_seconds":      collector.cpuGuest,
		"cpu_core_throttles":     collector.cpuCoreThrottle,
		"cpu_package_throttles":  collector.cpuPackageThrottle,
		"cpu_isolated":           collector.cpuIsolated,
		"cpu_online":             collector.cpuOnline,
	}
	
	for expectedName, desc := range descriptors {
		metricName := desc.String()
		if !strings.Contains(metricName, "node_cpu") {
			t.Errorf("Metric name should contain 'node_cpu': %s", metricName)
		}
		
		// Check for specific expected substrings
		switch expectedName {
		case "cpu_seconds_total":
			if !strings.Contains(metricName, "seconds_total") {
				t.Errorf("CPU metric should contain 'seconds_total': %s", metricName)
			}
		case "cpu_frequency_hertz":
			if !strings.Contains(metricName, "frequency_hertz") {
				t.Errorf("Frequency metric should contain 'frequency_hertz': %s", metricName)
			}
		case "cpu_info":
			if !strings.Contains(metricName, "info") {
				t.Errorf("Info metric should contain 'info': %s", metricName)
			}
		}
	}
}

func TestCPUCollectorConstants(t *testing.T) {
	// Test constants are reasonable
	if cpuCollectorSubsystem != "cpu" {
		t.Errorf("Expected cpuCollectorSubsystem to be 'cpu', got %s", cpuCollectorSubsystem)
	}
	
	if jumpBackSeconds != 3.0 {
		t.Errorf("Expected jumpBackSeconds to be 3.0, got %f", jumpBackSeconds)
	}
} 
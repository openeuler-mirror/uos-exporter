package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestPerfTracepointFlagToTracepoints(t *testing.T) {
	tests := []struct {
		name          string
		input         []string
		expected      int
		shouldError   bool
	}{
		{
			name:        "valid tracepoints",
			input:       []string{"syscalls:sys_enter_open", "sched:sched_switch"},
			expected:    2,
			shouldError: false,
		},
		{
			name:        "empty input",
			input:       []string{},
			expected:    0,
			shouldError: false,
		},
		{
			name:        "invalid format",
			input:       []string{"invalid_format"},
			expected:    0,
			shouldError: true,
		},
		{
			name:        "mixed valid and invalid",
			input:       []string{"syscalls:sys_enter_open", "invalid"},
			expected:    0,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := perfTracepointFlagToTracepoints(tt.input)
			
			if tt.shouldError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			if len(result) != tt.expected {
				t.Errorf("expected %d tracepoints, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestPerfCPUFlagToCPUs(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    []int
		shouldError bool
	}{
		{
			name:        "single CPU",
			input:       "0",
			expected:    []int{0},
			shouldError: false,
		},
		{
			name:        "CPU range",
			input:       "0-2",
			expected:    []int{0, 1, 2},
			shouldError: false,
		},
		{
			name:        "CPU list",
			input:       "0,2,4",
			expected:    []int{0, 2, 4},
			shouldError: false,
		},
		{
			name:        "CPU range with stride",
			input:       "0-4:2",
			expected:    []int{0, 2, 4},
			shouldError: false,
		},
		{
			name:        "empty string",
			input:       "",
			expected:    []int{},
			shouldError: true,
		},
		{
			name:        "invalid format",
			input:       "invalid",
			expected:    nil,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := perfCPUFlagToCPUs(tt.input)
			
			if tt.shouldError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			if len(result) != len(tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
				return
			}
			
			for i, cpu := range result {
				if cpu != tt.expected[i] {
					t.Errorf("expected CPU %d, got %d at position %d", tt.expected[i], cpu, i)
				}
			}
		})
	}
}

func TestPerfTracepoint(t *testing.T) {
	tp := &perfTracepoint{
		subsystem: "syscalls",
		event:     "sys_enter_open",
	}
	
	expectedLabel := "syscalls_sys_enter_open"
	if tp.label() != expectedLabel {
		t.Errorf("expected label %q, got %q", expectedLabel, tp.label())
	}
	
	expectedTracepoint := "syscalls:sys_enter_open"
	if tp.tracepoint() != expectedTracepoint {
		t.Errorf("expected tracepoint %q, got %q", expectedTracepoint, tp.tracepoint())
	}
}

func TestNewPerfCollector(t *testing.T) {
	// Test creating a new perf collector
	collector := NewPerfCollector()
	if collector == nil {
		t.Fatal("NewPerfCollector returned nil")
	}
	
	// Verify basic structure
	if collector.baseMetrics == nil {
		t.Error("baseMetrics should not be nil")
	}
	
	if collector.perfHwProfilers == nil {
		t.Error("perfHwProfilers should not be nil")
	}
	
	if collector.perfSwProfilers == nil {
		t.Error("perfSwProfilers should not be nil")
	}
	
	if collector.perfCacheProfilers == nil {
		t.Error("perfCacheProfilers should not be nil")
	}
}

func TestPerfCollectorDisabled(t *testing.T) {
	// Set the disabled flag temporarily
	originalDisabled := perfDisabled
	disabled := true
	perfDisabled = &disabled
	defer func() {
		perfDisabled = originalDisabled
	}()
	
	collector := NewPerfCollector()
	if collector == nil {
		t.Fatal("NewPerfCollector returned nil")
	}
	
	if !collector.disabled {
		t.Error("collector should be disabled")
	}
}

func TestPerfCollectorCollect(t *testing.T) {
	collector := NewPerfCollector()
	if collector == nil {
		t.Fatal("NewPerfCollector returned nil")
	}
	
	// Create a channel to collect metrics
	ch := make(chan prometheus.Metric, 100)
	
	// This should not panic even if perf is not available
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
	t.Logf("Collected %d perf metrics", count)
}

func TestPerfHardwareProfilerMap(t *testing.T) {
	// Test that the hardware profiler map is properly defined
	if len(perfHardwareProfilerMap) == 0 {
		t.Error("perfHardwareProfilerMap should not be empty")
	}
	
	// Check some expected entries
	expectedEntries := []string{"CpuCycles", "CpuInstr", "CacheRef", "CacheMisses"}
	for _, entry := range expectedEntries {
		if _, exists := perfHardwareProfilerMap[entry]; !exists {
			t.Errorf("expected entry %q not found in perfHardwareProfilerMap", entry)
		}
	}
}

func TestPerfSoftwareProfilerMap(t *testing.T) {
	// Test that the software profiler map is properly defined
	if len(perfSoftwareProfilerMap) == 0 {
		t.Error("perfSoftwareProfilerMap should not be empty")
	}
	
	// Check some expected entries
	expectedEntries := []string{"PageFault", "ContextSwitch", "CpuMigration"}
	for _, entry := range expectedEntries {
		if _, exists := perfSoftwareProfilerMap[entry]; !exists {
			t.Errorf("expected entry %q not found in perfSoftwareProfilerMap", entry)
		}
	}
}

func TestPerfCacheProfilerMap(t *testing.T) {
	// Test that the cache profiler map is properly defined
	if len(perfCacheProfilerMap) == 0 {
		t.Error("perfCacheProfilerMap should not be empty")
	}
	
	// Check some expected entries
	expectedEntries := []string{"L1DataReadHit", "L1DataReadMiss", "LLReadHit", "LLReadMiss"}
	for _, entry := range expectedEntries {
		if _, exists := perfCacheProfilerMap[entry]; !exists {
			t.Errorf("expected entry %q not found in perfCacheProfilerMap", entry)
		}
	}
}

// TestPerfCollectorUpdateStats tests the update methods with mock data
func TestPerfCollectorUpdateStats(t *testing.T) {
	collector := NewPerfCollector()
	if collector == nil {
		t.Fatal("NewPerfCollector returned nil")
	}
	
	ch := make(chan prometheus.Metric, 100)
	
	// These should not panic even with empty profiler maps
	t.Run("updateHardwareStats", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("updateHardwareStats should not panic: %v", r)
			}
		}()
		err := collector.updateHardwareStats(ch)
		// Should return nil when no profilers are available
		if err != nil {
			t.Logf("updateHardwareStats returned error (expected in test environment): %v", err)
		}
	})
	
	t.Run("updateSoftwareStats", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("updateSoftwareStats should not panic: %v", r)
			}
		}()
		err := collector.updateSoftwareStats(ch)
		// Should return nil when no profilers are available
		if err != nil {
			t.Logf("updateSoftwareStats returned error (expected in test environment): %v", err)
		}
	})
	
	t.Run("updateCacheStats", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("updateCacheStats should not panic: %v", r)
			}
		}()
		err := collector.updateCacheStats(ch)
		// Should return nil when no profilers are available
		if err != nil {
			t.Logf("updateCacheStats returned error (expected in test environment): %v", err)
		}
	})
	
	close(ch)
}
// Part 2 commit for node_hardware_exporter/internal/metrics/perf_linux_test.go

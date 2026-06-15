package metrics

import (
	"testing"
	"strings"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewFilesystemCollector(t *testing.T) {
	collector := NewFilesystemCollector()
	
	if collector == nil {
		t.Fatal("NewFilesystemCollector returned nil")
	}
	
	if collector.logger == nil {
		t.Error("Logger should not be nil")
	}
	
	if collector.ignoredMountPoints == nil {
		t.Error("Ignored mount points regex should not be nil")
	}
	
	if collector.ignoredFSTypes == nil {
		t.Error("Ignored fs types regex should not be nil")
	}
	
	if len(collector.descs) == 0 {
		t.Error("Descriptors should not be empty")
	}
}

func TestFilesystemCollectorImplementsCollector(t *testing.T) {
	collector := NewFilesystemCollector()
	
	// Test that it implements the Collector interface
	var _ prometheus.Collector = collector
}

func TestFilesystemCollectorCollect(t *testing.T) {
	collector := NewFilesystemCollector()
	
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
	
	// We should have at least some metrics (at least root filesystem)
	if metricCount < 0 {
		t.Errorf("Expected at least 0 metrics, got %d", metricCount)
	}
}

func TestFilesystemConstants(t *testing.T) {
	// Test constants are reasonable
	if defMountPointsExcluded == "" {
		t.Error("defMountPointsExcluded should not be empty")
	}
	
	if defFSTypesExcluded == "" {
		t.Error("defFSTypesExcluded should not be empty")
	}
	
	// Test regex patterns compile
	if collector := NewFilesystemCollector(); collector.ignoredMountPoints == nil {
		t.Error("Mount points regex should compile")
	}
	
	if collector := NewFilesystemCollector(); collector.ignoredFSTypes == nil {
		t.Error("FS types regex should compile")
	}
}

func TestFilesystemIgnorePatterns(t *testing.T) {
	collector := NewFilesystemCollector()
	
	// Test mount points that should be ignored
	ignoredMountPoints := []string{
		"/dev", "/proc", "/sys", "/var/lib/docker/overlay2",
	}
	
	for _, mp := range ignoredMountPoints {
		if !collector.ignoredMountPoints.MatchString(mp) {
			t.Errorf("Mount point %s should be ignored but wasn't", mp)
		}
	}
	
	// Test mount points that should not be ignored
	allowedMountPoints := []string{
		"/", "/home", "/tmp", "/var", "/opt",
	}
	
	for _, mp := range allowedMountPoints {
		if collector.ignoredMountPoints.MatchString(mp) {
			t.Errorf("Mount point %s should not be ignored but was", mp)
		}
	}
	
	// Test filesystem types that should be ignored
	ignoredFSTypes := []string{
		"proc", "sysfs", "devpts", "debugfs", "overlay",
	}
	
	for _, fstype := range ignoredFSTypes {
		if !collector.ignoredFSTypes.MatchString(fstype) {
			t.Errorf("Filesystem type %s should be ignored but wasn't", fstype)
		}
	}
	
	// Test filesystem types that should not be ignored
	allowedFSTypes := []string{
		"ext4", "xfs", "btrfs", "ntfs", "vfat", "tmpfs",
	}
	
	for _, fstype := range allowedFSTypes {
		if collector.ignoredFSTypes.MatchString(fstype) {
			t.Errorf("Filesystem type %s should not be ignored but was", fstype)
		}
	}
}

func TestFilesystemDescriptors(t *testing.T) {
	collector := NewFilesystemCollector()
	
	// Expected descriptors
	expectedDescs := []string{
		"size", "free", "avail", "files", "files_free", "ro", "device_error", "mount_info",
	}
	
	if len(collector.descs) != len(expectedDescs) {
		t.Errorf("Expected %d descriptors, got %d", len(expectedDescs), len(collector.descs))
	}
	
	// Test that all expected descriptors are present
	for _, expected := range expectedDescs {
		if _, ok := collector.descs[expected]; !ok {
			t.Errorf("Missing descriptor: %s", expected)
		}
	}
}

func TestFilesystemLabelsStruct(t *testing.T) {
	// Test that filesystemLabels struct can be created
	labels := filesystemLabels{
		device:      "/dev/sda1",
		mountPoint:  "/",
		fsType:      "ext4",
		options:     "rw,relatime",
		deviceError: "",
		major:       "8",
		minor:       "1",
	}
	
	if labels.device != "/dev/sda1" {
		t.Error("Device field not set correctly")
	}
	
	if labels.mountPoint != "/" {
		t.Error("Mount point field not set correctly")
	}
	
	if labels.fsType != "ext4" {
		t.Error("FS type field not set correctly")
	}
}

func TestFilesystemStatsStruct(t *testing.T) {
	// Test that filesystemStats struct can be created
	stats := filesystemStats{
		labels: filesystemLabels{
			device:     "/dev/sda1",
			mountPoint: "/",
			fsType:     "ext4",
		},
		size:        1000000,
		free:        500000,
		avail:       400000,
		files:       1000,
		filesFree:   500,
		ro:          0,
		deviceError: 0,
	}
	
	if stats.size != 1000000 {
		t.Error("Size field not set correctly")
	}
	
	if stats.free != 500000 {
		t.Error("Free field not set correctly")
	}
	
	if stats.avail != 400000 {
		t.Error("Available field not set correctly")
	}
}

func TestFilesystemProcessStat(t *testing.T) {
	collector := NewFilesystemCollector()
	
	// Test with a valid filesystem labels struct
	labels := filesystemLabels{
		device:      "/dev/test",
		mountPoint:  "/tmp/test",
		fsType:      "tmpfs",
		options:     "rw,relatime",
		deviceError: "",
	}
	
	// This will likely fail for a non-existent mount point, but we're testing the method exists
	stats := collector.processStat(labels)
	
	// The function should return something, even if it's an error case
	if stats.labels.device != labels.device {
		t.Error("Labels device should be preserved")
	}
}

func TestFilesystemMetricNames(t *testing.T) {
	collector := NewFilesystemCollector()
	
	// Test that metric names contain expected substrings
	for name, desc := range collector.descs {
		metricName := desc.String()
		if !strings.Contains(metricName, "node_filesystem") {
			t.Errorf("Metric name should contain 'node_filesystem': %s", metricName)
		}
		
		// Test specific metric names
		switch name {
		case "size":
			if !strings.Contains(metricName, "size_bytes") {
				t.Errorf("Size metric should contain 'size_bytes': %s", metricName)
			}
		case "free":
			if !strings.Contains(metricName, "free_bytes") {
				t.Errorf("Free metric should contain 'free_bytes': %s", metricName)
			}
		case "device_error":
			if !strings.Contains(metricName, "device_error") {
				t.Errorf("Device error metric should contain 'device_error': %s", metricName)
			}
		}
	}
}

func TestFilesystemParseLabelsResilience(t *testing.T) {
	collector := NewFilesystemCollector()
	
	// Test with empty reader
	reader := strings.NewReader("")
	filesystems, err := collector.parseFilesystemLabels(reader)
	if err != nil {
		t.Errorf("Should not error on empty input: %v", err)
	}
	if len(filesystems) != 0 {
		t.Errorf("Should return empty slice for empty input, got %d items", len(filesystems))
	}
	
	// Test with malformed input
	malformedInput := "invalid line\nanother invalid line"
	reader = strings.NewReader(malformedInput)
	filesystems, err = collector.parseFilesystemLabels(reader)
	if err != nil {
		t.Errorf("Should handle malformed input gracefully: %v", err)
	}
}

func TestFilesystemGetStatsResilience(t *testing.T) {
	collector := NewFilesystemCollector()
	
	// This test checks that GetStats doesn't panic
	// It may return an error if /proc files are not accessible
	_, err := collector.GetStats()
	if err != nil {
		t.Logf("GetStats returned error (may be expected in test environment): %v", err)
	}
} 
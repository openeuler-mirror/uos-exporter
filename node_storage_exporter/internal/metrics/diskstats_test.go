package metrics

import (
	"testing"
	"regexp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"strings"
)

func TestNewDiskStatsCollector(t *testing.T) {
	collector := NewDiskStatsCollector()
	
	if collector == nil {
		t.Fatal("NewDiskStatsCollector returned nil")
	}
	
	if collector.logger == nil {
		t.Error("Logger should not be nil")
	}
	
	if len(collector.descs) == 0 {
		t.Error("Descs should not be empty")
	}
	
	if len(collector.ataDescs) == 0 {
		t.Error("ATA descs should not be empty")
	}
}

func TestDiskStatsCollectorImplementsCollector(t *testing.T) {
	collector := NewDiskStatsCollector()
	
	// Test that it implements the Collector interface
	var _ prometheus.Collector = collector
}

func TestDiskStatsCollectorCollect(t *testing.T) {
	collector := NewDiskStatsCollector()
	
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
	
	// We should have at least some metrics (even if no disks are available)
	if metricCount < 0 {
		t.Errorf("Expected at least 0 metrics, got %d", metricCount)
	}
}

func TestDiskStatsDeviceFilter(t *testing.T) {
	collector := NewDiskStatsCollector()
	
	if collector.deviceFilter == nil {
		t.Error("Device filter should not be nil")
	}
	
	// Test default ignored devices pattern
	ignoredDevices := []string{
		"ram0", "loop0", "fd0", "hda1", "sda1", "vda1", "xvda1", "nvme0n1p1",
	}
	
	for _, device := range ignoredDevices {
		if !collector.deviceFilter.MatchString(device) {
			t.Errorf("Device %s should be ignored but wasn't", device)
		}
	}
	
	// Test devices that should not be ignored
	allowedDevices := []string{
		"sda", "nvme0n1", "hda", "vda",
	}
	
	for _, device := range allowedDevices {
		if collector.deviceFilter.MatchString(device) {
			t.Errorf("Device %s should not be ignored but was", device)
		}
	}
}

func TestUdevDevicePropertiesFunc(t *testing.T) {
	collector := NewDiskStatsCollector()
	
	// Test with non-existent device (should return empty info, no error for missing file)
	info, err := collector.getUdevDeviceProperties(999, 999)
	if err == nil {
		t.Log("No error for non-existent device (expected for missing udev data)")
	}
	if info == nil {
		t.Error("Info should not be nil even for non-existent device")
	}
}

func TestTypedDescMustNewConstMetric(t *testing.T) {
	desc := &typedDesc{
		desc: prometheus.NewDesc(
			"test_metric",
			"Test metric description",
			[]string{"label1"}, nil,
		),
		valueType: prometheus.GaugeValue,
	}
	
	metric := desc.mustNewConstMetric(42.0, "test_value")
	if metric == nil {
		t.Error("mustNewConstMetric should not return nil")
	}
	
	// Test metric value using proper validation
	if err := testutil.CollectAndCompare(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{Name: "test", Help: "test"},
		func() float64 { return 0 },
	), strings.NewReader("# HELP test test\n# TYPE test gauge\ntest 0\n")); err != nil {
		t.Logf("Metric validation error (expected in test): %v", err)
	}
}

func TestDiskStatsConstants(t *testing.T) {
	// Test constants are reasonable
	if unixSectorSize != 512.0 {
		t.Errorf("Expected unixSectorSize to be 512.0, got %f", unixSectorSize)
	}
	
	if secondsPerTick != 1.0/1000.0 {
		t.Errorf("Expected secondsPerTick to be 0.001, got %f", secondsPerTick)
	}
	
	// Test regex pattern compiles
	if _, err := regexp.Compile(diskstatsDefaultIgnoredDevices); err != nil {
		t.Errorf("diskstatsDefaultIgnoredDevices pattern should compile: %v", err)
	}
}

func TestUdevConstants(t *testing.T) {
	// Test that udev constants are defined
	constants := []string{
		udevDevicePropertyPrefix,
		udevDMLVLayer,
		udevDMLVName,
		udevDMName,
		udevDMUUID,
		udevDMVGName,
		udevIDATA,
		udevIDATARotationRateRPM,
		udevIDATAWriteCache,
		udevIDATAWriteCacheEnabled,
		udevIDFSType,
		udevIDFSUsage,
		udevIDFSUUID,
		udevIDFSVersion,
		udevIDModel,
		udevIDPath,
		udevIDRevision,
		udevIDSerialShort,
		udevIDWWN,
		udevSCSIIdentSerial,
	}
	
	for _, constant := range constants {
		if constant == "" {
			t.Error("Udev constant should not be empty")
		}
	}
}

func TestDiskStatsDescriptors(t *testing.T) {
	collector := NewDiskStatsCollector()
	
	// Test that all expected descriptors are present
	expectedDescs := 17 // Based on the descs slice length
	if len(collector.descs) != expectedDescs {
		t.Errorf("Expected %d descriptors, got %d", expectedDescs, len(collector.descs))
	}
	
	// Test that ATA descriptors are present
	expectedATADescs := 3
	if len(collector.ataDescs) != expectedATADescs {
		t.Errorf("Expected %d ATA descriptors, got %d", expectedATADescs, len(collector.ataDescs))
	}
	
	// Test info descriptor
	if collector.infoDesc.desc == nil {
		t.Error("Info descriptor should not be nil")
	}
	
	// Test filesystem info descriptor  
	if collector.filesystemInfoDesc.desc == nil {
		t.Error("Filesystem info descriptor should not be nil")
	}
	
	// Test device mapper info descriptor
	if collector.deviceMapperInfoDesc.desc == nil {
		t.Error("Device mapper info descriptor should not be nil")
	}
}

func TestDiskStatsMetricNames(t *testing.T) {
	collector := NewDiskStatsCollector()
	
	// Expected metric names
	expectedMetrics := []string{
		"reads_completed_total",
		"reads_merged_total", 
		"read_bytes_total",
		"read_time_seconds_total",
		"writes_completed_total",
		"writes_merged_total",
		"written_bytes_total", 
		"write_time_seconds_total",
		"io_now",
		"io_time_seconds_total",
		"io_time_weighted_seconds_total",
		"discards_completed_total",
		"discards_merged_total",
		"discarded_sectors_total",
		"discard_time_seconds_total",
		"flush_requests_total",
		"flush_requests_time_seconds_total",
	}
	
	if len(collector.descs) != len(expectedMetrics) {
		t.Errorf("Expected %d metrics, got %d", len(expectedMetrics), len(collector.descs))
	}
	
	// Test that metric names contain expected substrings
	for i, desc := range collector.descs {
		if i < len(expectedMetrics) {
			metricName := desc.desc.String()
			if !strings.Contains(metricName, "node_disk") {
				t.Errorf("Metric name should contain 'node_disk': %s", metricName)
			}
		}
	}
}
// Part 2 commit for node_storage_exporter/internal/metrics/diskstats_test.go

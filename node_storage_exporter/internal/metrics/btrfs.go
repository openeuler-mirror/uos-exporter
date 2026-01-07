package metrics

import (
	"log/slog"
	"path"
	"strings"
	"syscall"

	dennwc "github.com/dennwc/btrfs"
	"node_storage_exporter/internal/exporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs/btrfs"
)

func init() {
	exporter.Register(NewBtrfsCollector())
}

type btrfsIoctlFsDevStats struct {
	path string
	uuid string

	bytesUsed  uint64
	totalBytes uint64

	writeErrs      uint64
	readErrs       uint64
	flushErrs      uint64
	corruptionErrs uint64
	generationErrs uint64
}

type btrfsIoctlFsStats struct {
	uuid    string
	devices []btrfsIoctlFsDevStats
}

type btrfsMetric struct {
	name            string
	metricType      prometheus.ValueType
	desc            string
	value           float64
	extraLabel      []string
	extraLabelValue []string
}

type BtrfsCollector struct {
	fs     btrfs.FS
	logger *slog.Logger
}

func NewBtrfsCollector() *BtrfsCollector {
	logger := slog.Default()
	
	fs, err := btrfs.NewFS("/sys")
	if err != nil {
		logger.Debug("Failed to open Btrfs filesystem", "error", err)
		return &BtrfsCollector{
			logger: logger,
		}
	}

	return &BtrfsCollector{
		fs:     fs,
		logger: logger,
	}
}

func (c *BtrfsCollector) Collect(ch chan<- prometheus.Metric) {
	if c.fs == (btrfs.FS{}) {
		c.logger.Debug("Btrfs filesystem not available")
		return
	}

	stats, err := c.fs.Stats()
	if err != nil {
		c.logger.Debug("Failed to retrieve Btrfs stats from procfs", "error", err)
		return
	}

	ioctlStatsMap, err := c.getIoctlStats()
	if err != nil {
		c.logger.Debug("Error querying btrfs device stats with ioctl", "error", err)
		ioctlStatsMap = make(map[string]*btrfsIoctlFsStats)
	}

	for _, s := range stats {
		// match up procfs and ioctl info by filesystem UUID (without dashes)
		var fsUUID = strings.Replace(s.UUID, "-", "", -1)
		ioctlStats := ioctlStatsMap[fsUUID]
		c.updateBtrfsStats(ch, s, ioctlStats)
	}
}

func (c *BtrfsCollector) updateBtrfsStats(ch chan<- prometheus.Metric, s *btrfs.Stats, ioctlStats *btrfsIoctlFsStats) {
	const subsystem = "btrfs"
	devLabels := []string{"uuid"}

	for _, m := range c.getMetrics(s, ioctlStats) {
		labels := append(devLabels, m.extraLabel...)

		desc := prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, m.name),
			m.desc,
			labels,
			nil,
		)

		labelValues := []string{s.UUID}
		if len(m.extraLabelValue) > 0 {
			labelValues = append(labelValues, m.extraLabelValue...)
		}

		ch <- prometheus.MustNewConstMetric(
			desc,
			m.metricType,
			m.value,
			labelValues...,
		)
	}
}

func (c *BtrfsCollector) getMetrics(s *btrfs.Stats, ioctlStats *btrfsIoctlFsStats) []btrfsMetric {
	metrics := []btrfsMetric{
		{
			name:            "info",
			desc:            "Filesystem information",
			value:           1,
			metricType:      prometheus.GaugeValue,
			extraLabel:      []string{"label"},
			extraLabelValue: []string{s.Label},
		},
		{
			name:       "global_rsv_size_bytes",
			desc:       "Size of global reserve.",
			metricType: prometheus.GaugeValue,
			value:      float64(s.Allocation.GlobalRsvSize),
		},
		{
			name:       "commits_total",
			desc:       "The total number of commits that have occurred.",
			metricType: prometheus.CounterValue,
			value:      float64(s.CommitStats.Commits),
		},
		{
			name:       "last_commit_seconds",
			desc:       "Duration of the most recent commit, in seconds.",
			metricType: prometheus.GaugeValue,
			value:      float64(s.CommitStats.LastCommitMs) / 1000,
		},
		{
			name:       "max_commit_seconds",
			desc:       "Duration of the slowest commit, in seconds.",
			metricType: prometheus.GaugeValue,
			value:      float64(s.CommitStats.MaxCommitMs) / 1000,
		},
		{
			name:       "commit_seconds_total",
			desc:       "Sum of the duration of all commits, in seconds.",
			metricType: prometheus.CounterValue,
			value:      float64(s.CommitStats.TotalCommitMs) / 1000,
		},
	}

	// Information about data, metadata and system data.
	metrics = append(metrics, c.getAllocationStats("data", s.Allocation.Data)...)
	metrics = append(metrics, c.getAllocationStats("metadata", s.Allocation.Metadata)...)
	metrics = append(metrics, c.getAllocationStats("system", s.Allocation.System)...)

	// Information about devices.
	if ioctlStats == nil {
		for n, dev := range s.Devices {
			metrics = append(metrics, btrfsMetric{
				name:            "device_size_bytes",
				desc:            "Size of a device that is part of the filesystem.",
				metricType:      prometheus.GaugeValue,
				value:           float64(dev.Size),
				extraLabel:      []string{"device"},
				extraLabelValue: []string{n},
			})
		}
		return metrics
	}

	for _, dev := range ioctlStats.devices {
		// trim the path prefix from the device name so the value should match
		// the value used in the fallback branch above.
		// e.g. /dev/sda -> sda, /rootfs/dev/md1 -> md1
		_, device := path.Split(dev.path)

		extraLabels := []string{"device", "btrfs_dev_uuid"}
		extraLabelValues := []string{device, dev.uuid}

		metrics = append(metrics,
			btrfsMetric{
				name:            "device_size_bytes",
				desc:            "Size of a device that is part of the filesystem.",
				metricType:      prometheus.GaugeValue,
				value:           float64(dev.totalBytes),
				extraLabel:      extraLabels,
				extraLabelValue: extraLabelValues,
			},
			btrfsMetric{
				name:            "device_unused_bytes",
				desc:            "Unused bytes unused on a device that is part of the filesystem.",
				metricType:      prometheus.GaugeValue,
				value:           float64(dev.totalBytes - dev.bytesUsed),
				extraLabel:      extraLabels,
				extraLabelValue: extraLabelValues,
			})

		errorLabels := append([]string{"type"}, extraLabels...)
		values := []uint64{
			dev.writeErrs,
			dev.readErrs,
			dev.flushErrs,
			dev.corruptionErrs,
			dev.generationErrs,
		}
		btrfsErrorTypeNames := []string{
			"write",
			"read",
			"flush",
			"corruption",
			"generation",
		}

		for i, errorType := range btrfsErrorTypeNames {
			metrics = append(metrics,
				btrfsMetric{
					name:            "device_errors_total",
					desc:            "Errors reported for the device",
					metricType:      prometheus.CounterValue,
					value:           float64(values[i]),
					extraLabel:      errorLabels,
					extraLabelValue: append([]string{errorType}, extraLabelValues...),
				})
		}
	}

	return metrics
}

func (c *BtrfsCollector) getAllocationStats(a string, s *btrfs.AllocationStats) []btrfsMetric {
	metrics := []btrfsMetric{
		{
			name:            "reserved_bytes",
			desc:            "Amount of space reserved for a data type",
			metricType:      prometheus.GaugeValue,
			value:           float64(s.ReservedBytes),
			extraLabel:      []string{"block_group_type"},
			extraLabelValue: []string{a},
		},
	}

	// Add all layout statistics.
	for layout, stats := range s.Layouts {
		metrics = append(metrics, c.getLayoutStats(a, layout, stats)...)
	}

	return metrics
}

func (c *BtrfsCollector) getLayoutStats(a, l string, s *btrfs.LayoutUsage) []btrfsMetric {
	return []btrfsMetric{
		{
			name:            "used_bytes",
			desc:            "Amount of used space by a layout/data type",
			metricType:      prometheus.GaugeValue,
			value:           float64(s.UsedBytes),
			extraLabel:      []string{"block_group_type", "mode"},
			extraLabelValue: []string{a, l},
		},
		{
			name:            "size_bytes",
			desc:            "Amount of space allocated for a layout/data type",
			metricType:      prometheus.GaugeValue,
			value:           float64(s.TotalBytes),
			extraLabel:      []string{"block_group_type", "mode"},
			extraLabelValue: []string{a, l},
		},
		{
			name:            "allocation_ratio",
			desc:            "Data allocation ratio for a layout/data type",
			metricType:      prometheus.GaugeValue,
			value:           s.Ratio,
			extraLabel:      []string{"block_group_type", "mode"},
			extraLabelValue: []string{a, l},
		},
	}
}

func (c *BtrfsCollector) getIoctlStats() (map[string]*btrfsIoctlFsStats, error) {
	// Instead of introducing more ioctl calls to scan for all btrfs
	// filesystems re-use our mount point utils to find known mounts
	mountsList, err := c.mountPointDetails()
	if err != nil {
		return nil, err
	}

	// Track devices we have successfully scanned, by device path.
	devicesDone := make(map[string]struct{})
	// Filesystems scan results by UUID.
	fsStats := make(map[string]*btrfsIoctlFsStats)

	for _, mount := range mountsList {
		if mount.fsType != "btrfs" {
			continue
		}

		if _, found := devicesDone[mount.device]; found {
			// We already found this filesystem by another mount point.
			continue
		}

		mountPath := mount.mountPoint

		fs, err := dennwc.Open(mountPath, true)
		if err != nil {
			// Failed to open this mount point, maybe we didn't have permission
			// maybe we'll find another mount point for this FS later.
			c.logger.Debug("Error inspecting btrfs mountpoint", "mountPoint", mountPath, "error", err)
			continue
		}
		defer fs.Close()

		fsInfo, err := fs.Info()
		if err != nil {
			// Failed to get the FS info for some reason,
			// perhaps it'll work with a different mount point
			c.logger.Debug("Error querying btrfs filesystem", "mountPoint", mountPath, "error", err)
			continue
		}

		fsID := fsInfo.FSID.String()
		if _, found := fsStats[fsID]; found {
			// We already found this filesystem by another mount point
			continue
		}

		deviceStats, err := c.getIoctlDeviceStats(fs, &fsInfo)
		if err != nil {
			c.logger.Debug("Error querying btrfs device stats", "mountPoint", mountPath, "error", err)
			continue
		}

		devicesDone[mount.device] = struct{}{}
		fsStats[fsID] = &btrfsIoctlFsStats{
			uuid:    fsID,
			devices: deviceStats,
		}
	}

	return fsStats, nil
}

func (c *BtrfsCollector) getIoctlDeviceStats(fs *dennwc.FS, fsInfo *dennwc.Info) ([]btrfsIoctlFsDevStats, error) {
	devices := make([]btrfsIoctlFsDevStats, 0, fsInfo.NumDevices)

	for i := uint64(0); i <= fsInfo.MaxID; i++ {
		deviceInfo, err := fs.GetDevInfo(i)

		if err != nil {
			if errno, ok := err.(syscall.Errno); ok && errno == syscall.ENODEV {
				// Device IDs do not consistently start at 0, nor are ranges contiguous, so we expect this.
				continue
			}
			return nil, err
		}

		deviceStats, err := fs.GetDevStats(i)
		if err != nil {
			return nil, err
		}

		devices = append(devices, btrfsIoctlFsDevStats{
			path:       deviceInfo.Path,
			uuid:       deviceInfo.UUID.String(),
			bytesUsed:  deviceInfo.BytesUsed,
			totalBytes: deviceInfo.TotalBytes,

			writeErrs:      deviceStats.WriteErrs,
			readErrs:       deviceStats.ReadErrs,
			flushErrs:      deviceStats.FlushErrs,
			corruptionErrs: deviceStats.CorruptionErrs,
			generationErrs: deviceStats.GenerationErrs,
		})
	}

	return devices, nil
}

// Simple mount point details for btrfs
type mountInfo struct {
	device     string
	mountPoint string
	fsType     string
}

func (c *BtrfsCollector) mountPointDetails() ([]mountInfo, error) {
	// This is a simplified version - in a real implementation,
	// you would parse /proc/mounts or /proc/self/mountinfo
	// For now, return empty slice to avoid dependency issues
	return []mountInfo{}, nil
}

func (c *BtrfsCollector) Describe(ch chan<- *prometheus.Desc) {
	// Btrfs collector creates dynamic descriptors
} 
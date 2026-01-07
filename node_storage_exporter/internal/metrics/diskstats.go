package metrics

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"node_storage_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs/blockdevice"
)

const (
	diskstatsDefaultIgnoredDevices = `^(z?ram|loop|fd|(h|s|v|xv)d[a-z]|nvme\d+n\d+p)\d+$`
	unixSectorSize                 = 512.0
	secondsPerTick                 = 1.0 / 1000.0

	// Udev device properties
	udevDevicePropertyPrefix    = "E:"
	udevDMLVLayer               = "DM_LV_LAYER"
	udevDMLVName                = "DM_LV_NAME"
	udevDMName                  = "DM_NAME"
	udevDMUUID                  = "DM_UUID"
	udevDMVGName                = "DM_VG_NAME"
	udevIDATA                   = "ID_ATA"
	udevIDATARotationRateRPM    = "ID_ATA_ROTATION_RATE_RPM"
	udevIDATASATA               = "ID_ATA_SATA"
	udevIDATASATASignalRateGen1 = "ID_ATA_SATA_SIGNAL_RATE_GEN1"
	udevIDATASATASignalRateGen2 = "ID_ATA_SATA_SIGNAL_RATE_GEN2"
	udevIDATAWriteCache         = "ID_ATA_WRITE_CACHE"
	udevIDATAWriteCacheEnabled  = "ID_ATA_WRITE_CACHE_ENABLED"
	udevIDFSType                = "ID_FS_TYPE"
	udevIDFSUsage               = "ID_FS_USAGE"
	udevIDFSUUID                = "ID_FS_UUID"
	udevIDFSVersion             = "ID_FS_VERSION"
	udevIDModel                 = "ID_MODEL"
	udevIDPath                  = "ID_PATH"
	udevIDRevision              = "ID_REVISION"
	udevIDSerialShort           = "ID_SERIAL_SHORT"
	udevIDWWN                   = "ID_WWN"
	udevSCSIIdentSerial         = "SCSI_IDENT_SERIAL"
)

type udevInfo map[string]string

type typedDesc struct {
	desc      *prometheus.Desc
	valueType prometheus.ValueType
}

func (d *typedDesc) mustNewConstMetric(value float64, labels ...string) prometheus.Metric {
	return prometheus.MustNewConstMetric(d.desc, d.valueType, value, labels...)
}

func init() {
	exporter.Register(NewDiskStatsCollector())
}

type DiskStatsCollector struct {
	logger               *slog.Logger
	deviceFilter         *regexp.Regexp
	fs                   *blockdevice.FS
	infoDesc             typedDesc
	descs                []typedDesc
	filesystemInfoDesc   typedDesc
	deviceMapperInfoDesc typedDesc
	ataDescs             map[string]typedDesc
}

func NewDiskStatsCollector() *DiskStatsCollector {
	logger := slog.Default()

	// 编译设备过滤正则表达式
	deviceFilter, err := regexp.Compile(diskstatsDefaultIgnoredDevices)
	if err != nil {
		logger.Debug("Failed to compile device filter regex", "error", err)
		deviceFilter = nil
	}

	// 初始化 blockdevice 文件系统
	var fs *blockdevice.FS
	if bfs, err := blockdevice.NewFS("/proc", "/sys"); err != nil {
		logger.Debug("Failed to open blockdevice FS", "error", err)
		fs = nil
	} else {
		fs = &bfs
	}

	const subsystem = "disk"
	diskLabels := []string{"device"}

	return &DiskStatsCollector{
		logger:       logger,
		deviceFilter: deviceFilter,
		fs:           fs,
		infoDesc: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName("node", subsystem, "info"),
				"Info of /sys/block/<block_device>.",
				[]string{"device", "major", "minor", "path", "wwn", "model", "serial", "revision", "rotational"}, nil,
			),
			valueType: prometheus.GaugeValue,
		},
		descs: []typedDesc{
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName("node", subsystem, "reads_completed_total"),
					"The total number of reads completed successfully.",
					diskLabels, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName("node", subsystem, "reads_merged_total"),
					"The total number of reads merged.",
					diskLabels, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName("node", subsystem, "read_bytes_total"),
					"The total number of bytes read successfully.",
					diskLabels, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName("node", subsystem, "read_time_seconds_total"),
					"The total number of seconds spent by all reads.",
					diskLabels, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName("node", subsystem, "writes_completed_total"),
					"The total number of writes completed successfully.",
					diskLabels, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName("node", subsystem, "writes_merged_total"),
					"The number of writes merged.",
					diskLabels, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName("node", subsystem, "written_bytes_total"),
					"The total number of bytes written successfully.",
					diskLabels, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName("node", subsystem, "write_time_seconds_total"),
					"This is the total number of seconds spent by all writes.",
					diskLabels, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName("node", subsystem, "io_now"),
					"The number of I/Os currently in progress.",
					diskLabels, nil,
				),
				valueType: prometheus.GaugeValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName("node", subsystem, "io_time_seconds_total"),
					"Total seconds spent doing I/Os.",
					diskLabels, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName("node", subsystem, "io_time_weighted_seconds_total"),
					"The weighted # of seconds spent doing I/Os.",
					diskLabels, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName("node", subsystem, "discards_completed_total"),
					"The total number of discards completed successfully.",
					diskLabels, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName("node", subsystem, "discards_merged_total"),
					"The total number of discards merged.",
					diskLabels, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName("node", subsystem, "discarded_sectors_total"),
					"The total number of sectors discarded successfully.",
					diskLabels, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName("node", subsystem, "discard_time_seconds_total"),
					"This is the total number of seconds spent by all discards.",
					diskLabels, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName("node", subsystem, "flush_requests_total"),
					"The total number of flush requests completed successfully",
					diskLabels, nil,
				),
				valueType: prometheus.CounterValue,
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName("node", subsystem, "flush_requests_time_seconds_total"),
					"This is the total number of seconds spent by all flush requests.",
					diskLabels, nil,
				),
				valueType: prometheus.CounterValue,
			},
		},
		filesystemInfoDesc: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName("node", subsystem, "filesystem_info"),
				"Info about disk filesystem.",
				[]string{"device", "type", "usage", "uuid", "version"}, nil,
			),
			valueType: prometheus.GaugeValue,
		},
		deviceMapperInfoDesc: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName("node", subsystem, "device_mapper_info"),
				"Info about disk device mapper.",
				[]string{"device", "name", "uuid", "vg_name", "lv_name", "lv_layer"}, nil,
			),
			valueType: prometheus.GaugeValue,
		},
		ataDescs: map[string]typedDesc{
			udevIDATAWriteCache: {
				desc: prometheus.NewDesc(
					prometheus.BuildFQName("node", subsystem, "ata_write_cache"),
					"ATA disk has a write cache.",
					[]string{"device"}, nil,
				),
				valueType: prometheus.GaugeValue,
			},
			udevIDATAWriteCacheEnabled: {
				desc: prometheus.NewDesc(
					prometheus.BuildFQName("node", subsystem, "ata_write_cache_enabled"),
					"ATA disk has its write cache enabled.",
					[]string{"device"}, nil,
				),
				valueType: prometheus.GaugeValue,
			},
			udevIDATARotationRateRPM: {
				desc: prometheus.NewDesc(
					prometheus.BuildFQName("node", subsystem, "ata_rotation_rate_rpm"),
					"ATA disk rotation rate in RPMs (0 for SSDs).",
					[]string{"device"}, nil,
				),
				valueType: prometheus.GaugeValue,
			},
		},
	}
}

func (c *DiskStatsCollector) Collect(ch chan<- prometheus.Metric) {
	if err := c.updateDiskStats(ch); err != nil {
		c.logger.Debug("Error updating disk stats", "error", err)
	}
}

func (c *DiskStatsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.infoDesc.desc
	for _, desc := range c.descs {
		ch <- desc.desc
	}
	ch <- c.filesystemInfoDesc.desc
	ch <- c.deviceMapperInfoDesc.desc
	for _, desc := range c.ataDescs {
		ch <- desc.desc
	}
}

func (c *DiskStatsCollector) updateDiskStats(ch chan<- prometheus.Metric) error {
	if c.fs == nil {
		return fmt.Errorf("blockdevice filesystem not available")
	}

	diskStats, err := c.fs.ProcDiskstats()
	if err != nil {
		return fmt.Errorf("couldn't get diskstats: %w", err)
	}

	for _, stats := range diskStats {
		dev := stats.DeviceName
		if c.deviceFilter != nil && c.deviceFilter.MatchString(dev) {
			continue
		}

		info, err := c.getUdevDeviceProperties(stats.MajorNumber, stats.MinorNumber)
		if err != nil {
			c.logger.Debug("Failed to parse udev info", "err", err)
		}

		// This is usually the serial printed on the disk label.
		serial := info[udevSCSIIdentSerial]

		// If it's undefined, fallback to ID_SERIAL_SHORT instead.
		if serial == "" {
			serial = info[udevIDSerialShort]
		}

		queueStats, err := c.fs.SysBlockDeviceQueueStats(dev)
		// Block Device Queue stats may not exist for all devices.
		if err != nil && !os.IsNotExist(err) {
			c.logger.Debug("Failed to get block device queue stats", "device", dev, "err", err)
		}

		ch <- c.infoDesc.mustNewConstMetric(1.0, dev,
			fmt.Sprint(stats.MajorNumber),
			fmt.Sprint(stats.MinorNumber),
			info[udevIDPath],
			info[udevIDWWN],
			info[udevIDModel],
			serial,
			info[udevIDRevision],
			strconv.FormatUint(queueStats.Rotational, 2),
		)

		statCount := stats.IoStatsCount - 3 // Total diskstats record count, less MajorNumber, MinorNumber and DeviceName

		for i, val := range []float64{
			float64(stats.ReadIOs),
			float64(stats.ReadMerges),
			float64(stats.ReadSectors) * unixSectorSize,
			float64(stats.ReadTicks) * secondsPerTick,
			float64(stats.WriteIOs),
			float64(stats.WriteMerges),
			float64(stats.WriteSectors) * unixSectorSize,
			float64(stats.WriteTicks) * secondsPerTick,
			float64(stats.IOsInProgress),
			float64(stats.IOsTotalTicks) * secondsPerTick,
			float64(stats.WeightedIOTicks) * secondsPerTick,
			float64(stats.DiscardIOs),
			float64(stats.DiscardMerges),
			float64(stats.DiscardSectors),
			float64(stats.DiscardTicks) * secondsPerTick,
			float64(stats.FlushRequestsCompleted),
			float64(stats.TimeSpentFlushing) * secondsPerTick,
		} {
			if i >= statCount {
				break
			}
			ch <- c.descs[i].mustNewConstMetric(val, dev)
		}

		// Handle filesystem info
		if fsType := info[udevIDFSType]; fsType != "" {
			ch <- c.filesystemInfoDesc.mustNewConstMetric(1.0, dev,
				fsType,
				info[udevIDFSUsage],
				info[udevIDFSUUID],
				info[udevIDFSVersion],
			)
		}

		// Handle device mapper info
		if name := info[udevDMName]; name != "" {
			ch <- c.deviceMapperInfoDesc.mustNewConstMetric(1.0, dev,
				name,
				info[udevDMUUID],
				info[udevDMVGName],
				info[udevDMLVName],
				info[udevDMLVLayer],
			)
		}

		// Handle ATA info
		if ata := info[udevIDATA]; ata != "" {
			for attr, desc := range c.ataDescs {
				str, ok := info[attr]
				if !ok {
					c.logger.Debug("Udev attribute does not exist", "attribute", attr)
					continue
				}

				if value, err := strconv.ParseFloat(str, 64); err == nil {
					ch <- desc.mustNewConstMetric(value, dev)
				} else {
					c.logger.Error("Failed to parse ATA value", "err", err)
				}
			}
		}
	}

	return nil
}

func (c *DiskStatsCollector) getUdevDeviceProperties(major, minor uint32) (udevInfo, error) {
	filename := fmt.Sprintf("/run/udev/data/b%d:%d", major, minor)
	cleanPath := filepath.Clean(filename)
	statDir := "/run/udev/data"
	if !strings.HasPrefix(cleanPath, statDir) {
		return nil, fmt.Errorf("udev data file must be located within %s", statDir)
	}
	data, err := os.Open(filename)
	if err != nil {
		return udevInfo{}, err
	}
	defer data.Close()

	info := make(udevInfo)

	scanner := bufio.NewScanner(data)
	for scanner.Scan() {
		line := scanner.Text()

		// We're only interested in device properties.
		if !strings.HasPrefix(line, udevDevicePropertyPrefix) {
			continue
		}

		line = strings.TrimPrefix(line, udevDevicePropertyPrefix)

		if name, value, found := strings.Cut(line, "="); found {
			info[name] = value
		}
	}

	return info, nil
}

package metrics

import (
	"bufio"
	"errors"
	"io"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"node_storage_exporter/internal/exporter"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sys/unix"
)

func init() {
	exporter.Register(NewFilesystemCollector())
}

const (
	defMountPointsExcluded = "^/(dev|proc|run/credentials/.+|sys|var/lib/docker/.+|var/lib/containers/storage/.+)($|/)"
	defFSTypesExcluded     = "^(autofs|binfmt_misc|bpf|cgroup2?|configfs|debugfs|devpts|devtmpfs|fusectl|hugetlbfs|iso9660|mqueue|nsfs|overlay|proc|procfs|pstore|rpc_pipefs|securityfs|selinuxfs|squashfs|sysfs|tracefs)$"
	mountTimeout           = 5 * time.Second
)

type filesystemLabels struct {
	device      string
	mountPoint  string
	fsType      string
	options     string
	deviceError string
	major       string
	minor       string
}

type filesystemStats struct {
	labels      filesystemLabels
	size        float64
	free        float64
	avail       float64
	files       float64
	filesFree   float64
	ro          float64
	deviceError float64
}

type FilesystemCollector struct {
	logger                *slog.Logger
	ignoredMountPoints    *regexp.Regexp
	ignoredFSTypes        *regexp.Regexp
	stalenessTtimeout     time.Duration
	filesystemLabelNames  []string
	descs                 map[string]*prometheus.Desc
}

func NewFilesystemCollector() *FilesystemCollector {
	logger := slog.Default()

	subsystem := "filesystem"
	labels := []string{"device", "mountpoint", "fstype", "options", "major", "minor"}

	return &FilesystemCollector{
		ignoredMountPoints: regexp.MustCompile(defMountPointsExcluded),
		ignoredFSTypes:     regexp.MustCompile(defFSTypesExcluded),
		logger:             logger,
		descs: map[string]*prometheus.Desc{
			"size": prometheus.NewDesc(
				prometheus.BuildFQName("node", subsystem, "size_bytes"),
				"Filesystem size in bytes.",
				labels, nil,
			),
			"free": prometheus.NewDesc(
				prometheus.BuildFQName("node", subsystem, "free_bytes"),
				"Filesystem free space in bytes.",
				labels, nil,
			),
			"avail": prometheus.NewDesc(
				prometheus.BuildFQName("node", subsystem, "avail_bytes"),
				"Filesystem space available to non-root users in bytes.",
				labels, nil,
			),
			"files": prometheus.NewDesc(
				prometheus.BuildFQName("node", subsystem, "files"),
				"Filesystem total file nodes.",
				labels, nil,
			),
			"files_free": prometheus.NewDesc(
				prometheus.BuildFQName("node", subsystem, "files_free"),
				"Filesystem total free file nodes.",
				labels, nil,
			),
			"ro": prometheus.NewDesc(
				prometheus.BuildFQName("node", subsystem, "readonly"),
				"Filesystem read-only status.",
				labels, nil,
			),
			"device_error": prometheus.NewDesc(
				prometheus.BuildFQName("node", subsystem, "device_error"),
				"Whether an error occurred while getting statistics for the given device.",
				labels, nil,
			),
			"mount_info": prometheus.NewDesc(
				prometheus.BuildFQName("node", subsystem, "mount_info"),
				"Filesystem mount information.",
				[]string{"device", "mountpoint", "fstype", "options", "major", "minor"}, nil,
			),
		},
	}
}

func (c *FilesystemCollector) Collect(ch chan<- prometheus.Metric) {
	stats, err := c.GetStats()
	if err != nil {
		c.logger.Debug("Error getting filesystem stats", "error", err)
		return
	}

	for _, s := range stats {
		ch <- prometheus.MustNewConstMetric(
			c.descs["size"], prometheus.GaugeValue,
			s.size, s.labels.device, s.labels.mountPoint, s.labels.fsType, s.labels.options, s.labels.major, s.labels.minor,
		)
		ch <- prometheus.MustNewConstMetric(
			c.descs["free"], prometheus.GaugeValue,
			s.free, s.labels.device, s.labels.mountPoint, s.labels.fsType, s.labels.options, s.labels.major, s.labels.minor,
		)
		ch <- prometheus.MustNewConstMetric(
			c.descs["avail"], prometheus.GaugeValue,
			s.avail, s.labels.device, s.labels.mountPoint, s.labels.fsType, s.labels.options, s.labels.major, s.labels.minor,
		)
		ch <- prometheus.MustNewConstMetric(
			c.descs["files"], prometheus.GaugeValue,
			s.files, s.labels.device, s.labels.mountPoint, s.labels.fsType, s.labels.options, s.labels.major, s.labels.minor,
		)
		ch <- prometheus.MustNewConstMetric(
			c.descs["files_free"], prometheus.GaugeValue,
			s.filesFree, s.labels.device, s.labels.mountPoint, s.labels.fsType, s.labels.options, s.labels.major, s.labels.minor,
		)
		ch <- prometheus.MustNewConstMetric(
			c.descs["ro"], prometheus.GaugeValue,
			s.ro, s.labels.device, s.labels.mountPoint, s.labels.fsType, s.labels.options, s.labels.major, s.labels.minor,
		)
		ch <- prometheus.MustNewConstMetric(
			c.descs["device_error"], prometheus.GaugeValue,
			s.deviceError, s.labels.device, s.labels.mountPoint, s.labels.fsType, s.labels.options, s.labels.major, s.labels.minor,
		)
		ch <- prometheus.MustNewConstMetric(
			c.descs["mount_info"], prometheus.GaugeValue,
			1, s.labels.device, s.labels.mountPoint, s.labels.fsType, s.labels.options, s.labels.major, s.labels.minor,
		)
	}
}

func (c *FilesystemCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, desc := range c.descs {
		ch <- desc
	}
}

func (c *FilesystemCollector) GetStats() ([]filesystemStats, error) {
	mps, err := c.mountPointDetails()
	if err != nil {
		return nil, err
	}

	stats := []filesystemStats{}
	labelChan := make(chan filesystemLabels)
	statChan := make(chan filesystemStats)
	wg := sync.WaitGroup{}

	// Use 4 workers for stat calls
	workerCount := 4
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for labels := range labelChan {
				statChan <- c.processStat(labels)
			}
		}()
	}

	go func() {
		for _, labels := range mps {
			if c.ignoredMountPoints.MatchString(labels.mountPoint) {
				c.logger.Debug("Ignoring mount point", "mountpoint", labels.mountPoint)
				continue
			}
			if c.ignoredFSTypes.MatchString(labels.fsType) {
				c.logger.Debug("Ignoring fs type", "type", labels.fsType)
				continue
			}

			labelChan <- labels
		}
		close(labelChan)
		wg.Wait()
		close(statChan)
	}()

	for stat := range statChan {
		stats = append(stats, stat)
	}
	return stats, nil
}

func (c *FilesystemCollector) processStat(labels filesystemLabels) filesystemStats {
	var ro float64
	for _, option := range strings.Split(labels.options, ",") {
		if option == "ro" {
			ro = 1
			break
		}
	}

	buf := new(unix.Statfs_t)
	err := unix.Statfs(labels.mountPoint, buf)

	if err != nil {
		labels.deviceError = err.Error()
		c.logger.Debug("Error on statfs() system call", "mountpoint", labels.mountPoint, "err", err)
		return filesystemStats{
			labels:      labels,
			deviceError: 1,
			ro:          ro,
		}
	}

	return filesystemStats{
		labels:    labels,
		size:      float64(buf.Blocks) * float64(buf.Bsize),
		free:      float64(buf.Bfree) * float64(buf.Bsize),
		avail:     float64(buf.Bavail) * float64(buf.Bsize),
		files:     float64(buf.Files),
		filesFree: float64(buf.Ffree),
		ro:        ro,
	}
}

func (c *FilesystemCollector) mountPointDetails() ([]filesystemLabels, error) {
	file, err := os.Open("/proc/1/mountinfo")
	if errors.Is(err, os.ErrNotExist) {
		// Fallback to `/proc/self/mountinfo` if `/proc/1/mountinfo` is missing due hidepid.
		c.logger.Debug("Reading root mounts failed, falling back to self mounts", "err", err)
		file, err = os.Open("/proc/self/mountinfo")
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return c.parseFilesystemLabels(file)
}

func (c *FilesystemCollector) parseFilesystemLabels(r io.Reader) ([]filesystemLabels, error) {
	var filesystems []filesystemLabels

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())

		if len(parts) < 10 {
			continue
		}

		// Find the separator field
		separatorIndex := -1
		for i, part := range parts {
			if part == "-" {
				separatorIndex = i
				break
			}
		}

		if separatorIndex == -1 || separatorIndex+3 >= len(parts) {
			continue
		}

		device := parts[separatorIndex+2]
		mountPoint := parts[4]
		fsType := parts[separatorIndex+1]
		options := parts[5]
		major := parts[2][:strings.Index(parts[2], ":")]
		minor := parts[2][strings.Index(parts[2], ":")+1:]

		filesystems = append(filesystems, filesystemLabels{
			device:      device,
			mountPoint:  mountPoint,
			fsType:      fsType,
			options:     options,
			deviceError: "",
			major:       major,
			minor:       minor,
		})
	}

	return filesystems, scanner.Err()
}
// Part 2 commit for node_storage_exporter/internal/metrics/filesystem.go

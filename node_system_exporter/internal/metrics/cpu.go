package metrics

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"node_system_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
	"github.com/prometheus/procfs/sysfs"
)

const (
	cpuCollectorSubsystem = "cpu"
	jumpBackSeconds       = 3.0
)

func init() {
	exporter.Register(NewCPUCollector())
}

type CPUCollector struct {
	*baseMetrics
	procfs             procfs.FS
	sysfs              sysfs.FS
	cpu                *prometheus.Desc
	cpuInfo            *prometheus.Desc
	cpuFrequencyHz     *prometheus.Desc
	cpuFlagsInfo       *prometheus.Desc
	cpuBugsInfo        *prometheus.Desc
	cpuGuest           *prometheus.Desc
	cpuCoreThrottle    *prometheus.Desc
	cpuPackageThrottle *prometheus.Desc
	cpuIsolated        *prometheus.Desc
	cpuOnline          *prometheus.Desc
	cpuStats           map[int64]procfs.CPUStat
	cpuStatsMutex      sync.Mutex
	isolatedCpus       []uint16
	logger             *slog.Logger

	cpuFlagsIncludeRegexp *regexp.Regexp
	cpuBugsIncludeRegexp  *regexp.Regexp
}

func NewCPUCollector() *CPUCollector {
	logger := slog.Default()

	pfs, err := procfs.NewFS("/proc")
	if err != nil {
		logger.Error("failed to open procfs", "error", err)
		return nil
	}

	sfs, err := sysfs.NewFS("/sys")
	if err != nil {
		logger.Error("failed to open sysfs", "error", err)
		return nil
	}

	isolcpus, err := sfs.IsolatedCPUs()
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Error("Unable to get isolated cpus", "error", err)
		}
	}

	return &CPUCollector{
		procfs: pfs,
		sysfs:  sfs,
		cpu: prometheus.NewDesc(
			prometheus.BuildFQName("node", cpuCollectorSubsystem, "seconds_total"),
			"Seconds the CPUs spent in each mode.",
			[]string{"cpu", "mode"}, nil,
		),
		cpuInfo: prometheus.NewDesc(
			prometheus.BuildFQName("node", cpuCollectorSubsystem, "info"),
			"CPU information from /proc/cpuinfo.",
			[]string{"package", "core", "cpu", "vendor", "family", "model", "model_name", "microcode", "stepping", "cachesize"}, nil,
		),
		cpuFrequencyHz: prometheus.NewDesc(
			prometheus.BuildFQName("node", cpuCollectorSubsystem, "frequency_hertz"),
			"CPU frequency in hertz from /proc/cpuinfo.",
			[]string{"package", "core", "cpu"}, nil,
		),
		cpuFlagsInfo: prometheus.NewDesc(
			prometheus.BuildFQName("node", cpuCollectorSubsystem, "flag_info"),
			"The `flags` field of CPU information from /proc/cpuinfo taken from the first core.",
			[]string{"flag"}, nil,
		),
		cpuBugsInfo: prometheus.NewDesc(
			prometheus.BuildFQName("node", cpuCollectorSubsystem, "bug_info"),
			"The `bugs` field of CPU information from /proc/cpuinfo taken from the first core.",
			[]string{"bug"}, nil,
		),
		cpuGuest: prometheus.NewDesc(
			prometheus.BuildFQName("node", cpuCollectorSubsystem, "guest_seconds_total"),
			"Seconds the CPUs spent in guests (VMs) for each mode.",
			[]string{"cpu", "mode"}, nil,
		),
		cpuCoreThrottle: prometheus.NewDesc(
			prometheus.BuildFQName("node", cpuCollectorSubsystem, "core_throttles_total"),
			"Number of times this CPU core has been throttled.",
			[]string{"package", "core"}, nil,
		),
		cpuPackageThrottle: prometheus.NewDesc(
			prometheus.BuildFQName("node", cpuCollectorSubsystem, "package_throttles_total"),
			"Number of times this CPU package has been throttled.",
			[]string{"package"}, nil,
		),
		cpuIsolated: prometheus.NewDesc(
			prometheus.BuildFQName("node", cpuCollectorSubsystem, "isolated"),
			"Whether each core is isolated, information from /sys/devices/system/cpu/isolated.",
			[]string{"cpu"}, nil,
		),
		cpuOnline: prometheus.NewDesc(
			prometheus.BuildFQName("node", cpuCollectorSubsystem, "online"),
			"CPUs that are online and being scheduled.",
			[]string{"cpu"}, nil,
		),
		logger:       logger,
		isolatedCpus: isolcpus,
		cpuStats:     make(map[int64]procfs.CPUStat),
	}
}

func (c *CPUCollector) Collect(ch chan<- prometheus.Metric) {
	if err := c.updateStat(ch); err != nil {
		c.logger.Error("Error updating CPU statistics", "error", err)
		return
	}

	if c.isolatedCpus != nil {
		c.updateIsolated(ch)
	}

	if err := c.updateThermalThrottle(ch); err != nil {
		c.logger.Error("Error updating thermal throttle", "error", err)
	}

	if err := c.updateOnline(ch); err != nil {
		c.logger.Error("Error updating online status", "error", err)
	}
}

func (c *CPUCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.cpu
	ch <- c.cpuInfo
	ch <- c.cpuFrequencyHz
	ch <- c.cpuFlagsInfo
	ch <- c.cpuBugsInfo
	ch <- c.cpuGuest
	ch <- c.cpuCoreThrottle
	ch <- c.cpuPackageThrottle
	ch <- c.cpuIsolated
	ch <- c.cpuOnline
}

func (c *CPUCollector) updateStat(ch chan<- prometheus.Metric) error {
	stats, err := c.procfs.Stat()
	if err != nil {
		return fmt.Errorf("failed to get CPU stats: %w", err)
	}

	c.cpuStatsMutex.Lock()
	defer c.cpuStatsMutex.Unlock()

	newStats := make(map[int64]procfs.CPUStat)
	for i, cpuStat := range stats.CPU {
		newStats[int64(i)] = cpuStat
	}

	c.updateCPUStats(newStats)

	for i, cpuStat := range stats.CPU {
		cpuNum := strconv.Itoa(int(i))

		ch <- prometheus.MustNewConstMetric(c.cpu, prometheus.CounterValue, cpuStat.User, cpuNum, "user")
		ch <- prometheus.MustNewConstMetric(c.cpu, prometheus.CounterValue, cpuStat.Nice, cpuNum, "nice")
		ch <- prometheus.MustNewConstMetric(c.cpu, prometheus.CounterValue, cpuStat.System, cpuNum, "system")
		ch <- prometheus.MustNewConstMetric(c.cpu, prometheus.CounterValue, cpuStat.Idle, cpuNum, "idle")
		ch <- prometheus.MustNewConstMetric(c.cpu, prometheus.CounterValue, cpuStat.Iowait, cpuNum, "iowait")
		ch <- prometheus.MustNewConstMetric(c.cpu, prometheus.CounterValue, cpuStat.IRQ, cpuNum, "irq")
		ch <- prometheus.MustNewConstMetric(c.cpu, prometheus.CounterValue, cpuStat.SoftIRQ, cpuNum, "softirq")
		ch <- prometheus.MustNewConstMetric(c.cpu, prometheus.CounterValue, cpuStat.Steal, cpuNum, "steal")

		// Guest CPU times
		ch <- prometheus.MustNewConstMetric(c.cpuGuest, prometheus.CounterValue, cpuStat.Guest, cpuNum, "user")
		ch <- prometheus.MustNewConstMetric(c.cpuGuest, prometheus.CounterValue, cpuStat.GuestNice, cpuNum, "nice")
	}

	return nil
}

func (c *CPUCollector) updateCPUStats(newStats map[int64]procfs.CPUStat) {
	// Convert new stats to the correct map type for storage
	convertedStats := make(map[int64]procfs.CPUStat)
	for cpu, newStat := range newStats {
		convertedStats[cpu] = newStat

		if oldStat, ok := c.cpuStats[cpu]; ok {
			// Check for idle counter jumping backwards
			if newStat.Idle < oldStat.Idle {
				timeDelta := newStat.Idle - oldStat.Idle
				if timeDelta < -jumpBackSeconds {
					c.logger.Debug("CPU Idle counter jumped backwards, possible hotplug event, resetting CPU stats", "cpu", cpu)
					delete(c.cpuStats, cpu)
				}
			}
		}
	}
	c.cpuStats = convertedStats
}

func (c *CPUCollector) updateThermalThrottle(ch chan<- prometheus.Metric) error {
	cpus, err := filepath.Glob("/sys/devices/system/cpu/cpu[0-9]*")
	if err != nil {
		return err
	}

	packageThrottles := make(map[uint64]uint64)
	packageCoreMap := make(map[uint64]uint64)

	for _, cpu := range cpus {
		// Parse core throttles
		corePath := filepath.Join(cpu, "thermal_throttle", "core_throttle_count")
		cleanCorePath := filepath.Clean(corePath)
		statDir := "/sys/devices/system/cpu"
		if !strings.HasPrefix(cleanCorePath, statDir) {
			c.logger.Debug("core throttle file must be located within sysfs", "cpu", cpu)
			continue
		}
		coreBytes, err := os.ReadFile(corePath)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			c.logger.Debug("Failed to read core throttle count", "path", corePath, "error", err)
			continue
		}

		coreThrottle, err := strconv.ParseUint(string(coreBytes), 10, 64)
		if err != nil {
			c.logger.Debug("Failed to parse core throttle count", "value", string(coreBytes), "error", err)
			continue
		}

		// Parse package throttles
		packagePath := filepath.Join(cpu, "thermal_throttle", "package_throttle_count")
		cleanPackagePath := filepath.Clean(packagePath)
		// statDir := "/sys/devices/system/cpu"
		if !strings.HasPrefix(cleanPackagePath, statDir) {
			c.logger.Debug("package throttle file must be located within sysfs", "cpu", cpu)
			continue
		}
		packageBytes, err := os.ReadFile(packagePath)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			c.logger.Debug("Failed to read package throttle count", "path", packagePath, "error", err)
			continue
		}

		packageThrottle, err := strconv.ParseUint(string(packageBytes), 10, 64)
		if err != nil {
			c.logger.Debug("Failed to parse package throttle count", "value", string(packageBytes), "error", err)
			continue
		}

		// Get topology info
		physicalPackageID, err := c.getTopologyValue(cpu, "physical_package_id")
		if err != nil {
			c.logger.Debug("Failed to read physical package ID", "cpu", cpu, "error", err)
			continue
		}

		coreID, err := c.getTopologyValue(cpu, "core_id")
		if err != nil {
			c.logger.Debug("Failed to read core ID", "cpu", cpu, "error", err)
			continue
		}

		ch <- prometheus.MustNewConstMetric(c.cpuCoreThrottle, prometheus.CounterValue, float64(coreThrottle), strconv.FormatUint(physicalPackageID, 10), strconv.FormatUint(coreID, 10))

		if _, exists := packageThrottles[physicalPackageID]; !exists {
			packageThrottles[physicalPackageID] = packageThrottle
			packageCoreMap[physicalPackageID] = coreID
		}
	}

	for pkg, throttle := range packageThrottles {
		ch <- prometheus.MustNewConstMetric(c.cpuPackageThrottle, prometheus.CounterValue, float64(throttle), strconv.FormatUint(pkg, 10))
	}

	return nil
}

func (c *CPUCollector) getTopologyValue(cpuPath, topology string) (uint64, error) {
	topologyPath := filepath.Join(cpuPath, "topology", topology)
	cleanTopologyPath := filepath.Clean(topologyPath)
	statDir := "/sys/devices/system/cpu"
	if !strings.HasPrefix(cleanTopologyPath, statDir) {
		return 0, fmt.Errorf("topology file must be located within sysfs")
	}
	topologyBytes, err := os.ReadFile(cleanTopologyPath)
	if err != nil {
		return 0, err
	}

	return strconv.ParseUint(string(topologyBytes), 10, 64)
}

func (c *CPUCollector) updateIsolated(ch chan<- prometheus.Metric) {
	if c.isolatedCpus == nil {
		return
	}

	cpus, err := filepath.Glob("/sys/devices/system/cpu/cpu[0-9]*")
	if err != nil {
		c.logger.Debug("Failed to glob CPU directories", "error", err)
		return
	}

	for _, cpu := range cpus {
		cpuName := filepath.Base(cpu)
		cpuNum := cpuName[3:] // Remove "cpu" prefix

		cpuID, err := strconv.ParseUint(cpuNum, 10, 16)
		if err != nil {
			c.logger.Debug("Failed to parse CPU number", "cpu", cpuName, "error", err)
			continue
		}

		isolated := 0.0
		for _, isolatedCPU := range c.isolatedCpus {
			if uint16(cpuID) == isolatedCPU {
				isolated = 1.0
				break
			}
		}

		ch <- prometheus.MustNewConstMetric(c.cpuIsolated, prometheus.GaugeValue, isolated, cpuNum)
	}
}

func (c *CPUCollector) updateOnline(ch chan<- prometheus.Metric) error {
	cpus, err := filepath.Glob("/sys/devices/system/cpu/cpu[0-9]*")
	if err != nil {
		return err
	}

	for _, cpu := range cpus {
		cpuName := filepath.Base(cpu)
		cpuNum := cpuName[3:] // Remove "cpu" prefix

		// Check if CPU0 (always online) or read online status
		online := 1.0
		if cpuNum != "0" {
			onlinePath := filepath.Join(cpu, "online")
			cleanPath := filepath.Clean(onlinePath)
			statDir := "/sys/devices/system/cpu"
			if !strings.HasPrefix(cleanPath, statDir) {
				c.logger.Debug("online file must be located within sysfs", "cpu", cpuName)
				continue
			}
			onlineBytes, err := os.ReadFile(cleanPath)
			if os.IsNotExist(err) {
				// If online file doesn't exist, assume it's online
				online = 1.0
			} else if err != nil {
				c.logger.Debug("Failed to read CPU online status", "cpu", cpuName, "error", err)
				continue
			} else {
				onlineStr := string(onlineBytes)
				if onlineStr == "0\n" {
					online = 0.0
				}
			}
		}

		ch <- prometheus.MustNewConstMetric(c.cpuOnline, prometheus.GaugeValue, online, cpuNum)
	}

	return nil
}

package metrics

import (
	"log/slog"
	"strings"

	"node_system_exporter/internal/exporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs/sysfs"
)

func init() {
	exporter.Register(NewCPUFreqCollector())
}

type CPUFreqCollector struct {
	*baseMetrics
	fs                       sysfs.FS
	cpuFreqHertz            *prometheus.Desc
	cpuFreqMin              *prometheus.Desc
	cpuFreqMax              *prometheus.Desc
	cpuFreqScalingFreq      *prometheus.Desc
	cpuFreqScalingFreqMin   *prometheus.Desc
	cpuFreqScalingFreqMax   *prometheus.Desc
	cpuFreqScalingGovernor  *prometheus.Desc
	logger                  *slog.Logger
}

func NewCPUFreqCollector() *CPUFreqCollector {
	logger := slog.Default()
	
	fs, err := sysfs.NewFS("/sys")
	if err != nil {
		logger.Error("failed to open sysfs", "error", err)
		return nil
	}

	return &CPUFreqCollector{
		fs: fs,
		cpuFreqHertz: prometheus.NewDesc(
			prometheus.BuildFQName("node", "cpu", "frequency_hertz"),
			"Current cpu thread frequency in hertz.",
			[]string{"cpu"}, nil,
		),
		cpuFreqMin: prometheus.NewDesc(
			prometheus.BuildFQName("node", "cpu", "frequency_min_hertz"),
			"Minimum cpu thread frequency in hertz.",
			[]string{"cpu"}, nil,
		),
		cpuFreqMax: prometheus.NewDesc(
			prometheus.BuildFQName("node", "cpu", "frequency_max_hertz"),
			"Maximum cpu thread frequency in hertz.",
			[]string{"cpu"}, nil,
		),
		cpuFreqScalingFreq: prometheus.NewDesc(
			prometheus.BuildFQName("node", "cpu", "scaling_frequency_hertz"),
			"Current scaled CPU thread frequency in hertz.",
			[]string{"cpu"}, nil,
		),
		cpuFreqScalingFreqMin: prometheus.NewDesc(
			prometheus.BuildFQName("node", "cpu", "scaling_frequency_min_hertz"),
			"Minimum scaled CPU thread frequency in hertz.",
			[]string{"cpu"}, nil,
		),
		cpuFreqScalingFreqMax: prometheus.NewDesc(
			prometheus.BuildFQName("node", "cpu", "scaling_frequency_max_hertz"),
			"Maximum scaled CPU thread frequency in hertz.",
			[]string{"cpu"}, nil,
		),
		cpuFreqScalingGovernor: prometheus.NewDesc(
			prometheus.BuildFQName("node", "cpu", "scaling_governor"),
			"Current enabled CPU frequency governor.",
			[]string{"cpu", "governor"}, nil,
		),
		logger: logger,
	}
}

func (c *CPUFreqCollector) Collect(ch chan<- prometheus.Metric) {
	cpuFreqs, err := c.fs.SystemCpufreq()
	if err != nil {
		c.logger.Debug("Error getting CPU frequency stats", "error", err)
		return
	}

	// sysfs cpufreq values are kHz, thus multiply by 1000 to export base units (hz).
	// See https://www.kernel.org/doc/Documentation/cpu-freq/user-guide.txt
	for _, stats := range cpuFreqs {
		if stats.CpuinfoCurrentFrequency != nil {
			ch <- prometheus.MustNewConstMetric(
				c.cpuFreqHertz,
				prometheus.GaugeValue,
				float64(*stats.CpuinfoCurrentFrequency)*1000.0,
				stats.Name,
			)
		}
		if stats.CpuinfoMinimumFrequency != nil {
			ch <- prometheus.MustNewConstMetric(
				c.cpuFreqMin,
				prometheus.GaugeValue,
				float64(*stats.CpuinfoMinimumFrequency)*1000.0,
				stats.Name,
			)
		}
		if stats.CpuinfoMaximumFrequency != nil {
			ch <- prometheus.MustNewConstMetric(
				c.cpuFreqMax,
				prometheus.GaugeValue,
				float64(*stats.CpuinfoMaximumFrequency)*1000.0,
				stats.Name,
			)
		}
		if stats.ScalingCurrentFrequency != nil {
			ch <- prometheus.MustNewConstMetric(
				c.cpuFreqScalingFreq,
				prometheus.GaugeValue,
				float64(*stats.ScalingCurrentFrequency)*1000.0,
				stats.Name,
			)
		}
		if stats.ScalingMinimumFrequency != nil {
			ch <- prometheus.MustNewConstMetric(
				c.cpuFreqScalingFreqMin,
				prometheus.GaugeValue,
				float64(*stats.ScalingMinimumFrequency)*1000.0,
				stats.Name,
			)
		}
		if stats.ScalingMaximumFrequency != nil {
			ch <- prometheus.MustNewConstMetric(
				c.cpuFreqScalingFreqMax,
				prometheus.GaugeValue,
				float64(*stats.ScalingMaximumFrequency)*1000.0,
				stats.Name,
			)
		}
		if stats.Governor != "" {
			availableGovernors := strings.Split(stats.AvailableGovernors, " ")
			for _, g := range availableGovernors {
				state := 0
				if g == stats.Governor {
					state = 1
				}
				ch <- prometheus.MustNewConstMetric(
					c.cpuFreqScalingGovernor,
					prometheus.GaugeValue,
					float64(state),
					stats.Name,
					g,
				)
			}
		}
	}
}

func (c *CPUFreqCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.cpuFreqHertz
	ch <- c.cpuFreqMin
	ch <- c.cpuFreqMax
	ch <- c.cpuFreqScalingFreq
	ch <- c.cpuFreqScalingFreqMin
	ch <- c.cpuFreqScalingFreqMax
	ch <- c.cpuFreqScalingGovernor
} 
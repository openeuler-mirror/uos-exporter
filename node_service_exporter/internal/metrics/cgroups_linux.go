//go:build !nostat
// +build !nostat

package metrics

import (
	"fmt"
	"log/slog"
	"node_service_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
)

func init() {
	exporter.Register(NewCgroupSummaryCollectorWrapper())
}

const cgroupsCollectorSubsystem = "cgroups"

// CgroupSummaryCollectorWrapper wraps the old collector to work with new framework
type CgroupSummaryCollectorWrapper struct {
	collector *CgroupSummaryCollector
}

func NewCgroupSummaryCollectorWrapper() *CgroupSummaryCollectorWrapper {
	collector, err := NewCgroupSummaryCollector(nil)
	if err != nil {
		return nil
	}
	return &CgroupSummaryCollectorWrapper{
		collector: collector,
	}
}

func (c *CgroupSummaryCollectorWrapper) Collect(ch chan<- prometheus.Metric) {
	if c.collector != nil {
		if err := c.collector.Collect(ch); err != nil {
			fmt.Printf("Error collecting metrics: %v\n", err)
		}
	}
}

// CgroupsMetric represents cgroups metrics
type CgroupsMetric struct {
	*baseMetrics
}

// newCgroupsMetric creates a new cgroups metric
func newCgroupsMetric(name, help string) *CgroupsMetric {
	return &CgroupsMetric{
		baseMetrics: &baseMetrics{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, cgroupsCollectorSubsystem, name),
				help,
				[]string{"subsys_name"},
				nil,
			),
		},
	}
}

// Collect implements the Collect method for CgroupsMetric
func (c *CgroupsMetric) Collect(ch chan<- prometheus.Metric, value float64, subsysName string) {
	ch <- prometheus.MustNewConstMetric(
		c.desc,
		prometheus.GaugeValue,
		value,
		subsysName,
	)
}

// CgroupSummaryCollector collects cgroups summary metrics
type CgroupSummaryCollector struct {
	fs      procfs.FS
	cgroups *CgroupsMetric
	enabled *CgroupsMetric
	logger  *slog.Logger
}

// NewCgroupSummaryCollector creates a new cgroup summary collector
func NewCgroupSummaryCollector(logger *slog.Logger) (*CgroupSummaryCollector, error) {
	if logger == nil {
		logger = slog.Default()
	}

	fs, err := procfs.NewFS("/proc")
	if err != nil {
		return nil, fmt.Errorf("failed to open procfs: %w", err)
	}

	return &CgroupSummaryCollector{
		fs: fs,
		cgroups: newCgroupsMetric(
			"cgroups",
			"Current cgroup number of the subsystem.",
		),
		enabled: newCgroupsMetric(
			"enabled",
			"Current cgroup number of the subsystem.",
		),
		logger: logger,
	}, nil
}

// Collect implements the Collector interface
func (c *CgroupSummaryCollector) Collect(ch chan<- prometheus.Metric) error {
	if c == nil {
		return fmt.Errorf("CgroupSummaryCollector is nil")
	}

	cgroupSummarys, err := c.fs.CgroupSummarys()
	if err != nil {
		return fmt.Errorf("failed to get cgroup summaries: %w", err)
	}

	for _, cs := range cgroupSummarys {
		if cs.SubsysName == "" {
			c.logger.Debug("skipping cgroup summary with empty subsys name")
			continue
		}
		
		c.cgroups.Collect(ch, float64(cs.Cgroups), cs.SubsysName)
		c.enabled.Collect(ch, float64(cs.Enabled), cs.SubsysName)
	}

	return nil
} 

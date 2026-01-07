//go:build !notime
// +build !notime

package metrics

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"
	"node_service_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs/sysfs"
)

func init() {
	exporter.Register(NewTimeCollectorWrapper())
}

// typedDesc represents a metric descriptor with its value type
type typedDesc struct {
	desc      *prometheus.Desc
	valueType prometheus.ValueType
}

// mustNewConstMetric creates a new constant metric
func (d *typedDesc) mustNewConstMetric(value float64, labels ...string) prometheus.Metric {
	return prometheus.MustNewConstMetric(d.desc, d.valueType, value, labels...)
}

// TimeCollectorWrapper wraps the old collector to work with new framework
type TimeCollectorWrapper struct {
	collector *TimeCollector
}

func NewTimeCollectorWrapper() *TimeCollectorWrapper {
	collector, err := NewTimeCollector(nil)
	if err != nil {
		return nil
	}
	return &TimeCollectorWrapper{
		collector: collector,
	}
}

func (t *TimeCollectorWrapper) Collect(ch chan<- prometheus.Metric) {
	if t.collector != nil {
		if err := t.collector.Collect(ch); err != nil {
			fmt.Printf("Error collecting metrics: %v\n", err)
		}
	}
}

// TimeCollector collects time-related metrics
type TimeCollector struct {
	now                   typedDesc
	zone                  typedDesc
	clocksourcesAvailable typedDesc
	clocksourceCurrent    typedDesc
	logger                *slog.Logger
}

// NewTimeCollector creates a new time collector
func NewTimeCollector(logger *slog.Logger) (*TimeCollector, error) {
	if logger == nil {
		logger = slog.Default()
	}

	const subsystem = "time"
	return &TimeCollector{
		now: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, subsystem, "seconds"),
				"System time in seconds since epoch (1970).",
				nil,
				nil,
			),
			valueType: prometheus.GaugeValue,
		},
		zone: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, subsystem, "zone_offset_seconds"),
				"System time zone offset in seconds.",
				[]string{"time_zone"},
				nil,
			),
			valueType: prometheus.GaugeValue,
		},
		clocksourcesAvailable: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, subsystem, "clocksource_available_info"),
				"Available clocksources read from '/sys/devices/system/clocksource'.",
				[]string{"device", "clocksource"},
				nil,
			),
			valueType: prometheus.GaugeValue,
		},
		clocksourceCurrent: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, subsystem, "clocksource_current_info"),
				"Current clocksource read from '/sys/devices/system/clocksource'.",
				[]string{"device", "clocksource"},
				nil,
			),
			valueType: prometheus.GaugeValue,
		},
		logger: logger,
	}, nil
}

// Collect implements the Collector interface
func (c *TimeCollector) Collect(ch chan<- prometheus.Metric) error {
	if c == nil {
		return fmt.Errorf("TimeCollector is nil")
	}

	now := time.Now()
	nowSec := float64(now.UnixNano()) / 1e9
	zone, zoneOffset := now.Zone()

	c.logger.Debug("Return time", "now", nowSec)
	ch <- c.now.mustNewConstMetric(nowSec)
	c.logger.Debug("Zone offset", "offset", zoneOffset, "time_zone", zone)
	ch <- c.zone.mustNewConstMetric(float64(zoneOffset), zone)

	return c.updateClocksources(ch)
}

// updateClocksources updates clocksource metrics (Linux-specific)
func (c *TimeCollector) updateClocksources(ch chan<- prometheus.Metric) error {
	fs, err := sysfs.NewFS("/sys")
	if err != nil {
		c.logger.Debug("failed to open sysfs, skipping clocksource metrics", "err", err)
		return nil
	}

	clocksources, err := fs.ClockSources()
	if err != nil {
		c.logger.Debug("couldn't get clocksources, skipping", "err", err)
		return nil
	}

	c.logger.Debug("clocksources found", "clocksources", fmt.Sprintf("%v", clocksources))

	for i, clocksource := range clocksources {
		is := strconv.Itoa(i)
		for _, cs := range clocksource.Available {
			if cs == "" {
				continue
			}
			ch <- c.clocksourcesAvailable.mustNewConstMetric(1.0, is, cs)
		}
		if clocksource.Current != "" {
			ch <- c.clocksourceCurrent.mustNewConstMetric(1.0, is, clocksource.Current)
		}
	}

	return nil
}
// Part 2 commit for node_service_exporter/internal/metrics/time.go

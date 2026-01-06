package metrics

import (
	"errors"
	"fmt"
	"node_hardware_exporter/internal/exporter"
	"os"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs/sysfs"
)

func init() {
	exporter.Register(NewRaplCollector())
}

type RaplCollector struct {
	*baseMetrics
	fs               sysfs.FS
	joulesMetricDesc *prometheus.Desc
	enableZoneLabel  bool
}

func NewRaplCollector() *RaplCollector {
	fs, err := sysfs.NewFS("/sys")
	if err != nil {
		return &RaplCollector{
			baseMetrics: NewMetrics("node_rapl_collector", "RAPL collector metrics", []string{}),
		}
	}

	joulesMetricDesc := prometheus.NewDesc(
		"node_rapl_joules_total",
		"Current RAPL value in joules",
		[]string{"index", "path", "rapl_zone"}, nil,
	)

	return &RaplCollector{
		baseMetrics:      NewMetrics("node_rapl_collector", "RAPL collector metrics", []string{}),
		fs:               fs,
		joulesMetricDesc: joulesMetricDesc,
		enableZoneLabel:  true, // 默认启用zone标签
	}
}

func (c *RaplCollector) Collect(ch chan<- prometheus.Metric) {
	if c.fs == (sysfs.FS{}) {
		fs, err := sysfs.NewFS("/sys")
		if err != nil {
			return
		}
		c.fs = fs
	}

	// nil zones are fine when platform doesn't have powercap files present.
	zones, err := sysfs.GetRaplZones(c.fs)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission) {
			return
		}
		return
	}

	for _, rz := range zones {
		microJoules, err := rz.GetEnergyMicrojoules()
		if err != nil {
			if errors.Is(err, os.ErrPermission) {
				return
			}
			return
		}

		joules := float64(microJoules) / 1000000.0

		if c.enableZoneLabel {
			ch <- c.joulesMetricWithZoneLabel(rz, joules)
		} else {
			ch <- c.joulesMetric(rz, joules)
		}
	}
}

func (c *RaplCollector) joulesMetric(z sysfs.RaplZone, v float64) prometheus.Metric {
	index := strconv.Itoa(z.Index)
	sanitizedName := sanitizeMetricName(z.Name)
	descriptor := prometheus.NewDesc(
		fmt.Sprintf("node_rapl_%s_joules_total", sanitizedName),
		fmt.Sprintf("Current RAPL %s value in joules", z.Name),
		[]string{"index", "path"}, nil,
	)

	return prometheus.MustNewConstMetric(
		descriptor,
		prometheus.CounterValue,
		v,
		index,
		z.Path,
	)
}

func (c *RaplCollector) joulesMetricWithZoneLabel(z sysfs.RaplZone, v float64) prometheus.Metric {
	index := strconv.Itoa(z.Index)

	return prometheus.MustNewConstMetric(
		c.joulesMetricDesc,
		prometheus.CounterValue,
		v,
		index,
		z.Path,
		z.Name,
	)
}

// 辅助函数，将非字母数字字符替换为下划线
func sanitizeMetricName(name string) string {
	result := ""
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			result += string(c)
		} else {
			result += "_"
		}
	}
	return result
} 
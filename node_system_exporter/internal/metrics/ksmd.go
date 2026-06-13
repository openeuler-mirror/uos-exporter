package metrics

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"node_system_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	exporter.Register(NewKSMDCollector())
}

var (
	ksmdFiles = []string{"full_scans", "merge_across_nodes", "pages_shared", "pages_sharing",
		"pages_to_scan", "pages_unshared", "pages_volatile", "run", "sleep_millisecs"}
)

type KSMDCollector struct {
	*baseMetrics
	metricDescs map[string]*prometheus.Desc
	logger      *slog.Logger
}

func NewKSMDCollector() *KSMDCollector {
	logger := slog.Default()
	subsystem := "ksmd"
	descs := make(map[string]*prometheus.Desc)

	for _, n := range ksmdFiles {
		descs[n] = prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, getCanonicalMetricName(n)),
			fmt.Sprintf("ksmd '%s' file.", n), nil, nil)
	}

	return &KSMDCollector{
		metricDescs: descs,
		logger:      logger,
	}
}

func getCanonicalMetricName(filename string) string {
	switch filename {
	case "full_scans":
		return filename + "_total"
	case "sleep_millisecs":
		return "sleep_seconds"
	default:
		return filename
	}
}

func (c *KSMDCollector) Collect(ch chan<- prometheus.Metric) {
	for _, n := range ksmdFiles {
		val, err := c.readUintFromFile(filepath.Join("/sys/kernel/mm/ksm", n))
		if err != nil {
			c.logger.Debug("Error reading ksmd file", "file", n, "error", err)
			continue
		}

		t := prometheus.GaugeValue
		v := float64(val)
		switch n {
		case "full_scans":
			t = prometheus.CounterValue
		case "sleep_millisecs":
			v /= 1000
		}
		ch <- prometheus.MustNewConstMetric(c.metricDescs[n], t, v)
	}
}

func (c *KSMDCollector) readUintFromFile(path string) (uint64, error) {
	cleanPath := filepath.Clean(path)
	statDir := "/sys/kernel/mm/ksm"
	if !strings.HasPrefix(cleanPath, statDir) {
		return 0, fmt.Errorf("stat file must be located within %s", statDir)
	}
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return 0, err
	}

	value, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, err
	}

	return value, nil
}

func (c *KSMDCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, desc := range c.metricDescs {
		ch <- desc
	}
}

package metrics

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"node_system_exporter/internal/exporter"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	exporter.Register(NewLoadAvgCollector())
}

type LoadAvgCollector struct {
	*baseMetrics
	load1  *prometheus.Desc
	load5  *prometheus.Desc
	load15 *prometheus.Desc
	logger *slog.Logger
}

func NewLoadAvgCollector() *LoadAvgCollector {
	return &LoadAvgCollector{
		load1: prometheus.NewDesc(
			prometheus.BuildFQName("node", "", "load1"),
			"1m load average.",
			nil, nil,
		),
		load5: prometheus.NewDesc(
			prometheus.BuildFQName("node", "", "load5"),
			"5m load average.",
			nil, nil,
		),
		load15: prometheus.NewDesc(
			prometheus.BuildFQName("node", "", "load15"),
			"15m load average.",
			nil, nil,
		),
		logger: slog.Default(),
	}
}

func (c *LoadAvgCollector) Collect(ch chan<- prometheus.Metric) {
	loads, err := c.getLoad()
	if err != nil {
		c.logger.Error("Error getting load average", "error", err)
		return
	}

	if len(loads) >= 3 {
		ch <- prometheus.MustNewConstMetric(c.load1, prometheus.GaugeValue, loads[0])
		ch <- prometheus.MustNewConstMetric(c.load5, prometheus.GaugeValue, loads[1])
		ch <- prometheus.MustNewConstMetric(c.load15, prometheus.GaugeValue, loads[2])
	}
}

func (c *LoadAvgCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.load1
	ch <- c.load5
	ch <- c.load15
}

// Read loadavg from /proc.
func (c *LoadAvgCollector) getLoad() (loads []float64, err error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return nil, err
	}
	loads, err = c.parseLoad(string(data))
	if err != nil {
		return nil, err
	}
	return loads, nil
}

// Parse /proc loadavg and return 1m, 5m and 15m.
func (c *LoadAvgCollector) parseLoad(data string) (loads []float64, err error) {
	loads = make([]float64, 3)
	parts := strings.Fields(data)
	if len(parts) < 3 {
		return nil, fmt.Errorf("unexpected content in /proc/loadavg")
	}
	for i, load := range parts[0:3] {
		loads[i], err = strconv.ParseFloat(load, 64)
		if err != nil {
			return nil, fmt.Errorf("could not parse load '%s': %w", load, err)
		}
	}
	return loads, nil
} 
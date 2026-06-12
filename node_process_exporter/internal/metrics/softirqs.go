package metrics

import (
	"fmt"
	"log/slog"
	"strconv"

	"node_process_exporter/internal/exporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
)

var (
	softirqLabelNames = []string{"cpu", "type"}
)

func init() {
	exporter.Register(NewSoftirqsCollector())
}

type softirqsCollector struct {
	*baseMetrics
	fs     procfs.FS
	desc   typedDesc
	logger *slog.Logger
}

func NewSoftirqsCollector() *softirqsCollector {
	logger := slog.Default()

	desc := typedDesc{prometheus.NewDesc(
		"node_softirqs_functions_total",
		"Softirq counts per CPU.",
		softirqLabelNames, nil,
	), prometheus.CounterValue}

	fs, err := procfs.NewFS("/proc")
	if err != nil {
		logger.Error("failed to open procfs", "error", err)
	}

	return &softirqsCollector{
		baseMetrics: NewMetrics("node_softirqs_collect_errors_total", "Number of errors that occurred during softirqs collection", []string{}),
		fs:          fs,
		desc:        desc,
		logger:      logger,
	}
}

func (c *softirqsCollector) Collect(ch chan<- prometheus.Metric) {
	if err := c.Update(ch); err != nil {
		c.logger.Error("Error updating softirqs metrics", "error", err)
		ch <- prometheus.MustNewConstMetric(c.baseMetrics.desc, prometheus.CounterValue, 1)
	}
}

func (c *softirqsCollector) Update(ch chan<- prometheus.Metric) error {
	softirqs, err := c.fs.Softirqs()
	if err != nil {
		return fmt.Errorf("couldn't get softirqs: %w", err)
	}

	for cpuNo, value := range softirqs.Hi {
		ch <- c.desc.mustNewConstMetric(float64(value), strconv.Itoa(cpuNo), "HI")
	}
	for cpuNo, value := range softirqs.Timer {
		ch <- c.desc.mustNewConstMetric(float64(value), strconv.Itoa(cpuNo), "TIMER")
	}
	for cpuNo, value := range softirqs.NetTx {
		ch <- c.desc.mustNewConstMetric(float64(value), strconv.Itoa(cpuNo), "NET_TX")
	}
	for cpuNo, value := range softirqs.NetRx {
		ch <- c.desc.mustNewConstMetric(float64(value), strconv.Itoa(cpuNo), "NET_RX")
	}
	for cpuNo, value := range softirqs.Block {
		ch <- c.desc.mustNewConstMetric(float64(value), strconv.Itoa(cpuNo), "BLOCK")
	}
	for cpuNo, value := range softirqs.IRQPoll {
		ch <- c.desc.mustNewConstMetric(float64(value), strconv.Itoa(cpuNo), "IRQ_POLL")
	}
	for cpuNo, value := range softirqs.Tasklet {
		ch <- c.desc.mustNewConstMetric(float64(value), strconv.Itoa(cpuNo), "TASKLET")
	}
	for cpuNo, value := range softirqs.Sched {
		ch <- c.desc.mustNewConstMetric(float64(value), strconv.Itoa(cpuNo), "SCHED")
	}
	for cpuNo, value := range softirqs.HRTimer {
		ch <- c.desc.mustNewConstMetric(float64(value), strconv.Itoa(cpuNo), "HRTIMER")
	}
	for cpuNo, value := range softirqs.RCU {
		ch <- c.desc.mustNewConstMetric(float64(value), strconv.Itoa(cpuNo), "RCU")
	}

	return nil
} 
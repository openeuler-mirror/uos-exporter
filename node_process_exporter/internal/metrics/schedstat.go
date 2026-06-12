package metrics

import (
	"errors"
	"log/slog"
	"os"

	"node_process_exporter/internal/exporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
)

const nsPerSec = 1e9

var (
	runningSecondsTotal = prometheus.NewDesc(
		"node_schedstat_running_seconds_total",
		"Number of seconds CPU spent running a process.",
		[]string{"cpu"},
		nil,
	)

	waitingSecondsTotal = prometheus.NewDesc(
		"node_schedstat_waiting_seconds_total",
		"Number of seconds spent by processing waiting for this CPU.",
		[]string{"cpu"},
		nil,
	)

	timeslicesTotal = prometheus.NewDesc(
		"node_schedstat_timeslices_total",
		"Number of timeslices executed by CPU.",
		[]string{"cpu"},
		nil,
	)
)

func init() {
	exporter.Register(NewSchedstatCollector())
}

type schedstatCollector struct {
	*baseMetrics
	fs     procfs.FS
	logger *slog.Logger
}

func NewSchedstatCollector() *schedstatCollector {
	logger := slog.Default()

	fs, err := procfs.NewFS("/proc")
	if err != nil {
		logger.Error("failed to open procfs", "error", err)
	}

	return &schedstatCollector{
		baseMetrics: NewMetrics("node_schedstat_collect_errors_total", "Number of errors that occurred during schedstat collection", []string{}),
		fs:          fs,
		logger:      logger,
	}
}

func (c *schedstatCollector) Collect(ch chan<- prometheus.Metric) {
	if err := c.Update(ch); err != nil {
		c.logger.Error("Error updating schedstat metrics", "error", err)
		ch <- prometheus.MustNewConstMetric(c.baseMetrics.desc, prometheus.CounterValue, 1)
	}
}

func (c *schedstatCollector) Update(ch chan<- prometheus.Metric) error {
	stats, err := c.fs.Schedstat()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			c.logger.Debug("schedstat file does not exist")
			return nil // 不返回错误，类似ErrNoData的处理
		}
		return err
	}

	for _, cpu := range stats.CPUs {
		ch <- prometheus.MustNewConstMetric(
			runningSecondsTotal,
			prometheus.CounterValue,
			float64(cpu.RunningNanoseconds)/nsPerSec,
			cpu.CPUNum,
		)

		ch <- prometheus.MustNewConstMetric(
			waitingSecondsTotal,
			prometheus.CounterValue,
			float64(cpu.WaitingNanoseconds)/nsPerSec,
			cpu.CPUNum,
		)

		ch <- prometheus.MustNewConstMetric(
			timeslicesTotal,
			prometheus.CounterValue,
			float64(cpu.RunTimeslices),
			cpu.CPUNum,
		)
	}

	return nil
} 
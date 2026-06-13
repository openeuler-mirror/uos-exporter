package metrics

import (
	"log/slog"

	"node_system_exporter/internal/exporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
)

func init() {
	exporter.Register(NewStatCollector())
}

type StatCollector struct {
	*baseMetrics
	fs           procfs.FS
	intr         *prometheus.Desc
	ctxt         *prometheus.Desc
	forks        *prometheus.Desc
	btime        *prometheus.Desc
	procsRunning *prometheus.Desc
	procsBlocked *prometheus.Desc
	softIRQ      *prometheus.Desc
	logger       *slog.Logger
}

func NewStatCollector() *StatCollector {
	logger := slog.Default()
	
	fs, err := procfs.NewFS("/proc")
	if err != nil {
		logger.Error("failed to open procfs", "error", err)
		return nil
	}

	return &StatCollector{
		fs: fs,
		intr: prometheus.NewDesc(
			prometheus.BuildFQName("node", "", "intr_total"),
			"Total number of interrupts serviced.",
			nil, nil,
		),
		ctxt: prometheus.NewDesc(
			prometheus.BuildFQName("node", "", "context_switches_total"),
			"Total number of context switches.",
			nil, nil,
		),
		forks: prometheus.NewDesc(
			prometheus.BuildFQName("node", "", "forks_total"),
			"Total number of forks.",
			nil, nil,
		),
		btime: prometheus.NewDesc(
			prometheus.BuildFQName("node", "", "boot_time_seconds"),
			"Node boot time, in unixtime.",
			nil, nil,
		),
		procsRunning: prometheus.NewDesc(
			prometheus.BuildFQName("node", "", "procs_running"),
			"Number of processes in runnable state.",
			nil, nil,
		),
		procsBlocked: prometheus.NewDesc(
			prometheus.BuildFQName("node", "", "procs_blocked"),
			"Number of processes blocked waiting for I/O to complete.",
			nil, nil,
		),
		softIRQ: prometheus.NewDesc(
			prometheus.BuildFQName("node", "", "softirqs_total"),
			"Number of softirq calls.",
			[]string{"vector"}, nil,
		),
		logger: logger,
	}
}

func (c *StatCollector) Collect(ch chan<- prometheus.Metric) {
	stats, err := c.fs.Stat()
	if err != nil {
		c.logger.Error("Error getting system statistics", "error", err)
		return
	}

	ch <- prometheus.MustNewConstMetric(c.intr, prometheus.CounterValue, float64(stats.IRQTotal))
	ch <- prometheus.MustNewConstMetric(c.ctxt, prometheus.CounterValue, float64(stats.ContextSwitches))
	ch <- prometheus.MustNewConstMetric(c.forks, prometheus.CounterValue, float64(stats.ProcessCreated))

	ch <- prometheus.MustNewConstMetric(c.btime, prometheus.GaugeValue, float64(stats.BootTime))

	ch <- prometheus.MustNewConstMetric(c.procsRunning, prometheus.GaugeValue, float64(stats.ProcessesRunning))
	ch <- prometheus.MustNewConstMetric(c.procsBlocked, prometheus.GaugeValue, float64(stats.ProcessesBlocked))

	// Export softirq calls per vector
	si := stats.SoftIRQ

	for _, vec := range []struct {
		name  string
		value uint64
	}{
		{name: "hi", value: si.Hi},
		{name: "timer", value: si.Timer},
		{name: "net_tx", value: si.NetTx},
		{name: "net_rx", value: si.NetRx},
		{name: "block", value: si.Block},
		{name: "block_iopoll", value: si.BlockIoPoll},
		{name: "tasklet", value: si.Tasklet},
		{name: "sched", value: si.Sched},
		{name: "hrtimer", value: si.Hrtimer},
		{name: "rcu", value: si.Rcu},
	} {
		ch <- prometheus.MustNewConstMetric(c.softIRQ, prometheus.CounterValue, float64(vec.value), vec.name)
	}
}

func (c *StatCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.intr
	ch <- c.ctxt
	ch <- c.forks
	ch <- c.btime
	ch <- c.procsRunning
	ch <- c.procsBlocked
	ch <- c.softIRQ
} 
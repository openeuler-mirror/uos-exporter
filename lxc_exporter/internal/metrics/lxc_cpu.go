package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"lxc_exporter/internal/exporter"
	lxc2 "lxc_exporter/internal/lxc"
)

func init() {
	exporter.Register(
		NewLxcCpu("lxc_cpu_usage",
			"lxc cpu usage",
			[]string{"name",
				"type"}))
}

type LxcCpu struct {
	*baseMetrics
}

func NewLxcCpu(fqname, help string, labels []string) *LxcCpu {
	return &LxcCpu{NewMetrics(fqname, help, labels)}
}

func (c *LxcCpu) Collect(ch chan<- prometheus.Metric) {
	lxc := lxc2.NewLxc()
	lxc.UpdateContainerNameAll()
	if len(lxc.GetContainerNameAll()) == 0 {
		logrus.Warnf("No container found")
		return
	}
	logrus.Debugf("Get container name all: %v",
		lxc.GetContainerNameAll())
	for _, name := range lxc.GetContainerNameAll() {
		stat, err := lxc.GetCPUStat(name)
		if err != nil {
			logrus.Warnf("Get container %s cpu stat failed: %v",
				name,
				err)
			continue
		}
		logrus.Debugf("Get container %s cpu stat: %v",
			name,
			stat)
		c.baseMetrics.collect(ch,
			stat.Usage,
			[]string{name,
				"usage"})
		c.baseMetrics.collect(ch,
			stat.System,
			[]string{name,
				"system"})
		c.baseMetrics.collect(ch,
			stat.User,
			[]string{name,
				"user"})
		c.baseMetrics.collect(ch,
			stat.ThrottledUsec,
			[]string{name,
				"throttled"})
		c.baseMetrics.collect(ch,
			stat.BurstUsec,
			[]string{name,
				"burst"})
		c.baseMetrics.collect(ch,
			stat.ForceIdle,
			[]string{name,
				"force_idle"})
		c.baseMetrics.collect(ch,
			stat.NrPeriods,
			[]string{name,
				"nr_periods"})
		c.baseMetrics.collect(ch,
			stat.NrThrottled,
			[]string{name,
				"nr_throttled"})
		c.baseMetrics.collect(ch,
			stat.Nrbursts,
			[]string{name,
				"nr_bursts"})
	}
}

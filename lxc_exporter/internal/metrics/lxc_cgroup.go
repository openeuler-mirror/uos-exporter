package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"lxc_exporter/internal/exporter"
	lxc2 "lxc_exporter/internal/lxc"
)

func init() {
	exporter.Register(
		NewLxcCgroup("lxc_cgroup_info",
			"lxc cgroup info",
			[]string{"name",
				"type"}))
}

type LxcCgroup struct {
	*baseMetrics
}

func NewLxcCgroup(fqname, help string, labels []string) *LxcCgroup {
	return &LxcCgroup{NewMetrics(fqname, help, labels)}
}

func (c *LxcCgroup) Collect(ch chan<- prometheus.Metric) {
	lxc := lxc2.NewLxc()
	lxc.UpdateContainerNameAll()
	if len(lxc.GetContainerNameAll()) == 0 {
		logrus.Warnf("No container found")
		return
	}
	logrus.Debugf("Get container name all: %v",
		lxc.GetContainerNameAll())
	for _, name := range lxc.GetContainerNameAll() {
		stat, err := lxc.GetCgroupStat(name)
		if err != nil {
			logrus.Warnf("Get container %s cgroup stat failed: %v",
				name,
				err)
			continue
		}
		logrus.Debugf("Get container %s cgroup stat: %v",
			name,
			stat)
		c.baseMetrics.collect(ch,
			stat.NrDescendants,
			[]string{name,
				"nr_descendants"})
		c.baseMetrics.collect(ch,
			stat.NrDyingDescendants,
			[]string{name,
				"nr_dying_descendants"})
	}
}

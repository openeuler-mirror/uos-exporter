package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"systemd_resolved_exporter/internal/exporter"
	"systemd_resolved_exporter/internal/systemd"
)

func init() {
	exporter.Register(
		NewCache("cache_info",
			"cache info",
			[]string{"type"}))
}

type Cache struct {
	*baseMetrics
}

func NewCache(fqname, help string, labels []string) *Cache {
	return &Cache{NewMetrics(fqname, help, labels)}
}

func (c *Cache) Collect(ch chan<- prometheus.Metric) {
	stats := systemd.GetSystemdResolvedStats()
	value, ok := stats["Current Cache Size"]
	if !ok {
		logrus.Errorf("get Current Cache failed")
	} else {
		c.baseMetrics.collect(ch, value, []string{"size"})
	}
	value, ok = stats["Cache Hits"]
	if !ok {
		logrus.Errorf("get Cache Hits failed")
	} else {
		c.baseMetrics.collect(ch, value, []string{"hits"})
	}
	value, ok = stats["Cache Misses"]
	if !ok {
		logrus.Errorf("get Cache Misses failed")
	} else {
		c.baseMetrics.collect(ch, value, []string{"misses"})
	}
}

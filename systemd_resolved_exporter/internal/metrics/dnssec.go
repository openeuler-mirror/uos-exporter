package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"systemd_resolved_exporter/internal/exporter"
	"systemd_resolved_exporter/internal/systemd"
)

func init() {
	exporter.Register(
		NewDnsSec("dns_sec_info",
			"dns sec info",
			[]string{"type"}))
}

type DnsSec struct {
	*baseMetrics
}

func NewDnsSec(fqname, help string, labels []string) *DnsSec {
	return &DnsSec{NewMetrics(fqname, help, labels)}
}

func (c *DnsSec) Collect(ch chan<- prometheus.Metric) {
	stats := systemd.GetSystemdResolvedStats()
	value, ok := stats["Secure"]
	if !ok {
		logrus.Errorf("get Secure failed")
	} else {
		c.baseMetrics.collect(ch, value, []string{"secure"})
	}
	value, ok = stats["Insecure"]
	if !ok {
		logrus.Errorf("get Insecure failed")
	} else {
		c.baseMetrics.collect(ch, value, []string{"insecure"})
	}
	value, ok = stats["Bogus"]
	if !ok {
		logrus.Errorf("get Bogus failed")
	} else {
		c.baseMetrics.collect(ch, value, []string{"bogus"})
	}
	value, ok = stats["Indeterminate"]
	if !ok {
		logrus.Errorf("get Indeterminate failed")
	} else {
		c.baseMetrics.collect(ch, value, []string{"Indeterminate"})
	}
}

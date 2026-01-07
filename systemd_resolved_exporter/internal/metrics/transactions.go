package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"systemd_resolved_exporter/internal/exporter"
	"systemd_resolved_exporter/internal/systemd"
)

func init() {
	exporter.Register(
		NewTransactions("transactions_info",
			"transactions info",
			[]string{"type"}))
}

type Transactions struct {
	*baseMetrics
}

func NewTransactions(fqname, help string, labels []string) *Transactions {
	return &Transactions{NewMetrics(fqname, help, labels)}
}

func (c *Transactions) Collect(ch chan<- prometheus.Metric) {
	stats := systemd.GetSystemdResolvedStats()
	value, ok := stats["Current Transactions"]
	if !ok {
		logrus.Errorf("get Current Transactions failed")
	} else {
		c.baseMetrics.collect(ch, value, []string{"current"})
	}
	value, ok = stats["Total Transactions"]
	if !ok {
		logrus.Errorf("get Total Transactions failed")
	} else {
		c.baseMetrics.collect(ch, value, []string{"total"})
	}
}

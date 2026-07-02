package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"nftables_exporter/internal/exporter"
)

func init() {
	exporter.Register(
		NewStatus("exporter_status",
			"exporter status",
			nil))
}

type Status struct {
	*baseMetrics
}

func NewStatus(fqname, help string, labels []string) *Status {
	return &Status{NewMetrics(fqname, help, labels)}
}

func (c *Status) Collect(ch chan<- prometheus.Metric) {
	c.baseMetrics.collect(ch, 1, nil)
}

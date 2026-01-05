package metrics

import (
	"fluentd_exporter/internal/exporter"
	"github.com/prometheus/client_golang/prometheus"
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

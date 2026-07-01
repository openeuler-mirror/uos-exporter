//go:build example

package metrics

import (
	"fluentd_exporter/internal/exporter"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	exporter.Register(
		NewFluentdRetryExample("fluentd_retry_count_example",
			"fluentd retry count example",
			[]string{"plugin_id", "plugin_category"}))
}

type FluentdRetryExample struct {
	*baseMetrics
}

func NewFluentdRetryExample(fqname, help string, labels []string) *FluentdRetryExample {
	return &FluentdRetryExample{NewMetrics(fqname, help, labels)}
}

func (fr *FluentdRetryExample) Collect(ch chan<- prometheus.Metric) {
	value := 32
	fr.baseMetrics.collect(ch, float64(value), []string{"aa", "bb"})
	fr.baseMetrics.collect(ch, float64(value+1), []string{"aa1", "bb"})
}

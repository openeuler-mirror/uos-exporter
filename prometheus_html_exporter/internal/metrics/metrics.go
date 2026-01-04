package metrics

import (
	"prometheus_html_exporter/internal/exporter"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	Name    = "prometheus-html-exporter"
	Version = "1.0.0"
)

// Register 注册一个指标收集器
func Register(collector prometheus.Collector) {
	exporter.Register(collector)
}

type baseMetrics struct {
	labels []string
	desc   *prometheus.Desc
}

func NewMetrics(fqname, help string, labels []string) *baseMetrics {
	return &baseMetrics{
		labels: labels,
		desc: prometheus.NewDesc(
			fqname,
			help,
			labels,
			nil),
	}
}

func (c *baseMetrics) collect(ch chan<- prometheus.Metric, value float64, labels []string) {
	ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, value, labels...)
}

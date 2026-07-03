package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

var (
	Name    = "nextdns_exporter"
	Version = "1.0.0"
)

// Register registers all metrics with the Prometheus registry
func Register(registry *prometheus.Registry) {
	// This is a placeholder - the actual registration is done in the server
}

type baseMetrics struct {
	name    string
	labels  []string
	desc    *prometheus.Desc
	logger  *logrus.Logger
}

func (m *baseMetrics) init(name string) {
	m.name = name
	m.logger = logrus.New()
	m.desc = prometheus.NewDesc(
		prometheus.BuildFQName(name, "", "up"),
		"Was the last scrape successful.",
		nil, nil,
	)
}

func (m *baseMetrics) Describe(ch chan<- *prometheus.Desc) {
	ch <- m.desc
}

func (m *baseMetrics) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(
		m.desc,
		prometheus.GaugeValue,
		1.0,
	)
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

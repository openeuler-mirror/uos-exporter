package metrics

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	Name    = "newrelic_exporter"
	Version = "1.0.0"
)

type baseMetrics struct {
	labels []string
	desc   *prometheus.Desc
}

// NewMetrics 创建一个新的基础指标
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

// collect 收集指标并发送到 Prometheus 通道
func (c *baseMetrics) collect(ch chan<- prometheus.Metric, value float64, labels []string) {
	ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, value, labels...)
}

// newMetric 创建一个新的指标 (与 NewMetrics 相同，但名称格式化为 namespace_name)
func newMetric(namespace, name, help string, labels []string) *baseMetrics {
	return NewMetrics(
		prometheus.BuildFQName(namespace, "", name),
		help,
		labels,
	)
}

// Observe 添加一个观察值到指标中
func (c *baseMetrics) Observe(ch chan<- prometheus.Metric, value float64, labels ...string) error {
	if len(labels) != len(c.labels) {
		return fmt.Errorf("标签数量不匹配: 预期 %d, 收到 %d", len(c.labels), len(labels))
	}
	c.collect(ch, value, labels)
	return nil
}

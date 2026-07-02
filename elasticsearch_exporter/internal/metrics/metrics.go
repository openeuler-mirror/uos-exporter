package metrics

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

var (
	Name    = "elasticsearch_exporter"
	Version = "1.0.0"
)

const (
	namespace     = "elasticsearch"
	defaultTimeout = 5 * time.Second
)

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

// collect 根据提供的标签收集指标
// 支持两种方式传递标签值：
// 1. 直接传递[]string类型的标签值数组
// 2. 传递prometheus.Labels类型的标签键值对
func (c *baseMetrics) collect(ch chan<- prometheus.Metric, value float64, labelValues interface{}) {
	switch labelValues := labelValues.(type) {
	case []string:
		// 检查标签值数量是否与预期一致
		if len(labelValues) != len(c.labels) {
			panic(fmt.Sprintf("inconsistent label cardinality: expected %d label values but got %d", 
				len(c.labels), len(labelValues)))
		}
		ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, value, labelValues...)
	case prometheus.Labels:
		// 将Labels转换为有序的[]string，与c.labels中定义的标签顺序一致
		var labelValuesList []string
		for _, labelName := range c.labels {
			labelValuesList = append(labelValuesList, labelValues[labelName])
		}
		ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, value, labelValuesList...)
	case nil:
		// 检查是否期望没有标签
		if len(c.labels) > 0 {
			panic(fmt.Sprintf("inconsistent label cardinality: expected %d label values but got 0", 
				len(c.labels)))
		}
		// 没有标签的情况
		ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, value)
	default:
		// 不支持其他类型，不做任何操作
	}
}
// Part 2 commit for elasticsearch_exporter/internal/metrics/metrics.go

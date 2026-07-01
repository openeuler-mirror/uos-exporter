package exporter

import "github.com/prometheus/client_golang/prometheus"

type Metric interface {
	Collect(ch chan<- prometheus.Metric)
	// 可选的Describe方法，用于描述指标
	Describe(ch chan<- *prometheus.Desc)
}

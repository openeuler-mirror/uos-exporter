package exporter

import "github.com/prometheus/client_golang/prometheus"

type Metric interface {
	Describe(ch chan<- *prometheus.Desc)
	Collect(ch chan<- prometheus.Metric)
}

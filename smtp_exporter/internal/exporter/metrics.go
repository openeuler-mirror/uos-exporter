package exporter

import "github.com/prometheus/client_golang/prometheus"

type Metric interface {
	Collect(ch chan<- prometheus.Metric)
}

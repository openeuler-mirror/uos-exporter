package exporter

import "github.com/prometheus/client_golang/prometheus"

type Metric interface {
	Collect(ch chan<- prometheus.Metric)
}
// Final commit for elasticsearch_exporter/internal/exporter/metrics.go

package exporter

import "github.com/prometheus/client_golang/prometheus"

type Metric interface {
	Collect(ch chan<- prometheus.Metric)
}
// Part 2 commit for node_process_exporter/internal/exporter/metrics.go

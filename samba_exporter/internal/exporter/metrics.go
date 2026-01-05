package exporter

import "github.com/prometheus/client_golang/prometheus"

type Metric interface {
	Describe(ch chan<- *prometheus.Desc)
	Collect(ch chan<- prometheus.Metric)
}
// Part 2 commit for samba_exporter/internal/exporter/metrics.go

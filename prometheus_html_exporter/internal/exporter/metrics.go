package exporter

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Interface for metrics that expose Collect method
type Metric interface {
	prometheus.Collector
}

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	Name    = "node_system_exporter"
	Version = "1.0.0"
)

type baseMetrics struct {
	labels []string
	desc   *prometheus.Desc
}


// TODO: implement functions

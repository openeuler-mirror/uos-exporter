package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	Name    = "mail_exporter"
	Version = "1.0.0"
)

type baseMetrics struct {
	desc *prometheus.Desc
}


// TODO: implement functions

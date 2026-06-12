package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type baseMetrics struct {
	labels []string
	desc   *prometheus.Desc
}


// TODO: implement

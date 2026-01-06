package metrics

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	Name    = "newrelic_exporter"
	Version = "1.0.0"
)

type baseMetrics struct {
	labels []string
	desc   *prometheus.Desc
}

// NewMetrics 创建一个新的基础指标

// TODO: implement functions

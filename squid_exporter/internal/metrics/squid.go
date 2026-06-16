package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"log"
)

type SquidConfig struct {
	Hostname     string
	Port         int
	Login        string
	Password     string
	Headers      []string
	ExtractTimes bool
}

// SquidCollector 是主Squid指标收集器
type SquidCollector struct {
	client       SquidClient
	hostname     string
	port         int
	extractTimes bool
	up           prometheus.Gauge
}

// NewSquidCollector 创建一个新的Squid指标收集器

// TODO: implement functions

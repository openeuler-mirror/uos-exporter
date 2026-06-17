package exporter

import (
	"dhcpd_leases_exporter/pkg/collector"
	"github.com/prometheus/client_golang/prometheus"
)

// MetricsCollector 实现 prometheus.Collector 接口
type MetricsCollector struct {
	leasesFile string
	dhcpdInfo  *collector.DHCPDInfo

	validLeases   *prometheus.Desc
	expiredLeases *prometheus.Desc
	totalLeases   *prometheus.Desc
	fileTime      *prometheus.Desc
	activeLease   *prometheus.Desc
}

// NewMetricsCollector 创建一个新的指标收集器

// TODO: implement functions

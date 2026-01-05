package exporter

import (
	"context"
	"log/slog"
	"net"
	"time"

	"network_exporter/config"
	"network_exporter/internal/metrics"

	"github.com/prometheus/client_golang/prometheus"
)

// NetworkExporter 网络监控导出器
type NetworkExporter struct {
	logger      *slog.Logger
	config      *config.SafeConfig
	resolver    *net.Resolver
	pingMetrics *metrics.PingMetrics
	tcpMetrics  *metrics.TCPMetrics
	httpMetrics *metrics.HTTPMetrics
	mtrMetrics  *metrics.MTRMetrics
}

// NewNetworkExporter 创建新的网络导出器

// TODO: implement functions

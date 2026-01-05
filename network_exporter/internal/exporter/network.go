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
func NewNetworkExporter(logger *slog.Logger, cfg *config.SafeConfig) *NetworkExporter {
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: time.Second * 5,
			}
			return d.DialContext(ctx, network, address)
		},
	}

	return &NetworkExporter{
		logger:      logger,
		config:      cfg,
		resolver:    resolver,
		pingMetrics: metrics.NewPingMetrics(logger, resolver),
		tcpMetrics:  metrics.NewTCPMetrics(logger, resolver),
		httpMetrics: metrics.NewHTTPMetrics(logger, resolver),
		mtrMetrics:  metrics.NewMTRMetrics(logger, resolver),
	}
}

// Describe 实现prometheus.Collector接口
func (ne *NetworkExporter) Describe(ch chan<- *prometheus.Desc) {
	ne.pingMetrics.Describe(ch)
	ne.tcpMetrics.Describe(ch)
	ne.httpMetrics.Describe(ch)
	ne.mtrMetrics.Describe(ch)
}

// Collect 实现prometheus.Collector接口
func (ne *NetworkExporter) Collect(ch chan<- prometheus.Metric) {
	cfg := ne.config.Cfg
	if cfg == nil {
		ne.logger.Warn("Config is nil")
		return
	}

	// 收集各种metrics数据
	ne.pingMetrics.Collect(cfg)
	ne.tcpMetrics.Collect(cfg)
	ne.httpMetrics.Collect(cfg)
	ne.mtrMetrics.Collect(cfg)

	// 将所有metrics发送到channel
	ne.pingMetrics.CollectMetrics(ch)
	ne.tcpMetrics.CollectMetrics(ch)
	ne.httpMetrics.CollectMetrics(ch)
	ne.mtrMetrics.CollectMetrics(ch)
}

// Start 启动网络监控
func (ne *NetworkExporter) Start() {
	// 启动各种监控器
	ne.logger.Info("Starting network monitors...")
	
	// TODO: 实现PING、MTR、TCP、HTTP监控逻辑
} 
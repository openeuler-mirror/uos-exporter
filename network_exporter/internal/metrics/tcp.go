package metrics

import (
	"context"
	"log/slog"
	"net"
	"time"

	"network_exporter/config"
	"network_exporter/pkg/common"
	"network_exporter/pkg/tcp"
	"github.com/prometheus/client_golang/prometheus"
)

// TCPMetrics TCP相关的metrics
type TCPMetrics struct {
	*baseMetrics
	logger   *slog.Logger
	resolver *net.Resolver
}

// NewTCPMetrics 创建新的TCP metrics实例
func NewTCPMetrics(logger *slog.Logger, resolver *net.Resolver) *TCPMetrics {
	base := newBaseMetrics("tcp")
	
	// 添加TCP特有的metrics，保持与旧项目一致的名称和标签
	base.addMetric("connection_seconds", "Connection time in seconds", []string{"name", "target", "target_ip", "source_ip", "port"})
	base.addMetric("connection_status", "Connection Status", []string{"name", "target", "target_ip", "source_ip", "port"})
	base.addMetric("targets", "Number of active targets", nil)
	base.addMetric("up", "Exporter state", nil)

	return &TCPMetrics{
		baseMetrics: base,
		logger:      logger,
		resolver:    resolver,
	}
}

// Describe 实现prometheus.Collector接口
func (t *TCPMetrics) Describe(ch chan<- *prometheus.Desc) {
	t.baseMetrics.Describe(ch)
}

// CollectMetrics 实现prometheus.Collector接口
func (t *TCPMetrics) CollectMetrics(ch chan<- prometheus.Metric) {
	t.baseMetrics.Collect(ch)
}

// Collect 收集TCP metrics
func (t *TCPMetrics) Collect(cfg *config.NetworkConfig) {
	if cfg == nil {
		t.logger.Warn("TCP config is nil")
		return
	}

	// 设置up状态
	t.setMetric("up", 1)

	targetCount := 0
	for _, target := range cfg.Targets {
		if target.Type != "TCP" {
			continue
		}

		targetCount++
		
		// 解析目标地址
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		ipAddrs, err := common.DestAddrs(ctx, target.Host, t.resolver, 5*time.Second)
		cancel()
		
		if err != nil || len(ipAddrs) == 0 {
			t.logger.Warn("Failed to resolve target", "host", target.Host, "error", err)
			// 设置失败状态
			t.setMetricWithLabels("connection_status", 0, map[string]string{
				"name":      target.Name,
				"target":    target.Host,
				"target_ip": "unknown",
				"source_ip": target.SourceIp,
				"port":      target.Port,
			})
			continue
		}

		for _, ip := range ipAddrs {
			// 执行TCP连接测试
			result := tcp.TestTCPPort(target.Host, ip, target.Port, target.SourceIp, cfg.TCP.Timeout.Duration())
			
			labels := map[string]string{
				"name":      target.Name,
				"target":    target.Host,
				"target_ip": ip,
				"source_ip": result.SrcIp,
				"port":      target.Port,
			}

			// 设置metrics
			if result.Success {
				t.setMetricWithLabels("connection_status", 1, labels)
				t.setMetricWithLabels("connection_seconds", result.ConTime.Seconds(), labels)
			} else {
				t.setMetricWithLabels("connection_status", 0, labels)
			}
		}
	}

	t.setMetric("targets", float64(targetCount))
} 
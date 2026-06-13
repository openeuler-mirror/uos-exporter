package metrics

import (
	"context"
	"log/slog"
	"net"
	"sync"
	"time"

	"network_exporter/config"
	"network_exporter/pkg/common"
	"network_exporter/pkg/ping"
	
	"github.com/prometheus/client_golang/prometheus"
)

// PingCacheEntry PING缓存条目
type PingCacheEntry struct {
	result    *ping.PingResult
	timestamp time.Time
}

// PingMetrics ping相关的metrics
type PingMetrics struct {
	*baseMetrics
	logger   *slog.Logger
	resolver *net.Resolver
	icmpID   *common.IcmpID
	cache    map[string]*PingCacheEntry
	cacheMux sync.RWMutex
	cacheTTL time.Duration
}

// NewPingMetrics 创建新的ping metrics实例
func NewPingMetrics(logger *slog.Logger, resolver *net.Resolver) *PingMetrics {
	base := newBaseMetrics("ping")
	
	// 添加ping特有的metrics，保持与旧项目一致的名称和标签
	base.addMetric("status", "Ping Status", []string{"name", "target", "target_ip"})
	base.addMetric("rtt_seconds", "Round Trip Time in seconds", []string{"name", "target", "target_ip", "type"})
	base.addMetric("rtt_snt_count", "Packet sent count", []string{"name", "target", "target_ip"})
	base.addMetric("rtt_snt_fail_count", "Packet sent fail count", []string{"name", "target", "target_ip"})
	base.addMetric("rtt_snt_seconds", "Packet sent time total", []string{"name", "target", "target_ip"})
	base.addMetric("loss_percent", "Packet loss in percent", []string{"name", "target", "target_ip"})
	base.addMetric("targets", "Number of active targets", nil)
	base.addMetric("up", "Exporter state", nil)

	return &PingMetrics{
		baseMetrics: base,
		logger:      logger,
		resolver:    resolver,
		icmpID:      &common.IcmpID{},
		cache:       make(map[string]*PingCacheEntry),
		cacheTTL:    15 * time.Second, // 缓存15秒
	}
}

// Describe 实现prometheus.Collector接口
func (p *PingMetrics) Describe(ch chan<- *prometheus.Desc) {
	p.baseMetrics.Describe(ch)
}

// CollectMetrics 实现prometheus.Collector接口
func (p *PingMetrics) CollectMetrics(ch chan<- prometheus.Metric) {
	p.baseMetrics.Collect(ch)
}

// getCachedResult 获取缓存的结果
func (p *PingMetrics) getCachedResult(key string) *ping.PingResult {
	p.cacheMux.RLock()
	defer p.cacheMux.RUnlock()
	
	if entry, exists := p.cache[key]; exists {
		// 检查缓存是否过期
		if time.Since(entry.timestamp) < p.cacheTTL {
			return entry.result
		}
	}
	return nil
}

// setCachedResult 设置缓存结果
func (p *PingMetrics) setCachedResult(key string, result *ping.PingResult) {
	p.cacheMux.Lock()
	defer p.cacheMux.Unlock()
	
	p.cache[key] = &PingCacheEntry{
		result:    result,
		timestamp: time.Now(),
	}
}

// Collect 收集ping metrics数据
func (p *PingMetrics) Collect(cfg *config.NetworkConfig) {
	if cfg == nil {
		p.logger.Warn("ICMP config is nil")
		return
	}

	// 设置up状态
	p.setMetric("up", 1)

	targetCount := 0
	for _, target := range cfg.Targets {
		if target.Type != "ICMP" && target.Type != "ICMP+MTR" {
			continue
		}

		targetCount++
		
		// 解析目标地址
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		ipAddrs, err := common.DestAddrs(ctx, target.Host, p.resolver, 3*time.Second)
		cancel()
		
		if err != nil || len(ipAddrs) == 0 {
			p.logger.Warn("Failed to resolve target", "host", target.Host, "error", err)
			// 设置失败状态
			p.setMetricWithLabels("status", 0, map[string]string{
				"name":      target.Name,
				"target":    target.Host,
				"target_ip": "unknown",
			})
			continue
		}

		for _, ip := range ipAddrs {
			// 创建缓存键
			cacheKey := target.Name + "_" + ip
			
			// 先尝试从缓存获取结果
			result := p.getCachedResult(cacheKey)
			if result == nil {
				// 缓存中没有或已过期，执行ping
				p.logger.Debug("Executing PING", "target", target.Name, "ip", ip)
				result = ping.Ping(target.Host, ip, 3*time.Second, 3)
				// 缓存结果
				p.setCachedResult(cacheKey, result)
			} else {
				p.logger.Debug("Using cached PING result", "target", target.Name, "ip", ip)
			}
			
			labels := map[string]string{
				"name":      target.Name,
				"target":    target.Host,
				"target_ip": ip,
			}

			// 设置基本状态metrics
			if result.Success {
				p.setMetricWithLabels("status", 1, labels)
				
				// 各种类型的RTT指标
				p.setMetricWithLabels("rtt_seconds", result.BestTime.Seconds(), map[string]string{
					"name":      target.Name,
					"target":    target.Host,
					"target_ip": ip,
					"type":      "best",
				})
				p.setMetricWithLabels("rtt_seconds", result.AvgTime.Seconds(), map[string]string{
					"name":      target.Name,
					"target":    target.Host,
					"target_ip": ip,
					"type":      "mean",
				})
				p.setMetricWithLabels("rtt_seconds", result.WorstTime.Seconds(), map[string]string{
					"name":      target.Name,
					"target":    target.Host,
					"target_ip": ip,
					"type":      "worst",
				})
				p.setMetricWithLabels("rtt_seconds", result.SumTime.Seconds(), map[string]string{
					"name":      target.Name,
					"target":    target.Host,
					"target_ip": ip,
					"type":      "sum",
				})
				p.setMetricWithLabels("rtt_seconds", result.SquaredDeviationTime.Seconds(), map[string]string{
					"name":      target.Name,
					"target":    target.Host,
					"target_ip": ip,
					"type":      "sd",
				})
				p.setMetricWithLabels("rtt_seconds", result.UncorrectedSDTime.Seconds(), map[string]string{
					"name":      target.Name,
					"target":    target.Host,
					"target_ip": ip,
					"type":      "usd",
				})
				p.setMetricWithLabels("rtt_seconds", result.CorrectedSDTime.Seconds(), map[string]string{
					"name":      target.Name,
					"target":    target.Host,
					"target_ip": ip,
					"type":      "csd",
				})
				p.setMetricWithLabels("rtt_seconds", result.RangeTime.Seconds(), map[string]string{
					"name":      target.Name,
					"target":    target.Host,
					"target_ip": ip,
					"type":      "range",
				})
				
				p.setMetricWithLabels("loss_percent", result.DropRate, labels)
			} else {
				p.setMetricWithLabels("status", 0, labels)
				p.setMetricWithLabels("loss_percent", 1.0, labels)
			}
			
			// 设置包统计数据
			p.setMetricWithLabels("rtt_snt_count", float64(result.SntSummary), labels)
			p.setMetricWithLabels("rtt_snt_fail_count", float64(result.SntFailSummary), labels)
			p.setMetricWithLabels("rtt_snt_seconds", result.SntTimeSummary.Seconds(), labels)
		}
	}

	p.setMetric("targets", float64(targetCount))
} 
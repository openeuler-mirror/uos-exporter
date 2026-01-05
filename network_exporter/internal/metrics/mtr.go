package metrics

import (
	"context"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"network_exporter/config"
	"network_exporter/pkg/common"
	"network_exporter/pkg/mtr"
	"github.com/prometheus/client_golang/prometheus"
)

// MTRCacheEntry 缓存条目
type MTRCacheEntry struct {
	result    *mtr.MtrResult
	timestamp time.Time
}

// MTRMetrics MTR相关的metrics
type MTRMetrics struct {
	*baseMetrics
	logger   *slog.Logger
	resolver *net.Resolver
	cache    map[string]*MTRCacheEntry
	cacheMux sync.RWMutex
	cacheTTL time.Duration
}

// NewMTRMetrics 创建新的MTR metrics实例
func NewMTRMetrics(logger *slog.Logger, resolver *net.Resolver) *MTRMetrics {
	base := newBaseMetrics("mtr")
	
	// 添加MTR特有的metrics，保持与旧项目一致的名称和标签
	base.addMetric("rtt_seconds", "Round Trip Time in seconds", []string{"name", "target", "ttl", "path", "type"})
	base.addMetric("rtt_snt_count", "Round Trip Send Package Total", []string{"name", "target", "ttl", "path"})
	base.addMetric("rtt_snt_fail_count", "Round Trip Send Package Fail Total", []string{"name", "target", "ttl", "path"})
	base.addMetric("rtt_snt_seconds", "Round Trip Send Package Time Total", []string{"name", "target", "ttl", "path"})
	base.addMetric("hops", "Number of route hops", []string{"name", "target"})
	base.addMetric("targets", "Number of active targets", nil)
	base.addMetric("up", "Exporter state", nil)

	return &MTRMetrics{
		baseMetrics: base,
		logger:      logger,
		resolver:    resolver,
		cache:       make(map[string]*MTRCacheEntry),
		cacheTTL:    30 * time.Second, // 缓存30秒
	}
}

// Describe 实现prometheus.Collector接口
func (m *MTRMetrics) Describe(ch chan<- *prometheus.Desc) {
	m.baseMetrics.Describe(ch)
}

// CollectMetrics 实现prometheus.Collector接口
func (m *MTRMetrics) CollectMetrics(ch chan<- prometheus.Metric) {
	m.baseMetrics.Collect(ch)
}

// getCachedResult 获取缓存的结果
func (m *MTRMetrics) getCachedResult(key string) *mtr.MtrResult {
	m.cacheMux.RLock()
	defer m.cacheMux.RUnlock()
	
	if entry, exists := m.cache[key]; exists {
		// 检查缓存是否过期
		if time.Since(entry.timestamp) < m.cacheTTL {
			return entry.result
		}
	}
	return nil
}

// setCachedResult 设置缓存结果
func (m *MTRMetrics) setCachedResult(key string, result *mtr.MtrResult) {
	m.cacheMux.Lock()
	defer m.cacheMux.Unlock()
	
	m.cache[key] = &MTRCacheEntry{
		result:    result,
		timestamp: time.Now(),
	}
}

// Collect 收集MTR metrics
func (m *MTRMetrics) Collect(cfg *config.NetworkConfig) {
	if cfg == nil {
		m.logger.Warn("MTR config is nil")
		return
	}

	// 设置up状态
	m.setMetric("up", 1)

	targetCount := 0
	for _, target := range cfg.Targets {
		if target.Type != "MTR" && target.Type != "ICMP+MTR" {
			continue
		}

		targetCount++
		
		// 解析目标地址
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		ipAddrs, err := common.DestAddrs(ctx, target.Host, m.resolver, 5*time.Second)
		cancel()
		
		if err != nil || len(ipAddrs) == 0 {
			m.logger.Warn("Failed to resolve target", "host", target.Host, "error", err)
			continue
		}

		for _, ip := range ipAddrs {
			// 创建缓存键
			cacheKey := target.Name + "_" + ip
			
			// 先尝试从缓存获取结果
			result := m.getCachedResult(cacheKey)
			if result == nil {
				// 缓存中没有或已过期，执行MTR
				m.logger.Debug("Executing MTR", "target", target.Name, "ip", ip)
				result = mtr.RunMTR(ip, "", cfg.MTR.Timeout.Duration(), cfg.MTR.MaxHops, cfg.MTR.Count)
				// 缓存结果
				m.setCachedResult(cacheKey, result)
			} else {
				m.logger.Debug("Using cached MTR result", "target", target.Name, "ip", ip)
			}
			
			// 设置hops总数 (使用旧项目的标签格式)
			baseLabels := map[string]string{
				"name":   target.Name,
				"target": result.DestAddr,
			}
			m.setMetricWithLabels("hops", float64(len(result.Hops)), baseLabels)

			// 设置每一跳的详细metrics (匹配旧项目的指标)
			for _, hop := range result.Hops {
				// 各种类型的RTT指标
				m.setMetricWithLabels("rtt_seconds", hop.LastTime.Seconds(), 
					map[string]string{
						"name":   target.Name,
						"target": result.DestAddr,
						"ttl":    strconv.Itoa(hop.TTL),
						"path":   hop.AddressTo,
						"type":   "last",
					})
				m.setMetricWithLabels("rtt_seconds", hop.SumTime.Seconds(), 
					map[string]string{
						"name":   target.Name,
						"target": result.DestAddr,
						"ttl":    strconv.Itoa(hop.TTL),
						"path":   hop.AddressTo,
						"type":   "sum",
					})
				m.setMetricWithLabels("rtt_seconds", hop.BestTime.Seconds(), 
					map[string]string{
						"name":   target.Name,
						"target": result.DestAddr,
						"ttl":    strconv.Itoa(hop.TTL),
						"path":   hop.AddressTo,
						"type":   "best",
					})
				m.setMetricWithLabels("rtt_seconds", hop.AvgTime.Seconds(), 
					map[string]string{
						"name":   target.Name,
						"target": result.DestAddr,
						"ttl":    strconv.Itoa(hop.TTL),
						"path":   hop.AddressTo,
						"type":   "mean",
					})
				m.setMetricWithLabels("rtt_seconds", hop.WorstTime.Seconds(), 
					map[string]string{
						"name":   target.Name,
						"target": result.DestAddr,
						"ttl":    strconv.Itoa(hop.TTL),
						"path":   hop.AddressTo,
						"type":   "worst",
					})
				m.setMetricWithLabels("rtt_seconds", hop.SquaredDeviationTime.Seconds(), 
					map[string]string{
						"name":   target.Name,
						"target": result.DestAddr,
						"ttl":    strconv.Itoa(hop.TTL),
						"path":   hop.AddressTo,
						"type":   "sd",
					})
				m.setMetricWithLabels("rtt_seconds", hop.UncorrectedSDTime.Seconds(), 
					map[string]string{
						"name":   target.Name,
						"target": result.DestAddr,
						"ttl":    strconv.Itoa(hop.TTL),
						"path":   hop.AddressTo,
						"type":   "usd",
					})
				m.setMetricWithLabels("rtt_seconds", hop.CorrectedSDTime.Seconds(), 
					map[string]string{
						"name":   target.Name,
						"target": result.DestAddr,
						"ttl":    strconv.Itoa(hop.TTL),
						"path":   hop.AddressTo,
						"type":   "csd",
					})
				m.setMetricWithLabels("rtt_seconds", hop.RangeTime.Seconds(), 
					map[string]string{
						"name":   target.Name,
						"target": result.DestAddr,
						"ttl":    strconv.Itoa(hop.TTL),
						"path":   hop.AddressTo,
						"type":   "range",
					})
				m.setMetricWithLabels("rtt_seconds", hop.Loss, 
					map[string]string{
						"name":   target.Name,
						"target": result.DestAddr,
						"ttl":    strconv.Itoa(hop.TTL),
						"path":   hop.AddressTo,
						"type":   "loss",
					})
			}

			// 设置HopSummaryMap的metrics
			for ttlKey, summary := range result.HopSummaryMap {
				ttlParts := strings.Split(ttlKey, "_")
				ttlStr := ttlParts[0]
				
				summaryLabels := map[string]string{
					"name":   target.Name,
					"target": result.DestAddr,
					"ttl":    ttlStr,
					"path":   summary.AddressTo,
				}

				m.setMetricWithLabels("rtt_snt_count", float64(summary.Snt), summaryLabels)
				m.setMetricWithLabels("rtt_snt_fail_count", float64(summary.SntFail), summaryLabels)
				m.setMetricWithLabels("rtt_snt_seconds", summary.SntTime.Seconds(), summaryLabels)
			}
		}
	}

	m.setMetric("targets", float64(targetCount))
} 
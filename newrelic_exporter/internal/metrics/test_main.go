package metrics

import (
	"os"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"newrelic_exporter/internal/exporter"
	"newrelic_exporter/pkg/newrelic"
)

// 全局测试状态
var (
	originalCollector *NewRelicMetricsCollector
)

// TestMain 控制测试环境
func TestMain(m *testing.M) {
	// 备份原始 collector
	if collector != nil {
		originalCollector = &NewRelicMetricsCollector{
			config:   collector.config,
			apps:     collector.apps,
			names:    collector.names,
			metrics:  collector.metrics,
			mockMode: collector.mockMode,
		}
	}
	
	// 确保 collector 的 mockMode 在开始时为 false
	if collector != nil {
		collector.mockMode = false
	}
	
	// 运行测试
	result := m.Run()
	
	// 恢复原始状态
	if originalCollector != nil && collector != nil {
		collector.mockMode = originalCollector.mockMode
		collector.config = originalCollector.config
		collector.apps = originalCollector.apps
		collector.names = originalCollector.names
		collector.metrics = originalCollector.metrics
	}
	
	// 退出
	os.Exit(result)
}

// resetCollector 重置 collector 的状态，用于测试
func resetCollector() {
	collector = &NewRelicMetricsCollector{
		config:   &exporter.DefaultConfig.NewRelic,
		apps:     make([]newrelic.Application, 0),
		names:    make(map[int][]newrelic.MetricName),
		metrics:  make(map[string]*baseMetrics),
		mockMode: false,
		duration: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: NameSpace,
			Name:      "exporter_last_scrape_duration_seconds",
			Help:      "The last scrape duration.",
		}),
		error: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: NameSpace,
			Name:      "exporter_last_scrape_error",
			Help:      "The last scrape error status.",
		}),
		totalScrapes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: NameSpace,
			Name:      "exporter_scrapes_total",
			Help:      "Total scraped metrics",
		}),
	}
} 
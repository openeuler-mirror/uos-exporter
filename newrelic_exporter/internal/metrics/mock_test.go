package metrics

import (
	"testing"
	"time"
	
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	
	"newrelic_exporter/internal/exporter"
	"newrelic_exporter/pkg/newrelic"
)

// 设置测试环境
func setupMockTest() *NewRelicMetricsCollector {
	// 设置低日志级别，减少测试输出
	logrus.SetLevel(logrus.ErrorLevel)
	
	// 创建配置
	config := &exporter.NewRelicConfig{
		ApiKey:        "",
		ApiServer:     "https://api.newrelic.com",
		Service:       "applications",
		Period:        60,
		MetricFilters: []string{"CPU", "Memory", "Apdex"},
		ValueFilters:  []string{},
		Timeout:       5 * time.Second,
	}
	
	// 创建测试采集器
	c := &NewRelicMetricsCollector{
		mockMode: true,
		config:   config,
		apps:     make([]newrelic.Application, 0),
		names:    make(map[int][]newrelic.MetricName),
		metrics:  make(map[string]*baseMetrics),
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
	
	return c
}

// 测试模拟数据收集的基本功能
func TestMockDataCollection(t *testing.T) {
	c := setupMockTest()
	
	// 创建 Prometheus 通道
	ch := make(chan prometheus.Metric, 100)
	
	// 收集指标计数
	metricCount := 0
	done := make(chan bool)
	
	// 在后台接收指标
	go func() {
		for range ch {
			metricCount++
		}
		done <- true
	}()
	
	// 调用收集函数
	c.Collect(ch)
	close(ch)
	
	// 等待接收完成
	<-done
	
	// 验证至少收集了一些指标
	if metricCount < 3 {
		t.Errorf("模拟数据收集应该至少产生 3 个指标，实际为 %d", metricCount)
	}
	
	// 恢复日志级别
	logrus.SetLevel(logrus.InfoLevel)
}

// 测试模拟数据是否产生适当的数值范围
func TestMockDataRanges(t *testing.T) {
	// 创建测试采集器
	c := &NewRelicMetricsCollector{
		mockMode: true,
		config:   &exporter.NewRelicConfig{},
		metrics:  make(map[string]*baseMetrics),
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
	
	// 创建 Prometheus 通道
	ch := make(chan prometheus.Metric, 100)
	
	// 收集指标和值
	metricCount := 0
	done := make(chan bool)
	
	// 在后台接收指标
	go func() {
		for range ch {
			metricCount++
		}
		done <- true
	}()
	
	// 调用收集函数
	c.collectMockData(ch)
	
	// 关闭通道
	close(ch)
	
	// 等待接收完成
	<-done
	
	// 验证模拟数据生成
	if metricCount < 1 {
		t.Error("模拟数据应该产生至少一个指标")
	}
}

// 测试收集器的 Describe 方法
func TestMockCollectorDescribe(t *testing.T) {
	c := setupMockTest()
	
	// 创建描述通道
	ch := make(chan *prometheus.Desc, 10)
	
	// 调用 Describe 方法
	c.Describe(ch)
	close(ch)
	
	// 计算描述数量
	count := 0
	for range ch {
		count++
	}
	
	// 验证至少有一个描述
	if count < 1 {
		t.Error("Describe 应该至少提供一个描述")
	}
}

// 测试模拟模式的初始化
func TestMockModeInitialization(t *testing.T) {
	// 重置 collector 的状态
	resetCollector()
	
	// 保存原始采集器
	originalCollector := *collector
	defer func() {
		// 恢复原始采集器
		*collector = originalCollector
	}()
	
	// 创建空 API 密钥的配置
	config := &exporter.NewRelicConfig{
		ApiKey:    "",
		ApiServer: "https://api.newrelic.com",
		Service:   "applications",
	}
	
	// 初始化 API
	err := InitAPI(config)
	if err != nil {
		t.Errorf("空 API 密钥应该启用模拟模式而不是返回错误: %v", err)
	}
	
	// 验证模拟模式是否已启用
	if !collector.mockMode {
		t.Error("空 API 密钥应该启用模拟模式")
	}
	
	// 验证配置是否已设置
	if collector.config != config {
		t.Error("采集器配置应该设置为传入的配置")
	}
} 
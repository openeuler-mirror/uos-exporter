package metrics

import (
	"testing"
	"time"
	
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/http/httptest"
	"net/url"
	
	"newrelic_exporter/internal/exporter"
	"newrelic_exporter/pkg/newrelic"
)

// 测试初始化函数是否被正确调用
func TestInit(t *testing.T) {
	// 重置 collector 的状态
	resetCollector()
	
	// 确保 collector 被初始化
	if collector == nil {
		t.Fatal("collector 未被初始化")
	}
	
	// 验证 collector 的初始状态
	if collector.mockMode != false {
		t.Error("collector 的 mockMode 应该默认为 false")
	}
	
	if collector.config == nil {
		t.Error("collector 的 config 不应该为 nil")
	}
	
	if len(collector.metrics) != 0 {
		t.Error("collector 的 metrics 应该初始为空")
	}
}

// 测试 InitAPI 函数
func TestInitAPI(t *testing.T) {
	// 重置 collector 的状态
	resetCollector()
	
	// 备份原始 collector
	originalCollector := *collector
	// 测试后恢复原始值
	defer func() {
		*collector = originalCollector
	}()
	
	// 测试空配置
	err := InitAPI(nil)
	if err == nil {
		t.Error("空配置应该返回错误")
	}
	
	// 测试空 API 密钥
	config := &exporter.NewRelicConfig{
		ApiKey: "",
		ApiServer: "https://api.newrelic.com",
		Service: "applications",
	}
	
	err = InitAPI(config)
	if err != nil {
		t.Errorf("空 API 密钥应该启用模拟模式，而不是返回错误: %v", err)
	}
	
	if !collector.mockMode {
		t.Error("空 API 密钥应该启用模拟模式")
	}
	
	// 测试缺少服务名称
	config = &exporter.NewRelicConfig{
		ApiKey: "test-api-key",
		ApiServer: "https://api.newrelic.com",
		Service: "",
	}
	
	err = InitAPI(config)
	if err == nil {
		t.Error("缺少服务名称应该返回错误")
	}
	
	// 测试缺少 API 服务器
	config = &exporter.NewRelicConfig{
		ApiKey: "test-api-key",
		ApiServer: "",
		Service: "applications",
	}
	
	err = InitAPI(config)
	if err == nil {
		t.Error("缺少 API 服务器应该返回错误")
	}
}

// 测试 isNameFiltered 函数
func TestIsNameFiltered(t *testing.T) {
	c := &NewRelicMetricsCollector{
		config: &exporter.NewRelicConfig{
			MetricFilters: []string{"CPU", "Memory", "Apdex"},
		},
	}
	
	// 测试包含的指标名称
	if !c.isNameFiltered("CPU/Utilization") {
		t.Error("CPU/Utilization 应该通过过滤器")
	}
	
	if !c.isNameFiltered("Memory/Physical") {
		t.Error("Memory/Physical 应该通过过滤器")
	}
	
	// 测试不包含的指标名称
	if c.isNameFiltered("Network/Throughput") {
		t.Error("Network/Throughput 不应该通过过滤器")
	}
}

// 测试 isValueFiltered 函数
func TestIsValueFiltered(t *testing.T) {
	c := &NewRelicMetricsCollector{
		config: &exporter.NewRelicConfig{
			ValueFilters: []string{"average_response_time", "error_rate"},
		},
	}
	
	// 测试包含的值名称
	if !c.isValueFiltered("average_response_time") {
		t.Error("average_response_time 应该通过过滤器")
	}
	
	// 测试不包含的值名称
	if c.isValueFiltered("throughput") {
		t.Error("throughput 不应该通过过滤器")
	}
	
	// 测试空值过滤器
	c = &NewRelicMetricsCollector{
		config: &exporter.NewRelicConfig{
			ValueFilters: []string{},
		},
	}
	
	// 空值过滤器应该通过所有名称
	if !c.isValueFiltered("any_value") {
		t.Error("空值过滤器应该通过所有名称")
	}
}

// 模拟 NewRelic API 服务器
func mockNewRelicServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *newrelic.API) {
	server := httptest.NewServer(handler)
	
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("解析服务器 URL 失败: %v", err)
	}
	
	config := &exporter.NewRelicConfig{
		ApiKey: "test-api-key",
		Service: "applications",
		Timeout: 5 * time.Second,
		Period: 60,
	}
	
	api := &newrelic.API{
		Period: config.Period,
	}
	
	// 使用测试服务器 URL
	api.SetServer(*serverURL)
	api.SetAPIKey(config.ApiKey)
	api.SetService(config.Service)
	api.SetClient(&http.Client{Timeout: config.Timeout})
	
	return server, api
}

// 测试收集指标时处理模拟数据
func TestCollectWithMockMode(t *testing.T) {
	// 创建测试采集器
	c := &NewRelicMetricsCollector{
		mockMode: true,
		config: &exporter.NewRelicConfig{},
		metrics: make(map[string]*baseMetrics),
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
	ch := make(chan prometheus.Metric)
	
	// 在 goroutine 中接收通道数据，以避免阻塞
	go func() {
		for range ch {
			// 简单消费通道
		}
	}()
	
	// 调用收集函数
	c.Collect(ch)
	close(ch)
	
	// 没有具体断言，我们只是确保在模拟模式下函数能够正常执行
}

// 测试 baseMetrics
func TestBaseMetrics(t *testing.T) {
	// 创建测试指标
	metric := NewMetrics(
		"test_metric",
		"This is a test metric",
		[]string{"label1", "label2"},
	)
	
	// 验证指标属性
	if len(metric.labels) != 2 {
		t.Errorf("预期标签数量为 2，实际为 %d", len(metric.labels))
	}
	
	// 创建收集通道
	ch := make(chan prometheus.Metric, 1)
	
	// 测试 Observe 函数
	err := metric.Observe(ch, 42.0, "value1", "value2")
	if err != nil {
		t.Errorf("Observe 函数返回错误: %v", err)
	}
	
	// 测试标签数量不匹配的情况
	err = metric.Observe(ch, 42.0, "value1")
	if err == nil {
		t.Error("标签数量不匹配应该返回错误")
	}
	
	close(ch)
}

// 测试 Describe 函数
func TestDescribe(t *testing.T) {
	c := &NewRelicMetricsCollector{
		metrics: make(map[string]*baseMetrics),
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
	
	// 创建描述通道
	ch := make(chan *prometheus.Desc, 10)
	
	// 调用 Describe 函数
	c.Describe(ch)
	close(ch)
	
	// 验证至少有一个描述
	count := 0
	for range ch {
		count++
	}
	
	if count < 1 {
		t.Error("Describe 应该至少发送一个描述")
	}
}

// 测试模拟数据收集 - 只是确保函数不会崩溃
func TestCollectMockData(t *testing.T) {
	// 设置低日志级别，减少测试输出
	logrus.SetLevel(logrus.ErrorLevel)
	
	// 创建测试采集器
	c := &NewRelicMetricsCollector{
		mockMode: true,
		config: &exporter.NewRelicConfig{},
		metrics: make(map[string]*baseMetrics),
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
	ch := make(chan prometheus.Metric)
	
	// 在 goroutine 中接收通道数据，以避免阻塞
	go func() {
		for range ch {
			// 简单消费通道
		}
	}()
	
	// 调用收集函数
	c.collectMockData(ch)
	close(ch)
	
	// 恢复日志级别
	logrus.SetLevel(logrus.InfoLevel)
}

// 测试 NewMetrics 和 newMetric 函数
func TestMetricCreation(t *testing.T) {
	// 测试 NewMetrics 函数
	metric1 := NewMetrics(
		"test_metric_1",
		"This is test metric 1",
		[]string{"label1"},
	)
	
	if metric1 == nil {
		t.Fatal("NewMetrics 应该返回一个非空对象")
	}
	
	if len(metric1.labels) != 1 {
		t.Errorf("预期标签数量为 1，实际为 %d", len(metric1.labels))
	}
	
	// 测试 newMetric 函数
	metric2 := newMetric(
		"test_namespace",
		"test_name",
		"This is test metric 2",
		[]string{"label1", "label2"},
	)
	
	if metric2 == nil {
		t.Fatal("newMetric 应该返回一个非空对象")
	}
	
	if len(metric2.labels) != 2 {
		t.Errorf("预期标签数量为 2，实际为 %d", len(metric2.labels))
	}
} 
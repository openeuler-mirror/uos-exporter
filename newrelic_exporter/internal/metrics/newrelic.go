package metrics

import (
	"fmt"
	"crypto/rand"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"newrelic_exporter/internal/exporter"
	"newrelic_exporter/pkg/newrelic"
)

// cryptoRandReader implements io.Reader using crypto/rand for secure random data
type cryptoRandReader struct{}

func (r *cryptoRandReader) Read(p []byte) (n int, err error) {
	return rand.Read(p)
}

// generateSecureRandomFloat64 generates a secure random float64 between min and max
func generateSecureRandomFloat64(min, max float64, reader *cryptoRandReader) float64 {
	if min >= max {
		return min
	}

	// Generate a random integer in the range [0, 2^53-1]
	maxInt := big.NewInt(1 << 53)
	n, err := rand.Int(rand.Reader, maxInt)
	if err != nil {
		// Fallback to min if crypto/rand fails
		return min
	}

	// Convert to float64 in [0, 1) range
	ratio := float64(n.Int64()) / float64(maxInt.Int64())

	// Scale to desired range
	return min + ratio*(max-min)
}

// NewRelic指标的命名空间
const NameSpace = "newrelic"

type Metric struct {
	App   string
	Name  string
	Value float64
	Label string
}

type NewRelicMetricsCollector struct {
	mu                                       sync.Mutex
	api                                      *newrelic.API
	config                                   *exporter.NewRelicConfig
	apps                                     []newrelic.Application
	names                                    map[int][]newrelic.MetricName
	metrics                                  map[string]*baseMetrics
	appListLastScrape, metricNamesLastScrape time.Time
	mockMode                                 bool  // 用于启用模拟数据模式
	duration                                 prometheus.Gauge // 用于记录最后一次抓取的持续时间
	error                                    prometheus.Gauge // 用于记录最后一次抓取的错误状态
	totalScrapes                             prometheus.Counter // 用于记录抓取总次数
}

// 全局缓存的单例
var (
	collector *NewRelicMetricsCollector
)

func init() {
	// 注册一个初始的采集器，配置会在后续运行时更新
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
	exporter.Register(collector)
}

// 初始化 API - 这将在服务器启动过程中调用，确保配置已加载
func InitAPI(config *exporter.NewRelicConfig) error {
	if config == nil {
		return fmt.Errorf("无法使用空配置初始化 NewRelic API")
	}

	logrus.Info("正在初始化 NewRelic API ...")

	// 验证配置
	if config.ApiKey == "" {
		logrus.Warn("API 密钥为空，将启用模拟数据模式")
		collector.mu.Lock()
		collector.mockMode = true
		collector.config = config
		collector.mu.Unlock()
		logrus.Info("成功启用模拟数据模式")
		return nil
	}

	if config.ApiServer == "" {
		return fmt.Errorf("API 服务器地址为空，请在配置文件中设置 newrelic.api_server")
	}

	if config.Service == "" {
		return fmt.Errorf("服务名称为空，请在配置文件中设置 newrelic.service")
	}

	logrus.Infof("初始化 NewRelic API，服务器地址：%s，服务：%s", config.ApiServer, config.Service)

	api, err := newrelic.NewAPI(config)
	if err != nil {
		logrus.Warnf("初始化 NewRelic API 失败: %v, 将启用模拟数据模式", err)
		collector.mu.Lock()
		collector.mockMode = true
		collector.config = config
		collector.mu.Unlock()
		logrus.Info("由于API初始化失败，成功启用模拟数据模式")
		return nil
	}

	collector.mu.Lock()
	collector.api = api
	collector.config = config
	collector.mockMode = false
	collector.mu.Unlock()

	logrus.Info("成功初始化 NewRelic API")
	return nil
}

func (c *NewRelicMetricsCollector) Collect(ch chan<- prometheus.Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()

	logrus.Info("NewRelicMetricsCollector.Collect() 被调用")

	// 确保 metrics 映射已初始化
	if c.metrics == nil {
		c.metrics = make(map[string]*baseMetrics)
	}

	// 增加抓取计数
	c.totalScrapes.Inc()
	// 重置错误状态
	c.error.Set(0)

	// 记录抓取开始时间
	startTime := time.Now()

	// 在模拟数据模式下，使用模拟数据
	if c.mockMode {
		logrus.Info("使用模拟数据模式")
		c.collectMockData(ch)
		return
	}

	// 检查配置
	if c.config == nil {
		logrus.Error("配置为空，无法继续")
		c.error.Set(1)
		// 发送基本指标
		ch <- c.duration
		ch <- c.error
		ch <- c.totalScrapes
		return
	}

	// 检查 API 是否已初始化
	if c.api == nil {
		logrus.Error("NewRelic API 未初始化，将启用模拟数据模式")
		c.mockMode = true
		c.error.Set(1)
		// 发送基本指标
		ch <- c.duration
		ch <- c.error
		ch <- c.totalScrapes
		return
	}

	// 定义抓取时间窗口
	var from, to time.Time
	from = time.Now().Add(-1 * time.Minute).Truncate(time.Minute)
	to = from.Add(time.Minute)

	metricChan := make(chan Metric)

	// 在一个 goroutine 中启动抓取，以便不阻塞 Collect 流程
	go func() {
		c.scrape(from, to, metricChan)
		close(metricChan) // 确保在完成后关闭通道
	}()

	// 处理收到的指标
	c.receive(metricChan, ch)

	// 计算并设置抓取持续时间
	durationSeconds := float64(time.Now().UnixNano() - startTime.UnixNano()) / 1e9
	c.duration.Set(durationSeconds)
	logrus.Infof("收集完成，总耗时: %.3f 秒", durationSeconds)

	// 发送基本指标
	ch <- c.duration
	ch <- c.error
	ch <- c.totalScrapes
}

// 生成模拟数据用于测试
func (c *NewRelicMetricsCollector) collectMockData(ch chan<- prometheus.Metric) {
	logrus.Info("使用模拟数据模式收集指标")

	// 确保 metrics 映射已初始化
	if c.metrics == nil {
		c.metrics = make(map[string]*baseMetrics)
	}

	// 增加抓取计数
	c.totalScrapes.Inc()
	c.error.Set(0) // 重置错误状态

	// 记录开始时间
	startTime := time.Now()

	// 使用加密安全的随机数生成器
	cryptoRand := &cryptoRandReader{}

	// 模拟应用列表
	mockApps := []string{
		"MockApp1", 
		"MockApp2", 
		"ApiService", 
		"WebFrontend", 
		"DatabaseServer",
	}
	
	// 模拟指标类型
	mockMetricTypes := []struct {
		label string
		names []struct {
			name  string
			min   float64
			max   float64
		}
	}{
		{
			label: "application_summary",
			names: []struct {
				name  string
				min   float64
				max   float64
			}{
				{"response_time", 5, 200},
				{"throughput", 100, 2000},
				{"error_rate", 0, 10},
				{"apdex", 0, 1},
				{"memory_used", 100, 1000},
				{"cpu_utilization", 5, 95},
			},
		},
		{
			label: "Database",
			names: []struct {
				name  string
				min   float64
				max   float64
			}{
				{"query_time", 1, 100},
				{"throughput", 50, 1500},
				{"connections", 10, 200},
				{"pool_size", 5, 50},
				{"wait_time", 0, 30},
			},
		},
		{
			label: "WebTransaction",
			names: []struct {
				name  string
				min   float64
				max   float64
			}{
				{"response_time", 10, 500},
				{"throughput", 50, 1500},
				{"error_rate", 0, 20},
				{"apdex", 0, 1},
			},
		},
		{
			label: "HttpDispatcher",
			names: []struct {
				name  string
				min   float64
				max   float64
			}{
				{"response_time", 5, 300},
				{"request_count", 1000, 10000},
				{"error_count", 0, 100},
			},
		},
	}

	// 创建模拟指标数据
	var mockMetrics []Metric

	// 为每个应用生成所有指标类型的数据
	for _, app := range mockApps {
		for _, metricType := range mockMetricTypes {
			for _, metric := range metricType.names {
				// 生成安全的随机值
				randomValue := generateSecureRandomFloat64(metric.min, metric.max, cryptoRand)

				mockMetrics = append(mockMetrics, Metric{
					App:   app,
					Name:  metric.name,
					Value: randomValue,
					Label: metricType.label,
				})
			}
		}
	}
	
	logrus.Infof("生成了 %d 条模拟指标数据", len(mockMetrics))
	
	// 创建指标通道并发送数据
	metricChan := make(chan Metric, len(mockMetrics))
	for _, m := range mockMetrics {
		metricChan <- m
		logrus.Debugf("发送模拟指标: App=%s, Name=%s, Value=%.2f, Label=%s", m.App, m.Name, m.Value, m.Label)
	}
	close(metricChan)
	
	// 接收并注册模拟指标
	c.receive(metricChan, ch)
	
	// 计算并设置抓取持续时间
	durationSeconds := float64(time.Now().UnixNano() - startTime.UnixNano()) / 1e9
	c.duration.Set(durationSeconds)
	logrus.Infof("模拟数据抓取完成，耗时: %.3f 秒", durationSeconds)
	
	// 发送基本指标
	ch <- c.duration
	ch <- c.error
	ch <- c.totalScrapes
	
	logrus.Info("模拟数据收集完成")
}

func (c *NewRelicMetricsCollector) scrape(from time.Time, to time.Time, ch chan<- Metric) {
	// 重置错误状态
	c.error.Set(0)
	
	// 记录开始时间 - 仅用于日志记录，不影响持续时间指标
	startTime := time.Now()
	logrus.Infof("开始新的抓取，时间: %v，抓取周期: %v 到 %v", startTime, from.Format(time.Stamp), to.Format(time.Stamp))
	
	// 检查API是否已初始化
	if c.api == nil {
		logrus.Error("无法执行scrape: API未初始化")
		c.error.Set(1)
		return
	}
	
	// 如果距上次抓取应用列表的时间超过缓存时间，重新抓取
	if time.Since(c.appListLastScrape) > c.config.AppsListCacheTime {
		// 获取应用列表
		apps, err := c.api.GetApplications()
		if err != nil {
			logrus.Errorf("获取应用列表错误: %v", err)
			c.error.Set(1)
			// 不要close(ch)，由调用者关闭
			return
		}
		c.apps = apps
		c.appListLastScrape = time.Now()
		logrus.Debugf("应用列表更新于 %v", c.appListLastScrape)
	} else {
		logrus.Debug("使用缓存的应用列表")
	}

	// 如果没有可用的应用，返回
	if len(c.apps) == 0 {
		logrus.Warn("没有找到应用")
		return
	}

	logrus.Debugf("发现 %d 个应用", len(c.apps))

	// 遍历每个应用
	for _, app := range c.apps {
		// 首先发送应用程序摘要数据
		for name, value := range app.AppSummary {
			ch <- Metric{
				App:   app.Name,
				Name:  name,
				Value: value,
				Label: "application_summary",
			}
		}
		
		// 发送用户摘要数据（如果有）
		for name, value := range app.UsrSummary {
			ch <- Metric{
				App:   app.Name,
				Name:  name,
				Value: value,
				Label: "end_user_summary",
			}
		}
		
		// 尝试使用指标名称缓存
		names, exists := c.names[app.ID]
		
		// 如果没有缓存或缓存过期，重新获取
		if !exists || time.Since(c.metricNamesLastScrape) > c.config.MetricNamesCacheTime {
			// 获取指标名称
			metricNames, err := c.api.GetMetricNames(app.ID)
			if err != nil {
				logrus.Warnf("获取应用 %s (ID=%d) 的指标名称错误: %v，跳过该应用", app.Name, app.ID, err)
				c.error.Set(1)
				continue
			}
			names = metricNames
			c.names[app.ID] = names
			c.metricNamesLastScrape = time.Now()
			logrus.Debugf("指标名称列表更新于 %v", c.metricNamesLastScrape)
		} else {
			logrus.Debug("使用缓存的指标名称列表")
		}

		// 过滤指标名称
		var filteredNames []newrelic.MetricName
		for _, name := range names {
			if c.isNameFiltered(name.Name) {
				filteredNames = append(filteredNames, name)
			}
		}

		// 如果没有可用的指标名称，跳过
		if len(filteredNames) == 0 {
			logrus.Warnf("应用 %s 没有匹配的指标", app.Name)
			continue
		}

		logrus.Debugf("应用 %s 发现 %d 个指标", app.Name, len(filteredNames))

		// 获取指标数据 - 调整为匹配 API 方法签名
		metrics, err := c.api.GetMetricData(app.ID, filteredNames, from, to)
		if err != nil {
			logrus.Warnf("获取应用 %s (ID=%d) 的指标数据错误: %v，跳过该应用", app.Name, app.ID, err)
			c.error.Set(1)
			continue
		}

		// 检查是否有可用的指标数据
		if len(metrics) == 0 {
			logrus.Warnf("应用 %s 没有返回任何指标数据", app.Name)
			continue
		}
		
		logrus.Debugf("应用 %s 收到 %d 条指标数据", app.Name, len(metrics))

		// 发送指标数据
		for _, metric := range metrics {
			// 处理 timeslices 中的数据
			if len(metric.Timeslices) > 0 {
				for valueName, valueObj := range metric.Timeslices[0].Values {
					if !c.isValueFiltered(valueName) {
						// 转换为 float64
						if value, ok := valueObj.(float64); ok {
							ch <- Metric{
								App:   app.Name,
								Name:  valueName,
								Value: value,
								Label: metric.Name,
							}
						}
					}
				}
			}
		}
	}
	
	// 记录抓取完成日志
	logrus.Infof("抓取过程完成，耗时: %v", time.Since(startTime))
}

func (c *NewRelicMetricsCollector) receive(metrics <-chan Metric, promCh chan<- prometheus.Metric) {
	// 确保 metrics 映射已初始化
	if c.metrics == nil {
		c.metrics = make(map[string]*baseMetrics)
	}
	
	count := 0
	for m := range metrics {
		logrus.Debugf("处理指标: %s/%s/%s = %.2f", m.App, m.Label, m.Name, m.Value)
		
		// 获取或创建指标
		key := fmt.Sprintf("%s/%s/%s", m.App, m.Label, m.Name)
		metric, ok := c.metrics[key]
		if !ok {
			// 新指标
			metric = newMetric(
				NameSpace,
				fmt.Sprintf("%s_%s", m.Label, m.Name),
				fmt.Sprintf("NewRelic %s %s", m.Label, m.Name),
				[]string{"app"},
			)
			c.metrics[key] = metric
		}
		
		// 观察指标值
		if metric != nil {
			if err := metric.Observe(promCh, m.Value, m.App); err != nil {
				logrus.Warnf("观察指标失败: %v", err)
			} else {
				count++
			}
		} else {
			logrus.Warnf("创建指标失败: %s/%s/%s", m.App, m.Label, m.Name)
		}
	}
	
	logrus.Infof("成功导出 %d 个指标", count)
}

// 实现 Describe 方法，确保模拟数据模式下也能正确注册
func (c *NewRelicMetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	// 描述基本指标
	ch <- c.duration.Desc()
	ch <- c.error.Desc()
	ch <- c.totalScrapes.Desc()
	
	logrus.Debug("已描述基本指标")
}

// isNameFiltered 检查指标名称是否应该被过滤
func (c *NewRelicMetricsCollector) isNameFiltered(name string) bool {
	// 如果没有过滤器，保留所有
	if len(c.config.MetricFilters) == 0 {
		return true
	}
	
	// 检查名称是否匹配任何过滤器
	for _, filter := range c.config.MetricFilters {
		if strings.Contains(name, filter) {
			return true
		}
	}
	
	return false
}

// isValueFiltered 检查值名称是否应该被过滤
func (c *NewRelicMetricsCollector) isValueFiltered(name string) bool {
	// 如果没有过滤器，保留所有
	if len(c.config.ValueFilters) == 0 {
		return true
	}
	
	// 检查名称是否匹配任何过滤器
	for _, filter := range c.config.ValueFilters {
		if strings.Contains(name, filter) {
			return true
		}
	}
	
	return false
} 

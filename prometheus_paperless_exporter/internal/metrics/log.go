package metrics

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/hansmi/paperhooks/pkg/client"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

// LogCollectorConfig 定义日志收集器的配置选项
type LogCollectorConfig struct {
	// MaxConcurrentRequests 定义并发请求的最大数量
	MaxConcurrentRequests int

	// LogEntryBufferSize 定义日志条目缓冲区大小
	LogEntryBufferSize int

	// LogProcessingTimeout 定义单个日志处理的最大持续时间
	LogProcessingTimeout time.Duration

	// MetricsPrefix 为所有指标添加前缀
	MetricsPrefix string

	// EnableDuplicateDetection 启用重复日志条目检测
	EnableDuplicateDetection bool
}

// DefaultLogCollectorConfig 返回默认的日志收集器配置
func DefaultLogCollectorConfig() LogCollectorConfig {
	return LogCollectorConfig{
		MaxConcurrentRequests:    runtime.GOMAXPROCS(0),
		LogEntryBufferSize:      1000,
		LogProcessingTimeout:    30 * time.Second,
		MetricsPrefix:           "paperless_",
		EnableDuplicateDetection: true,
	}
}

// logClient 定义获取日志所需的接口
type logClient interface {
	ListLogs(context.Context) ([]string, *client.Response, error)
	GetLog(context.Context, string) ([]client.LogEntry, *client.Response, error)
}

// logPosition 记录日志文件中的位置信息
type logPosition struct {
	valid  bool
	time   time.Time
	module string
	level  string
}

// NewLogPosition 创建一个新的日志位置记录
func NewLogPosition(e client.LogEntry) logPosition {
	return logPosition{
		valid:  true,
		time:   e.Time,
		module: e.Module,
		level:  e.Level,
	}
}

// equal 检查日志条目是否与记录的位置匹配
func (p logPosition) equal(e client.LogEntry) bool {
	return p.valid && e.Time.Equal(p.time) && e.Module == p.module && e.Level == p.level
}

// logCollector 实现了 prometheus.Collector 接口，用于收集日志指标
type logCollector struct {
	cl     logClient
	config LogCollectorConfig

	mu sync.RWMutex

	// 日志位置状态
	seen map[string]logPosition

	// 指标向量
	totalVec       *prometheus.CounterVec
	errorCounter   prometheus.Counter
	processedFiles prometheus.Counter
	duplicates     prometheus.Counter
	processingTime prometheus.Histogram
}

// NewLogCollector 创建一个新的日志收集器实例
func NewLogCollector(cl logClient) *logCollector {
	return NewLogCollectorWithConfig(cl, DefaultLogCollectorConfig())
}

// NewLogCollectorWithConfig 使用自定义配置创建日志收集器
func NewLogCollectorWithConfig(cl logClient, config LogCollectorConfig) *logCollector {
	if config.MaxConcurrentRequests <= 0 {
		config.MaxConcurrentRequests = runtime.GOMAXPROCS(0)
	}

	if config.LogEntryBufferSize <= 0 {
		config.LogEntryBufferSize = 1000
	}

	if config.LogProcessingTimeout <= 0 {
		config.LogProcessingTimeout = 30 * time.Second
	}

	if config.MetricsPrefix == "" {
		config.MetricsPrefix = "paperless_"
	}

	return &logCollector{
		cl:     cl,
		config: config,

		seen: make(map[string]logPosition),

		totalVec: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: config.MetricsPrefix + "log_entries_total",
				Help: "Best-effort count of log entries.",
			},
			[]string{"name", "module", "level"},
		),

		errorCounter: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: config.MetricsPrefix + "log_collector_errors_total",
				Help: "Total number of errors encountered during log collection.",
			},
		),

		processedFiles: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: config.MetricsPrefix + "log_files_processed_total",
				Help: "Total number of log files processed.",
			},
		),

		duplicates: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: config.MetricsPrefix + "log_duplicates_total",
				Help: "Total number of duplicate log entries detected.",
			},
		),

		processingTime: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    config.MetricsPrefix + "log_processing_time_seconds",
				Help:    "Time spent processing log files.",
				Buckets: prometheus.DefBuckets,
			},
		),
	}
}

// Describe 发送所有指标描述符到提供的通道
func (c *logCollector) Describe(ch chan<- *prometheus.Desc) {
	c.totalVec.Describe(ch)
	c.errorCounter.Describe(ch)
	c.processedFiles.Describe(ch)
	c.duplicates.Describe(ch)
	c.processingTime.Describe(ch)
}

// processLogEntry 处理单个日志条目
func (c *logCollector) processLogEntry(name string, entry client.LogEntry) {
	labels := prometheus.Labels{
		"name":   name,
		"module": entry.Module,
		"level":  strings.ToLower(entry.Level),
	}

	c.totalVec.With(labels).Inc()
}

// collectOne 收集单个日志文件的指标
func (c *logCollector) collectOne(ctx context.Context, name string) error {
	startTime := time.Now()
	defer func() {
		c.processingTime.Observe(time.Since(startTime).Seconds())
	}()

	entries, _, err := c.cl.GetLog(ctx, name)
	if err != nil {
		var reqErr *client.RequestError
		if errors.As(err, &reqErr) && reqErr.StatusCode == http.StatusNotFound {
			return nil
		}
		return err
	}

	if len(entries) == 0 {
		return nil
	}

	c.processedFiles.Inc()

	// entryLabels := func(e client.LogEntry) prometheus.Labels {
	// 	return prometheus.Labels{
	// 		"name":   name,
	// 		"module": e.Module,
	// 		"level":  strings.ToLower(e.Level),
	// 	}
	// }

	c.mu.Lock()
	defer c.mu.Unlock()

	start := 0
	duplicateCount := 0

	if c.config.EnableDuplicateDetection {
		if seen, exists := c.seen[name]; exists && seen.valid {
			for idx, entry := range entries {
				if seen.equal(entry) {
					start = idx + 1
					duplicateCount = len(entries) - start
					break
				}
			}
		}
	}

	for _, entry := range entries[start:] {
		c.processLogEntry(name, entry)
	}

	if duplicateCount > 0 {
		c.duplicates.Add(float64(duplicateCount))
	}

	newest := entries[len(entries)-1]
	newest.Message = ""
	c.seen[name] = NewLogPosition(newest)

	return nil
}

// Collect 收集所有日志指标并发送到提供的通道
func (c *logCollector) Collect(ctx context.Context, 
	ch chan<- prometheus.Metric) error {

	names, _, err := c.cl.ListLogs(ctx)
	if err != nil {
		c.errorCounter.Inc()
		return fmt.Errorf("listing log names: %w", err)
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(c.config.MaxConcurrentRequests)

	for _, name := range names {
		name := name

		g.Go(func() error {

			entryCtx, cancel := context.WithTimeout(ctx, 
				c.config.LogProcessingTimeout)
				
			defer cancel()

			if err := c.collectOne(entryCtx, name); err != nil {
				c.errorCounter.Inc()
				return fmt.Errorf("log %s: %w", name, err)
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	c.totalVec.Collect(ch)
	c.errorCounter.Collect(ch)
	c.processedFiles.Collect(ch)
	c.duplicates.Collect(ch)
	c.processingTime.Collect(ch)

	return nil
}

package metrics

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hansmi/paperhooks/pkg/client"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// TaskCollectorConfig 定义任务收集器的配置选项
type TaskCollectorConfig struct {
	// Logger 用于记录日志
	Logger *zap.Logger

	// RequestTimeout 定义获取任务列表的最大持续时间
	RequestTimeout time.Duration

	// MetricsPrefix 为所有指标添加前缀
	MetricsPrefix string

	// EnableTaskCaching 启用任务缓存
	EnableTaskCaching bool

	// CacheTTL 定义任务信息的缓存时间
	CacheTTL time.Duration

	// MaxConcurrentRequests 定义并发请求的最大数量
	MaxConcurrentRequests int
}

// taskClient 定义获取任务所需的接口
type taskClient interface {
	ListTasks(context.Context) ([]client.Task, *client.Response, error)
}

// TaskCollector 实现了 prometheus.Collector 接口，用于收集任务指标
type TaskCollector struct {
	mu sync.RWMutex

	client taskClient
	config TaskCollectorConfig

	// 指标描述符
	infoDesc     *prometheus.Desc
	createdDesc  *prometheus.Desc
	doneDesc     *prometheus.Desc
	statusDesc   *prometheus.Desc
	filenameDesc *prometheus.Desc

	// 状态信息向量
	statusInfoVec *prometheus.GaugeVec

	// 内部状态
	cachedTasks    []client.Task
	lastFetchTime  time.Time
	lastFetchError error

	// 内部指标
	fetchDuration    prometheus.Histogram
	fetchErrors      prometheus.Counter
	tasksProcessed   prometheus.Counter
	cacheHits       prometheus.Counter
	statusChanges   *prometheus.CounterVec
}

// optionalTimestamp 将可选的时间戳转换为Unix时间戳(秒)
func optionalTimestamp(t *time.Time) float64 {
	if t == nil || t.IsZero() {
		return 0
	}
	return float64(t.UnixMilli()) / 1000
}

// NewTaskCollector 创建一个新的TaskCollector实例
func NewTaskCollector(cl taskClient) *TaskCollector {
	return NewTaskCollectorWithConfig(cl, TaskCollectorConfig{
		Logger:                zap.NewNop(),
		RequestTimeout:        30 * time.Second,
		MetricsPrefix:         "paperless_",
		EnableTaskCaching:     true,
		CacheTTL:             5 * time.Minute,
		MaxConcurrentRequests: 5,
	})
}

// NewTaskCollectorWithConfig 使用自定义配置创建TaskCollector
func NewTaskCollectorWithConfig(cl taskClient, config TaskCollectorConfig) *TaskCollector {
	if config.Logger == nil {
		config.Logger = zap.NewNop()
	}

	if config.RequestTimeout <= 0 {
		config.RequestTimeout = 30 * time.Second
	}

	if config.MetricsPrefix == "" {
		config.MetricsPrefix = "paperless_"
	}

	if config.CacheTTL <= 0 {
		config.CacheTTL = 5 * time.Minute
	}

	if config.MaxConcurrentRequests <= 0 {
		config.MaxConcurrentRequests = 5
	}

	return &TaskCollector{
		client: cl,
		config: config,

		infoDesc: prometheus.NewDesc(
			config.MetricsPrefix+"task_info",
			"Static information about a task.",
			[]string{"id", "task_id", "type"},
			nil,
		),

		createdDesc: prometheus.NewDesc(
			config.MetricsPrefix+"task_created_timestamp_seconds",
			"Number of seconds since 1970 of the task creation.",
			[]string{"id"},
			nil,
		),

		doneDesc: prometheus.NewDesc(
			config.MetricsPrefix+"task_done_timestamp_seconds",
			"Number of seconds since 1970 of when the task finished.",
			[]string{"id"},
			nil,
		),

		statusDesc: prometheus.NewDesc(
			config.MetricsPrefix+"task_status",
			"Task status.",
			[]string{"id", "status"},
			nil,
		),

		filenameDesc: prometheus.NewDesc(
			config.MetricsPrefix+"task_filename",
			"Filename associated with the task (if any).",
			[]string{"id", "filename"},
			nil,
		),

		statusInfoVec: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: config.MetricsPrefix + "task_status_info",
				Help: "Task status names.",
			},
			[]string{"status"},
		),

		fetchDuration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    config.MetricsPrefix + "task_fetch_duration_seconds",
				Help:    "Duration of task fetch operations",
				Buckets: []float64{0.1, 0.5, 1, 2, 5, 10},
			},
		),

		fetchErrors: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: config.MetricsPrefix + "task_fetch_errors_total",
				Help: "Total number of errors during task fetch",
			},
		),

		tasksProcessed: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: config.MetricsPrefix + "tasks_processed_total",
				Help: "Total number of tasks processed",
			},
		),

		cacheHits: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: config.MetricsPrefix + "task_cache_hits_total",
				Help: "Total number of cache hits for task information",
			},
		),

		statusChanges: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: config.MetricsPrefix + "task_status_changes_total",
				Help: "Total number of task status changes",
			},
			[]string{"from", "to"},
		),
	}
}

// ensureStatusInfo 确保状态信息向量中包含指定的状态
func (c *TaskCollector) ensureStatusInfo(s client.TaskStatus) string {
	status := strings.ToLower(s.String())
	c.statusInfoVec.With(prometheus.Labels{
		"status": status,
	}).Set(1)
	return status
}

// fetchTasks 实际获取任务列表
func (c *TaskCollector) fetchTasks(ctx context.Context) ([]client.Task, error) {
	timer := prometheus.NewTimer(c.fetchDuration)
	defer timer.ObserveDuration()

	tasks, _, err := c.client.ListTasks(ctx)
	if err != nil {
		c.fetchErrors.Inc()
		return nil, err
	}

	return tasks, nil
}

// maybeFetchTasks 检查是否需要获取任务列表
func (c *TaskCollector) maybeFetchTasks(ctx context.Context) ([]client.Task, error) {
	c.mu.RLock()

	// 检查缓存是否有效
	if c.config.EnableTaskCaching && time.Since(c.lastFetchTime) < c.config.CacheTTL && c.cachedTasks != nil {
		c.cacheHits.Inc()
		tasks := c.cachedTasks
		c.mu.RUnlock()
		return tasks, nil
	}

	c.mu.RUnlock()

	// 获取写锁
	c.mu.Lock()
	defer c.mu.Unlock()

	// 再次检查缓存，防止竞态条件
	if c.config.EnableTaskCaching && time.Since(c.lastFetchTime) < c.config.CacheTTL && c.cachedTasks != nil {
		c.cacheHits.Inc()
		return c.cachedTasks, nil
	}

	// 实际获取任务
	tasks, err := c.fetchTasks(ctx)
	if err != nil {
		c.lastFetchError = err
		return nil, err
	}

	c.cachedTasks = tasks
	c.lastFetchTime = time.Now()
	c.lastFetchError = nil

	return tasks, nil
}

// Describe 发送所有指标描述符到提供的通道
func (c *TaskCollector) Describe(ch chan<- *prometheus.Desc) {
	c.statusInfoVec.Describe(ch)
	ch <- c.infoDesc
	ch <- c.createdDesc
	ch <- c.doneDesc
	ch <- c.statusDesc
	ch <- c.filenameDesc

	ch <- c.fetchDuration.Desc()
	ch <- c.fetchErrors.Desc()
	ch <- c.tasksProcessed.Desc()
	ch <- c.cacheHits.Desc()
	c.statusChanges.Describe(ch)
}

// Collect 收集任务指标并发送到提供的通道
func (c *TaskCollector) Collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	if c.config.RequestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.config.RequestTimeout)
		defer cancel()
	}

	tasks, err := c.maybeFetchTasks(ctx)
	if err != nil {
		if c.config.Logger != nil {
			c.config.Logger.Error("Failed to fetch tasks",
				zap.Error(err),
				zap.Duration("cache_ttl", c.config.CacheTTL),
			)
		}
		return err
	}

	for _, task := range tasks {
		var filename string
		if task.TaskFileName != nil {
			filename = *task.TaskFileName
		}

		id := strconv.FormatInt(task.ID, 10)

		ch <- prometheus.MustNewConstMetric(
			c.infoDesc,
			prometheus.GaugeValue,
			1,
			id,
			task.TaskID,
			task.Type,
		)

		ch <- prometheus.MustNewConstMetric(
			c.createdDesc,
			prometheus.GaugeValue,
			optionalTimestamp(task.Created),
			id,
		)

		ch <- prometheus.MustNewConstMetric(
			c.doneDesc,
			prometheus.GaugeValue,
			optionalTimestamp(task.Done),
			id,
		)

		status := c.ensureStatusInfo(task.Status)
		ch <- prometheus.MustNewConstMetric(
			c.statusDesc,
			prometheus.GaugeValue,
			1,
			id,
			status,
		)

		ch <- prometheus.MustNewConstMetric(
			c.filenameDesc,
			prometheus.GaugeValue,
			1,
			id,
			filename,
		)

		c.tasksProcessed.Inc()
	}

	// 收集内部指标
	c.statusInfoVec.Collect(ch)
	ch <- c.fetchDuration
	ch <- c.fetchErrors
	ch <- c.tasksProcessed
	ch <- c.cacheHits
	c.statusChanges.Collect(ch)

	return nil
}

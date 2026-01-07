package metrics

import (
	"context"
	"strconv"
	"reflect"
	"sync"
	"time"

	"github.com/hansmi/paperhooks/pkg/client"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// GroupCollectorConfig 定义组收集器的配置选项
type GroupCollectorConfig struct {
	// Logger 用于记录日志
	Logger *zap.Logger

	// RequestTimeout 定义获取组列表的最大持续时间
	RequestTimeout time.Duration

	// MetricsPrefix 为所有指标添加前缀
	MetricsPrefix string

	// EnableCaching 启用组信息缓存
	EnableCaching bool

	// CacheTTL 定义组信息的缓存时间
	CacheTTL time.Duration

	// MaxConcurrentRequests 定义并发请求的最大数量
	MaxConcurrentRequests int

	// DetailedMetrics 启用详细指标收集
	DetailedMetrics bool
}

// groupClient 定义获取组所需的接口
type groupClient interface {
	ListGroups(context.Context, client.ListGroupsOptions) ([]client.Group, *client.Response, error)
}

// GroupCollector 实现了 prometheus.Collector 接口，用于收集组指标
type GroupCollector struct {
	mu sync.RWMutex

	client groupClient
	config GroupCollectorConfig

	// 指标描述符
	countDesc          *prometheus.Desc
	groupDetailsDesc   *prometheus.Desc
	lastUpdatedDesc    *prometheus.Desc
	permissionDesc     *prometheus.Desc

	// 内部状态
	cachedGroups      []client.Group
	cachedItemCount   int
	lastFetchTime     time.Time
	lastFetchError    error
	fetchInProgress   bool

	// 内部指标
	fetchDuration      prometheus.Histogram
	fetchErrors        prometheus.Counter
	cacheHits         prometheus.Counter
	groupsProcessed   prometheus.Counter
	permissionChanges *prometheus.CounterVec
}

// NewGroupCollector 创建一个新的 GroupCollector 实例
func NewGroupCollector(cl groupClient) *GroupCollector {
	return NewGroupCollectorWithConfig(cl, GroupCollectorConfig{
		Logger:                zap.NewNop(),
		RequestTimeout:        30 * time.Second,
		MetricsPrefix:         "paperless_",
		EnableCaching:         true,
		CacheTTL:             5 * time.Minute,
		MaxConcurrentRequests: 5,
		DetailedMetrics:       false,
	})
}

// NewGroupCollectorWithConfig 使用自定义配置创建 GroupCollector
func NewGroupCollectorWithConfig(cl groupClient, 
	config GroupCollectorConfig) *GroupCollector {

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

	collector := &GroupCollector{
		client: cl,
		config: config,

		countDesc: prometheus.NewDesc(
			config.MetricsPrefix+"groups_total",
			"Number of user groups.",
			nil, 
			nil,
		),

		fetchDuration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    config.MetricsPrefix+"group_fetch_duration_seconds",
				Help:    "Duration of group fetch operations",
				Buckets: []float64{0.1, 0.5, 1, 2, 5},
			},
		),

		fetchErrors: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: config.MetricsPrefix+"group_fetch_errors_total",
				Help: "Total number of errors during group fetch",
			},
		),

		cacheHits: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: config.MetricsPrefix+"group_cache_hits_total",
				Help: "Total number of cache hits for group information",
			},
		),

		groupsProcessed: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: config.MetricsPrefix+"groups_processed_total",
				Help: "Total number of groups processed",
			},
		),
	}

	if config.DetailedMetrics {
		collector.groupDetailsDesc = prometheus.NewDesc(
			config.MetricsPrefix+"group_details",
			"Detailed information about user groups.",
			[]string{"id", "name"}, 
			nil,
		)

		collector.lastUpdatedDesc = prometheus.NewDesc(
			config.MetricsPrefix+"group_last_updated_timestamp_seconds",
			"Unix timestamp of when the group was last updated.",
			[]string{"id"}, 
			nil,
		)

		collector.permissionDesc = prometheus.NewDesc(
			config.MetricsPrefix+"group_permissions",
			"Group permissions information.",
			[]string{"id", "permission"}, 
			nil,
		)

		collector.permissionChanges = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: config.MetricsPrefix+"group_permission_changes_total",
				Help: "Total number of permission changes detected",
			},
			[]string{"type"},
		)
	}

	return collector
}

// fetchGroups 实际获取组列表
func (c *GroupCollector) fetchGroups(ctx context.Context) ([]client.Group, 
	int, 
	error) {
		
	timer := prometheus.NewTimer(c.fetchDuration)
	defer timer.ObserveDuration()

	groups, resp, err := c.client.ListGroups(ctx, 
		client.ListGroupsOptions{})

	if err != nil {
		c.fetchErrors.Inc()
		return nil, 0, err
	}

	itemCount := 0
	if resp.ItemCount != client.ItemCountUnknown {
		itemCount = int(resp.ItemCount)
	}

	return groups, itemCount, nil
}

// maybeFetchGroups 检查是否需要获取组列表
func (c *GroupCollector) maybeFetchGroups(ctx context.Context) ([]client.Group, 
	int, 
	error) {

	c.mu.RLock()

	// 检查缓存是否有效
	if c.config.EnableCaching && time.Since(c.lastFetchTime) < c.config.CacheTTL && c.cachedGroups != nil {
		c.cacheHits.Inc()
		groups := c.cachedGroups
		count := c.cachedItemCount
		c.mu.RUnlock()
		return groups, count, nil
	}

	c.mu.RUnlock()

	// 获取写锁
	c.mu.Lock()
	defer c.mu.Unlock()

	// 再次检查缓存，防止竞态条件
	if c.config.EnableCaching && time.Since(c.lastFetchTime) < c.config.CacheTTL && c.cachedGroups != nil {
		c.cacheHits.Inc()
		return c.cachedGroups, c.cachedItemCount, nil
	}

	// 实际获取组
	groups, itemCount, err := c.fetchGroups(ctx)
	if err != nil {
		c.lastFetchError = err
		return nil, 0, err
	}

	c.cachedGroups = groups
	c.cachedItemCount = itemCount
	c.lastFetchTime = time.Now()
	c.lastFetchError = nil

	return groups, itemCount, nil
}

// Describe 发送所有指标描述符到提供的通道
func (c *GroupCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.countDesc

	if c.groupDetailsDesc != nil {
		ch <- c.groupDetailsDesc
		ch <- c.lastUpdatedDesc
		ch <- c.permissionDesc
	}

	c.fetchDuration.Describe(ch)
	c.fetchErrors.Describe(ch)
	c.cacheHits.Describe(ch)
	c.groupsProcessed.Describe(ch)

	if c.permissionChanges != nil {
		c.permissionChanges.Describe(ch)
	}
}

func getUpdateTime(group client.Group) time.Time {
    val := reflect.ValueOf(group)
    for _, fieldName := range []string{"UpdatedAt", 
		"ModifiedAt", 
		"LastUpdated", 
		"UpdateTime"} {

        field := val.FieldByName(fieldName)
        if field.IsValid() && !field.IsZero() {
            return field.Interface().(time.Time)
        }
    }
    return time.Time{}
}

// Collect 收集组指标并发送到提供的通道
func (c *GroupCollector) Collect(ctx context.Context, 
	ch chan<- prometheus.Metric) error {
	
    if c.config.RequestTimeout > 0 {
        var cancel context.CancelFunc

        ctx, cancel = context.WithTimeout(ctx, 
			c.config.RequestTimeout)

        defer cancel()
    }

    groups, itemCount, err := c.maybeFetchGroups(ctx)
    if err != nil {
        return err
    }

    // 发送组数量指标
    if itemCount > 0 {
        ch <- prometheus.MustNewConstMetric(c.countDesc, 
			prometheus.GaugeValue, 
			float64(itemCount))
    }

    // 收集详细指标
    if c.config.DetailedMetrics {
        for _, group := range groups {
            groupID := strconv.FormatInt(group.ID, 10)

            // 组详细信息
            if c.groupDetailsDesc != nil {
                ch <- prometheus.MustNewConstMetric(
                    c.groupDetailsDesc,
                    prometheus.GaugeValue,
                    1,
                    groupID,
                    group.Name,
                )
            }

			if c.lastUpdatedDesc != nil {
				updatedTime := getUpdateTime(group)
				if !updatedTime.IsZero() {
					ch <- prometheus.MustNewConstMetric(
						c.lastUpdatedDesc,
						prometheus.GaugeValue,
						float64(updatedTime.Unix()),
						groupID,
					)
				}
			}

            c.groupsProcessed.Inc()
        }
    }

    // 收集内部指标
    c.fetchDuration.Collect(ch)
    c.fetchErrors.Collect(ch)
    c.cacheHits.Collect(ch)
    c.groupsProcessed.Collect(ch)

    if c.permissionChanges != nil {
        c.permissionChanges.Collect(ch)
    }

    return nil
}
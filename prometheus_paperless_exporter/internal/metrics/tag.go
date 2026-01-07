package metrics

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/hansmi/paperhooks/pkg/client"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// TagInfo 封装标签信息
type TagInfo struct {
	ID            int64
	Name          string
	Slug          string
	DocumentCount int64
	IsInboxTag    bool
	LastUpdated   time.Time
}

// TagCache 标签信息缓存
type TagCache struct {
	mu    sync.RWMutex
	items map[int64]TagInfo
}

type TagClient interface {
	ListAllTags(context.Context, 
		client.ListTagsOptions, 
		func(context.Context, client.Tag) error) error
}

func NewTagCache() *TagCache {
	return &TagCache{
		items: make(map[int64]TagInfo),
	}
}

func (c *TagCache) Update(info TagInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	info.LastUpdated = time.Now()
	c.items[info.ID] = info
}

func (c *TagCache) Get(id int64) (TagInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	item, exists := c.items[id]
	return item, exists
}

func (c *TagCache) Purge(olderThan time.Duration) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	count := 0
	cutoff := time.Now().Add(-olderThan)
	
	for id, item := range c.items {
		if item.LastUpdated.Before(cutoff) {
			delete(c.items, id)
			count++
		}
	}
	return count
}

// TagMetrics 封装指标相关操作
type TagMetrics struct {
	infoDesc                 *prometheus.Desc
	docCountDesc             *prometheus.Desc
	inboxDesc                *prometheus.Desc
	cacheSizeDesc            *prometheus.Desc
	collectErrors            prometheus.Counter
	collectionDuration       prometheus.Histogram
	cacheHits                prometheus.Counter
	cacheMisses              prometheus.Counter
	cachePurges              prometheus.Counter
}

func NewTagMetrics() *TagMetrics {
	return &TagMetrics{
		infoDesc: prometheus.NewDesc(
			"paperless_tag_info",
			"Static information about a tag.",
			[]string{"id", "name", "slug"}, nil,
		),
		docCountDesc: prometheus.NewDesc(
			"paperless_tag_document_count",
			"Number of documents associated with a tag.",
			[]string{"id"}, nil,
		),
		inboxDesc: prometheus.NewDesc(
			"paperless_tag_inbox",
			"Whether the tag is marked as an inbox tag.",
			[]string{"id"}, nil,
		),
		cacheSizeDesc: prometheus.NewDesc(
			"paperless_tag_cache_size",
			"Number of tags cached.",
			nil, nil,
		),
		collectErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "paperless_tag_collect_errors_total",
			Help: "Total number of errors encountered while collecting tag metrics.",
		}),
		collectionDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "paperless_tag_collection_duration_seconds",
			Help:    "Time taken to collect tag metrics.",
			Buckets: prometheus.DefBuckets,
		}),
		cacheHits: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "paperless_tag_cache_hits_total",
			Help: "Total number of cache hits.",
		}),
		cacheMisses: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "paperless_tag_cache_misses_total",
			Help: "Total number of cache misses.",
		}),
		cachePurges: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "paperless_tag_cache_purges_total",
			Help: "Total number of cache purges.",
		}),
	}
}

// TagCollector 主收集器实现
type TagCollector struct {
	cl      TagClient
	cache   *TagCache
	metrics *TagMetrics
	logger  *zap.Logger
	mu      sync.Mutex
	
	// 配置项
	cacheTTL time.Duration
	enabled  bool
}

// NewTagCollector 创建新的收集器实例
func NewTagCollector(cl TagClient) *TagCollector {
	return &TagCollector{
		cl:        cl,
		cache:     NewTagCache(),
		metrics:   NewTagMetrics(),
		logger:    zap.NewNop(),
		cacheTTL:  5 * time.Minute,
		enabled:   true,
	}
}

// WithLogger 设置日志记录器
func (c *TagCollector) WithLogger(logger *zap.Logger) *TagCollector {
	c.logger = logger
	return c
}

// WithCacheTTL 设置缓存TTL
func (c *TagCollector) WithCacheTTL(ttl time.Duration) *TagCollector {
	c.cacheTTL = ttl
	return c
}

// Enable 启用收集器
func (c *TagCollector) Enable() *TagCollector {
	c.enabled = true
	return c
}

// Disable 禁用收集器
func (c *TagCollector) Disable() *TagCollector {
	c.enabled = false
	return c
}

// Describe 实现prometheus.Collector接口
func (c *TagCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.metrics.infoDesc
	ch <- c.metrics.docCountDesc
	ch <- c.metrics.inboxDesc
	ch <- c.metrics.cacheSizeDesc
	c.metrics.collectErrors.Describe(ch)
	c.metrics.collectionDuration.Describe(ch)
	c.metrics.cacheHits.Describe(ch)
	c.metrics.cacheMisses.Describe(ch)
	c.metrics.cachePurges.Describe(ch)
}

// Collect 实现prometheus.Collector接口
func (c *TagCollector) Collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	if !c.enabled {
		c.logger.Debug("Tag collector is disabled")
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 清理过期缓存
	purged := c.cache.Purge(c.cacheTTL)
	if purged > 0 {
		c.logger.Debug("Purged expired cache entries",
			zap.Int("count", purged))
		c.metrics.cachePurges.Add(float64(purged))
	}

	// 记录收集耗时
	timer := prometheus.NewTimer(c.metrics.collectionDuration)
	defer timer.ObserveDuration()

	if err := c.collectTags(ctx, ch); err != nil {
		c.logger.Error("Failed to collect tag metrics",
			zap.Error(err))
		c.metrics.collectErrors.Inc()
		return err
	}

	// 报告缓存大小
	ch <- prometheus.MustNewConstMetric(
		c.metrics.cacheSizeDesc,
		prometheus.GaugeValue,
		float64(len(c.cache.items)),
	)

	// 报告指标
	ch <- c.metrics.collectErrors
	c.metrics.collectionDuration.Collect(ch)
	c.metrics.cacheHits.Collect(ch)
	c.metrics.cacheMisses.Collect(ch)
	c.metrics.cachePurges.Collect(ch)

	return nil
}

// collectTags 实际收集标签指标
func (c *TagCollector) collectTags(ctx context.Context, ch chan<- prometheus.Metric) error {
	var opts client.ListTagsOptions
	opts.Ordering.Field = "name"

	return c.cl.ListAllTags(ctx, opts, func(_ context.Context, tag client.Tag) error {
		info := TagInfo{
			ID:            tag.ID,
			Name:          tag.Name,
			Slug:          tag.Slug,
			DocumentCount: tag.DocumentCount,
			IsInboxTag:    tag.IsInboxTag,
		}

		c.cache.Update(info)

		id := strconv.FormatInt(info.ID, 10)

		ch <- prometheus.MustNewConstMetric(
			c.metrics.infoDesc,
			prometheus.GaugeValue,
			1,
			id,
			info.Name,
			info.Slug,
		)

		ch <- prometheus.MustNewConstMetric(
			c.metrics.docCountDesc,
			prometheus.GaugeValue,
			float64(info.DocumentCount),
			id,
		)

		isInboxTag := 0.0
		if info.IsInboxTag {
			isInboxTag = 1.0
		}

		ch <- prometheus.MustNewConstMetric(
			c.metrics.inboxDesc,
			prometheus.GaugeValue,
			isInboxTag,
			id,
		)

		return nil
	})
}

// GetTagInfo 获取特定标签信息
func (c *TagCollector) GetTagInfo(id int64) (TagInfo, bool) {
	if info, exists := c.cache.Get(id); exists {
		c.metrics.cacheHits.Inc()
		return info, true
	}
	c.metrics.cacheMisses.Inc()
	return TagInfo{}, false
}


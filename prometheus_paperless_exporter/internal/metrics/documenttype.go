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

// DocumentTypeInfo 封装文档类型信息
type DocumentTypeInfo struct {
	ID            int64
	Name          string
	Slug          string
	DocumentCount int64
	LastUpdated   time.Time
}

// DocumentTypeCache 文档类型信息缓存
type DocumentTypeCache struct {
	mu    sync.RWMutex
	items map[int64]DocumentTypeInfo
}

type documentTypeClient interface {
	ListAllDocumentTypes(context.Context, 
		client.ListDocumentTypesOptions, 
		func(context.Context, client.DocumentType) error) error
}

func NewDocumentTypeCache() *DocumentTypeCache {
	return &DocumentTypeCache{
		items: make(map[int64]DocumentTypeInfo),
	}
}

func (c *DocumentTypeCache) Update(info DocumentTypeInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	info.LastUpdated = time.Now()
	c.items[info.ID] = info
}

func (c *DocumentTypeCache) Get(id int64) (DocumentTypeInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	item, exists := c.items[id]
	return item, exists
}

func (c *DocumentTypeCache) Purge(olderThan time.Duration) int {
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

// DocumentTypeMetrics 封装指标相关操作
type DocumentTypeMetrics struct {
	infoDesc                 *prometheus.Desc
	docCountDesc             *prometheus.Desc
	cacheSizeDesc            *prometheus.Desc
	collectErrors            prometheus.Counter
	collectionDuration       prometheus.Histogram
	cacheHits                prometheus.Counter
	cacheMisses              prometheus.Counter
	cachePurges              prometheus.Counter
}

func NewDocumentTypeMetrics() *DocumentTypeMetrics {
	return &DocumentTypeMetrics{
		infoDesc: prometheus.NewDesc(
			"paperless_document_type_info",
			"Static information about a document type.",
			[]string{"id", "name", "slug"}, 
			nil,
		),
		docCountDesc: prometheus.NewDesc(
			"paperless_document_type_document_count",
			"Number of documents associated with a document type.",
			[]string{"id"}, 
			nil,
		),
		cacheSizeDesc: prometheus.NewDesc(
			"paperless_document_type_cache_size",
			"Number of document types cached.",
			nil, 
			nil,
		),
		collectErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "paperless_document_type_collect_errors_total",
			Help: "Total number of errors encountered while collecting document type metrics.",
		}),
		collectionDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "paperless_document_type_collection_duration_seconds",
			Help:    "Time taken to collect document type metrics.",
			Buckets: prometheus.DefBuckets,
		}),
		cacheHits: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "paperless_document_type_cache_hits_total",
			Help: "Total number of cache hits.",
		}),
		cacheMisses: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "paperless_document_type_cache_misses_total",
			Help: "Total number of cache misses.",
		}),
		cachePurges: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "paperless_document_type_cache_purges_total",
			Help: "Total number of cache purges.",
		}),
	}
}

// DocumentTypeCollector 主收集器实现
type DocumentTypeCollector struct {
	cl      documentTypeClient
	cache   *DocumentTypeCache
	metrics *DocumentTypeMetrics
	logger  *zap.Logger
	mu      sync.Mutex
	
	// 配置项
	cacheTTL time.Duration
	enabled  bool
}

// NewDocumentTypeCollector 创建新的收集器实例
func NewDocumentTypeCollector(cl documentTypeClient) *DocumentTypeCollector {
	return &DocumentTypeCollector{
		cl:        cl,
		cache:     NewDocumentTypeCache(),
		metrics:   NewDocumentTypeMetrics(),
		logger:    zap.NewNop(),
		cacheTTL:  5 * time.Minute,
		enabled:   true,
	}
}

// WithLogger 设置日志记录器
func (c *DocumentTypeCollector) WithLogger(logger *zap.Logger) *DocumentTypeCollector {
	c.logger = logger
	return c
}

// WithCacheTTL 设置缓存TTL
func (c *DocumentTypeCollector) WithCacheTTL(ttl time.Duration) *DocumentTypeCollector {
	c.cacheTTL = ttl
	return c
}

// Enable 启用收集器
func (c *DocumentTypeCollector) Enable() *DocumentTypeCollector {
	c.enabled = true
	return c
}

// Disable 禁用收集器
func (c *DocumentTypeCollector) Disable() *DocumentTypeCollector {
	c.enabled = false
	return c
}

// Describe 实现prometheus.Collector接口
func (c *DocumentTypeCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.metrics.infoDesc
	ch <- c.metrics.docCountDesc
	ch <- c.metrics.cacheSizeDesc
	c.metrics.collectErrors.Describe(ch)
	c.metrics.collectionDuration.Describe(ch)
	c.metrics.cacheHits.Describe(ch)
	c.metrics.cacheMisses.Describe(ch)
	c.metrics.cachePurges.Describe(ch)
}

// Collect 实现prometheus.Collector接口
func (c *DocumentTypeCollector) Collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	if !c.enabled {
		c.logger.Debug("Document type collector is disabled")
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

	if err := c.collectDocumentTypes(ctx, ch); err != nil {
		c.logger.Error("Failed to collect document type metrics",
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

// collectDocumentTypes 实际收集文档类型指标
func (c *DocumentTypeCollector) collectDocumentTypes(ctx context.Context, ch chan<- prometheus.Metric) error {
	var opts client.ListDocumentTypesOptions
	opts.Ordering.Field = "name"

	return c.cl.ListAllDocumentTypes(ctx, opts, func(_ context.Context, doctype client.DocumentType) error {
		info := DocumentTypeInfo{
			ID:            doctype.ID,
			Name:          doctype.Name,
			Slug:          doctype.Slug,
			DocumentCount: doctype.DocumentCount,
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

		return nil
	})
}

// GetDocumentTypeInfo 获取特定文档类型信息
func (c *DocumentTypeCollector) GetDocumentTypeInfo(id int64) (DocumentTypeInfo, bool) {
	if info, exists := c.cache.Get(id); exists {
		c.metrics.cacheHits.Inc()
		return info, true
	}
	c.metrics.cacheMisses.Inc()
	return DocumentTypeInfo{}, false
}


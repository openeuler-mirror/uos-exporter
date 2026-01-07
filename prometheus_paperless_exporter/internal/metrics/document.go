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

// DocumentStats 封装文档统计信息
type DocumentStats struct {
	TotalCount     int64
	ByCorrespondent map[int64]int64
	ByDocumentType  map[int64]int64
	ByTag          map[int64]int64
	LastUpdated    time.Time
}

// DocumentCache 文档信息缓存
type DocumentCache struct {
	mu    sync.RWMutex
	stats DocumentStats
}

type documentClient interface {
	ListDocuments(context.Context, 
		client.ListDocumentsOptions) ([]client.Document, 
			*client.Response, error)
}

func NewDocumentCache() *DocumentCache {
	return &DocumentCache{
		stats: DocumentStats{
			ByCorrespondent: make(map[int64]int64),
			ByDocumentType:  make(map[int64]int64),
			ByTag:          make(map[int64]int64),
		},
	}
}

func (c *DocumentCache) Update(stats DocumentStats) {
	c.mu.Lock()
	defer c.mu.Unlock()
	stats.LastUpdated = time.Now()
	c.stats = stats
}

func (c *DocumentCache) Get() DocumentStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

func (c *DocumentCache) IsStale(threshold time.Duration) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Since(c.stats.LastUpdated) > threshold
}

// DocumentMetrics 封装指标相关操作
type DocumentMetrics struct {
	totalCountDesc      *prometheus.Desc
	correspondentDesc   *prometheus.Desc
	documentTypeDesc    *prometheus.Desc
	tagDesc            *prometheus.Desc
	cacheStaleDesc      *prometheus.Desc
	collectErrors       prometheus.Counter
	collectionDuration  prometheus.Histogram
	cacheUpdates       prometheus.Counter
}

func NewDocumentMetrics() *DocumentMetrics {
	return &DocumentMetrics{
		totalCountDesc: prometheus.NewDesc(
			"paperless_documents_total",
			"Total number of documents.",
			nil, 
			nil,
		),
		correspondentDesc: prometheus.NewDesc(
			"paperless_documents_by_correspondent",
			"Number of documents by correspondent.",
			[]string{"correspondent_id"}, 
			nil,
		),
		documentTypeDesc: prometheus.NewDesc(
			"paperless_documents_by_type",
			"Number of documents by document type.",
			[]string{"document_type_id"}, 
			nil,
		),
		tagDesc: prometheus.NewDesc(
			"paperless_documents_by_tag",
			"Number of documents by tag.",
			[]string{"tag_id"}, 
			nil,
		),
		cacheStaleDesc: prometheus.NewDesc(
			"paperless_documents_cache_stale",
			"Indicates if document cache is stale (1) or fresh (0).",
			nil, 
			nil,
		),
		collectErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "paperless_document_collect_errors_total",
			Help: "Total number of errors encountered while collecting document metrics.",
		}),
		collectionDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "paperless_document_collection_duration_seconds",
			Help:    "Time taken to collect document metrics.",
			Buckets: prometheus.DefBuckets,
		}),
		cacheUpdates: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "paperless_document_cache_updates_total",
			Help: "Total number of document cache updates.",
		}),
	}
}

// DocumentCollector 主收集器实现
type DocumentCollector struct {
	cl      documentClient
	cache   *DocumentCache
	metrics *DocumentMetrics
	logger  *zap.Logger
	mu      sync.Mutex
	
	// 配置项
	cacheTTL      time.Duration
	enabled       bool
	collectDetail bool
}

// NewDocumentCollector 创建新的收集器实例
func NewDocumentCollector(cl documentClient) *DocumentCollector {
	return &DocumentCollector{
		cl:          cl,
		cache:       NewDocumentCache(),
		metrics:     NewDocumentMetrics(),
		logger:      zap.NewNop(),
		cacheTTL:    5 * time.Minute,
		enabled:     true,
		collectDetail: true,
	}
}

// WithLogger 设置日志记录器
func (c *DocumentCollector) WithLogger(logger *zap.Logger) *DocumentCollector {
	c.logger = logger
	return c
}

// WithCacheTTL 设置缓存TTL
func (c *DocumentCollector) WithCacheTTL(ttl time.Duration) *DocumentCollector {
	c.cacheTTL = ttl
	return c
}

// Enable 启用收集器
func (c *DocumentCollector) Enable() *DocumentCollector {
	c.enabled = true
	return c
}

// Disable 禁用收集器
func (c *DocumentCollector) Disable() *DocumentCollector {
	c.enabled = false
	return c
}

// EnableDetailCollection 启用详细收集
func (c *DocumentCollector) EnableDetailCollection() *DocumentCollector {
	c.collectDetail = true
	return c
}

// DisableDetailCollection 禁用详细收集
func (c *DocumentCollector) DisableDetailCollection() *DocumentCollector {
	c.collectDetail = false
	return c
}

// Describe 实现prometheus.Collector接口
func (c *DocumentCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.metrics.totalCountDesc
	ch <- c.metrics.correspondentDesc
	ch <- c.metrics.documentTypeDesc
	ch <- c.metrics.tagDesc
	ch <- c.metrics.cacheStaleDesc
	c.metrics.collectErrors.Describe(ch)
	c.metrics.collectionDuration.Describe(ch)
	c.metrics.cacheUpdates.Describe(ch)
}

// Collect 实现prometheus.Collector接口
func (c *DocumentCollector) Collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	if !c.enabled {
		c.logger.Debug("Document collector is disabled")
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 记录收集耗时
	timer := prometheus.NewTimer(c.metrics.collectionDuration)
	defer timer.ObserveDuration()

	// 检查缓存是否过期
	stale := c.cache.IsStale(c.cacheTTL)
	ch <- prometheus.MustNewConstMetric(
		c.metrics.cacheStaleDesc,
		prometheus.GaugeValue,
		boolToFloat64(stale),
	)

	if stale || c.collectDetail {
		if err := c.collectDocumentStats(ctx, ch); err != nil {
			c.logger.Error("Failed to collect document metrics",
				zap.Error(err))
			c.metrics.collectErrors.Inc()
			return err
		}
	}

	// 报告缓存中的指标
	c.reportCachedMetrics(ch)

	// 报告指标
	c.metrics.collectErrors.Collect(ch)
	c.metrics.collectionDuration.Collect(ch)
	c.metrics.cacheUpdates.Collect(ch)

	return nil
}

// collectDocumentStats 收集文档统计信息
func (c *DocumentCollector) collectDocumentStats(ctx context.Context, ch chan<- prometheus.Metric) error {
	stats := DocumentStats{
		ByCorrespondent: make(map[int64]int64),
		ByDocumentType:  make(map[int64]int64),
		ByTag:          make(map[int64]int64),
	}

	// 获取文档总数
	_, response, err := c.cl.ListDocuments(ctx, client.ListDocumentsOptions{})
	if err != nil {
		return err
	}

	if response.ItemCount != client.ItemCountUnknown {
		stats.TotalCount = response.ItemCount
	}

	// 如果启用详细收集，获取分类统计
	if c.collectDetail {
		// 这里可以添加按correspondent、document type和tag的分类统计逻辑
		// 实际实现需要调用相应的API端点
	}

	c.cache.Update(stats)
	c.metrics.cacheUpdates.Inc()

	return nil
}

// reportCachedMetrics 报告缓存中的指标
func (c *DocumentCollector) reportCachedMetrics(ch chan<- prometheus.Metric) {
	stats := c.cache.Get()

	ch <- prometheus.MustNewConstMetric(
		c.metrics.totalCountDesc,
		prometheus.GaugeValue,
		float64(stats.TotalCount),
	)

	for id, count := range stats.ByCorrespondent {
		ch <- prometheus.MustNewConstMetric(
			c.metrics.correspondentDesc,
			prometheus.GaugeValue,
			float64(count),
			int64ToStr(id),
		)
	}

	for id, count := range stats.ByDocumentType {
		ch <- prometheus.MustNewConstMetric(
			c.metrics.documentTypeDesc,
			prometheus.GaugeValue,
			float64(count),
			int64ToStr(id),
		)
	}

	for id, count := range stats.ByTag {
		ch <- prometheus.MustNewConstMetric(
			c.metrics.tagDesc,
			prometheus.GaugeValue,
			float64(count),
			int64ToStr(id),
		)
	}
}

// GetTotalCount 获取文档总数
func (c *DocumentCollector) GetTotalCount() int64 {
	return c.cache.Get().TotalCount
}

// GetCountByCorrespondent 获取按correspondent分类的文档数
func (c *DocumentCollector) GetCountByCorrespondent(id int64) int64 {
	return c.cache.Get().ByCorrespondent[id]
}

// GetCountByDocumentType 获取按document type分类的文档数
func (c *DocumentCollector) GetCountByDocumentType(id int64) int64 {
	return c.cache.Get().ByDocumentType[id]
}

// GetCountByTag 获取按tag分类的文档数
func (c *DocumentCollector) GetCountByTag(id int64) int64 {
	return c.cache.Get().ByTag[id]
}

// helper functions
func boolToFloat64(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func int64ToStr(i int64) string {
	return strconv.FormatInt(i, 10)
}

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

// StoragePathInfo 封装存储路径信息
type StoragePathInfo struct {
	ID            int64
	Name          string
	Slug          string
	DocumentCount int64
	Path          string
	LastUpdated   time.Time
}

// StoragePathCache 存储路径信息缓存
type StoragePathCache struct {
	mu    sync.RWMutex
	items map[int64]StoragePathInfo
}

type storagePathClient interface {
	ListAllStoragePaths(context.Context, 
		client.ListStoragePathsOptions, 
		func(context.Context, client.StoragePath) error) error
}

func NewStoragePathCache() *StoragePathCache {
	return &StoragePathCache{
		items: make(map[int64]StoragePathInfo),
	}
}

func (c *StoragePathCache) Update(info StoragePathInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	info.LastUpdated = time.Now()
	c.items[info.ID] = info
}

func (c *StoragePathCache) Get(id int64) (StoragePathInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	item, exists := c.items[id]
	return item, exists
}

func (c *StoragePathCache) Purge(olderThan time.Duration) int {
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

// StoragePathMetrics 封装指标相关操作
type StoragePathMetrics struct {
	infoDesc                 *prometheus.Desc
	docCountDesc             *prometheus.Desc
	pathDesc                 *prometheus.Desc
	cacheSizeDesc            *prometheus.Desc
	collectErrors            prometheus.Counter
	collectionDuration       prometheus.Histogram
	cacheHits                prometheus.Counter
	cacheMisses              prometheus.Counter
	cachePurges              prometheus.Counter
}

func NewStoragePathMetrics() *StoragePathMetrics {
	return &StoragePathMetrics{
		infoDesc: prometheus.NewDesc(
			"paperless_storage_path_info",
			"Static information about a storage path.",
			[]string{"id", "name", "slug"}, nil,
		),
		docCountDesc: prometheus.NewDesc(
			"paperless_storage_path_document_count",
			"Number of documents associated with a storage path.",
			[]string{"id"}, nil,
		),
		pathDesc: prometheus.NewDesc(
			"paperless_storage_path_path",
			"Filesystem path of the storage location.",
			[]string{"id", "path"}, nil,
		),
		cacheSizeDesc: prometheus.NewDesc(
			"paperless_storage_path_cache_size",
			"Number of storage paths cached.",
			nil, nil,
		),
		collectErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "paperless_storage_path_collect_errors_total",
			Help: "Total number of errors encountered while collecting storage path metrics.",
		}),
		collectionDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "paperless_storage_path_collection_duration_seconds",
			Help:    "Time taken to collect storage path metrics.",
			Buckets: prometheus.DefBuckets,
		}),
		cacheHits: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "paperless_storage_path_cache_hits_total",
			Help: "Total number of cache hits.",
		}),
		cacheMisses: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "paperless_storage_path_cache_misses_total",
			Help: "Total number of cache misses.",
		}),
		cachePurges: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "paperless_storage_path_cache_purges_total",
			Help: "Total number of cache purges.",
		}),
	}
}

// StoragePathCollector 主收集器实现
type StoragePathCollector struct {
	cl      storagePathClient
	cache   *StoragePathCache
	metrics *StoragePathMetrics
	logger  *zap.Logger
	mu      sync.Mutex
	
	// 配置项
	cacheTTL time.Duration
	enabled  bool
}

// NewStoragePathCollector 创建新的收集器实例
func NewStoragePathCollector(cl storagePathClient) *StoragePathCollector {
	return &StoragePathCollector{
		cl:        cl,
		cache:     NewStoragePathCache(),
		metrics:   NewStoragePathMetrics(),
		logger:    zap.NewNop(),
		cacheTTL:  5 * time.Minute,
		enabled:   true,
	}
}

// WithLogger 设置日志记录器
func (c *StoragePathCollector) WithLogger(logger *zap.Logger) *StoragePathCollector {
	c.logger = logger
	return c
}

// WithCacheTTL 设置缓存TTL
func (c *StoragePathCollector) WithCacheTTL(ttl time.Duration) *StoragePathCollector {
	c.cacheTTL = ttl
	return c
}

// Enable 启用收集器
func (c *StoragePathCollector) Enable() *StoragePathCollector {
	c.enabled = true
	return c
}

// Disable 禁用收集器
func (c *StoragePathCollector) Disable() *StoragePathCollector {
	c.enabled = false
	return c
}

// Describe 实现prometheus.Collector接口
func (c *StoragePathCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.metrics.infoDesc
	ch <- c.metrics.docCountDesc
	ch <- c.metrics.pathDesc
	ch <- c.metrics.cacheSizeDesc
	c.metrics.collectErrors.Describe(ch)
	c.metrics.collectionDuration.Describe(ch)
	c.metrics.cacheHits.Describe(ch)
	c.metrics.cacheMisses.Describe(ch)
	c.metrics.cachePurges.Describe(ch)
}

// Collect 实现prometheus.Collector接口
func (c *StoragePathCollector) Collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	if !c.enabled {
		c.logger.Debug("Storage path collector is disabled")
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

	if err := c.collectStoragePaths(ctx, ch); err != nil {
		c.logger.Error("Failed to collect storage path metrics",
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

// collectStoragePaths 实际收集存储路径指标
func (c *StoragePathCollector) collectStoragePaths(ctx context.Context, ch chan<- prometheus.Metric) error {
	var opts client.ListStoragePathsOptions
	opts.Ordering.Field = "name"

	return c.cl.ListAllStoragePaths(ctx, opts, func(_ context.Context, sp client.StoragePath) error {
		info := StoragePathInfo{
			ID:            sp.ID,
			Name:          sp.Name,
			Slug:          sp.Slug,
			DocumentCount: sp.DocumentCount,
			// Path:          sp.Path,
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

		ch <- prometheus.MustNewConstMetric(
			c.metrics.pathDesc,
			prometheus.GaugeValue,
			1,
			id,
			info.Path,
		)

		return nil
	})
}

// GetStoragePathInfo 获取特定存储路径信息
func (c *StoragePathCollector) GetStoragePathInfo(id int64) (StoragePathInfo, bool) {
	if info, exists := c.cache.Get(id); exists {
		c.metrics.cacheHits.Inc()
		return info, true
	}
	c.metrics.cacheMisses.Inc()
	return StoragePathInfo{}, false
}

// GetStoragePathByPath 通过路径查找存储路径
func (c *StoragePathCollector) GetStoragePathByPath(path string) (StoragePathInfo, bool) {
	c.cache.mu.RLock()
	defer c.cache.mu.RUnlock()

	for _, item := range c.cache.items {
		if item.Path == path {
			c.metrics.cacheHits.Inc()
			return item, true
		}
	}

	c.metrics.cacheMisses.Inc()
	return StoragePathInfo{}, false
}

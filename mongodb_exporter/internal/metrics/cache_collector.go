package metrics

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/bson"
)

// MongoDBCachecollector 负责采集 MongoDB 的缓存相关指标（仅适用于 WiredTiger 引擎）
type MongoDBCachecollector struct {
	pool         *MongoDBClientPool
	instanceName string
	instanceURI  string
	metrics      *cacheMetrics
}

type cacheMetrics struct {
	cacheBytesInCacheMetric        *baseMetrics // 当前缓存中的数据量
	cacheBytesDirtyMetric          *baseMetrics // 当前脏页大小
	cacheUsedRatioMetric           *baseMetrics // 缓存使用率
	cacheDirtyRatioMetric          *baseMetrics // 脏页占比
	cacheEvictedMetric             *baseMetrics // 已淘汰缓存项
	cacheFlushedMetric             *baseMetrics // 已刷新到磁盘的缓存项
	cacheCurrentSizeMetric         *baseMetrics // 当前缓存大小
	cacheMaximumSizeMetric         *baseMetrics // 最大缓存大小
	cacheIncrResizeMetric          *baseMetrics // 缓存增加次数
	cacheDecrResizeMetric          *baseMetrics // 缓存减少次数
	cachePagesReadMetric           *baseMetrics // 从磁盘读入的页面数
	cachePagesWriteMetric          *baseMetrics // 写入磁盘的页面数
	cacheTrackDirtyMetric          *baseMetrics // 脏页跟踪数量
	cacheTrackedDirtyBytesMetric   *baseMetrics // 脏页字节数
	cacheDirtyUnpinnedMetric       *baseMetrics // 未 pin 的脏页数
	cacheDirtyUnpinnedBytesMetric  *baseMetrics // 未 pin 的脏页字节数
	cacheMaxCachedPageSizeMetric   *baseMetrics // 最大的单个缓存页大小
	cacheMaxBucketAllocationMetric *baseMetrics // 最大的 bucket 分配大小
	cacheBucketsMetric             *baseMetrics // 缓存桶数量
	cacheModifiedDataInCacheMetric *baseMetrics // 修改过的数据在缓存中
}

func newCacheMetrics() *cacheMetrics {
	return &cacheMetrics{
		cacheBytesInCacheMetric: NewMetrics(
			"mongodb_cache_bytes_in_the_cache",
			"The total size of data currently in the cache.",
			[]string{"instance", "uri"},
		),
		cacheBytesDirtyMetric: NewMetrics(
			"mongodb_cache_bytes_dirty",
			"The total size of dirty data in the cache.",
			[]string{"instance", "uri"},
		),
		cacheUsedRatioMetric: NewMetrics(
			"mongodb_cache_used_ratio",
			"The ratio of used cache (bytes_currently_in_cache / maximum_bytes_configured).",
			[]string{"instance", "uri"},
		),
		cacheDirtyRatioMetric: NewMetrics(
			"mongodb_cache_dirty_ratio",
			"The ratio of dirty bytes in cache to maximum configured cache size.",
			[]string{"instance", "uri"},
		),
		cacheEvictedMetric: NewMetrics(
			"mongodb_cache_evicted_total",
			"The total number of pages evicted from the cache.",
			[]string{"instance", "uri"},
		),
		cacheFlushedMetric: NewMetrics(
			"mongodb_cache_flushes_total",
			"The total number of times data has been flushed to disk.",
			[]string{"instance", "uri"},
		),
		cacheCurrentSizeMetric: NewMetrics(
			"mongodb_cache_current_size_bytes",
			"The current size of the cache in bytes.",
			[]string{"instance", "uri"},
		),
		cacheMaximumSizeMetric: NewMetrics(
			"mongodb_cache_max_size_bytes",
			"The maximum size of the cache in bytes.",
			[]string{"instance", "uri"},
		),
		cacheIncrResizeMetric: NewMetrics(
			"mongodb_cache_resizes_increased_total",
			"The number of times the cache increased in size.",
			[]string{"instance", "uri"},
		),
		cacheDecrResizeMetric: NewMetrics(
			"mongodb_cache_resizes_decreased_total",
			"The number of times the cache decreased in size.",
			[]string{"instance", "uri"},
		),
		cachePagesReadMetric: NewMetrics(
			"mongodb_cache_pages_read_total",
			"The number of pages read into the cache from disk.",
			[]string{"instance", "uri"},
		),
		cachePagesWriteMetric: NewMetrics(
			"mongodb_cache_pages_write_total",
			"The number of pages written from cache to disk.",
			[]string{"instance", "uri"},
		),
		cacheTrackDirtyMetric: NewMetrics(
			"mongodb_cache_tracked_dirty_objects_total",
			"The number of tracked dirty objects in the cache.",
			[]string{"instance", "uri"},
		),
		cacheTrackedDirtyBytesMetric: NewMetrics(
			"mongodb_cache_tracked_dirty_bytes",
			"The total number of bytes of dirty data in cache.",
			[]string{"instance", "uri"},
		),
		cacheDirtyUnpinnedMetric: NewMetrics(
			"mongodb_cache_dirty_unpinned_objects_total",
			"The number of unpinned dirty objects in the cache.",
			[]string{"instance", "uri"},
		),
		cacheDirtyUnpinnedBytesMetric: NewMetrics(
			"mongodb_cache_dirty_unpinned_bytes",
			"The total number of bytes of unpinned dirty data in cache.",
			[]string{"instance", "uri"},
		),
		cacheMaxCachedPageSizeMetric: NewMetrics(
			"mongodb_cache_max_page_size_bytes",
			"The maximum page size that can be cached.",
			[]string{"instance", "uri"},
		),
		cacheMaxBucketAllocationMetric: NewMetrics(
			"mongodb_cache_max_bucket_allocation_bytes",
			"The maximum chunk size allocated for cache buckets.",
			[]string{"instance", "uri"},
		),
		cacheBucketsMetric: NewMetrics(
			"mongodb_cache_buckets_count",
			"The number of buckets in the cache hash table.",
			[]string{"instance", "uri"},
		),
		cacheModifiedDataInCacheMetric: NewMetrics(
			"mongodb_cache_modified_data_total",
			"The number of modified data items in the cache.",
			[]string{"instance", "uri"},
		),
	}
}

// NewMongoDBCachecollector 创建一个新的 Cache collector
func NewMongoDBCachecollector(pool *MongoDBClientPool, instanceName, instanceURI string) *MongoDBCachecollector {
	return &MongoDBCachecollector{
		pool:         pool,
		instanceName: instanceName,
		instanceURI:  instanceURI,
		metrics:      newCacheMetrics(),
	}
}

// Describe implements Prometheus collector interface

// collect implements Prometheus collector interface
func (c *MongoDBCachecollector) collect(ch chan<- prometheus.Metric) {
	client, _ := c.pool.GetClient()
	ctx := context.Background()
	adminDB := client.Database("admin")

	var result bson.M
	err := adminDB.RunCommand(ctx, bson.D{{Key: "serverStatus", Value: 1}}).Decode(&result)
	if err != nil {
		fmt.Printf("Failed to run serverStatus command: %v\n", err)
		return
	}

	wt, ok := result["wiredTiger"].(bson.M)
	if !ok {
		fmt.Println("WiredTiger engine not found")
		return
	}

	cache, ok := wt["cache"].(bson.M)
	if !ok {
		fmt.Println("WiredTiger cache stats not found")
		return
	}

	labels := []string{c.instanceName, c.instanceURI}

	// bytes currently in cache
	if val, ok := cache["bytes currently in the cache"].(int64); ok {
		c.metrics.cacheBytesInCacheMetric.collect(
			ch,
			float64(val),
			labels,
		)
	}

	// dirty bytes in cache
	if val, ok := cache["tracked dirty bytes in cache"].(int64); ok {
		c.metrics.cacheBytesDirtyMetric.collect(
			ch,
			float64(val),
			labels,
		)

		// 获取最大缓存配置值
		if max, ok := cache["maximum bytes configured"].(int64); ok && max > 0 {
			ratio := float64(val) / float64(max)
			c.metrics.cacheDirtyRatioMetric.collect(
				ch,
				ratio,
				labels,
			)
		}
	}

	// used ratio = bytes_currently_in_cache / maximum_bytes_configured
	if current, ok := cache["bytes currently in the cache"].(int64); ok {
		if max, ok := cache["maximum bytes configured"].(int64); ok && max > 0 {
			ratio := float64(current) / float64(max)
			c.metrics.cacheUsedRatioMetric.collect(
				ch,
				ratio,
				labels,
			)
		}
	}

	// eviction count
	if val, ok := cache["eviction server - eviction passes"].(int64); ok {
		c.metrics.cacheEvictedMetric.collect(
			ch,
			float64(val),
			labels,
		)
	}

	// flushes
	if val, ok := cache["pages read into cache"].([]interface{}); ok && len(val) >= 2 {
		if f, ok := val[1].(float64); ok {
			c.metrics.cacheFlushedMetric.collect(
				ch,
				f,
				labels,
			)
		}
	}

	// current size
	if val, ok := cache["current cache size"].(bson.M); ok {
		if sz, ok := val["bytes"].(int64); ok {
			c.metrics.cacheCurrentSizeMetric.collect(
				ch,
				float64(sz),
				labels,
			)
		}
	}

	// max size
	if val, ok := cache["maximum bytes configured"].(int64); ok {
		c.metrics.cacheMaximumSizeMetric.collect(
			ch,
			float64(val),
			labels,
		)
	}

	// resize increases
	if val, ok := cache["cache overflow score for resize"].(int32); ok {
		c.metrics.cacheIncrResizeMetric.collect(
			ch,
			float64(val),
			labels,
		)
	}

	// resize decreases
	if val, ok := cache["cache underflow score for resize"].(int32); ok {
		c.metrics.cacheDecrResizeMetric.collect(
			ch,
			float64(val),
			labels,
		)
	}

	// pages read
	if val, ok := cache["pages read into cache"].(int64); ok {
		c.metrics.cachePagesReadMetric.collect(
			ch,
			float64(val),
			labels,
		)
	}

	// pages write
	if val, ok := cache["pages written out to disk"].(int64); ok {
		c.metrics.cachePagesWriteMetric.collect(
			ch,
			float64(val),
			labels,
		)
	}

	// tracked dirty objects
	if val, ok := cache["tracked dirty objects in the cache"].(int64); ok {
		c.metrics.cacheTrackDirtyMetric.collect(
			ch,
			float64(val),
			labels,
		)
	}

	// tracked dirty bytes
	if val, ok := cache["tracked dirty bytes in cache"].(int64); ok {
		c.metrics.cacheTrackedDirtyBytesMetric.collect(
			ch,
			float64(val),
			labels,
		)
	}

	// dirty unpinned objects
	if val, ok := cache["unpinned modified pages in the cache"].(int64); ok {
		c.metrics.cacheDirtyUnpinnedMetric.collect(
			ch,
			float64(val),
			labels,
		)
	}

	// dirty unpinned bytes
	if val, ok := cache["unpinned modified pages in the cache (bytes)"].(int64); ok {
		c.metrics.cacheDirtyUnpinnedBytesMetric.collect(
			ch,
			float64(val),
			labels,
		)
	}

	// max page size
	if val, ok := cache["maximum page size"].(int64); ok {
		c.metrics.cacheMaxCachedPageSizeMetric.collect(
			ch,
			float64(val),
			labels,
		)
	}

	// max bucket allocation
	if val, ok := cache["bucket allocation size"].(int64); ok {
		c.metrics.cacheMaxBucketAllocationMetric.collect(
			ch,
			float64(val),
			labels,
		)
	}

	// buckets
	if val, ok := cache["buckets in hash table"].(int64); ok {
		c.metrics.cacheBucketsMetric.collect(
			ch,
			float64(val),
			labels,
		)
	}

	// modified data
	if val, ok := cache["modified data in cache"].(int64); ok {
		c.metrics.cacheModifiedDataInCacheMetric.collect(
			ch,
			float64(val),
			labels,
		)
	}
}

package metrics

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hansmi/paperhooks/pkg/client"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// StatisticsCollectorConfig 定义收集器配置
type StatisticsCollectorConfig struct {
	RefreshInterval   time.Duration     // 数据刷新间隔
	Logger           *zap.Logger       // 日志记录器
	EnableFileTypes  bool              // 是否启用文件类型统计
	CustomLabels     map[string]string // 自定义标签
	Timeout          time.Duration     // 请求超时时间
	HistogramBuckets []float64         // 直方图分桶设置
}

// statisticsClient 接口保持不变
type statisticsClient interface {
	GetStatistics(context.Context) (*client.Statistics, *client.Response, error)
}

// fileTypeStats 文件类型统计缓存
type fileTypeStats struct {
	mimeType string
	count    int64
	lastSeen time.Time
}

// statisticsCollector 重构后的收集器实现
type statisticsCollector struct {
	cl     statisticsClient
	config StatisticsCollectorConfig
	mtx    sync.RWMutex

	// 文档基础指标
	documentsTotalDesc      *prometheus.Desc
	documentsInboxDesc     *prometheus.Desc
	documentsProcessedDesc *prometheus.Desc
	documentsDeletedDesc   *prometheus.Desc

	// 文件类型指标
	documentFileTypeCountsDesc *prometheus.Desc
	documentFileTypeStats      map[string]fileTypeStats

	// 元数据指标
	characterCountDesc     *prometheus.Desc
	tagCountDesc          *prometheus.Desc
	correspondentCountDesc *prometheus.Desc
	documentTypeCountDesc *prometheus.Desc
	storagePathCountDesc *prometheus.Desc
	asnCountDesc        *prometheus.Desc

	// 性能指标
	collectionDurationDesc   *prometheus.Desc
	collectionSuccessDesc    *prometheus.Desc
	collectionCountDesc      *prometheus.Desc
	collectionErrorCountDesc *prometheus.Desc
	requestDurationHistogram *prometheus.HistogramVec

	// 缓存状态
	lastStatistics   *client.Statistics
	lastError        error
	lastCollectTime  time.Time
	collectionStats  struct {
		total   int
		success int
		errors  int
	}
}

// NewStatisticsCollector 创建新的统计收集器（接口保持不变）
func NewStatisticsCollector(cl statisticsClient) *statisticsCollector {
	// 默认配置
	config := StatisticsCollectorConfig{
		Logger:          zap.NewNop(),
		EnableFileTypes: true,
		Timeout:         5 * time.Second,
		HistogramBuckets: []float64{0.1, 0.5, 1, 2, 5, 10},
	}

	c := &statisticsCollector{
		cl:                  cl,
		config:             config,
		documentFileTypeStats: make(map[string]fileTypeStats),
	}

	// 初始化指标描述符
	c.initDescriptors()

	return c
}

// initDescriptors 初始化所有Prometheus指标描述符
func (c *statisticsCollector) initDescriptors() {
	labelNames := make([]string, 0, len(c.config.CustomLabels))
	for k := range c.config.CustomLabels {
		labelNames = append(labelNames, k)
	}

	// 文档基础指标
	c.documentsTotalDesc = prometheus.NewDesc(
		"paperless_statistics_documents_total",
		"Total number of documents.",
		labelNames, 
		nil)

	c.documentsInboxDesc = prometheus.NewDesc(
		"paperless_statistics_documents_inbox_count",
		"Total number of documents that have the defined 'Inbox' tag.",
		labelNames, 
		nil)

	c.documentsProcessedDesc = prometheus.NewDesc(
		"paperless_statistics_documents_processed_count",
		"Total number of processed documents.",
		labelNames, 
		nil)

	c.documentsDeletedDesc = prometheus.NewDesc(
		"paperless_statistics_documents_deleted_count",
		"Total number of deleted documents.",
		labelNames, 
		nil)

	// 文件类型指标
	if c.config.EnableFileTypes {
		c.documentFileTypeCountsDesc = prometheus.NewDesc(
			"paperless_statistics_documents_file_type_counts",
			"Total number of documents per MIME type.",
			append(labelNames, "mime_type"), 
			nil)
	}

	// 元数据指标
	c.characterCountDesc = prometheus.NewDesc(
		"paperless_statistics_character_count",
		"Number of characters stored across the total number of documents.",
		labelNames, 
		nil)

	c.tagCountDesc = prometheus.NewDesc(
		"paperless_statistics_tag_count",
		"Total number of tags.",
		labelNames, 
		nil)

	c.correspondentCountDesc = prometheus.NewDesc(
		"paperless_statistics_correspondent_count",
		"Total number of correspondents.",
		labelNames, 
		nil)

	c.documentTypeCountDesc = prometheus.NewDesc(
		"paperless_statistics_document_type_count",
		"Total number of document types.",
		labelNames, 
		nil)

	c.storagePathCountDesc = prometheus.NewDesc(
		"paperless_statistics_storage_path_count",
		"Total number of storage paths.",
		labelNames, 
		nil)

	c.asnCountDesc = prometheus.NewDesc(
		"paperless_statistics_asn_count",
		"Total number of unique ASNs (Archive Serial Numbers).",
		labelNames, 
		nil)

	// 性能指标
	c.collectionDurationDesc = prometheus.NewDesc(
		"paperless_statistics_collection_duration_seconds",
		"Duration of the last statistics collection in seconds.",
		labelNames, 
		nil)

	c.collectionSuccessDesc = prometheus.NewDesc(
		"paperless_statistics_collection_success",
		"Status of the last collection (1=success, 0=failure).",
		labelNames, 
		nil)

	c.collectionCountDesc = prometheus.NewDesc(
		"paperless_statistics_collection_count",
		"Total number of statistics collections performed.",
		labelNames, 
		nil)

	c.collectionErrorCountDesc = prometheus.NewDesc(
		"paperless_statistics_collection_error_count",
		"Total number of failed statistics collections.",
		labelNames, 
		nil)

	// 请求直方图
	c.requestDurationHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "paperless_statistics_request_duration_seconds",
			Help:    "Histogram of request latencies for statistics collection.",
			Buckets: c.config.HistogramBuckets,
		},
		[]string{"method", "status"},
	)
}

// Describe 实现prometheus.Collector接口
func (c *statisticsCollector) Describe(ch chan<- *prometheus.Desc) {
	// 文档基础指标
	ch <- c.documentsTotalDesc
	ch <- c.documentsInboxDesc
	ch <- c.documentsProcessedDesc
	ch <- c.documentsDeletedDesc

	// 文件类型指标
	if c.config.EnableFileTypes {
		ch <- c.documentFileTypeCountsDesc
	}

	// 元数据指标
	ch <- c.characterCountDesc
	ch <- c.tagCountDesc
	ch <- c.correspondentCountDesc
	ch <- c.documentTypeCountDesc
	ch <- c.storagePathCountDesc
	ch <- c.asnCountDesc

	// 性能指标
	ch <- c.collectionDurationDesc
	ch <- c.collectionSuccessDesc
	ch <- c.collectionCountDesc
	ch <- c.collectionErrorCountDesc

	// 直方图
	c.requestDurationHistogram.Describe(ch)
}

// Collect 实现prometheus.Collector接口
func (c *statisticsCollector) Collect(ctx context.Context, 
	ch chan<- prometheus.Metric) error {

	startTime := time.Now()
	var stats *client.Statistics
	var err error

	// 检查是否有缓存数据
	c.mtx.RLock()
	if c.lastStatistics != nil && time.Since(c.lastCollectTime) < c.config.RefreshInterval {
		stats = c.lastStatistics
		err = c.lastError
	}
	c.mtx.RUnlock()

	// 如果没有缓存或缓存过期，则实时获取
	if stats == nil {
		timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
			status := "success"
			if err != nil {
				status = "error"
			}
			c.requestDurationHistogram.WithLabelValues("GetStatistics", status).Observe(v)
		}))
		defer timer.ObserveDuration()

		reqCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
		defer cancel()

		stats, _, err = c.cl.GetStatistics(reqCtx)
		if err != nil {
			c.config.Logger.Error("Failed to get statistics",
				zap.Error(err),
				zap.Duration("timeout", c.config.Timeout))

			c.mtx.Lock()
			c.collectionStats.errors++
			c.collectionStats.total++
			c.mtx.Unlock()

			return fmt.Errorf("failed to get statistics: %w", err)
		}

		// 更新缓存
		c.mtx.Lock()
		c.lastStatistics = stats
		c.lastError = nil
		c.lastCollectTime = time.Now()
		c.collectionStats.success++
		c.collectionStats.total++
		c.mtx.Unlock()
	}

	// 记录收集性能指标
	duration := time.Since(startTime).Seconds()
	success := float64(0)
	if err == nil {
		success = 1
	}

	ch <- prometheus.MustNewConstMetric(c.collectionDurationDesc, 
		prometheus.GaugeValue, 
		duration)

	ch <- prometheus.MustNewConstMetric(c.collectionSuccessDesc, 
		prometheus.GaugeValue, 
		success)

	c.mtx.RLock()
	ch <- prometheus.MustNewConstMetric(c.collectionCountDesc, 
		prometheus.CounterValue, 
		float64(c.collectionStats.total))

	ch <- prometheus.MustNewConstMetric(c.collectionErrorCountDesc, 
		prometheus.CounterValue, 
		float64(c.collectionStats.errors))

	c.mtx.RUnlock()

	// 如果获取统计失败，只返回性能指标
	if err != nil || stats == nil {
		return nil
	}

	// 收集文档指标
	c.collectDocumentMetrics(ch, stats)

	// 收集文件类型指标
	if c.config.EnableFileTypes {
		c.collectFileTypeMetrics(ch, stats)
	}

	// 收集元数据指标
	c.collectMetadataMetrics(ch, stats)

	// 收集直方图指标
	c.requestDurationHistogram.Collect(ch)

	return nil
}

// collectDocumentMetrics 收集文档相关指标
func (c *statisticsCollector) collectDocumentMetrics(ch chan<- prometheus.Metric, 
	stats *client.Statistics) {

	ch <- prometheus.MustNewConstMetric(c.documentsTotalDesc, 
		prometheus.GaugeValue, 
		float64(stats.DocumentsTotal))

	ch <- prometheus.MustNewConstMetric(c.documentsInboxDesc, 
		prometheus.GaugeValue, 
		float64(stats.DocumentsInbox))

	// 计算处理文档数（总文档数减去收件箱文档数）
	processed := stats.DocumentsTotal - stats.DocumentsInbox
	ch <- prometheus.MustNewConstMetric(c.documentsProcessedDesc, 
		prometheus.GaugeValue, 
		float64(processed))

	// 假设有删除文档数（实际API可能不提供此数据）
	deleted := float64(0) // 可以扩展为从其他API获取
	ch <- prometheus.MustNewConstMetric(c.documentsDeletedDesc, 
		prometheus.GaugeValue, 
		deleted)
}

// collectFileTypeMetrics 收集文件类型指标
func (c *statisticsCollector) collectFileTypeMetrics(ch chan<- prometheus.Metric, 
	stats *client.Statistics) {

	now := time.Now()

	// 更新文件类型缓存
	c.mtx.Lock()
	for _, ft := range stats.DocumentFileTypeCounts {
		c.documentFileTypeStats[ft.MimeType] = fileTypeStats{
			mimeType: ft.MimeType,
			count:    ft.MimeTypeCount,
			lastSeen: now,
		}
	}

	// 清理过期的文件类型缓存
	for mimeType, stat := range c.documentFileTypeStats {
		if now.Sub(stat.lastSeen) > 24*time.Hour {
			delete(c.documentFileTypeStats, mimeType)
		}
	}
	c.mtx.Unlock()

	// 发送文件类型指标
	for _, ft := range stats.DocumentFileTypeCounts {
		ch <- prometheus.MustNewConstMetric(
			c.documentFileTypeCountsDesc,
			prometheus.GaugeValue,
			float64(ft.MimeTypeCount),
			ft.MimeType,
		)
	}
}

// collectMetadataMetrics 收集元数据指标
func (c *statisticsCollector) collectMetadataMetrics(ch chan<- prometheus.Metric, 
	stats *client.Statistics) {

	ch <- prometheus.MustNewConstMetric(c.characterCountDesc, 
		prometheus.GaugeValue, 
		float64(stats.CharacterCount))

	ch <- prometheus.MustNewConstMetric(c.tagCountDesc, 
		prometheus.GaugeValue, 
		float64(stats.TagCount))

	ch <- prometheus.MustNewConstMetric(c.correspondentCountDesc, 
		prometheus.GaugeValue, 
		float64(stats.CorrespondentCount))

	ch <- prometheus.MustNewConstMetric(c.documentTypeCountDesc, 
		prometheus.GaugeValue, 
		float64(stats.DocumentTypeCount))

	ch <- prometheus.MustNewConstMetric(c.storagePathCountDesc, 
		prometheus.GaugeValue, 
		float64(stats.StoragePathCount))

	// 假设有ASN计数（实际API可能不提供此数据）
	asnCount := float64(0) // 可以扩展为从其他API获取
	ch <- prometheus.MustNewConstMetric(c.asnCountDesc, 
		prometheus.GaugeValue, 
		asnCount)

}

// GetCollectionStats 获取收集统计信息
func (c *statisticsCollector) GetCollectionStats() (total, 
	success, 
	errors int) {

	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.collectionStats.total, 
			c.collectionStats.success, 
			c.collectionStats.errors
}

// GetFileTypeStats 获取文件类型统计信息
func (c *statisticsCollector) GetFileTypeStats() map[string]int64 {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	result := make(map[string]int64)
	for mimeType, stat := range c.documentFileTypeStats {
		result[mimeType] = stat.count
	}
	return result
}

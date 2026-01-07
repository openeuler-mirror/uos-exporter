package metrics

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hansmi/paperhooks/pkg/client"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// StatusCollectorConfig 定义收集器配置
type StatusCollectorConfig struct {
	RefreshInterval    time.Duration     // 状态刷新间隔
	Logger            *zap.Logger       // 日志记录器
	StorageMultiplier float64           // 存储单位转换系数
	EnableSubsystems  map[string]bool   // 启用的子系统
	CustomLabels      map[string]string // 自定义标签
	Timeout           time.Duration     // 请求超时时间
}

// statusClient 接口保持不变
type statusClient interface {
	GetStatus(ctx context.Context) (*client.SystemStatus, 
		*client.Response, error)
}

// subsystemStatus 定义子系统状态
type subsystemStatus struct {
	name         string
	status       string
	lastChecked time.Time
}

// statusCollector 重构后的收集器实现
type statusCollector struct {
	cl     statusClient
	config StatusCollectorConfig
	mtx    sync.RWMutex

	// 基础指标描述符
	storageTotalDesc       *prometheus.Desc
	storageAvailableDesc   *prometheus.Desc
	storageUsedDesc        *prometheus.Desc
	storageUsageRatioDesc  *prometheus.Desc

	// 子系统状态指标描述符
	subsystemStatusDescs    map[string]*prometheus.Desc
	subsystemTimestampDescs map[string]*prometheus.Desc

	// 数据库迁移指标
	migrationStatusDesc      *prometheus.Desc
	migrationCountDesc       *prometheus.Desc
	migrationPendingDesc     *prometheus.Desc

	// 性能指标
	collectionDurationDesc   *prometheus.Desc
	collectionSuccessDesc    *prometheus.Desc
	collectionCountDesc      *prometheus.Desc
	collectionErrorCountDesc *prometheus.Desc

	// 缓存状态
	lastStatus       *client.SystemStatus
	lastError        error
	lastCollectTime  time.Time
	subsystemCache   map[string]subsystemStatus
	collectionStats  struct {
		total   int
		success int
		errors  int
	}
}

// NewStatusCollector 创建新的状态收集器（接口保持不变）
func NewStatusCollector(cl statusClient) *statusCollector {
	// 默认配置
	config := StatusCollectorConfig{
		Logger:            zap.NewNop(),
		StorageMultiplier: 1.0,
		Timeout:           5 * time.Second,
		EnableSubsystems: map[string]bool{
			"database":  true,
			"redis":     true,
			"celery":    true,
			"index":     true,
			"classifier": true,
		},
	}

	c := &statusCollector{
		cl:            cl,
		config:        config,
		subsystemCache: make(map[string]subsystemStatus),
	}

	// 初始化指标描述符
	c.initDescriptors()

	return c
}

// initDescriptors 初始化所有Prometheus指标描述符
func (c *statusCollector) initDescriptors() {
	labelNames := make([]string, 0, len(c.config.CustomLabels))
	for k := range c.config.CustomLabels {
		labelNames = append(labelNames, k)
	}

	// 存储相关指标
	c.storageTotalDesc = prometheus.NewDesc(
		"paperless_status_storage_total_bytes",
		"Total storage capacity in bytes.",
		labelNames, 
		nil)

	c.storageAvailableDesc = prometheus.NewDesc(
		"paperless_status_storage_available_bytes",
		"Available storage in bytes.",
		labelNames, 
		nil)

	c.storageUsedDesc = prometheus.NewDesc(
		"paperless_status_storage_used_bytes",
		"Used storage in bytes.",
		labelNames, 
		nil)

	c.storageUsageRatioDesc = prometheus.NewDesc(
		"paperless_status_storage_usage_ratio",
		"Storage usage ratio (0-1).",
		labelNames, 
		nil)

	// 子系统状态指标
	c.subsystemStatusDescs = make(map[string]*prometheus.Desc)
	c.subsystemTimestampDescs = make(map[string]*prometheus.Desc)

	for name, enabled := range c.config.EnableSubsystems {
		if enabled {
			c.subsystemStatusDescs[name] = prometheus.NewDesc(
				"paperless_status_"+name+"_status",
				"Status of the "+name+" subsystem (1=OK, 0=not OK).",
				labelNames, 
				nil)

			c.subsystemTimestampDescs[name] = prometheus.NewDesc(
				"paperless_status_"+name+"_last_checked_timestamp_seconds",
				"Timestamp when "+name+" status was last checked.",
				labelNames, 
				nil)
		}
	}

	// 数据库迁移指标
	c.migrationStatusDesc = prometheus.NewDesc(
		"paperless_status_database_migration_status",
		"Database migration status (1=no pending migrations, 0=pending migrations exist).",
		labelNames, 
		nil)

	c.migrationCountDesc = prometheus.NewDesc(
		"paperless_status_database_unapplied_migrations",
		"Number of unapplied database migrations.",
		labelNames, 
		nil)

	c.migrationPendingDesc = prometheus.NewDesc(
		"paperless_status_database_migration_pending",
		"Database migration pending status (1=pending, 0=no pending).",
		labelNames, 
		nil)

	// 收集器性能指标
	c.collectionDurationDesc = prometheus.NewDesc(
		"paperless_status_collection_duration_seconds",
		"Duration of the last status collection in seconds.",
		labelNames, 
		nil)

	c.collectionSuccessDesc = prometheus.NewDesc(
		"paperless_status_collection_success",
		"Status of the last collection (1=success, 0=failure).",
		labelNames, 
		nil)

	c.collectionCountDesc = prometheus.NewDesc(
		"paperless_status_collection_count",
		"Total number of status collections performed.",
		labelNames, 
		nil)

	c.collectionErrorCountDesc = prometheus.NewDesc(
		"paperless_status_collection_error_count",
		"Total number of failed status collections.",
		labelNames, 
		nil)
}

// Describe 实现prometheus.Collector接口
func (c *statusCollector) Describe(ch chan<- *prometheus.Desc) {
	// 存储指标
	ch <- c.storageTotalDesc
	ch <- c.storageAvailableDesc
	ch <- c.storageUsedDesc
	ch <- c.storageUsageRatioDesc

	// 子系统状态指标
	for _, desc := range c.subsystemStatusDescs {
		ch <- desc
	}
	for _, desc := range c.subsystemTimestampDescs {
		ch <- desc
	}

	// 数据库迁移指标
	ch <- c.migrationStatusDesc
	ch <- c.migrationCountDesc
	ch <- c.migrationPendingDesc

	// 收集器性能指标
	ch <- c.collectionDurationDesc
	ch <- c.collectionSuccessDesc
	ch <- c.collectionCountDesc
	ch <- c.collectionErrorCountDesc
}

// Collect 实现prometheus.Collector接口
func (c *statusCollector) Collect(ctx context.Context, 
	ch chan<- prometheus.Metric) error {

	startTime := time.Now()
	var status *client.SystemStatus
	var err error

	// 检查是否有缓存数据
	c.mtx.RLock()
	if c.lastStatus != nil && time.Since(c.lastCollectTime) < c.config.RefreshInterval {
		status = c.lastStatus
		err = c.lastError
	}
	c.mtx.RUnlock()

	// 如果没有缓存或缓存过期，则实时获取
	if status == nil {
		reqCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
		defer cancel()

		status, _, err = c.cl.GetStatus(reqCtx)
		if err != nil {
			c.config.Logger.Error("Failed to get system status", 
				zap.Error(err),
				zap.Duration("timeout", c.config.Timeout))
			
			c.mtx.Lock()
			c.collectionStats.errors++
			c.collectionStats.total++
			c.mtx.Unlock()
			
			return fmt.Errorf("failed to get system status: %w", err)
		}

		// 更新缓存
		c.mtx.Lock()
		c.lastStatus = status
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

	// 如果获取状态失败，只返回性能指标
	if err != nil || status == nil {
		return nil
	}

	// 收集存储指标
	c.collectStorageMetrics(ch, status)

	// 收集子系统状态指标
	c.collectSubsystemMetrics(ch, status)

	// 收集数据库迁移指标
	c.collectMigrationMetrics(ch, status)

	return nil
}

// collectStorageMetrics 收集存储相关指标
func (c *statusCollector) collectStorageMetrics(ch chan<- prometheus.Metric, 
	status *client.SystemStatus) {

	storageTotal := float64(status.Storage.Total) * c.config.StorageMultiplier
	storageAvailable := float64(status.Storage.Available) * c.config.StorageMultiplier
	storageUsed := storageTotal - storageAvailable
	usageRatio := storageUsed / storageTotal

	ch <- prometheus.MustNewConstMetric(c.storageTotalDesc, 
		prometheus.GaugeValue, 
		storageTotal)

	ch <- prometheus.MustNewConstMetric(c.storageAvailableDesc, 
		prometheus.GaugeValue, 
		storageAvailable)

	ch <- prometheus.MustNewConstMetric(c.storageUsedDesc, 
		prometheus.GaugeValue, 
		storageUsed)

	ch <- prometheus.MustNewConstMetric(c.storageUsageRatioDesc, 
		prometheus.GaugeValue, 
		usageRatio)
}

// collectSubsystemMetrics 收集子系统状态指标
func (c *statusCollector) collectSubsystemMetrics(ch chan<- prometheus.Metric, 
	status *client.SystemStatus) {

	now := time.Now()

	// 数据库状态
	if desc, ok := c.subsystemStatusDescs["database"]; ok {
		statusValue := c.isOK(status.Database.Status)

		ch <- prometheus.MustNewConstMetric(desc, 
			prometheus.GaugeValue, 
			statusValue)

		c.updateSubsystemCache("database", 
			status.Database.Status, 
			now)
	}

	// Redis状态
	if desc, ok := c.subsystemStatusDescs["redis"]; ok {
		statusValue := c.isOK(status.Tasks.RedisStatus)
		ch <- prometheus.MustNewConstMetric(desc, 
			prometheus.GaugeValue, 
			statusValue)

		c.updateSubsystemCache("redis", 
			status.Tasks.RedisStatus, 
			now)
	}

	// Celery状态
	if desc, ok := c.subsystemStatusDescs["celery"]; ok {
		statusValue := c.isOK(status.Tasks.CeleryStatus)

		ch <- prometheus.MustNewConstMetric(desc, 
			prometheus.GaugeValue, 
			statusValue)

		c.updateSubsystemCache("celery", 
			status.Tasks.CeleryStatus, 
			now)
	}

	// 索引状态
	if desc, ok := c.subsystemStatusDescs["index"]; ok {

		statusValue := c.isOK(status.Tasks.IndexStatus)

		ch <- prometheus.MustNewConstMetric(desc, 
			prometheus.GaugeValue, 
			statusValue)

		if tsDesc, ok := c.subsystemTimestampDescs["index"]; ok {
			ch <- prometheus.MustNewConstMetric(tsDesc, 
				prometheus.GaugeValue, 
				float64(status.Tasks.IndexLastModified.Unix()))
		}
		c.updateSubsystemCache("index", 
			status.Tasks.IndexStatus, 
			now)
	}

	// 分类器状态
	if desc, ok := c.subsystemStatusDescs["classifier"]; ok {

		statusValue := c.isOK(status.Tasks.ClassifierStatus)

		ch <- prometheus.MustNewConstMetric(desc, 
			prometheus.GaugeValue, 
			statusValue)

		if tsDesc, ok := c.subsystemTimestampDescs["classifier"]; ok {
			ch <- prometheus.MustNewConstMetric(tsDesc, 
				prometheus.GaugeValue, 
				float64(status.Tasks.ClassifierLastTrained.Unix()))
		}
		c.updateSubsystemCache("classifier", 
			status.Tasks.ClassifierStatus, 
			now)
	}
}

// collectMigrationMetrics 收集数据库迁移指标
func (c *statusCollector) collectMigrationMetrics(ch chan<- prometheus.Metric, 
	status *client.SystemStatus) {

	unappliedMigrations := float64(len(status.Database.MigrationStatus.UnappliedMigrations))
	migrationStatus := float64(0)
	if unappliedMigrations == 0 {
		migrationStatus = 1
	}

	ch <- prometheus.MustNewConstMetric(c.migrationStatusDesc, 
		prometheus.GaugeValue, 
		migrationStatus)

	ch <- prometheus.MustNewConstMetric(c.migrationCountDesc, 
		prometheus.GaugeValue, 
		unappliedMigrations)

	ch <- prometheus.MustNewConstMetric(c.migrationPendingDesc, 
		prometheus.GaugeValue, 
		1-migrationStatus)
}

// updateSubsystemCache 更新子系统缓存
func (c *statusCollector) updateSubsystemCache(name, 
	status string, 
	timestamp time.Time) {

	c.mtx.Lock()
	defer c.mtx.Unlock()
	
	c.subsystemCache[name] = subsystemStatus{
		name:         name,
		status:       status,
		lastChecked:  timestamp,
	}
}

// isOK 判断状态是否为OK
func (c *statusCollector) isOK(status string) float64 {
	if strings.EqualFold(status, "OK") {
		return 1
	}
	return 0
}

// GetCollectionStats 获取收集统计信息
func (c *statusCollector) GetCollectionStats() (total, 
	success, 
	errors int) {

	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.collectionStats.total, 
		c.collectionStats.success, 
		c.collectionStats.errors
}

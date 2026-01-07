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

// VersionInfo 封装版本信息
type VersionInfo struct {
	CurrentVersion    string
	LatestVersion     string
	UpdateAvailable   bool
	LastChecked       time.Time
	CheckError        error
	VersionComponents map[string]int // 分解版本号为组件
}

// VersionCache 版本信息缓存
type VersionCache struct {
	mu    sync.RWMutex
	info  VersionInfo
}

type RemoteVersionClient interface {
	GetRemoteVersion(ctx context.Context) (*client.RemoteVersion, *client.Response, error)
}

func NewVersionCache() *VersionCache {
	return &VersionCache{
		info: VersionInfo{
			VersionComponents: make(map[string]int),
		},
	}
}

func (c *VersionCache) Update(info VersionInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	info.LastChecked = time.Now()
	c.info = info
}

func (c *VersionCache) Get() VersionInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.info
}

func (c *VersionCache) IsStale(threshold time.Duration) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Since(c.info.LastChecked) > threshold
}

// VersionMetrics 封装指标相关操作
type VersionMetrics struct {
	updateAvailableDesc   *prometheus.Desc
	versionCheckErrorDesc *prometheus.Desc
	versionAgeDesc        *prometheus.Desc
	versionComponentsDesc *prometheus.Desc
	collectErrors         prometheus.Counter
	collectionDuration    prometheus.Histogram
	versionChecks        prometheus.Counter
}

func NewVersionMetrics() *VersionMetrics {
	return &VersionMetrics{
		updateAvailableDesc: prometheus.NewDesc(
			"paperless_remote_version_update_available",
			"Whether an update is available (1) or not (0).",
			[]string{"current_version", "latest_version"}, nil,
		),
		versionCheckErrorDesc: prometheus.NewDesc(
			"paperless_remote_version_check_error",
			"Indicates if version check failed (1) or succeeded (0).",
			nil, nil,
		),
		versionAgeDesc: prometheus.NewDesc(
			"paperless_remote_version_age_seconds",
			"Time since last successful version check.",
			nil, nil,
		),
		versionComponentsDesc: prometheus.NewDesc(
			"paperless_remote_version_component",
			"Numerical components of the version number.",
			[]string{"component"}, nil,
		),
		collectErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "paperless_version_collect_errors_total",
			Help: "Total number of errors encountered while collecting version metrics.",
		}),
		collectionDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "paperless_version_collection_duration_seconds",
			Help:    "Time taken to collect version metrics.",
			Buckets: prometheus.DefBuckets,
		}),
		versionChecks: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "paperless_version_checks_total",
			Help: "Total number of version checks performed.",
		}),
	}
}

// RemoteVersionCollector 主收集器实现
type RemoteVersionCollector struct {
	cl      RemoteVersionClient
	cache   *VersionCache
	metrics *VersionMetrics
	logger  *zap.Logger
	mu      sync.Mutex
	
	// 配置项
	cacheTTL      time.Duration
	enabled       bool
	parseVersions bool
}

// NewRemoteVersionCollector 创建新的收集器实例
func NewRemoteVersionCollector(cl RemoteVersionClient) *RemoteVersionCollector {
	return &RemoteVersionCollector{
		cl:          cl,
		cache:       NewVersionCache(),
		metrics:     NewVersionMetrics(),
		logger:      zap.NewNop(),
		cacheTTL:    1 * time.Hour, // 默认1小时检查一次
		enabled:     true,
		parseVersions: true,
	}
}

// WithLogger 设置日志记录器
func (c *RemoteVersionCollector) WithLogger(logger *zap.Logger) *RemoteVersionCollector {
	c.logger = logger
	return c
}

// WithCacheTTL 设置缓存TTL
func (c *RemoteVersionCollector) WithCacheTTL(ttl time.Duration) *RemoteVersionCollector {
	c.cacheTTL = ttl
	return c
}

// Enable 启用收集器
func (c *RemoteVersionCollector) Enable() *RemoteVersionCollector {
	c.enabled = true
	return c
}

// Disable 禁用收集器
func (c *RemoteVersionCollector) Disable() *RemoteVersionCollector {
	c.enabled = false
	return c
}

// EnableVersionParsing 启用版本号解析
func (c *RemoteVersionCollector) EnableVersionParsing() *RemoteVersionCollector {
	c.parseVersions = true
	return c
}

// DisableVersionParsing 禁用版本号解析
func (c *RemoteVersionCollector) DisableVersionParsing() *RemoteVersionCollector {
	c.parseVersions = false
	return c
}

// Describe 实现prometheus.Collector接口
func (c *RemoteVersionCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.metrics.updateAvailableDesc
	ch <- c.metrics.versionCheckErrorDesc
	ch <- c.metrics.versionAgeDesc
	ch <- c.metrics.versionComponentsDesc
	c.metrics.collectErrors.Describe(ch)
	c.metrics.collectionDuration.Describe(ch)
	c.metrics.versionChecks.Describe(ch)
}

// Collect 实现prometheus.Collector接口
func (c *RemoteVersionCollector) Collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	if !c.enabled {
		c.logger.Debug("Remote version collector is disabled")
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 记录收集耗时
	timer := prometheus.NewTimer(c.metrics.collectionDuration)
	defer timer.ObserveDuration()

	// 检查缓存是否过期
	stale := c.cache.IsStale(c.cacheTTL)
	if stale {
		if err := c.checkRemoteVersion(ctx); err != nil {
			c.logger.Error("Failed to check remote version",
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
	c.metrics.versionChecks.Collect(ch)

	return nil
}

// checkRemoteVersion 检查远程版本
func (c *RemoteVersionCollector) checkRemoteVersion(ctx context.Context) error {
	info := VersionInfo{
		VersionComponents: make(map[string]int),
	}

	remoteVersion, _, err := c.cl.GetRemoteVersion(ctx)
	c.metrics.versionChecks.Inc()

	if err != nil {
		info.CheckError = fmt.Errorf("fetching remote version: %w", err)
		c.cache.Update(info)
		return err
	}

	info.CurrentVersion = remoteVersion.Version
	info.LatestVersion = remoteVersion.Version // 假设最新版本就是当前版本
	info.UpdateAvailable = remoteVersion.UpdateAvailable

	if c.parseVersions {
		// 这里可以添加版本号解析逻辑，将版本号分解为组件
		// 例如: "v1.2.3" -> {"major":1, "minor":2, "patch":3}
	}

	c.cache.Update(info)
	return nil
}

// reportCachedMetrics 报告缓存中的指标
func (c *RemoteVersionCollector) reportCachedMetrics(ch chan<- prometheus.Metric) {
	info := c.cache.Get()

	// 报告更新可用性
	var updateAvailable float64
	if info.UpdateAvailable {
		updateAvailable = 1
	}

	ch <- prometheus.MustNewConstMetric(
		c.metrics.updateAvailableDesc,
		prometheus.GaugeValue,
		updateAvailable,
		info.CurrentVersion,
		info.LatestVersion,
	)

	// 报告检查错误状态
	var checkError float64
	if info.CheckError != nil {
		checkError = 1
	}

	ch <- prometheus.MustNewConstMetric(
		c.metrics.versionCheckErrorDesc,
		prometheus.GaugeValue,
		checkError,
	)

	// 报告版本检查年龄
	if !info.LastChecked.IsZero() {
		ch <- prometheus.MustNewConstMetric(
			c.metrics.versionAgeDesc,
			prometheus.GaugeValue,
			time.Since(info.LastChecked).Seconds(),
		)
	}

	// 报告版本组件
	for component, value := range info.VersionComponents {
		ch <- prometheus.MustNewConstMetric(
			c.metrics.versionComponentsDesc,
			prometheus.GaugeValue,
			float64(value),
			component,
		)
	}
}

// GetVersionInfo 获取版本信息
func (c *RemoteVersionCollector) GetVersionInfo() VersionInfo {
	return c.cache.Get()
}

// IsUpdateAvailable 检查是否有更新可用
func (c *RemoteVersionCollector) IsUpdateAvailable() bool {
	return c.cache.Get().UpdateAvailable
}

// GetLastCheckTime 获取最后检查时间
func (c *RemoteVersionCollector) GetLastCheckTime() time.Time {
	return c.cache.Get().LastChecked
}

// GetVersionComponents 获取版本组件
func (c *RemoteVersionCollector) GetVersionComponents() map[string]int {
	return c.cache.Get().VersionComponents
}

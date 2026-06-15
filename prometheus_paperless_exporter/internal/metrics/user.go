package metrics

import (
	"context"
	"sync"
	"time"

	"github.com/hansmi/paperhooks/pkg/client"
	"github.com/prometheus/client_golang/prometheus"
)

// UserMetrics 定义用户指标结构
type UserMetrics struct {
	TotalCount      float64
	ActiveCount     float64
	InactiveCount   float64
	SuperuserCount  float64
	LastLoginTime   time.Time
	CreationTime    time.Time
}

// UserClient 扩展用户客户端接口
type UserClient interface {
	ListUsers(context.Context, 
		client.ListUsersOptions) ([]client.User, 
			*client.Response, error)
}

// UserCollectorConfig 收集器配置
type UserCollectorConfig struct {
	EnableDetailedMetrics bool
	RefreshInterval       time.Duration
	RequestTimeout       time.Duration
}

// userCollector 重构后的用户收集器
type userCollector struct {
	mu sync.Mutex

	cl     UserClient
	config UserCollectorConfig

	// 基础指标
	countDesc        *prometheus.Desc
	activeDesc       *prometheus.Desc
	inactiveDesc     *prometheus.Desc
	superuserDesc    *prometheus.Desc

	// 内部指标
	scrapeDuration   prometheus.Summary
	scrapeErrors     prometheus.Counter
	requestDuration  prometheus.Histogram

	apiTotalDesc *prometheus.Desc
}

// NewUserCollector 创建新的用户收集器 (保持接口不变)
func NewUserCollector(cl UserClient) *userCollector {
	return &userCollector{
		cl: cl,
		config: UserCollectorConfig{
			RefreshInterval: 5 * time.Minute,
			RequestTimeout: 30 * time.Second,
		},
        apiTotalDesc: prometheus.NewDesc(
            "paperless_users_api_total",
            "Total number of users reported by API.",
            nil, 
			nil,
        ),
		countDesc: prometheus.NewDesc(
			"paperless_users_total",
			"Total number of users.",
			nil, 
			nil,
		),
		activeDesc: prometheus.NewDesc(
			"paperless_users_active",
			"Number of active users.",
			nil, 
			nil,
		),
		inactiveDesc: prometheus.NewDesc(
			"paperless_users_inactive",
			"Number of inactive users.",
			nil, 
			nil,
		),
		superuserDesc: prometheus.NewDesc(
			"paperless_users_superuser",
			"Number of superusers.",
			nil, 
			nil,
		),

		// 内部指标
		scrapeDuration: prometheus.NewSummary(
			prometheus.SummaryOpts{
				Name: "paperless_users_scrape_duration_seconds",
				Help: "Duration of user metrics collection.",
			},
		),
		scrapeErrors: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "paperless_users_scrape_errors_total",
				Help: "Total number of errors while collecting user metrics.",
			},
		),
		requestDuration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "paperless_users_request_duration_seconds",
				Help:    "Duration of API requests for user data.",
				Buckets: prometheus.DefBuckets,
			},
		),
	}
}

func (c *userCollector) Describe(ch chan<- *prometheus.Desc) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 基础指标
	ch <- c.countDesc
	ch <- c.activeDesc
	ch <- c.inactiveDesc
	ch <- c.superuserDesc

    ch <- prometheus.NewDesc(
        "paperless_users_api_total",
        "Total number of users reported by API.",
        nil, 
		nil,
    )
	// 内部指标
	c.scrapeDuration.Describe(ch)
	c.scrapeErrors.Describe(ch)
	c.requestDuration.Describe(ch)
}

func (c *userCollector) Collect(ctx context.Context, 
	ch chan<- prometheus.Metric) error {

	c.mu.Lock()
	defer c.mu.Unlock()

	startTime := time.Now()
	defer func() {
		c.scrapeDuration.Observe(time.Since(startTime).Seconds())
	}()

	// 设置请求超时
	if c.config.RequestTimeout > 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(ctx, 
			c.config.RequestTimeout)

		defer cancel()
	}

	// 收集用户数据
	users, response, err := c.collectUsers(ctx)
	if err != nil {
		c.scrapeErrors.Inc()
		return err
	}

	// 发送基础指标
	if err := c.sendBaseMetrics(ch, users, response); err != nil {
		return err
	}

	// 发送内部指标
	c.sendInternalMetrics(ch)

	return nil
}

func (c *userCollector) collectUsers(ctx context.Context) ([]client.User, 
	*client.Response, error) {

	startTime := time.Now()
	defer func() {
		c.requestDuration.Observe(time.Since(startTime).Seconds())
	}()

	return c.cl.ListUsers(ctx, client.ListUsersOptions{})
}

func (c *userCollector) sendBaseMetrics(ch chan<- prometheus.Metric, 
	users []client.User, 
	response *client.Response) error {

	metrics := UserMetrics{
		TotalCount: float64(len(users)),
	}

	for _, user := range users {
		if user.IsActive {
			metrics.ActiveCount++
		} else {
			metrics.InactiveCount++
		}

		if user.IsSuperuser {
			metrics.SuperuserCount++
		}
	}

	// 发送聚合指标
	ch <- prometheus.MustNewConstMetric(c.countDesc, 
		prometheus.GaugeValue, 
		metrics.TotalCount)

	ch <- prometheus.MustNewConstMetric(c.activeDesc, 
		prometheus.GaugeValue, 
		metrics.ActiveCount)

	ch <- prometheus.MustNewConstMetric(c.inactiveDesc, 
		prometheus.GaugeValue, 
		metrics.InactiveCount)

	ch <- prometheus.MustNewConstMetric(c.superuserDesc, 
		prometheus.GaugeValue, 
		metrics.SuperuserCount)

	// 使用response中的ItemCount（如果可用）
	if response != nil && response.ItemCount != client.ItemCountUnknown {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				"paperless_users_api_total",
				"Total number of users reported by API.",
				nil, 
				nil,
			),
			prometheus.GaugeValue,
			float64(response.ItemCount),
		)
	}

	return nil
}

func (c *userCollector) sendInternalMetrics(ch chan<- prometheus.Metric) {
	c.scrapeDuration.Collect(ch)
	c.scrapeErrors.Collect(ch)
	c.requestDuration.Collect(ch)
}

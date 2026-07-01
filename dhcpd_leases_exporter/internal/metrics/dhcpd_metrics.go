package metrics

import (
	"dhcpd_leases_exporter/pkg/collector"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// 定义命名空间常量
const (
	namespace = "dhcpd_leases"
)

// 全局变量定义
var (
	// 互斥锁，用于保护共享资源
	mux sync.Mutex

	// 定义命令行参数
	dhcpdLeasesFile = kingpin.Flag(
		"dhcpd.leases-file",
		"Path to dhcpd.leases file",
	).Default("/var/lib/dhcpd/dhcpd.leases").String()

	testMode = kingpin.Flag(
		"test-mode",
		"Create a test lease file for testing",
	).Bool()

	subnetsConfig = kingpin.Flag(
		"dhcpd.subnets",
		"DHCP 子网配置，格式：subnet1=start1-end1,subnet2=start2-end2",
	).String()

	// 初始化标志
	initialized = false

	// DHCP信息收集器，改为大写，使其可导出
	DHCPDInfo *collector.DHCPDInfo
)

// DHCPDMetrics 是 DHCPD 租约指标的主要收集器
type DHCPDMetrics struct {
	// 基础统计指标
	validLeases   *prometheus.Desc
	expiredLeases *prometheus.Desc
	totalLeases   *prometheus.Desc
	fileTimestamp *prometheus.Desc

	// 活跃租约指标
	activeLeases *prometheus.Desc

	// 收集器指标 - 统计相关
	scrapesTotalStats       prometheus.Counter
	scrapeErrorsTotalStats  prometheus.Counter
	lastScrapeErrorStats    *prometheus.Desc
	lastScrapeTimeStats     *prometheus.Desc
	lastScrapeDurationStats *prometheus.Desc

	// 收集器指标 - 活跃租约相关
	scrapesTotalActive       prometheus.Counter
	scrapeErrorsTotalActive  prometheus.Counter
	lastScrapeErrorActive    *prometheus.Desc
	lastScrapeTimeActive     *prometheus.Desc
	lastScrapeDurationActive *prometheus.Desc
}

// 创建测试租约文件
func createTestLeaseFile() {
	// 定义测试文件路径
	testFile := "/tmp/dhcpd_test.leases"

	// 设置租约文件路径为测试文件
	*dhcpdLeasesFile = testFile

	// 生成测试租约内容
	content := fmt.Sprintf(
		`lease 192.168.1.100 {
		starts 6 %s;
		ends 6 %s;
		hardware ethernet 00:11:22:33:44:55;
		client-hostname "test-host-1";
	}
	lease 192.168.1.101 {
		starts 6 %s;
		ends 6 %s;
		hardware ethernet 00:11:22:33:44:66;
		client-hostname "test-host-2";
		abandoned;
	}`,
		time.Now().AddDate(0, 0, -1).Format("2006/01/02 15:04:05"),
		time.Now().AddDate(0, 0, 7).Format("2006/01/02 15:04:05"),
		time.Now().AddDate(0, 0, -2).Format("2006/01/02 15:04:05"),
		time.Now().AddDate(0, 0, -1).Format("2006/01/02 15:04:05"),
	)

	// 写入测试文件
	err := os.WriteFile(testFile, []byte(content), 0600)

	// 处理可能的错误
	if err != nil {
		logrus.Errorf("创建测试租约文件失败: %v", err)
	} else {
		logrus.Infof("创建测试租约文件成功: %s", testFile)
	}
}

// NewDHCPDMetrics 创建新的 DHCP 租约指标收集器
func NewDHCPDMetrics() *DHCPDMetrics {
	// 创建新的指标收集器实例
	metrics := &DHCPDMetrics{
		// 基础统计指标
		validLeases: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "stats", "valid"),
			"The number of leases in dhcpd.leases that have not yet expired",
			nil,
			nil,
		),

		expiredLeases: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "stats", "expired"),
			"The number of leases in dhcpd.leases that have expired",
			nil,
			nil,
		),

		totalLeases: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "stats", "count"),
			"The number of leases in dhcpd.leases",
			nil,
			nil,
		),

		fileTimestamp: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "stats", "filetime"),
			"The file timestamp in seconds since epoch of the dhcpd.leases file",
			nil,
			nil,
		),

		// 活跃租约指标
		activeLeases: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "active", "client"),
			"The number of leases in dhcpd.leases that have not yet expired",
			[]string{"hostname", "ip", "mac"},
			nil,
		),

		// 收集器指标 - Stats
		scrapesTotalStats: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "stats",
				Name:      "scrapes_total",
				Help:      "Total number of scrapes",
			},
		),

		scrapeErrorsTotalStats: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "stats",
				Name:      "scrape_errors_total",
				Help:      "Total number of scrapes errors",
			},
		),

		lastScrapeErrorStats: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "stats", "last_scrape_error"),
			"Whether the last scrape of stats resulted in an error (1 for error, 0 for success).",
			nil,
			nil,
		),

		lastScrapeTimeStats: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "stats", "last_scrape_timestamp"),
			"Number of seconds since 1970 since last scrape of stat metrics.",
			nil,
			nil,
		),

		lastScrapeDurationStats: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "stats", "last_scrape_duration_seconds"),
			"Number of seconds since 1970 since last scrape of stat metrics.",
			nil,
			nil,
		),

		// 收集器指标 - Active
		scrapesTotalActive: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "active",
				Name:      "scrapes_total",
				Help:      "Total number of scrapes",
			},
		),

		scrapeErrorsTotalActive: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "active",
				Name:      "scrape_errors_total",
				Help:      "Total number of scrapes errors",
			},
		),

		lastScrapeErrorActive: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "active", "last_scrape_error"),
			"Whether the last scrape resulted in an error (1 for error, 0 for success).",
			nil,
			nil,
		),

		lastScrapeTimeActive: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "active", "last_scrape_timestamp"),
			"Number of seconds since 1970 since last scrape.",
			nil,
			nil,
		),

		lastScrapeDurationActive: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "active", "last_scrape_duration_seconds"),
			"Number of seconds the last scrape took",
			nil,
			nil,
		),
	}

	// 返回创建的指标收集器
	return metrics
}

// InitDHCPDInfo 初始化全局 DHCP 信息收集器
func InitDHCPDInfo() {
	// 加锁保护共享资源
	mux.Lock()
	defer mux.Unlock()

	// 只有在 DHCPDInfo 为空时才进行初始化
	if DHCPDInfo == nil {
		// 如果处于测试模式，创建测试租约文件
		if *testMode && !initialized {
			// 创建测试租约文件
			createTestLeaseFile()

			// 标记已初始化
			initialized = true
		}

		// 确保租约文件路径不为空
		if *dhcpdLeasesFile == "" {
			// 使用默认路径
			*dhcpdLeasesFile = "/var/lib/dhcpd/dhcpd.leases"

			// 记录警告日志
			logrus.Warnf("租约文件路径为空，使用默认路径: %s", *dhcpdLeasesFile)
		}

		// 检查租约文件是否存在
		if _, err := os.Stat(*dhcpdLeasesFile); os.IsNotExist(err) {
			// 记录警告日志
			logrus.Warnf("租约文件不存在: %s", *dhcpdLeasesFile)

			// 非测试模式下尝试创建空文件
			if !*testMode {
				logrus.Infof("尝试创建空的租约文件")

				// 创建空文件
				if err := os.WriteFile(*dhcpdLeasesFile, []byte(""), 0600); err != nil {
					logrus.Errorf("创建空租约文件失败: %v", err)
				}
			}
		}

		// 使用命令行参数中的租约文件路径
		logrus.Infof("使用租约文件: %s", *dhcpdLeasesFile)

		// 创建新的 DHCP 信息收集器
		DHCPDInfo = collector.NewDHCPDInfo(*dhcpdLeasesFile)

		// 配置子网信息
		if *subnetsConfig != "" {
			// 按逗号分割子网配置
			for _, subnet := range strings.Split(*subnetsConfig, ",") {
				// 按等号分割子网和地址范围
				parts := strings.Split(subnet, "=")

				// 检查格式是否正确
				if len(parts) != 2 {
					logrus.Errorf("无效的子网配置: %s", subnet)
					continue
				}

				// 获取子网 CIDR
				subnetCIDR := parts[0]

				// 按短横线分割起始和结束地址
				rangeParts := strings.Split(parts[1], "-")

				// 检查格式是否正确
				if len(rangeParts) != 2 {
					logrus.Errorf("无效的地址范围配置: %s", parts[1])
					continue
				}

				// 添加子网信息
				if err := DHCPDInfo.AddSubnet(subnetCIDR, rangeParts[0], rangeParts[1]); err != nil {
					logrus.Errorf("添加子网失败 %s: %v", subnetCIDR, err)
					continue
				}

				// 记录成功日志
				logrus.Infof(
					"已添加子网 %s (范围: %s - %s)",
					subnetCIDR,
					rangeParts[0],
					rangeParts[1],
				)
			}
		}

		// 设置服务器信息
		startTime := time.Now().Add(-time.Hour) // 假设服务器已运行 1 小时
		DHCPDInfo.SetServerInfo("ISC DHCP", time.Since(startTime))

		// 记录初始化完成日志
		logrus.Infof(
			"DHCP 信息收集器初始化完成，租约文件路径: %s",
			*dhcpdLeasesFile,
		)
	}
}

// initDHCPDInfo 初始化 DHCP 信息收集器（内部使用）
func initDHCPDInfo() {
	// 加锁保护共享资源
	mux.Lock()
	defer mux.Unlock()

	// 只有在 DHCPDInfo 为空时才进行初始化
	if DHCPDInfo == nil {
		InitDHCPDInfo()
	}
}

// Describe 实现 prometheus.Collector 接口
func (m *DHCPDMetrics) Describe(ch chan<- *prometheus.Desc) {
	// 基础统计指标
	ch <- m.validLeases
	ch <- m.expiredLeases
	ch <- m.totalLeases
	ch <- m.fileTimestamp

	// 活跃租约指标
	ch <- m.activeLeases

	// 收集器指标 - Stats
	m.scrapesTotalStats.Describe(ch)
	m.scrapeErrorsTotalStats.Describe(ch)
	ch <- m.lastScrapeErrorStats
	ch <- m.lastScrapeTimeStats
	ch <- m.lastScrapeDurationStats

	// 收集器指标 - Active
	m.scrapesTotalActive.Describe(ch)
	m.scrapeErrorsTotalActive.Describe(ch)
	ch <- m.lastScrapeErrorActive
	ch <- m.lastScrapeTimeActive
	ch <- m.lastScrapeDurationActive
}

// Collect 实现 prometheus.Collector 接口
func (m *DHCPDMetrics) Collect(ch chan<- prometheus.Metric) {
	// 记录开始时间
	var begun = time.Now()

	// 初始化错误计数
	var err_num = 0

	// 确保 DHCP 信息收集器已初始化
	initDHCPDInfo()

	// 加锁保护共享资源
	mux.Lock()
	defer mux.Unlock()

	// 增加 Stats 收集计数
	m.scrapesTotalStats.Inc()

	// 读取 DHCP 租约信息
	if err := DHCPDInfo.Read(); err != nil {
		// 记录错误日志
		logrus.Errorf("读取 DHCP 租约信息失败: %v", err)

		// 设置错误标志和计数
		err_num = 1
		m.scrapeErrorsTotalStats.Inc()
	}

	// 收集基础统计指标
	ch <- prometheus.MustNewConstMetric(
		m.validLeases,
		prometheus.GaugeValue,
		float64(DHCPDInfo.GetValidLeases()),
	)

	ch <- prometheus.MustNewConstMetric(
		m.expiredLeases,
		prometheus.GaugeValue,
		float64(DHCPDInfo.GetExpiredLeases()),
	)

	ch <- prometheus.MustNewConstMetric(
		m.totalLeases,
		prometheus.GaugeValue,
		float64(DHCPDInfo.GetTotalLeases()),
	)

	ch <- prometheus.MustNewConstMetric(
		m.fileTimestamp,
		prometheus.GaugeValue,
		float64(DHCPDInfo.GetModTime().Unix()),
	)

	// 收集 Stats 收集器指标
	ch <- m.scrapesTotalStats
	ch <- m.scrapeErrorsTotalStats

	ch <- prometheus.MustNewConstMetric(
		m.lastScrapeErrorStats,
		prometheus.GaugeValue,
		float64(err_num),
	)

	ch <- prometheus.MustNewConstMetric(
		m.lastScrapeTimeStats,
		prometheus.GaugeValue,
		float64(time.Now().Unix()),
	)

	ch <- prometheus.MustNewConstMetric(
		m.lastScrapeDurationStats,
		prometheus.GaugeValue,
		time.Since(begun).Seconds(),
	)

	// 重置开始时间和错误计数，用于 Active 收集器
	begun = time.Now()
	err_num = 0

	// 增加 Active 收集计数
	m.scrapesTotalActive.Inc()

	// 收集活跃租约指标
	activeLeases := DHCPDInfo.GetActiveLeases()

	// 记录调试日志
	logrus.Debugf("找到 %d 个活跃租约", len(activeLeases))

	// 遍历活跃租约
	for _, lease := range activeLeases {
		// 为每个活跃租约创建指标
		ch <- prometheus.MustNewConstMetric(
			m.activeLeases,
			prometheus.GaugeValue,
			1,
			lease.Hostname,
			lease.IP,
			lease.HardwareAddress,
		)
	}

	// 收集 Active 收集器指标
	ch <- m.scrapesTotalActive
	ch <- m.scrapeErrorsTotalActive

	ch <- prometheus.MustNewConstMetric(
		m.lastScrapeErrorActive,
		prometheus.GaugeValue,
		float64(err_num),
	)

	ch <- prometheus.MustNewConstMetric(
		m.lastScrapeTimeActive,
		prometheus.GaugeValue,
		float64(time.Now().Unix()),
	)

	ch <- prometheus.MustNewConstMetric(
		m.lastScrapeDurationActive,
		prometheus.GaugeValue,
		time.Since(begun).Seconds(),
	)
}

// 初始化函数，在包被导入时自动执行
func init() {
	// 记录注册信息
	logrus.Info("=== 注册 DHCP 租约指标收集器 ===")

	// 注册指标收集器
	prometheus.MustRegister(NewDHCPDMetrics())

	// 记录注册完成信息
	logrus.Info("DHCP Leases Exporter 指标注册完成")
}

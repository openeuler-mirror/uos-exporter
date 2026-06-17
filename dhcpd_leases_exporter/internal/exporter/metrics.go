package exporter

import (
	"dhcpd_leases_exporter/pkg/collector"
	"github.com/prometheus/client_golang/prometheus"
)

// MetricsCollector 实现 prometheus.Collector 接口
type MetricsCollector struct {
	leasesFile string
	dhcpdInfo  *collector.DHCPDInfo

	validLeases   *prometheus.Desc
	expiredLeases *prometheus.Desc
	totalLeases   *prometheus.Desc
	fileTime      *prometheus.Desc
	activeLease   *prometheus.Desc
}

// NewMetricsCollector 创建一个新的指标收集器
func NewMetricsCollector(leasesFile string) *MetricsCollector {
	dhcpdInfo := collector.NewDHCPDInfo(leasesFile)

	return &MetricsCollector{
		leasesFile: leasesFile,
		dhcpdInfo:  dhcpdInfo,

		validLeases: prometheus.NewDesc(
			"dhcpd_leases_valid_total",
			"当前有效的 DHCP 租约数量",
			nil, nil,
		),
		expiredLeases: prometheus.NewDesc(
			"dhcpd_leases_expired_total",
			"已过期的 DHCP 租约数量",
			nil, nil,
		),
		totalLeases: prometheus.NewDesc(
			"dhcpd_leases_total",
			"DHCP 租约文件中的总租约数量",
			nil, nil,
		),
		fileTime: prometheus.NewDesc(
			"dhcpd_leases_file_time_seconds",
			"DHCP 租约文件的最后修改时间（Unix 时间戳）",
			nil, nil,
		),
		activeLease: prometheus.NewDesc(
			"dhcpd_lease_active",
			"活跃的 DHCP 租约信息",
			[]string{"hostname", "ip", "mac"}, nil,
		),
	}
}

// Describe 实现 prometheus.Collector 接口
func (c *MetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.validLeases
	ch <- c.expiredLeases
	ch <- c.totalLeases
	ch <- c.fileTime
	ch <- c.activeLease
}

// Collect 实现 prometheus.Collector 接口
func (c *MetricsCollector) Collect(ch chan<- prometheus.Metric) {
	// 读取最新的租约信息
	if err := c.dhcpdInfo.Read(); err != nil {
		return
	}

	// 发送统计指标
	ch <- prometheus.MustNewConstMetric(
		c.validLeases,
		prometheus.GaugeValue,
		float64(c.dhcpdInfo.GetValidLeases()),
	)

	ch <- prometheus.MustNewConstMetric(
		c.expiredLeases,
		prometheus.GaugeValue,
		float64(c.dhcpdInfo.GetExpiredLeases()),
	)

	ch <- prometheus.MustNewConstMetric(
		c.totalLeases,
		prometheus.GaugeValue,
		float64(c.dhcpdInfo.GetTotalLeases()),
	)

	ch <- prometheus.MustNewConstMetric(
		c.fileTime,
		prometheus.GaugeValue,
		float64(c.dhcpdInfo.GetModTime().Unix()),
	)

	// 发送活跃租约指标
	for _, lease := range c.dhcpdInfo.GetActiveLeases() {
		ch <- prometheus.MustNewConstMetric(
			c.activeLease,
			prometheus.GaugeValue,
			1,
			lease.Hostname,
			lease.IP,
			lease.HardwareAddress,
		)
	}
}

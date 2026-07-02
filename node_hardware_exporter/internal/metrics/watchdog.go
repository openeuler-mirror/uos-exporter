package metrics

import (
	"errors"
	"node_hardware_exporter/internal/exporter"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs/sysfs"
)

func init() {
	exporter.Register(NewWatchdogCollector())
}

type WatchdogCollector struct {
	*baseMetrics
	fs                    sysfs.FS
	watchdogBootstatusDesc *prometheus.Desc
	watchdogFwVersionDesc  *prometheus.Desc
	watchdogNowayoutDesc   *prometheus.Desc
	watchdogTimeleftDesc   *prometheus.Desc
	watchdogTimeoutDesc    *prometheus.Desc
	watchdogPretimeoutDesc *prometheus.Desc
	watchdogAccessCs0Desc  *prometheus.Desc
	watchdogInfoDesc       *prometheus.Desc
}

func NewWatchdogCollector() *WatchdogCollector {
	fs, err := sysfs.NewFS("/sys")
	if err != nil {
		return &WatchdogCollector{
			baseMetrics: NewMetrics("node_watchdog_collector", "Watchdog collector metrics", []string{}),
		}
	}

	return &WatchdogCollector{
		baseMetrics: NewMetrics("node_watchdog_collector", "Watchdog collector metrics", []string{}),
		fs:          fs,
		watchdogBootstatusDesc: prometheus.NewDesc(
			"node_watchdog_bootstatus",
			"Value of /sys/class/watchdog/<watchdog>/bootstatus",
			[]string{"name"}, nil,
		),
		watchdogFwVersionDesc: prometheus.NewDesc(
			"node_watchdog_fw_version",
			"Value of /sys/class/watchdog/<watchdog>/fw_version",
			[]string{"name"}, nil,
		),
		watchdogNowayoutDesc: prometheus.NewDesc(
			"node_watchdog_nowayout",
			"Value of /sys/class/watchdog/<watchdog>/nowayout",
			[]string{"name"}, nil,
		),
		watchdogTimeleftDesc: prometheus.NewDesc(
			"node_watchdog_timeleft_seconds",
			"Value of /sys/class/watchdog/<watchdog>/timeleft",
			[]string{"name"}, nil,
		),
		watchdogTimeoutDesc: prometheus.NewDesc(
			"node_watchdog_timeout_seconds",
			"Value of /sys/class/watchdog/<watchdog>/timeout",
			[]string{"name"}, nil,
		),
		watchdogPretimeoutDesc: prometheus.NewDesc(
			"node_watchdog_pretimeout_seconds",
			"Value of /sys/class/watchdog/<watchdog>/pretimeout",
			[]string{"name"}, nil,
		),
		watchdogAccessCs0Desc: prometheus.NewDesc(
			"node_watchdog_access_cs0",
			"Value of /sys/class/watchdog/<watchdog>/access_cs0",
			[]string{"name"}, nil,
		),
		watchdogInfoDesc: prometheus.NewDesc(
			"node_watchdog_info",
			"Info of /sys/class/watchdog/<watchdog>",
			[]string{"name", "options", "identity", "state", "status", "pretimeout_governor"}, nil,
		),
	}
}

func toLabelValue(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func (c *WatchdogCollector) Collect(ch chan<- prometheus.Metric) {
	if c.fs == (sysfs.FS{}) {
		fs, err := sysfs.NewFS("/sys")
		if err != nil {
			return
		}
		c.fs = fs
		
		// 初始化metrics描述符
		c.watchdogBootstatusDesc = prometheus.NewDesc(
			"node_watchdog_bootstatus",
			"Value of /sys/class/watchdog/<watchdog>/bootstatus",
			[]string{"name"}, nil,
		)
		c.watchdogFwVersionDesc = prometheus.NewDesc(
			"node_watchdog_fw_version",
			"Value of /sys/class/watchdog/<watchdog>/fw_version",
			[]string{"name"}, nil,
		)
		c.watchdogNowayoutDesc = prometheus.NewDesc(
			"node_watchdog_nowayout",
			"Value of /sys/class/watchdog/<watchdog>/nowayout",
			[]string{"name"}, nil,
		)
		c.watchdogTimeleftDesc = prometheus.NewDesc(
			"node_watchdog_timeleft_seconds",
			"Value of /sys/class/watchdog/<watchdog>/timeleft",
			[]string{"name"}, nil,
		)
		c.watchdogTimeoutDesc = prometheus.NewDesc(
			"node_watchdog_timeout_seconds",
			"Value of /sys/class/watchdog/<watchdog>/timeout",
			[]string{"name"}, nil,
		)
		c.watchdogPretimeoutDesc = prometheus.NewDesc(
			"node_watchdog_pretimeout_seconds",
			"Value of /sys/class/watchdog/<watchdog>/pretimeout",
			[]string{"name"}, nil,
		)
		c.watchdogAccessCs0Desc = prometheus.NewDesc(
			"node_watchdog_access_cs0",
			"Value of /sys/class/watchdog/<watchdog>/access_cs0",
			[]string{"name"}, nil,
		)
		c.watchdogInfoDesc = prometheus.NewDesc(
			"node_watchdog_info",
			"Info of /sys/class/watchdog/<watchdog>",
			[]string{"name", "options", "identity", "state", "status", "pretimeout_governor"}, nil,
		)
	}

	watchdogClass, err := c.fs.WatchdogClass()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission) || errors.Is(err, os.ErrInvalid) {
			return
		}
		return
	}

	for _, wd := range watchdogClass {
		if wd.Bootstatus != nil {
			ch <- prometheus.MustNewConstMetric(c.watchdogBootstatusDesc, prometheus.GaugeValue, float64(*wd.Bootstatus), wd.Name)
		}
		if wd.FwVersion != nil {
			ch <- prometheus.MustNewConstMetric(c.watchdogFwVersionDesc, prometheus.GaugeValue, float64(*wd.FwVersion), wd.Name)
		}
		if wd.Nowayout != nil {
			ch <- prometheus.MustNewConstMetric(c.watchdogNowayoutDesc, prometheus.GaugeValue, float64(*wd.Nowayout), wd.Name)
		}
		if wd.Timeleft != nil {
			ch <- prometheus.MustNewConstMetric(c.watchdogTimeleftDesc, prometheus.GaugeValue, float64(*wd.Timeleft), wd.Name)
		}
		if wd.Timeout != nil {
			ch <- prometheus.MustNewConstMetric(c.watchdogTimeoutDesc, prometheus.GaugeValue, float64(*wd.Timeout), wd.Name)
		}
		if wd.Pretimeout != nil {
			ch <- prometheus.MustNewConstMetric(c.watchdogPretimeoutDesc, prometheus.GaugeValue, float64(*wd.Pretimeout), wd.Name)
		}
		if wd.AccessCs0 != nil {
			ch <- prometheus.MustNewConstMetric(c.watchdogAccessCs0Desc, prometheus.GaugeValue, float64(*wd.AccessCs0), wd.Name)
		}

		ch <- prometheus.MustNewConstMetric(c.watchdogInfoDesc, prometheus.GaugeValue, 1.0,
			wd.Name, toLabelValue(wd.Options), toLabelValue(wd.Identity), toLabelValue(wd.State), toLabelValue(wd.Status), toLabelValue(wd.PretimeoutGovernor))
	}
} 
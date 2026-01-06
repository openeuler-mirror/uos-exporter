package metrics

import (
	"log/slog"

	"node_system_exporter/internal/exporter"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sys/unix"
)

func init() {
	exporter.Register(NewUnameCollector())
}

type UnameCollector struct {
	*baseMetrics
	info   *prometheus.Desc
	logger *slog.Logger
}

type uname struct {
	SysName    string
	Release    string
	Version    string
	Machine    string
	NodeName   string
	DomainName string
}

func NewUnameCollector() *UnameCollector {
	return &UnameCollector{
		info: prometheus.NewDesc(
			prometheus.BuildFQName("node", "uname", "info"),
			"Labeled system information as provided by the uname system call.",
			[]string{
				"sysname",
				"release",
				"version",
				"machine",
				"nodename",
				"domainname",
			},
			nil,
		),
		logger: slog.Default(),
	}
}

func (c *UnameCollector) Collect(ch chan<- prometheus.Metric) {
	uname, err := c.getUname()
	if err != nil {
		c.logger.Error("Error getting uname information", "error", err)
		return
	}

	ch <- prometheus.MustNewConstMetric(
		c.info,
		prometheus.GaugeValue,
		1,
		uname.SysName,
		uname.Release,
		uname.Version,
		uname.Machine,
		uname.NodeName,
		uname.DomainName,
	)
}

func (c *UnameCollector) getUname() (uname, error) {
	var utsname unix.Utsname
	if err := unix.Uname(&utsname); err != nil {
		return uname{}, err
	}

	output := uname{
		SysName:    unix.ByteSliceToString(utsname.Sysname[:]),
		Release:    unix.ByteSliceToString(utsname.Release[:]),
		Version:    unix.ByteSliceToString(utsname.Version[:]),
		Machine:    unix.ByteSliceToString(utsname.Machine[:]),
		NodeName:   unix.ByteSliceToString(utsname.Nodename[:]),
		DomainName: unix.ByteSliceToString(utsname.Domainname[:]),
	}

	return output, nil
}

func (c *UnameCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.info
} 
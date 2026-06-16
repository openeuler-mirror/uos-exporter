package exporter

import (
	"dhcpd_leases_exporter/pkg/logger"
	"dhcpd_leases_exporter/pkg/utils"
	"fmt"
	"github.com/alecthomas/kingpin/v2"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"os"
	"time"
)

var (
	Configfile = kingpin.Flag("config", "Configuration file").Short('c').Default("/etc/uos-exporter/dhcpd-leases-exporter.yaml").String()
	// 注释掉重复的命令行参数，这些已经在 main.go 中定义
	// ListenAddress = kingpin.Flag("web.listen-address", "Address to listen on for web interface and telemetry").Default("0.0.0.0:8090").String()
	// MetricsPath   = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics").Default("/metrics").String()
	DefaultConfig = Config{
		Logging: logger.Config{
			Level:   "debug",
			LogPath: "/var/log/uos-export/dhcpd-leases-exporter.log",
			MaxSize: "10MB",
			MaxAge:  time.Hour * 24 * 7},
		Address:     "0.0.0.0",
		Port:        8090,
		MetricsPath: "/metrics",
	}
)


// TODO: implement functions

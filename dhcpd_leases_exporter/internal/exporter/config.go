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

func init() {
	kingpin.HelpFlag.Short('h')
	// Configfile = kingpin.Flag("config", "Configuration file").
	//
	//	Short('c').
	//	Default("/etc/uos-exporter/dhcpd_leases_exporter.yaml").
	//	String()
}

type Config struct {
	Logging     logger.Config `yaml:"log"`
	Address     string        `yaml:"address"`
	Port        int           `yaml:"port"`
	MetricsPath string        `yaml:"metricsPath"`
}

func Unpack(config interface{}) error {
	// 尝试将默认配置复制到提供的 config 中
	c, ok := config.(*Config)
	if ok {
		*c = DefaultConfig
	}

	// 如果没有找到配置文件，直接使用默认配置
	if !utils.FileExists(*Configfile) {
		logrus.Warnf("配置文件 %s 未找到，使用默认配置", *Configfile)

		// 由于 ListenAddress 和 MetricsPath 已经被注释掉，这里不再需要检查它们
		// 如果需要从命令行获取这些值，应该在 main.go 中处理

		return nil
	}

	file, err := os.Open(*Configfile)
	if err != nil {
		logrus.Error("打开配置文件失败: ", err)
		return fmt.Errorf("打开配置文件失败: %v", err)
	}
	defer file.Close()

	err = yaml.NewDecoder(file).Decode(config)
	if err != nil {
		logrus.Error("解析配置文件失败: ", err)
		return fmt.Errorf("解析配置文件失败: %v", err)
	}

	return nil
}

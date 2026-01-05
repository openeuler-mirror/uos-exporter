package exporter

import (
	"fmt"
	"network_exporter/pkg/logger"
	"network_exporter/pkg/utils"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var (
	Configfile        *string
	NetworkConfigfile *string
	DefaultConfig     = Config{
		Logging: logger.Config{
			Level:   "debug",
			LogPath: "./network_exporter.log",
			MaxSize: "10MB",
			MaxAge:  time.Hour * 24 * 7},
		Address:     "127.0.0.1",
		Port:        9118,
		MetricsPath: "/metrics",
	}
)

func init() {
	Configfile = kingpin.Flag("config", "Configuration file").
		Short('c').
		Default("/etc/uos-exporter/network-exporter.yaml").
		String()

	// 添加兼容旧项目的命令行参数
	NetworkConfigfile = kingpin.Flag("config.file", "Network exporter configuration file").
		Default("/etc/uos-exporter/network-exporter.yaml").
		String()
}

type Config struct {
	Logging     logger.Config `yaml:"log"`
	Address     string        `yaml:"address"`
	Port        int           `yaml:"port"`
	MetricsPath string        `yaml:"metricsPath"`
}

func Unpack(config interface{}) error {
	configPath := *Configfile
	if utils.FileExists(*NetworkConfigfile) {
		// 如果使用了旧项目的命令行参数格式，优先使用它
		configPath = *NetworkConfigfile
	}
	cleanPath := filepath.Clean(configPath)
	configDir := "/etc/uos-exporter"
	if !strings.HasPrefix(cleanPath, configDir) {
		return fmt.Errorf("config file must be located within %s", configDir)
	}
	if !utils.FileExists(cleanPath) {
		logrus.Errorf("%s file not found", configPath)
	} else {
		file, err := os.Open(configPath)
		if err != nil {
			logrus.Error("Failed to open config file: ", err)
			return err
		}
		err = yaml.NewDecoder(file).Decode(config)
		if err != nil {
			return err
		}
	}
	return nil
}

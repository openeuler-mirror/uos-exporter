package exporter

import (
	"node_network_exporter/pkg/utils"
	"os"
	"github.com/sirupsen/logrus"
	"github.com/alecthomas/kingpin/v2"
	"gopkg.in/yaml.v2"
)

var (
	Configfile    *string
	DefaultConfig = Config{
		Logging: Logging{
			Level:   "info",
			LogPath: "/var/log/uos-exporter/node_network_exporter.log",
			MaxSize: "10MB",
			MaxAge:  7,
		},
		Address:     "0.0.0.0",
		Port:        9122,
		MetricsPath: "/metrics",
	}
)

func init() {
	kingpin.HelpFlag.Short('h')
	Configfile = kingpin.Flag("config", "Configuration file").
		Short('c').
		Default("/etc/uos-exporter/node-network-exporter.yaml").
		String()
}

type Logging struct {
	Level   string `yaml:"level"`
	LogPath string `yaml:"log_path"`
	MaxSize string `yaml:"max_size"`
	MaxAge  int    `yaml:"max_age"`
}

type Config struct {
	Logging     Logging `yaml:"log"`
	Address     string  `yaml:"address"`
	Port        int     `yaml:"port"`
	MetricsPath string  `yaml:"metricsPath"`
}

func Unpack(config interface{}) error {
	if !utils.FileExists(*Configfile) {
		logrus.Errorf("%s file not found", *Configfile)
	} else {
		file, err := os.Open(*Configfile)
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

package exporter

import (
	"os"
	"pacemaker_exporter/pkg/logger"
	"pacemaker_exporter/pkg/utils"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var (
	Configfile    *string
	DefaultConfig = Config{
		Logging: logger.Config{
			Level:   "debug",
			LogPath: "/var/log/exporter.log",
			MaxSize: "10MB",
			MaxAge:  time.Hour * 24 * 7},
		Address:     "127.0.0.1",
		Port:        8080,
		MetricsPath: "/metrics",
	}
)

func init() {
	kingpin.HelpFlag.Short('h')
	Configfile = kingpin.Flag("config", "Configuration file").
		Short('c').
		Default("/etc/uos-exporter/pacemaker-exporter.yaml").
		String()
}

type Config struct {
	Logging     logger.Config `yaml:"log"`
	Address     string        `yaml:"address"`
	Port        int           `yaml:"port"`
	MetricsPath string        `yaml:"metricsPath"`
}

func Unpack(config interface{}) error {
	if !utils.FileExists(*Configfile) {
		logrus.Errorf("%s file not found", *Configfile)
		logrus.Debug("Use default config")
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

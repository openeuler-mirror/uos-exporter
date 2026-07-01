package exporter

import (
	"node_service_exporter/pkg/logger"
	"node_service_exporter/pkg/utils"
	"github.com/alecthomas/kingpin"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"os"
	"time"
)

var (
	Configfile    *string
	DefaultConfig = Config{
		Logging: logger.Config{
			Level:   "debug",
			LogPath: "/var/log/uos-exporter/node_service_exporter.log",
			MaxSize: "10MB",
			MaxAge:  time.Hour * 24 * 7},
		Address:     "0.0.0.0",
		Port:        9125,
		MetricsPath: "/metrics",
	}
)

func init() {
	Configfile = kingpin.Flag("config", "Configuration file").
		Short('c').
		Default("/etc/uos-exporter/node-service-exporter.yaml").
		String()
}

type Config struct {
	Logging          logger.Config `yaml:"log"`
	Address          string        `yaml:"address"`
	Port             int           `yaml:"port"`
	MetricsPath      string        `yaml:"metricsPath"`
	TelemetryAddress string        `yaml:"telemetryAddress"`
	TelemetryPort    int           `yaml:"telemetryPort"`
	TelemetryPath    string        `yaml:"telemetryPath"`
	LogLevel         string        `yaml:"logLevel"`
	LogPath          string        `yaml:"logPath"`
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

func NewConfig() *Config {
	return &Config{
		Logging: logger.Config{
			Level:   "debug",
			LogPath: "/var/log/node-service-exporter/node_service_exporter.log",
			MaxSize: "10MB",
			MaxAge:  time.Hour * 24 * 7},
		Address:     "0.0.0.0",
		Port:        9125,
		MetricsPath: "/metrics",
	}
}
// Part 2 commit for node_service_exporter/internal/exporter/config.go

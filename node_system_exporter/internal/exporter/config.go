package exporter

import (
	"node_system_exporter/pkg/logger"
	"node_system_exporter/pkg/utils"
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
			LogPath: "/var/log/uos-exporter/node_system_exporter.log",
			MaxSize: "10MB",
			MaxAge:  time.Hour * 24 * 7},
		Address:     "0.0.0.0",
		Port:        9120,
		MetricsPath: "/metrics",
		LogLevel:    "info",
	}
)


// TODO: implement

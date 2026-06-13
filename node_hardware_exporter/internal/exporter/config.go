package exporter

import (
	"node_hardware_exporter/pkg/logger"
	"node_hardware_exporter/pkg/utils"
	"github.com/alecthomas/kingpin/v2"
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
			LogPath: "/var/log/uos-exporter/node_hardware_exporter.log",
			MaxSize: "10MB",
			MaxAge:  time.Hour * 24 * 7},
		Address:     "127.0.0.1",
		Port:        9124,
		MetricsPath: "/metrics",
	}
)


// TODO: implement functions

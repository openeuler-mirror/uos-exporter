package exporter

import (
	"keepalived_container_exporter/pkg/logger"
	"keepalived_container_exporter/pkg/utils"
	"os"
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


// TODO: implement functions

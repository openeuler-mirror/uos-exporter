package exporter

import (
	"github.com/alecthomas/kingpin"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"os"
	"tc_exporter/pkg/logger"
	"tc_exporter/pkg/utils"
	"time"
)

var (
	Configfile    *string
	DefaultConfig = Config{
		Logging: logger.Config{
			Level:   "debug",
			LogPath: "/var/log/tc-exporter.log",
			MaxSize: "10MB",
			MaxAge:  time.Hour * 24 * 7},
		Address:     "127.0.0.1",
		Port:        9062,
		MetricsPath: "/metrics",
	}
)


// TODO: implement

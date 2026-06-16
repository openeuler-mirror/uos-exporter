package exporter

import (
	"ssl_exporter/pkg/logger"
	"ssl_exporter/pkg/utils"
	"github.com/alecthomas/kingpin"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"net/url"
	"os"
	"time"
)

var (
	Configfile    *string
	DefaultConfig = Config{
		Logging: logger.Config{
			Level:   "debug",
			LogPath: "/var/log/uos-exporter/ssl_exporter.log",
			MaxSize: "10MB",
			MaxAge:  time.Hour * 24 * 7},
		Address:     "127.0.0.1",
		Port:        9219,
		MetricsPath: "/metrics",
		SSL: SSLConfig{
			DefaultModule: "https",
			Modules: map[string]ModuleConfig{
				"https": {
					Prober: "https",
				},
				"tcp": {
					Prober: "tcp",
				},
			},
		},
	}
)


// TODO: implement functions

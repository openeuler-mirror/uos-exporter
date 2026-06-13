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


// TODO: implement functions

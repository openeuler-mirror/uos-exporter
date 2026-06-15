package exporter

import (
	"newrelic_exporter/pkg/logger"
	"newrelic_exporter/pkg/utils"
	"newrelic_exporter/pkg/cmdline"
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
			LogPath: "/var/log/uos-exporter/newrelic_exporter.log",
			MaxSize: "10MB",
			MaxAge:  time.Hour * 24 * 7},
		Address:     "127.0.0.1",
		Port:        9126,
		MetricsPath: "/metrics",
		NewRelic: NewRelicConfig{
			ApiKey:               "",
			ApiServer:            "https://api.newrelic.com",
			Period:               60,
			Timeout:              time.Second * 15,
			Service:              "applications",
			AppsListCacheTime:    time.Hour,
			MetricNamesCacheTime: time.Hour,
			MetricFilters: []string{
				"Apdex",
				"WebTransaction",
				"HttpDispatcher",
				"Database",
				"CPU", 
				"Memory",
			},
			ValueFilters: []string{},
		},
	}
)


// TODO: implement functions

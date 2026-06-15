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

func init() {
	Configfile = cmdline.App.Flag("config", "Configuration file").
		Short('c').
		Default("/etc/uos-exporter/newrelic-exporter.yaml").
		String()
}

type Config struct {
	Logging     logger.Config   `yaml:"log"`
	Address     string          `yaml:"address"`
	Port        int             `yaml:"port"`
	MetricsPath string          `yaml:"metricsPath"`
	NewRelic    NewRelicConfig  `yaml:"newrelic"`
}

type NewRelicConfig struct {
	ApiKey               string        `yaml:"api_key"`
	ApiServer            string        `yaml:"api_server"`
	Period               int           `yaml:"period"`
	Timeout              time.Duration `yaml:"timeout"`
	Service              string        `yaml:"service"`
	AppsListCacheTime    time.Duration `yaml:"apps_list_cache_time"`
	MetricNamesCacheTime time.Duration `yaml:"metric_names_cache_time"`
	MetricFilters        []string      `yaml:"metric_filters"`
	ValueFilters         []string      `yaml:"value_filters"`
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

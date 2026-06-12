package config

import (
	"fmt"
	"net"
	"network_exporter/pkg/utils"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var (
	ScrapeUrl       *string
	Insecure        *bool
	DefaultSettings = Settings{
		//ScrapeUri: "http://127.0.0.1:24220/api/plugins.json",
		Insecure: false,
	}
)

func init() {
	ScrapeUrl = kingpin.Flag("scrape_uri",
		"Scrape URI").
		Short('s').
		String()
	Insecure = kingpin.Flag("insecure",
		"Ignore server certificate if using https, Default: false.").
		Bool()
	if *ScrapeUrl != "" {
		if err := utils.ValidateURI(*ScrapeUrl); err != nil {
			logrus.Warnf("Invalid scrape uri: %s", err)
			logrus.Warnf("Use default scrape uri: %s", DefaultSettings.ScrapeUri)
			*ScrapeUrl = DefaultSettings.ScrapeUri
		}
	}

	if *Insecure {
		logrus.Warn("Insecure mode enabled, this is not recommended for production use.")
	}
}

type Settings struct {
	ScrapeUri string `yaml:"scrape_uri"`
	Insecure  bool   `yaml:"insecure"`
}

// 网络监控相关配置

type Targets []struct {
	Name     string   `yaml:"name" json:"name"`
	Host     string   `yaml:"host" json:"host"`
	Port     string   `yaml:"port" json:"port"`
	Type     string   `yaml:"type" json:"type"`
	Proxy    string   `yaml:"proxy" json:"proxy"`
	Probe    []string `yaml:"probe" json:"probe"`
	SourceIp string   `yaml:"source_ip" json:"source_ip"`
	Labels   extraKV  `yaml:"labels,omitempty" json:"labels,omitempty"`
}

type HTTPGet struct {
	Interval duration `yaml:"interval" json:"interval" default:"15s"`
	Timeout  duration `yaml:"timeout" json:"timeout" default:"14s"`
}

type TCP struct {

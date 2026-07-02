package config

import (
	"syslog_ng_exporter/pkg/utils"
	"github.com/alecthomas/kingpin"
	"github.com/sirupsen/logrus"
)

var (
	ScrapeUrl       *string
	Insecure        *bool
	SocketPath      *string
	DefaultSettings = Settings{
		//ScrapeUri: "http://127.0.0.1:24220/api/plugins.json",
		Insecure:   false,
		SocketPath: "/var/lib/syslog-ng/syslog-ng.ctl",
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
	SocketPath = kingpin.Flag("socket.path",
		"Path to syslog-ng control socket, Default: /var/lib/syslog-ng/syslog-ng.ctl").
		Default("/var/lib/syslog-ng/syslog-ng.ctl").
		String()
		
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
	ScrapeUri  string `yaml:"scrape_uri"`
	Insecure   bool   `yaml:"insecure"`
	SocketPath string `yaml:"socket.path"`
}

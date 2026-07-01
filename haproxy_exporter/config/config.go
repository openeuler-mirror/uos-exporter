package config

import (
	"github.com/alecthomas/kingpin"
	"github.com/sirupsen/logrus"
	"haproxy_exporter/pkg/utils"
)

var (
	ScrapeUrl       *string
	Insecure        *bool
	DefaultSettings = Settings{
		ScrapeUri: "http://127.0.0.1:24220/api/plugins.json",
		Insecure:  false,
	}
)

func init() {
	ScrapeUrl = kingpin.Flag("scrape_uri",
		"haproxy Scrape URI").
		Short('s').
		Default("http://127.0.0.1:8404/haproxy?stats;csv").
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

package config

import (
	"squid_exporter/pkg/utils"
	"github.com/alecthomas/kingpin"
	"github.com/sirupsen/logrus"
)

var (
	ScrapeUrl       *string
	Insecure        *bool
	SquidHostname   *string
	SquidPort       *int
	Login           *string
	Password        *string
	ExtractTimes    *bool
	DefaultSettings = Settings{
		//ScrapeUri: "http://127.0.0.1:24220/api/plugins.json",
		Insecure: false,
		SquidHostname: "localhost",
		SquidPort:     3128,
		Login:         "",
		Password:      "",
		ExtractTimes:  true,
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
	SquidHostname = kingpin.Flag("squid.hostname",
		"Hostname of the Squid server").
		Default("localhost").
		String()
	SquidPort = kingpin.Flag("squid.port",
		"Port of the Squid server").
		Default("3128").
		Int()
	Login = kingpin.Flag("squid.login",
		"Login for the Squid server").
		Default("").
		String()
	Password = kingpin.Flag("squid.password",
		"Password for the Squid server").
		Default("").
		String()
	ExtractTimes = kingpin.Flag("squid.extractTimes",
		"Extract service time metrics").
		Default("true").
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
	SquidHostname string `yaml:"hostname"`
	SquidPort     int    `yaml:"port"`
	Login         string `yaml:"login"`
	Password      string `yaml:"password"`
	ExtractTimes  bool   `yaml:"extractTimes"`
}

type SquidSettings struct {
	Settings Settings `yaml:"squid"`
}

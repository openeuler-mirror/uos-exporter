package config

import (
	"elasticsearch_exporter/pkg/utils"
	"github.com/alecthomas/kingpin/v2"
	"github.com/sirupsen/logrus"
)

var (
	ScrapeUrl       *string
	Insecure        *bool
	TasksActionsFilter *string
	DefaultSettings = Settings{
		//ScrapeUri: "http://127.0.0.1:24220/api/plugins.json",
		Insecure: false,
		TasksActionsFilter: "indices:*",
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
	TasksActionsFilter = kingpin.Flag("tasks.actions",
		"Filter on task actions. Used in same way as Task API actions param, Default: indices:*").
		Default("indices:*").
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
	ScrapeUri string `yaml:"scrape_uri"`
	Insecure  bool   `yaml:"insecure"`
	TasksActionsFilter string `yaml:"tasks_actions_filter"`
}
// Part 2 commit for elasticsearch_exporter/config/config.go

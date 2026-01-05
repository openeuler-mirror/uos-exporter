package config

import (
	"github.com/sirupsen/logrus"

	"newrelic_exporter/pkg/cmdline"
	"newrelic_exporter/pkg/utils"
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
	ScrapeUrl = cmdline.App.Flag("scrape_uri",
		"Scrape URI").
		Short('s').
		String()
	Insecure = cmdline.App.Flag("insecure",
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

type Proxy struct {
	Url      string `yaml:"url"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Insecure bool   `yaml:"insecure"`
}

// func LoadConfig(configFile string) (*exporter.Config, error) {
// 	cfg := exporter.DefaultConfig
// 	err := load(configFile, &cfg)
// 	if err != nil {
// 		logrus.Errorf("Load config error: %v, use default config.", err)
// 	}
// 	return &cfg, nil
// }

// func load(configFile string, cfg interface{}) error {
// 	if configFile == "" {
// 		_, filename, _, ok := runtime.Caller(0)
// 		if !ok {
// 			logrus.Error("Failed to get caller information")
// 			return nil
// 		}
// 		configFile = filepath.Join(filepath.Dir(filename), "newrelic-exporter.yaml")
// 	}

// 	bytes, err := ioutil.ReadFile(configFile)
// 	if err != nil {
// 		logrus.Errorf("Read config file error: %v", err)
// 		return err
// 	}

// 	err = yaml.Unmarshal(bytes, cfg)
// 	if err != nil {
// 		logrus.Errorf("Parse config error: %v", err)
// 		return err
// 	}

// 	return nil
// }

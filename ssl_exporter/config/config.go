package config

import (
	"time"

	"ssl_exporter/pkg/utils"

	"github.com/alecthomas/kingpin"
	"github.com/sirupsen/logrus"
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

// SSLConfig 配置 SSL Exporter
type SSLConfig struct {
	DefaultModule string            `yaml:"default_module"`
	Modules       map[string]Module `yaml:"modules"`
}

// Module 配置探针
type Module struct {
	Prober    string        `yaml:"prober,omitempty"`
	Target    string        `yaml:"target,omitempty"`
	Timeout   time.Duration `yaml:"timeout,omitempty"`
	TLSConfig TLSConfig     `yaml:"tls_config,omitempty"`
	TCP       TCPProbe      `yaml:"tcp,omitempty"`
}

// TLSConfig 配置 TLS
type TLSConfig struct {
	CAFile             string `yaml:"ca_file,omitempty"`
	CertFile           string `yaml:"cert_file,omitempty"`
	KeyFile            string `yaml:"key_file,omitempty"`
	ServerName         string `yaml:"server_name,omitempty"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify"`
	Renegotiation      string `yaml:"renegotiation,omitempty"`
}

// TCPProbe 配置 TCP 探针
type TCPProbe struct {
	StartTLS string `yaml:"starttls,omitempty"`
}

// LoadConfig 从文件加载配置
// func LoadConfig(path string) (*SSLConfig, error) {
// 	data, err := ioutil.ReadFile(path)
// 	if err != nil {
// 		return nil, fmt.Errorf("read config file: %v", err)
// 	}

// 	var cfg SSLConfig
// 	if err := yaml.Unmarshal(data, &cfg); err != nil {
// 		return nil, fmt.Errorf("parse config file: %v", err)
// 	}

// 	return &cfg, nil
// }

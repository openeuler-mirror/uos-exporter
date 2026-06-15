package config

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"openldap_exporter/pkg/utils"

	"github.com/alecthomas/kingpin"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var (
	ScrapeUrl       *string
	Insecure        *bool
	DefaultSettings = Settings{
		// ScrapeUri: "http://127.0.0.1:24220/api/plugins.json",
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

// OpenLDAPConfig 包含OpenLDAP exporter的所有配置
type OpenLDAPConfig struct {
	LDAP LDAPClientConfig `yaml:"ldap"`
}

// LDAPClientConfig 是连接 OpenLDAP 所需的配置
type LDAPClientConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	BindDN   string `yaml:"bind_dn"`
	BindPass string `yaml:"bind_password"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() OpenLDAPConfig {
	return OpenLDAPConfig{
		LDAP: LDAPClientConfig{
			Host:     "127.0.0.1",
			Port:     "1389",
			BindDN:   "cn=admin,dc=example,dc=com",
			BindPass: "admin",
		},
	}
}

// LoadConfig 从文件加载配置
func LoadConfig(path string) (*OpenLDAPConfig, error) {
	cleanPath := filepath.Clean(path)
	data, err := ioutil.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("read config file: %v", err)
	}

	var cfg OpenLDAPConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %v", err)
	}

	// 设置默认值
	if cfg.LDAP.Host == "" {
		cfg.LDAP.Host = "127.0.0.1"
	}
	if cfg.LDAP.Port == "" {
		cfg.LDAP.Port = "389"
	}

	return &cfg, nil
}

// Validate 验证配置是否有效
func (cfg *OpenLDAPConfig) Validate() error {
	if cfg.LDAP.Host == "" {
		return fmt.Errorf("LDAP host is required")
	}
	if cfg.LDAP.Port == "" {
		return fmt.Errorf("LDAP port is required")
	}
	return nil
}

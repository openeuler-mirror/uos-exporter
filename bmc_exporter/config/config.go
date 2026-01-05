package config

import (
	"bmc_exporter/pkg/utils"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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

type BMCConfig struct {
	Host     string        `yaml:"host"`
	User     string        `yaml:"user"`
	Password string        `yaml:"password"`
	Timeout  time.Duration `yaml:"timeout"`
	Retries  int           `yaml:"retries"`
	CacheTTL time.Duration `yaml:"cache_ttl"`
	// Power    PowerConfig   `yaml:"power"`
}

type Config struct {
	BMC *BMCConfig `yaml:"bmc"`
}

// type PowerConfig struct {
// 	RefreshInterval  time.Duration `yaml:"refresh_interval"`
// 	VoltageThreshold struct {
// 		Min float64 `yaml:"min"`
// 		Max float64 `yaml:"max"`
// 	} `yaml:"voltage_threshold"`
// }

func LoadBMCConfig(path string) (*Config, error) {

	cleanPath := filepath.Clean(path)
	// 限制文件扩展名
	ext := filepath.Ext(cleanPath)
	if ext != ".yaml" && ext != ".yml" && ext != "" {
		return nil, fmt.Errorf("invalid file extension: only .yaml or .yml files are allowed")
	}
	configDir := "/etc/uos-exporter"
	if !strings.HasPrefix(cleanPath, configDir) {
		return nil, fmt.Errorf("config file must be located within %s", configDir)
	}
	file, err := os.Open(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("无法打开配置文件: %v", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("读取配置失败: %v", err)
	}

	var cfg Config // 改为完整配置结构
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析YAML失败: %v", err)
	}

	// 设置BMC默认超时
	if cfg.BMC.Timeout == 0 {
		cfg.BMC.Timeout = 10 * time.Second
	}

	if cfg.BMC.CacheTTL == 0 {
		cfg.BMC.CacheTTL = 30 * time.Second
	}

	return &cfg, nil
}

func (c *Config) GetBMC() *BMCConfig {
	if c.BMC == nil {
		return &BMCConfig{} // 返回空配置防止panic
	}
	return c.BMC
}

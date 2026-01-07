package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"smtp_exporter/pkg/utils"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/kingpin"
	pconfig "github.com/prometheus/common/config"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var (
	DefaultModule = Module{
		SMTP: DefaultSMTPProbe,
	}

	DefaultSMTPProbe = SMTPProbe{
		IPProtocol:         "ip6",
		IPProtocolFallback: true,
		EHLO:               "localhost",
		TLS:                "no",
	}

	ScrapeUrl       *string
	Insecure        *bool
	ConfigFile      *string
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
	ConfigFile = kingpin.Flag("config.file",
		"SMTP exporter configuration file.").
		Default("/etc/uos-exporter/smtp-exporter.yaml").
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

type Config struct {
	Modules     map[string]Module `yaml:"modules"`
	Address     string            `yaml:"address"`
	Port        int               `yaml:"port"`
	MetricsPath string            `yaml:"metricsPath"`
	Log         LogConfig         `yaml:"log"`
}

type LogConfig struct {
	Level   string `yaml:"level"`
	LogPath string `yaml:"log_path"`
}

type SafeConfig struct {
	sync.RWMutex
	C *Config
}

type Settings struct {
	ScrapeUri string `yaml:"scrape_uri"`
	Insecure  bool   `yaml:"insecure"`
}

type SMTPAuth struct {
	Username string         `yaml:"username,omitempty"`
	Password pconfig.Secret `yaml:"password,omitempty"`
}

type Module struct {
	Prober  string        `yaml:"prober"`
	Timeout time.Duration `yaml:"timeout,omitempty"`
	SMTP    SMTPProbe     `yaml:"smtp,omitempty"`
}

type SMTPProbe struct {
	IPProtocol         string            `yaml:"preferred_ip_protocol,omitempty"`
	IPProtocolFallback bool              `yaml:"ip_protocol_fallback,omitempty"`
	TLS                string            `yaml:"tls,omitempty"`
	TLSConfig          pconfig.TLSConfig `yaml:"tls_config,omitempty"`
	Auth               SMTPAuth          `yaml:"auth,omitempty"`
	EHLO               string            `yaml:"ehlo,omitempty"`
	MailFrom           string            `yaml:"mail_from,omitempty"`
	MailTo             string            `yaml:"mail_to,omitempty"`
	Headers            map[string]string `yaml:"headers,omitempty"`
	Body               string            `yaml:"body,omitempty"`
	ValidStatusCodes   []int             `yaml:"valid_status_codes,omitempty"`
}

// LoadConfig 从文件加载配置
func LoadConfig(configFile string) (*Config, error) {
	if configFile == "" {
		// 默认配置文件路径
		configFile = "/etc/uos-exporter/smtp-exporter.yaml"
	}
	cleanConfigFile := filepath.Clean(configFile)
	if !strings.HasPrefix(cleanConfigFile, "/etc/uos-exporter/") {
		return nil, fmt.Errorf("config path must be under %s", "/etc/uos-exporter/")
	}

	filename, _ := filepath.Abs(cleanConfigFile)
	// 首先尝试加载yaml扩展名的文件
	cleanFileName := filepath.Clean(filename)
	if !strings.HasPrefix(cleanFileName, "/etc/uos-exporter/") {
		return nil, fmt.Errorf("config path must be under %s", "/etc/uos-exporter/")
	}
	yamlFile, err := ioutil.ReadFile(cleanFileName)
	if err != nil {
		// 如果找不到yaml扩展名的文件，尝试yml扩展名
		// 检查文件扩展名是否为.yaml
		if filepath.Ext(filename) == ".yaml" {
			// 尝试使用.yml扩展名
			altFilename := filename[:len(filename)-5] + ".yml"
			cleanAltFilename := filepath.Clean(altFilename)
			if !strings.HasPrefix(cleanAltFilename, "/etc/uos-exporter/") {
				return nil, fmt.Errorf("config path must be under %s", "/etc/uos-exporter/")
			}
			yamlFile, err = ioutil.ReadFile(cleanAltFilename)
			if err != nil {
				return nil, fmt.Errorf("读取配置文件错误：%v，也尝试了替代文件 %s", err, cleanAltFilename)
			}
			// 找到了yml文件，记录日志
			logrus.Infof("使用替代配置文件：%s", cleanAltFilename)
		} else {
			return nil, fmt.Errorf("读取配置文件错误：%v", err)
		}
	}

	var config Config
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		return nil, fmt.Errorf("解析配置文件错误：%v", err)
	}

	// 设置默认值
	if config.Port == 0 {
		config.Port = 9125
	}
	if config.Address == "" {
		config.Address = "0.0.0.0"
	}
	if config.MetricsPath == "" {
		config.MetricsPath = "/metrics"
	}

	return &config, nil
}

func (sc *SafeConfig) ReloadConfig(confFile string) (err error) {
	var c = &Config{}

	defer func() {
		// 更新指标的方式将由metrics包统一管理，在此处导入会造成循环依赖
		// 所以这里不再直接设置metrics的值
	}()

	// 如果配置文件路径为空，使用默认路径
	if confFile == "" {
		confFile = "/etc/uos-exporter/smtp-exporter.yaml"
	}

	// 检查文件是否存在
	if _, statErr := os.Stat(confFile); os.IsNotExist(statErr) {
		// 如果找不到yaml扩展名的文件，尝试yml扩展名
		if filepath.Ext(confFile) == ".yaml" {
			// 尝试使用.yml扩展名
			altFile := confFile[:len(confFile)-5] + ".yml"
			if _, altStatErr := os.Stat(altFile); os.IsNotExist(altStatErr) {
				return fmt.Errorf("config file not found: %s (also tried %s)", confFile, altFile)
			} else {
				// 找到了yml文件，切换到这个文件
				confFile = altFile
				logrus.Infof("Using alternative config file: %s", altFile)
			}
		} else {
			return fmt.Errorf("config file not found: %s", confFile)
		}
	}
	cleanPath := filepath.Clean(confFile)
	allowedPrefix := "/etc/uos-exporter/"
	if !strings.HasPrefix(cleanPath, allowedPrefix) {
		return fmt.Errorf("config path must be under %s", allowedPrefix)
	}

	yamlFile, err := os.ReadFile(cleanPath)
	if err != nil {
		return fmt.Errorf("error reading config file: %s", err)
	}

	if err = yaml.Unmarshal(yamlFile, c); err != nil {
		return fmt.Errorf("error parsing config file: %s", err)
	}

	sc.Lock()
	sc.C = c
	sc.Unlock()

	return nil
}

func (sc *SafeConfig) GetModule(moduleName string) (Module, bool) {
	sc.RLock()
	defer sc.RUnlock()

	module, ok := sc.C.Modules[moduleName]
	return module, ok
}
// Part 2 commit for smtp_exporter/config/config.go

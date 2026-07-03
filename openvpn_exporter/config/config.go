package config

import (
	"fmt"
	"openvpn_exporter/pkg/utils"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kingpin"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var (
	// 全局kingpin应用
	App = kingpin.New("openvpn-exporter", "OpenVPN Exporter for Prometheus")

	ScrapeUrl         *string
	Insecure          *bool
	StatusPaths       *string
	IgnoreIndividuals *bool
	DefaultSettings   = Settings{
		//ScrapeUri: "http://127.0.0.1:24220/api/plugins.json",
		Insecure:          false,
		StatusPaths:       "/var/run/openvpn/status",
		IgnoreIndividuals: false,
	}

	// 存储从配置文件加载的设置
	LoadedSettings Settings
)

func init() {
	ScrapeUrl = App.Flag("scrape_uri",
		"Scrape URI").
		Short('s').
		String()
	Insecure = App.Flag("insecure",
		"Ignore server certificate if using https, Default: false.").
		Bool()
	StatusPaths = App.Flag("openvpn.status_paths",
		"Paths at which OpenVPN places its status files (comma separated)").
		Default("/var/run/openvpn/status").
		String()
	IgnoreIndividuals = App.Flag("ignore.individuals",
		"If ignoring metrics for individuals").
		Bool()
}

func Parse() error {
	_, err := App.Parse(os.Args[1:])
	return err
}

// LoadConfigFile 从配置文件加载设置
func LoadConfigFile(configPath string) error {
	if !utils.FileExists(configPath) {
		logrus.Warnf("Config file %s not found, using defaults", configPath)
		LoadedSettings = DefaultSettings
		return nil
	}
	cleanPath := filepath.Clean(configPath)
	if !strings.HasPrefix(cleanPath, "/etc/uos-exporter/") {
		return fmt.Errorf("config file path must be under /etc/uos-exporter/")
	}

	content, err := os.ReadFile(cleanPath)
	if err != nil {
		logrus.Errorf("Failed to read config file %s: %v", configPath, err)
		LoadedSettings = DefaultSettings
		return err
	}

	err = yaml.Unmarshal(content, &LoadedSettings)
	if err != nil {
		logrus.Errorf("Failed to parse config file %s: %v", configPath, err)
		LoadedSettings = DefaultSettings
		return err
	}

	logrus.Infof("Loaded config from file: %s", configPath)
	logrus.Debugf("Config: StatusPaths=%s, IgnoreIndividuals=%v", LoadedSettings.StatusPaths, LoadedSettings.IgnoreIndividuals)

	return nil
}

func ValidateConfig() {
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
	ScrapeUri         string `yaml:"scrape_uri"`
	Insecure          bool   `yaml:"insecure"`
	StatusPaths       string `yaml:"status_paths"`
	IgnoreIndividuals bool   `yaml:"ignore_individuals"`
}

// GetStatusPaths 获取状态文件路径列表，优先使用命令行参数，然后是配置文件，最后是默认值
func GetStatusPaths() []string {
	// 1. 命令行参数优先
	if StatusPaths != nil && *StatusPaths != DefaultSettings.StatusPaths {
		logrus.Debugf("Using status paths from command line: %s", *StatusPaths)
		return strings.Split(*StatusPaths, ",")
	}

	// 2. 配置文件次之
	if LoadedSettings.StatusPaths != "" {
		logrus.Debugf("Using status paths from config file: %s", LoadedSettings.StatusPaths)
		return strings.Split(LoadedSettings.StatusPaths, ",")
	}

	// 3. 默认值最后
	logrus.Debugf("Using default status paths: %s", DefaultSettings.StatusPaths)
	return strings.Split(DefaultSettings.StatusPaths, ",")
}

// GetIgnoreIndividuals 获取是否忽略个体指标，优先使用命令行参数，然后是配置文件，最后是默认值
func GetIgnoreIndividuals() bool {
	// 1. 命令行参数优先
	if IgnoreIndividuals != nil && *IgnoreIndividuals != DefaultSettings.IgnoreIndividuals {
		logrus.Debugf("Using ignore_individuals from command line: %v", *IgnoreIndividuals)
		return *IgnoreIndividuals
	}

	// 2. 配置文件次之
	if LoadedSettings.StatusPaths != "" { // 如果配置文件被加载过
		logrus.Debugf("Using ignore_individuals from config file: %v", LoadedSettings.IgnoreIndividuals)
		return LoadedSettings.IgnoreIndividuals
	}

	// 3. 默认值最后
	logrus.Debugf("Using default ignore_individuals: %v", DefaultSettings.IgnoreIndividuals)
	return DefaultSettings.IgnoreIndividuals
}
// Part 2 commit for openvpn_exporter/config/config.go

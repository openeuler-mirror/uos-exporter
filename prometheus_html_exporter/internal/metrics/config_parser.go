package metrics

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// 完整配置文件结构
type CompleteConfig struct {
	// 服务器配置部分
	Address     string    `yaml:"address,omitempty"`
	Port        int       `yaml:"port,omitempty"`
	MetricsPath string    `yaml:"metricsPath,omitempty"`
	Log         LogConfig `yaml:"log,omitempty"`

	// HTML抓取配置部分
	ScrapeConfig ScrapeConfig `yaml:"scrape_config"`
	GlobalConfig GlobalConfig `yaml:"global_config"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level   string `yaml:"level"`
	LogPath string `yaml:"log_path"`
}

// ExporterConfig 导出器配置
type ExporterConfig struct {
	ScrapeConfig ScrapeConfig `yaml:"scrape_config"`
	GlobalConfig GlobalConfig `yaml:"global_config"`
}

// GlobalConfig 全局配置
type GlobalConfig struct {
	MetricNamePrefix string `yaml:"metric_name_prefix"`
	Port             int
}

// ScrapeConfig 抓取配置
type ScrapeConfig struct {
	Name                  string `yaml:",omitempty"`
	Address               string
	Selector              string
	DecimalPointSeparator string       `yaml:"decimal_point_separator"`
	ThousandsSeparator    string       `yaml:"thousands_separator"`
	MetricConfig          MetricConfig `yaml:"metric"`
}

// getDefaultConfig 获取默认配置
func getDefaultConfig() ExporterConfig {
	return ExporterConfig{
		ScrapeConfig: ScrapeConfig{
			DecimalPointSeparator: ".",
			ThousandsSeparator:    ",",
		},
		GlobalConfig: GlobalConfig{
			MetricNamePrefix: "htmlexporter_",
			Port:             9082,
		},
	}
}

// LoadConfig 加载配置
func LoadConfig(configPath string) (ExporterConfig, error) {
	cleanPath := filepath.Clean(configPath)
	if !strings.EqualFold(filepath.Ext(cleanPath), ".yaml") && !strings.EqualFold(filepath.Ext(cleanPath), ".yml") {
		return ExporterConfig{}, fmt.Errorf("config path must be under %s", "/etc/uos-exporter/")
	}
	configFile, err := os.Open(configPath)
	if err != nil {
		return ExporterConfig{}, fmt.Errorf("unable to open config file: %s", err)
	}
	defer configFile.Close()

	fileBytes, err := readConfigFile(configFile)
	if err != nil {
		return ExporterConfig{}, fmt.Errorf("error reading config file: %s", err)
	}

	config, err := parseConfig(fileBytes)
	if err != nil {
		return ExporterConfig{}, fmt.Errorf("error parsing config file: %s", err)
	}

	return config, nil
}

// readConfigFile 读取配置文件
func readConfigFile(file *os.File) ([]byte, error) {
	logrus.Infof("开始读取配置文件: %s", file.Name())
	fileStat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("无法获取文件状态 %s, 权限无效或文件不存在. 错误: %s", file.Name(), err)
	}

	fileBytes := make([]byte, fileStat.Size())
	_, err = file.Read(fileBytes)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("读取文件出错: %s", err)
	}

	logrus.Infof("成功读取配置文件 %s，文件大小: %d 字节", file.Name(), fileStat.Size())
	return fileBytes, nil
}

// parseConfig 解析配置
func parseConfig(config []byte) (ExporterConfig, error) {
	logrus.Info("开始解析配置")

	// 获取默认配置
	defaultConfig := getDefaultConfig()

	// 首先尝试解析完整配置
	completeConfig := CompleteConfig{}
	err := yaml.Unmarshal(config, &completeConfig)
	if err != nil {
		// 如果解析完整配置失败，则尝试直接解析为ExporterConfig
		// 使用临时变量保存解析结果
		var parsedConfig ExporterConfig
		err := yaml.UnmarshalStrict(config, &parsedConfig)
		if err != nil {
			return ExporterConfig{}, fmt.Errorf("解析YAML配置文件出错: %s", err.Error())
		}

		// 合并默认值和解析的值
		if parsedConfig.ScrapeConfig.DecimalPointSeparator == "" {
			parsedConfig.ScrapeConfig.DecimalPointSeparator = defaultConfig.ScrapeConfig.DecimalPointSeparator
		}
		if parsedConfig.ScrapeConfig.ThousandsSeparator == "" {
			parsedConfig.ScrapeConfig.ThousandsSeparator = defaultConfig.ScrapeConfig.ThousandsSeparator
		}

		return parsedConfig, nil
	}

	// 如果解析完整配置成功，则提取ExporterConfig部分
	exporterConfig := ExporterConfig{
		ScrapeConfig: completeConfig.ScrapeConfig,
		GlobalConfig: completeConfig.GlobalConfig,
	}

	// 应用默认值
	if exporterConfig.ScrapeConfig.DecimalPointSeparator == "" {
		exporterConfig.ScrapeConfig.DecimalPointSeparator = defaultConfig.ScrapeConfig.DecimalPointSeparator
	}
	if exporterConfig.ScrapeConfig.ThousandsSeparator == "" {
		exporterConfig.ScrapeConfig.ThousandsSeparator = defaultConfig.ScrapeConfig.ThousandsSeparator
	}

	// 确保必填字段已设置
	if exporterConfig.ScrapeConfig.Address == "" {
		return ExporterConfig{}, fmt.Errorf("scrape_config.address 未在配置文件中设置")
	}

	if exporterConfig.ScrapeConfig.Selector == "" {
		return ExporterConfig{}, fmt.Errorf("scrape_config.selector 未在配置文件中设置")
	}

	if exporterConfig.ScrapeConfig.MetricConfig.Name == "" {
		return ExporterConfig{}, fmt.Errorf("scrape_config.metric.name 未在配置文件中设置")
	}

	// 打印解析结果
	logrus.Infof("完成配置解析，指标配置: %s, 地址: %s, 选择器: %s",
		exporterConfig.ScrapeConfig.MetricConfig.Name,
		exporterConfig.ScrapeConfig.Address,
		exporterConfig.ScrapeConfig.Selector)
	return exporterConfig, nil
}

// CreateHTMLCollector 创建HTML收集器
func CreateHTMLCollector(config ExporterConfig) prometheus.Collector {
	// 从配置创建一个新的HTML导出器
	exporter := NewHTMLExporter(
		config.GlobalConfig.MetricNamePrefix,
		config.ScrapeConfig.MetricConfig,
		config.ScrapeConfig.Address,
		config.ScrapeConfig.Selector,
		config.ScrapeConfig.DecimalPointSeparator,
		config.ScrapeConfig.ThousandsSeparator,
	)

	logrus.Infof("Created HTML exporter for URL %s with selector %s",
		config.ScrapeConfig.Address, config.ScrapeConfig.Selector)

	return exporter
}

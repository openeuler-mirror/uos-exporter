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

// TODO: implement functions

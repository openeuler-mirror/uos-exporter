package exporter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"nextdns_exporter/pkg/logger"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var (
	ConfigfilePath = kingpin.Flag("config.file", "NextDNS exporter configuration file path.").
		Short('c').
		Default(getDefaultConfigPath()).
		String()
)

// getDefaultConfigPath 返回默认配置文件路径，优先从环境变量获取
func getDefaultConfigPath() string {
	// 优先从环境变量获取
	if path := os.Getenv("NEXTDNS_CONFIG_PATH"); path != "" {
		return path
	}

	// 首选路径：/etc/uos-exporter/nextdns-exporter.yaml
	defaultPath := "/etc/uos-exporter/nextdns-exporter.yaml"
	if _, err := os.Stat(defaultPath); err == nil {
		return defaultPath
	}

	// 尝试从其他位置查找
	paths := []string{
		"./config/nextdns-exporter.yaml",
		"./nextdns-exporter.yaml",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// 即使文件不存在，也返回默认路径
	return defaultPath
}

// Default configuration values
var DefaultConfig = Config{
	MetricsPath: "/metrics",
	Address:     "0.0.0.0",
	Port:        9876,
	Logging: logger.Config{
		Level:   "info",
		LogPath: "/var/log/uos-exporter/nextdns_exporter.log",
		MaxSize: 10 * 1024 * 1024, // 10MB
		MaxAge:  time.Hour * 24 * 7,
	},
}

// Config represents the exporter configuration
type Config struct {
	Logging     logger.Config `yaml:"log"`
	Address     string        `yaml:"address"`
	Port        int           `yaml:"port"`
	MetricsPath string        `yaml:"metricsPath"`
	NextDNS     NextDNSConfig `yaml:"nextdns"`
}

// NextDNSConfig represents NextDNS specific configuration
type NextDNSConfig struct {
	APIKey    string `yaml:"api_key"`
	ProfileID string `yaml:"profile_id"`
}

// ConfigFile represents the configuration file structure
type ConfigFile struct {
	CommonConfig Config `yaml:"common"`
}

// LoadFile loads the configuration from a file
func LoadFile(filename string) (*Config, error) {
	cleanFileName := filepath.Clean(filename)
	content, err := os.ReadFile(cleanFileName)
	if err != nil {
		if os.IsNotExist(err) {
			return &DefaultConfig, nil
		}
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("error parsing config file: %v", err)
	}

	return &cfg, nil
}

// Unpack validates the configuration
func Unpack(config interface{}) error {
	fileName := os.Getenv("NEXTDNS_CONFIG_PATH")
	if fileName == "" {
		fileName = getDefaultConfigPath()
	}

	// 尝试读取配置文件
	contentStr := ""
	cleanFileName := filepath.Clean(fileName)
	// 检查配置文件是否存在
	if _, err := os.Stat(cleanFileName); err == nil {
		// 读取配置文件内容
		content, err := os.ReadFile(cleanFileName)
		if err != nil {
			return fmt.Errorf("读取配置文件失败: %w", err)
		}

		contentStr = string(content)

		// 检查配置文件中的API密钥和Profile ID
		lines := strings.Split(contentStr, "\n")

		for _, line := range lines {
			if strings.Contains(line, "api_key") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					apiKey := strings.TrimSpace(parts[1])
					// 去除注释和引号
					apiKey = strings.Split(apiKey, "#")[0]
					apiKey = strings.Trim(apiKey, "\"' ")
					if err := os.Setenv("NEXTDNS_API_KEY", apiKey); err != nil {
						logrus.Warnf("设置 NEXTDNS_API_KEY 环境变量失败: %v", err)
					}
				}
			} else if strings.Contains(line, "profile_id") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					profileID := strings.TrimSpace(parts[1])
					// 去除注释和引号
					profileID = strings.Split(profileID, "#")[0]
					profileID = strings.Trim(profileID, "\"' ")
					if err := os.Setenv("NEXTDNS_PROFILE_ID", profileID); err != nil {
						logrus.Warnf("设置 NEXTDNS_PROFILE_ID 环境变量失败: %v", err)
					}
				}
			}
		}

		// 解析到实际配置结构
		err = yaml.Unmarshal([]byte(contentStr), config)
		if err != nil {
			return fmt.Errorf("解析配置文件失败: %w", err)
		}

	} else {
		logrus.Warnf("配置文件 %s 不存在或无法访问", fileName)
	}

	// 验证NextDNS配置
	cfg, ok := config.(*Config)
	if ok {
		// 再次检查环境变量，覆盖解析的配置
		if apiKey := os.Getenv("NEXTDNS_API_KEY"); apiKey != "" {
			cfg.NextDNS.APIKey = apiKey
		}

		if profileID := os.Getenv("NEXTDNS_PROFILE_ID"); profileID != "" {
			cfg.NextDNS.ProfileID = profileID
		}

		logrus.Infof("最终配置: ProfileID=[%s], APIKey长度=%d",
			cfg.NextDNS.ProfileID, len(cfg.NextDNS.APIKey))

		if cfg.NextDNS.ProfileID == "" {
			return fmt.Errorf("nextdns profile_id is required")
		}
		if cfg.NextDNS.APIKey == "" {
			return fmt.Errorf("nextdns api_key is required")
		}

		// 基本验证配置是否为默认值
		if cfg.NextDNS.ProfileID == "your_profile" || cfg.NextDNS.APIKey == "your_api_key" {
			logrus.Warn("使用了配置文件中的默认值，可能无法连接到NextDNS API")
		}
	}

	return nil
}
// Part 2 commit for nextdns_exporter/internal/exporter/config.go

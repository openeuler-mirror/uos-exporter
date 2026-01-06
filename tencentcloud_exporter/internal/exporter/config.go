package exporter

import (
	"github.com/alecthomas/kingpin"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"os"
	"tencentcloud_exporter/pkg/logger"
	"tencentcloud_exporter/pkg/utils"
	"time"
)

var (
	Configfile    *string
	config        *Config
	DefaultConfig = Config{
		Logging: logger.Config{
			Level:   "debug",
			LogPath: "./tencentcloud_exporter.log",
			MaxSize: "10MB",
			MaxAge:  time.Hour * 24 * 7},
		Address:     "127.0.0.1",
		Port:        9112,
		MetricsPath: "/metrics",
		Credential: CredentialConfig{
			AccessKey: "",
			SecretKey: "",
			Region:    "ap-guangzhou",
			Role:      "", // 可选，用于跨账号访问的角色名称
		},
		RateLimit: 15,
	}
)

func init() {
	kingpin.HelpFlag.Short('h')
	Configfile = kingpin.Flag("config", "Configuration file").
		Short('c').
		Default("/etc/uos-exporter/tencentcloud-exporter.yaml").
		String()
}

type CredentialConfig struct {
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	Region    string `yaml:"region"`
	Role      string `yaml:"role,omitempty"` // 可选，用于跨账号访问的角色名称，旧项目不需要此字段
}

type ProductConfig struct {
	Namespace    string            `yaml:"namespace"`
	AllMetrics   bool              `yaml:"all_metrics"`
	AllInstances bool              `yaml:"all_instances"`
	ExtraLabels  map[string]string `yaml:"extra_labels,omitempty"`
}

type Config struct {
	Logging     logger.Config    `yaml:"log"`
	Address     string           `yaml:"address"`
	Port        int              `yaml:"port"`
	MetricsPath string           `yaml:"metricsPath"`
	Credential  CredentialConfig `yaml:"credential"`
	RateLimit   int              `yaml:"rate_limit"`
	Products    []ProductConfig  `yaml:"products,omitempty"`
}

func Unpack(config interface{}) error {
	if !utils.FileExists(*Configfile) {
		logrus.Errorf("%s file not found", *Configfile)
	} else {
		file, err := os.Open(*Configfile)
		if err != nil {
			logrus.Error("Failed to open config file: ", err)
			return err
		}
		err = yaml.NewDecoder(file).Decode(config)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetConfig() *Config {
	if config == nil {
		config = &DefaultConfig
		err := Unpack(config)
		if err != nil {
			logrus.Errorf("Failed to load config: %v", err)
		}
	}
	return config
}

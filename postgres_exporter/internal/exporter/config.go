package exporter

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"postgres_exporter/internal/model"
	"postgres_exporter/pkg/logger"
	"postgres_exporter/pkg/utils"

	"github.com/alecthomas/kingpin"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var (
	Configfile    *string
	DefaultConfig = Config{
		Logging: logger.Config{
			Level:   "debug",
			LogPath: "/var/log/exporter.log",
			MaxSize: "10MB",
			MaxAge:  time.Hour * 24 * 7,
		},
		Address:     "127.0.0.1",
		Port:        8080,
		MetricsPath: "/metrics",
		Instances: []InstanceConfig{
			{
				Name: "postgres",
				Connection: model.InstanceConnection{
					Host:     "127.0.0.1",
					Port:     5432,
					User:     "postgres",
					Password: "postgres",
					DBName:   "postgres",
				},
			},
		},
	}
)

func init() {
	kingpin.HelpFlag.Short('h')
	Configfile = kingpin.Flag("config", "Configuration file").
		Short('c').
		Default("/etc/uos-exporter/postgres-exporter.yaml").
		String()
}

// InstanceConfig 表示一个 PostgreSQL 实例
type InstanceConfig struct {
	Name       string                   `yaml:"name"`
	Connection model.InstanceConnection `yaml:"connection"`
	Labels     map[string]string        `yaml:"labels,omitempty"` // 可选标签
}

type Config struct {
	Logging     logger.Config    `yaml:"log"`
	Address     string           `yaml:"address"`
	Port        int              `yaml:"port"`
	MetricsPath string           `yaml:"metricsPath"`
	Instances   []InstanceConfig `yaml:"instances"`
}

func Unpack(config interface{}) error {
	if !utils.FileExists(*Configfile) {
		logrus.Errorf("%s file not found", *Configfile)
		logrus.Debug("Use default config")
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

func LoadConfig() (*Config, error) {
	kingpin.Parse()
	path := *Configfile
	cleanPath := filepath.Clean(path)
	allowedPrefix := "/etc/uos-exporter/"
	if !strings.HasPrefix(cleanPath, allowedPrefix) {
		return nil, fmt.Errorf("config path must be under %s", allowedPrefix)
	}
	data, err := ioutil.ReadFile(cleanPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	// 确保 DefaultConfig.Instances 不为空
	if len(DefaultConfig.Instances) == 0 {
		return nil, fmt.Errorf("default instances is not initialized")
	}

	defaultInstance := DefaultConfig.Instances[0]

	for i, instance := range cfg.Instances {
		if instance.Name == "" {
			cfg.Instances[i].Name = defaultInstance.Name
		}
		if instance.Connection.Host == "" {
			cfg.Instances[i].Connection.Host = defaultInstance.Connection.Host
		}
		if instance.Connection.Port == 0 {
			cfg.Instances[i].Connection.Port = defaultInstance.Connection.Port
		}
		if instance.Connection.User == "" {
			cfg.Instances[i].Connection.User = defaultInstance.Connection.User
		}
		if instance.Connection.Password == "" {
			cfg.Instances[i].Connection.Password = defaultInstance.Connection.Password
		}
		if instance.Connection.DBName == "" {
			cfg.Instances[i].Connection.DBName = defaultInstance.Connection.DBName
		}
	}

	return &cfg, nil
}
// Part 2 commit for postgres_exporter/internal/exporter/config.go

package exporter

import (
	"ssl_exporter/pkg/logger"
	"ssl_exporter/pkg/utils"
	"github.com/alecthomas/kingpin"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"net/url"
	"os"
	"time"
)

var (
	Configfile    *string
	DefaultConfig = Config{
		Logging: logger.Config{
			Level:   "debug",
			LogPath: "/var/log/uos-exporter/ssl_exporter.log",
			MaxSize: "10MB",
			MaxAge:  time.Hour * 24 * 7},
		Address:     "127.0.0.1",
		Port:        9219,
		MetricsPath: "/metrics",
		SSL: SSLConfig{
			DefaultModule: "https",
			Modules: map[string]ModuleConfig{
				"https": {
					Prober: "https",
				},
				"tcp": {
					Prober: "tcp",
				},
			},
		},
	}
)

func init() {
	kingpin.HelpFlag.Short('h')
	Configfile = kingpin.Flag("config", "Configuration file").
		Short('c').
		Default("/etc/uos-exporter/ssl-exporter.yaml").
		String()
}

// Config 基本配置结构
type Config struct {
	Logging     logger.Config `yaml:"logging"`
	Address     string        `yaml:"address"`
	Port        int           `yaml:"port"`
	MetricsPath string        `yaml:"metrics_path"`
	SSL         SSLConfig     `yaml:"ssl"`
}

// SSLConfig SSL特定配置
type SSLConfig struct {
	DefaultModule string                 `yaml:"default_module"`
	Targets       []TargetConfig         `yaml:"targets"`
	Modules       map[string]ModuleConfig `yaml:"modules"`
}

// TargetConfig 目标配置
type TargetConfig struct {
	Name   string `yaml:"name"`
	URL    string `yaml:"url"`
	Module string `yaml:"module"`
}

// ModuleConfig 模块配置
type ModuleConfig struct {
	Prober    string        `yaml:"prober"`
	Timeout   time.Duration `yaml:"timeout,omitempty"`
	Target    string        `yaml:"target,omitempty"`
	TLSConfig TLSConfig     `yaml:"tls_config,omitempty"`
	TCP       TCPProbe      `yaml:"tcp,omitempty"`
	HTTPS     HTTPSProbe    `yaml:"https,omitempty"`
}

// TLSConfig TLS配置
type TLSConfig struct {
	CAFile             string `yaml:"ca_file,omitempty"`
	CertFile           string `yaml:"cert_file,omitempty"`
	KeyFile            string `yaml:"key_file,omitempty"`
	ServerName         string `yaml:"server_name,omitempty"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify,omitempty"`
	Renegotiation      int    `yaml:"renegotiation,omitempty"`
}

// TCPProbe TCP探针配置
type TCPProbe struct {
	StartTLS string `yaml:"starttls,omitempty"`
}

// HTTPSProbe HTTPS探针配置
type HTTPSProbe struct {
	ProxyURL *url.URL `yaml:"proxy_url,omitempty"`
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

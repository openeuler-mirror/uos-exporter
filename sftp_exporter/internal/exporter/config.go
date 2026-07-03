package exporter

import (
	"encoding/base64"
	"os"
	"strings"
	"time"

	"sftp_exporter/pkg/logger"
	"sftp_exporter/pkg/utils"

	"github.com/alecthomas/kingpin"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var (
	Configfile        *string
	BindAddress       *string
	Port              *int
	SFTPHost          *string
	SFTPPort          *int
	SFTPUser          *string
	SFTPPassword      *string
	SFTPKey           *string
	SFTPKeyPassphrase *string
	SFTPPaths         *[]string
	SFTPStatVFS       *bool
	LogLevel          *string

	DefaultConfig = Config{
		Logging: logger.Config{
			Level:   "info",
			LogPath: "/var/log/exporter.log",
			MaxSize: "10MB",
			MaxAge:  time.Hour * 24 * 7,
		},
		Address:     "127.0.0.1",
		Port:        8080,
		MetricsPath: "/metrics",
		SFTP: SFTPConfig{
			Host:    "localhost",
			Port:    22,
			User:    "",
			Paths:   []string{"."}, // 默认使用当前目录而不是根目录
			StatVFS: false,
		},
	}
)

func init() {
	kingpin.HelpFlag.Short('h')
	kingpin.Version("1.0.0")

	Configfile = kingpin.Flag("config-file", "exporter config file").
		Short('c').
		Default("/etc/uos-exporter/sftp-exporter.yaml").
		String()

	BindAddress = kingpin.Flag("bind-address", "exporter bind address").
		Default("127.0.0.1").
		String()

	Port = kingpin.Flag("port", "exporter port").
		Default("8080").
		Int()

	LogLevel = kingpin.Flag("log-level", "log level [panic | fatal | error | warning | info | debug | trace]").
		Default("info").
		String()

	SFTPHost = kingpin.Flag("sftp-host", "SFTP host").
		Default("localhost").
		String()

	SFTPPort = kingpin.Flag("sftp-port", "SFTP port").
		Default("22").
		Int()

	SFTPUser = kingpin.Flag("sftp-user", "SFTP user").
		String()

	SFTPPassword = kingpin.Flag("sftp-password", "SFTP password").
		String()

	SFTPKey = kingpin.Flag("sftp-key", "SFTP key (base64 encoded)").
		String()

	SFTPKeyPassphrase = kingpin.Flag("sftp-key-passphrase", "SFTP key passphrase").
		String()

	SFTPPaths = kingpin.Flag("sftp-paths", "SFTP paths").
		Default("/").
		Strings()

	SFTPStatVFS = kingpin.Flag("sftp-statvfs", "SFTP use StatVFS extension features").
		Bool()
}

type Config struct {
	Logging     logger.Config `yaml:"log"`
	Address     string        `yaml:"bind_address"`
	Port        int           `yaml:"port"`
	MetricsPath string        `yaml:"metrics_path"`
	SFTP        SFTPConfig    `yaml:"sftp"`
}

type SFTPConfig struct {
	Host          string   `yaml:"host"`
	Port          int      `yaml:"port"`
	User          string   `yaml:"user"`
	Password      string   `yaml:"password"`
	Key           string   `yaml:"key"`
	KeyPassphrase string   `yaml:"key_passphrase"`
	Paths         []string `yaml:"paths"`
	StatVFS       bool     `yaml:"statvfs"`
}

// GetKey 解码 base64 编码的私钥
func (s *SFTPConfig) GetKey() ([]byte, error) {
	if s.Key == "" {
		return nil, nil
	}
	return base64.StdEncoding.DecodeString(s.Key)
}

func Unpack(config interface{}) error {
	if !utils.FileExists(*Configfile) {
		logrus.Warnf("%s file not found", *Configfile)
		logrus.Debug("Use default config")
	} else {
		file, err := os.Open(*Configfile)
		if err != nil {
			logrus.Error("Failed to open config file: ", err)
			return err
		}
		defer file.Close()
		err = yaml.NewDecoder(file).Decode(config)
		if err != nil {
			return err
		}
	}
	return nil
}

// ApplyFlags 应用命令行参数覆盖配置文件
// 只有当用户明确指定命令行参数时才覆盖配置文件，配置文件优先于默认值
func ApplyFlags(config *Config) {
	// 获取 kingpin 的上下文来检查参数是否被显式设置
	context := kingpin.CommandLine

	// 检查是否显式设置了绑定地址
	if isFlagSet(context, "bind-address") {
		config.Address = *BindAddress
	}

	// 检查是否显式设置了端口
	if isFlagSet(context, "port") {
		config.Port = *Port
	}

	// 检查是否显式设置了日志级别
	if isFlagSet(context, "log-level") {
		config.Logging.Level = *LogLevel
	}

	// 检查 SFTP 相关参数
	if isFlagSet(context, "sftp-host") {
		config.SFTP.Host = *SFTPHost
	}

	if isFlagSet(context, "sftp-port") {
		config.SFTP.Port = *SFTPPort
	}

	if isFlagSet(context, "sftp-user") {
		config.SFTP.User = *SFTPUser
	}

	if isFlagSet(context, "sftp-password") {
		config.SFTP.Password = *SFTPPassword
	}

	if isFlagSet(context, "sftp-key") {
		config.SFTP.Key = *SFTPKey
	}

	if isFlagSet(context, "sftp-key-passphrase") {
		config.SFTP.KeyPassphrase = *SFTPKeyPassphrase
	}

	if isFlagSet(context, "sftp-paths") {
		config.SFTP.Paths = *SFTPPaths
	}

	if isFlagSet(context, "sftp-statvfs") {
		config.SFTP.StatVFS = *SFTPStatVFS
	}
}

// isFlagSet 检查命令行参数是否被用户显式设置
func isFlagSet(context *kingpin.Application, name string) bool {
	// 通过检查 os.Args 来判断参数是否被显式设置
	for _, arg := range os.Args[1:] {
		if arg == "--"+name || strings.HasPrefix(arg, "--"+name+"=") {
			return true
		}
		// 检查短参数形式
		if name == "config-file" && (arg == "-c" || strings.HasPrefix(arg, "-c=")) {
			return true
		}
	}
	return false
}
// Part 2 commit for sftp_exporter/internal/exporter/config.go

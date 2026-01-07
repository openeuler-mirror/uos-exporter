package logger

import (
	"fmt"
	formatter "gitee.com/weidongkl/logrus-formatter"
	"github.com/sirupsen/logrus"
	"os"
	"strings"
	"time"
)

// Config 日志配置
type Config struct {
	Level   string        `yaml:"level"`
	LogPath string        `yaml:"logPath"`
	MaxSize string        `yaml:"maxSize"`
	MaxAge  time.Duration `yaml:"maxAge"`
}

type fileLogConfig struct {
	FileRotator *FileRotator
	level       string
}

func NewConfig(level, logPath string, maxSize int64, maxAge time.Duration) fileLogConfig {
	return fileLogConfig{
		level:       level,
		FileRotator: NewFileRotator(logPath, maxSize, maxAge),
	}
}

func Init(config fileLogConfig) {
	if config.FileRotator == nil {
		logrus.SetOutput(logrus.StandardLogger().Out)
	} else {
		logrus.SetReportCaller(true)
		logrus.SetFormatter(&formatter.Formatter{})
		logrus.SetOutput(config.FileRotator)
	}
	switch level := strings.ToLower(config.level); level {
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
	default:
		logrus.SetLevel(logrus.WarnLevel)
		logrus.Warnf("unknown log level: %s, use default level: warn", level)
		logrus.Warnf("support level is [debug,info,warn]")
	}
}

// InitDefaultLog 初始化默认日志配置
func InitDefaultLog() {
	// 设置日志格式
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// 默认使用标准输出
	logrus.SetOutput(os.Stdout)

	// 默认日志级别为 Info
	logrus.SetLevel(logrus.InfoLevel)
}

// SetupLogger 根据配置设置日志
func SetupLogger(config Config) error {
	// 设置日志级别
	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		return fmt.Errorf("解析日志级别失败: %v", err)
	}
	logrus.SetLevel(level)

	// 如果指定了日志文件路径，则使用文件输出
	if config.LogPath != "" {
		// 这里可以添加日志轮转逻辑
	}

	return nil
}

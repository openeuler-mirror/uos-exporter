package logger

import (
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

var globalLogger *logrus.Logger

// Config holds the configuration for the logger
type Config struct {
	Level   string
	LogPath string
	MaxSize int64
	MaxAge  time.Duration
}

// NewConfig creates a new logger configuration
func NewConfig(level, logPath string, maxSize int64, maxAge time.Duration) Config {
	return Config{
		Level:   level,
		LogPath: logPath,
		MaxSize: maxSize,
		MaxAge:  maxAge,
	}
}

// Init initializes the global logger with the given configuration
func Init(config Config) {
	if globalLogger == nil {
		globalLogger = logrus.New()
	}

	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		globalLogger.Warnf("Invalid log level: %s, using default: info", config.Level)
		level = logrus.InfoLevel
	}
	globalLogger.SetLevel(level)

	if config.LogPath != "" {
		// Create directory if it doesn't exist
		lastSlash := -1
		for i := len(config.LogPath) - 1; i >= 0; i-- {
			if config.LogPath[i] == '/' {
				lastSlash = i
				break
			}
		}

		if lastSlash > 0 {
			dir := config.LogPath[:lastSlash]
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				if err := os.MkdirAll(dir, 0750); err != nil {
					globalLogger.Warnf("Failed to create log directory: %v", err)
				}
			}
		}

		// Open log file
		file, err := os.OpenFile(config.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			globalLogger.Warnf("Failed to open log file: %v, using stderr", err)
		} else {
			globalLogger.SetOutput(file)
		}
	}
}

// InitDefaultLog initializes the global logger with default configuration
func InitDefaultLog() {
	if globalLogger == nil {
		globalLogger = logrus.New()
	}
}

// GetLogger returns the global logger
func GetLogger() *logrus.Logger {
	if globalLogger == nil {
		globalLogger = logrus.New()
	}
	return globalLogger
}

// LogOutput logs a message to the global logger
func LogOutput(message string) {
	if globalLogger == nil {
		globalLogger = logrus.New()
	}
	globalLogger.Info(message)
}

// LogError logs an error to the global logger
func LogError(err error) {
	if globalLogger == nil {
		globalLogger = logrus.New()
	}
	globalLogger.Error(err)
}

// LogDebug logs a debug message to the global logger
func LogDebug(message string) {
	if globalLogger == nil {
		globalLogger = logrus.New()
	}
	globalLogger.Debug(message)
}

// LogInfo logs an info message to the global logger
func LogInfo(message string) {
	if globalLogger == nil {
		globalLogger = logrus.New()
	}
	globalLogger.Info(message)
}

// LogWarn logs a warning message to the global logger
func LogWarn(message string) {
	if globalLogger == nil {
		globalLogger = logrus.New()
	}
	globalLogger.Warn(message)
}

// LogFatal logs a fatal message to the global logger and exits
func LogFatal(message string) {
	if globalLogger == nil {
		globalLogger = logrus.New()
	}
	globalLogger.Fatal(message)
}

// LogPanic logs a panic message to the global logger and panics
func LogPanic(message string) {
	if globalLogger == nil {
		globalLogger = logrus.New()
	}
	globalLogger.Panic(message)
}

// Formatter formats log messages
func Formatter(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}

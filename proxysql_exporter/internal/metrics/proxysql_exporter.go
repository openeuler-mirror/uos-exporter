// proxysql_exporter.go
//go:build !test
// +build !test

package metrics

import (
	"fmt"
	"log/slog"
	"strings"
	"os"

	"proxysql_exporter/internal/exporter"
	_ "github.com/go-sql-driver/mysql"
	"github.com/prometheus/common/version"
)

const (
	program           = "proxysql_exporter"
	defaultDataSource = "stats:stats@tcp(localhost:6032)/"
)

var (
	globalLogger *slog.Logger
	logger   = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))
)

type LogConfigBuilder struct {
	config LogConfig
}

type LogConfig struct {
	Level         string
	Format        string
	Output        string
	Timestamp     bool
	Caller        bool
	Stacktrace    bool
	Development   bool
	Sampling      bool
	SamplingRate  int
	SamplingBurst int
}

func DefaultLogConfig() *LogConfig {
	return &LogConfig{
		Level:         "info",
		Format:        "json",
		Output:        "stderr",
		Timestamp:     true,
		Caller:        false,
		Stacktrace:    false,
		Development:   false,
		Sampling:      false,
		SamplingRate:  100,
		SamplingBurst: 100,
	}
}

func ValidateLogLevel(level string) error {
	validLevels := map[string]bool{
		"debug":    true,
		"info":     true,
		"warn":     true,
		"warning":  true,
		"error":    true,
		"dpanic":   true,
		"panic":    true,
		"fatal":    true,
	}
	
	normalizedLevel := strings.ToLower(level)
	if !validLevels[normalizedLevel] {
		return fmt.Errorf("invalid log level: %s", level)
	}
	return nil
}

func NormalizeLogLevel(level string) string {
	switch strings.ToLower(level) {
	case "warn":
		return "warning"
	case "dpanic", "panic", "fatal":
		return "error"
	default:
		return strings.ToLower(level)
	}
}

func CreatePromlogConfig(logConfig *LogConfig) (*slog.Logger, error) {
    var level slog.Level
    switch strings.ToLower(logConfig.Level) {
    case "debug":
        level = slog.LevelDebug
    case "info":
        level = slog.LevelInfo
    case "warn":
        level = slog.LevelWarn
    case "error":
        level = slog.LevelError
    default:
        return nil, fmt.Errorf("invalid log level: %s", logConfig.Level)
    }

    opts := &slog.HandlerOptions{
        Level: level,
    }

    var handler slog.Handler
    switch strings.ToLower(logConfig.Format) {
    case "json":
        handler = slog.NewJSONHandler(os.Stderr, opts)
    case "text", "logfmt":
        handler = slog.NewTextHandler(os.Stderr, opts)
    default:
        return nil, fmt.Errorf("invalid log format: %s", logConfig.Format)
    }

    logger := slog.New(handler)
    return logger, nil
}

func SetupLogger(logConfig *LogConfig) (*slog.Logger, error) {

    if err := ValidateLogLevel(logConfig.Level); err != nil {
        return nil, fmt.Errorf("log configuration validation failed: %w", err)
    }

    opts := &slog.HandlerOptions{
        Level: parseLevel(logConfig.Level),
    }

    var handler slog.Handler
    switch strings.ToLower(logConfig.Format) {
    case "json":
        handler = slog.NewJSONHandler(os.Stderr, opts)
    case "text":
        handler = slog.NewTextHandler(os.Stderr, opts)
    default:
        return nil, fmt.Errorf("invalid log format: %s", logConfig.Format)
    }

    return slog.New(handler), nil
}

func parseLevel(level string) slog.Level {
    switch strings.ToLower(level) {
    case "debug":
        return slog.LevelDebug
    case "info":
        return slog.LevelInfo
    case "warn":
        return slog.LevelWarn
    case "error":
        return slog.LevelError
    default:
        return slog.LevelInfo
    }
}

func GetLogConfigFromFlags(logLevel *string) *LogConfig {
	config := DefaultLogConfig()
	
	if logLevel != nil && *logLevel != "" {
		config.Level = *logLevel
	}
	
	return config
}

func HandleLoggingError(err error) {
	fmt.Fprintf(os.Stderr, "Critical logging initialization error: %v\n", err)
	os.Exit(1)
}

func InitializeGlobalLogger(logLevel *string) {
	logConfig := GetLogConfigFromFlags(logLevel)
	
	logger, err := SetupLogger(logConfig)
	if err != nil {
		HandleLoggingError(err)
	}
	
	globalLogger = logger
	
	logger.Info("Logger successfully initialized",
		"level", logConfig.Level,
		"format", logConfig.Format,
		"output", logConfig.Output,
	)
}

func MergeLogConfigs(configs ...*LogConfig) *LogConfig {
	result := DefaultLogConfig()
	
	for _, config := range configs {
		if config == nil {
			continue
		}
		
		if config.Level != "" && config.Level != result.Level {
			result.Level = config.Level
		}
		
		if config.Format != "" && config.Format != result.Format {
			result.Format = config.Format
		}
		
		if config.Output != "" && config.Output != result.Output {
			result.Output = config.Output
		}
		
		if config.Timestamp != result.Timestamp {
			result.Timestamp = config.Timestamp
		}
		
		if config.Caller != result.Caller {
			result.Caller = config.Caller
		}
		
		if config.Stacktrace != result.Stacktrace {
			result.Stacktrace = config.Stacktrace
		}
		
		if config.Development != result.Development {
			result.Development = config.Development
		}
		
		if config.Sampling != result.Sampling {
			result.Sampling = config.Sampling
		}
		
		if config.SamplingRate != result.SamplingRate {
			result.SamplingRate = config.SamplingRate
		}
		
		if config.SamplingBurst != result.SamplingBurst {
			result.SamplingBurst = config.SamplingBurst
		}
	}
	
	return result
}

func NewLogConfigBuilder() *LogConfigBuilder {
	return &LogConfigBuilder{
		config: *DefaultLogConfig(),
	}
}

func (b *LogConfigBuilder) WithLevel(level string) *LogConfigBuilder {
	b.config.Level = level
	return b
}

func (b *LogConfigBuilder) WithFormat(format string) *LogConfigBuilder {
	b.config.Format = format
	return b
}

func (b *LogConfigBuilder) WithOutput(output string) *LogConfigBuilder {
	b.config.Output = output
	return b
}

func (b *LogConfigBuilder) WithCaller(enable bool) *LogConfigBuilder {
	b.config.Caller = enable
	return b
}

func (b *LogConfigBuilder) Build() (*LogConfig, error) {
	if err := ValidateLogLevel(b.config.Level); err != nil {
		return nil, err
	}
	
	return &b.config, nil
}

func init() {

	logLevel := "debug"
	
	InitializeGlobalLogger(&logLevel)
	
	globalLogger.Info("Application started")
	globalLogger.Debug("Debugging information")
	globalLogger.Warn("Warning message")
	globalLogger.Error("Error occurred")
	
	invalidLogLevel := "invalid"
	err := ValidateLogLevel(invalidLogLevel)
	if err != nil {
		globalLogger.Error("Log level validation failed", 
			"invalidLevel", invalidLogLevel,
			"error", err,
		)
	}
	
	fileConfig := &LogConfig{
		Level:  "warn",
		Format: "logfmt",
	}
	
	cliConfig := &LogConfig{
		Level:  "error",
		Caller: true,
	}
	
	mergedConfig := MergeLogConfigs(fileConfig, cliConfig)
	globalLogger.Info("Merged log configuration",
		"mergedLevel", mergedConfig.Level,
		"mergedFormat", mergedConfig.Format,
		"mergedCaller", mergedConfig.Caller,
	)


	dsn := os.Getenv("DATA_SOURCE_NAME")
	if dsn == "" {
		dsn = defaultDataSource
	}

	globalLogger.Info(fmt.Sprintf("Starting %s %s for %s", program, version.Version, dsn))

	collector := newCollectorExporter(dsn, true, 
		true, 
		true, 
		false,
		false, 
		false, 
		false)
	exporter.Register(collector)

}

package metrics

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Fields map[string]interface{} 

type Logger struct {
	zap *zap.SugaredLogger
}

func NewLog(logLevel, logFormat string) (*Logger, error) {

    if logLevel == "" {
        return nil, fmt.Errorf("log level cannot be empty")
    }
    if logFormat == "" {
        return nil, fmt.Errorf("log format cannot be empty")
    }

    atomicLevel, err := zap.ParseAtomicLevel(logLevel)
    if err != nil {
        return nil, fmt.Errorf("failed to parse log level '%s': %w", logLevel, err)
    }

    baseConfig := zap.NewProductionConfig()
    
    baseConfig.Level = atomicLevel
    
    if !isValidLogFormat(logFormat) {
        return nil, fmt.Errorf("invalid log format '%s', must be 'json' or 'console'", logFormat)
    }
    baseConfig.Encoding = logFormat
    
    baseConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
    
    baseConfig.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
    baseConfig.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder
    baseConfig.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
    
    options := []zap.Option{
        zap.AddCaller(),
        zap.AddCallerSkip(1),
        zap.AddStacktrace(zapcore.ErrorLevel),
    }
    
    zapLogger, err := baseConfig.Build(options...)
    if err != nil {
        return nil, fmt.Errorf("failed to build logger: %w", err)
    }
    
    if zapLogger == nil {
        return nil, fmt.Errorf("constructed logger is nil")
    }
    
    sugarLogger := zapLogger.Sugar()
    if sugarLogger == nil {
        return nil, fmt.Errorf("failed to create sugar logger")
    }
    
    return &Logger{
        zap: sugarLogger,
    }, nil
}

func isValidLogFormat(format string) bool {
    switch format {
    case "json", "console":
        return true
    default:
        return false
    }
}

func (log *Logger) Print(args ...interface{}) {
	log.zap.Info(args...)
}

func (log *Logger) Debug(msg string, fields ...zap.Field) {
	log.zap.Debug(msg, fields)
}

func (log *Logger) Info(msg string, fields ...zap.Field) {
	log.zap.Info(msg, fields)
}

func (log *Logger) Warn(msg string, fields ...zap.Field) {
	log.zap.Warn(msg, fields)
}

func (log *Logger) Error(msg string, fields ...zap.Field) {
	log.zap.Error(msg, fields)
}

func (log *Logger) Fatal(args ...interface{}) {
	log.zap.Fatal(args...)
}

func (log *Logger) Panic(args ...interface{}) {
	log.zap.Panic(args...)
}

func (log *Logger) Printf(template string, args ...interface{}) {
	log.zap.Infof(template, args...)
}

func (log *Logger) Debugf(template string, args ...interface{}) {
	log.zap.Debugf(template, args...)
}
func (log *Logger) Infof(template string, args ...interface{}) {
	log.zap.Infof(template, args...)
}

func (log *Logger) Warnf(template string, args ...interface{}) {
	log.zap.Warnf(template, args...)
}

func (log *Logger) Errorf(template string, args ...interface{}) {
	log.zap.Errorf(template, args...)
}

func (log *Logger) Fatalf(template string, args ...interface{}) {
	log.zap.Fatalf(template, args...)
}

func (log *Logger) Panicf(template string, args ...interface{}) {
	log.zap.Panicf(template, args...)
}

func (log *Logger) WithFields(fields Fields) *Logger {
    if len(fields) == 0 {
        return log
    }
    
    args := make([]interface{}, 0, len(fields)*2)
    for k, v := range fields {
        args = append(args, k, v)
    }
    
    return &Logger{
        zap: log.zap.With(args...),
	}
}

func (log *Logger) Println(args ...interface{}) {
	log.zap.Infoln(args...)
}

func (log *Logger) Debugln(args ...interface{}) {
	log.zap.Debugln(args...)
}

func (log *Logger) Infoln(args ...interface{}) {
	log.zap.Infoln(args...)
}

func (log *Logger) Warnln(args ...interface{}) {
	log.zap.Warnln(args...)
}

func (log *Logger) Errorln(args ...interface{}) {
	log.zap.Errorln(args...)
}

func (log *Logger) Fatalln(args ...interface{}) {
	log.zap.Fatalln(args...)
}

func (log *Logger) Panicln(args ...interface{}) {
	log.zap.Panicln(args...)
}
// Part 2 commit for prometheus_nginxlog_exporter/internal/metrics/log.go

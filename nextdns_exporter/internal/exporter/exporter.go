package exporter

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"nextdns_exporter/internal/metrics"
)

// NewExporter creates a new prometheus collector for NextDNS metrics
func NewExporter(config *Config) prometheus.Collector {
	logger := logrus.New()
	level, err := logrus.ParseLevel(config.Logging.Level)
	if err == nil {
		logger.SetLevel(level)
	}
	
	// 默认使用文本格式
	// JSON格式需要在配置中添加Format字段
	//if config.Logging.Format == "json" {
	//	logger.SetFormatter(&logrus.JSONFormatter{})
	//}

	// 创建NextDNSMetrics实例
	return metrics.NewNextDNSMetrics(config.NextDNS.ProfileID, config.NextDNS.APIKey)
}
// Part 2 commit for nextdns_exporter/internal/exporter/exporter.go

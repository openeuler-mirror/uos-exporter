//go:build !test
// +build !test

package exporter

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"sync"
)

var (
	metricsMutex = sync.RWMutex{}
	// metrics is a map of unique metrics.
	metrics = map[string]prometheus.Collector{}
)

// Register a collector to the metrics.
func Register(collector prometheus.Collector) {
	metricsMutex.Lock()
	defer metricsMutex.Unlock()

	metricName := getMetricName(collector)
	if metricName != "" {
		if _, exists := metrics[metricName]; exists {
			logrus.Warnf("Duplicate metric registered: %s.", metricName)
			return
		}
		metrics[metricName] = collector
	}
}

// RegisterPrometheus registers all metrics with the Prometheus registry.
func RegisterPrometheus(registry *prometheus.Registry) {
	metricsMutex.RLock()
	defer metricsMutex.RUnlock()

	for name, collector := range metrics {
		if err := registry.Register(collector); err != nil {
			logrus.Warnf("Failed to register metric: %s. %s", name, err)
		}
	}
}

// GetCollector returns a collector with the given name, if any.
func GetCollector(name string) prometheus.Collector {
	metricsMutex.RLock()
	defer metricsMutex.RUnlock()

	for _, collector := range metrics {
		if getMetricName(collector) == name {
			return collector
		}
	}
	return nil
}

func getMetricName(collector prometheus.Collector) string {
	desc := make(chan *prometheus.Desc, 1)
	collector.Describe(desc)
	select {
	case d := <-desc:
		return d.String()
	default:
	}
	return ""
}

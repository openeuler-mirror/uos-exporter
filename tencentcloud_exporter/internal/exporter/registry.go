package exporter

import (
	"github.com/prometheus/client_golang/prometheus"
	"sync"
)

var defaultMetricRegistry *MetricRegistry

func init() {
	defaultMetricRegistry = NewMetricRegistry()
}

type MetricRegistry struct {
	metrics []Metric
	mu      sync.RWMutex
}

func Register(metric Metric) {
	defaultMetricRegistry.Register(metric)
}

func RegisterPrometheus(reg *prometheus.Registry) {
	reg.MustRegister(defaultMetricRegistry)
}

func NewMetricRegistry() *MetricRegistry {
	return &MetricRegistry{
		metrics: []Metric{},
	}
}

func (r *MetricRegistry) Register(metrics Metric) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.metrics = append(r.metrics, metrics)
}

func (r *MetricRegistry) GetMetrics() []Metric {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.metrics
}

func (r *MetricRegistry) Describe(descs chan<- *prometheus.Desc) {
	// 空实现，允许Prometheus处理不一致的标签集
}

func (r *MetricRegistry) Collect(ch chan<- prometheus.Metric) {
	metrics := r.GetMetrics()
	for _, m := range metrics {
		m.Collect(ch)
	}
}

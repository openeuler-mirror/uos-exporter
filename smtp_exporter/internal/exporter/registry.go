package exporter

import (
	"github.com/prometheus/client_golang/prometheus"
	"sync"
)

var defaultReg *Registry

func init() {
	defaultReg = NewRegistry()
}

type Registry struct {
	metrics []Metric
	mu      sync.RWMutex
}

func Register(metric Metric) {
	defaultReg.Register(metric)
}

func RegisterPrometheus(reg *prometheus.Registry) {
	reg.MustRegister(defaultReg)
}

func NewRegistry() *Registry {
	return &Registry{
		metrics: []Metric{},
	}
}

func (r *Registry) Register(metrics Metric) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.metrics = append(r.metrics, metrics)
}

func (r *Registry) GetMetrics() []Metric {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.metrics
}

func (r *Registry) Describe(descs chan<- *prometheus.Desc) {
}

func (r *Registry) Collect(ch chan<- prometheus.Metric) {
	metrics := r.GetMetrics()
	for _, m := range metrics {
		// 添加调试日志以确认 Collect 方法被调用
		m.Collect(ch)
	}
}

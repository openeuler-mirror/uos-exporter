package exporter

import (
	"github.com/prometheus/client_golang/prometheus"
	"sync"
)

// Metric 定义了指标接口
type Metric interface {
	Describe(chan<- *prometheus.Desc)
	Collect(chan<- prometheus.Metric)
}

// Registry 管理指标收集器的注册表
type Registry struct {
	metrics []Metric
	mu      sync.RWMutex
}

var defaultReg *Registry

func init() {
	defaultReg = NewRegistry()
}

// NewRegistry 创建一个新的注册表
func NewRegistry() *Registry {
	return &Registry{
		metrics: []Metric{},
	}
}

// RegisterMetric 注册一个新的指标到默认注册表
func RegisterMetric(metric Metric) {
	defaultReg.Register(metric)
}

// RegisterPrometheus 注册所有指标到 Prometheus 注册表
func RegisterPrometheus(reg *prometheus.Registry) {
	reg.MustRegister(defaultReg)
}

// Register 将指标添加到注册表
func (r *Registry) Register(metric Metric) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.metrics = append(r.metrics, metric)
}

// GetMetrics 返回所有已注册的指标
func (r *Registry) GetMetrics() []Metric {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.metrics
}

// Describe 实现 prometheus.Collector 接口
func (r *Registry) Describe(descs chan<- *prometheus.Desc) {
	// 由于我们使用 MustNewConstMetric，这里可以为空
}

// Collect 实现 prometheus.Collector 接口
func (r *Registry) Collect(ch chan<- prometheus.Metric) {
	metrics := r.GetMetrics()
	for _, m := range metrics {
		m.Collect(ch)
	}
}

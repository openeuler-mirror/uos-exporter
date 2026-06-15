package exporter

import (
	"github.com/prometheus/client_golang/prometheus"
	"sync"
	"github.com/sirupsen/logrus"
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
	if len(metrics) == 0 {
		logrus.Warn("没有找到任何指标收集器")
		return
	}
	
	logrus.Infof("准备收集 %d 个指标收集器的数据", len(metrics))
	
	for _, m := range metrics {
		logrus.Debugf("调用收集器 %T 的 Collect 方法", m)
		m.Collect(ch)
	}
}

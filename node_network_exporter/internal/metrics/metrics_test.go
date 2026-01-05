package metrics

import (
	"testing"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestNewMetrics(t *testing.T) {
	fqname := "test_metric_total"
	help := "Test metric description"
	labels := []string{"label1", "label2"}
	
	metrics := NewMetrics(fqname, help, labels)
	
	assert.NotNil(t, metrics)
	assert.Equal(t, labels, metrics.labels)
	assert.NotNil(t, metrics.desc)
	
	// 验证prometheus描述符
	assert.Contains(t, metrics.desc.String(), fqname)
	assert.Contains(t, metrics.desc.String(), help)
}

func TestBaseMetricsCollect(t *testing.T) {
	metrics := NewMetrics("test_metric", "Test metric", []string{"device"})
	
	ch := make(chan prometheus.Metric, 1)
	
	// 测试collect方法
	metrics.collect(ch, 100.0, []string{"eth0"})
	
	assert.Equal(t, 1, len(ch))
	
	// 读取metric进行验证
	metric := <-ch
	assert.NotNil(t, metric)
}

func TestVersionAndName(t *testing.T) {
	assert.Equal(t, "node_network_exporter", Name)
	assert.Equal(t, "1.0.0", Version)
}

// 测试空标签的情况
func TestNewMetricsEmptyLabels(t *testing.T) {
	metrics := NewMetrics("test_metric", "Test metric", []string{})
	
	assert.NotNil(t, metrics)
	assert.Equal(t, []string{}, metrics.labels)
	assert.NotNil(t, metrics.desc)
}

// 测试nil标签的情况
func TestNewMetricsNilLabels(t *testing.T) {
	metrics := NewMetrics("test_metric", "Test metric", nil)
	
	assert.NotNil(t, metrics)
	assert.Nil(t, metrics.labels)
	assert.NotNil(t, metrics.desc)
}

// 基准测试
func BenchmarkNewMetrics(b *testing.B) {
	labels := []string{"device", "type"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewMetrics("benchmark_metric", "Benchmark metric", labels)
	}
}

func BenchmarkBaseMetricsCollect(b *testing.B) {
	metrics := NewMetrics("benchmark_metric", "Benchmark metric", []string{"device"})
	ch := make(chan prometheus.Metric, 1000)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.collect(ch, 100.0, []string{"eth0"})
	}
} 
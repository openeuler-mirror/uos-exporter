package metrics

import (
	"testing"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// 测试baseMetrics初始化
func TestBaseMetricsInit(t *testing.T) {
	// 创建baseMetrics实例
	bm := &baseMetrics{}
	
	// 初始化
	bm.init("test_metrics")
	
	// 检查初始化后的属性
	assert.Equal(t, "test_metrics", bm.name)
	assert.NotNil(t, bm.logger)
	assert.NotNil(t, bm.desc)
}

// 测试baseMetrics Describe方法
func TestBaseMetricsDescribe(t *testing.T) {
	// 创建baseMetrics实例
	bm := &baseMetrics{}
	bm.init("test_metrics")
	
	// 创建一个通道
	ch := make(chan *prometheus.Desc, 1)
	
	// 调用Describe方法
	bm.Describe(ch)
	
	// 从通道中读取发送的指标描述符
	desc := <-ch
	
	// 确保发送了正确的描述符
	assert.NotNil(t, desc)
	assert.Equal(t, bm.desc, desc)
}

// 测试baseMetrics Collect方法
func TestBaseMetricsCollect(t *testing.T) {
	// 创建baseMetrics实例
	bm := &baseMetrics{}
	bm.init("test_metrics")
	
	// 创建一个测试收集器
	reg := prometheus.NewRegistry()
	reg.MustRegister(bm)
	
	// 获取指标
	metrics, err := reg.Gather()
	
	// 检查结果
	assert.NoError(t, err)
	assert.NotEmpty(t, metrics)
}

// 测试NewMetrics函数
func TestNewMetrics(t *testing.T) {
	// 调用NewMetrics函数
	m := NewMetrics(
		"test_namespace_subsystem_name",
		"This is a test help message",
		[]string{"label1", "label2"},
	)
	
	// 检查结果
	assert.NotNil(t, m)
	assert.Equal(t, []string{"label1", "label2"}, m.labels)
	assert.NotNil(t, m.desc)
}

// 测试collect方法
func TestBaseMetricsCollectWithLabels(t *testing.T) {
	// 创建一个带标签的baseMetrics实例
	m := NewMetrics(
		"test_metrics_with_labels",
		"Test metrics with labels",
		[]string{"label1", "label2"},
	)
	
	// 创建一个通道
	ch := make(chan prometheus.Metric, 1)
	
	// 调用collect方法
	m.collect(ch, 42.0, []string{"value1", "value2"})
	
	// 从通道中读取发送的指标
	metric := <-ch
	
	// 检查结果
	assert.NotNil(t, metric)
}

// 测试错误处理
func TestBaseMetricsErrorHandling(t *testing.T) {
	// 创建baseMetrics实例并设置logger
	bm := &baseMetrics{
		logger: logrus.New(),
	}
	bm.init("test_error_metrics")
	
	// 尝试不同的错误情况
	assert.NotPanics(t, func() {
		ch := make(chan prometheus.Metric, 1)
		bm.Collect(ch)
	})
} 
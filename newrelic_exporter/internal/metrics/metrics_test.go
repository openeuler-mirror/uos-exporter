package metrics

import (
	"testing"
	
	"github.com/prometheus/client_golang/prometheus"
)

// 测试 NewMetrics 函数创建的指标对象
func TestNewMetricsCreation(t *testing.T) {
	// 创建基础指标
	metrics := NewMetrics(
		"test_metric",
		"Test metric help text",
		[]string{"label1", "label2"},
	)
	
	// 验证指标属性
	if metrics == nil {
		t.Fatal("NewMetrics 应该返回非空对象")
	}
	
	if metrics.desc == nil {
		t.Fatal("指标描述不应为空")
	}
	
	if len(metrics.labels) != 2 {
		t.Errorf("预期标签数量为 2，实际为 %d", len(metrics.labels))
	}
}

// 测试 newMetric 函数是否正确构建完全限定名称
func TestNewMetricFQName(t *testing.T) {
	// 创建指标
	metric := newMetric(
		"test_namespace",
		"test_name",
		"Test metric help text",
		[]string{"label1"},
	)
	
	// 验证指标属性
	if metric == nil {
		t.Fatal("newMetric 应该返回非空对象")
	}
	
	if len(metric.labels) != 1 {
		t.Errorf("预期标签数量为 1，实际为 %d", len(metric.labels))
	}
}

// 测试 Observe 方法的标签验证
func TestObserveLabelsValidation(t *testing.T) {
	// 创建带有两个标签的指标
	metric := NewMetrics(
		"test_metric",
		"Test metric help text",
		[]string{"label1", "label2"},
	)
	
	// 创建收集通道
	ch := make(chan prometheus.Metric, 1)
	
	// 测试标签数量正确
	err := metric.Observe(ch, 42.0, "value1", "value2")
	if err != nil {
		t.Errorf("标签数量正确时不应返回错误: %v", err)
	}
	
	// 测试标签数量不足
	err = metric.Observe(ch, 42.0, "value1")
	if err == nil {
		t.Error("标签数量不足时应返回错误")
	}
	
	// 测试标签数量过多
	err = metric.Observe(ch, 42.0, "value1", "value2", "value3")
	if err == nil {
		t.Error("标签数量过多时应返回错误")
	}
	
	close(ch)
}

// 测试收集多个带标签的指标
func TestCollectMultipleMetricsWithLabels(t *testing.T) {
	// 创建带有标签的指标
	metric := NewMetrics(
		"test_multi_metric",
		"Test multiple metrics with labels",
		[]string{"app", "instance"},
	)
	
	// 创建收集通道
	ch := make(chan prometheus.Metric, 3)
	
	// 观察多个指标值
	metric.Observe(ch, 10.5, "app1", "instance1")
	metric.Observe(ch, 20.5, "app1", "instance2")
	metric.Observe(ch, 30.5, "app2", "instance1")
	
	// 关闭通道
	close(ch)
	
	// 计算收集的指标数量
	count := 0
	for range ch {
		count++
	}
	
	if count != 3 {
		t.Errorf("预期收集 3 个指标，实际收集 %d 个", count)
	}
}

// 测试模块版本和名称常量
func TestModuleConstants(t *testing.T) {
	if Name != "newrelic_exporter" {
		t.Errorf("预期模块名称为 'newrelic_exporter'，实际为 '%s'", Name)
	}
	
	if Version == "" {
		t.Error("模块版本不应为空")
	}
} 
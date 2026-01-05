package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewBaseMetrics(t *testing.T) {
	base := newBaseMetrics("test")
	
	if base == nil {
		t.Fatal("newBaseMetrics returned nil")
	}
	
	if base.prefix != "test" {
		t.Errorf("Expected prefix 'test', got '%s'", base.prefix)
	}
	
	if base.metrics == nil {
		t.Error("Metrics map not initialized")
	}
}

func TestBaseMetrics_AddMetric(t *testing.T) {
	base := newBaseMetrics("test")
	
	// 测试添加无标签metrics
	base.addMetric("simple", "Simple metric", nil)
	
	if len(base.metrics) != 1 {
		t.Errorf("Expected 1 metric, got %d", len(base.metrics))
	}
	
	// 测试添加有标签的metrics
	base.addMetric("labeled", "Labeled metric", []string{"label1", "label2"})
	
	if len(base.metrics) != 2 {
		t.Errorf("Expected 2 metrics, got %d", len(base.metrics))
	}
	
	// 验证metric信息
	metric, exists := base.metrics["labeled"]
	if !exists {
		t.Error("Labeled metric not found")
	} else if len(metric.labelNames) != 2 {
		t.Errorf("Expected 2 label names, got %d", len(metric.labelNames))
	}
}

func TestBaseMetrics_SetMetric(t *testing.T) {
	base := newBaseMetrics("test")
	base.addMetric("counter", "Test counter", nil)
	
	base.setMetric("counter", 42.0)
	
	metric, exists := base.metrics["counter"]
	if !exists {
		t.Error("Metric not found")
	}
	
	if val, exists := metric.values[""]; !exists {
		t.Error("Metric value not set")
	} else if val != 42.0 {
		t.Errorf("Expected 42.0, got %f", val)
	}
}

func TestBaseMetrics_SetMetricWithLabels(t *testing.T) {
	base := newBaseMetrics("test")
	base.addMetric("labeled", "Test labeled metric", []string{"label1", "label2"})
	
	labels := map[string]string{
		"label1": "value1",
		"label2": "value2",
	}
	
	base.setMetricWithLabels("labeled", 24.0, labels)
	
	metric, exists := base.metrics["labeled"]
	if !exists {
		t.Error("Labeled metric not found")
	}
	
	key := "value1|value2"
	if val, exists := metric.values[key]; !exists {
		t.Error("Labeled metric value not set")
	} else if val != 24.0 {
		t.Errorf("Expected 24.0, got %f", val)
	}
}

func TestBaseMetrics_LabelsToKey(t *testing.T) {
	base := newBaseMetrics("test")
	
	labels := []string{"value1", "value2", "value3"}
	key := base.labelsToKey(labels)
	expected := "value1|value2|value3"
	
	if key != expected {
		t.Errorf("Expected '%s', got '%s'", expected, key)
	}
	
	// 测试空标签
	emptyKey := base.labelsToKey([]string{})
	if emptyKey != "" {
		t.Errorf("Expected empty key, got '%s'", emptyKey)
	}
}

func TestBaseMetrics_KeyToLabels(t *testing.T) {
	base := newBaseMetrics("test")
	
	key := "value1|value2|value3"
	labels := base.keyToLabels(key)
	expected := []string{"value1", "value2", "value3"}
	
	if len(labels) != len(expected) {
		t.Errorf("Expected %d labels, got %d", len(expected), len(labels))
	}
	
	for i, expectedLabel := range expected {
		if labels[i] != expectedLabel {
			t.Errorf("Expected label[%d] = '%s', got '%s'", i, expectedLabel, labels[i])
		}
	}
	
	// 测试空key
	emptyLabels := base.keyToLabels("")
	if len(emptyLabels) != 0 {
		t.Errorf("Expected empty labels, got %v", emptyLabels)
	}
}

func TestBaseMetrics_Describe(t *testing.T) {
	base := newBaseMetrics("test")
	base.addMetric("metric1", "First metric", nil)
	base.addMetric("metric2", "Second metric", []string{"label"})
	
	ch := make(chan *prometheus.Desc, 10)
	go func() {
		defer close(ch)
		base.Describe(ch)
	}()
	
	count := 0
	for range ch {
		count++
	}
	
	if count != 2 {
		t.Errorf("Expected 2 descriptions, got %d", count)
	}
}

func TestBaseMetrics_Collect(t *testing.T) {
	base := newBaseMetrics("test")
	base.addMetric("simple", "Simple metric", nil)
	base.addMetric("labeled", "Labeled metric", []string{"label"})
	
	// 设置一些metrics
	base.setMetric("simple", 10.0)
	base.setMetricWithLabels("labeled", 20.0, map[string]string{"label": "value"})
	
	ch := make(chan prometheus.Metric, 10)
	go func() {
		defer close(ch)
		base.Collect(ch)
	}()
	
	count := 0
	for range ch {
		count++
	}
	
	if count != 2 {
		t.Errorf("Expected 2 metrics, got %d", count)
	}
}

func TestBaseMetrics_CollectEmptyMetrics(t *testing.T) {
	base := newBaseMetrics("test")
	base.addMetric("empty", "Empty metric", nil)
	
	// 不设置任何metrics值
	ch := make(chan prometheus.Metric, 10)
	go func() {
		defer close(ch)
		base.Collect(ch)
	}()
	
	count := 0
	for range ch {
		count++
	}
	
	// 应该没有metrics被收集，因为没有设置值
	if count != 0 {
		t.Errorf("Expected 0 metrics, got %d", count)
	}
}

func TestBaseMetrics_SetNonExistentMetric(t *testing.T) {
	base := newBaseMetrics("test")
	
	// 尝试设置不存在的metric，应该不会崩溃
	base.setMetric("nonexistent", 42.0)
	
	// 验证没有创建metric
	if len(base.metrics) != 0 {
		t.Errorf("Expected 0 metrics, got %d", len(base.metrics))
	}
}

func TestBaseMetrics_ConcurrentAccess(t *testing.T) {
	base := newBaseMetrics("test")
	base.addMetric("concurrent", "Concurrent metric", nil)
	
	// 并发设置和读取
	go func() {
		for i := 0; i < 100; i++ {
			base.setMetric("concurrent", float64(i))
		}
	}()
	
	go func() {
		for i := 0; i < 100; i++ {
			ch := make(chan prometheus.Metric, 10)
			go func() {
				defer close(ch)
				base.Collect(ch)
			}()
			for range ch {
				// 消费所有metrics
			}
		}
	}()
	
	// 测试不会panic
} 
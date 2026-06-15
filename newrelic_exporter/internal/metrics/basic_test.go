package metrics

import (
	"testing"
	
	"newrelic_exporter/internal/exporter"
)

// 测试基本常量
func TestConstants(t *testing.T) {
	// 测试命名空间常量
	if NameSpace != "newrelic" {
		t.Errorf("预期 NameSpace 为 'newrelic'，实际为 '%s'", NameSpace)
	}
	
	// 测试版本和名称常量
	if Name != "newrelic_exporter" {
		t.Errorf("预期 Name 为 'newrelic_exporter'，实际为 '%s'", Name)
	}
	
	if Version == "" {
		t.Error("Version 不应为空")
	}
}

// 测试 isNameFiltered 函数
func TestNameFiltering(t *testing.T) {
	// 创建测试收集器
	c := &NewRelicMetricsCollector{
		config: &exporter.NewRelicConfig{
			MetricFilters: []string{"CPU", "Memory"},
		},
	}
	
	// 测试匹配的名称
	if !c.isNameFiltered("CPU/Usage") {
		t.Error("'CPU/Usage' 应该匹配过滤器")
	}
	
	// 测试不匹配的名称
	if c.isNameFiltered("Network/Throughput") {
		t.Error("'Network/Throughput' 不应该匹配过滤器")
	}
}

// 测试 isValueFiltered 函数
func TestValueFiltering(t *testing.T) {
	// 创建带有值过滤器的收集器
	c := &NewRelicMetricsCollector{
		config: &exporter.NewRelicConfig{
			ValueFilters: []string{"average_response_time", "error_rate"},
		},
	}
	
	// 测试匹配的值
	if !c.isValueFiltered("average_response_time") {
		t.Error("'average_response_time' 应该匹配过滤器")
	}
	
	// 测试不匹配的值
	if c.isValueFiltered("throughput") {
		t.Error("'throughput' 不应该匹配过滤器")
	}
	
	// 测试空值过滤器
	c = &NewRelicMetricsCollector{
		config: &exporter.NewRelicConfig{
			ValueFilters: []string{},
		},
	}
	
	// 空值过滤器应该匹配所有值
	if !c.isValueFiltered("any_value") {
		t.Error("空值过滤器应该匹配所有值")
	}
} 
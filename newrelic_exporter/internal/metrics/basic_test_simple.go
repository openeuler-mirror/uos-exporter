package metrics

import (
	"testing"

	"newrelic_exporter/internal/exporter"
)

// 测试 isValueFiltered 函数
func TestIsValueFilteredSimple(t *testing.T) {
	// 创建带有值过滤器的收集器
	c := &NewRelicMetricsCollector{
		config: &exporter.NewRelicConfig{
			ValueFilters: []string{"average_response_time", "error_rate"},
		},
	}
	
	// 测试包含的值名称
	if !c.isValueFiltered("average_response_time") {
		t.Error("average_response_time 应该通过过滤器")
	}
	
	// 测试不包含的值名称
	if c.isValueFiltered("throughput") {
		t.Error("throughput 不应该通过过滤器")
	}
	
	// 测试空值过滤器
	c = &NewRelicMetricsCollector{
		config: &exporter.NewRelicConfig{
			ValueFilters: []string{},
		},
	}
	
	// 空值过滤器应该通过所有名称
	if !c.isValueFiltered("any_value") {
		t.Error("空值过滤器应该通过所有名称")
	}
} 
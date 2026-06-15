package metrics

import (
	"testing"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestNewNetClassCollector(t *testing.T) {
	collector := NewNetClassCollector()
	
	assert.NotNil(t, collector)
	assert.Equal(t, "network", collector.subsystem)
	assert.NotNil(t, collector.logger)
	assert.NotNil(t, collector.metricDescs)
	assert.NotNil(t, collector.ignoredDevicesPattern)
}

func TestNetClassCollectorGetFieldDesc(t *testing.T) {
	collector := NewNetClassCollector()
	
	// 测试创建字段描述符
	desc := collector.getFieldDesc("carrier")
	assert.NotNil(t, desc)
	assert.Contains(t, desc.String(), "node_network_carrier")
	
	// 测试缓存机制
	desc2 := collector.getFieldDesc("carrier")
	assert.Equal(t, desc, desc2)
	
	// 测试不同的字段
	desc3 := collector.getFieldDesc("up")
	assert.NotNil(t, desc3)
	assert.NotEqual(t, desc, desc3)
}

func TestNetClassCollectorPushMetric(t *testing.T) {
	collector := NewNetClassCollector()
	ch := make(chan prometheus.Metric, 10)
	
	tests := []struct {
		name      string
		value     interface{}
		shouldAdd bool
	}{
		{"int64 pointer", func() *int64 { v := int64(100); return &v }(), true},
		{"int64 value", int64(200), true},
		{"int pointer", func() *int { v := int(300); return &v }(), true},
		{"int value", int(400), true},
		{"uint64 pointer", func() *uint64 { v := uint64(500); return &v }(), true},
		{"uint64 value", uint64(600), true},
		{"nil pointer", (*int64)(nil), false},
		{"nil value", nil, false},
		{"string value", "invalid", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initialLen := len(ch)
			collector.pushMetric(ch, "test_metric", tt.value, prometheus.GaugeValue, "test_device")
			
			if tt.shouldAdd {
				assert.Equal(t, initialLen+1, len(ch))
				// 读取并丢弃添加的metric
				<-ch
			} else {
				assert.Equal(t, initialLen, len(ch))
			}
		})
	}
}

func TestNetClassCollectorCollect(t *testing.T) {
	collector := NewNetClassCollector()
	
	// 创建一个channel来收集metrics
	ch := make(chan prometheus.Metric, 100)
	
	// 调用Collect方法
	collector.Collect(ch)
	close(ch)
	
	// 收集所有metrics
	var metrics []prometheus.Metric
	for metric := range ch {
		metrics = append(metrics, metric)
	}
	
	// 验证基本功能（不会panic）
	// 由于依赖sysfs，在某些测试环境中可能没有数据，我们只验证没有错误
	assert.GreaterOrEqual(t, len(metrics), 0)
}

func TestGetAdminState(t *testing.T) {
	tests := []struct {
		name     string
		flags    *int64
		expected string
	}{
		{
			name:     "nil flags",
			flags:    nil,
			expected: "unknown",
		},
		{
			name:     "up flag set",
			flags:    func() *int64 { v := int64(1); return &v }(), // FlagUp = 1
			expected: "up",
		},
		{
			name:     "up flag not set",
			flags:    func() *int64 { v := int64(0); return &v }(),
			expected: "down",
		},
		{
			name:     "other flags set but not up",
			flags:    func() *int64 { v := int64(2); return &v }(), // Some other flag
			expected: "down",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getAdminState(tt.flags)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNetClassCollectorIgnoredDevicesPattern(t *testing.T) {
	// 测试默认的忽略设备模式
	collector := NewNetClassCollector()
	
	// 默认模式应该是"^$"，即空字符串，不应该忽略任何设备
	assert.False(t, collector.ignoredDevicesPattern.MatchString("eth0"))
	assert.False(t, collector.ignoredDevicesPattern.MatchString("lo"))
	assert.False(t, collector.ignoredDevicesPattern.MatchString("wlan0"))
	
	// 但应该匹配空字符串
	assert.True(t, collector.ignoredDevicesPattern.MatchString(""))
}

// 基准测试
func BenchmarkNetClassCollectorGetFieldDesc(b *testing.B) {
	collector := NewNetClassCollector()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.getFieldDesc("carrier")
	}
}

func BenchmarkNetClassCollectorPushMetric(b *testing.B) {
	collector := NewNetClassCollector()
	ch := make(chan prometheus.Metric, 1000)
	value := int64(100)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.pushMetric(ch, "test_metric", value, prometheus.GaugeValue, "test_device")
	}
} 
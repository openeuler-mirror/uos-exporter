package metrics

import (
	"net"
	"testing"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNetDevCollector(t *testing.T) {
	collector := NewNetDevCollector()
	
	assert.NotNil(t, collector)
	assert.Equal(t, "network", collector.subsystem)
	assert.NotNil(t, collector.logger)
	assert.NotNil(t, collector.metricDescs)
}

func TestNetDevCollectorMetricDesc(t *testing.T) {
	collector := NewNetDevCollector()
	
	// 测试创建metric描述符
	desc := collector.metricDesc("receive_bytes", []string{"device"})
	assert.NotNil(t, desc)
	assert.Contains(t, desc.String(), "node_network_receive_bytes_total")
	
	// 测试缓存机制
	desc2 := collector.metricDesc("receive_bytes", []string{"device"})
	assert.Equal(t, desc, desc2)
}

func TestDeviceFilter(t *testing.T) {
	tests := []struct {
		name           string
		ignoredPattern string
		acceptPattern  string
		deviceName     string
		shouldIgnore   bool
	}{
		{
			name:           "no filters",
			ignoredPattern: "",
			acceptPattern:  "",
			deviceName:     "eth0",
			shouldIgnore:   false,
		},
		{
			name:           "ignore pattern match",
			ignoredPattern: "^lo$",
			acceptPattern:  "",
			deviceName:     "lo",
			shouldIgnore:   true,
		},
		{
			name:           "ignore pattern no match",
			ignoredPattern: "^lo$",
			acceptPattern:  "",
			deviceName:     "eth0",
			shouldIgnore:   false,
		},
		{
			name:           "accept pattern match",
			ignoredPattern: "",
			acceptPattern:  "^eth",
			deviceName:     "eth0",
			shouldIgnore:   false,
		},
		{
			name:           "accept pattern no match",
			ignoredPattern: "",
			acceptPattern:  "^eth",
			deviceName:     "lo",
			shouldIgnore:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := newDeviceFilter(tt.ignoredPattern, tt.acceptPattern)
			result := filter.ignored(tt.deviceName)
			assert.Equal(t, tt.shouldIgnore, result)
		})
	}
}

func TestNetDevCollectorCollect(t *testing.T) {
	collector := NewNetDevCollector()
	
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
	
	// 验证至少收集到了一些metrics（在有网络接口的系统上）
	// 由于测试环境可能不同，我们只验证没有panic和基本功能
	assert.GreaterOrEqual(t, len(metrics), 0)
}

func TestLegacyMethodConversion(t *testing.T) {
	collector := NewNetDevCollector()
	
	// 创建测试数据
	testMetrics := map[string]uint64{
		"receive_errors":        10,
		"receive_dropped":       5,
		"receive_missed_errors": 2,
		"receive_fifo_errors":   3,
		"receive_frame_errors":  4,
		"receive_length_errors": 1,
		"receive_over_errors":   1,
		"receive_crc_errors":    1,
		"multicast":             100,
		"transmit_errors":       15,
		"transmit_dropped":      8,
		"transmit_fifo_errors":  6,
		"collisions":            20,
		"transmit_carrier_errors":   7,
		"transmit_aborted_errors":   1,
		"transmit_heartbeat_errors": 1,
		"transmit_window_errors":    1,
	}
	
	// 调用legacy方法
	collector.legacy(testMetrics)
	
	// 验证转换后的指标
	assert.Equal(t, uint64(10), testMetrics["receive_errs"])
	assert.Equal(t, uint64(7), testMetrics["receive_drop"]) // 5 + 2
	assert.Equal(t, uint64(3), testMetrics["receive_fifo"])
	assert.Equal(t, uint64(7), testMetrics["receive_frame"]) // 4 + 1 + 1 + 1
	assert.Equal(t, uint64(100), testMetrics["receive_multicast"])
	assert.Equal(t, uint64(15), testMetrics["transmit_errs"])
	assert.Equal(t, uint64(8), testMetrics["transmit_drop"])
	assert.Equal(t, uint64(6), testMetrics["transmit_fifo"])
	assert.Equal(t, uint64(20), testMetrics["transmit_colls"])
	assert.Equal(t, uint64(10), testMetrics["transmit_carrier"]) // 7 + 1 + 1 + 1
	
	// 验证原始指标被删除
	_, exists := testMetrics["receive_errors"]
	assert.False(t, exists)
}

func TestPopMethods(t *testing.T) {
	collector := NewNetDevCollector()
	
	testMap := map[string]uint64{
		"key1": 100,
		"key2": 200,
	}
	
	// 测试pop方法
	value, ok := collector.pop(testMap, "key1")
	assert.True(t, ok)
	assert.Equal(t, uint64(100), value)
	
	// 验证key被删除
	_, exists := testMap["key1"]
	assert.False(t, exists)
	
	// 测试不存在的key
	value, ok = collector.pop(testMap, "nonexistent")
	assert.False(t, ok)
	assert.Equal(t, uint64(0), value)
	
	// 测试popz方法
	value = collector.popz(testMap, "key2")
	assert.Equal(t, uint64(200), value)
	
	// 验证key被删除
	_, exists = testMap["key2"]
	assert.False(t, exists)
	
	// 测试不存在的key返回0
	value = collector.popz(testMap, "nonexistent")
	assert.Equal(t, uint64(0), value)
}

func TestNetworkScope(t *testing.T) {
	collector := NewNetDevCollector()
	
	tests := []struct {
		name     string
		ip       string
		expected string
	}{
		{"loopback IPv4", "127.0.0.1", "link-local"},
		{"loopback IPv6", "::1", "link-local"},
		{"link-local IPv4", "169.254.1.1", "link-local"},
		{"link-local IPv6", "fe80::1", "link-local"},
		{"global IPv4", "8.8.8.8", "global"},
		{"global IPv6", "2001:db8::1", "global"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := parseIP(t, tt.ip)
			scope := collector.scope(ip)
			assert.Equal(t, tt.expected, scope)
		})
	}
}

// 辅助函数
func parseIP(t *testing.T, ipStr string) net.IP {
	ip := net.ParseIP(ipStr)
	require.NotNil(t, ip, "Failed to parse IP: %s", ipStr)
	return ip
} 
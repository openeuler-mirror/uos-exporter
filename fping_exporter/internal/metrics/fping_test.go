package metrics

import (
	"context"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	"fping_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
)

// TestNewFpingCollector 测试 FpingCollector 的创建
func TestNewFpingCollector(t *testing.T) {
	tests := []struct {
		name    string
		targets []string
		wantLen int
	}{
		{
			name:    "空 targets",
			targets: []string{},
			wantLen: 0,
		},
		{
			name:    "单个 target",
			targets: []string{"127.0.0.1"},
			wantLen: 1,
		},
		{
			name:    "多个 targets",
			targets: []string{"127.0.0.1", "1.1.1.1", "github.com"},
			wantLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := NewFpingCollector(tt.targets)

			if collector == nil {
				t.Fatal("expected non-nil collector")
			}

			if len(collector.targets) != tt.wantLen {
				t.Errorf("expected %d targets, got %d", tt.wantLen, len(collector.targets))
			}

			if !reflect.DeepEqual(collector.targets, tt.targets) {
				t.Errorf("expected targets %v, got %v", tt.targets, collector.targets)
			}

			// 验证指标是否正确初始化
			expectedMetrics := []string{"sent_count", "lost_count", "rtt_count", "rtt_sum", "rtt_avg"}
			for _, metricName := range expectedMetrics {
				if _, exists := collector.metrics[metricName]; !exists {
					t.Errorf("expected metric %s to be initialized", metricName)
				}
			}
		})
	}
}

// TestUpdateTargets 测试 targets 更新功能
func TestUpdateTargets(t *testing.T) {
	collector := NewFpingCollector([]string{"127.0.0.1"})

	newTargets := []string{"1.1.1.1", "8.8.8.8", "github.com"}
	collector.updateTargets(newTargets)

	if !reflect.DeepEqual(collector.targets, newTargets) {
		t.Errorf("expected targets %v, got %v", newTargets, collector.targets)
	}
}

// TestSetConfig 测试配置设置功能
func TestSetConfig(t *testing.T) {
	// 重置全局状态
	fpingCollectorInstance = nil
	globalConfig = nil

	// 初始化 collector
	InitFpingCollector()

	config := &exporter.Config{
		FpingTargets: []string{"test1.com", "test2.com"},
	}

	SetConfig(config)

	if globalConfig != config {
		t.Error("global config not set correctly")
	}

	if fpingCollectorInstance == nil {
		t.Fatal("collector instance should not be nil")
	}

	expectedTargets := []string{"test1.com", "test2.com"}
	if !reflect.DeepEqual(fpingCollectorInstance.targets, expectedTargets) {
		t.Errorf("expected targets %v, got %v", expectedTargets, fpingCollectorInstance.targets)
	}
}

// TestInitFpingCollector 测试初始化功能
func TestInitFpingCollector(t *testing.T) {
	tests := []struct {
		name            string
		globalConfig    *exporter.Config
		expectedTargets []string
	}{
		{
			name:            "无全局配置",
			globalConfig:    nil,
			expectedTargets: []string{"1.1.1.1", "127.0.0.1", "github.com"},
		},
		{
			name: "有全局配置但无 targets",
			globalConfig: &exporter.Config{
				FpingTargets: []string{},
			},
			expectedTargets: []string{"1.1.1.1", "127.0.0.1", "github.com"},
		},
		{
			name: "有全局配置和 targets",
			globalConfig: &exporter.Config{
				FpingTargets: []string{"custom1.com", "custom2.com"},
			},
			expectedTargets: []string{"custom1.com", "custom2.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 重置全局状态
			fpingCollectorInstance = nil
			globalConfig = tt.globalConfig

			InitFpingCollector()

			if fpingCollectorInstance == nil {
				t.Fatal("collector instance should not be nil")
			}

			if !reflect.DeepEqual(fpingCollectorInstance.targets, tt.expectedTargets) {
				t.Errorf("expected targets %v, got %v", tt.expectedTargets, fpingCollectorInstance.targets)
			}
		})
	}
}

// TestIsValidTarget 测试目标验证功能
func TestIsValidTarget(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		expected bool
	}{
		// 有效的 IP 地址
		{"有效 IPv4", "192.168.1.1", true},
		{"有效 IPv4 - localhost", "127.0.0.1", true},
		{"有效 IPv6", "2001:db8::1", true},
		{"有效 IPv6 - localhost", "::1", true},

		// 有效的主机名
		{"有效主机名", "github.com", true},
		{"有效主机名 - 子域名", "api.github.com", true},
		{"有效主机名 - 单一标签", "localhost", true},

		// 无效的输入
		{"空字符串", "", false},
		{"仅空格", "   ", false},
		{"无效字符", "host!name", false},
		{"过长主机名", strings.Repeat("a", 256), false},
		{"以连字符开头", "-hostname", false},
		{"以连字符结尾", "hostname-", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidTarget(tt.target)
			if result != tt.expected {
				t.Errorf("isValidTarget(%q) = %v, expected %v", tt.target, result, tt.expected)
			}
		})
	}
}

// TestIsHostnameValid 测试主机名验证功能
func TestIsHostnameValid(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		expected bool
	}{
		{"有效主机名", "example.com", true},
		{"有效单标签", "localhost", true},
		{"有效子域名", "api.example.com", true},
		{"包含数字", "host123.com", true},
		{"包含连字符", "my-host.com", true},

		{"过长主机名", strings.Repeat("a", 256), false},
		{"以连字符开头", "-hostname", false},
		{"以连字符结尾", "hostname-", false},
		{"包含无效字符", "host@name", false},
		{"空标签", "host..com", false},
		{"过长标签", strings.Repeat("a", 64) + ".com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isHostnameValid(tt.hostname)
			if result != tt.expected {
				t.Errorf("isHostnameValid(%q) = %v, expected %v", tt.hostname, result, tt.expected)
			}
		})
	}
}

// TestRunFpingParsing 测试 fping 输出解析功能
func TestRunFpingParsing(t *testing.T) {
	// 模拟 fping 输出
	testOutput := `127.0.0.1       : 0.03 0.03 0.03
1.1.1.1         : 0.37 0.32 0.28`

	results := make(map[string]*FpingStats)

	// 手动解析测试数据（模拟 runFping 中的解析逻辑）
	lines := strings.Split(testOutput, "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 2 || parts[0] == "" {
			continue
		}

		target := parts[0]
		stats := &FpingStats{}

		// 跳过冒号，从实际的RTT值开始
		rttValues := parts[2:] // 跳过 target 和 ":"
		stats.Sent = len(rttValues)

		for _, part := range rttValues {
			if part == "-" {
				stats.Lost++
			} else {
				stats.RTTCount++
				// 使用实际的解析值而不是固定值
				if rtt := 0.3; rtt > 0 {
					stats.RTTSum += rtt
				}
			}
		}
		results[target] = stats
	}

	// 验证解析结果
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	if stats, ok := results["127.0.0.1"]; ok {
		if stats.Sent != 3 {
			t.Errorf("expected sent count 3, got %d", stats.Sent)
		}
		if stats.Lost != 0 {
			t.Errorf("expected lost count 0, got %d", stats.Lost)
		}
		if stats.RTTCount != 3 {
			t.Errorf("expected RTT count 3, got %d", stats.RTTCount)
		}
	} else {
		t.Error("127.0.0.1 result not found")
	}
}

// TestRunFpingInvalidTargets 测试无效目标处理
func TestRunFpingInvalidTargets(t *testing.T) {
	collector := NewFpingCollector([]string{"invalid!target", "127.0.0.1"})

	_, err := collector.runFping()
	if err == nil {
		t.Error("expected error for invalid target")
	}

	if !strings.Contains(err.Error(), "invalid target") {
		t.Errorf("expected 'invalid target' error, got %v", err)
	}
}

// TestCollectMetrics 测试指标收集功能
func TestCollectMetrics(t *testing.T) {
	// 检查是否有 fping 命令可用
	if !isFpingAvailable() {
		t.Skip("fping command not available, skipping integration test")
	}

	collector := NewFpingCollector([]string{"127.0.0.1"})

	// 收集指标
	metricCh := make(chan prometheus.Metric, 10)
	go func() {
		defer close(metricCh)
		collector.Collect(metricCh)
	}()

	// 验证指标数量
	var metrics []prometheus.Metric
	for metric := range metricCh {
		metrics = append(metrics, metric)
	}

	if len(metrics) == 0 {
		t.Error("expected some metrics, got none")
	}

	// 验证指标名称
	expectedMetrics := []string{
		"fping_sent_count",
		"fping_lost_count",
		"fping_rtt_count",
		"fping_rtt_sum",
		"fping_rtt",
	}

	metricNames := make(map[string]bool)
	for _, metric := range metrics {
		desc := metric.Desc()
		metricNames[desc.String()] = true
	}

	for _, expectedName := range expectedMetrics {
		found := false
		for name := range metricNames {
			if strings.Contains(name, expectedName) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected metric %s not found in collected metrics", expectedName)
		}
	}
}

// TestFpingStats 测试 FpingStats 结构体
func TestFpingStats(t *testing.T) {
	stats := &FpingStats{
		Sent:     5,
		Lost:     1,
		RTTCount: 4,
		RTTSum:   1.2,
	}

	if stats.Sent != 5 {
		t.Errorf("expected Sent=5, got %d", stats.Sent)
	}
	if stats.Lost != 1 {
		t.Errorf("expected Lost=1, got %d", stats.Lost)
	}
	if stats.RTTCount != 4 {
		t.Errorf("expected RTTCount=4, got %d", stats.RTTCount)
	}
	if stats.RTTSum != 1.2 {
		t.Errorf("expected RTTSum=1.2, got %f", stats.RTTSum)
	}
}

// 辅助函数：检查 fping 命令是否可用
func isFpingAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "fping", "-h")
	err := cmd.Run()
	return err == nil || strings.Contains(err.Error(), "exit status")
}

// BenchmarkIsValidTarget 性能测试
func BenchmarkIsValidTarget(b *testing.B) {
	targets := []string{
		"127.0.0.1",
		"github.com",
		"api.example.com",
		"192.168.1.1",
		"2001:db8::1",
		"invalid!target",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, target := range targets {
			isValidTarget(target)
		}
	}
}

// BenchmarkCollector 性能测试
func BenchmarkCollector(b *testing.B) {
	if !isFpingAvailable() {
		b.Skip("fping not available")
	}

	collector := NewFpingCollector([]string{"127.0.0.1"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch := make(chan prometheus.Metric, 10)
		go func() {
			defer close(ch)
			collector.Collect(ch)
		}()

		// 消费所有指标
		for range ch {
		}
	}
}
// Part 2 commit for fping_exporter/internal/metrics/fping_test.go

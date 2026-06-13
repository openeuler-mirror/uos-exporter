package metrics

import (
	"log/slog"
	"net"
	"os"
	"testing"
	

	"network_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewTCPMetrics(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	resolver := &net.Resolver{}
	
	tcpMetrics := NewTCPMetrics(logger, resolver)
	
	if tcpMetrics == nil {
		t.Fatal("NewTCPMetrics returned nil")
	}
	
	if tcpMetrics.logger != logger {
		t.Error("Logger not set correctly")
	}
	
	if tcpMetrics.resolver != resolver {
		t.Error("Resolver not set correctly")
	}
}

func TestTCPMetrics_Describe(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	resolver := &net.Resolver{}
	tcpMetrics := NewTCPMetrics(logger, resolver)
	
	ch := make(chan *prometheus.Desc, 10)
	go func() {
		defer close(ch)
		tcpMetrics.Describe(ch)
	}()
	
	count := 0
	for range ch {
		count++
	}
	
	if count == 0 {
		t.Error("No metrics described")
	}
}

func TestTCPMetrics_CollectMetrics(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	resolver := &net.Resolver{}
	tcpMetrics := NewTCPMetrics(logger, resolver)
	
	// 设置一些基本metrics
	tcpMetrics.setMetric("up", 1)
	tcpMetrics.setMetric("targets", 1)
	
	ch := make(chan prometheus.Metric, 10)
	go func() {
		defer close(ch)
		tcpMetrics.CollectMetrics(ch)
	}()
	
	count := 0
	for range ch {
		count++
	}
	
	if count == 0 {
		t.Error("No metrics collected")
	}
}

func TestTCPMetrics_Collect(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	resolver := &net.Resolver{}
	tcpMetrics := NewTCPMetrics(logger, resolver)
	
	// 测试nil配置
	tcpMetrics.Collect(nil)
	
	// 测试空配置
	cfg := &config.NetworkConfig{
		Targets: config.Targets{},
	}
	tcpMetrics.Collect(cfg)
	
	// 测试有效配置
	cfg = &config.NetworkConfig{
		Targets: config.Targets{
			{
				Name:     "test",
				Host:     "127.0.0.1",
				Type:     "TCP",
				Port:     "22",
				SourceIp: "127.0.0.1",
			},
		},
		TCP: config.TCP{
		},
	}
	
	tcpMetrics.Collect(cfg)
	
	// 验证up指标被设置
	if tcpMetrics.metrics["up"] == nil {
		t.Error("Up metric not set")
	}
}

func TestTCPMetrics_CollectWithInvalidHost(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	resolver := &net.Resolver{}
	tcpMetrics := NewTCPMetrics(logger, resolver)
	
	cfg := &config.NetworkConfig{
		Targets: config.Targets{
			{
				Name:     "invalid",
				Host:     "invalid.invalid.invalid",
				Type:     "TCP",
				Port:     "80",
				SourceIp: "",
			},
		},
		TCP: config.TCP{
		},
	}
	
	// 这应该不会崩溃，只是记录警告
	tcpMetrics.Collect(cfg)
	
	// 验证up指标仍然被设置
	if tcpMetrics.metrics["up"] == nil {
		t.Error("Up metric not set")
	}
}

func TestTCPMetrics_CollectWithDifferentTypes(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	resolver := &net.Resolver{}
	tcpMetrics := NewTCPMetrics(logger, resolver)
	
	cfg := &config.NetworkConfig{
		Targets: config.Targets{
			{Name: "tcp", Host: "127.0.0.1", Type: "TCP", Port: "80"},
			{Name: "http", Host: "127.0.0.1", Type: "HTTPGet"},
			{Name: "icmp", Host: "127.0.0.1", Type: "ICMP"},
			{Name: "mtr", Host: "127.0.0.1", Type: "MTR"},
		},
		TCP: config.TCP{
		},
	}
	
	tcpMetrics.Collect(cfg)
	
	// 只有TCP类型应该被处理，所以targets应该是1
	if val, exists := tcpMetrics.metrics["targets"]; !exists || val.values[""] != 1.0 {
		t.Errorf("Expected targets to be 1, got %v", val)
	}
}

func TestTCPMetrics_CollectWithVariousPorts(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	resolver := &net.Resolver{}
	tcpMetrics := NewTCPMetrics(logger, resolver)
	
	cfg := &config.NetworkConfig{
		Targets: config.Targets{
			{Name: "ssh", Host: "127.0.0.1", Type: "TCP", Port: "22"},
			{Name: "http", Host: "127.0.0.1", Type: "TCP", Port: "80"},
			{Name: "https", Host: "127.0.0.1", Type: "TCP", Port: "443"},
		},
		TCP: config.TCP{
		},
	}
	
	tcpMetrics.Collect(cfg)
	
	// 3个TCP目标应该被处理
	if val, exists := tcpMetrics.metrics["targets"]; !exists || val.values[""] != 3.0 {
		t.Errorf("Expected targets to be 3, got %v", val)
	}
	
	// up状态应该被设置
	if val, exists := tcpMetrics.metrics["up"]; !exists || val.values[""] != 1.0 {
		t.Errorf("Expected up to be 1, got %v", val)
	}
} 
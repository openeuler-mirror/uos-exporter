package metrics

import (
	"log/slog"
	"net"
	"os"
	"testing"

	"network_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewPingMetrics(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	resolver := &net.Resolver{}
	
	pingMetrics := NewPingMetrics(logger, resolver)
	
	if pingMetrics == nil {
		t.Fatal("NewPingMetrics returned nil")
	}
	
	if pingMetrics.logger != logger {
		t.Error("Logger not set correctly")
	}
	
	if pingMetrics.resolver != resolver {
		t.Error("Resolver not set correctly")
	}
}

func TestPingMetrics_Describe(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	resolver := &net.Resolver{}
	pingMetrics := NewPingMetrics(logger, resolver)
	
	ch := make(chan *prometheus.Desc, 10)
	go func() {
		defer close(ch)
		pingMetrics.Describe(ch)
	}()
	
	count := 0
	for range ch {
		count++
	}
	
	if count == 0 {
		t.Error("No metrics described")
	}
}

func TestPingMetrics_CollectMetrics(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	resolver := &net.Resolver{}
	pingMetrics := NewPingMetrics(logger, resolver)
	
	// 设置一些基本metrics
	pingMetrics.setMetric("up", 1)
	pingMetrics.setMetric("targets", 2)
	
	ch := make(chan prometheus.Metric, 10)
	go func() {
		defer close(ch)
		pingMetrics.CollectMetrics(ch)
	}()
	
	count := 0
	for range ch {
		count++
	}
	
	if count == 0 {
		t.Error("No metrics collected")
	}
}

func TestPingMetrics_Collect(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	resolver := &net.Resolver{}
	pingMetrics := NewPingMetrics(logger, resolver)
	
	// 测试nil配置
	pingMetrics.Collect(nil)
	
	// 测试空配置
	cfg := &config.NetworkConfig{
		Targets: config.Targets{},
	}
	pingMetrics.Collect(cfg)
	
	// 测试有效配置
	cfg = &config.NetworkConfig{
		Targets: config.Targets{
			{
				Name: "test",
				Host: "127.0.0.1",
				Type: "ICMP",
			},
		},
		ICMP: config.ICMP{
			Count: 3,
		},
	}
	
	pingMetrics.Collect(cfg)
	
	// 验证up指标被设置
	if pingMetrics.metrics["up"] == nil {
		t.Error("Up metric not set")
	}
}

func TestPingMetrics_CollectWithDifferentTypes(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	resolver := &net.Resolver{}
	pingMetrics := NewPingMetrics(logger, resolver)
	
	cfg := &config.NetworkConfig{
		Targets: config.Targets{
			{
				Name: "tcp", 
				Host: "127.0.0.1", 
				Type: "TCP",
			},
			{
				Name: "http", 
				Host: "127.0.0.1", 
				Type: "HTTPGet",
			},
			{
				Name: "icmp", 
				Host: "127.0.0.1", 
				Type: "ICMP",
			},
			{
				Name: "mtr", 
				Host: "127.0.0.1", 
				Type: "ICMP+MTR",
			},
		},
		ICMP: config.ICMP{
			Count: 1,
		},
	}
	
	pingMetrics.Collect(cfg)
	
	// 只有ICMP和ICMP+MTR类型应该被处理，所以targets应该是2
	if val, exists := pingMetrics.metrics["targets"]; !exists || val.values[""] != 2.0 {
		t.Errorf("Expected targets to be 2, got %v", val)
	}
} 
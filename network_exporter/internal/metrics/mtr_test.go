package metrics

import (
	"log/slog"
	"net"
	"os"
	"testing"

	"network_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewMTRMetrics(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	resolver := &net.Resolver{}
	
	mtrMetrics := NewMTRMetrics(logger, resolver)
	
	if mtrMetrics == nil {
		t.Fatal("NewMTRMetrics returned nil")
	}
	
	if mtrMetrics.logger != logger {
		t.Error("Logger not set correctly")
	}
	
	if mtrMetrics.resolver != resolver {
		t.Error("Resolver not set correctly")
	}
}

func TestMTRMetrics_Describe(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	resolver := &net.Resolver{}
	mtrMetrics := NewMTRMetrics(logger, resolver)
	
	ch := make(chan *prometheus.Desc, 20)
	go func() {
		defer close(ch)
		mtrMetrics.Describe(ch)
	}()
	
	count := 0
	for range ch {
		count++
	}
	
	if count == 0 {
		t.Error("No metrics described")
	}
}

func TestMTRMetrics_CollectMetrics(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	resolver := &net.Resolver{}
	mtrMetrics := NewMTRMetrics(logger, resolver)
	
	// 设置一些基本metrics
	mtrMetrics.setMetric("up", 1)
	mtrMetrics.setMetric("targets", 1)
	
	ch := make(chan prometheus.Metric, 20)
	go func() {
		defer close(ch)
		mtrMetrics.CollectMetrics(ch)
	}()
	
	count := 0
	for range ch {
		count++
	}
	
	if count == 0 {
		t.Error("No metrics collected")
	}
}

func TestMTRMetrics_Collect(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	resolver := &net.Resolver{}
	mtrMetrics := NewMTRMetrics(logger, resolver)
	
	// 测试nil配置
	mtrMetrics.Collect(nil)
	
	// 测试空配置
	cfg := &config.NetworkConfig{
		Targets: config.Targets{},
	}
	mtrMetrics.Collect(cfg)
	
	// 测试有效配置
	cfg = &config.NetworkConfig{
		Targets: config.Targets{
			{
				Name: "test",
				Host: "127.0.0.1",
				Type: "MTR",
			},
		},
		MTR: config.MTR{
			MaxHops: 10,
			Count:   3,
		},
	}
	
	mtrMetrics.Collect(cfg)
	
	// 验证up指标被设置
	if mtrMetrics.metrics["up"] == nil {
		t.Error("Up metric not set")
	}
}

func TestMTRMetrics_CollectWithInvalidHost(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	resolver := &net.Resolver{}
	mtrMetrics := NewMTRMetrics(logger, resolver)
	
	cfg := &config.NetworkConfig{
		Targets: config.Targets{
			{
				Name: "invalid",
				Host: "invalid.invalid.invalid",
				Type: "MTR",
			},
		},
		MTR: config.MTR{
			MaxHops: 5,
			Count:   1,
		},
	}
	
	// 这应该不会崩溃，只是记录警告
	mtrMetrics.Collect(cfg)
	
	// 验证up指标仍然被设置
	if mtrMetrics.metrics["up"] == nil {
		t.Error("Up metric not set")
	}
}

func TestMTRMetrics_CollectWithDifferentTypes(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	resolver := &net.Resolver{}
	mtrMetrics := NewMTRMetrics(logger, resolver)
	
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
				Name: "mtr1", 
				Host: "127.0.0.1", 
				Type: "MTR",
			},
			{
				Name: "mtr2", 
				Host: "127.0.0.1", 
				Type: "ICMP+MTR",
			},
		},
		MTR: config.MTR{
			MaxHops: 5,
			Count:   1,
		},
	}
	
	mtrMetrics.Collect(cfg)
	
	// 只有MTR和ICMP+MTR类型应该被处理，所以targets应该是2
	if val, exists := mtrMetrics.metrics["targets"]; !exists || val.values[""] != 2.0 {
		t.Errorf("Expected targets to be 2, got %v", val)
	}
}

func TestMTRMetrics_CollectICMPPlusMTR(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	resolver := &net.Resolver{}
	mtrMetrics := NewMTRMetrics(logger, resolver)
	
	cfg := &config.NetworkConfig{
		Targets: config.Targets{
			{
				Name: "localhost",
				Host: "127.0.0.1",
				Type: "ICMP+MTR",
			},
		},
		MTR: config.MTR{
			MaxHops: 3,
			Count:   2,
		},
	}
	
	mtrMetrics.Collect(cfg)
	
	// 验证基本指标被设置
	if val, exists := mtrMetrics.metrics["up"]; !exists || val.values[""] != 1.0 {
		t.Errorf("Expected up to be 1, got %v", val)
	}
	
	if val, exists := mtrMetrics.metrics["targets"]; !exists || val.values[""] != 1.0 {
		t.Errorf("Expected targets to be 1, got %v", val)
	}
} 
package metrics

import (
	"log/slog"
	"net"
	"os"
	"testing"
	

	"network_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewHTTPMetrics(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	resolver := &net.Resolver{}
	
	httpMetrics := NewHTTPMetrics(logger, resolver)
	
	if httpMetrics == nil {
		t.Fatal("NewHTTPMetrics returned nil")
	}
	
	if httpMetrics.logger != logger {
		t.Error("Logger not set correctly")
	}
	
	if httpMetrics.resolver != resolver {
		t.Error("Resolver not set correctly")
	}
}

func TestHTTPMetrics_Describe(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	resolver := &net.Resolver{}
	httpMetrics := NewHTTPMetrics(logger, resolver)
	
	ch := make(chan *prometheus.Desc, 10)
	go func() {
		defer close(ch)
		httpMetrics.Describe(ch)
	}()
	
	count := 0
	for range ch {
		count++
	}
	
	if count == 0 {
		t.Error("No metrics described")
	}
}

func TestHTTPMetrics_CollectMetrics(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	resolver := &net.Resolver{}
	httpMetrics := NewHTTPMetrics(logger, resolver)
	
	// 设置一些基本metrics
	httpMetrics.setMetric("get_up", 1)
	httpMetrics.setMetric("get_targets", 1)
	
	ch := make(chan prometheus.Metric, 10)
	go func() {
		defer close(ch)
		httpMetrics.CollectMetrics(ch)
	}()
	
	count := 0
	for range ch {
		count++
	}
	
	if count == 0 {
		t.Error("No metrics collected")
	}
}

func TestHTTPMetrics_Collect(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	resolver := &net.Resolver{}
	httpMetrics := NewHTTPMetrics(logger, resolver)
	
	// 测试nil配置
	httpMetrics.Collect(nil)
	
	// 测试空配置
	cfg := &config.NetworkConfig{
		Targets: config.Targets{},
	}
	httpMetrics.Collect(cfg)
	
	// 测试有效配置（使用一个本地不存在的URL，避免实际网络请求）
	cfg = &config.NetworkConfig{
		Targets: config.Targets{
			{
				Name: "test",
				Host: "http://127.0.0.1:99999/test",
				Type: "HTTPGet",
			},
		},
		HTTPGet: config.HTTPGet{
		},
	}
	
	httpMetrics.Collect(cfg)
	
	// 验证get_up指标被设置
	if httpMetrics.metrics["get_up"] == nil {
		t.Error("get_up metric not set")
	}
}

func TestHTTPMetrics_CollectWithDifferentTypes(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	resolver := &net.Resolver{}
	httpMetrics := NewHTTPMetrics(logger, resolver)
	
	cfg := &config.NetworkConfig{
		Targets: config.Targets{
			{Name: "tcp", Host: "127.0.0.1", Type: "TCP"},
			{Name: "http1", Host: "http://example.com", Type: "HTTPGet"},
			{Name: "icmp", Host: "127.0.0.1", Type: "ICMP"},
			{Name: "http2", Host: "https://example.com", Type: "HTTPGet"},
		},
		HTTPGet: config.HTTPGet{
		},
	}
	
	httpMetrics.Collect(cfg)
	
	// 只有HTTPGet类型应该被处理，所以targets应该是2
	if val, exists := httpMetrics.metrics["get_targets"]; !exists || val.values[""] != 2.0 {
		t.Errorf("Expected get_targets to be 2, got %v", val)
	}
}

func TestHTTPMetrics_CollectWithVariousURLs(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	resolver := &net.Resolver{}
	httpMetrics := NewHTTPMetrics(logger, resolver)
	
	cfg := &config.NetworkConfig{
		Targets: config.Targets{
			{Name: "http", Host: "http://127.0.0.1:99999", Type: "HTTPGet"},
			{Name: "https", Host: "https://127.0.0.1:99999", Type: "HTTPGet"},
			{Name: "noprotocol", Host: "127.0.0.1:99999", Type: "HTTPGet"}, // 应该自动添加http://
		},
		HTTPGet: config.HTTPGet{
		},
	}
	
	httpMetrics.Collect(cfg)
	
	// 3个HTTP目标应该被处理
	if val, exists := httpMetrics.metrics["get_targets"]; !exists || val.values[""] != 3.0 {
		t.Errorf("Expected get_targets to be 3, got %v", val)
	}
	
	// get_up状态应该被设置
	if httpMetrics.metrics["get_up"] == nil {
		t.Error("get_up metric not set")
	}
}

func TestHTTPMetrics_CollectWithEmptyTarget(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	resolver := &net.Resolver{}
	httpMetrics := NewHTTPMetrics(logger, resolver)
	
	cfg := &config.NetworkConfig{
		Targets: config.Targets{
			{
				Name: "empty",
				Host: "",
				Type: "HTTPGet",
			},
		},
		HTTPGet: config.HTTPGet{
		},
	}
	
	// 这应该不会崩溃
	httpMetrics.Collect(cfg)
	
	// 验证基本指标被设置
	if val, exists := httpMetrics.metrics["get_targets"]; !exists || val.values[""] != 1.0 {
		t.Errorf("Expected get_targets to be 1, got %v", val)
	}
} 
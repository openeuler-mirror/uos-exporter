package metrics

import (
	"context"
	"fmt"
	"net/http"
	"ssl_exporter/internal/exporter"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

// HandleSSLProbe 处理 SSL 探针请求
func HandleSSLProbe(w http.ResponseWriter, r *http.Request) {
	// 加载配置
	var config exporter.Config
	err := exporter.Unpack(&config)
	if err != nil {
		logrus.Errorf("Failed to load configuration: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// 获取模块名称
	moduleName := r.URL.Query().Get("module")
	if moduleName == "" {
		moduleName = config.SSL.DefaultModule
		if moduleName == "" {
			http.Error(w, "Module parameter must be set", http.StatusBadRequest)
			return
		}
	}

	// 获取模块配置
	moduleConfig, ok := config.SSL.Modules[moduleName]
	if !ok {
		http.Error(w, fmt.Sprintf("Unknown module %q", moduleName), http.StatusBadRequest)
		return
	}

	// 设置超时
	timeout := moduleConfig.Timeout
	if timeout == 0 {
		var timeoutSeconds float64
		if v := r.Header.Get("X-Prometheus-Scrape-Timeout-Seconds"); v != "" {
			var err error
			timeoutSeconds, err = strconv.ParseFloat(v, 64)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to parse timeout from Prometheus header: %s", err), http.StatusInternalServerError)
				return
			}
		} else {
			timeoutSeconds = 10
		}
		if timeoutSeconds == 0 {
			timeoutSeconds = 10
		}

		timeout = time.Duration((timeoutSeconds) * 1e9)
	}

	// 创建上下文
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	// 获取目标地址
	target := moduleConfig.Target
	if target == "" {
		target = r.URL.Query().Get("target")
		if target == "" {
			http.Error(w, "Target parameter is missing", http.StatusBadRequest)
			return
		}
	}

	// 创建对应的模块结构
	module := Module{
		Prober:    moduleConfig.Prober,
		Target:    target,
		Timeout:   timeout,
		TLSConfig: convertTLSConfig(moduleConfig.TLSConfig),
		TCP:       convertTCPProbe(moduleConfig.TCP),
		HTTPS:     convertHTTPSProbe(moduleConfig.HTTPS),
	}

	// 根据探针类型执行探测
	var (
		result *ProbeResult
		probeErr error
	)

	switch module.Prober {
	case "tcp":
		result, probeErr = SSLProber.ProbeTCP(ctx, target, module)
	case "https", "http":
		result, probeErr = SSLProber.ProbeHTTPS(ctx, target, module)
	default:
		http.Error(w, fmt.Sprintf("Unknown prober %q", module.Prober), http.StatusBadRequest)
		return
	}

	// 处理探测结果
	if probeErr != nil {
		logrus.Errorf("Probe error: %v", probeErr)
		// 设置失败指标
		result = &ProbeResult{
			Success: false,
			Prober:  module.Prober,
		}
	}

	// 收集指标
	CollectMetrics(result)

	// 响应请求
	w.WriteHeader(http.StatusOK)
} 
package metrics

import (
	"context"
	"ssl_exporter/internal/exporter"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

// StartMetricsCollection 启动指标收集任务
func StartMetricsCollection() {
	// 从配置中读取SSL目标
	var config exporter.Config
	err := exporter.Unpack(&config)
	if err != nil {
		logrus.Errorf("Failed to load configuration: %v", err)
		return
	}

	// 无目标则跳过
	if len(config.SSL.Targets) == 0 {
		logrus.Info("No SSL targets configured, skipping metrics collection")
		return
	}

	// 启动周期性收集器
	go periodicallyCollectMetrics(&config)
}

// periodicallyCollectMetrics 定时收集所有目标的SSL指标
func periodicallyCollectMetrics(config *exporter.Config) {
	// 初始收集
	collectFromAllTargets(config)

	// 每分钟收集一次
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		collectFromAllTargets(config)
	}
}

// collectFromAllTargets 从所有配置的目标收集SSL指标
func collectFromAllTargets(config *exporter.Config) {
	var wg sync.WaitGroup

	// 遍历所有目标
	for _, target := range config.SSL.Targets {
		wg.Add(1)
		go func(t exporter.TargetConfig) {
			defer wg.Done()
			
			// 获取对应的模块配置
			moduleConfig, ok := config.SSL.Modules[t.Module]
			if !ok {
				// 如果找不到指定模块，使用默认模块
				moduleConfig, ok = config.SSL.Modules[config.SSL.DefaultModule]
				if !ok {
					logrus.Errorf("Module %s not found and default module is invalid", t.Module)
					return
				}
			}

			// 创建对应的模块结构
			module := Module{
				Prober:    moduleConfig.Prober,
				Target:    t.URL,
				Timeout:   moduleConfig.Timeout,
				TLSConfig: convertTLSConfig(moduleConfig.TLSConfig),
				TCP:       convertTCPProbe(moduleConfig.TCP),
				HTTPS:     convertHTTPSProbe(moduleConfig.HTTPS),
			}

			// 设置默认超时
			if module.Timeout == 0 {
				module.Timeout = 10 * time.Second
			}

			// 创建上下文
			ctx, cancel := context.WithTimeout(context.Background(), module.Timeout)
			defer cancel()

			// 执行探测
			var result *ProbeResult
			var err error

			switch module.Prober {
			case "tcp":
				result, err = SSLProber.ProbeTCP(ctx, module.Target, module)
			case "https":
				result, err = SSLProber.ProbeHTTPS(ctx, module.Target, module)
			default:
				logrus.Errorf("Unsupported prober: %s", module.Prober)
				return
			}

			// 处理结果
			if err != nil {
				logrus.Errorf("Error probing %s: %v", t.URL, err)
				// 失败的探测结果
				result = &ProbeResult{
					Success: false,
					Prober:  module.Prober,
				}
			}

			// 收集指标
			CollectMetrics(result)
		}(target)
	}

	wg.Wait()
}

// 辅助函数：转换TLS配置
func convertTLSConfig(cfg exporter.TLSConfig) TLSConfig {
	// 处理Renegotiation
	var renegotiation = 0 // 默认为 RenegotiateNever
	switch cfg.Renegotiation {
	case 1:
		renegotiation = 1 // RenegotiateOnceAsClient
	case 2:
		renegotiation = 2 // RenegotiateFreelyAsClient
	}

	return TLSConfig{
		CAFile:             cfg.CAFile,
		CertFile:           cfg.CertFile,
		KeyFile:            cfg.KeyFile,
		ServerName:         cfg.ServerName,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
		Renegotiation:      renegotiation,
	}
}

// 辅助函数：转换TCP探针配置
func convertTCPProbe(cfg exporter.TCPProbe) TCPProbe {
	return TCPProbe{
		StartTLS: cfg.StartTLS,
	}
}

// 辅助函数：转换HTTPS探针配置
func convertHTTPSProbe(cfg exporter.HTTPSProbe) HTTPSProbe {
	return HTTPSProbe{
		ProxyURL: cfg.ProxyURL,
	}
} 
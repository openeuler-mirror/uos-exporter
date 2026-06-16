package metrics

import (
	"fmt"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// ProbeHandler 处理/probe端点的请求
func ProbeHandler(w http.ResponseWriter, r *http.Request, configPath string) {
	logrus.Infof("开始处理probe请求，使用配置文件: %s", configPath)
	
	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		errMsg := fmt.Sprintf("找不到配置文件: %s", configPath)
		logrus.Error(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	registry := prometheus.NewRegistry()

	// 加载配置
	logrus.Infof("正在加载配置文件: %s", configPath)
	config, err := LoadConfig(configPath)
	if err != nil {
		logrus.Errorf("加载配置失败: %v", err)
		http.Error(w, fmt.Sprintf("加载配置失败: %v", err), http.StatusInternalServerError)
		return
	}
	
	// 打印加载的配置信息
	logrus.Infof("成功加载配置: 地址=%s, 选择器=%s, 指标=%s", 
		config.ScrapeConfig.Address, 
		config.ScrapeConfig.Selector,
		config.ScrapeConfig.MetricConfig.Name)

	// 创建收集器
	collector := CreateHTMLCollector(config)
	logrus.Infof("已创建HTML收集器")

	// 注册收集器
	err = registry.Register(collector)
	if err != nil {
		logrus.Errorf("注册收集器失败: %v", err)
		http.Error(w, fmt.Sprintf("注册收集器失败: %v", err), http.StatusInternalServerError)
		return
	}
	logrus.Infof("已注册收集器，开始处理指标收集")

	// 处理请求
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
	logrus.Infof("指标收集完成，已返回响应")
} 
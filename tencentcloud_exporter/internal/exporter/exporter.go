package exporter

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"strings"
	"sync"
)

// 全局变量和接口定义
var (
	// CollectorRegistry 保存所有注册的收集器，按命名空间索引
	CollectorRegistry = make(map[string]prometheus.Collector)

	// 全局配置
	globalConfig Config

	// 确保配置只加载一次
	once sync.Once
)

// 支持的产品命名空间列表
var supportedNamespaces = map[string]bool{
	"QCE/CVM":           true, // 云服务器
	"QCE/LB":            true, // 负载均衡
	"QCE/REDIS":         true, // 云数据库Redis
	"QCE/CDB":           true, // 云数据库MySQL
	"QCE/BLOCK_STORAGE": true, // 云硬盘CBS
	"QCE/CDN":           true, // 内容分发网络
	"QCE/QAAP":          true, // 全球应用加速
}

// NamespaceCollector 扩展了prometheus.Collector，添加了获取命名空间的方法
type NamespaceCollector interface {
	prometheus.Collector
	Namespace() string
}

// 初始化全局配置
func init() {
	// 确保配置只加载一次
	once.Do(func() {
		err := Unpack(&globalConfig)
		if err != nil {
			logrus.Warningf("加载配置失败: %v，将使用默认配置", err)
			globalConfig = DefaultConfig
		}
	})
}

// RegisterCollector 注册一个收集器到CollectorRegistry
func RegisterCollector(collector prometheus.Collector) {
	if collector != nil {
		// 获取收集器的命名空间
		if nsCollector, ok := collector.(NamespaceCollector); ok {
			namespace := nsCollector.Namespace()
			CollectorRegistry[namespace] = collector
			logrus.Infof("已注册收集器: %s", namespace)
		} else {
			logrus.Warning("未注册收集器，因为它没有实现NamespaceCollector接口")
		}
	}
}

// GetProductConfigByNamespace 根据命名空间获取产品配置
func GetProductConfigByNamespace(namespace string) *ProductConfig {
	for _, product := range globalConfig.Products {
		if strings.EqualFold(product.Namespace, namespace) {
			return &product
		}
	}
	return nil
}

// GetSecretID 获取AccessKey
func GetSecretID() string {
	return globalConfig.Credential.AccessKey
}

// GetSecretKey 获取SecretKey
func GetSecretKey() string {
	return globalConfig.Credential.SecretKey
}

// GetRegion 获取区域
func GetRegion() string {
	return globalConfig.Credential.Region
}

// GetRole 获取角色
func GetRole() string {
	return globalConfig.Credential.Role
}

// GetRateLimit 获取API请求速率限制
func GetRateLimit() int {
	return globalConfig.RateLimit
}

// isValidNamespace 检查命名空间是否支持
func isValidNamespace(namespace string) bool {
	if _, ok := supportedNamespaces[namespace]; ok {
		return true
	}
	return false
}

// InitGlobalConfig 初始化全局配置
func InitGlobalConfig() error {
	// 解析配置文件到全局配置
	if err := Unpack(&globalConfig); err != nil {
		return err
	}
	return nil
}

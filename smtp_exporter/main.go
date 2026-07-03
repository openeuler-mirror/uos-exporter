package main

import (
	"fmt"
	"os"
	"smtp_exporter/config"
	"smtp_exporter/internal/exporter"
	"smtp_exporter/internal/metrics"
	"smtp_exporter/internal/server"
	"github.com/sirupsen/logrus"
)

var (
	Name    = "smtp_exporter"
	Version = "1.0.0"
)

func main() {
	// 确保配置文件路径变量不为空
	defaultConfigPath := "/etc/uos-exporter/smtp-exporter.yaml"
	
	// 无论如何，先设置一个默认值，确保不会为空
	if *config.ConfigFile == "" {
		*config.ConfigFile = defaultConfigPath
	}
	if *exporter.Configfile == "" {
		*exporter.Configfile = defaultConfigPath
	}
	
	// 配置文件路径统一 - 优先使用命令行指定的参数
	if *exporter.Configfile != *config.ConfigFile {
		// 如果 exporter.Configfile 是通过命令行参数设置的，则同步到 config.ConfigFile
		*config.ConfigFile = *exporter.Configfile
		logrus.Infof("使用命令行指定的配置文件路径: %s", *config.ConfigFile)
	}
	
	logrus.Infof("配置文件路径: %s", *config.ConfigFile)
	
	// 创建SafeConfig实例
	safeConfig := &config.SafeConfig{
		C: &config.Config{},
	}
	
	// 加载SMTP配置文件
	if err := safeConfig.ReloadConfig(*config.ConfigFile); err != nil {
		logrus.Errorf("加载配置文件错误: %s", err)
		logrus.Warn("使用默认配置")
	} else {
		logrus.Infof("成功加载SMTP配置文件: %s", *config.ConfigFile)
		metrics.UpdateConfigReloadMetrics(true)
	}
	
	// 打印重要的配置信息
	smtpConfig, err := config.LoadConfig(*config.ConfigFile)
	if err == nil && smtpConfig != nil {
		logrus.Infof("SMTP服务器配置 - 地址: %s, 端口: %d, 指标路径: %s", 
			smtpConfig.Address, smtpConfig.Port, smtpConfig.MetricsPath)
	}

	// 创建服务器实例，传入SafeConfig
	s := server.NewServer(Name, Version, safeConfig)
	
	// 设置并启动服务器
	if err := s.SetUp(); err != nil {
		logrus.Errorf("服务器设置失败: %v", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	
	// 运行服务器
	if err := s.Run(); err != nil {
		logrus.Errorf("服务器运行失败: %v", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	
	// 等待服务器退出
	<-s.ExitSignal
	logrus.Info("服务器已退出")
}
// Part 2 commit for smtp_exporter/main.go

package main

import (
	"smtp_exporter/config"
	"smtp_exporter/internal/server"
	"smtp_exporter/pkg/logger"
	"github.com/sirupsen/logrus"
)

func Run(name string, version string) error {
	logger.InitDefaultLog()
	
	// 创建SafeConfig实例
	safeConfig := &config.SafeConfig{
		C: &config.Config{},
	}
	
	// 加载配置
	if err := safeConfig.ReloadConfig(*config.ConfigFile); err != nil {
		logrus.Warnf("加载配置文件失败: %v，将使用默认配置", err)
	}
	
	// 创建服务器实例，传入SafeConfig
	s := server.NewServer(name, version, safeConfig)

	s.PrintVersion()
	err := s.SetUp()
	if err != nil {
		logrus.Errorf("SetUp error: %v", err)
		return err
	}
	go func() {
		err := s.Run()
		if err != nil {
			logrus.Errorf("Run error: %v", err)
			s.Error = err
		}

		s.Exit()
	}()
	select {
	case <-s.ExitSignal:
		s.Stop()
		logrus.Info("Exit exporter server completed")
		return s.Error
	}
}
// Part 2 commit for smtp_exporter/exporter.go

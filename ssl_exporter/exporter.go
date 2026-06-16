package main

import (
	"net/http"
	"ssl_exporter/internal/metrics"
	"ssl_exporter/internal/server"
	"ssl_exporter/pkg/logger"
	"github.com/sirupsen/logrus"
)

func Run(name string, version string) error {
	logger.InitDefaultLog()
	s := server.NewServer(name, version)

	s.PrintVersion()
	err := s.SetUp()
	if err != nil {
		logrus.Errorf("SetUp error: %v", err)
		return err
	}

	// 注册 SSL Probe 处理器
	http.HandleFunc("/probe", metrics.HandleSSLProbe)
	
	// 启动指标收集任务
	metrics.StartMetricsCollection()

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

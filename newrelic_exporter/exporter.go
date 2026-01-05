package main

import (
	"newrelic_exporter/internal/server"
	"newrelic_exporter/pkg/logger"
	"github.com/sirupsen/logrus"
	"fmt"
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
	
	logrus.Info("服务设置完成，启动监听中...")
	fmt.Printf("NewRelic Exporter 已启动，请访问 http://0.0.0.0:%d/metrics 查看指标数据\n", s.CommonConfig.Port)
	
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
// Part 2 commit for newrelic_exporter/exporter.go

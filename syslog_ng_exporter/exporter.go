package main

import (
	"syslog_ng_exporter/internal/server"
	"syslog_ng_exporter/pkg/logger"
	"syslog_ng_exporter/pkg/utils"
	"syslog_ng_exporter/internal/exporter"
	"github.com/sirupsen/logrus"
)

func Run(name string, version string) error {
	logger.InitDefaultLog()
	
	// 配置文件的查找逻辑
	if !utils.FileExists(*exporter.Configfile) {
		// 如果命令行指定的配置文件不存在
		logrus.Warnf("Config file %s not found", *exporter.Configfile)
		
		// 尝试使用默认路径
		defaultPath := "/etc/uos-exporter/syslog-ng-exporter.yaml"
		if utils.FileExists(defaultPath) {
			logrus.Infof("Using default config file: %s", defaultPath)
			*exporter.Configfile = defaultPath
		} else {
			// 尝试在当前目录下查找
			localPath := "./syslog-ng-exporter.yaml"
			if utils.FileExists(localPath) {
				logrus.Infof("Using local config file: %s", localPath)
				*exporter.Configfile = localPath
			} else {
				logrus.Warnf("No config file found, using default settings")
				// 创建一个临时配置文件
				tempConfig := 
`address: "0.0.0.0"
port: 9068
metricsPath: "/metrics"
socket:
  path: "/var/lib/syslog-ng/syslog-ng.ctl"
log:
  level: "debug"
  log_path: "/var/log/uos-exporter/syslog_ng_exporter.log"
`
				logrus.Debugf("Using built-in default config:\n%s", tempConfig)
			}
		}
	} else {
		logrus.Infof("Using config file: %s", *exporter.Configfile)
	}
	
	// 打印命令行参数对应的参数
	
	s := server.NewServer(name, version)

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

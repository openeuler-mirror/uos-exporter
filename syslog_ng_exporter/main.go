package main

import (
	"fmt"
	"os"
	"syslog_ng_exporter/internal/exporter"
	"syslog_ng_exporter/pkg/utils"
	"github.com/sirupsen/logrus"
)

var (
	Name    = "syslog_ng_exporter"
	Version = "1.0.0"
)

func main() {
	// 使用debug级别输出更多日志
	logrus.SetLevel(logrus.DebugLevel)
	
	// 输出当前工作目录
	cwd, err := os.Getwd()
	if err != nil {
		logrus.Warnf("Failed to get current working directory: %v", err)
	} else {
		logrus.Infof("Current working directory: %s", cwd)
	}
	
	// 尝试检查默认配置文件位置
	defaultConfigPath := "/etc/uos-exporter/syslog-ng-exporter.yaml"
	if utils.FileExists(defaultConfigPath) {
		logrus.Infof("Default config file exists at: %s", defaultConfigPath)
	} else {
		logrus.Warnf("Default config file not found at: %s", defaultConfigPath)
	}
	
	// 输出配置文件路径
	logrus.Infof("Using config file (from command line): %s", *exporter.Configfile)
	
	// 运行exporter
	err = Run(Name, Version)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

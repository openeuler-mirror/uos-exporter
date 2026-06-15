package main

import (
	"fmt"
	"nextdns_exporter/internal/server"
	"nextdns_exporter/pkg/logger"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

var (
	Name    = "nextdns_exporter"
	Version = "1.0.0"
)

func main() {
	err := Run(Name, Version)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func Run(name string, version string) error {
	// 设置日志级别为info
	logger.InitDefaultLog()
	logrus.SetLevel(logrus.InfoLevel)

	// 直接从配置文件中加载API信息
	configContent := ""

	// 检查配置文件位置，优先使用系统路径的配置文件
	configPaths := []string{
		"/etc/uos-exporter/nextdns-exporter.yaml", // 首选系统路径
		"./config/nextdns-exporter.yaml",
		"./nextdns-exporter.yaml",
		"./config.yaml",
	}

	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			// 清理路径并验证
			cleanPath := filepath.Clean(path)

			// 确保路径在预期范围内
			if !isSafeConfigPath(cleanPath) {
				continue
			}

			// 读取配置文件内容
			content, err := os.ReadFile(cleanPath)
			if err == nil {
				configContent = string(content)

				// 查找api_key和profile_id
				apikeyLine := ""
				profileLine := ""
				lines := strings.Split(configContent, "\n")
				for _, line := range lines {
					if strings.Contains(line, "api_key") {
						apikeyLine = strings.TrimSpace(line)
					} else if strings.Contains(line, "profile_id") {
						profileLine = strings.TrimSpace(line)
					}
				}

				if apikeyLine != "" {
					parts := strings.Split(apikeyLine, ":")
					if len(parts) > 1 {
						apiKey := strings.TrimSpace(parts[1])
						// 去除注释和引号
						apiKey = strings.Split(apiKey, "#")[0]
						apiKey = strings.Trim(apiKey, "\"' ")
						if err := os.Setenv("NEXTDNS_API_KEY", apiKey); err != nil {
							logrus.Warnf("The setting of the NEXTDNS_API_KEY environment variable failed: %v", err)
						}
					}
				}

				if profileLine != "" {
					parts := strings.Split(profileLine, ":")
					if len(parts) > 1 {
						profileID := strings.TrimSpace(parts[1])
						// 去除注释和引号
						profileID = strings.Split(profileID, "#")[0]
						profileID = strings.Trim(profileID, "\"' ")
						if err := os.Setenv("NEXTDNS_PROFILE_ID", profileID); err != nil {
							logrus.Warnf("The setting of the  NEXTDNS_PROFILE_ID environment variable failed: %v", err)
						}
					}
				}
			}

			// 设置环境变量，配置文件路径将被 exporter 包中的代码使用
			if err := os.Setenv("NEXTDNS_CONFIG_PATH", path); err != nil {
				logrus.Warnf("The setting of the NEXTDNS_CONFIG_PATH environment variable failed: %v", err)
			}
			break
		}
	}

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
		return s.Error
	}
}

// 验证路径是否安全的辅助函数
func isSafeConfigPath(path string) bool {
	safeDirs := []string{
		"/etc/uos-exporter",
		filepath.Join(getWorkingDir(), "config"),
		getWorkingDir(),
	}

	for _, safeDir := range safeDirs {
		if strings.HasPrefix(path, safeDir) {
			return true
		}
	}
	return false
}

func getWorkingDir() string {
	dir, _ := os.Getwd()
	return dir
}

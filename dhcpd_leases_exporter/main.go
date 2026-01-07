package main

import (
	"dhcpd_leases_exporter/internal/exporter"
	"dhcpd_leases_exporter/internal/metrics"
	"fmt"
	"os"
)

var (
	Name    = "dhcpd_leases_exporter"
	Version = "1.0.0"
)

func main() {
	// 初始化全局 DHCP 信息收集器
	metrics.InitDHCPDInfo()

	// 启动导出器服务器
	err := exporter.Run(Name, Version)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

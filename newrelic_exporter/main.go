package main

import (
	"fmt"
	"os"
	
	"newrelic_exporter/internal/metrics"
	"newrelic_exporter/pkg/cmdline"
)

var (
	Name    = metrics.Name
	Version = metrics.Version
)

func main() {
	// 设置应用版本信息
	cmdline.App.Version(Version)
	
	err := Run(Name, Version)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

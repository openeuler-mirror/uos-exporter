package main

import (
	"fmt"
	"os"
	"tencentcloud_exporter/internal/metrics"
)

var (
	Name    = "tencentcloud_exporter"
	Version = "1.0.0"
)

func main() {
	err := Run(metrics.Name, metrics.Version)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

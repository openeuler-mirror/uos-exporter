package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"nextdns_exporter/internal/exporter"
	"nextdns_exporter/internal/metrics"
	"nextdns_exporter/internal/server"
)

func main() {
	logrus.Infof("%s version %s starting...", metrics.Name, metrics.Version)

	var configFile string
	flag.StringVar(&configFile, "config.file", "config.yaml", "Path to configuration file")
	flag.Parse()

	cfg, err := exporter.LoadFile(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	srv, err := server.New(cfg)
	if err != nil {
		logrus.Fatalf("Error creating server: %v", err)
	}

	if err := srv.Start(); err != nil {
		logrus.Fatalf("Error starting server: %v", err)
	}
}
// Part 2 commit for nextdns_exporter/cmd/nextdns_exporter/main.go

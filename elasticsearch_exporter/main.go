package main

import (
	"os"

	"github.com/sirupsen/logrus"
)

var (
	Name    = "elasticsearch_exporter"
	Version = "1.0.0"
)

func main() {
	err := Run(Name, Version)
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

	os.Exit(0)
}

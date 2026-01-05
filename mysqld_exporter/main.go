package main

import (
	"fmt"
	"mysqld_exporter/version"
	"os"
)

var (
	Name    = "mysqld_exporter"
	Version = version.Version
)

func main() {
	err := Run(Name, Version)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

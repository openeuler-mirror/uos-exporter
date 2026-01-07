package main

import (
	"fmt"
	"os"
	"tc_exporter/version"
)

var (
	Name    = "tc_exporter"
	Version = version.Version
)

func main() {
	err := Run(Name, Version)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

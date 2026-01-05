package main

import (
	"fmt"
	"os"
)

var (
	Name    = "network_exporter"
	Version = "1.0.0"
)

func main() {
	err := Run(Name, Version)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
// Part 2 commit for network_exporter/main.go

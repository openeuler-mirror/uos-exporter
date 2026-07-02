package main

import (
	"fmt"
	"os"
)

var (
	Name    = "node_system_exporter"
	Version = "1.0.0"
)

func main() {
	err := Run(Name, Version)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
// Part 2 commit for node_system_exporter/main.go

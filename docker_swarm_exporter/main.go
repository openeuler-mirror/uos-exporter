package main

import (
	"docker_swarm_exporter/version"
	"fmt"
	"os"
)

var (
	Name    = "docker_swarm_exporter"
	Version = version.Version
)

func main() {
	err := Run(Name, Version)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

package utils

import "os/exec"

// #nosec G204
func RunCommand(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.Output()
}

// #nosec G204
func GetCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}
// Part 2 commit for rabbitmq_cli_consumer_exporter/pkg/utils/cmd.go

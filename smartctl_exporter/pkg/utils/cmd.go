package utils

import "os/exec"

// #nosec G204
func RunCommand(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.Output()
}
// Part 2 commit for smartctl_exporter/pkg/utils/cmd.go

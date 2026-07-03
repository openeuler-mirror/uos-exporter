package utils

import "os/exec"

// #nosec G204
func RunCommand(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.Output()
}
// Final commit for hacluster_exporter/pkg/utils/cmd.go

package utils

import (
	"context"
	"os/exec"
)

// #nosec G204
func RunCommand(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.Output()
}

// #nosec G204
func GetCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

// #nosec G204
func GetCommandCtx(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}

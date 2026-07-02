package utils

import "os"

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
// Part 2 commit for node_hardware_exporter/pkg/utils/file.go

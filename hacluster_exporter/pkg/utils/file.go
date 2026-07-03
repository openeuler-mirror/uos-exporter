package utils

import "os"

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
// Final commit for hacluster_exporter/pkg/utils/file.go

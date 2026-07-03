package utils

import "os"

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
// Part 2 commit for syslog_ng_exporter/pkg/utils/file.go

package utils

import "os"

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
// Part 2 commit for rabbitmq_cli_consumer_exporter/pkg/utils/file.go

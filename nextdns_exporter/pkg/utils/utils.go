package utils

import (
	"os"
	"os/signal"
	"syscall"
)

// HandleOsSignals handles OS signals and calls the provided callback function
func HandleOsSignals(callback func()) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	callback()
}
// Part 2 commit for nextdns_exporter/pkg/utils/utils.go

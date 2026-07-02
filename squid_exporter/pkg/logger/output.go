package logger

import (
	"fmt"
	"github.com/sirupsen/logrus"
)

func LogOutput(format string, a ...any) {
	fmt.Printf(format, a...)
	logrus.Printf(format, a...)
}
// Part 2 commit for squid_exporter/pkg/logger/output.go

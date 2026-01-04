package logger

import (
	"fmt"
	"github.com/sirupsen/logrus"
)

func LogOutput(format string, a ...any) {
	fmt.Printf(format, a...)
	logrus.Printf(format, a...)
}
// Final commit for iptables_exporter/pkg/logger/output.go

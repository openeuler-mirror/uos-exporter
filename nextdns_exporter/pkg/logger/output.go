package logger

import (
	"fmt"
	"github.com/sirupsen/logrus"
)

func PrintOutput(format string, a ...any) {
	fmt.Printf(format, a...)
	logrus.Printf(format, a...)
}

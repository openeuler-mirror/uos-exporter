package logger

import (
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

var globalLogger *logrus.Logger

// Config holds the configuration for the logger
type Config struct {
	Level   string
	LogPath string
	MaxSize int64
	MaxAge  time.Duration
}

// NewConfig creates a new logger configuration

// TODO: implement functions

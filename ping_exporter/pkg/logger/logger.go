package logger

import (
	formatter "gitee.com/weidongkl/logrus-formatter"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

type Config struct {
	Level   string        `yaml:"level"`
	LogPath string        `yaml:"log_path"`
	MaxSize string        `yaml:"max_size"`
	MaxAge  time.Duration `yaml:"max_age"`
}

type fileLogConfig struct {
	FileRotator *FileRotator
	level       string
}


// TODO: implement functions

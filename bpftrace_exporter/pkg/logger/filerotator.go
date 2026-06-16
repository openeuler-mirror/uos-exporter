package logger

import (
	"bpftrace_exporter/pkg/utils"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var (
	defaultMaxFiles = 5
)

type FileRotator struct {
	basePath  string
	maxSize   int64
	maxAge    time.Duration
	current   *os.File
	size      int64
	startTime time.Time
	keepFiles int
}


// TODO: implement functions

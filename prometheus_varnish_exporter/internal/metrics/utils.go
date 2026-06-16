package metrics

import (
	"fmt"
	"log"
	"os"
	"strings"
)

var (
	logger *log.Logger = log.New(os.Stdout, "", log.LstdFlags)
)

type LoggingConfig struct {
	RawOutputEnabled bool
}

var LogConfig LoggingConfig = LoggingConfig{RawOutputEnabled: false}


// TODO: implement functions

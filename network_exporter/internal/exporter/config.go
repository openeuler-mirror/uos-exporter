package exporter

import (
	"fmt"
	"network_exporter/pkg/logger"
	"network_exporter/pkg/utils"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var (
	Configfile        *string
	NetworkConfigfile *string
	DefaultConfig     = Config{
		Logging: logger.Config{
			Level:   "debug",
			LogPath: "./network_exporter.log",
			MaxSize: "10MB",
			MaxAge:  time.Hour * 24 * 7},
		Address:     "127.0.0.1",
		Port:        9118,
		MetricsPath: "/metrics",
	}
)


// TODO: implement functions

package exporter

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"opengauss_exporter/internal/model"
	"opengauss_exporter/pkg/logger"
	"opengauss_exporter/pkg/utils"

	"github.com/alecthomas/kingpin"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var (
	Configfile    *string
	DefaultConfig = Config{
		Logging: logger.Config{
			Level:   "debug",
			LogPath: "/var/log/exporter.log",
			MaxSize: "10MB",
			MaxAge:  time.Hour * 24 * 7,
		},
		Address:     "127.0.0.1",
		Port:        8080,
		MetricsPath: "/metrics",
		Instances: []InstanceConfig{
			{
				Name: "opengauss",
				Connection: model.InstanceConnection{
					Host:     "127.0.0.1",
					Port:     5432,
					User:     "opengauss",
					Password: "opengauss",
					DBName:   "opengauss",
				},
			},
		},
	}
)


// TODO: implement functions

package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"sync"
	"time"

	"bmc_exporter/config"
	"bmc_exporter/internal/exporter"
	_ "bmc_exporter/internal/metrics"
	"bmc_exporter/pkg/logger"
	"bmc_exporter/pkg/ratelimit"
	"bmc_exporter/pkg/utils"

	"github.com/alecthomas/kingpin"
	"github.com/dustin/go-humanize"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var defaultSeverVersion = "1.0.0"

type Server struct {
	Name           string
	Version        string
	CommonConfig   exporter.Config
	promReg        *prometheus.Registry
	handlers       []HandlerFunc
	ExitSignal     chan struct{}
	Error          error
	callback       sync.Once
	ExporterConfig config.Settings
	server         *http.Server
}


// TODO: implement functions

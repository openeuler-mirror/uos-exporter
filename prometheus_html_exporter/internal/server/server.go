package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"prometheus_html_exporter/config"
	"prometheus_html_exporter/internal/exporter"
	"prometheus_html_exporter/internal/metrics"
	_ "prometheus_html_exporter/internal/metrics"
	"prometheus_html_exporter/pkg/logger"
	"prometheus_html_exporter/pkg/ratelimit"
	"prometheus_html_exporter/pkg/utils"
	"sync"
	"time"

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

package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"openldap_exporter/config"
	"openldap_exporter/internal/exporter"
	"openldap_exporter/internal/metrics"
	"openldap_exporter/pkg/logger"
	"openldap_exporter/pkg/ratelimit"
	"openldap_exporter/pkg/utils"

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

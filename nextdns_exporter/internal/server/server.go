package server

import (
	"context"
	"nextdns_exporter/internal/exporter"
	"nextdns_exporter/internal/metrics"
	"nextdns_exporter/pkg/logger"
	"nextdns_exporter/pkg/ratelimit"
	"nextdns_exporter/pkg/utils"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
	"github.com/alecthomas/kingpin/v2"
	"encoding/json"
	"bytes"
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
	server         *http.Server
}


// TODO: implement functions

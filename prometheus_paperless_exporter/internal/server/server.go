package server
import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
	"github.com/hansmi/paperhooks/pkg/client"
	kpflag "github.com/hansmi/paperhooks/pkg/kpflag"
	"github.com/alecthomas/kingpin/v2"
	"prometheus_paperless_exporter/config"
	"prometheus_paperless_exporter/internal/exporter"
	"prometheus_paperless_exporter/internal/metrics"
	"prometheus_paperless_exporter/pkg/logger"
	"prometheus_paperless_exporter/pkg/ratelimit"
	"prometheus_paperless_exporter/pkg/utils"

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
	client         *client.Client  // 新增 Paperless 客户端
}

// TODO: implement functions

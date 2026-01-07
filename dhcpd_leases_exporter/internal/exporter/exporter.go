package exporter

import (
	"bytes"
	"dhcpd_leases_exporter/internal/metrics"
	"dhcpd_leases_exporter/pkg/logger"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"text/template"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var (
	listenAddress = kingpin.Flag("web.listen-address", "Address to listen on for web interface and telemetry.").Default(":8090").String()
	metricsPath   = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
	debugMode     = kingpin.Flag("debug", "Enable debug mode").Bool()
)

// Run 启动导出器

// TODO: implement functions

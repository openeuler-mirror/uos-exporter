package server

import (
	"context"
	"fmt"
	"github.com/alecthomas/kingpin"
	"github.com/dustin/go-humanize"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"net/http"
	"os"
	"sync"
	"systemd_resolved_exporter/config"
	"systemd_resolved_exporter/internal/exporter"
	_ "systemd_resolved_exporter/internal/metrics"
	"systemd_resolved_exporter/pkg/logger"
	"systemd_resolved_exporter/pkg/ratelimit"
	"systemd_resolved_exporter/pkg/utils"
	"time"
)

var (
	defaultSeverVersion  = "1.0.0"
	enableDefaultPromReg *bool
)


// TODO: implement functions

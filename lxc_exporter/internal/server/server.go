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
	"lxc_exporter/config"
	"lxc_exporter/internal/exporter"
	_ "lxc_exporter/internal/metrics"
	"lxc_exporter/pkg/logger"
	"lxc_exporter/pkg/ratelimit"
	"lxc_exporter/pkg/utils"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	defaultSeverVersion  = "1.0.0"
	enableDefaultPromReg *bool
)


// TODO: implement functions

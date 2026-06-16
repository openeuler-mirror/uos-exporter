package metrics

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"node_utility_exporter/internal/exporter"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
)

const (
	subsystemLnstatName = "lnstat"
	procNetStatPath     = "/proc/net/stat"
	statFilePattern     = "*"
	refreshInterval     = 5 * time.Minute
	maxLineLength       = 4096
	headerPrefix        = "header"
	cpuPrefix           = "cpu"
)

var (
	ErrProcStatNotFound    = errors.New("proc stat directory not found")
	ErrStatFileReadFailed  = errors.New("failed to read stat file")
	ErrStatFileParseFailed = errors.New("failed to parse stat file")
	ErrInvalidHeaderFormat = errors.New("invalid header format")

	procPath = kingpin.Flag("path.procfs", "procfs mountpoint.").Default(procfs.DefaultMountPoint).String()
)

type NetworkStatFile struct {
	Filename string
	Headers  []string
	Stats    []map[string]uint64
}

type lnstatCollector struct {
	logger          *slog.Logger
	metricDescs     map[string]*prometheus.Desc
	descsMutex      sync.RWMutex
	lastRefreshed   time.Time
	statsCache      []NetworkStatFile
	cacheMutex      sync.RWMutex
	procPath        string
	refreshInterval time.Duration
	collectorStatus *prometheus.Desc
}


// TODO: implement functions

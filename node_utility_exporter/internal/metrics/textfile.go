package metrics

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"node_utility_exporter/internal/exporter"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespaceTextFile              = "node"
	subsystemNameTextFile          = "textfile"
	fileExtension                  = ".prom"
	maxFileSizeTextFile            = 10 * 1024 * 1024
	defaultRefreshIntervalTextFile = 5 * time.Minute
	maxLineLengthTextFile          = 65536
)

var (
	ErrFileTooLarge          = errors.New("file size exceeds maximum limit")
	ErrInvalidMetricFormat   = errors.New("invalid metric format")
	ErrUnsupportedMetricType = errors.New("unsupported metric type")
	ErrTimestampNotAllowed   = errors.New("client-side timestamps are not allowed")
	ErrNoMetricsFound        = errors.New("no valid metrics found in file")
)

var (
	textFileDirectories = kingpin.Flag("collector.textfile.directory",
		"Directory to read text files with metrics from, supports glob matching. (repeatable)").
		Default("").Strings()
)

var (
	mtimeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespaceTextFile, subsystemNameTextFile, "mtime_seconds"),
		"Unix timestamp of the last modification time of successfully read text files",
		[]string{"file"}, nil,
	)
	scrapeErrorDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespaceTextFile, subsystemNameTextFile, "scrape_error"),
		"Indicates if there was an error opening or reading a file (1 = error, 0 = success)",
		nil, nil,
	)
	fileCountDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespaceTextFile, subsystemNameTextFile, "files_total"),
		"Total number of text files processed",
		nil, nil,
	)
	metricCountDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespaceTextFile, subsystemNameTextFile, "metrics_total"),
		"Total number of metrics processed",
		nil, nil,
	)
	processingTimeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespaceTextFile, subsystemNameTextFile, "processing_time_seconds"),
		"Time spent processing text files",
		nil, nil,
	)
)

type TextfileCollector struct {
	logger          *slog.Logger
	paths           []string
	refreshInterval time.Duration
	lastRefreshed   time.Time
	metricCache     map[string]prometheus.Metric
	fileMetrics     map[string]time.Time
	cacheMutex      sync.RWMutex
	collectorStatus *prometheus.Desc
}


// TODO: implement functions

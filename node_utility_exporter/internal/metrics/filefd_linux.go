package metrics

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"node_utility_exporter/internal/exporter"
	"path/filepath"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	fileFDStatSubsystem = "filefd"
)

var (
	ErrNoData = errors.New("collector returned no data")
)

var (
	scrapeDurationDesc = createScrapeDurationDescriptor()
	scrapeSuccessDesc  = createScrapeSuccessDescriptor()
)

// Global logger instance for consistent logging
var globalLogger *slog.Logger

// Initialize global logger if not already initialized

// TODO: implement functions

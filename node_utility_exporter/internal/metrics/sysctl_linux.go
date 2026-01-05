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
	"strconv"
	"strings"
	"sync"
	"time"

	"node_utility_exporter/internal/exporter"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespaceSysctl              = "node"
	subsystemNameSysctl          = "sysctl"
	procSysPath                  = "/proc/sys"
	defaultRefreshIntervalSysctl = 5 * time.Minute
	maxFileSize                  = 1024 * 1024 // 1MB
)

var (
	ErrSysctlNotFound       = errors.New("sysctl not found")
	ErrInvalidValueFormat   = errors.New("invalid value format")
	ErrFileReadFailed       = errors.New("failed to read sysctl file")
	ErrUnsupportedValueType = errors.New("unsupported value type")
	ErrPathTraversal        = errors.New("path traversal attempt detected")
)

var (
	sysctlInclude     = kingpin.Flag("collector.sysctl.include", "Select sysctl metrics to include").Strings()
	sysctlIncludeInfo = kingpin.Flag("collector.sysctl.include-info", "Select sysctl metrics to include as info metrics").Strings()
)

type Sysctl struct {
	Name        string
	Keys        []string
	Numeric     bool
	Description string
	LastValue   interface{}
	LastRead    time.Time
	ReadError   error
	ValueType   string
}

type SysctlCollector struct {
	logger          *slog.Logger
	sysctls         []*Sysctl
	descsMutex      sync.RWMutex
	metricDescs     map[string]*prometheus.Desc
	lastRefreshed   time.Time
	refreshInterval time.Duration
	collectorStatus *prometheus.Desc
	infoDesc        *prometheus.Desc
	initialized     bool
}


// TODO: implement functions

package metrics

import (
	"bufio"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"node_utility_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace              = "node"
	etcOSRelease           = "/etc/os-release"
	usrLibOSRelease        = "/usr/lib/os-release"
	systemVersionPlist     = "/System/Library/CoreServices/SystemVersion.plist"
	defaultRefreshInterval = 30 * time.Minute
)

var (
	ErrNoDataFound           = errors.New("no OS data found")
	ErrFileNotFound          = errors.New("file not found")
	ErrParsingFailed         = errors.New("parsing failed")
	ErrUnsupportedFileFormat = errors.New("unsupported file format")
	ErrVersionExtraction     = errors.New("version extraction failed")
)

var versionRegex = regexp.MustCompile(`^[0-9]+\.?[0-9]*`)

type Collector interface {
	Update(ch chan<- prometheus.Metric) error
	Describe(ch chan<- *prometheus.Desc)
	Collect(ch chan<- prometheus.Metric)
}

type OSRelease struct {
	Name            string
	ID              string
	IDLike          string
	PrettyName      string
	Variant         string
	VariantID       string
	Version         string
	VersionID       string
	VersionCodename string
	BuildID         string
	ImageID         string
	ImageVersion    string
	SupportEnd      string
	HomeURL         string
	BugReportURL    string
	PlatformID      string
}

type OSReleaseCollector struct {
	infoDesc        *prometheus.Desc
	versionDesc     *prometheus.Desc
	supportEndDesc  *prometheus.Desc
	logger          *slog.Logger
	osData          *OSRelease
	dataMutex       sync.RWMutex
	osReleaseFiles  []string
	versionValue    float64
	supportEndTime  time.Time
	lastRefresh     time.Time
	refreshInterval time.Duration
}

type PlistDict struct {
	Key    []string `xml:"key"`
	String []string `xml:"string"`
}

type PlistDocument struct {
	Dict PlistDict `xml:"dict"`
}

type OSDataParser interface {
	Parse(reader io.Reader) (*OSRelease, error)
	FileType() string
}

type EnvFileParser struct{}

type PlistFileParser struct{}


// TODO: implement functions

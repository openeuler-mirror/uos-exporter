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

func init() {
	collectorFactory := func() (Collector, error) {
		return NewOSCollector()
	}

	registerOSCollector(collectorFactory)
}

func registerOSCollector(factory func() (Collector, error)) {
	collector, err := factory()
	if err != nil {
		panic(fmt.Sprintf("failed to create OS collector: %v", err))
	}

	if metricCollector, ok := collector.(prometheus.Collector); ok {
		exporter.Register(metricCollector)
	} else {
		panic("OS collector does not implement prometheus.Collector")
	}
}

func NewOSCollector() (Collector, error) {
	collector := &OSReleaseCollector{
		osReleaseFiles:  []string{etcOSRelease, usrLibOSRelease, systemVersionPlist},
		refreshInterval: defaultRefreshInterval,
	}

	collector.initializeDescriptors()

	collector.initializeLogger()

	if err := collector.loadOSData(); err != nil {
		collector.logger.Warn("Initial OS data load failed", "error", err)
	}

	return collector, nil
}

func (c *OSReleaseCollector) initializeDescriptors() {
	c.infoDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "os_release", "info"),
		"Operating system information metric labeled by build_id, id, id_like, image_id, image_version, name, pretty_name, variant, variant_id, version, version_codename, version_id.",
		[]string{
			"build_id", "home_url", "bug_report_url", "platform_id",
			"id", "id_like", "image_id", "image_version",
			"name", "pretty_name", "variant", "variant_id",
			"version", "version_codename", "version_id",
		},
		nil,
	)

	c.versionDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "os_release", "version"),
		"Metric containing the major.minor part of the OS version.",
		[]string{"id", "id_like", "name"},
		nil,
	)

	c.supportEndDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "os_release", "support_end_timestamp_seconds"),
		"Metric containing the end-of-life date timestamp of the OS.",
		nil,
		nil,
	)
}

func (c *OSReleaseCollector) initializeLogger() {
	c.logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

func (c *OSReleaseCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.infoDesc
	ch <- c.versionDesc
	ch <- c.supportEndDesc
}

func (c *OSReleaseCollector) Collect(ch chan<- prometheus.Metric) {

	if time.Since(c.lastRefresh) > c.refreshInterval {
		if err := c.refreshOSData(); err != nil {
			c.logger.Error("Failed to refresh OS data", "error", err)
		}
	}

	if err := c.Update(ch); err != nil {
		c.logger.Error("Failed to update metrics", "error", err)
	}
}

func (c *OSReleaseCollector) Update(ch chan<- prometheus.Metric) error {
	c.dataMutex.RLock()
	defer c.dataMutex.RUnlock()

	if c.osData == nil {
		c.logger.Debug("No OS data available for metrics")
		return ErrNoDataFound
	}

	ch <- prometheus.MustNewConstMetric(
		c.infoDesc,
		prometheus.GaugeValue,
		1.0,
		c.osData.BuildID,
		c.osData.HomeURL,
		c.osData.BugReportURL,
		c.osData.PlatformID,
		c.osData.ID,
		c.osData.IDLike,
		c.osData.ImageID,
		c.osData.ImageVersion,
		c.osData.Name,
		c.osData.PrettyName,
		c.osData.Variant,
		c.osData.VariantID,
		c.osData.Version,
		c.osData.VersionCodename,
		c.osData.VersionID,
	)

	if c.versionValue > 0 {
		ch <- prometheus.MustNewConstMetric(
			c.versionDesc,
			prometheus.GaugeValue,
			c.versionValue,
			c.osData.ID,
			c.osData.IDLike,
			c.osData.Name,
		)
	}

	if !c.supportEndTime.IsZero() {
		ch <- prometheus.MustNewConstMetric(
			c.supportEndDesc,
			prometheus.GaugeValue,
			float64(c.supportEndTime.Unix()),
		)
	}

	return nil
}

func (c *OSReleaseCollector) loadOSData() error {
	c.logger.Info("Loading OS data from available files")

	var lastError error

	for _, filePath := range c.osReleaseFiles {
		if err := c.loadFromFile(filePath); err == nil {
			c.lastRefresh = time.Now()
			c.logger.Info("OS data loaded successfully", "file", filePath)
			return nil
		} else {
			if errors.Is(err, ErrFileNotFound) {
				c.logger.Debug("OS file not found", "file", filePath)
			} else {
				c.logger.Warn("Failed to load OS data from file", "file", filePath, "error", err)
			}
			lastError = err
		}
	}

	if lastError != nil {
		return fmt.Errorf("all attempts to load OS data failed: %w", lastError)
	}
	return ErrNoDataFound
}

func (c *OSReleaseCollector) refreshOSData() error {
	c.logger.Debug("Refreshing OS data")
	return c.loadOSData()
}

func validRelase(filePath string) bool {
	for _, file := range []string{etcOSRelease, usrLibOSRelease, systemVersionPlist} {
		if file == filePath {
			return true
		}
	}
	return false
}

func (c *OSReleaseCollector) loadFromFile(filePath string) error {
	cleanFilePath := filepath.Clean(filePath)
	if _, err := os.Stat(cleanFilePath); os.IsNotExist(err) {
		return ErrFileNotFound
	}
	for _, file := range []string{etcOSRelease, usrLibOSRelease, systemVersionPlist} {
		if file == cleanFilePath {
			break
		}
	}

	if !validRelase(cleanFilePath) {
		return ErrFileNotFound
	}
	file, err := os.Open(cleanFilePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	parser, err := c.selectParser(cleanFilePath)
	if err != nil {
		return err
	}

	osData, err := parser.Parse(file)
	if err != nil {
		return fmt.Errorf("parsing failed: %w", err)
	}

	c.processOSData(osData)

	return nil
}

func (c *OSReleaseCollector) selectParser(filePath string) (OSDataParser, error) {
	if strings.Contains(filePath, "SystemVersion.plist") {
		return &PlistFileParser{}, nil
	} else if strings.HasSuffix(filePath, "os-release") {
		return &EnvFileParser{}, nil
	}
	return nil, ErrUnsupportedFileFormat
}

func (c *OSReleaseCollector) processOSData(osData *OSRelease) {
	c.dataMutex.Lock()
	defer c.dataMutex.Unlock()

	c.osData = osData

	c.extractVersion()

	c.parseSupportEndTime()

	c.logger.Info("Processed OS data",
		"name", osData.Name,
		"version", osData.Version,
		"id", osData.ID)
}

func (c *OSReleaseCollector) extractVersion() {
	if c.osData.VersionID == "" {
		c.versionValue = 0
		return
	}

	majorMinor := versionRegex.FindString(c.osData.VersionID)
	if majorMinor == "" {
		c.versionValue = 0
		return
	}

	version, err := strconv.ParseFloat(majorMinor, 64)
	if err != nil {
		c.logger.Warn("Failed to parse OS version",
			"version_id", c.osData.VersionID,
			"error", err)
		c.versionValue = 0
	} else {
		c.versionValue = version
	}
}

func (c *OSReleaseCollector) parseSupportEndTime() {
	if c.osData.SupportEnd == "" {
		c.supportEndTime = time.Time{}
		return
	}

	supportEnd, err := time.Parse(time.DateOnly, c.osData.SupportEnd)
	if err != nil {
		c.logger.Warn("Failed to parse support end date",
			"date", c.osData.SupportEnd,
			"error", err)
		c.supportEndTime = time.Time{}
	} else {
		c.supportEndTime = supportEnd
	}
}

func (p *EnvFileParser) Parse(reader io.Reader) (*OSRelease, error) {
	envData := make(map[string]string)
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()

		if len(line) == 0 || line[0] == '#' {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if len(value) > 1 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		}

		envData[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning failed: %w", err)
	}

	return &OSRelease{
		Name:            envData["NAME"],
		ID:              envData["ID"],
		IDLike:          envData["ID_LIKE"],
		PrettyName:      envData["PRETTY_NAME"],
		Variant:         envData["VARIANT"],
		VariantID:       envData["VARIANT_ID"],
		Version:         envData["VERSION"],
		VersionID:       envData["VERSION_ID"],
		VersionCodename: envData["VERSION_CODENAME"],
		BuildID:         envData["BUILD_ID"],
		ImageID:         envData["IMAGE_ID"],
		ImageVersion:    envData["IMAGE_VERSION"],
		SupportEnd:      envData["SUPPORT_END"],
		HomeURL:         envData["HOME_URL"],
		BugReportURL:    envData["BUG_REPORT_URL"],
		PlatformID:      envData["PLATFORM_ID"],
	}, nil
}

func (p *EnvFileParser) FileType() string {
	return "env"
}

func (p *PlistFileParser) Parse(reader io.Reader) (*OSRelease, error) {

	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("reading failed: %w", err)
	}

	var plist PlistDocument
	if err := xml.Unmarshal(content, &plist); err != nil {
		return nil, fmt.Errorf("XML unmarshalling failed: %w", err)
	}

	var osVersionID, osVersionName, osBuildID string
	dict := plist.Dict

	if len(dict.Key) > 0 {
		for i, key := range dict.Key {
			if i < len(dict.String) {
				switch key {
				case "ProductVersion":
					osVersionID = dict.String[i]
				case "ProductName":
					osVersionName = dict.String[i]
				case "ProductBuildVersion":
					osBuildID = dict.String[i]
				}
			}
		}
	}

	return &OSRelease{
		Name:      osVersionName,
		Version:   osVersionID,
		VersionID: osVersionID,
		BuildID:   osBuildID,
	}, nil
}

func (p *PlistFileParser) FileType() string {
	return "plist"
}

func (c *OSReleaseCollector) GetOSName() string {
	c.dataMutex.RLock()
	defer c.dataMutex.RUnlock()

	if c.osData == nil {
		return ""
	}
	return c.osData.Name
}

func (c *OSReleaseCollector) GetOSVersion() string {
	c.dataMutex.RLock()
	defer c.dataMutex.RUnlock()

	if c.osData == nil {
		return ""
	}
	return c.osData.Version
}

func (c *OSReleaseCollector) GetOSVersionValue() float64 {
	c.dataMutex.RLock()
	defer c.dataMutex.RUnlock()
	return c.versionValue
}

func (c *OSReleaseCollector) GetSupportEndTime() time.Time {
	c.dataMutex.RLock()
	defer c.dataMutex.RUnlock()
	return c.supportEndTime
}

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
func initializeGlobalLogger() {
	if globalLogger == nil {
		logHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
		globalLogger = slog.New(logHandler)
	}
}

// Construct scrape duration metric descriptor
func createScrapeDurationDescriptor() *prometheus.Desc {
	return prometheus.NewDesc(
		prometheus.BuildFQName(getNamespace(), "scrape", "collector_duration_seconds"),
		"filefd_node_exporter: Duration of a collector scrape operation",
		[]string{"collector"},
		nil,
	)
}

// Construct scrape success metric descriptor
func createScrapeSuccessDescriptor() *prometheus.Desc {
	return prometheus.NewDesc(
		prometheus.BuildFQName(getNamespace(), "scrape", "collector_success"),
		"filefd_node_exporter: Indicates successful collector execution",
		[]string{"collector"},
		nil,
	)
}

// Retrieve the namespace for metric naming
func getNamespace() string {
	return namespace
}

// Check if error is the specific no-data error
func IsNoDataError(err error) bool {
	return errors.Is(err, ErrNoData)
}

// Interface for file descriptor statistic collection
type FileFDStatCollector interface {
	Collector
	DescribeMetrics(ch chan<- *prometheus.Desc)
	CollectMetrics(ch chan<- prometheus.Metric) error
}

// Data structure for holding file descriptor statistics
type FileFDStats struct {
	Allocated uint64
	Maximum   uint64
}

// Collector implementation for file descriptor metrics
type fileFDStatCollector struct {
	logger  *slog.Logger
	metrics FileFDStats
}

// Package initialization function
func init() {
	initializeGlobalLogger()
	registerFileFDCollector()
}

func procFilePath(name string) string {
	return filepath.Join(*procPath, name)
}

// Register the collector with the exporter
func registerFileFDCollector() {
	collector, err := instantiateFileFDStatCollector()
	if err != nil {
		globalLogger.Error("Collector instantiation failure", "error", err)
		panic(fmt.Sprintf("cannot create filefd collector: %v", err))
	}

	if prometheusCollector, ok := collector.(prometheus.Collector); ok {
		exporter.Register(prometheusCollector)
	} else {
		globalLogger.Error("Type assertion to prometheus.Collector failed")
		panic("filefd collector is not a prometheus.Collector")
	}
}

// Create an instance of the file descriptor collector
func instantiateFileFDStatCollector() (Collector, error) {
	return NewFileFDStatCollector()
}

// Constructor for file descriptor collector
func NewFileFDStatCollector() (Collector, error) {
	initializeGlobalLogger()
	return &fileFDStatCollector{
		logger: globalLogger,
	}, nil
}

// Update method implementation for the collector
func (c *fileFDStatCollector) Update(ch chan<- prometheus.Metric) error {
	return c.CollectMetrics(ch)
}

// Describe method for prometheus collector interface
func (c *fileFDStatCollector) Describe(ch chan<- *prometheus.Desc) {
	c.DescribeMetrics(ch)
}

// Implement Collect method to satisfy prometheus.Collector interface
func (c *fileFDStatCollector) Collect(ch chan<- prometheus.Metric) {
	startTimestamp := time.Now()
	collectionSuccessful := true

	err := c.CollectMetrics(ch)
	if err != nil {
		collectionSuccessful = false
		if IsNoDataError(err) {
			c.logger.Debug("Collector returned empty dataset",
				"collector", fileFDStatSubsystem,
				"error", err)
		} else {
			c.logger.Error("Collector execution failed",
				"collector", fileFDStatSubsystem,
				"error", err)
		}
	} else {
		c.logger.Debug("Collector executed successfully",
			"collector", fileFDStatSubsystem)
	}

	collectionDuration := time.Since(startTimestamp)
	successValue := 0.0
	if collectionSuccessful {
		successValue = 1.0
	}

	// Emit collection duration metric
	ch <- prometheus.MustNewConstMetric(
		scrapeDurationDesc,
		prometheus.GaugeValue,
		collectionDuration.Seconds(),
		fileFDStatSubsystem,
	)

	// Emit collection success metric
	ch <- prometheus.MustNewConstMetric(
		scrapeSuccessDesc,
		prometheus.GaugeValue,
		successValue,
		fileFDStatSubsystem,
	)
}

// Describe collector metrics
func (c *fileFDStatCollector) DescribeMetrics(ch chan<- *prometheus.Desc) {
	ch <- constructAllocatedDescriptors()
	ch <- constructMaximumDescriptors()
}

// Create descriptor for allocated file descriptors metric
func constructAllocatedDescriptors() *prometheus.Desc {
	return prometheus.NewDesc(
		prometheus.BuildFQName(getNamespace(), fileFDStatSubsystem, "allocated"),
		"Number of allocated file descriptors",
		nil, nil,
	)
}

// Create descriptor for maximum file descriptors metric
func constructMaximumDescriptors() *prometheus.Desc {
	return prometheus.NewDesc(
		prometheus.BuildFQName(getNamespace(), fileFDStatSubsystem, "maximum"),
		"Maximum number of file descriptors",
		nil, nil,
	)
}

// Collect metrics implementation
func (c *fileFDStatCollector) CollectMetrics(ch chan<- prometheus.Metric) error {
	err := c.retrieveFileFDStatistics()
	if err != nil {
		return fmt.Errorf("statistics retrieval failure: %w", err)
	}

	c.emitCollectedMetrics(ch)
	return nil
}

// Retrieve file descriptor statistics from system
func (c *fileFDStatCollector) retrieveFileFDStatistics() error {
	procFile := getProcFilesystemPath("sys/fs/file-nr")
	stats, err := parseFileFDStatistics(procFile)
	if err != nil {
		return fmt.Errorf("cannot parse file descriptor stats: %w", err)
	}

	c.metrics.Allocated = stats.Allocated
	c.metrics.Maximum = stats.Maximum
	return nil
}

// Emit collected metrics to the channel
func (c *fileFDStatCollector) emitCollectedMetrics(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(
		constructAllocatedDescriptors(),
		prometheus.GaugeValue,
		float64(c.metrics.Allocated),
	)

	ch <- prometheus.MustNewConstMetric(
		constructMaximumDescriptors(),
		prometheus.GaugeValue,
		float64(c.metrics.Maximum),
	)
}

// Get path in proc filesystem
func getProcFilesystemPath(relativePath string) string {
	return procFilePath(relativePath)
}

// Parse file descriptor statistics from file
func parseFileFDStatistics(filename string) (*FileFDStats, error) {
	fileHandle, err := accessSystemFile(filename)
	if err != nil {
		return nil, err
	}
	defer safelyCloseFile(fileHandle)

	fileContent, err := readFileData(fileHandle)
	if err != nil {
		return nil, err
	}

	return interpretFileContent(fileContent, filename)
}

// Open system file for reading
func accessSystemFile(path string) (*os.File, error) {
	cleanPath := filepath.Clean(path)
	statDir := *procPath
	if !strings.HasPrefix(cleanPath, statDir) {
		return nil, fmt.Errorf("file must be located within %s", statDir)
	}
	return os.Open(path)
}

// Safely close file handle with error handling
func safelyCloseFile(file *os.File) {
	if err := file.Close(); err != nil {
		globalLogger.Warn("File closure encountered issue",
			"error", err)
	}
}

// Read all data from file
func readFileData(file *os.File) ([]byte, error) {
	return io.ReadAll(file)
}

// Interpret and process file content
func interpretFileContent(content []byte, filename string) (*FileFDStats, error) {
	trimmedContent := bytes.TrimSpace(content)
	dataSegments := bytes.Split(trimmedContent, []byte("\u0009"))

	if len(dataSegments) < 3 {
		return nil, fmt.Errorf("unexpected data format in %q", filename)
	}

	allocatedValue, err := convertToUint64(dataSegments[0], "allocated")
	if err != nil {
		return nil, err
	}

	maximumValue, err := convertToUint64(dataSegments[2], "maximum")
	if err != nil {
		return nil, err
	}

	return &FileFDStats{
		Allocated: allocatedValue,
		Maximum:   maximumValue,
	}, nil
}

// Convert byte data to uint64 with error context
func convertToUint64(data []byte, fieldName string) (uint64, error) {
	value, err := strconv.ParseUint(string(data), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("conversion error for %s: %w", fieldName, err)
	}
	return value, nil
}

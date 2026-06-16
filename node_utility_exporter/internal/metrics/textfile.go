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

func init() {
	collectorFactory := func() (Collector, error) {
		return NewTextFileCollector()
	}
	registerTextfileCollector(collectorFactory)
}

func registerTextfileCollector(factory func() (Collector, error)) {
	collector, err := factory()
	if err != nil {
		panic(fmt.Sprintf("failed to create textfile collector: %v", err))
	}

	if metricCollector, ok := collector.(prometheus.Collector); ok {
		exporter.Register(metricCollector)
	} else {
		panic("textfile collector does not implement prometheus.Collector")
	}
}

func NewTextFileCollector() (Collector, error) {
	collector := &TextfileCollector{
		refreshInterval: defaultRefreshIntervalTextFile,
		metricCache:     make(map[string]prometheus.Metric),
		fileMetrics:     make(map[string]time.Time),
	}

	collector.initializeLogger()

	collector.initializeDescriptors()

	if err := collector.initializePaths(); err != nil {
		collector.logger.Warn("Initial textfile paths setup failed", "error", err)
	}

	return collector, nil
}

func (c *TextfileCollector) initializeLogger() {
	c.logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

func (c *TextfileCollector) initializeDescriptors() {
	c.collectorStatus = prometheus.NewDesc(
		prometheus.BuildFQName(namespaceTextFile, subsystemNameTextFile, "collector_status"),
		"Textfile collector status (1 = success, 0 = failure)",
		[]string{"subsystem"}, nil,
	)
}

func (c *TextfileCollector) initializePaths() error {
	if textFileDirectories == nil {
		return errors.New("command-line flags not parsed yet")
	}

	c.paths = *textFileDirectories
	c.logger.Info("Textfile directories initialized", "count", len(c.paths))
	return nil
}

func (c *TextfileCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.collectorStatus
	ch <- mtimeDesc
	ch <- scrapeErrorDesc
	ch <- fileCountDesc
	ch <- metricCountDesc
	ch <- processingTimeDesc
}

func (c *TextfileCollector) Collect(ch chan<- prometheus.Metric) {
	startTime := time.Now()

	if time.Since(c.lastRefreshed) > c.refreshInterval {
		if err := c.refreshFiles(); err != nil {
			c.logger.Error("Failed to refresh text files", "error", err)
			ch <- prometheus.MustNewConstMetric(
				c.collectorStatus, prometheus.GaugeValue, 0, "refresh")
		} else {
			ch <- prometheus.MustNewConstMetric(
				c.collectorStatus, prometheus.GaugeValue, 1, "refresh")
		}
	}

	if err := c.Update(ch); err != nil {
		c.logger.Error("Failed to update textfile metrics", "error", err)
		ch <- prometheus.MustNewConstMetric(
			c.collectorStatus, prometheus.GaugeValue, 0, "update")
	} else {
		ch <- prometheus.MustNewConstMetric(
			c.collectorStatus, prometheus.GaugeValue, 1, "update")
	}

	duration := time.Since(startTime).Seconds()
	ch <- prometheus.MustNewConstMetric(
		processingTimeDesc, prometheus.GaugeValue, duration)
}

func (c *TextfileCollector) Update(ch chan<- prometheus.Metric) error {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	c.exportMTimes(ch)

	metricCount := 0
	for _, metric := range c.metricCache {
		ch <- metric
		metricCount++
	}

	fileCount := len(c.fileMetrics)
	ch <- prometheus.MustNewConstMetric(
		fileCountDesc, prometheus.GaugeValue, float64(fileCount))

	ch <- prometheus.MustNewConstMetric(
		metricCountDesc, prometheus.GaugeValue, float64(metricCount))

	ch <- prometheus.MustNewConstMetric(
		scrapeErrorDesc, prometheus.GaugeValue, 0)

	c.logger.Debug("Textfile metrics updated",
		"files", fileCount,
		"metrics", metricCount)

	return nil
}

func (c *TextfileCollector) refreshFiles() error {
	c.logger.Info("Refreshing text files")
	startTime := time.Now()

	filePaths, err := c.getAllFilePaths()
	if err != nil {
		return err
	}

	newCache := make(map[string]prometheus.Metric)
	newFileMetrics := make(map[string]time.Time)
	fileCount := 0
	metricCount := 0
	errorCount := 0

	for _, filePath := range filePaths {
		metrics, mtime, err := c.processFile(filePath)
		if err != nil {
			c.logger.Warn("Failed to process text file",
				"file", filePath, "error", err)
			errorCount++
			continue
		}

		if len(metrics) == 0 {
			c.logger.Debug("No valid metrics found in file", "file", filePath)
			continue
		}

		for _, metric := range metrics {
			key := c.generateMetricKey(metric)
			newCache[key] = metric
			metricCount++
		}

		newFileMetrics[filePath] = *mtime
		fileCount++
	}

	c.cacheMutex.Lock()
	c.metricCache = newCache
	c.fileMetrics = newFileMetrics
	c.lastRefreshed = time.Now()
	c.cacheMutex.Unlock()

	duration := time.Since(startTime).Seconds()
	c.logger.Info("Text files refreshed",
		"files", fileCount,
		"metrics", metricCount,
		"errors", errorCount,
		"duration_seconds", duration)

	return nil
}

func (c *TextfileCollector) getAllFilePaths() ([]string, error) {
	var allPaths []string

	for _, globPath := range c.paths {
		matches, err := filepath.Glob(globPath)
		if err != nil {
			c.logger.Warn("Invalid glob pattern", "pattern", globPath, "error", err)
			continue
		}

		if len(matches) == 0 {
			matches = []string{globPath}
		}

		for _, path := range matches {
			fileInfo, err := os.Stat(path)
			if err != nil {
				if os.IsNotExist(err) {
					c.logger.Debug("Path does not exist", "path", path)
					continue
				}
				return nil, fmt.Errorf("failed to stat path %s: %w", path, err)
			}

			if fileInfo.IsDir() {
				files, err := os.ReadDir(path)
				if err != nil {
					c.logger.Warn("Failed to read directory", "path", path, "error", err)
					continue
				}

				for _, file := range files {
					if file.IsDir() {
						continue
					}
					if strings.HasSuffix(file.Name(), fileExtension) {
						filePath := filepath.Join(path, file.Name())
						allPaths = append(allPaths, filePath)
					}
				}
			} else {
				if strings.HasSuffix(path, fileExtension) {
					allPaths = append(allPaths, path)
				}
			}
		}
	}

	uniquePaths := make(map[string]struct{})
	var uniqueList []string
	for _, path := range allPaths {
		if _, exists := uniquePaths[path]; !exists {
			uniquePaths[path] = struct{}{}
			uniqueList = append(uniqueList, path)
		}
	}

	sort.Strings(uniqueList)
	c.logger.Debug("Found text files", "count", len(uniqueList))
	return uniqueList, nil
}

func (c *TextfileCollector) processFile(filePath string) ([]prometheus.Metric, *time.Time, error) {
	cleanPath := filepath.Clean(filePath)
	file, err := os.Open(cleanPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get file info: %w", err)
	}

	if fileInfo.Size() > maxFileSizeTextFile {
		return nil, nil, fmt.Errorf("%w: %d bytes", ErrFileTooLarge, fileInfo.Size())
	}

	content, err := io.ReadAll(io.LimitReader(file, maxFileSizeTextFile))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file content: %w", err)
	}

	metrics, err := c.parseMetrics(content, filePath)
	if err != nil {
		return nil, nil, err
	}

	mtime := fileInfo.ModTime()
	return metrics, &mtime, nil
}

func (c *TextfileCollector) parseMetrics(content []byte, filePath string) ([]prometheus.Metric, error) {
	var metrics []prometheus.Metric
	scanner := bufio.NewScanner(bytes.NewReader(content))
	scanner.Buffer(make([]byte, maxLineLengthTextFile), maxLineLengthTextFile)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		metric, err := c.parseMetricLine(line, filePath, lineNum)
		if err != nil {
			c.logger.Warn("Failed to parse metric line",
				"file", filePath,
				"line", lineNum,
				"error", err)
			continue
		}

		metrics = append(metrics, metric)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	if len(metrics) == 0 {
		return nil, ErrNoMetricsFound
	}

	return metrics, nil
}

func (c *TextfileCollector) parseMetricLine(line, filePath string, lineNum int) (prometheus.Metric, error) {
	// Basic format: <metric_name>{<labels>} <value> [<timestamp>]

	parts := strings.SplitN(line, " ", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("%w: missing value", ErrInvalidMetricFormat)
	}

	namePart := strings.TrimSpace(parts[0])
	valuePart := strings.TrimSpace(parts[1])

	valueParts := strings.Fields(valuePart)
	if len(valueParts) > 1 {
		if _, err := strconv.ParseInt(valueParts[1], 10, 64); err == nil {
			return nil, fmt.Errorf("%w: line %d", ErrTimestampNotAllowed, lineNum)
		}

		valuePart = valueParts[0]
	}

	var metricName string
	var labels map[string]string

	if strings.Contains(namePart, "{") {
		braceIndex := strings.Index(namePart, "{")
		metricName = strings.TrimSpace(namePart[:braceIndex])
		labelPart := strings.TrimSpace(namePart[braceIndex+1 : len(namePart)-1])

		var err error
		labels, err = c.parseLabels(labelPart)
		if err != nil {
			return nil, fmt.Errorf("label parsing failed: %w", err)
		}
	} else {
		metricName = namePart
		labels = make(map[string]string)
	}

	labels["file"] = filePath
	labels["line"] = strconv.Itoa(lineNum)

	value, err := strconv.ParseFloat(valuePart, 64)
	if err != nil {
		return nil, fmt.Errorf("value parsing failed: %w", err)
	}

	metricType := prometheus.GaugeValue

	desc := prometheus.NewDesc(
		metricName,
		"Metric collected from text file",
		nil, labels,
	)

	return prometheus.NewConstMetric(desc, metricType, value)
}

func (c *TextfileCollector) parseLabels(labelPart string) (map[string]string, error) {
	labels := make(map[string]string)

	pairs := strings.Split(labelPart, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("%w: invalid label pair '%s'", ErrInvalidMetricFormat, pair)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		}

		labels[key] = value
	}

	return labels, nil
}

func (c *TextfileCollector) exportMTimes(ch chan<- prometheus.Metric) {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	if len(c.fileMetrics) == 0 {
		return
	}

	filePaths := make([]string, 0, len(c.fileMetrics))
	for path := range c.fileMetrics {
		filePaths = append(filePaths, path)
	}
	sort.Strings(filePaths)

	for _, path := range filePaths {
		mtime := c.fileMetrics[path]
		mtimeSec := float64(mtime.Unix())
		ch <- prometheus.MustNewConstMetric(mtimeDesc, prometheus.GaugeValue, mtimeSec, path)
	}
}

func (c *TextfileCollector) generateMetricKey(metric prometheus.Metric) string {
	desc := metric.Desc()

	key := desc.String()

	return sanitizeKeyPart(key)
}

func sanitizeKeyPart(part string) string {
	replacer := strings.NewReplacer(
		" ", "_",
		".", "_",
		":", "_",
		"-", "_",
		"/", "_",
		"\\", "_",
		"{", "_",
		"}", "_",
		",", "_",
		"=", "_",
		"\"", "",
		"'", "",
		"(", "",
		")", "",
		"[", "",
		"]", "",
	)
	return replacer.Replace(part)
}

func (c *TextfileCollector) GetProcessedFiles() []string {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	files := make([]string, 0, len(c.fileMetrics))
	for file := range c.fileMetrics {
		files = append(files, file)
	}
	sort.Strings(files)
	return files
}

func (c *TextfileCollector) GetFileMetrics(filePath string) int {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	count := 0
	for _, metric := range c.metricCache {
		if desc := metric.Desc().String(); strings.Contains(desc, filePath) {
			count++
		}
	}
	return count
}

func (c *TextfileCollector) RefreshNow() error {
	return c.refreshFiles()
}

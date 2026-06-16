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

func init() {

	collectorFactory := func() (Collector, error) {
		return NewLnstatCollector()
	}

	registerLnstatCollector(collectorFactory)
}

func registerLnstatCollector(factory func() (Collector, error)) {
	collector, err := factory()
	if err != nil {
		panic(fmt.Sprintf("failed to create lnstat collector: %v", err))
	}

	if metricCollector, ok := collector.(prometheus.Collector); ok {
		exporter.Register(metricCollector)
	} else {
		panic("lnstat collector does not implement prometheus.Collector")
	}
}

func NewLnstatCollector() (Collector, error) {
	collector := &lnstatCollector{
		procPath:        "/proc",
		refreshInterval: refreshInterval,
		metricDescs:     make(map[string]*prometheus.Desc),
	}

	collector.initializeLogger()

	collector.initializeDescriptors()

	if err := collector.refreshStats(); err != nil {
		collector.logger.Warn("Initial lnstat data load failed", "error", err)
	}

	return collector, nil
}

func (c *lnstatCollector) initializeLogger() {

	c.logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

func (c *lnstatCollector) initializeDescriptors() {
	c.collectorStatus = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subsystemLnstatName, "collector_status"),
		"Lnstat collector status (1 = success, 0 = failure)",
		[]string{"subsystem"}, nil,
	)
}

func (c *lnstatCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.collectorStatus

	c.descsMutex.RLock()
	defer c.descsMutex.RUnlock()
	for _, desc := range c.metricDescs {
		ch <- desc
	}
}

func (c *lnstatCollector) Collect(ch chan<- prometheus.Metric) {
	if time.Since(c.lastRefreshed) > c.refreshInterval {
		if err := c.refreshStats(); err != nil {
			c.logger.Error("Failed to refresh lnstat data", "error", err)
			ch <- prometheus.MustNewConstMetric(
				c.collectorStatus, prometheus.GaugeValue, 0, "refresh")
		} else {
			ch <- prometheus.MustNewConstMetric(
				c.collectorStatus, prometheus.GaugeValue, 1, "refresh")
		}
	}

	if err := c.Update(ch); err != nil {
		c.logger.Error("Failed to update lnstat metrics", "error", err)
		ch <- prometheus.MustNewConstMetric(
			c.collectorStatus, prometheus.GaugeValue, 0, "update")
	} else {
		ch <- prometheus.MustNewConstMetric(
			c.collectorStatus, prometheus.GaugeValue, 1, "update")
	}
}

func (c *lnstatCollector) Update(ch chan<- prometheus.Metric) error {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	if len(c.statsCache) == 0 {
		c.logger.Debug("No network statistics data available")
		return nil
	}

	var lastErr error
	successCount := 0

	for _, netStatFile := range c.statsCache {
		if err := c.processStatFile(netStatFile, ch); err != nil {
			c.logger.Warn("Failed to process stat file",
				"file", netStatFile.Filename,
				"error", err)
			lastErr = err
		} else {
			successCount++
		}
	}

	c.logger.Debug("Processed network statistics",
		"files", len(c.statsCache),
		"success", successCount)

	return lastErr
}

func (c *lnstatCollector) refreshStats() error {
	c.logger.Info("Refreshing network statistics data")

	statFiles, err := c.getStatFiles()
	if err != nil {
		return err
	}

	var stats []NetworkStatFile
	for _, file := range statFiles {
		statFile, err := c.parseStatFile(file)
		if err != nil {
			c.logger.Warn("Failed to parse stat file", "file", file, "error", err)
			continue
		}
		stats = append(stats, *statFile)
	}

	c.cacheMutex.Lock()
	c.statsCache = stats
	c.lastRefreshed = time.Now()
	c.cacheMutex.Unlock()

	c.logger.Info("Network statistics refreshed", "files", len(stats))
	return nil
}

func (c *lnstatCollector) getStatFiles() ([]string, error) {
	statDir := filepath.Join(c.procPath, "net", "stat")

	if _, err := os.Stat(statDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", ErrProcStatNotFound, statDir)
	}

	files, err := filepath.Glob(filepath.Join(statDir, statFilePattern))
	if err != nil {
		return nil, fmt.Errorf("failed to list stat files: %w", err)
	}

	var statFiles []string
	for _, file := range files {
		if fileInfo, err := os.Stat(file); err == nil && !fileInfo.IsDir() {
			statFiles = append(statFiles, file)
		}
	}

	return statFiles, nil
}

func (c *lnstatCollector) parseStatFile(filePath string) (*NetworkStatFile, error) {
	cleanPath := filepath.Clean(filePath)
	statDir := "/proc"
	if !strings.HasPrefix(cleanPath, statDir) {
		return nil, fmt.Errorf("stat file must be located within %s", statDir)
	}
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrStatFileReadFailed, err)
	}
	defer file.Close()

	statFile := &NetworkStatFile{
		Filename: filepath.Base(filePath),
	}

	scanner := bufio.NewScanner(file)
	buf := make([]byte, maxLineLength)
	scanner.Buffer(buf, maxLineLength)

	lineCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		if lineCount == 1 {
			if !strings.HasPrefix(line, headerPrefix) {
				return nil, fmt.Errorf("%w: expected '%s' prefix", ErrInvalidHeaderFormat, headerPrefix)
			}

			headers := strings.Fields(line)
			if len(headers) < 2 {
				return nil, ErrInvalidHeaderFormat
			}
			statFile.Headers = headers[1:]
			continue
		}

		if strings.HasPrefix(line, cpuPrefix) {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}

			cpuID, err := strconv.Atoi(fields[0][len(cpuPrefix):])
			if err != nil {
				c.logger.Warn("Invalid CPU ID", "line", line, "error", err)
				continue
			}

			if len(statFile.Stats) <= cpuID {
				newStats := make([]map[string]uint64, cpuID+1)
				copy(newStats, statFile.Stats)
				statFile.Stats = newStats
			}

			if statFile.Stats[cpuID] == nil {
				statFile.Stats[cpuID] = make(map[string]uint64)
			}

			for i, field := range fields[1:] {
				if i >= len(statFile.Headers) {
					break
				}

				value, err := strconv.ParseUint(field, 16, 64)
				if err != nil {
					c.logger.Warn("Failed to parse value",
						"file", statFile.Filename,
						"cpu", cpuID,
						"header", statFile.Headers[i],
						"value", field,
						"error", err)
					continue
				}

				statFile.Stats[cpuID][statFile.Headers[i]] = value
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	return statFile, nil
}

func (c *lnstatCollector) processStatFile(file NetworkStatFile, ch chan<- prometheus.Metric) error {
	if len(file.Headers) == 0 {
		return fmt.Errorf("no headers in file %s", file.Filename)
	}

	var processErr error

	for _, header := range file.Headers {
		metricName := sanitizeMetricName(header) + "_total"

		desc := c.getOrCreateDesc(metricName, file.Filename)

		for cpuID, cpuStats := range file.Stats {
			if cpuStats == nil {
				continue
			}

			value, exists := cpuStats[header]
			if !exists {
				continue
			}

			ch <- prometheus.MustNewConstMetric(
				desc,
				prometheus.CounterValue,
				float64(value),
				file.Filename,
				strconv.Itoa(cpuID),
			)
		}
	}

	return processErr
}

func (c *lnstatCollector) getOrCreateDesc(metricName, filename string) *prometheus.Desc {
	descKey := filename + "_" + metricName

	c.descsMutex.RLock()
	desc, exists := c.metricDescs[descKey]
	c.descsMutex.RUnlock()

	if exists {
		return desc
	}

	newDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subsystemLnstatName, metricName),
		"Linux network cache statistics",
		[]string{"subsystem", "cpu"}, nil,
	)

	c.descsMutex.Lock()
	c.metricDescs[descKey] = newDesc
	c.descsMutex.Unlock()

	return newDesc
}

func sanitizeMetricName(name string) string {
	replacer := strings.NewReplacer(
		" ", "_",
		"-", "_",
		".", "_",
		":", "_",
	)

	return strings.ToLower(replacer.Replace(name))
}

func (c *lnstatCollector) GetStatFileNames() []string {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	names := make([]string, len(c.statsCache))
	for i, file := range c.statsCache {
		names[i] = file.Filename
	}
	return names
}

func (c *lnstatCollector) GetStatHeaders(filename string) []string {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	for _, file := range c.statsCache {
		if file.Filename == filename {
			return file.Headers
		}
	}
	return nil
}

func (c *lnstatCollector) GetStatValue(filename, header string, cpu int) (uint64, bool) {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	for _, file := range c.statsCache {
		if file.Filename == filename && len(file.Stats) > cpu {
			if value, exists := file.Stats[cpu][header]; exists {
				return value, true
			}
		}
	}
	return 0, false
}

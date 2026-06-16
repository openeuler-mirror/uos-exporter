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

func init() {
	collectorFactory := func() (Collector, error) {
		return NewSysctlCollector()
	}

	registerSysctlCollector(collectorFactory)
}

func registerSysctlCollector(factory func() (Collector, error)) {
	collector, err := factory()
	if err != nil {
		panic(fmt.Sprintf("failed to create sysctl collector: %v", err))
	}

	if metricCollector, ok := collector.(prometheus.Collector); ok {
		exporter.Register(metricCollector)
	} else {
		panic("sysctl collector does not implement prometheus.Collector")
	}
}

func NewSysctlCollector() (Collector, error) {
	collector := &SysctlCollector{
		refreshInterval: defaultRefreshIntervalSysctl,
		metricDescs:     make(map[string]*prometheus.Desc),
	}

	collector.initializeLogger()

	collector.initializeDescriptors()

	if err := collector.initializeSysctls(); err != nil {
		collector.logger.Warn("Initial sysctl configuration failed", "error", err)
	}

	return collector, nil
}

func (c *SysctlCollector) initializeLogger() {
	c.logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

func (c *SysctlCollector) initializeDescriptors() {
	c.collectorStatus = prometheus.NewDesc(
		prometheus.BuildFQName(namespaceSysctl, subsystemNameSysctl, "collector_status"),
		"Sysctl collector status (1 = success, 0 = failure)",
		[]string{"subsystem"}, nil,
	)

	c.infoDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespaceSysctl, subsystemNameSysctl, "info"),
		"Sysctl configuration information",
		[]string{"name", "value", "index"}, nil,
	)
}

func (c *SysctlCollector) initializeSysctls() error {
	if c.initialized {
		return nil
	}

	c.logger.Info("Initializing sysctl collector configuration")

	if sysctlInclude == nil || sysctlIncludeInfo == nil {
		return errors.New("command-line flags not parsed yet")
	}

	for _, include := range *sysctlInclude {
		s, err := c.parseSysctlConfig(include, true)
		if err != nil {
			c.logger.Warn("Failed to parse sysctl include", "include", include, "error", err)
			continue
		}
		c.sysctls = append(c.sysctls, s)
	}

	for _, include := range *sysctlIncludeInfo {
		s, err := c.parseSysctlConfig(include, false)
		if err != nil {
			c.logger.Warn("Failed to parse sysctl include info", "include", include, "error", err)
			continue
		}
		c.sysctls = append(c.sysctls, s)
	}

	c.initialized = true
	c.logger.Info("Sysctl collector initialized", "count", len(c.sysctls))
	return nil
}

func (c *SysctlCollector) parseSysctlConfig(include string, numeric bool) (*Sysctl, error) {
	parts := strings.SplitN(include, ":", 2)
	s := &Sysctl{
		Numeric: numeric,
	}

	if len(parts) == 1 {
		s.Name = parts[0]
	} else if len(parts) == 2 {
		s.Name = parts[0]
		s.Keys = strings.Split(parts[1], ",")
	}

	s.Name = strings.TrimSpace(s.Name)
	if s.Name == "" {
		return nil, errors.New("empty sysctl name")
	}

	s.Description = fmt.Sprintf("sysctl %s", s.Name)

	return s, nil
}

func (c *SysctlCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.collectorStatus
	ch <- c.infoDesc

	if err := c.initializeSysctls(); err != nil {
		c.logger.Error("Failed to initialize sysctl collector for Describe", "error", err)
		return
	}

	for _, s := range c.sysctls {
		if s.Numeric {
			if len(s.Keys) > 0 {
				for _, key := range s.Keys {
					desc := c.getOrCreateMetricDesc(s, key)
					ch <- desc
				}
			} else {
				desc := c.getOrCreateMetricDesc(s, "")
				ch <- desc
			}
		} else {
			ch <- c.infoDesc
		}
	}
}

func (c *SysctlCollector) Collect(ch chan<- prometheus.Metric) {
	if time.Since(c.lastRefreshed) > c.refreshInterval {
		if err := c.refreshSysctlValues(); err != nil {
			c.logger.Error("Failed to refresh sysctl values", "error", err)
			ch <- prometheus.MustNewConstMetric(
				c.collectorStatus, prometheus.GaugeValue, 0, "refresh")
		} else {
			ch <- prometheus.MustNewConstMetric(
				c.collectorStatus, prometheus.GaugeValue, 1, "refresh")
		}
	}

	if err := c.Update(ch); err != nil {
		c.logger.Error("Failed to update sysctl metrics", "error", err)
		ch <- prometheus.MustNewConstMetric(
			c.collectorStatus, prometheus.GaugeValue, 0, "update")
	} else {
		ch <- prometheus.MustNewConstMetric(
			c.collectorStatus, prometheus.GaugeValue, 1, "update")
	}
}

func (c *SysctlCollector) Update(ch chan<- prometheus.Metric) error {
	if err := c.initializeSysctls(); err != nil {
		return err
	}

	var lastErr error
	successCount := 0

	for _, s := range c.sysctls {
		metrics, err := c.createMetrics(s)
		if err != nil {
			c.logger.Warn("Failed to create metrics for sysctl",
				"sysctl", s.Name, "error", err)
			lastErr = err
			continue
		}

		for _, metric := range metrics {
			ch <- metric
		}
		successCount++
	}

	c.logger.Debug("Processed sysctl metrics",
		"sysctls", len(c.sysctls),
		"success", successCount)

	return lastErr
}

func (c *SysctlCollector) refreshSysctlValues() error {
	c.logger.Info("Refreshing sysctl values")

	for _, s := range c.sysctls {
		value, err := c.readSysctlValue(s.Name)
		if err != nil {
			s.ReadError = err
			c.logger.Warn("Failed to read sysctl value",
				"sysctl", s.Name, "error", err)
			continue
		}

		s.LastValue = value
		s.LastRead = time.Now()
		s.ReadError = nil

		switch value.(type) {
		case []int:
			s.ValueType = "int_array"
		case []string:
			s.ValueType = "string_array"
		case int:
			s.ValueType = "int"
		case string:
			s.ValueType = "string"
		default:
			s.ValueType = "unknown"
		}
	}

	c.lastRefreshed = time.Now()
	c.logger.Info("Sysctl values refreshed", "count", len(c.sysctls))
	return nil
}

func (c *SysctlCollector) readSysctlValue(name string) (interface{}, error) {
	filePath, err := c.sysctlToPath(name)
	if err != nil {
		return nil, err
	}

	content, err := c.readSysctlFile(filePath)
	if err != nil {
		return nil, err
	}

	if intValues, err := c.parseAsIntArray(content); err == nil {
		return intValues, nil
	}

	if stringValues, err := c.parseAsStringArray(content); err == nil {
		return stringValues, nil
	}

	if intValue, err := strconv.Atoi(strings.TrimSpace(string(content))); err == nil {
		return intValue, nil
	}

	return strings.TrimSpace(string(content)), nil
}

func (c *SysctlCollector) sysctlToPath(name string) (string, error) {

	if strings.Contains(name, "..") || strings.HasPrefix(name, "/") {
		return "", ErrPathTraversal
	}

	path := filepath.Join(procSysPath, strings.ReplaceAll(name, ".", "/"))

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", fmt.Errorf("%w: %s", ErrSysctlNotFound, path)
	}

	return path, nil
}

func (c *SysctlCollector) readSysctlFile(path string) ([]byte, error) {
	cleanPath := filepath.Clean(path)
	if !strings.HasPrefix(cleanPath, procSysPath) {
		return nil, ErrPathTraversal
	}
	file, err := os.Open(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrFileReadFailed, err)
	}
	defer file.Close()

	limitedReader := io.LimitReader(file, maxFileSize)

	content, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	return content, nil
}

func (c *SysctlCollector) parseAsIntArray(content []byte) ([]int, error) {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	scanner.Split(bufio.ScanWords)

	var values []int
	for scanner.Scan() {
		value, err := strconv.Atoi(scanner.Text())
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidValueFormat, scanner.Text())
		}
		values = append(values, value)
	}

	if len(values) == 0 {
		return nil, ErrInvalidValueFormat
	}

	return values, nil
}

func (c *SysctlCollector) parseAsStringArray(content []byte) ([]string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	scanner.Split(bufio.ScanWords)

	var values []string
	for scanner.Scan() {
		values = append(values, scanner.Text())
	}

	if len(values) == 0 {
		return nil, ErrInvalidValueFormat
	}

	return values, nil
}

func (c *SysctlCollector) createMetrics(s *Sysctl) ([]prometheus.Metric, error) {
	if s.LastValue == nil || time.Since(s.LastRead) > c.refreshInterval {
		if value, err := c.readSysctlValue(s.Name); err == nil {
			s.LastValue = value
			s.LastRead = time.Now()
		} else if s.ReadError != nil {
			return nil, s.ReadError
		}
	}

	switch v := s.LastValue.(type) {
	case []int:
		return c.createMetricsForIntArray(s, v)
	case []string:
		return c.createMetricsForStringArray(s, v)
	case int:
		return c.createMetricsForInt(s, v)
	case string:
		return c.createMetricsForString(s, v)
	default:
		return nil, fmt.Errorf("%w: %T", ErrUnsupportedValueType, v)
	}
}

func (c *SysctlCollector) createMetricsForIntArray(s *Sysctl, values []int) ([]prometheus.Metric, error) {
	switch len(values) {
	case 0:
		return nil, fmt.Errorf("sysctl %s has no values", s.Name)
	case 1:
		if len(s.Keys) > 0 {
			return nil, fmt.Errorf("sysctl %s has only one value, but expected %v", s.Name, s.Keys)
		}
		return []prometheus.Metric{c.createConstMetric(s, values[0])}, nil
	default:
		if len(s.Keys) == 0 {
			return c.createIndexedMetrics(s, values), nil
		}

		if len(values) != len(s.Keys) {
			return nil, fmt.Errorf("sysctl %s has %d values but only %d keys defined", s.Name, len(values), len(s.Keys))
		}

		return c.createMappedMetrics(s, values)
	}
}

func (c *SysctlCollector) createMetricsForStringArray(s *Sysctl, values []string) ([]prometheus.Metric, error) {
	switch len(values) {
	case 0:
		return nil, fmt.Errorf("sysctl %s has no values", s.Name)
	case 1:
		if len(s.Keys) > 0 {
			return nil, fmt.Errorf("sysctl %s has only one value, but expected %v", s.Name, s.Keys)
		}
		return []prometheus.Metric{c.createInfoMetric(s, values[0], "0")}, nil
	default:
		if len(s.Keys) == 0 {
			return c.createIndexedInfoMetrics(s, values), nil
		}

		if len(values) != len(s.Keys) {
			return nil, fmt.Errorf("sysctl %s has %d values but only %d keys defined", s.Name, len(values), len(s.Keys))
		}

		return nil, errors.New("mapped sysctl string values not supported")
	}
}

func (c *SysctlCollector) createMetricsForInt(s *Sysctl, value int) ([]prometheus.Metric, error) {
	if len(s.Keys) > 0 {
		return nil, fmt.Errorf("sysctl %s is single value, but keys were specified", s.Name)
	}
	return []prometheus.Metric{c.createConstMetric(s, value)}, nil
}

func (c *SysctlCollector) createMetricsForString(s *Sysctl, value string) ([]prometheus.Metric, error) {
	if len(s.Keys) > 0 {
		return nil, fmt.Errorf("sysctl %s is single value, but keys were specified", s.Name)
	}
	return []prometheus.Metric{c.createInfoMetric(s, value, "0")}, nil
}

func (c *SysctlCollector) createConstMetric(s *Sysctl, value int) prometheus.Metric {
	desc := c.getOrCreateMetricDesc(s, "")
	return prometheus.MustNewConstMetric(
		desc,
		prometheus.GaugeValue,
		float64(value),
	)
}

func (c *SysctlCollector) createInfoMetric(s *Sysctl, value, index string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		c.infoDesc,
		prometheus.GaugeValue,
		1.0,
		s.Name,
		value,
		index,
	)
}

func (c *SysctlCollector) createIndexedMetrics(s *Sysctl, values []int) []prometheus.Metric {
	desc := c.getOrCreateMetricDesc(s, "")

	metrics := make([]prometheus.Metric, len(values))
	for i, value := range values {
		metrics[i] = prometheus.MustNewConstMetric(
			desc,
			prometheus.GaugeValue,
			float64(value),
			strconv.Itoa(i),
		)
	}
	return metrics
}

func (c *SysctlCollector) createIndexedInfoMetrics(s *Sysctl, values []string) []prometheus.Metric {
	metrics := make([]prometheus.Metric, len(values))
	for i, value := range values {
		metrics[i] = c.createInfoMetric(s, value, strconv.Itoa(i))
	}
	return metrics
}

func (c *SysctlCollector) createMappedMetrics(s *Sysctl, values []int) ([]prometheus.Metric, error) {
	metrics := make([]prometheus.Metric, len(values))
	for i, value := range values {
		if i >= len(s.Keys) {
			break
		}

		key := s.Keys[i]
		desc := c.getOrCreateMetricDesc(s, key)

		metrics[i] = prometheus.MustNewConstMetric(
			desc,
			prometheus.GaugeValue,
			float64(value),
		)
	}
	return metrics, nil
}

func (c *SysctlCollector) getOrCreateMetricDesc(s *Sysctl, key string) *prometheus.Desc {
	descKey := s.Name
	if key != "" {
		descKey += "_" + key
	}

	c.descsMutex.RLock()
	desc, exists := c.metricDescs[descKey]
	c.descsMutex.RUnlock()

	if exists {
		return desc
	}

	var newDesc *prometheus.Desc
	if key != "" {
		newDesc = prometheus.NewDesc(
			prometheus.BuildFQName(namespaceSysctl, subsystemNameSysctl, sanitizeSysctlMetricName(s.Name)+"_"+sanitizeSysctlMetricName(key)),
			fmt.Sprintf("%s, field %s", s.Description, key),
			nil, nil,
		)
	} else {
		newDesc = prometheus.NewDesc(
			prometheus.BuildFQName(namespaceSysctl, subsystemNameSysctl, sanitizeSysctlMetricName(s.Name)),
			s.Description,
			[]string{"index"}, nil,
		)
	}

	c.descsMutex.Lock()
	c.metricDescs[descKey] = newDesc
	c.descsMutex.Unlock()

	return newDesc
}

func sanitizeSysctlMetricName(name string) string {
	replacer := strings.NewReplacer(
		" ", "_",
		".", "_",
		":", "_",
		"-", "_",
		"/", "_",
	)

	return strings.ToLower(replacer.Replace(name))
}

func (c *SysctlCollector) GetSysctlValue(name string) (interface{}, error) {
	for _, s := range c.sysctls {
		if s.Name == name {
			return s.LastValue, s.ReadError
		}
	}
	return nil, ErrSysctlNotFound
}

func (c *SysctlCollector) GetSysctlNames() []string {
	names := make([]string, len(c.sysctls))
	for i, s := range c.sysctls {
		names[i] = s.Name
	}
	return names
}

func (c *SysctlCollector) RefreshNow() error {
	return c.refreshSysctlValues()
}

package metrics

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/hcl"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type FileFormat int

const (
	// TypeHCL describes the HCL (Hashicorp configuration language) file format
	TypeHCL FileFormat = iota
	// TypeYAML describes the YAML file format
	TypeYAML
)

// StartupFlags is a struct containing options that can be passed via the
// command line
type StartupFlags struct {
	ConfigFile                 string
	Filenames                  []string
	Parser                     string
	Format                     string
	Namespace                  string
	ListenAddress              string
	ListenPort                 int
	EnableExperimentalFeatures bool
	MetricsEndpoint            string
	VerifyConfig               bool
	Version                    bool

	LogLevel  string
	LogFormat string

	CPUProfile string
	MemProfile string

	MaxWorkers               int
	BufferSize               int
	EnableTLS                bool
	TLSCertFile              string
	TLSKeyFile               string
	EnableAuth               bool
	AuthToken                string
	EnableRateLimiting       bool
	RateLimit                int
	EnableMetricsCache       bool
	MetricsCacheTTL          time.Duration
	EnableHealthCheck        bool
	HealthCheckInterval      time.Duration
	EnablePprof              bool
	PprofPort                int
	EnableFileRotation       bool
	RotationInterval         time.Duration
	EnableSyslog             bool
	SyslogAddress            string
	EnableDebugEndpoints     bool
	DebugPort                int
	EnableRequestLogging     bool
	RequestLoggingLevel      string
	EnableGzip               bool
	GzipLevel                int
	EnablePrometheus         bool
	PrometheusPath           string
	EnableStatsD             bool
	StatsDAddress            string
	StatsDPrefix             string
	EnableTracing            bool
	TracingServiceName       string
	TracingCollectorEndpoint string
	TracingSamplingRate      float64
}

// Config models the application's configuration
type Config struct {
	Listen                     ListenConfig
	Consul                     *ConsulConfig
	Namespaces                 []NamespaceConfig `hcl:"namespace"`
	EnableExperimentalFeatures bool              `hcl:"enable_experimental" yaml:"enable_experimental"`

	// In YAML, the EnableExperimentalFeatures property was originally set by the
	// "enableexperimentalfeatures" property (although documented as "enable_experimental").
	// This property is here for enabling the config to behave as documented, while keeping BC.
	EnableExperimentalFeaturesOld bool `yaml:"enableexperimentalfeatures"`
}

// ListenConfig is a struct describing the built-in webserver configuration
type ListenConfig struct {
	Port            int
	Address         string
	MetricsEndpoint string `hcl:"metrics_endpoint" yaml:"metrics_endpoint"`
}

// ConsulConfig describes the connection to a Consul server that the exporter should
// register itself at
type ConsulConfig struct {
	Enable     bool
	Address    string
	Datacenter string
	Scheme     string
	Token      string
	Service    *ConsulServiceConfig
}

// ConsulServiceConfig describes the Consul service that the exporter should use
type ConsulServiceConfig struct {
	ID      string
	Name    string
	Address string
	Tags    []string
}

// StabilityWarnings tests if the Config or any of its sub-objects uses any
// configuration settings that are not yet declared "stable"
func (c *Config) StabilityWarnings() error {
	if c.EnableExperimentalFeatures || c.EnableExperimentalFeaturesOld {
		return nil
	}

	for i := range c.Namespaces {
		if err := c.Namespaces[i].StabilityWarnings(); err != nil {
			return err
		}
	}

	return nil
}

// MetricsEndpointOrDefault returns the configured metrics endpoint or the
// default value if no configuration was provided.
func (l *ListenConfig) MetricsEndpointOrDefault() string {
	if l.MetricsEndpoint == "" {
		return "/metrics"
	}

	return l.MetricsEndpoint
}

// RelabelConfig is a struct describing a single re-labeling configuration for taking
// over label values from an access log line into a Prometheus metric
type RelabelConfig struct {
	TargetLabel string              `hcl:",key" yaml:"target_label"`
	SourceValue string              `hcl:"from" yaml:"from"`
	Whitelist   []string            `hcl:"whitelist"`
	Matches     []RelabelValueMatch `hcl:"match"`
	Split       int                 `hcl:"split"`
	Separator   string              `hcl:"separator"`
	OnlyCounter bool                `hcl:"only_counter" yaml:"only_counter"`

	WhitelistExists bool
	WhitelistMap    map[string]interface{}
}

// RelabelValueMatch describes a single label match statement
type RelabelValueMatch struct {
	RegexpString string `hcl:",key" yaml:"regexp"`
	Replacement  string `hcl:"replacement"`

	CompiledRegexp *regexp.Regexp
}

// Compile compiles expressions and lookup tables for efficient later use
func (c *RelabelConfig) Compile() error {
	c.WhitelistMap = make(map[string]interface{})
	c.WhitelistExists = len(c.Whitelist) > 0

	for i := range c.Whitelist {
		c.WhitelistMap[c.Whitelist[i]] = nil
	}

	for i := range c.Matches {
		if c.Matches[i].RegexpString != "" {
			r, err := regexp.Compile(c.Matches[i].RegexpString)
			if err != nil {
				return fmt.Errorf("could not compile regexp '%s': %s", c.Matches[i].RegexpString, err.Error())
			}

			c.Matches[i].CompiledRegexp = r
		}
	}

	return nil
}

// NamespaceConfig is a struct describing single metric namespaces
type NamespaceConfig struct {
	Name string `hcl:",key"`

	NamespaceLabelName string `hcl:"namespace_label" yaml:"namespace_label"`
	NamespaceLabels    map[string]string

	MetricsOverride *struct {
		Prefix string `hcl:"prefix" yaml:"prefix"`
	} `hcl:"metrics_override" yaml:"metrics_override"`
	NamespacePrefix string

	SourceFiles      []string          `hcl:"source_files" yaml:"source_files"`
	SourceData       SourceData        `hcl:"source" yaml:"source"`
	Parser           string            `hcl:"parser" yaml:"parser"`
	Format           string            `hcl:"format" yaml:"format"`
	Labels           map[string]string `hcl:"labels" yaml:"labels"`
	RelabelConfigs   []RelabelConfig   `hcl:"relabel" yaml:"relabel_configs"`
	HistogramBuckets []float64         `hcl:"histogram_buckets" yaml:"histogram_buckets"`

	PrintLog bool `hcl:"print_log" yaml:"print_log"`

	OrderedLabelNames  []string
	OrderedLabelValues []string
}

type SourceData struct {
	Files  FileSource    `hcl:"files" yaml:"files"`
	Syslog *SyslogSource `hcl:"syslog" yaml:"syslog"`
}

type FileSource []string

type SyslogSource struct {
	ListenAddress string   `hcl:"listen_address" yaml:"listen_address"`
	Format        string   `hcl:"format" yaml:"format"`
	Tags          []string `hcl:"tags" yaml:"tags"`
}

// StabilityWarnings tests if the NamespaceConfig uses any configuration settings
// that are not yet declared "stable"
func (c *NamespaceConfig) StabilityWarnings() error {
	return nil
}

// DeprecationWarnings tests if the NamespaceConfig uses any deprecated
// configuration settings
func (c *NamespaceConfig) DeprecationWarnings() error {
	if len(c.SourceFiles) > 0 {
		return errors.New("you are using the 'source_files' configuration parameter")
	}

	return nil
}

// MustCompile compiles the configuration (mostly regular expressions that are used
// in configuration variables) for later use
func (c *NamespaceConfig) MustCompile() {
	err := c.Compile()
	if err != nil {
		panic(err)
	}
}

// ResolveDeprecations converts any values from depreated fields into the new
// structures
func (c *NamespaceConfig) ResolveDeprecations() {
	if len(c.SourceFiles) > 0 {
		c.SourceData.Files = FileSource(c.SourceFiles)
	}
}

// ResolveGlobs finds globs in file sources and expand them to the actual
// list of files
func (c *NamespaceConfig) ResolveGlobs(logger *Logger) error {
	if len(c.SourceData.Files) > 0 {
		resolvedFiles := make([]string, 0)
		for _, sf := range c.SourceData.Files {
			if strings.Contains(sf, "*") {
				matches, err := filepath.Glob(sf)
				if err != nil {
					return err
				}
				logger.Infof("Resolved globs %v to %v", sf, matches)
				resolvedFiles = append(resolvedFiles, matches...)
			} else {
				logger.Warnf("No globs for %v", sf)
				resolvedFiles = append(resolvedFiles, sf)
			}
		}

		// update fields with new list of files
		c.SourceData.Files = resolvedFiles
		c.SourceFiles = resolvedFiles
	}
	return nil
}

// Compile compiles the configuration (mostly regular expressions that are used
// in configuration variables) for later use
func (c *NamespaceConfig) Compile() error {
	for i := range c.RelabelConfigs {
		if err := c.RelabelConfigs[i].Compile(); err != nil {
			return err
		}
	}
	if c.NamespaceLabelName != "" {
		c.NamespaceLabels = make(map[string]string)
		c.NamespaceLabels[c.NamespaceLabelName] = c.Name
	}

	c.OrderLabels()
	c.NamespacePrefix = c.Name
	if c.MetricsOverride != nil {
		c.NamespacePrefix = c.MetricsOverride.Prefix
	}

	return nil
}

// OrderLabels builds two lists of label keys and values, ordered by label name
func (c *NamespaceConfig) OrderLabels() {
	keys := make([]string, 0, len(c.Labels))
	values := make([]string, len(c.Labels))

	for k := range c.Labels {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for i, k := range keys {
		values[i] = c.Labels[k]
	}

	c.OrderedLabelNames = keys
	c.OrderedLabelValues = values
}

func (f FileFormat) String() string {
	switch f {
	case TypeHCL:
		return "HCL"
	case TypeYAML:
		return "YAML"
	default:
		return fmt.Sprintf("UnknownFormat(%d)", f)
	}
}

// LoadConfigFile fills a configuration object (passed as parameter) with
// values read from a configuration file (pass as parameter by filename). The
// configuration file needs to be in HCL format.
func LoadConfigFile(logger *Logger, config *Config, filename string) error {
	cleanFilename := filepath.Clean(filename)
	var typ FileFormat
	if strings.EqualFold(filepath.Ext(cleanFilename), ".hcl") {
		typ = TypeHCL
	} else if strings.EqualFold(filepath.Ext(cleanFilename), ".yaml") || strings.EqualFold(filepath.Ext(cleanFilename), ".yml") {
		typ = TypeYAML
	} else {
		return fmt.Errorf("config file '%s' has unsupported file type", cleanFilename)
	}
	reader, err := os.Open(cleanFilename)
	if err != nil {
		return err
	}

	defer reader.Close()

	if strings.HasSuffix(filename, ".hcl") {
		typ = TypeHCL
	} else if strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml") {
		typ = TypeYAML
	} else {
		return fmt.Errorf("config file '%s' has unsupported file type", filename)
	}

	return LoadConfigStream(logger, config, reader, typ)
}

// LoadConfigStream fills a configuration object (passed as parameter) with
// values read from a Reader interface (passed as parameter).
func LoadConfigStream(logger *Logger, config *Config, stream io.Reader, typ FileFormat) error {

	if logger == nil {
		return fmt.Errorf("logger cannot be nil")
	}
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	if stream == nil {
		return fmt.Errorf("stream reader cannot be nil")
	}

	logger.Debug("Starting configuration loading process",
		zap.String("file_format", typ.String()),
	)

	var loadErr error
	switch typ {
	case TypeHCL:
		logger.Debug("Loading HCL configuration")
		loadErr = LoadConfigHCL(config, stream)
		if loadErr != nil {
			logger.Error("Failed to load HCL configuration",
				zap.Error(loadErr),
			)
			return fmt.Errorf("HCL configuration load failed: %w", loadErr)
		}

	case TypeYAML:
		logger.Debug("Loading YAML configuration")
		loadErr = LoadConfigYAML(config, stream)
		if loadErr != nil {
			logger.Error("Failed to load YAML configuration",
				zap.Error(loadErr),
			)
			return fmt.Errorf("YAML configuration load failed: %w", loadErr)
		}

	default:
		errMsg := fmt.Sprintf("Unsupported configuration type: %d", typ)
		logger.Error(errMsg)
		return fmt.Errorf(errMsg)
	}

	if err := validateConfigStructure(config); err != nil {
		logger.Error("Configuration structure validation failed",
			zap.Error(err),
		)
		return fmt.Errorf("configuration structure invalid: %w", err)
	}

	logger.Debug("Processing configuration namespaces",
		zap.Int("namespace_count", len(config.Namespaces)),
	)

	for i := range config.Namespaces {
		ns := &config.Namespaces[i]

		logger.Debug("Processing namespace",
			zap.String("namespace", ns.Name),
		)

		logger.Debug("Resolving deprecations for namespace",
			zap.String("namespace", ns.Name),
		)
		ns.ResolveDeprecations()

		logger.Debug("Resolving glob patterns for namespace",
			zap.String("namespace", ns.Name),
		)
		if err := ns.ResolveGlobs(logger); err != nil {
			logger.Error("Failed to resolve glob patterns",
				zap.String("namespace", ns.Name),
				zap.Error(err),
			)
			return fmt.Errorf("failed to resolve globs for namespace %s: %w", ns.Name, err)
		}

		if err := validateNamespace(ns); err != nil {
			logger.Error("Namespace validation failed",
				zap.String("namespace", ns.Name),
				zap.Error(err),
			)
			return fmt.Errorf("namespace %s validation failed: %w", ns.Name, err)
		}
	}

	logger.Info("Configuration loaded successfully",
		zap.String("file_format", typ.String()),
		zap.Int("namespace_count", len(config.Namespaces)),
	)

	return nil
}

func validateConfigStructure(config *Config) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}
	return nil
}

func validateNamespace(ns *NamespaceConfig) error {
	if ns == nil {
		return fmt.Errorf("namespace is nil")
	}
	if ns.Name == "" {
		return fmt.Errorf("namespace name cannot be empty")
	}

	return nil
}

func LoadConfigFlagsInfo(config *Config, flags *StartupFlags) error {

	config.Listen = ListenConfig{
		Port:            flags.ListenPort,
		Address:         flags.ListenAddress,
		MetricsEndpoint: flags.MetricsEndpoint,
	}

	config.Namespaces = []NamespaceConfig{
		{
			Format: flags.Format,
			Parser: flags.Parser,
			Name:   flags.Namespace,
			SourceData: SourceData{
				Files: flags.Filenames,
			},
		},
	}

	return nil
}

func LoadConfigHCL(config *Config, file io.Reader) error {

	if config == nil {
		return fmt.Errorf("config parameter cannot be nil")
	}
	if file == nil {
		return fmt.Errorf("file reader cannot be nil")
	}

	buf, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if len(buf) == 0 {
		return fmt.Errorf("config file is empty")
	}

	hclText := string(buf)
	if hclText == "" {
		return fmt.Errorf("failed to convert file content to string")
	}

	if !isValidHCL(hclText) {
		return fmt.Errorf("invalid HCL syntax in config file")
	}

	if err := hcl.Decode(config, hclText); err != nil {
		return fmt.Errorf("failed to decode HCL config: %w", err)
	}

	if err := validateConfig(config); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	return nil
}

func isValidHCL(hclText string) bool {
	return strings.Contains(hclText, "=") ||
		strings.Contains(hclText, "{") ||
		strings.Contains(hclText, "}")
}

func validateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}
	return nil
}

func LoadConfigYAML(config *Config, file io.Reader) error {

	if config == nil {
		return fmt.Errorf("config parameter cannot be nil")
	}
	if file == nil {
		return fmt.Errorf("file reader cannot be nil")
	}

	buf, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read YAML file: %w", err)
	}

	if len(buf) == 0 {
		return fmt.Errorf("YAML file is empty")
	}

	if err := validateYAMLFormat(buf); err != nil {
		return fmt.Errorf("invalid YAML format: %w", err)
	}

	tempConfig := &Config{}

	if err := yaml.Unmarshal(buf, tempConfig); err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	if err := validateConfigContent(tempConfig); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	*config = *tempConfig

	logDebug("Successfully loaded YAML config")

	return nil
}

func validateYAMLFormat(data []byte) error {
	if !bytes.HasPrefix(bytes.TrimSpace(data), []byte("---")) {
		return fmt.Errorf("missing YAML document start marker")
	}
	return nil
}

func validateConfigContent(config *Config) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}
	return nil
}

func logDebug(message string) {
	if os.Getenv("DEBUG") == "true" {
		_ = fmt.Errorf("[DEBUG]: %s", message)
	}
}
// Part 2 commit for prometheus_nginxlog_exporter/internal/metrics/configloader.go

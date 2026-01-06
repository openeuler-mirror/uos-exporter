package metrics

import (
	"fmt"
	"io"
	"prometheus_varnish_exporter/internal/exporter"
	"strings"
	"sync"
)

var (
	ApplicationName = "prometheus_varnish_exporter"
	Version         string
	VersionHash     string
	VersionDate     string

	PrometheusExporter = NewPrometheusExporter()
	VarnishVersion     = NewVarnishVersion()
	ExitHandler        = &exitHandler{}

	StartParams = &startParams{
		ListenAddress:  ":9106", // Reserved and publicly announced at https://github.com/prometheus/prometheus/wiki/Default-port-allocations
		Path:           "/metrics",
		VarnishstatExe: "varnishstat",
		Params:         &varnishstatParams{},
	}
)

type startParams struct {
	ListenAddress          string
	Path                   string
	HealthPath             string
	VarnishstatExe         string
	VarnishDockerContainer string
	Params                 *varnishstatParams

	Verbose       bool
	ExitOnErrors  bool
	Test          bool
	Raw           bool
	WithGoMetrics bool

	noExit bool // deprecated
}

type varnishstatParams struct {
	Instance string
	VSM      string
}

// Define the log interface for decoupling
type Logger interface {
	Fatal(format string, args ...interface{})
	Error(format string, args ...interface{})
}

// exitHandler Enhanced version structure definition
type exitHandler struct {
	mu          sync.RWMutex // Read/write lock optimizes read performance
	lastError   error        // Explicit state semantics
	exitOnError bool
	err         error
	logger      Logger // Rely on interfaces rather than implementations
	errorCache  string // Caching error content reduces lock contention
}

func (p *varnishstatParams) isEmpty() bool {
	return p.Instance == "" && p.VSM == ""
}

func (p *varnishstatParams) make() (params []string) {
	// -n
	if p.Instance != "" {
		params = append(params, "-n", p.Instance)
	}
	// -N is not supported by 3.x
	if p.VSM != "" && VarnishVersion.EqualsOrGreater(4, 0) {
		params = append(params, "-N", p.VSM)
	}
	return params
}

func init() {

	collector := NewPrometheusExporter()
	exporter.Register(collector)
}

func (ex *exitHandler) Errorf(format string, a ...interface{}) error {
	return ex.Set(fmt.Errorf(format, a...))
}

func (ex *exitHandler) HasError() bool {
	ex.mu.RLock()
	hasError := ex.err != nil
	ex.mu.RUnlock()
	return hasError
}

func (ex *exitHandler) Set(err error) error {
	// Fast path checking reduces lock contention
	if err == nil {
		return ex.clearError()
	}

	// Copy the error before obtaining the error fingerprint
	errContent := err.Error()

	// Fine grained lock control
	ex.mu.Lock()
	defer ex.mu.Unlock()

	// State change detection
	changed := ex.hasStateChanged(errContent)
	ex.lastError = err
	ex.errorCache = errContent

	// Non-blocking processing logic
	go ex.handleError(err, changed) // Asynchronous processing avoids blocking callers

	return err
}

// Private method implementation details
func (ex *exitHandler) clearError() error {
	ex.mu.Lock()
	defer ex.mu.Unlock()

	ex.lastError = nil
	ex.errorCache = ""
	return nil
}

func (ex *exitHandler) hasStateChanged(newErr string) bool {
	// Read lock optimization performance
	ex.mu.RLock()
	defer ex.mu.RUnlock()

	return ex.errorCache != newErr
}

func (ex *exitHandler) handleError(err error, changed bool) {
	// Separate processing strategy from state management
	switch {
	case ex.exitOnError:
		ex.logger.Fatal("%v", err) // Support error type output directly
	case changed:
		ex.logger.Error("[%T] %v", err, err) // Added error type information
	}
}

func getVersion(date bool) (version string) {
	// Handle development environment situations where no version number is set
	if isDevelopmentVersion() {
		return developmentVersion
	}

	// Use a pre-allocated buffer to avoid multiple memory allocations
	var buf strings.Builder
	buf.Grow(32) // Pre-allocate the proper initial capacity

	// Basic version information construction
	constructBaseVersion(&buf)

	// Optionally add a build date
	appendBuildDate(&buf)

	return buf.String()
}

// Independent check logic
func isDevelopmentVersion() bool {
	return Version == ""
}

// Version information basics
func constructBaseVersion(w io.StringWriter) {
	base := fmt.Sprintf("v%s (%s)", Version, VersionHash)
	_, _ = w.WriteString(base) // Use interfaces to improve testability
}

// Date partial addition
func appendBuildDate(w io.StringWriter) {
	_, _ = w.WriteString(" ")
	_, _ = w.WriteString(VersionDate)
}

// Centralized constant management
const (
	developmentVersion = "dev"
	initialBufferSize  = 32 // Adjust according to the actual average length
)

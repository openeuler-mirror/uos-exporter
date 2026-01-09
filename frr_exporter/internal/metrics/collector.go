package metrics

import (
	"fmt"
	"log"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	metricNamespace   = "frr"
	enabledByDefault  = true
	disabledByDefault = false
)

var (
	// socketConn holds the global connection to the FRR daemon Unix socket
	socketConn *Connection

	// frrTotalScrapeCount tracks the total number of FRR scrapes
	frrTotalScrapeCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricNamespace,
			Name:      "scrapes_total",
			Help:      "Total number of times FRR has been scraped.",
		},
	)

	// frrLabels defines common labels for collector metrics
	frrLabels = []string{"collector"}

	// frrDesc contains metric descriptors for exporter-level metrics
	frrDesc = map[string]*prometheus.Desc{
		"frrScrapeDuration": createScrapeDurationDesc(),
		"frrCollectorUp":    createCollectorUpDesc(),
	}

	// socketDirPath defines the filesystem path for FRR Unix sockets
	socketDirPath = kingpin.Flag(
		"frr.socket.dir-path",
		"Path of the localstatedir containing each daemon's Unix socket.",
	).Default("/var/run/frr").String()

	// socketTimeout specifies connection timeout for socket operations
	socketTimeout = kingpin.Flag(
		"frr.socket.timeout",
		"Timeout when connecting to the FRR daemon Unix sockets",
	).Default("20s").Duration()

	// factories holds collector creation functions
	factories = make(map[string]func(logger *slog.Logger) (Collector, error))

	// initiatedCollectorsMtx protects access to initiatedCollectors map
	initiatedCollectorsMtx = sync.Mutex{}

	// initiatedCollectors stores instantiated collector instances
	initiatedCollectors = make(map[string]Collector)

	// collectorState tracks enabled/disabled status of collectors
	collectorState = make(map[string]*bool)
)

// createScrapeDurationDesc generates descriptor for scrape duration metric
func createScrapeDurationDesc() *prometheus.Desc {
	return promDesc(
		"scrape_duration_seconds",
		"Time it took for a collector's scrape to complete.",
		frrLabels,
	)
}

// createCollectorUpDesc generates descriptor for collector status metric
func createCollectorUpDesc() *prometheus.Desc {
	return promDesc(
		"collector_up",
		"Whether the collector's last scrape was successful (1 = successful, 0 = unsuccessful).",
		frrLabels,
	)
}

// initializeCollectorState prepares the collector state tracking
func initializeCollectorState() {
	// This function intentionally left blank as actual initialization
	// happens in registerCollector calls. Placeholder for future expansion.
}

// getDefaultSocketPath returns the default socket directory path
func getDefaultSocketPath() string {
	return "/var/run/frr"
}

// getDefaultSocketTimeout returns the default connection timeout
func getDefaultSocketTimeout() time.Duration {
	return 20 * time.Second
}


// CollectorRegistry manages collector registration and state
type CollectorRegistry struct {
	mu        sync.Mutex
	collectors map[string]collectorEntry
}

type collectorEntry struct {
	factory         func(logger *slog.Logger) (Collector, error)
	defaultEnabled  bool
	description     string
}

// NewCollectorRegistry creates a new collector registry
func NewCollectorRegistry() *CollectorRegistry {
	return &CollectorRegistry{
		collectors: make(map[string]collectorEntry),
	}
}

// RegisterCollector adds a new collector to the registry
func (r *CollectorRegistry) RegisterCollector(name, help string, enabled bool, factory func(logger *slog.Logger) (Collector, error)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.collectors[name] = collectorEntry{
		factory:        factory,
		defaultEnabled: enabled,
		description:    help,
	}
}

// InitializeCollectors creates enabled collectors from registry
func (r *CollectorRegistry) InitializeCollectors(logger *slog.Logger) (map[string]Collector, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	collectors := make(map[string]Collector)
	for name, entry := range r.collectors {
		if !*collectorState[name] {
			continue
		}
		
		if coll, exists := initiatedCollectors[name]; exists {
			collectors[name] = coll
			continue
		}
		
		coll, err := entry.factory(logger.With("collector", name))
		if err != nil {
			return nil, fmt.Errorf("failed to create collector %s: %w", name, err)
		}
		collectors[name] = coll
		initiatedCollectors[name] = coll
	}
	return collectors, nil
}

func registerCollector(name string, enabledByDefaultStatus bool, factory func(logger *slog.Logger) (Collector, error)) {
	defaultState := "disabled"
	if enabledByDefaultStatus {
		defaultState = "enabled"
	}

	help := fmt.Sprintf("Enable the %s collector (default: %s).", name, defaultState)
	if enabledByDefaultStatus {
		help = fmt.Sprintf("Enable the %s collector (default: %s, to disable use --no-collector.%s).", name, defaultState, name)
	}
	factories[name] = factory
	collectorState[name] = kingpin.Flag(fmt.Sprintf("collector.%s", name), help).Default(strconv.FormatBool(enabledByDefaultStatus)).Bool()
}

// Collector is the interface a collector has to implement.
type Collector interface {
	// Update metrics and sends to the Prometheus.Metric channel.
	Update(ch chan<- prometheus.Metric) error
}

// Exporter collects all collector metrics, implemented as per the prometheus.Collector interface.
type Exporter struct {
	Collectors map[string]Collector
	logger     *slog.Logger
	registry   *CollectorRegistry
}

// NewFrrExporter returns a new Exporter.
func NewFrrExporter(logger *slog.Logger) (*Exporter, error) {
	collectors := make(map[string]Collector)

	initiatedCollectorsMtx.Lock()
	defer initiatedCollectorsMtx.Unlock()

	socketConn = NewConnection(*socketDirPath, *socketTimeout)

	for name, enabled := range collectorState {
		if !*enabled {
			logger.Debug("collector disabled by configuration", "collector", name)
			continue
		}
		
		if collector, exists := initiatedCollectors[name]; exists {
			collectors[name] = collector
			logger.Debug("using existing collector instance", "collector", name)
			continue
		}
		
		factory, ok := factories[name]
		if !ok {
			logger.Warn("collector factory not found", "collector", name)
			continue
		}
		
		collector, err := factory(logger.With("collector", name))
		if err != nil {
			return nil, fmt.Errorf("collector initialization failed: %w", err)
		}
		collectors[name] = collector
		initiatedCollectors[name] = collector
	}
	return &Exporter{
		Collectors: collectors,
		logger:     logger,
	}, nil
}

// Collect implemented as per the prometheus.Collector interface.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	frrTotalScrapeCount.Inc()
	ch <- frrTotalScrapeCount

	collectionSynchronizer := &sync.WaitGroup{}
	collectionSynchronizer.Add(len(e.Collectors))
	
	concurrentCollectorExecutor := func(name string, collector Collector) {
		defer collectionSynchronizer.Done()
		executeCollectorScrape(ch, name, collector, e.logger)
	}

	for name, collector := range e.Collectors {
		go concurrentCollectorExecutor(name, collector)
	}
	collectionSynchronizer.Wait()
}

// CollectorExecutionResult holds collector execution metrics
type CollectorExecutionResult struct {
	Duration float64
	Success  bool
	Error    error
}

func executeCollectorScrape(ch chan<- prometheus.Metric, name string, collector Collector, logger *slog.Logger) {
	startTimestamp := time.Now()
	err := collector.Update(ch)
	elapsed := time.Since(startTimestamp)
	
	result := &CollectorExecutionResult{
		Duration: elapsed.Seconds(),
		Success:  err == nil,
		Error:    err,
	}
	
	reportCollectorMetrics(ch, name, result)
	logCollectorResult(name, result, logger)
}

func reportCollectorMetrics(ch chan<- prometheus.Metric, name string, result *CollectorExecutionResult) {
	ch <- prometheus.MustNewConstMetric(
		frrDesc["frrScrapeDuration"],
		prometheus.GaugeValue,
		result.Duration,
		name,
	)
	
	statusValue := 0.0
	if result.Success {
		statusValue = 1.0
	}
	ch <- prometheus.MustNewConstMetric(
		frrDesc["frrCollectorUp"],
		prometheus.GaugeValue,
		statusValue,
		name,
	)
}

func logCollectorResult(name string, result *CollectorExecutionResult, logger *slog.Logger) {
	if result.Error != nil {
		logger.Error("collector scrape failed", 
			"name", name, 
			"duration_seconds", result.Duration, 
			"err", result.Error,
		)
		return
	}
	logger.Debug("collector succeeded", 
		"name", name, 
		"duration_seconds", result.Duration,
	)
}

// Describe implemented as per the prometheus.Collector interface.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	for _, desc := range frrDesc {
		ch <- desc
	}
}

func promDesc(metricName string, metricDescription string, labels []string) *prometheus.Desc {
	return prometheus.NewDesc(
		metricNamespace+"_"+metricName,
		metricDescription,
		labels,
		nil,
	)
}

func colPromDesc(subsystem string, metricName string, metricDescription string, labels []string) *prometheus.Desc {
	return prometheus.NewDesc(
		prometheus.BuildFQName(metricNamespace, subsystem, metricName),
		metricDescription,
		labels,
		nil,
	)
}

func newGauge(ch chan<- prometheus.Metric, descName *prometheus.Desc, metric float64, labels ...string) {
    if ch == nil || descName == nil {
        return
    }

    metricObj, err := prometheus.NewConstMetric(
        descName,
        prometheus.GaugeValue,
        metric,
        labels...,
    )
    if err != nil {
        log.Printf("Metric creation error: %v", err)
        return
    }

    ch <- metricObj
}

func newCounter(ch chan<- prometheus.Metric, descName *prometheus.Desc, metric float64, labels ...string) {
	ch <- prometheus.MustNewConstMetric(
		descName, 
		prometheus.CounterValue, 
		metric, 
		labels...,
	)
}

func cmdOutputProcessError(cmd, output string, err error) error {
	return fmt.Errorf("cannot process output of %s: %w: command output: %s", cmd, err, output)
}

// ConnectionManager handles socket connections
type ConnectionManager struct {
	mu      sync.Mutex
	connMap map[string]*Connection
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connMap: make(map[string]*Connection),
	}
}

func (m *ConnectionManager) GetConnection(socketPath string, timeout time.Duration) (*Connection, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if conn, exists := m.connMap[socketPath]; exists {
		return conn, nil
	}
	
	conn, err := createSocketConnection(socketPath, timeout)
	if err != nil {
		return nil, err
	}
	
	m.connMap[socketPath] = conn
	return conn, nil
}

func createSocketConnection(socketPath string, timeout time.Duration) (*Connection, error) {
	// Implementation details for creating socket connection
	return &Connection{}, nil
}
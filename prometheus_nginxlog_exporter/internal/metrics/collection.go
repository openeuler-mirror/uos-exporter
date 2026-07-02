package metrics

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/nxadm/tail"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/satyrius/gonx"
	"gopkg.in/mcuadros/go-syslog.v2"
)

// Parser parses a line of log to a map[string]string.
type Parser interface {
	ParseString(line string) (map[string]string, error)
}

// JsonParser parse a JSON string.
type JsonParser struct{}

// TextParser parses variables patterns using config.NamespaceConfig.Format.
type TextParser struct {
	parser *gonx.Parser
}

// Relabeling contains a relabeling configuration and is responsible for
// executing the rules specified in the original configuration
type Relabeling struct {
	RelabelConfig
}

// DefaultRelabelings are hardcoded relabeling configs that are always there
// and do not need to be explicitly configured
var DefaultRelabelings = []*Relabeling{
	{
		RelabelConfig{
			TargetLabel: "method",
			SourceValue: "request",
			Split:       1,

			WhitelistExists: true,
			WhitelistMap: map[string]interface{}{
				"GET":     nil,
				"HEAD":    nil,
				"POST":    nil,
				"PUT":     nil,
				"DELETE":  nil,
				"CONNECT": nil,
				"OPTIONS": nil,
				"TRACE":   nil,
				"PATCH":   nil,
			},
		},
	},
	{
		RelabelConfig{
			TargetLabel: "status",
			SourceValue: "status",
		},
	},
}

type Follower interface {
	Lines() chan string
	OnError(func(error))
}

type followerImpl struct {
	logger *Logger

	filename string
	t        *tail.Tail
	line     chan string
}

type syslogFollower struct {
	tag  string
	line chan string

	channel syslog.LogPartsChannel
	server  *syslog.Server
}

// Collection is a struct containing pointers to all metrics that should be
// exposed to Prometheus
type Collection struct {
	CountTotal                 *prometheus.CounterVec
	ResponseBytesTotal         *prometheus.CounterVec
	RequestBytesTotal          *prometheus.CounterVec
	UpstreamSeconds            *prometheus.SummaryVec
	UpstreamSecondsHist        *prometheus.HistogramVec
	UpstreamConnectSeconds     *prometheus.SummaryVec
	UpstreamConnectSecondsHist *prometheus.HistogramVec
	ResponseSeconds            *prometheus.SummaryVec
	ResponseSecondsHist        *prometheus.HistogramVec
	ParseErrorsTotal           prometheus.Counter
	Metrics                    struct {
		ProcessedCount uint64
	}
	LastError error
}

type NamespaceMetrics struct {
	cfg      *NamespaceConfig
	registry *prometheus.Registry

	Collection
}

const (
	dedupStrategyFirstWin = "first_occurrence"
)

type DeduplicationConfig struct {
	Strategy      string
	CaseSensitive bool
}

var defaultDeduplicationConfig = &DeduplicationConfig{
	Strategy:      dedupStrategyFirstWin,
	CaseSensitive: true,
}

type processContext struct {
	config     *DeduplicationConfig
	seenLabels labelRegistry
	result     []*Relabeling
}

type labelRegistry map[string]struct{}

func (m *NamespaceMetrics) Describe(ch chan<- *prometheus.Desc) {
	m.CountTotal.Describe(ch)
	m.ResponseBytesTotal.Describe(ch)
	m.RequestBytesTotal.Describe(ch)
	m.UpstreamSeconds.Describe(ch)
	m.UpstreamSecondsHist.Describe(ch)
	m.UpstreamConnectSeconds.Describe(ch)
	m.UpstreamConnectSecondsHist.Describe(ch)
	m.ResponseSeconds.Describe(ch)
	m.ResponseSecondsHist.Describe(ch)
}

func (m *NamespaceMetrics) Collect(ch chan<- prometheus.Metric) {
	m.CountTotal.Collect(ch)
	m.ResponseBytesTotal.Collect(ch)
	m.RequestBytesTotal.Collect(ch)
	m.UpstreamSeconds.Collect(ch)
	m.UpstreamSecondsHist.Collect(ch)
	m.UpstreamConnectSeconds.Collect(ch)
	m.UpstreamConnectSecondsHist.Collect(ch)
	m.ResponseSeconds.Collect(ch)
	m.ResponseSecondsHist.Collect(ch)
	m.ParseErrorsTotal.Collect(ch)
}

func NewForNamespace(cfg *NamespaceConfig) *NamespaceMetrics {
	m := &NamespaceMetrics{
		cfg:      cfg,
		registry: prometheus.NewRegistry(),
	}
	m.Init(cfg)
	m.MustRegister(m.registry)

	return m
}

func (m *NamespaceMetrics) Gatherer() prometheus.Gatherer {
	return m.registry
}

func (c *Collection) MustRegister(r *prometheus.Registry) {
	r.MustRegister(c.CountTotal)
	r.MustRegister(c.RequestBytesTotal)
	r.MustRegister(c.ResponseBytesTotal)
	r.MustRegister(c.UpstreamSeconds)
	r.MustRegister(c.UpstreamSecondsHist)
	r.MustRegister(c.UpstreamConnectSeconds)
	r.MustRegister(c.UpstreamConnectSecondsHist)
	r.MustRegister(c.ResponseSeconds)
	r.MustRegister(c.ResponseSecondsHist)
	r.MustRegister(c.ParseErrorsTotal)
}

// Init initializes a metrics struct
func (m *Collection) Init(cfg *NamespaceConfig) {
	cfg.MustCompile()

	labels := cfg.OrderedLabelNames
	counterLabels := labels

	relabelings := NewRelabelings(cfg.RelabelConfigs)
	relabelings = append(relabelings, DefaultRelabelings...)
	relabelings = UniqueRelabelings(relabelings)

	for _, r := range relabelings {
		if !r.OnlyCounter {
			labels = append(labels, r.TargetLabel)
		}
		counterLabels = append(counterLabels, r.TargetLabel)
	}

	m.CountTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace:   cfg.NamespacePrefix,
		ConstLabels: cfg.NamespaceLabels,
		Name:        "http_response_count_total",
		Help:        "Amount of processed HTTP requests",
	}, counterLabels)

	m.ResponseBytesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace:   cfg.NamespacePrefix,
		ConstLabels: cfg.NamespaceLabels,
		Name:        "http_response_size_bytes",
		Help:        "Total amount of transferred bytes",
	}, labels)

	m.RequestBytesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace:   cfg.NamespacePrefix,
		ConstLabels: cfg.NamespaceLabels,
		Name:        "http_request_size_bytes",
		Help:        "Total amount of received bytes",
	}, labels)

	m.UpstreamSeconds = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:   cfg.NamespacePrefix,
		ConstLabels: cfg.NamespaceLabels,
		Name:        "http_upstream_time_seconds",
		Help:        "Time needed by upstream servers to handle requests",
		Objectives:  map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, labels)

	m.UpstreamSecondsHist = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace:   cfg.NamespacePrefix,
		ConstLabels: cfg.NamespaceLabels,
		Name:        "http_upstream_time_seconds_hist",
		Help:        "Time needed by upstream servers to handle requests",
		Buckets:     cfg.HistogramBuckets,
	}, labels)

	m.UpstreamConnectSeconds = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:   cfg.NamespacePrefix,
		ConstLabels: cfg.NamespaceLabels,
		Name:        "http_upstream_connect_time_seconds",
		Help:        "Time needed to connect to upstream servers",
		Objectives:  map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, labels)

	m.UpstreamConnectSecondsHist = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace:   cfg.NamespacePrefix,
		ConstLabels: cfg.NamespaceLabels,
		Name:        "http_upstream_connect_time_seconds_hist",
		Help:        "Time needed to connect to upstream servers",
		Buckets:     cfg.HistogramBuckets,
	}, labels)

	m.ResponseSeconds = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:   cfg.NamespacePrefix,
		ConstLabels: cfg.NamespaceLabels,
		Name:        "http_response_time_seconds",
		Help:        "Time needed by NGINX to handle requests",
		Objectives:  map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, labels)

	m.ResponseSecondsHist = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace:   cfg.NamespacePrefix,
		ConstLabels: cfg.NamespaceLabels,
		Name:        "http_response_time_seconds_hist",
		Help:        "Time needed by NGINX to handle requests",
		Buckets:     cfg.HistogramBuckets,
	}, labels)

	m.ParseErrorsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace:   cfg.NamespacePrefix,
		ConstLabels: cfg.NamespaceLabels,
		Name:        "parse_errors_total",
		Help:        "Total number of log file lines that could not be parsed",
	})
}

// // SetupCPUProfiling starts CPU profiling if an outputFile is specified
// func SetupCPUProfiling(outputFile string, stopChan <-chan bool, stopHandlers *sync.WaitGroup) {
// 	if outputFile == "" {
// 		return
// 	}

// 	f, err := os.Create(outputFile)
// 	if err != nil {
// 		panic(err)
// 	}

// 	fmt.Printf("writing CPU profile to file %s\n", outputFile)

// 	if err := pprof.StartCPUProfile(f); err != nil {
// 		panic(err)
// 	}

// 	stopHandlers.Add(1)

// 	go func() {
// 		<-stopChan

// 		fmt.Printf("stopping CPU profiling...\n")
// 		pprof.StopCPUProfile()

// 		stopHandlers.Done()
// 	}()
// }

// SetupMemoryProfiling starts memory profiling if an outputFile is specified
// func SetupMemoryProfiling(outputFile string, stopChan <-chan bool, stopHandlers *sync.WaitGroup) {
// 	if outputFile == "" {
// 		return
// 	}

// 	runtime.MemProfileRate = 1
// 	stopHandlers.Add(1)

// 	go func() {
// 		<-stopChan

// 		f, err := os.Create(outputFile)
// 		if err != nil {
// 			panic(err)
// 		}

// 		fmt.Printf("writing memory profile to file %s\n", outputFile)

// 		runtime.GC()
// 		if err := pprof.WriteHeapProfile(f); err != nil {
// 			panic(err)
// 		}

// 		f.Close()
// 		stopHandlers.Done()
// 	}()
// }

// NewTextParser returns a new text parser.
func NewTextParser(format string) *TextParser {
	return &TextParser{
		parser: gonx.NewParser(format),
	}
}

// ParseString implements the Parser interface.
func (t *TextParser) ParseString(line string) (map[string]string, error) {
	entry, err := t.parser.ParseString(line)
	if err != nil {
		return nil, fmt.Errorf("text log parsing err: %w", err)
	}

	return entry.Fields(), nil
}

// NewJsonParser returns a new json parser.
func NewJsonParser() *JsonParser {
	return &JsonParser{}
}

// ParseString implements the Parser interface.
// The value in the map is not necessarily a string, so it needs to be converted.
func (j *JsonParser) ParseString(line string) (map[string]string, error) {
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(line), &parsed)
	if err != nil {
		return nil, fmt.Errorf("json log parsing err: %w", err)
	}

	fields := make(map[string]string, len(parsed))
	for k, v := range parsed {
		if s, ok := v.(string); ok {
			fields[k] = s
		} else {
			fields[k] = fmt.Sprintf("%v", v)
		}
	}
	return fields, nil
}

// NewParser returns a Parser with the given config.NamespaceConfig.
func NewParser(nsCfg *NamespaceConfig) Parser {
	switch nsCfg.Parser {
	case "text":
		return NewTextParser(nsCfg.Format)
		// return textparser.NewTextParser(nsCfg.Format)
	case "json":
		return NewJsonParser()
		// return jsonparser.NewJsonParser()
	default:
		return NewTextParser(nsCfg.Format)
		// return textparser.NewTextParser(nsCfg.Format)
	}
}

// NewRelabelings creates a new set of relabelling runners from a list of
// configurations (which are typically read from the config file)
func NewRelabelings(cfgs []RelabelConfig) []*Relabeling {
	r := make([]*Relabeling, len(cfgs))

	for i := range cfgs {
		r[i] = NewRelabeling(&cfgs[i])
	}

	return r
}

// NewRelabeling creates a single new relabelling runner
func NewRelabeling(cfg *RelabelConfig) *Relabeling {
	return &Relabeling{*cfg}
}

func UniqueRelabelings(relabelings []*Relabeling) []*Relabeling {
	if len(relabelings) == 0 {
		return nil
	}

	ctx := &processContext{
		config:     defaultDeduplicationConfig,
		seenLabels: make(labelRegistry),
		result:     make([]*Relabeling, 0, len(relabelings)),
	}

	processRelabelings(relabelings, ctx)

	return optimizeResultSlice(ctx.result)
}

func processRelabelings(entries []*Relabeling, ctx *processContext) {
	for _, entry := range entries {
		processSingleEntry(entry, ctx)
	}
}

func processSingleEntry(entry *Relabeling, ctx *processContext) {
	if entry == nil {
		return
	}

	labelKey := normalizeLabelKey(entry.TargetLabel, ctx.config)

	if isLabelRegistered(labelKey, ctx) {
		return
	}

	registerLabel(labelKey, ctx)
	ctx.result = append(ctx.result, entry)
}

func normalizeLabelKey(key string, config *DeduplicationConfig) string {
	if !config.CaseSensitive {
		return strings.ToLower(key)
	}
	return key
}

func isLabelRegistered(key string, ctx *processContext) bool {
	_, exists := ctx.seenLabels[key]
	return exists
}

func registerLabel(key string, ctx *processContext) {
	ctx.seenLabels[key] = struct{}{}
}

func optimizeResultSlice(s []*Relabeling) []*Relabeling {
	if len(s) == cap(s) {
		return s
	}
	optimized := make([]*Relabeling, len(s))
	copy(optimized, s)
	return optimized
}

// StripOnlyCounterValues strips all values that are associated to relabelings only intended for the request counter
func StripOnlyCounterValues(values []string, relabelings []*Relabeling) []string {
	result := make([]string, 0, len(values))
	offset := len(values) - len(relabelings)
	for i := range values {
		if i >= offset && relabelings[i-offset].OnlyCounter {
			// skip if relabeling and only enabled for counter
			continue
		}
		result = append(result, values[i])
	}
	return result
}

// Map maps a sourceValue from the access log line according to the relabeling
// config (matching against whitelists, regular expressions etc.)
func (r *Relabeling) Map(sourceValue string) (string, error) {
	if r.Split > 0 {
		separator := r.Separator
		if separator == "" {
			separator = " "
		}

		values := strings.Split(sourceValue, separator)

		if len(values) >= r.Split {
			sourceValue = values[r.Split-1]
		} else {
			sourceValue = ""
		}
	}

	if r.WhitelistExists {
		if _, ok := r.WhitelistMap[sourceValue]; ok {
			return sourceValue, nil
		}

		return "other", nil
	}

	if len(r.Matches) > 0 {
		replacement := ""
		for i := range r.Matches {
			if r.Matches[i].CompiledRegexp.MatchString(sourceValue) {
				replacement = r.Matches[i].CompiledRegexp.ReplaceAllString(sourceValue, r.Matches[i].Replacement)
				break
			}
		}
		sourceValue = replacement
	}

	return sourceValue, nil
}

// NewSyslogFollower builds a new syslog follower from a previously constructed
// syslog server & channel
func NewSyslogFollower(tag string, server *syslog.Server, channel syslog.LogPartsChannel) (Follower, error) {
	s := &syslogFollower{
		tag:     tag,
		channel: channel,
		line:    make(chan string),
		server:  server,
	}
	return s, nil
}

func (s *syslogFollower) OnError(cb func(error)) {
	go func() {
		err := s.server.GetLastError()
		if err != nil {
			cb(err)
		}
	}()
}

func (s *syslogFollower) Lines() chan string {
	go func() {
		for line := range s.channel {
			if _, ok := line["tag"].(string); !ok {
				continue
			}

			if line["tag"].(string) == s.tag {
				s.line <- line["content"].(string)
			}
		}
	}()
	return s.line
}

// NewFileFollower creates a new Follower instance for a given file (given by name)
func NewFileFollower(logger *Logger, filename string) (Follower, error) {
	f := &followerImpl{
		filename: filename,
		line:     make(chan string),
		logger:   logger,
	}

	if err := f.start(); err != nil {
		return nil, err
	}

	return f, nil
}

func (f *followerImpl) start() error {
	var seekInfo *tail.SeekInfo

	_, err := os.Stat(f.filename)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else {
		seekInfo = &tail.SeekInfo{Offset: 0, Whence: io.SeekEnd}
	}

	t, err := tail.TailFile(f.filename, tail.Config{
		Follow:   true,
		ReOpen:   true,
		Poll:     true,
		Location: seekInfo,
		Logger:   f.logger,
	})

	if err != nil {
		return err
	}

	f.t = t
	return nil
}

func (f *followerImpl) OnError(cb func(error)) {
	go func() {
		err := f.t.Wait()
		if err != nil {
			cb(err)
		}
	}()
}

func (f *followerImpl) Lines() chan string {
	go func() {
		for n := range f.t.Lines {
			f.line <- n.Text
		}
	}()
	return f.line
}

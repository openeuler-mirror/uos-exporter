package metrics

import (
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	"fmt"
	"net/http"

	//_ "net/http/pprof"
	"net/url"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
)

func init() {
	// exporter.Register(NewSnmpExporter())
	// NewSnmpExporter()
	NewSnmpExporter()
	// exporter.Register(nil)
}

const (
	namespace = "snmp"
)

var (
	// Metrics about the SNMP exporter itself.
	snmpRequestErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "request_errors_total",
			Help:      "Errors in requests to the SNMP exporter",
		},
	)
	snmpCollectionDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "collection_duration_seconds",
			Help:      "Duration of collections by the SNMP exporter",
		},
		[]string{"module"},
	)
	sc = &SafeConfig{
		C: &Config{},
	}
	reloadCh   chan chan error
	configFile = []string{
		"snmp.yaml",
	}
	concurrency     = 1
	debug           = false
	expandEnvVars   = false
	buckets         = prometheus.ExponentialBuckets(0.0001, 2, 15)
	exporterMetrics = Metrics{
		SNMPCollectionDuration: snmpCollectionDuration,
		SNMPUnexpectedPduType: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "unexpected_pdu_type_total",
				Help:      "Unexpected Go types in a PDU.",
			},
		),
		SNMPDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "packet_duration_seconds",
				Help:      "A histogram of latencies for SNMP packets.",
				Buckets:   buckets,
			},
		),
		SNMPPackets: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "packets_total",
				Help:      "Number of SNMP packet sent, including retries.",
			},
		),
		SNMPRetries: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "packet_retries_total",
				Help:      "Number of SNMP packet retries.",
			},
		),
		SNMPInflight: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "request_in_flight",
				Help:      "Current number of SNMP scrapes being requested.",
			},
		),
	}
)

type SafeConfig struct {
	sync.RWMutex
	C *Config
}

func Handler(w http.ResponseWriter, r *http.Request) {
	handler(w, r, exporterMetrics, concurrency)
}

func handler(w http.ResponseWriter, r *http.Request, exporterMetrics Metrics, concurrency int) {
	query := r.URL.Query()
	debug := getDebugFlag(query)
	target := getTargetParam(w, query)
	if target == "" {
		return
	}

	authName := getAuthParam(w, query)
	snmpContext := getSnmpContextParam(w, query)
	modules := getModuleParams(query)
	if len(modules) == 0 {
		return
	}

	auth, nmodules, err := validateAndGetModules(authName, modules)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		snmpRequestErrors.Inc()
		return
	}

	processRequest(w, r, target, authName, snmpContext, auth, nmodules, exporterMetrics, concurrency, debug)
}

// Helper functions
func getDebugFlag(query url.Values) bool {
	if query.Get("snmp_debug_packets") == "true" {
		logrus.Debug("Debug query param enabled")
		return true
	}
	return false
}

func getTargetParam(w http.ResponseWriter, query url.Values) string {
	target := query.Get("target")
	if len(query["target"]) != 1 || target == "" {
		http.Error(w, "'target' parameter must be specified once", http.StatusBadRequest)
		snmpRequestErrors.Inc()
		return ""
	}
	return target
}

func getAuthParam(w http.ResponseWriter, query url.Values) string {
	authName := query.Get("auth")
	if len(query["auth"]) > 1 {
		http.Error(w, "'auth' parameter must only be specified once", http.StatusBadRequest)
		snmpRequestErrors.Inc()
		return ""
	}
	if authName == "" {
		return "public_v2"
	}
	return authName
}

func getSnmpContextParam(w http.ResponseWriter, query url.Values) string {
	snmpContext := query.Get("snmp_context")
	if len(query["snmp_context"]) > 1 {
		http.Error(w, "'snmp_context' parameter must only be specified once", http.StatusBadRequest)
		snmpRequestErrors.Inc()
		return ""
	}
	return snmpContext
}

func getModuleParams(query url.Values) []string {
	queryModule := query["module"]
	if len(queryModule) == 0 {
		queryModule = append(queryModule, "if_mib")
	}

	uniqueM := make(map[string]bool)
	var modules []string
	for _, qm := range queryModule {
		for _, m := range strings.Split(qm, ",") {
			if m != "" && !uniqueM[m] {
				uniqueM[m] = true
				modules = append(modules, m)
			}
		}
	}
	return modules
}

func validateAndGetModules(authName string, modules []string) (*Auth, []*NamedModule, error) {
	sc.RLock()
	defer sc.RUnlock()

	auth, authOk := sc.C.Auths[authName]
	if !authOk {
		return nil, nil, fmt.Errorf("Unknown auth '%s'", authName)
	}

	var nmodules []*NamedModule
	for _, m := range modules {
		module, moduleOk := sc.C.Modules[m]
		if !moduleOk {
			return nil, nil, fmt.Errorf("Unknown module '%s'", m)
		}
		nmodules = append(nmodules, NewNamedModule(m, module))
	}
	return auth, nmodules, nil
}

func processRequest(w http.ResponseWriter, r *http.Request, target, authName, snmpContext string,
	auth *Auth, nmodules []*NamedModule, exporterMetrics Metrics, concurrency int, debug bool) {

	fields := logrus.Fields{"auth": authName, "target": target}
	logger := logrus.WithFields(fields)
	registry := prometheus.NewRegistry()
	c := SnmpCollectorNew(r.Context(), target, authName, snmpContext, auth, nmodules, logger, exporterMetrics, concurrency, debug)
	registry.MustRegister(c)
	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}

//	func UpdateConfiguration(w http.ResponseWriter, r *http.Request) {
//		switch r.Method {
//		case "POST":
//			rc := make(chan error)
//			reloadCh <- rc
//			if err := <-rc; err != nil {
//				http.Error(w, fmt.Sprintf("failed to reload config: %s", err), http.StatusInternalServerError)
//			}
//		default:
//			http.Error(w, "POST method expected", http.StatusBadRequest)
//		}
//	}
func UpdateConfiguration(w http.ResponseWriter, r *http.Request) {
	if !isValidMethod(r) {
		sendMethodError(w)
		return
	}

	if err := reloadConfiguration(); err != nil {
		sendReloadError(w, err)
	}
}

func isValidMethod(r *http.Request) bool {
	return r.Method == "POST"
}

func sendMethodError(w http.ResponseWriter) {
	http.Error(w, "POST method expected", http.StatusBadRequest)
}

func reloadConfiguration() error {
	rc := make(chan error)
	reloadCh <- rc
	return <-rc
}

func sendReloadError(w http.ResponseWriter, err error) {
	http.Error(
		w,
		fmt.Sprintf("failed to reload config: %s", err),
		http.StatusInternalServerError,
	)
}

func (sc *SafeConfig) ReloadConfig(configFile []string, expandEnvVars bool) (err error) {
	conf, err := LoadFile(configFile, expandEnvVars)
	if err != nil {
		return err
	}
	sc.Lock()
	sc.C = conf
	// Initialize metrics.
	for module := range sc.C.Modules {
		snmpCollectionDuration.WithLabelValues(module)
	}
	sc.Unlock()
	return nil
}

func NewSnmpExporter() {
	config, err := LoadConfig("/etc/uos-exporter/snmp-exporter.yaml")
	if err != nil {
		logrus.Errorf("Error get snmp config %v\n", err)
	} else {
		configFile = config.SnmpConfigFile
		concurrency = config.SnmpConcurrency
		debug = config.SnmpDebug
		expandEnvVars = config.SnmpExpandEnvVars
	}

	if concurrency < 1 {
		concurrency = 1
	}

	logrus.Info("Starting snmp_exporter", "version", version.Info(), "concurrency", concurrency, "debug_snmp", debug)
	logrus.Info("operational information", "build_context", version.BuildContext())

	// Bail early if the config is bad.
	err = sc.ReloadConfig(configFile, expandEnvVars)
	if err != nil {
		logrus.Error("Error parsing config file", "err", err)
		os.Exit(1)
	}

	hup := make(chan os.Signal, 1)
	reloadCh = make(chan chan error)
	signal.Notify(hup, syscall.SIGHUP)

	go handleReloadSignals(hup)

	// 如果需要pprof功能，可以通过环境变量控制
	if os.Getenv("ENABLE_PPROF") == "true" {
		go func() {
			logrus.Info("Starting pprof server on :6060")
			server := &http.Server{
				Addr: ":6060",
				// 设置关键超时时间
				ReadTimeout:  15 * time.Second,
				WriteTimeout: 15 * time.Second,
				IdleTimeout:  60 * time.Second,
				// 可选：设置最大头部大小
				MaxHeaderBytes: 1 << 20, // 1MB
			}
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logrus.Error("Failed to start pprof server", "err", err)
			}
		}()
	}
}

func handleReloadSignals(hup chan os.Signal) {
	for {
		select {
		case <-hup:
			if err := sc.ReloadConfig(configFile, expandEnvVars); err != nil {
				logrus.Error("Error reloading config", "err", err)
			} else {
				logrus.Info("Loaded config file")
			}
		case rc := <-reloadCh:
			if err := sc.ReloadConfig(configFile, expandEnvVars); err != nil {
				logrus.Error("Error reloading config", "err", err)
				rc <- err
			} else {
				logrus.Info("Loaded config file")
				rc <- nil
			}
		}
	}
}

type SnmpConfig struct {
	SnmpConfigFile    []string `yaml:"snmp_config_file"`
	SnmpConcurrency   int      `yaml:"snmp_concurrency"`
	SnmpDebug         bool     `yaml:"snmp_debug"`
	SnmpExpandEnvVars bool     `yaml:"snmp_expand_env_vars"`
}

func LoadConfig(path string) (*SnmpConfig, error) {
	cleanPath := filepath.Clean(path)
	if !strings.HasPrefix(cleanPath, "/etc/uos-exporter/") {
		return nil, fmt.Errorf("invalid config path: %s", path)
	}
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, err
	}

	var config SnmpConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

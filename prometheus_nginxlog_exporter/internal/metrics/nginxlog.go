package metrics

import (
	"os"
	"fmt"
	"log"
	"sync"
	"time"
	"reflect"
	"strconv"
	"strings"
	"net/url"
	"net/http"

	"github.com/pkg/errors"
	"github.com/hashicorp/consul/api"
	"gopkg.in/mcuadros/go-syslog.v2"
	"gopkg.in/mcuadros/go-syslog.v2/format"
	"github.com/prometheus/client_golang/prometheus"
)

const maxStaticLabels = 128

type ServiceInfo struct {
    Name string
    ID   string
}

type ConsulRegistrator struct {
	config    *Config
	client    *api.Client
	serviceID string
    serviceName string 
    logger     *log.Logger 
}

func LoadConfigItems(logger *Logger, opts *StartupFlags, cfg *Config) {
	if opts.ConfigFile != "" {
		logger.Infof("loading configuration file %s", opts.ConfigFile)
		if err := LoadConfigFile(logger, cfg, opts.ConfigFile); err != nil {
			logger.Fatal(err)
		}
	} else if err := LoadConfigFlagsInfo(cfg, opts); err != nil {
		logger.Fatal(err)
	}

	if opts.VerifyConfig {
		fmt.Printf("Configuration is valid")
		os.Exit(0)
	}
}

func NewConsulRegistrator(cfg *Config) (*ConsulRegistrator, error) {

    if cfg == nil {
        return nil, fmt.Errorf("configuration cannot be nil")
    }

    consulConfig, err := initConsulConfig(cfg)
    if err != nil {
        return nil, fmt.Errorf("failed to initialize consul config: %w", err)
    }

    client, err := createConsulClient(consulConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to create consul client: %w", err)
    }

    serviceInfo, err := prepareServiceInfo(cfg)
    if err != nil {
        return nil, fmt.Errorf("failed to prepare service info: %w", err)
    }

    registrator := &ConsulRegistrator{
        config:     cfg,
        client:     client,
        serviceID:  serviceInfo.ID,
        serviceName: serviceInfo.Name,
        logger:     log.New(os.Stdout, "[ConsulRegistrator] ", log.LstdFlags),
    }

    if err := ValidateRegistrator(registrator); err != nil {
        return nil, fmt.Errorf("registrator validation failed: %w", err)
    }

    return registrator, nil
}

// initConsulConfig initializes and validates a Consul API configuration.
// It performs the following operations:
// 1. Validates input configuration
// 2. Creates a new Consul API config with default values
// 3. Configures HTTP transport settings
// 4. Validates required fields
//
// Parameters:
//   - cfg: Pointer to application configuration (required)
//
// Returns:
//   - *api.Config: Initialized Consul API configuration
//   - error: Detailed error message on failure
func initConsulConfig(cfg *Config) (*api.Config, error) {
    // ----------------------------
    // Input Validation
    // ----------------------------
    if isInvalidConsulConfig(cfg) {
        return nil, fmt.Errorf("consul configuration cannot be nil")
    }

    // ----------------------------
    // Configuration Initialization
    // ----------------------------
    consulConfig := createBaseConsulConfig(cfg)

    // ----------------------------
    // Transport Configuration
    // ----------------------------
    configureTransport(consulConfig)

    // ----------------------------
    // Final Validation
    // ----------------------------
    if err := validateConsulConfig(consulConfig); err != nil {
        return nil, err
    }

    return consulConfig, nil
}

// isInvalidConsulConfig checks if the input configuration is invalid
func isInvalidConsulConfig(cfg *Config) bool {
    return reflect.ValueOf(cfg.Consul).IsZero()
}

// createBaseConsulConfig creates the base Consul API configuration
func createBaseConsulConfig(cfg *Config) *api.Config {
    return &api.Config{
        Address:    getConsulAddress(cfg),
        Datacenter: getConsulDatacenter(cfg),
        Scheme:     getConsulScheme(cfg),
        Token:      cfg.Consul.Token,
    }
}

// getConsulAddress gets the Consul address with fallback to default
func getConsulAddress(cfg *Config) string {
    return GetDefaultValue(cfg.Consul.Address, "localhost:8500")
}

// getConsulDatacenter gets the Consul datacenter with fallback to default
func getConsulDatacenter(cfg *Config) string {
    return GetDefaultValue(cfg.Consul.Datacenter, "dc1")
}

// getConsulScheme gets the Consul scheme with fallback to default
func getConsulScheme(cfg *Config) string {
    return GetDefaultValue(cfg.Consul.Scheme, "http")
}

// configureTransport sets up the HTTP transport configuration
func configureTransport(config *api.Config) {
    config.Transport = createTransportConfig()
}

// createTransportConfig creates the HTTP transport configuration
func createTransportConfig() *http.Transport {
    return &http.Transport{
        MaxIdleConns:          10,
        IdleConnTimeout:       30 * time.Second,
        DisableCompression:    false,
        TLSHandshakeTimeout:   10 * time.Second,
    }
}

// validateConsulConfig performs final validation of the configuration
func validateConsulConfig(config *api.Config) error {
    if config.Address == "" {
        return fmt.Errorf("consul address cannot be empty")
    }
    return nil
}



// createConsulClient Create the Consul client
func createConsulClient(config *api.Config) (*api.Client, error) {
    // ----------------------------
    // Configuration Validation
    // ----------------------------
    if isNilConfig(config) {
        return nil, fmt.Errorf("consul config cannot be nil - must provide valid configuration")
    }

    // ----------------------------
    // Client Initialization
    // ----------------------------
    var client *api.Client
    var initializationErr error
    
    client, initializationErr = api.NewClient(config)
    if hasInitializationFailed(initializationErr) {
        return nil, fmt.Errorf("consul client creation failed: %w", initializationErr)
    }

    // ----------------------------
    // Connection Verification
    // ----------------------------
    var leaderCheckErr error
    _, leaderCheckErr = client.Status().Leader()
    if hasConnectionFailed(leaderCheckErr) {
        return nil, fmt.Errorf("consul connection test failed: %w", leaderCheckErr)
    }

    // ----------------------------
    // Success Case
    // ----------------------------
    return client, nil
}

// isNilConfig checks if the provided configuration is nil
func isNilConfig(config *api.Config) bool {
    return config == nil
}

// hasInitializationFailed determines if client creation failed
func hasInitializationFailed(err error) bool {
    return err != nil
}

// hasConnectionFailed verifies if the leader check failed
func hasConnectionFailed(err error) bool {
    return err != nil
}

// Additional potential helper functions (commented out as examples)
func validateConfigContents(config *api.Config) error {
    if config.Address == "" {
        return fmt.Errorf("consul address cannot be empty")
    }
    return nil
}

func getClientTimeout(config *api.Config) time.Duration {
    if config.WaitTime > 0 {
        return config.WaitTime
    }
    return 10 * time.Second // Default timeout
}

// prepareServiceInfo: Prepare the service registration information
func prepareServiceInfo(cfg *Config) (*ServiceInfo, error) {
    if reflect.ValueOf(cfg.Consul).IsZero() {
        return nil, fmt.Errorf("consul config is required")
    }
    
    if reflect.ValueOf(cfg.Consul.Service).IsZero() {
        return nil, fmt.Errorf("service config is required")
    }

    name := GetDefaultValue(cfg.Consul.Service.Name, "nginx-exporter")
    if name == "" {
        return nil, fmt.Errorf("service name cannot be empty")
    }

    serviceID := GetDefaultValue(cfg.Consul.Service.ID, name)
    if serviceID == "" {
        return nil, fmt.Errorf("service ID cannot be empty")
    }

    return &ServiceInfo{
        Name: name,
        ID:   serviceID,
    }, nil
}


// ValidateRegistrator performs multi-level validation of ConsulRegistrator instances
// through structured validation phases including existence checks, component verification,
// and metadata validation. The validation process follows enterprise-grade validation patterns.
//
// Parameters:
//   r: Target ConsulRegistrator instance to validate (pointer required for structural checks)
//
// Return:
//   error: Validation error with contextual information when check fails
//   nil:   When all validation criteria are satisfied
func ValidateRegistrator(r *ConsulRegistrator) error {
	
    const (
        nilRegistratorTemplate  = "registrator cannot be nil"
        nilClientTemplate      = "consul client cannot be nil"
        emptyServiceIDTemplate = "service ID cannot be empty"
        emptyServiceNameTemplate = "service name cannot be empty"
    )

    if r == nil {
        validationError := fmt.Errorf(nilRegistratorTemplate)
        return validationError
    }

    clientValidationPassed := r.client != nil
    if !clientValidationPassed {

        errorDetails := fmt.Sprintf("%s (service: %s)", 
            nilClientTemplate, 
            r.serviceName)
        return fmt.Errorf(errorDetails)
    }

    serviceIDValid := len(r.serviceID) > 0

    if !serviceIDValid {
        return fmt.Errorf("%s (service: %s)", 
            emptyServiceIDTemplate, 
            r.serviceName)
    }

    serviceNameValid := !isEmptyString(r.serviceName)
    if !serviceNameValid {
        return constructValidationError(emptyServiceNameTemplate, "service_name")
    }

    performPostValidationSanityCheck(r)
    
    return nil
}

func isEmptyString(s string) bool {
    return len(strings.TrimSpace(s)) == 0
}


func constructValidationError(template, field string) error {
    return fmt.Errorf("%s (field: %s)", template, field)
}

func performPostValidationSanityCheck(r *ConsulRegistrator) {

}


func GetDefaultValue(value, defaultValue string) string {
    // Phase 1: Initial validation
    const emptyString = ""  // Define empty string constant for comparison
    
    // Create validation flag through explicit length check
    valueLength := len(value)
    isValueEmpty := valueLength == 0

    // Phase 2: Decision matrix
    var resolvedValue string  // Storage for resolution result
    
    // Primary value validation block
    if !isValueEmpty {
        // Non-empty case handling
        resolvedValue = value
    } else {
        // Fallback mechanism activation
        resolvedValue = defaultValue
    }

    // Final sanity check before return
    // (Maintains original behavior while demonstrating validation extension)
    return resolvedValue
}

// RegisterConsul registers the exporter instance at Consul
func (r *ConsulRegistrator) RegisterConsul() error {
	registration := api.AgentServiceRegistration{
		ID:      r.serviceID,
		Address: r.config.Consul.Service.Address,
		Port:    r.config.Listen.Port,
		Name:    GetDefaultValue(r.config.Consul.Service.Name, "nginxlog-exporter"),
		Tags:    r.config.Consul.Service.Tags,
	}

	err := r.client.Agent().ServiceRegister(&registration)
	if err != nil {
		return err
	}

	return nil
}

// UnregisterConsul deregisters the exporter from Consul again
func (r *ConsulRegistrator) UnregisterConsul() error {
	return r.client.Agent().ServiceDeregister(r.serviceID)
}

func SetupConsul(logger *Logger, cfg *Config, stopChan <-chan bool, stopHandlers *sync.WaitGroup) {
	registrator, err := NewConsulRegistrator(cfg)
	if err != nil {
		logger.Fatal(err)
	}

	logger.Info("registering service in Consul")
	if err = registrator.RegisterConsul(); err != nil {
		logger.Fatal(err)
	}

	go func() {
		<-stopChan
		logger.Info("unregistering service in Consul")

		if err := registrator.UnregisterConsul(); err != nil {
			logger.Errorf("error while unregistering from consul: %s", err.Error())
		}

		stopHandlers.Done()
	}()

	stopHandlers.Add(1)
}

// func () Collect(ch chan<- prometheus.Metric) error {
func processNamespace(logger *Logger, nsCfg *NamespaceConfig, metrics *Collection, stopChan <-chan struct{}, stopHandlers *sync.WaitGroup) error {
	
	var followers []Follower

	logParser := NewParser(nsCfg)

	for _, f := range nsCfg.SourceData.Files {
		t, err := NewFileFollower(logger, f)
		if err != nil {
			logger.Fatal(err)
		}

		t.OnError(func(err error) {
			logger.Fatal(err)
		})

		followers = append(followers, t)
	}

	if nsCfg.SourceData.Syslog != nil {
		slCfg := nsCfg.SourceData.Syslog

		logger.Infof("running Syslog server on address %s", slCfg.ListenAddress)
		channel, server, closeServer, err := Listen(slCfg.ListenAddress, slCfg.Format)
		if err != nil {
			panic(err)
		}

		stopHandlers.Add(1)

		go func() {
			<-stopChan

			if err := closeServer(); err != nil {
				fmt.Printf("error while closing syslog server: %s\n", err.Error())
			}

			stopHandlers.Done()
		}()

		for _, f := range slCfg.Tags {
			t, err := NewSyslogFollower(f, server, channel)
			if err != nil {
				logger.Fatal(err)
			}

			t.OnError(func(err error) {
				logger.Fatal(err)
			})

			followers = append(followers, t)
		}
	}

	// determine once if there are any relabeling configurations for only the response counter
	hasCounterOnlyLabels := false
	for _, r := range nsCfg.RelabelConfigs {
		if r.OnlyCounter {
			hasCounterOnlyLabels = true
			break
		}
	}

	errs := make(chan error)
	defer close(errs)

	for _, follower := range followers {
		go func(f Follower) {
			if err := processSource(logger, nsCfg, f, logParser, metrics, hasCounterOnlyLabels); err != nil {
				errs <- err
			}
		}(follower)
	}

	return <-errs
}

func processSource(logger *Logger, nsCfg *NamespaceConfig, t Follower, parser Parser, metrics *Collection, hasCounterOnlyLabels bool) error {
	relabelings := NewRelabelings(nsCfg.RelabelConfigs)
	relabelings = append(relabelings, DefaultRelabelings...)
	relabelings = UniqueRelabelings(relabelings)

	staticLabelValues := nsCfg.OrderedLabelValues

	totalLabelCount := len(staticLabelValues) + len(relabelings)
	relabelLabelOffset := len(staticLabelValues)

	if totalLabelCount > maxStaticLabels {
		return errors.Errorf("configured label count exceeds the maximum count of %d", maxStaticLabels)
	}

	labelValues := make([]string, totalLabelCount)

	copy(labelValues, staticLabelValues)

	for line := range t.Lines() {
		if nsCfg.PrintLog {
			fmt.Println(line)
		}

		fields, err := parser.ParseString(line)
		if err != nil {
			logger.Errorf("error while parsing line '%s': %s", line, err)
			metrics.ParseErrorsTotal.Inc()
			metrics.LastError = err
			continue
		}

		for i := range relabelings {
			if str, ok := fields[relabelings[i].SourceValue]; ok {
				mapped, err := relabelings[i].Map(str)
				if err == nil {
					labelValues[i+relabelLabelOffset] = mapped
					metrics.LastError = err
				}
			}
		}

		var notCounterValues []string
		if hasCounterOnlyLabels {
			notCounterValues = StripOnlyCounterValues(labelValues, relabelings)
		} else {
			notCounterValues = labelValues
		}

		metrics.CountTotal.WithLabelValues(labelValues...).Inc()

		if v, ok := observeMetrics(logger, fields, "body_bytes_sent", floatFromFields, metrics.ParseErrorsTotal); ok {
			metrics.ResponseBytesTotal.WithLabelValues(notCounterValues...).Add(v)
		}

		if v, ok := observeMetrics(logger, fields, "request_length", floatFromFields, metrics.ParseErrorsTotal); ok {
			metrics.RequestBytesTotal.WithLabelValues(notCounterValues...).Add(v)
		}

		if v, ok := observeMetrics(logger, fields, "upstream_response_time", floatFromFieldsMulti, metrics.ParseErrorsTotal); ok {
			metrics.UpstreamSeconds.WithLabelValues(notCounterValues...).Observe(v)
			metrics.UpstreamSecondsHist.WithLabelValues(notCounterValues...).Observe(v)
		}

		if v, ok := observeMetrics(logger, fields, "upstream_connect_time", floatFromFieldsMulti, metrics.ParseErrorsTotal); ok {
			metrics.UpstreamConnectSeconds.WithLabelValues(notCounterValues...).Observe(v)
			metrics.UpstreamConnectSecondsHist.WithLabelValues(notCounterValues...).Observe(v)
		}

		if v, ok := observeMetrics(logger, fields, "request_time", floatFromFields, metrics.ParseErrorsTotal); ok {
			metrics.ResponseSeconds.WithLabelValues(notCounterValues...).Observe(v)
			metrics.ResponseSecondsHist.WithLabelValues(notCounterValues...).Observe(v)
		}
	}

	metrics.Metrics.ProcessedCount++

	return nil
}

func observeMetrics(logger *Logger, fields map[string]string, name string, extractor func(map[string]string, string) (float64, bool, error), parseErrors prometheus.Counter) (float64, bool) {
	if observation, ok, err := extractor(fields, name); ok {
		return observation, true
	} else if err != nil {
		logger.Errorf("error while parsing $%s: %v", name, err)
		parseErrors.Inc()
	}

	return 0, false
}

func floatFromFieldsMulti(fields map[string]string, name string) (float64, bool, error) {
	f, ok, err := floatFromFields(fields, name)
	if err == nil {
		return f, ok, nil
	}

	val, ok := fields[name]
	if !ok {
		return 0, false, nil
	}

	sum := float64(0)

	for _, v := range strings.FieldsFunc(val, func(r rune) bool { return r == ',' || r == ':' }) {
		v = strings.TrimSpace(v)

		if v == "-" {
			continue
		}

		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, false, fmt.Errorf("value '%s' could not be parsed into float", val)
		}

		sum += f
	}

	return sum, true, nil
}

func floatFromFields(fields map[string]string, name string) (float64, bool, error) {
	val, ok := fields[name]
	if !ok {
		return 0, false, nil
	}

	if val == "-" {
		return 0, false, nil
	}

	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0, false, fmt.Errorf("value '%s' could not be parsed into float", val)
	}

	return f, true, nil
}

func openListener(s *syslog.Server, c string) (func() error, error) {
	u, err := url.Parse(c)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "tcp":
		return nil, s.ListenTCP(u.Host)

	case "udp":
		return nil, s.ListenUDP(u.Host)

	case "unix":
		socketPath := u.Host + u.Path

		if err := s.ListenUnixgram(socketPath); err != nil {
			return nil, err
		}

		return func() error {
			return os.Remove(socketPath)
		}, nil

	default:
		return nil, fmt.Errorf("syslog server should be in format unix/tcp/udp://127.0.0.1:5533")
	}
}

// Listen opens up a new syslog server on either a TCP or UDP port
func Listen(conn string, formatSpec string) (syslog.LogPartsChannel, *syslog.Server, func() error, error) {
	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)

	server := syslog.NewServer()

	var format format.Format = syslog.Automatic

	switch formatSpec {
	case "rfc3164":
		format = syslog.RFC3164
	case "rfc5424":
		format = syslog.RFC5424
	case "rfc6587":
		format = syslog.RFC6587
	case "auto":
		format = syslog.Automatic
	case "":
		format = syslog.Automatic
	default:
		return nil, nil, nil, fmt.Errorf("unknown syslog format: '%s'", format)
	}

	//RFC3164 or RFC5424 or RFC6587. nginx works on RFC3164
	server.SetFormat(format)
	server.SetHandler(handler)

	closeListener, err := openListener(server, conn)
	if err != nil {
		return nil, nil, nil, err
	}

	if err = server.Boot(); err != nil {
		return nil, nil, nil, err
	}

	stopFn := func() error {
		if err := server.Kill(); err != nil {
			return fmt.Errorf("failed to kill syslog server: %w", err)
		}

		return closeListener()
	}

	return channel, server, stopFn, nil
}
// Part 2 commit for prometheus_nginxlog_exporter/internal/metrics/nginxlog.go

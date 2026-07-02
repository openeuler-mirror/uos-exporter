package metrics

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"prometheus_nginxlog_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
)

type extendedWaitGroup struct {
	sync.WaitGroup
}

type NginxlogCollector struct {
	mu           sync.RWMutex
	namespaces   []*NamespaceMetrics
	stopChan     chan struct{}
	stopHandlers *sync.WaitGroup
	logger       *Logger
}

func NewNginxlogCollector() *NginxlogCollector {
	collector := &NginxlogCollector{
		stopChan:     make(chan struct{}),
		stopHandlers: &sync.WaitGroup{},
	}

	collector.initConfig()

	collector.SignalHandleSet()

	collector.NamespaceStart()

	return collector
}

// 实现prometheus.Collector接口
func (c *NginxlogCollector) Describe(ch chan<- *prometheus.Desc) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, ns := range c.namespaces {
		ns.Describe(ch)
	}
}

func (c *NginxlogCollector) Collect(ch chan<- prometheus.Metric) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, ns := range c.namespaces {
		ns.Collect(ch)
	}
}

func (c *NginxlogCollector) initConfig() {
	var opts StartupFlags
	var cfg = Config{
		Listen: ListenConfig{
			Port:            9100,
			Address:         "0.0.0.0",
			MetricsEndpoint: "/metrics",
		},
	}

	flag.IntVar(&opts.ListenPort,
		"listen-port",
		9100,
		"HTTP port to listen on")

	flag.StringVar(&opts.ListenAddress,
		"listen-address",
		"0.0.0.0",
		"IP-address to bind")

	flag.StringVar(&opts.Parser,
		"parser",
		"text",
		"NGINX access log format parser. One of: [text, json]")

	flag.StringVar(&opts.Format,
		"format",
		`$remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent" "$http_x_forwarded_for"`,
		"NGINX access log format")

	flag.StringVar(&opts.Namespace,
		"namespace",
		"nginx",
		"namespace to use for metric names")

	flag.StringVar(&opts.ConfigFile,
		"config-file",
		"",
		"Configuration file to read from")

	flag.BoolVar(&opts.EnableExperimentalFeatures,
		"enable-experimental",
		false,
		"Set this flag to enable experimental features")

	flag.StringVar(&opts.CPUProfile,
		"cpuprofile",
		"",
		"write cpu profile to `file`")

	flag.StringVar(&opts.MemProfile,
		"memprofile",
		"",
		"write memory profile to `file`")

	flag.StringVar(&opts.MetricsEndpoint,
		"metrics-endpoint",
		cfg.Listen.MetricsEndpoint,
		"URL path at which to serve metrics")

	flag.StringVar(&opts.LogLevel,
		"log-level",
		"info",
		"level of logs. Allowed values: error, warning, info, debug")

	flag.StringVar(&opts.LogFormat,
		"log-format",
		"console",
		"Define log format. Allowed values: console, json")

	flag.BoolVar(&opts.VerifyConfig,
		"verify-config",
		false,
		"Enable this flag to check config file loads, then exit")

	flag.BoolVar(&opts.Version,
		"version",
		false,
		"set to print version information")

	flag.Parse()

	logger, err := NewLog(opts.LogLevel, opts.LogFormat)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	c.logger = logger

	c.logger.Infof("opts %-v", opts)
	LoadConfigItems(logger, &opts, &cfg)

	c.mu.Lock()
	defer c.mu.Unlock()

	for i := range cfg.Namespaces {
		namespace := &cfg.Namespaces[i]
		nsMetrics := NewForNamespace(namespace)
		c.namespaces = append(c.namespaces, nsMetrics)
	}
}

func (c *NginxlogCollector) NamespaceStart() {

	startTime := time.Now()
	c.logger.Debugf("Starting namespace processors at %s",
		startTime.Format(time.RFC3339))

	c.logger.Debug("Acquiring read lock for namespace processors")
	c.mu.RLock()
	defer func() {
		c.logger.Debug("Releasing read lock for namespace processors")
		c.mu.RUnlock()
	}()

	namespaceCount := len(c.namespaces)
	c.logger.Infof("Starting processors for %d namespaces", namespaceCount)

	if namespaceCount == 0 {
		c.logger.Warn("No namespaces configured to process")
		return
	}

	for idx, ns := range c.namespaces {

		c.logger.WithFields(map[string]interface{}{
			"namespace": ns.cfg.Name,
			"index":     idx + 1,
			"total":     namespaceCount,
		}).Info("Initializing namespace processor")

		go func(nsCfg *NamespaceConfig, collection *Collection, nsIndex int) {

			c.logger.WithFields(map[string]interface{}{
				"namespace": nsCfg.Name,
				"goroutine": runtime.NumGoroutine(),
			}).Debug("Starting namespace processing goroutine")

			c.stopHandlers.Add(1)
			defer func() {
				c.stopHandlers.Done()
				c.logger.WithFields(map[string]interface{}{
					"namespace": nsCfg.Name,
				}).Debug("Namespace processor goroutine completed")
			}()

			processStart := time.Now()
			c.logger.WithFields(map[string]interface{}{
				"namespace": nsCfg.Name,
			}).Debug("Namespace processor starting")

			_ = processNamespace(
				c.logger,
				nsCfg,
				collection,
				c.stopChan,
				c.stopHandlers,
			)

			// 记录处理耗时
			duration := time.Since(processStart)
			c.logger.WithFields(map[string]interface{}{
				"namespace": nsCfg.Name,
				"duration":  duration.String(),
			}).Info("Namespace processor completed")
		}(ns.cfg, &ns.Collection, idx)
	}

	// 记录总启动耗时
	totalDuration := time.Since(startTime)
	c.logger.WithFields(map[string]interface{}{
		"namespaces": namespaceCount,
		"duration":   totalDuration.String(),
	}).Info("All namespace processors started successfully")

	go c.monitorNamespaceProcessors()
}

func (c *NginxlogCollector) monitorNamespaceProcessors() {
	const monitorInterval = 1 * time.Minute
	ticker := time.NewTicker(monitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			activeGoroutines := runtime.NumGoroutine()
			c.logger.WithFields(map[string]interface{}{
				"goroutines": activeGoroutines,
			}).Debug("Namespace processors health check")

			if activeGoroutines > len(c.namespaces)*2+10 {
				c.logger.WithFields(map[string]interface{}{
					"goroutines": activeGoroutines,
				}).Warn("Potential goroutine leak detected")
			}

		case <-c.stopChan:
			c.logger.Debug("Stopping namespace processors health monitoring")
			return
		}
	}
}

func (c *NginxlogCollector) getNamespaceProcessorStatus() map[string]interface{} {
	status := make(map[string]interface{})
	c.mu.RLock()
	defer c.mu.RUnlock()

	status["total_namespaces"] = len(c.namespaces)
	status["active_goroutines"] = runtime.NumGoroutine()
	status["last_checked"] = time.Now().Format(time.RFC3339)

	namespaces := make([]map[string]interface{}, 0, len(c.namespaces))
	for _, ns := range c.namespaces {
		nsInfo := map[string]interface{}{
			"name":       ns.cfg.Name,
			"processed":  ns.Collection.Metrics.ProcessedCount,
			"last_error": ns.Collection.LastError,
		}
		namespaces = append(namespaces, nsInfo)
	}

	status["namespaces"] = namespaces
	return status
}

func (c *NginxlogCollector) SignalHandleSet() {

	sigChan := make(chan os.Signal, 1)

	signalsToCapture := []os.Signal{
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGINT,
	}

	signal.Notify(sigChan, signalsToCapture...)

	go func() {

		c.logger.Debug("Signal handling goroutine started")

		sig := <-sigChan

		switch sig {
		case os.Interrupt, syscall.SIGINT:
			c.logger.Infof("Received interrupt signal: %s. Initiating graceful shutdown...", sig)
		case syscall.SIGTERM:
			c.logger.Infof("Received termination signal: %s. Initiating graceful shutdown...", sig)
		default:
			c.logger.Infof("Received unexpected signal: %s. Initiating shutdown...", sig)
		}

		c.logger.Debug("Closing stop channel to notify all handlers")
		close(c.stopChan)

		c.logger.Debug("Waiting for all handlers to complete...")

		c.stopHandlers.Wait()

		c.logger.Info("All handlers have completed. Shutdown finished.")

		if sig == syscall.SIGTERM {
			c.logger.Info("Exiting with status code 0 after SIGTERM")
			os.Exit(0)
		} else {
			c.logger.Info("Exiting with status code 1 after interrupt")
			os.Exit(1)
		}
	}()

	c.logger.Info("Signal handlers have been successfully configured")

	go c.monitorSignalHandlerHealth(sigChan)
}

func (c *NginxlogCollector) monitorSignalHandlerHealth(sigChan chan os.Signal) {
	const monitorInterval = 30 * time.Second

	ticker := time.NewTicker(monitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.logger.Debug("Performing periodic health check of signal handler")

			if _, ok := <-sigChan; !ok {
				c.logger.Warn("Signal channel appears to be closed unexpectedly")
				return
			}

		case <-c.stopChan:

			c.logger.Debug("Stopping signal handler health monitoring")
			return
		}
	}
}

func (c *NginxlogCollector) logSignalStatistics() {
	c.logger.Info("Signal handling statistics:")
	c.logger.Infof(" - Stop channel status: %v", func() string {
		select {
		case <-c.stopChan:
			return "closed"
		default:
			return "open"
		}
	}())
}

func (ewg *extendedWaitGroup) Count() int {
	return 0
}

// 初始化注册
func init() {
	collector := NewNginxlogCollector()
	exporter.Register(collector)
}

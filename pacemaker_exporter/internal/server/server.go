package server

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"pacemaker_exporter/config"
	"pacemaker_exporter/internal/exporter"
	metrics "pacemaker_exporter/internal/metrics"
	"pacemaker_exporter/pkg/logger"
	"pacemaker_exporter/pkg/ratelimit"
	"pacemaker_exporter/pkg/utils"

	"github.com/alecthomas/kingpin"
	"github.com/dustin/go-humanize"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var defaultSeverVersion = "1.0.0"

type Server struct {
	Name           string
	Version        string
	CommonConfig   exporter.Config
	promReg        *prometheus.Registry
	handlers       []HandlerFunc
	ExitSignal     chan struct{}
	Error          error
	callback       sync.Once
	ExporterConfig config.Settings
	server         *http.Server
}

func NewServer(name, version string) *Server {
	if version == "" {
		version = defaultSeverVersion
	}
	s := &Server{
		Name:         name,
		Version:      version,
		CommonConfig: exporter.DefaultConfig,
		promReg:      prometheus.NewRegistry(),
		ExitSignal:   make(chan struct{}),
	}
	return s
}

// SetUp initializes the server with improved error handling and logging
func (s *Server) SetUp() error {
	setupSteps := []struct {
		name string
		fn   func() error
	}{
		{"parse command line arguments", s.parse},
		{"load configuration", s.loadConfig},
		{"setup logging", s.setupLog},
		{"setup HTTP server", s.setupHttpServer},
		{"unpack exporter config", s.setupExporterConfig},
	}

	for _, step := range setupSteps {
		logrus.WithField("step", step.name).Debug("Executing setup step")
		if err := step.fn(); err != nil {
			logrus.WithFields(logrus.Fields{
				"step":  step.name,
				"error": err,
			}).Error("Setup step failed")
			s.Error = err
			return fmt.Errorf("failed to %s: %w", step.name, err)
		}
	}

	logrus.Info("Server setup completed successfully")
	return nil
}

// setupExporterConfig handles exporter configuration setup
func (s *Server) setupExporterConfig() error {
	if err := exporter.Unpack(&s.ExporterConfig); err != nil {
		logrus.WithError(err).Warn("Failed to unpack config, using default")
	}

	// Override with command line parameters if provided
	if config.ScrapeUrl != nil {
		logrus.Info("Overriding scrape URL with command-line parameter")
		s.ExporterConfig.ScrapeUri = *config.ScrapeUrl
	}

	return nil
}

func (s *Server) setupLog() error {
	size, err := humanize.ParseBytes(s.CommonConfig.Logging.MaxSize)
	if err != nil {
		return fmt.Errorf("parsing log size failed: %w", err)
	}

	logConfig := logger.NewConfig(
		s.CommonConfig.Logging.Level,
		s.CommonConfig.Logging.LogPath,
		safeUint64ToInt64(size),
		s.CommonConfig.Logging.MaxAge,
	)
	logger.Init(logConfig)
	return nil
}

// healthzHandler provides a health check endpoint with improved error handling
func (s *Server) healthzHandler(w http.ResponseWriter, r *http.Request) {
	type healthzResponse struct {
		Status    string    `json:"status"`
		Message   string    `json:"message"`
		Timestamp time.Time `json:"timestamp"`
		Version   string    `json:"version"`
	}

	response := healthzResponse{
		Status:    "ok",
		Message:   fmt.Sprintf("%s is running normally.", s.Name),
		Timestamp: time.Now().UTC(),
		Version:   s.Version,
	}

	// Set JSON content type
	w.Header().Set("Content-Type", "application/json")

	// Encode directly to response writer for better performance
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logrus.WithFields(logrus.Fields{
			"method":     r.Method,
			"path":       r.URL.Path,
			"remote_ip":  r.RemoteAddr,
			"user_agent": r.Header.Get("User-Agent"),
			"error":      err,
		}).Error("Failed to encode healthz response")
		// Response already started, can't change status code
		return
	}

	logrus.WithFields(logrus.Fields{
		"method":    r.Method,
		"path":      r.URL.Path,
		"remote_ip": r.RemoteAddr,
		"status":    "ok",
	}).Debug("Health check requested")
}

// getName returns the server name (thread-safe)
func (s *Server) getName() string {
	return s.Name
}

func (s *Server) setupHttpServer() error {
	exporter.RegisterPrometheus(s.promReg)
	mux := http.NewServeMux()

	// Register health check endpoint
	mux.HandleFunc("/healthz", s.healthzHandler)

	// Register metrics and other endpoints
	mux.Handle(s.CommonConfig.MetricsPath, s)
	mux.HandleFunc("/html", metrics.HTMLHandler)
	mux.HandleFunc("/xml", metrics.XMLHandler)

	// Setup rate limiting if enabled
	if err := s.setupRateLimit(); err != nil {
		return fmt.Errorf("failed to setup rate limiting: %w", err)
	}

	// Setup server address and landing page
	addr := fmt.Sprintf("%s:%d", s.CommonConfig.Address, s.CommonConfig.Port)
	if err := s.setupLandingPage(mux); err != nil {
		return fmt.Errorf("failed to setup landing page: %w", err)
	}

	// Create HTTP server
	s.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logrus.WithFields(logrus.Fields{
		"address": addr,
		"name":    s.Name,
	}).Info("HTTP server configured successfully")

	return nil
}

// setupRateLimit configures rate limiting if enabled
func (s *Server) setupRateLimit() error {
	if !*UseRatelimit {
		return nil
	}

	rateLimiter, err := ratelimit.NewRateLimiter(*rateLimitInterval, *rateLimitSize)
	if err != nil {
		return fmt.Errorf("rate limiter initialization failed: %w", err)
	}

	s.Use(Ratelimit(rateLimiter))
	logrus.WithFields(logrus.Fields{
		"interval": *rateLimitInterval,
		"size":     *rateLimitSize,
	}).Info("Rate limiting enabled")

	return nil
}

// setupLandingPage configures the landing page and favicon
func (s *Server) setupLandingPage(mux *http.ServeMux) error {
	landConfig := LandingPageConfig{
		Name:    s.Name,
		Version: s.Version,
		Links: []LandingPageLinks{
			{Text: "Metrics", Address: s.CommonConfig.MetricsPath},
			{Text: "Health Check", Address: "/healthz"},
		},
	}

	landPage, err := NewLandingPage(landConfig)
	if err != nil {
		return fmt.Errorf("landing page creation failed: %w", err)
	}

	mux.HandleFunc("/", landPage.ServeHTTP)
	mux.Handle("/favicon.ico", NewFavicon())

	return nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req := s.createRequest(w, r)
	for _, handler := range s.handlers {
		handler(req)
		if req.Error != nil {
			return
		}
	}
	promhttp.HandlerFor(s.promReg, promhttp.HandlerOpts{}).ServeHTTP(w, r)
}

func (s *Server) Use(handlerFuncs ...HandlerFunc) {
	s.handlers = append(s.handlers, handlerFuncs...)
}

func (s *Server) createRequest(w http.ResponseWriter, r *http.Request) *Request {
	req := NewRequest(w, r)
	req.handlers = s.handlers
	return req
}

// Run starts the HTTP server with improved logging and signal handling
func (s *Server) Run() error {
	// Start signal handling in background
	go utils.HandleSignals(s.Exit)

	addr := s.server.Addr
	logrus.WithFields(logrus.Fields{
		"server":  s.Name,
		"version": s.Version,
		"address": addr,
	}).Info("Starting HTTP server")

	fmt.Fprintf(os.Stdout, "Listening and serving %s on [http://%s]\n", s.Name, addr)

	// Start server
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logrus.WithFields(logrus.Fields{
			"server": s.Name,
			"addr":   addr,
			"error":  err,
		}).Error("HTTP server failed")
		return fmt.Errorf("server listen and serve failed: %w", err)
	}

	logrus.WithField("server", s.Name).Info("HTTP server stopped")
	return nil
}

func (s *Server) PrintVersion() {
	logrus.WithFields(logrus.Fields{
		"name":    s.Name,
		"version": s.Version,
	}).Info("Server version information")
}

// Stop gracefully shuts down the server with configurable timeout
func (s *Server) Stop() {
	logrus.WithField("server", s.Name).Info("Initiating server shutdown")
	logger.LogOutput("Shutting down server...")

	// Increased timeout for more graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			logrus.WithField("timeout", "30s").Warn("Server shutdown timed out, forcing close")
		} else {
			logrus.WithError(err).Error("Server shutdown failed")
		}
	} else {
		logrus.WithField("server", s.Name).Info("Server gracefully stopped")
	}
}

func (s *Server) Exit() {
	s.callback.Do(func() {
		logrus.WithField("server", s.Name).Debug("Sending exit signal")
		close(s.ExitSignal)
	})
}

func (s *Server) parse() error {
	kingpin.Parse()
	return nil
}

// loadConfig loads configuration from file with improved error handling
func (s *Server) loadConfig() error {
	configFile := *exporter.Configfile
	cleanPath := filepath.Clean(configFile)
	if !strings.HasPrefix(cleanPath, "/etc/uos-exporter/") {
		return fmt.Errorf("config file path must be under /etc/uos-exporter/")
	}
	content, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			logrus.WithField("config_file", configFile).Warn("Config file not found, using default configuration")
		} else {
			logrus.WithFields(logrus.Fields{
				"config_file": configFile,
				"error":       err,
			}).Warn("Failed to read config file, using default configuration")
		}
		return nil // Not returning error as we can use defaults
	}

	if err := yaml.Unmarshal(content, &s.CommonConfig); err != nil {
		logrus.WithFields(logrus.Fields{
			"config_file": configFile,
			"error":       err,
		}).Warn("Failed to parse config file, using default configuration")
		return nil // Not returning error as we can use defaults
	}

	logrus.WithField("config_file", configFile).Info("Configuration loaded successfully")
	return nil
}

func safeUint64ToInt64(value uint64) int64 {
	if value > math.MaxInt64 {
		return int64(math.MaxInt64)
	}
	return int64(value)
}

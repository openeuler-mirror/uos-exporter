package server

import (
	"context"
	"nextdns_exporter/internal/exporter"
	"nextdns_exporter/internal/metrics"
	"nextdns_exporter/pkg/logger"
	"nextdns_exporter/pkg/ratelimit"
	"nextdns_exporter/pkg/utils"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
	"github.com/alecthomas/kingpin/v2"
	"encoding/json"
	"bytes"
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
	server         *http.Server
}

func NewServer(name, version string) *Server {
	if version == "" {
		version = defaultSeverVersion
	}
	
	// 初始化Prometheus registry
	promReg := prometheus.NewRegistry()
	
	// 使用默认配置创建NextDNS指标
	apiKey := os.Getenv("NEXTDNS_API_KEY")
	profileID := os.Getenv("NEXTDNS_PROFILE_ID")
	
	if apiKey == "" {
		logrus.Warn("未设置环境变量 NEXTDNS_API_KEY")
		apiKey = "" // 使用空字符串而不是硬编码的占位符
	}
	
	if profileID == "" {
		logrus.Warn("未设置环境变量 NEXTDNS_PROFILE_ID")
		profileID = "" // 使用空字符串而不是硬编码的占位符
	}
	
	// 创建NextDNS指标实例
	nextDNSMetrics := metrics.NewNextDNSMetrics(profileID, apiKey)
	
	// 注册到Prometheus
	promReg.MustRegister(nextDNSMetrics)
	
	// 创建服务器实例
	s := &Server{
		Name:         name,
		Version:      version,
		CommonConfig: exporter.DefaultConfig,
		promReg:      promReg,
		ExitSignal:   make(chan struct{}),
	}
	
	// 设置默认值
	s.CommonConfig.NextDNS.APIKey = apiKey
	s.CommonConfig.NextDNS.ProfileID = profileID
	
	// 如果环境变量定义了端口，使用环境变量的值
	if portStr := os.Getenv("NEXTDNS_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			s.CommonConfig.Port = port
		}
	}
	
	return s
}

// New creates a new server from config
func New(cfg *exporter.Config) (*Server, error) {
	// 创建Prometheus registry
	promReg := prometheus.NewRegistry()
	
	// 创建NextDNS指标实例
	nextDNSMetrics := metrics.NewNextDNSMetrics(cfg.NextDNS.ProfileID, cfg.NextDNS.APIKey)
	
	// 注册到Prometheus
	promReg.MustRegister(nextDNSMetrics)
	
	server := &Server{
		Name:         "nextdns_exporter",
		Version:      "1.0.0",
		CommonConfig: *cfg,
		promReg:      promReg,
		ExitSignal:   make(chan struct{}),
	}
	
	// 设置HTTP服务器
	err := server.setupHttpServer()
	if err != nil {
		return nil, fmt.Errorf("failed to setup HTTP server: %v", err)
	}
	
	return server, nil
}

func (s *Server) SetUp() error {
	defer func() {
		if s.Error != nil {
			logrus.Errorf("SetUp error: %v", s.Error)
		}
	}()
	err := s.parse()
	if err != nil {
		logrus.Errorf("Parsing command line arguments failed: %v", err)
		return err
	}
	
	// 加载通用配置
	err = exporter.Unpack(&s.CommonConfig)
	if err != nil {
		logrus.Errorf("Loading config file failed: %v", err)
		return err
	}
	
	// 设置日志
	err = s.setupLog()
	if err != nil {
		logrus.Errorf("SetUp error: %v", err)
		return err
	}

	// 设置HTTP服务器
	err = s.setupHttpServer()
	if err != nil {
		logrus.Errorf("SetUp error: %v", err)
		return err
	}
	
	return nil
}

func (s *Server) setupLog() error {
	// 直接使用Config中的值
	logConfig := logger.NewConfig(
		s.CommonConfig.Logging.Level,
		s.CommonConfig.Logging.LogPath,
		s.CommonConfig.Logging.MaxSize,
		s.CommonConfig.Logging.MaxAge,
	)
	logger.Init(logConfig)
	return nil
}

func (s *Server) healthzHandler(w http.ResponseWriter, r *http.Request) {
	// Health check response
	type healthzResponse struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	response := healthzResponse{
		Status:  "ok",
		Message: fmt.Sprintf("%s is running normally.", s.getName()),
	}

	// Set header
	w.Header().Set("Content-Type", "application/json")

	// Buffer JSON encoding
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(response); err != nil {
		logrus.WithFields(logrus.Fields{
			"method": r.Method,
			"path":   r.URL.Path,
			"error":  err,
		}).Error("Failed to encode healthz response")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Write status and body
	w.WriteHeader(http.StatusOK)
	if _, err := buf.WriteTo(w); err != nil {
		logrus.WithFields(logrus.Fields{
			"method": r.Method,
			"path":   r.URL.Path,
			"error":  err,
		}).Error("Failed to write healthz response to client")
	}
}

// Thread-safe method to get Name
func (s *Server) getName() string {
	return s.Name
}

func (s *Server) setupHttpServer() error {
	mux := http.NewServeMux()
	
	// 创建自定义metrics处理函数
	metricsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 设置Prometheus处理选项
		promHandler := promhttp.HandlerFor(
			s.promReg,
			promhttp.HandlerOpts{
				EnableOpenMetrics: true,
				Registry:          s.promReg,
			},
		)
		
		// 处理请求
		promHandler.ServeHTTP(w, r)
	})
	
	// 注册自定义metrics处理函数
	mux.Handle(s.CommonConfig.MetricsPath, metricsHandler)

	// Register health check
	mux.HandleFunc("/healthz", s.healthzHandler)

	// 启用ratelimit功能
	if *UseRatelimit {
		interval := *rateLimitInterval
		size := *rateLimitSize
		intervalStr := interval.String()
		
		rateLimiter, err := ratelimit.NewRateLimiter(intervalStr, size)
		if err != nil {
			logrus.Errorf("ratelimit middleware init error: %v", err)
		} else {
			s.Use(Ratelimit(rateLimiter))
		}
	}
	
	addr := fmt.Sprintf("%s:%d", s.CommonConfig.Address, s.CommonConfig.Port)
	schema := "http"
	fmt.Fprintf(os.Stdout, "Listening and serving %s on [%s://%s]\n", s.Name, schema, addr)
	server := &http.Server{
		Addr:        addr,
		Handler:     mux,
		ReadTimeout: 15 * time.Second,
	}
	
	// 创建登录页面
	landConfig := LandingPageConfig{
		Name:    s.Name,
		Version: s.Version,
		Links: []LandingPageLinks{
			{
				Text:    "Metrics",
				Address: s.CommonConfig.MetricsPath,
			},
			{
				Text:    "Health Check",
				Address: "/healthz",
			},
		},
	}
	
	landPage, err := NewLandingPage(landConfig)
	if err != nil {
		logrus.Errorf("Failed to create landing page: %v", err)
		return err
	}
	
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		landPage.ServeHTTP(w, r)
	})
	
	// 添加favicon
	faviconHandler := NewFavicon()
	mux.Handle("/favicon.ico", faviconHandler)
	
	s.server = server
	logrus.Infof("Server is running on %s", addr)
	
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

func (s *Server) Run() error {
	go utils.HandleOsSignals(s.Exit)
	logrus.Infof("%s sucessfully setup. SetUp running.", s.Name)

	logrus.Infof("Runing  %s", s.Name)
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logrus.Errorf("ListenAndServe Error: %s\n", err)
		return err
	}
	return nil
}

func (s *Server) PrintVersion() {
	logrus.Printf("%s version: %s\n", s.Name, s.Version)
}

func (s *Server) Stop() {
	logrus.Info("Stopping Server")
	logger.LogInfo("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			logrus.Warn("Server shutdown timed out")
		} else {
			logrus.Errorf("Server Shutdown Error: %s", err)
		}
	} else {
		logrus.Info("Server gracefully stopped")
	}
}

func (s *Server) Exit() {
	s.callback.Do(func() {
		close(s.ExitSignal)
	})
}

func (s *Server) parse() error {
	kingpin.Parse()
	return nil
}

func (s *Server) loadConfig() error {
	content, err := os.ReadFile(*exporter.ConfigfilePath)
	if err != nil {
		logrus.Errorf("Failed to read config file: %v", err)
		logrus.Info("Use default config")
		return nil
	}
	err = yaml.Unmarshal(content, &s.CommonConfig)
	if err != nil {
		logrus.Errorf("Failed to parse config file: %v", err)
		logrus.Info("Use default config")
		return nil
	}
	logrus.Infof("Loaded config file from: %s", *exporter.ConfigfilePath)
	logrus.Info("Configuration file loaded")
	return nil
}

// Start starts the HTTP server
func (s *Server) Start() error {
	go utils.HandleOsSignals(s.Exit)
	logrus.Infof("%s successfully setup. Starting server...", s.Name)

	err := s.Run()
	if err != nil {
		return err
	}
	return nil
}

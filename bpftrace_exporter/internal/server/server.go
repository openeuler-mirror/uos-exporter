package server

import (
	"bpftrace_exporter/config"
	"bpftrace_exporter/internal/exporter"
	_ "bpftrace_exporter/internal/metrics"
	"bpftrace_exporter/pkg/logger"
	"bpftrace_exporter/pkg/ratelimit"
	"bpftrace_exporter/pkg/utils"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

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

func (s *Server) SetUp() error {
	defer func() {
		if s.Error != nil {
			logrus.Errorf("SetUp error: %v", s.Error)
		}
	}()

	// 解析命令行参数
	logrus.Info("Parsing command line arguments...")
	err := s.parse()
	if err != nil {
		logrus.Errorf("Parsing command line arguments failed: %v", err)
		return err
	}

	// 加载基本配置
	logrus.Info("Loading configuration file...")
	err = s.loadConfig()
	if err != nil {
		logrus.Errorf("Loading config file failed: %v", err)
		return err
	}

	// 设置日志
	logrus.Info("Setting up logging...")
	err = s.setupLog()
	if err != nil {
		logrus.Errorf("SetUp error: %v", err)
		return err
	}

	// 加载BPFTrace配置文件
	logrus.Infof("Loading BPFTrace configuration from %s...", *exporter.Configfile)
	settings, err := config.LoadConfigFile(*exporter.Configfile)
	if err != nil {
		logrus.Warnf("Failed to load config file for BPFTrace settings: %v", err)
		logrus.Info("Using default or command-line settings")
	} else {
		// 使用配置文件中的设置
		s.ExporterConfig = *settings
		logrus.Infof("Successfully loaded BPFTrace settings from config file: %+v", *settings)
	}

	// 设置命令行参数（可能会覆盖配置文件中的值）
	logrus.Info("Applying command-line arguments...")
	s.setupCmdArg()

	// 设置HTTP服务器
	logrus.Info("Setting up HTTP server...")
	err = s.setupHttpServer()
	if err != nil {
		logrus.Errorf("SetUp error: %v", err)
		return err
	}

	// 初始化 BpftraceExporter
	logrus.Info("Initializing BpftraceExporter...")
	err = s.initializeBpftraceExporter()
	if err != nil {
		logrus.Errorf("Failed to initialize BpftraceExporter: %v", err)
		return err
	}

	logrus.Info("Setup completed successfully")
	return nil
}

func (s *Server) setupLog() error {
	size, err := humanize.ParseBytes(s.CommonConfig.Logging.MaxSize)
	if err != nil {
		logrus.Errorf("Parsing log size failed: %v", err)
		return err
	}
	logConfig := logger.NewConfig(s.CommonConfig.Logging.Level, s.CommonConfig.Logging.LogPath, safeUint64ToInt64(size), s.CommonConfig.Logging.MaxAge)
	logger.Init(logConfig)
	return nil
}

func (s *Server) setupCmdArg() {
	logrus.Infof("Current config before command line override - BpftracePath: %s, ScriptPath: %s, VarDefs: %s",
		s.ExporterConfig.BpftracePath, s.ExporterConfig.ScriptPath, s.ExporterConfig.VarDefs)

	if config.BpftracePath != nil && *config.BpftracePath != "" {
		logrus.Infof("Using command-line parameter to override BpftracePath: %s", *config.BpftracePath)
		s.ExporterConfig.BpftracePath = *config.BpftracePath
	}

	if config.ScriptPath != nil && *config.ScriptPath != "" {
		logrus.Infof("Using command-line parameter to override ScriptPath: %s", *config.ScriptPath)
		s.ExporterConfig.ScriptPath = *config.ScriptPath
	}

	if config.VarDefs != nil && *config.VarDefs != "" {
		logrus.Infof("Using command-line parameter to override VarDefs: %s", *config.VarDefs)
		s.ExporterConfig.VarDefs = *config.VarDefs
	}

	logrus.Infof("Final config - BpftracePath: %s, ScriptPath: %s, VarDefs: %s",
		s.ExporterConfig.BpftracePath, s.ExporterConfig.ScriptPath, s.ExporterConfig.VarDefs)
}

func (s *Server) healthzHandler(w http.ResponseWriter, r *http.Request) {
	// 构造健康检查响应
	type healthzResponse struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	response := healthzResponse{
		Status:  "ok",
		Message: fmt.Sprintf("%s is running normally.", s.getName()),
	}

	// 设置响应头为 JSON 格式
	w.Header().Set("Content-Type", "application/json")

	// 使用缓冲区编码 JSON 数据，避免部分写入问题
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(response); err != nil {
		// 记录详细的错误日志，包括请求上下文
		logrus.WithFields(logrus.Fields{
			"method": r.Method,
			"path":   r.URL.Path,
			"error":  err,
		}).Error("Failed to encode healthz response")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 写入状态码并发送响应体
	w.WriteHeader(http.StatusOK)
	if _, err := buf.WriteTo(w); err != nil {
		// 记录写入失败的日志
		logrus.WithFields(logrus.Fields{
			"method": r.Method,
			"path":   r.URL.Path,
			"error":  err,
		}).Error("Failed to write healthz response to client")
	}
}

// 获取 Name 字段的线程安全方法
func (s *Server) getName() string {
	// s.mu.RLock()
	// defer s.mu.RUnlock()
	return s.Name
}

func (s *Server) setupHttpServer() error {
	// 确保 exporter.RegisterPrometheus 被调用
	exporter.RegisterPrometheus(s.promReg)

	mux := http.NewServeMux()
	mux.Handle(s.CommonConfig.MetricsPath, promhttp.HandlerFor(s.promReg, promhttp.HandlerOpts{}))

	// 注册健康检查接口
	mux.HandleFunc("/healthz", s.healthzHandler)

	// 原有的路由注册逻辑

	if *UseRatelimit {
		rateLimiter, err := ratelimit.NewRateLimiter(*rateLimitInterval, *rateLimitSize)
		if err != nil {
			logrus.Errorf("ratelimit middleware init error: %v", err)
		}
		s.Use(Ratelimit(rateLimiter))
	}
	addr := fmt.Sprintf("%s:%d", s.CommonConfig.Address, s.CommonConfig.Port)
	schema := "http"
	fmt.Fprintf(os.Stdout, "Listening and serving %s on [%s://%s]\n", s.Name, schema, addr)
	server := &http.Server{
		Addr:        addr,
		Handler:     mux,
		ReadTimeout: 15 * time.Second,
	}
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
	favicon := NewFavicon()
	mux.Handle("/favicon.ico", favicon)
	s.server = server
	logrus.Infof("Server is running on %s", addr)
	if err != nil {
		logrus.Errorf("Configuring the exporter failed: %v", err)
		return err
	}
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
	go utils.HandleSignals(s.Exit)
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
	logger.LogOutput("Shutting down server...")
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
	if exporter.Configfile == nil || *exporter.Configfile == "" {
		*exporter.Configfile = "/etc/uos-exporter/bpftrace-exporter.yaml"
		logrus.Warnf("Config file path is empty, using default: %s", *exporter.Configfile)
	}
	return nil
}

func (s *Server) loadConfig() error {
	if exporter.Configfile == nil || *exporter.Configfile == "" {
		logrus.Error("Config file path is empty")
		return fmt.Errorf("config file path is empty")
	}

	content, err := os.ReadFile(*exporter.Configfile)
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
	logrus.Infof("Loaded config file from: %s", *exporter.Configfile)
	return nil
}

func (s *Server) initializeBpftraceExporter() error {
	// 获取注册的 metrics
	metrics := exporter.GetRegistry().GetMetrics()

	// 检查配置是否有效
	logrus.Infof("Checking bpftrace configuration - BpftracePath: %s, ScriptPath: %s",
		s.ExporterConfig.BpftracePath, s.ExporterConfig.ScriptPath)

	// 如果脚本路径为空，尝试从配置文件中读取
	if s.ExporterConfig.ScriptPath == "" {
		logrus.Warn("Script path is empty, checking config file...")

		// 读取配置文件内容以检查script_path是否存在
		content, err := os.ReadFile(*exporter.Configfile)
		if err == nil {
			var configMap map[string]interface{}
			err = yaml.Unmarshal(content, &configMap)
			if err == nil {
				if scriptPath, ok := configMap["script_path"].(string); ok && scriptPath != "" {
					logrus.Infof("Found script_path in config file: %s", scriptPath)
					s.ExporterConfig.ScriptPath = scriptPath
				}
			}
		}
	}

	// 再次检查脚本路径
	if s.ExporterConfig.ScriptPath == "" {
		return fmt.Errorf("script path is required, provide it via --script.path flag or script_path in config file")
	}

	for _, metric := range metrics {
		// 使用反射获取类型名
		metricType := reflect.TypeOf(metric).String()
		if strings.HasSuffix(metricType, "BpftraceExporter") {
			logrus.Infof("Found BpftraceExporter metric: %s", metricType)
			logrus.Infof("Initializing BpftraceExporter with script: %s", s.ExporterConfig.ScriptPath)

			// 检查脚本文件是否存在
			if _, err := os.Stat(s.ExporterConfig.ScriptPath); os.IsNotExist(err) {
				return fmt.Errorf("script file does not exist: %s", s.ExporterConfig.ScriptPath)
			}

			// 尝试调用Initialize方法
			initMethod := reflect.ValueOf(metric).MethodByName("Initialize")
			if !initMethod.IsValid() {
				return fmt.Errorf("BpftraceExporter does not have Initialize method")
			}

			// 调用Initialize方法
			results := initMethod.Call([]reflect.Value{
				reflect.ValueOf(s.ExporterConfig.BpftracePath),
				reflect.ValueOf(s.ExporterConfig.ScriptPath),
				reflect.ValueOf(s.ExporterConfig.VarDefs),
			})

			// 检查是否有错误返回
			if !results[0].IsNil() {
				return fmt.Errorf("failed to initialize BpftraceExporter: %v", results[0].Interface())
			}

			logrus.Info("BpftraceExporter initialized successfully")
			return nil
		}
	}

	return fmt.Errorf("no BpftraceExporter found in registered metrics")
}

func safeUint64ToInt64(value uint64) int64 {
	if value > math.MaxInt64 {
		return int64(math.MaxInt64)
	}
	return int64(value)
}

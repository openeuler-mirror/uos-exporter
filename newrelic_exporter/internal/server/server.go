package server

import (
	"context"
	"newrelic_exporter/config"
	"newrelic_exporter/internal/exporter"
	"newrelic_exporter/internal/metrics"
	"newrelic_exporter/pkg/logger"
	"newrelic_exporter/pkg/ratelimit"
	"newrelic_exporter/pkg/utils"
	"newrelic_exporter/pkg/cmdline"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
	"encoding/json"
	"bytes"
	"github.com/dustin/go-humanize"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"strings"
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
	err := s.parse()
	if err != nil {
		logrus.Errorf("Parsing command line arguments failed: %v", err)
		return err
	}
	err = s.loadConfig()
	if err != nil {
		logrus.Errorf("Loading config file failed: %v", err)
		return err
	}
	err = s.setupLog()
	if err != nil {
		logrus.Errorf("SetUp error: %v", err)
		return err
	}

	err = s.setupHttpServer()
	if err != nil {
		logrus.Errorf("SetUp error: %v", err)
		return err
	}
	err = exporter.Unpack(&s.ExporterConfig)
	if err != nil {
		logrus.Error("Failed to unpack config: ", err)
		logrus.Info("Use default config")
	}
	if config.ScrapeUrl != nil {
		logrus.Info("Using command-line parameters to override configuration parameters")
		s.ExporterConfig.ScrapeUri = *config.ScrapeUrl
	}
    
	// 初始化 NewRelic API，确保在配置加载后
	err = metrics.InitAPI(&s.CommonConfig.NewRelic)
	if err != nil {
		logrus.Errorf("初始化 NewRelic API 失败: %v", err)
		return err
	}

	return nil
}

func (s *Server) setupLog() error {
	size, err := humanize.ParseBytes(s.CommonConfig.Logging.MaxSize)
	if err != nil {
		logrus.Errorf("Parsing log size failed: %v", err)
		return err
	}
	// 安全转换：检查是否超出 int64 范围
	var safeSize int64
	if size > uint64(^uint64(0)>>1) {
		logrus.Warnf("日志文件大小 %d 超出 int64 最大值，将使用最大安全值", size)
		safeSize = int64(^uint64(0) >> 1) // int64 最大值
	} else {
		safeSize = int64(size)
	}

	logConfig := logger.NewConfig(s.CommonConfig.Logging.Level, s.CommonConfig.Logging.LogPath, safeSize, s.CommonConfig.Logging.MaxAge)
	logger.Init(logConfig)
	return nil
}

func (s *Server) setupCmdArg() {
	if config.ScrapeUrl != nil {
		logrus.Info("Using command-line parameters to override configuration parameters")
		s.ExporterConfig.ScrapeUri = *config.ScrapeUrl
	}
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
	// 使用统一的 kingpin 实例解析命令行参数
	_, err := cmdline.App.Parse(os.Args[1:])
	return err
}

func (s *Server) loadConfig() error {
	logrus.Infof("正在加载配置文件: %s", *exporter.Configfile)
	
	if !utils.FileExists(*exporter.Configfile) {
		logrus.Warnf("配置文件不存在: %s", *exporter.Configfile)
		
		// 检查相对路径
		relativeConfig := "./config/newrelic-exporter.yaml"
		logrus.Infof("尝试相对路径配置文件: %s", relativeConfig)
		
		if utils.FileExists(relativeConfig) {
			logrus.Infof("找到相对路径配置文件，将使用: %s", relativeConfig)
			*exporter.Configfile = relativeConfig
		} else {
			logrus.Warn("相对路径配置文件也不存在，将使用默认配置")
			return nil
		}
	}
	
	content, err := os.ReadFile(*exporter.Configfile)
	if err != nil {
		logrus.Errorf("读取配置文件失败: %v", err)
		logrus.Info("将使用默认配置")
		return nil
	}
	
	// 打印配置文件内容（但不包含敏感信息）
	configStr := string(content)
	logrus.Debug("配置文件内容（加载前）:")
	for _, line := range strings.Split(configStr, "\n") {
		// 隐藏敏感信息
		if strings.Contains(line, "api_key:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				line = parts[0] + ": <hidden>"
			}
		}
		logrus.Debug(line)
	}
	
	err = yaml.Unmarshal(content, &s.CommonConfig)
	if err != nil {
		logrus.Errorf("解析配置文件失败: %v", err)
		logrus.Info("将使用默认配置")
		return nil
	}
	
	// 检查API密钥是否为空，提供警告但不强制修改
	if s.CommonConfig.NewRelic.ApiKey == "" {
		logrus.Warn("警告：NewRelic API 密钥为空！将使用模拟数据模式")
	} else {
		// 移除示例密钥警告
		// if strings.HasSuffix(s.CommonConfig.NewRelic.ApiKey, "FFFFNRAL") {
		//	logrus.Warn("警告：检测到示例API密钥，请使用有效的NewRelic API密钥")
		// } else {
		logrus.Info("NewRelic API 密钥已配置，将使用真实数据模式")
		// }
	}
	
	logrus.Info("配置文件加载成功")
	
	// 确保敏感日志不会被泄露
	logrus.Infof("配置加载完成，NewRelic API 服务器: %s, 服务: %s", 
		s.CommonConfig.NewRelic.ApiServer, 
		s.CommonConfig.NewRelic.Service)
	
	logrus.Infof("已配置监听地址: %s:%d, 指标路径: %s", 
		s.CommonConfig.Address, 
		s.CommonConfig.Port, 
		s.CommonConfig.MetricsPath)
	
	return nil
}
// Part 2 commit for newrelic_exporter/internal/server/server.go

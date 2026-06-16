package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"prometheus_html_exporter/config"
	"prometheus_html_exporter/internal/exporter"
	"prometheus_html_exporter/internal/metrics"
	_ "prometheus_html_exporter/internal/metrics"
	"prometheus_html_exporter/pkg/logger"
	"prometheus_html_exporter/pkg/ratelimit"
	"prometheus_html_exporter/pkg/utils"
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
	return nil
}

func (s *Server) setupLog() error {
	size, err := humanize.ParseBytes(s.CommonConfig.Logging.MaxSize)
	if err != nil {
		logrus.Errorf("Parsing log size failed: %v", err)
		return err
	}
	var logSize int64
	if size > math.MaxInt64 {
		// Log a warning and use a safe default
		log.Printf("Warning: log size %d exceeds maximum, using default", size)
		logSize = 100 * 1024 * 1024 // 100MB default
	} else {
		logSize = int64(size)
	}
	logConfig := logger.NewConfig(s.CommonConfig.Logging.Level, s.CommonConfig.Logging.LogPath, logSize, s.CommonConfig.Logging.MaxAge)
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

// probeHandler 处理/probe端点的请求
func (s *Server) probeHandler(w http.ResponseWriter, r *http.Request) {
	// 使用配置文件处理
	configPath := r.URL.Query().Get("config")
	if configPath == "" {
		// 如果未提供config参数，使用主配置文件
		configPath = *exporter.Configfile
		logrus.Infof("未提供config参数，使用主配置文件: %s", configPath)
	}

	// 使用metrics包中的ProbeHandler处理请求
	metrics.ProbeHandler(w, r, configPath)
}

// probeFormHandler 处理/probe_form端点的请求，显示一个HTML表单
func (s *Server) probeFormHandler(w http.ResponseWriter, r *http.Request) {
	// HTML表单
	html := `<!DOCTYPE html>
<html>
<head>
    <title>HTML Exporter Probe Tool</title>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        body {
            font-family: 'Arial', sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 900px;
            margin: 0 auto;
            padding: 20px;
            background: linear-gradient(45deg, #f5f7fa, #c3cfe2);
            min-height: 100vh;
        }
        h1 {
            color: #2c3e50;
            text-align: center;
            margin-bottom: 30px;
            padding-bottom: 10px;
            border-bottom: 2px solid #3498db;
        }
        .container {
            background: #fff;
            border-radius: 8px;
            padding: 30px;
            box-shadow: 0 10px 30px rgba(0, 0, 0, 0.1);
        }
        form {
            display: flex;
            flex-direction: column;
        }
        .form-group {
            margin-bottom: 20px;
        }
        label {
            font-weight: bold;
            margin-bottom: 8px;
            display: block;
            color: #2c3e50;
        }
        input[type="text"], textarea, select {
            width: 100%;
            padding: 12px;
            border: 1px solid #ddd;
            border-radius: 4px;
            font-size: 16px;
            transition: border 0.3s;
        }
        input[type="text"]:focus, textarea:focus, select:focus {
            border-color: #3498db;
            outline: none;
        }
        button {
            background: #3498db;
            color: white;
            border: none;
            padding: 12px 20px;
            font-size: 16px;
            border-radius: 4px;
            cursor: pointer;
            transition: background 0.3s;
            align-self: flex-start;
        }
        button:hover {
            background: #2980b9;
        }
        .preview {
            margin-top: 30px;
            padding-top: 20px;
            border-top: 1px solid #eee;
        }
        .help-text {
            font-size: 14px;
            color: #7f8c8d;
            margin-top: 5px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>HTML Exporter Probe Tool</h1>
        <p>使用这个工具可以从HTML页面中抓取数值并转换为Prometheus指标。</p>
        
        <form action="/probe" method="get" target="_blank">
            <div class="form-group">
                <label for="url">目标URL:</label>
                <input type="text" id="url" name="target_url" placeholder="http://example.com" required>
                <div class="help-text">要抓取的HTML页面URL</div>
            </div>
            
            <div class="form-group">
                <label for="selector">XPath选择器:</label>
                <input type="text" id="selector" name="selector" placeholder="//div[@class='price']" required>
                <div class="help-text">用于从HTML中提取数值的XPath选择器</div>
            </div>
            
            <div class="form-group">
                <label for="metric_name">指标名称:</label>
                <input type="text" id="metric_name" name="metric_name" placeholder="example_price" required>
                <div class="help-text">Prometheus指标名称，将自动添加前缀</div>
            </div>
            
            <div class="form-group">
                <label for="metric_help">指标描述:</label>
                <input type="text" id="metric_help" name="metric_help" placeholder="Price extracted from example.com" required>
            </div>
            
            <div class="form-group">
                <label for="metric_type">指标类型:</label>
                <select id="metric_type" name="metric_type">
                    <option value="gauge">Gauge</option>
                    <option value="counter">Counter</option>
                </select>
            </div>
            
            <div class="form-group">
                <label for="decimal_separator">小数点分隔符:</label>
                <input type="text" id="decimal_separator" name="decimal_separator" value="." maxlength="1">
            </div>
            
            <div class="form-group">
                <label for="thousands_separator">千位分隔符:</label>
                <input type="text" id="thousands_separator" name="thousands_separator" value="," maxlength="1">
            </div>
            
            <button type="submit">抓取数据</button>
        </form>
        
        <div class="preview">
            <h3>或者直接使用API:</h3>
            <code>
                GET /probe?target_url=http://example.com&selector=//div[@class='price']&metric_name=example_price
            </code>
        </div>
    </div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}

func (s *Server) setupHttpServer() error {
	// 确保 exporter.RegisterPrometheus 被调用
	exporter.RegisterPrometheus(s.promReg)

	mux := http.NewServeMux()
	mux.Handle(s.CommonConfig.MetricsPath, promhttp.HandlerFor(s.promReg, promhttp.HandlerOpts{}))

	// 注册健康检查接口
	mux.HandleFunc("/healthz", s.healthzHandler)

	// 注册probe接口
	mux.HandleFunc("/probe", s.probeHandler)

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
				Text:    "Probe",
				Address: "/probe",
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
	return nil
}

func (s *Server) loadConfig() error {
	logrus.Infof("正在尝试从 %s 加载配置...", *exporter.Configfile)
	content, err := os.ReadFile(*exporter.Configfile)
	if err != nil {
		logrus.Errorf("无法读取配置文件: %v", err)
		logrus.Info("使用默认配置")
		// 打印出默认配置
		logrus.Infof("默认配置: Address=%s, Port=%d, MetricsPath=%s",
			s.CommonConfig.Address, s.CommonConfig.Port, s.CommonConfig.MetricsPath)
		return nil
	}
	err = yaml.Unmarshal(content, &s.CommonConfig)
	if err != nil {
		logrus.Errorf("解析配置文件失败: %v", err)
		logrus.Info("使用默认配置")
		return nil
	}
	logrus.Infof("从 %s 加载配置成功", *exporter.Configfile)
	logrus.Infof("配置: Address=%s, Port=%d, MetricsPath=%s",
		s.CommonConfig.Address, s.CommonConfig.Port, s.CommonConfig.MetricsPath)
	return nil
}

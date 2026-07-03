package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"smtp_exporter/config"
	"smtp_exporter/internal/exporter"
	"smtp_exporter/internal/metrics"
	_ "smtp_exporter/internal/metrics"
	"smtp_exporter/pkg/logger"
	"smtp_exporter/pkg/ratelimit"
	"smtp_exporter/pkg/utils"
	"strconv"
	"sync"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/dustin/go-humanize"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var (
	defaultSeverVersion = "1.0.0"
)

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
	config         *config.SafeConfig
}

// 创建一个SafeConfig实例
var sc = &config.SafeConfig{
	C: &config.Config{},
}

func NewServer(name, version string, cfg *config.SafeConfig) *Server {
	if version == "" {
		version = defaultSeverVersion
	}
	s := &Server{
		Name:         name,
		Version:      version,
		CommonConfig: exporter.DefaultConfig,
		promReg:      prometheus.NewRegistry(),
		ExitSignal:   make(chan struct{}),
		config:       cfg,
	}
	// 全局sc变量与传入的cfg保持一致
	sc = cfg
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

	// 加载SMTP配置文件
	if err := s.config.ReloadConfig(*config.ConfigFile); err != nil {
		logrus.Errorf("Error loading SMTP config: %v", err)
		metrics.UpdateConfigReloadMetrics(false)
		return err
	}
	logrus.Infof("Loaded SMTP config file: %s", *config.ConfigFile)
	metrics.UpdateConfigReloadMetrics(true)

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

	// 创建探针处理函数
	probeHandler := func(w http.ResponseWriter, r *http.Request) {
		// 从查询参数中获取目标地址和模块名
		target := r.URL.Query().Get("target")
		moduleName := r.URL.Query().Get("module")

		// 如果没有提供模块名，使用默认值
		if moduleName == "" {
			moduleName = "default"
		}

		// 检查模块是否存在
		module, ok := sc.GetModule(moduleName)
		if !ok {
			// 增加未知模块计数器
			metrics.ModuleUnknownCounter.Inc()
			logrus.Warnf("Unknown module requested: %s", moduleName)
			http.Error(w, fmt.Sprintf("Unknown module %q", moduleName), http.StatusBadRequest)
			return
		}

		// 设置超时
		timeoutSeconds := 10.0
		if module.Timeout > 0 {
			timeoutSeconds = module.Timeout.Seconds()
		}

		ctx, cancel := context.WithTimeout(r.Context(), time.Duration(timeoutSeconds*float64(time.Second)))
		defer cancel()

		// 创建新的注册表
		registry := prometheus.NewRegistry()

		// 重置指标
		metrics.ProbeSuccess.Set(0)
		metrics.ProbeDuration.Set(0)
		metrics.ProbeMessageSent.Set(0)
		metrics.ProbeSMTPStatusCode.Set(-1)
		metrics.ProbeSMTPEnhancedStatusCode.Set(-1)
		metrics.ProbeIsTLS.Set(0)
		metrics.ProbeTLSCertExpire.Set(0)
		metrics.ProbeTLSVersion.Reset()
		metrics.ProbeTLSInfo.Reset()

		// 执行探测
		start := time.Now()
		success := probeSMTP(ctx, target, &module)
		duration := time.Since(start).Seconds()

		// 更新指标
		metrics.ProbeDuration.Set(duration)
		metrics.ProbeSuccess.Set(boolToFloat(success))

		// 注册所有指标
		registry.MustRegister(metrics.ProbeMessageSent)
		registry.MustRegister(metrics.ProbeTLSVersion)
		registry.MustRegister(metrics.ProbeIsTLS)
		registry.MustRegister(metrics.ProbeTLSCertExpire)
		registry.MustRegister(metrics.ProbeTLSInfo)
		registry.MustRegister(metrics.ProbeSMTPStatusCode)
		registry.MustRegister(metrics.ProbeSMTPEnhancedStatusCode)
		registry.MustRegister(metrics.ProbeSuccess)
		registry.MustRegister(metrics.ProbeDuration)
		registry.MustRegister(metrics.ProbeDNSLookupTime)
		registry.MustRegister(metrics.ProbeIPProtocol)
		registry.MustRegister(metrics.ProbeIPAddrHash)

		// 使用Prometheus处理程序处理请求
		h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)
	}

	// 创建全局指标处理函数
	metricsHandler := func(w http.ResponseWriter, r *http.Request) {
		// 创建新的注册表
		registry := prometheus.NewRegistry()

		// 注册全局指标
		registry.MustRegister(metrics.ModuleUnknownCounter)
		registry.MustRegister(metrics.BuildInfoMetric)
		registry.MustRegister(metrics.ConfigReloadSuccessMetric)
		registry.MustRegister(metrics.ConfigReloadTimestampMetric)

		// 使用Prometheus处理程序处理请求
		h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)
	}

	// 分别注册 /metrics 和 /probe 路径
	mux.HandleFunc(s.CommonConfig.MetricsPath, metricsHandler)
	mux.HandleFunc("/probe", probeHandler)

	// 注册健康检查接口
	mux.HandleFunc("/healthz", s.healthzHandler)

	// 原有的路由注册逻辑

	if UseRatelimit != nil && *UseRatelimit {
		limiter, err := ratelimit.NewRateLimiter(*rateLimitInterval, *rateLimitSize)
		if err != nil {
			logrus.Errorf("ratelimit middleware init error: %v", err)
		} else {
			s.Use(Ratelimit(limiter))
		}
	}

	// 使用配置中的地址和端口
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
			{
				Text:    "Configuration",
				Address: "/config",
			},
			{
				Text:    "Probe smtp.163.com for tls_163",
				Address: "/probe?target=smtp.163.com:465&module=tls_163",
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

	// 添加配置信息展示页面
	mux.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		configHandler(w, r, sc)
	})

	favicon := NewFavicon()
	mux.Handle("/favicon.ico", favicon)
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
	// 检查配置文件路径是否有效
	if *exporter.Configfile == "" {
		logrus.Warn("Config file path is empty, using default configuration")
		return nil
	}

	// 检查文件是否存在
	if _, err := os.Stat(*exporter.Configfile); os.IsNotExist(err) {
		logrus.Warnf("Config file %s does not exist, using default configuration", *exporter.Configfile)
		return nil
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

	// 获取SMTP配置中的设置
	smtpConfig, err := config.LoadConfig(*config.ConfigFile)
	if err == nil && smtpConfig != nil {
		// 同步SMTP配置中的地址和端口到CommonConfig
		if smtpConfig.Address != "" {
			s.CommonConfig.Address = smtpConfig.Address
		}
		if smtpConfig.Port != 0 {
			s.CommonConfig.Port = smtpConfig.Port
		}
		if smtpConfig.MetricsPath != "" {
			s.CommonConfig.MetricsPath = smtpConfig.MetricsPath
		}
	}

	logrus.Infof("Loaded config file from: %s", *exporter.Configfile)
	logrus.Info("CommonConfig file loaded")
	return nil
}

func probeSMTP(ctx context.Context, target string, module *config.Module) bool {
	// 设置目标地址和模块名
	metrics.SetTarget(target, module.SMTP.EHLO)

	// 直接调用SmtpProbeInternal函数进行SMTP探测
	return metrics.SmtpProbeInternal(ctx, target, &module.SMTP)
}

func getTimeout(r *http.Request, lv float64, v float64) (timeoutSeconds float64, err error) {
	// 如果模块中指定了超时，使用模块的超时
	if lv != 0 {
		return lv, nil
	}
	// 否则使用请求头中的超时
	if v := r.Header.Get("X-Prometheus-Scrape-Timeout-Seconds"); v != "" {
		var err error
		timeoutSeconds, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, err
		}
	}
	if timeoutSeconds == 0 {
		timeoutSeconds = 10
	}
	return timeoutSeconds, nil
}

func boolToFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// configHandler 用于展示当前配置信息
func configHandler(w http.ResponseWriter, r *http.Request, sc *config.SafeConfig) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// 创建HTML页面
	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SMTP Exporter Configuration</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 0;
            padding: 20px;
            color: #333;
        }
        h1 {
            color: #1976D2;
            border-bottom: 1px solid #ddd;
            padding-bottom: 10px;
        }
        h2 {
            color: #2196F3;
            margin-top: 20px;
        }
        pre {
            background-color: #f5f5f5;
            padding: 15px;
            border-radius: 4px;
            overflow-x: auto;
        }
        code {
            font-family: 'Courier New', monospace;
        }
        .module {
            margin-bottom: 30px;
            border: 1px solid #e0e0e0;
            border-radius: 4px;
            padding: 15px;
        }
        .module-name {
            font-weight: bold;
            font-size: 1.2em;
            margin-bottom: 10px;
        }
        .hidden {
            color: #999;
            font-style: italic;
        }
    </style>
</head>
<body>
    <h1>SMTP Exporter Configuration</h1>`

	// 读取配置信息
	sc.RLock()
	defer sc.RUnlock()

	if sc.C == nil {
		html += `<p>配置未加载或无效</p>`
	} else {
		html += fmt.Sprintf(`
    <h2>全局设置</h2>
    <pre><code>地址: %s
端口: %d
指标路径: %s
</code></pre>`, sc.C.Address, sc.C.Port, sc.C.MetricsPath)

		html += `<h2>模块配置</h2>`

		if len(sc.C.Modules) == 0 {
			html += `<p>未配置任何模块</p>`
		} else {
			for name, module := range sc.C.Modules {
				html += fmt.Sprintf(`<div class="module">
    <div class="module-name">%s</div>
    <pre><code>探测器: %s
超时: %s
SMTP设置:
  首选IP协议: %s
  IP协议回退: %t
  TLS模式: %s
  EHLO: %s`, name, module.Prober, module.Timeout,
					module.SMTP.IPProtocol,
					module.SMTP.IPProtocolFallback,
					module.SMTP.TLS,
					module.SMTP.EHLO)

				if module.SMTP.MailFrom != "" {
					html += fmt.Sprintf(`
  发件人: %s`, module.SMTP.MailFrom)
				}

				if module.SMTP.MailTo != "" {
					html += fmt.Sprintf(`
  收件人: %s`, module.SMTP.MailTo)
				}

				if module.SMTP.Auth.Username != "" {
					html += fmt.Sprintf(`
  认证:
    用户名: %s
    密码: <hidden>`, module.SMTP.Auth.Username)
				}

				html += `
</code></pre>
</div>`
			}
		}
	}

	html += `
</body>
</html>`

	_, _ = w.Write([]byte(html))
}

package exporter

import (
	"bytes"
	"dhcpd_leases_exporter/internal/metrics"
	"dhcpd_leases_exporter/pkg/logger"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"text/template"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var (
	listenAddress = kingpin.Flag("web.listen-address", "Address to listen on for web interface and telemetry.").Default(":8090").String()
	metricsPath   = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
	debugMode     = kingpin.Flag("debug", "Enable debug mode").Bool()
)

// Run 启动导出器
func Run(name string, version string) error {
	// 解析命令行参数
	kingpin.Version(fmt.Sprintf("%s version %s", name, version))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	// 设置日志级别
	if *debugMode {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.Debug("调试模式已启用")
	}

	// 初始化日志
	logger.InitDefaultLog()

	// 打印版本信息
	logrus.Infof("%s version %s", name, version)

	// 加载配置文件
	config := DefaultConfig
	err := loadConfigFile(&config)
	if err != nil {
		logrus.Warnf("无法加载配置文件: %v, 使用默认配置", err)
	}

	// 命令行参数覆盖配置文件设置
	address := fmt.Sprintf("%s:%d", config.Address, config.Port)
	if *listenAddress != ":8090" {
		address = *listenAddress
	}

	metricsPathValue := config.MetricsPath
	if *metricsPath != "/metrics" {
		metricsPathValue = *metricsPath
	}

	// 启动指标服务器
	return StartMetricsServer(name, version, address, metricsPathValue)
}

// 加载配置文件
func loadConfigFile(config *Config) error {
	// 尝试读取配置文件
	content, err := os.ReadFile(*Configfile)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	// 解析配置文件
	err = yaml.Unmarshal(content, config)
	if err != nil {
		return fmt.Errorf("解析配置文件失败: %v", err)
	}

	logrus.Infof("成功从 %s 加载配置", *Configfile)
	return nil
}

// 首页配置
type LandingPageConfig struct {
	CSS     string
	Name    string
	Links   []LandingPageLink
	Version string
}

type LandingPageLink struct {
	Address string
	Text    string
}

// MetricsServer 表示指标服务器
type MetricsServer struct {
	name        string
	version     string
	address     string
	metricsPath string
}

// NewMetricsServer 创建一个新的指标服务器
func NewMetricsServer(name string, version string, address string, metricsPath string) *MetricsServer {
	return &MetricsServer{
		name:        name,
		version:     version,
		address:     address,
		metricsPath: metricsPath,
	}
}

// 健康检查处理函数
func (s *MetricsServer) healthzHandler(w http.ResponseWriter, r *http.Request) {
	// 构造健康检查响应
	type healthzResponse struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	response := healthzResponse{
		Status:  "ok",
		Message: fmt.Sprintf("%s is running normally.", s.name),
	}

	// 设置响应头为 JSON 格式
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// 编码响应
	_ = json.NewEncoder(w).Encode(response)
}

// 生成首页HTML
func (s *MetricsServer) generateLandingPage() ([]byte, error) {
	const (
		landingPageCSS = `
/* 基础设置 */
body {
    font-family: 'Arial', sans-serif;
    font-size: 18px;
    line-height: 1.8;
    color: #ffffff;
    background: linear-gradient(45deg, #6e7dff, #00b5e2);
    margin: 0;
    padding: 0;
    text-align: center;
    transition: all 0.3s ease-in-out;
    min-height: 100vh;
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    animation: gradientShift 8s infinite alternate ease-in-out;
}

/* 标题样式 */
h1 {
    font-size: 4.5em;
    font-weight: bold;
    margin: 20px 0;
    color: #fff;
    text-shadow: 3px 3px 15px rgba(0, 0, 0, 0.4);
    animation: bounce 3s infinite ease-in-out;
}

h2 {
    font-size: 2.8em;
    font-weight: 500;
    margin: 15px 0;
    color: #f0f0f0;
    text-shadow: 2px 2px 10px rgba(0, 0, 0, 0.3);
    animation: fadeInUp 2s infinite alternate ease-in-out;
}

/* 列表样式 */
ul {
    list-style: none;
    padding: 0;
    margin: 50px 0;
    display: flex;
    flex-direction: column;
    align-items: center;
}

ul li {
    width: 80%;
    max-width: 600px;
    background: rgba(255, 255, 255, 0.15);
    border-radius: 20px;
    margin: 15px 0;
    padding: 25px;
    font-size: 1.6em;
    backdrop-filter: blur(10px);
    transition: transform 0.4s ease, box-shadow 0.4s ease;
    cursor: pointer;
    animation: float 5s infinite alternate ease-in-out;
}

ul li:hover {
    transform: translateY(-5px) scale(1.05);
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.4);
}

/* 链接样式 */
a {
    color: #ffffff;
    font-weight: bold;
    font-size: 1.5em;
    padding: 12px 25px;
    border-radius: 12px;
    background: rgba(255, 255, 255, 0.15);
    display: inline-block;
    transition: background 0.3s ease, transform 0.3s ease;
    animation: pulse 3s infinite ease-in-out;
}

a:hover {
    background: rgba(255, 255, 255, 0.25);
    transform: scale(1.1);
}

/* 段落样式 */
p {
    font-size: 1.5em;
    color: #e0e0e0;
    max-width: 800px;
    margin-top: 20px;
    text-align: center;
}

p.version {
    font-size: 1.3em;
    color: #f0f0f0;
    margin-top: 50px;
}

/* 按钮样式 */
button {
    font-size: 1.4em;
    padding: 12px 30px;
    border: none;
    border-radius: 12px;
    background: rgba(255, 255, 255, 0.2);
    color: white;
    cursor: pointer;
    transition: all 0.3s ease;
    animation: pulse 4s infinite ease-in-out;
}

button:hover {
    background: rgba(255, 255, 255, 0.3);
    transform: scale(1.1);
}

/* 响应式优化 */
@media (max-width: 768px) {
    h1 {
        font-size: 3.5em;
    }
    h2 {
        font-size: 2.2em;
    }
    ul li {
        width: 90%;
        font-size: 1.4em;
    }
}

/* 动画效果 */
@keyframes gradientShift {
    from {
        background: linear-gradient(45deg, #6e7dff, #00b5e2);
    }
    to {
        background: linear-gradient(45deg, #00b5e2, #6e7dff);
    }
}

@keyframes bounce {
    0%, 100% {
        transform: translateY(0);
    }
    50% {
        transform: translateY(-10px);
    }
}

@keyframes fadeInUp {
    0%, 100% {
        opacity: 0.7;
        transform: translateY(10px);
    }
    50% {
        opacity: 1;
        transform: translateY(0);
    }
}

@keyframes float {
    from {
        transform: translateY(0);
    }
    to {
        transform: translateY(-10px);
    }
}

@keyframes pulse {
    0% {
        transform: scale(1);
    }
    50% {
        transform: scale(1.05);
    }
    100% {
        transform: scale(1);
    }
}
`
		landingPageHTML = `
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link rel="icon" type="image/x-icon" href="/favicon.ico">
	<title>{{.Name}}</title>
	<style>{{.CSS}}</style>
</head>
<body>
	<h1>
		{{.Name}}
	</h1>
	<ul>
		{{range .Links}}
			<li>
				<a href="{{.Address}}">
					{{.Text}}
				</a>
			</li>
		{{end}}
	</ul>
	<p class="version">
		Version: {{.Version}}
	</p>
</body>
</html>
`
	)

	// 创建首页配置
	config := LandingPageConfig{
		CSS:     landingPageCSS,
		Name:    s.name,
		Version: s.version,
		Links: []LandingPageLink{
			{
				Text:    "Metrics",
				Address: s.metricsPath,
			},
			{
				Text:    "Health Check",
				Address: "/healthz",
			},
		},
	}

	// 解析模板
	tmpl, err := template.New("landingPage").Parse(landingPageHTML)
	if err != nil {
		return nil, err
	}

	// 执行模板
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Start 启动指标服务器
func (s *MetricsServer) Start() error {
	// 使用默认注册表
	http.Handle(s.metricsPath, promhttp.Handler())

	// 注册健康检查接口
	http.HandleFunc("/healthz", s.healthzHandler)

	// 首页
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		landingPage, err := s.generateLandingPage()
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			logrus.Errorf("生成首页失败: %v", err)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		_, _ = w.Write(landingPage)
	})

	// 图标
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "/usr/share/icons/hicolor/48x48/apps/deepin-system-monitor.png")
	})

	logrus.Infof("启动指标服务器于 %s，指标路径：%s", s.address, s.metricsPath)
	// 创建配置了超时的 HTTP Server
	server := &http.Server{
		Addr:    s.address,
		Handler: nil, // 使用默认的 DefaultServeMux

		// 关键超时设置
		ReadTimeout:       15 * time.Second, // 读取整个请求的最大时间
		WriteTimeout:      15 * time.Second, // 写入响应的最大时间
		IdleTimeout:       60 * time.Second, // 保持空闲连接的最大时间
		ReadHeaderTimeout: 5 * time.Second,  // 读取请求头的最大时间
		MaxHeaderBytes:    1 << 20,          // 1MB 最大请求头大小
	}
	return server.ListenAndServe()
	// return http.ListenAndServe(s.address, nil)
}

// Version 返回导出器的版本信息
func Version() string {
	return fmt.Sprintf("%s version %s", metrics.Name, metrics.Version)
}

// StartMetricsServer 启动指标服务器
func StartMetricsServer(name string, version string, address string, metricsPath string) error {
	server := NewMetricsServer(name, version, address, metricsPath)
	return server.Start()
}

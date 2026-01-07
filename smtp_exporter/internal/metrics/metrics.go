package metrics

import (
	"context"
	"crypto/tls"
	"fmt"
	"hash/fnv"
	"log"
	"net"
	"smtp_exporter/config"
	"time"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	Name    = "smtp_exporter"
	Version = "1.0.0"

	// 构建信息指标
	BuildInfoMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "smtp_exporter_build_info",
			Help: "A metric with a constant '1' value labeled by version, revision, branch, and goversion from which smtp_exporter was built.",
		},
		[]string{"version", "revision", "branch", "goversion"},
	)

	// 配置重载成功指标
	ConfigReloadSuccessMetric = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "smtp_exporter_config_last_reload_successful",
			Help: "smtp exporter config loaded successfully.",
		},
	)

	// 配置重载时间戳指标
	ConfigReloadTimestampMetric = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "smtp_exporter_config_last_reload_success_timestamp_seconds",
			Help: "Timestamp of the last successful configuration reload.",
		},
	)

	// 添加ModuleUnknownCounter指标
	ModuleUnknownCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "smtp_module_unknown_total",
		Help: "Count of unknown modules requested by probes",
	})

	// SMTP探测指标
	ProbeMessageSent = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_message_sent",
		Help: "Indicates if the message was sent successfully",
	})

	ProbeTLSVersion = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "probe_tls_version_info",
			Help: "Contains the TLS version used",
		},
		[]string{"version"},
	)

	ProbeIsTLS = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_smtp_tls",
		Help: "Indicates if TLS was used",
	})

	ProbeTLSCertExpire = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_tls_cert_expire",
		Help: "Returns the TLS cert expire in unixtime",
	})

	ProbeTLSInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "probe_tls_info",
			Help: "Contains certificate information",
		},
		[]string{"fingerprint_sha256"},
	)

	ProbeSMTPStatusCode = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_smtp_status_code",
		Help: "Response smtp status code",
	})

	ProbeSMTPEnhancedStatusCode = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_smtp_enhanced_status_code",
		Help: "Response smtp enhanced status code",
	})

	ProbeSuccess = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_success",
		Help: "Displays whether or not the probe was a success",
	})

	ProbeDuration = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_duration_seconds",
		Help: "Returns how long the probe took to complete in seconds",
	})

	ProbeDNSLookupTime = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_dns_lookup_time_seconds",
		Help: "Returns the time taken for probe dns lookup in seconds",
	})

	ProbeIPProtocol = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_ip_protocol",
		Help: "Specifies whether probe ip protocol IP4 or IP6",
	})

	ProbeIPAddrHash = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_ip_addr_hash",
		Help: "Specifies the hash of IP address. It's useful to detect if the IP address changes.",
	})
)

// 初始化时注册自定义指标
func init() {
	// 注册到默认注册表
	prometheus.DefaultRegisterer.MustRegister(ModuleUnknownCounter)
	prometheus.DefaultRegisterer.MustRegister(ProbeMessageSent)
	prometheus.DefaultRegisterer.MustRegister(ProbeTLSVersion)
	prometheus.DefaultRegisterer.MustRegister(ProbeIsTLS)
	prometheus.DefaultRegisterer.MustRegister(ProbeTLSCertExpire)
	prometheus.DefaultRegisterer.MustRegister(ProbeTLSInfo)
	prometheus.DefaultRegisterer.MustRegister(ProbeSMTPStatusCode)
	prometheus.DefaultRegisterer.MustRegister(ProbeSMTPEnhancedStatusCode)
	prometheus.DefaultRegisterer.MustRegister(ProbeSuccess)
	prometheus.DefaultRegisterer.MustRegister(ProbeDuration)
	prometheus.DefaultRegisterer.MustRegister(ProbeDNSLookupTime)
	prometheus.DefaultRegisterer.MustRegister(ProbeIPProtocol)
	prometheus.DefaultRegisterer.MustRegister(ProbeIPAddrHash)

	// 注册构建信息指标
	prometheus.DefaultRegisterer.MustRegister(BuildInfoMetric)
	BuildInfoMetric.WithLabelValues(Version, "", "", "go1.22.8").Set(1)

	// 注册配置重载指标
	prometheus.DefaultRegisterer.MustRegister(ConfigReloadSuccessMetric)
	prometheus.DefaultRegisterer.MustRegister(ConfigReloadTimestampMetric)
}

// 更新配置重载指标
func UpdateConfigReloadMetrics(success bool) {
	if success {
		ConfigReloadSuccessMetric.Set(1)
		ConfigReloadTimestampMetric.SetToCurrentTime()
	} else {
		ConfigReloadSuccessMetric.Set(0)
	}
}

type baseMetrics struct {
	labels []string
	desc   *prometheus.Desc
}

func NewMetrics(fqname, help string, labels []string) *baseMetrics {
	return &baseMetrics{
		labels: labels,
		desc: prometheus.NewDesc(
			fqname,
			help,
			labels,
			nil),
	}
}

func (c *baseMetrics) collect(ch chan<- prometheus.Metric, value float64, labels []string) {
	ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, value, labels...)
}

// SmtpProbeInternal 执行 SMTP 探测并更新相关指标
func SmtpProbeInternal(ctx context.Context, target string, module *config.SMTPProbe) bool {
	start := time.Now()

	// 设置目标地址
	SetTarget(target, module.EHLO)

	// 解析目标地址
	host, _, err := net.SplitHostPort(target)
	if err != nil {
		log.Printf("Failed to parse target address: %v", err)
		ProbeSuccess.Set(0)
		return false
	}

	// DNS 查询
	dnsStart := time.Now()
	ips, err := net.LookupIP(host)
	if err != nil {
		log.Printf("DNS lookup failed: %v", err)
		ProbeSuccess.Set(0)
		return false
	}
	ProbeDNSLookupTime.Set(time.Since(dnsStart).Seconds())

	// 设置 IP 协议
	if len(ips) > 0 {
		if ips[0].To4() != nil {
			ProbeIPProtocol.Set(4)
		} else {
			ProbeIPProtocol.Set(6)
		}
		// 计算 IP 地址哈希
		h := fnv.New32a()
		_, _ = h.Write(ips[0])
		ProbeIPAddrHash.Set(float64(h.Sum32()))
	}

	// 创建 SMTP 客户端
	var client *smtp.Client
	if module.TLS == "tls" {
		// TLS 连接
		tlsConfig := &tls.Config{
			ServerName: host,
			// #nosec G402
			InsecureSkipVerify: module.TLSConfig.InsecureSkipVerify,
		}
		client, err = smtp.DialTLS(target, tlsConfig)
		if err != nil {
			log.Printf("TLS connection failed: %v", err)
			ProbeSuccess.Set(0)
			return false
		}
		ProbeIsTLS.Set(1)
	} else {
		// 普通连接
		client, err = smtp.Dial(target)
		if err != nil {
			log.Printf("Failed to connect to SMTP server: %v", err)
			ProbeSuccess.Set(0)
			return false
		}
		ProbeIsTLS.Set(0)
	}
	defer client.Close()

	// 发送 EHLO
	if err := client.Hello(module.EHLO); err != nil {
		log.Printf("Failed to send EHLO: %v", err)
		ProbeSuccess.Set(0)
		return false
	}

	// 检查 STARTTLS
	if module.TLS == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); ok {
			tlsConfig := &tls.Config{
				ServerName: host,
				// #nosec G402
				InsecureSkipVerify: module.TLSConfig.InsecureSkipVerify,
			}
			// 使用 NewClientStartTLS 创建带 STARTTLS 的客户端
			conn, err := tls.Dial("tcp", target, tlsConfig)
			if err != nil {
				log.Printf("STARTTLS failed: %v", err)
				ProbeSuccess.Set(0)
				return false
			}
			client = smtp.NewClient(conn)
			ProbeIsTLS.Set(1)
		}
	}

	// 获取 TLS 信息
	if state, ok := client.TLSConnectionState(); ok {
		ProbeTLSVersion.WithLabelValues(fmt.Sprintf("%d", state.Version)).Set(1)
		if len(state.PeerCertificates) > 0 {
			ProbeTLSCertExpire.Set(float64(state.PeerCertificates[0].NotAfter.Unix()))
			fingerprint := getFingerprint(&state)
			ProbeTLSInfo.WithLabelValues(fingerprint).Set(1)
		}
	}

	// 认证
	if module.Auth.Username != "" {
		auth := sasl.NewPlainClient("", module.Auth.Username, string(module.Auth.Password))
		if err := client.Auth(auth); err != nil {
			log.Printf("Authentication failed: %v", err)
			ProbeSuccess.Set(0)
			return false
		}
	}

	// MAIL FROM
	if err := client.Mail(module.MailFrom, &smtp.MailOptions{}); err != nil {
		log.Printf("MAIL FROM failed: %v", err)
		ProbeSuccess.Set(0)
		return false
	}

	// RCPT TO
	if err := client.Rcpt(module.MailTo, &smtp.RcptOptions{}); err != nil {
		log.Printf("RCPT TO failed: %v", err)
		ProbeSuccess.Set(0)
		return false
	}

	// DATA
	wc, err := client.Data()
	if err != nil {
		log.Printf("DATA failed: %v", err)
		ProbeSuccess.Set(0)
		return false
	}

	// 写入邮件内容
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: SMTP Probe\r\n\r\nThis is a test message from the SMTP exporter.", module.MailFrom, module.MailTo)
	if _, err := wc.Write([]byte(msg)); err != nil {
		log.Printf("Failed to write message: %v", err)
		ProbeSuccess.Set(0)
		return false
	}

	// 关闭数据连接
	if err := wc.Close(); err != nil {
		log.Printf("Failed to close data connection: %v", err)
		ProbeSuccess.Set(0)
		return false
	}

	// 更新成功指标
	ProbeSuccess.Set(1)
	ProbeMessageSent.Set(1)

	// 更新探测持续时间
	ProbeDuration.Set(time.Since(start).Seconds())

	return true
}

// 计算TLS证书指纹的辅助函数
func getFingerprint(state *tls.ConnectionState) string {
	if len(state.PeerCertificates) == 0 {
		return ""
	}
	cert := state.PeerCertificates[0]
	return fmt.Sprintf("%x", cert.SerialNumber)
}

// boolToFloat 将布尔值转换为浮点数
func boolToFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
// Part 2 commit for smtp_exporter/internal/metrics/metrics.go

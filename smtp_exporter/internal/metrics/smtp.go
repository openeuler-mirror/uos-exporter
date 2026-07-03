package metrics

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	sasl "github.com/emersion/go-sasl"
	smtp "github.com/emersion/go-smtp"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	pconfig "github.com/prometheus/common/config"
	"github.com/sirupsen/logrus"

	"smtp_exporter/config"
	"smtp_exporter/internal/exporter"
)

// 全局配置
var (
	sc = &config.SafeConfig{
		C: &config.Config{},
	}
	// 添加互斥锁保护并发访问
	probeResultMutex = &sync.Mutex{}
	// 最后一次探测结果缓存
	lastProbeResult = struct {
		target        string
		moduleName    string
		probeResult   bool
		messageSent   bool
		tlsVersion    string
		isTLS         bool
		tlsCertExpire int64
		tlsInfo       string
		statusCode    int
		enhancedCode  int
		duration      float64
		dnsLookupTime float64
		ipProtocol    float64
		ipAddrHash    float64
		timestamp     time.Time
	}{
		timestamp: time.Now().Add(-24 * time.Hour), // 初始化为过期状态
	}
	// 探测结果缓存时间（5秒）
	probeResultCacheDuration = 5 * time.Second
)

// 设置目标地址和模块名
func SetTarget(target, moduleName string) {
	probeResultMutex.Lock()
	defer probeResultMutex.Unlock()

	// 如果目标地址和模块名相同，且缓存未过期，不需要更新
	if target == lastProbeResult.target &&
		moduleName == lastProbeResult.moduleName &&
		time.Since(lastProbeResult.timestamp) < probeResultCacheDuration {
		return
	}

	// 更新目标地址和模块名
	lastProbeResult.target = target
	lastProbeResult.moduleName = moduleName
	// 重置时间戳，使缓存立即过期，强制下次探测
	lastProbeResult.timestamp = time.Now().Add(-24 * time.Hour)
}

// SMTP指标
type SMTPProbeMessageSent struct {
	*baseMetrics
}

type SMTPProbeTLSVersion struct {
	*baseMetrics
}

type SMTPProbeIsTLS struct {
	*baseMetrics
}

type SMTPProbeTLSCertExpire struct {
	*baseMetrics
}

type SMTPProbeTLSInformation struct {
	*baseMetrics
}

type SMTPProbeStatusCode struct {
	*baseMetrics
}

type SMTPProbeEnhancedStatusCode struct {
	*baseMetrics
}

type SMTPProbeSuccess struct {
	*baseMetrics
}

type SMTPProbeDuration struct {
	*baseMetrics
}

type SMTPProbeDNSLookupTime struct {
	*baseMetrics
}

type SMTPProbeIPProtocol struct {
	*baseMetrics
}

type SMTPProbeIPAddrHash struct {
	*baseMetrics
}

// 初始化配置
func InitConfig(configFile string) error {
	logrus.Infof("尝试加载配置文件: %s", configFile)

	// 检查文件是否存在
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// 检查扩展名，尝试替代文件
		if filepath.Ext(configFile) == ".yaml" {
			altFile := configFile[:len(configFile)-5] + ".yml"
			if _, altErr := os.Stat(altFile); os.IsNotExist(altErr) {
				logrus.Warnf("配置文件不存在: %s (也尝试了 %s)", configFile, altFile)
			} else {
				// 找到了替代文件
				logrus.Infof("使用替代配置文件: %s", altFile)
				configFile = altFile
			}
		}
	}

	if err := sc.ReloadConfig(configFile); err != nil {
		logrus.Errorf("加载配置文件失败: %v", err)
		return err
	}
	logrus.Infof("成功加载配置文件: %s", configFile)
	return nil
}

// 注册所有SMTP相关指标
func init() {
	exporter.Register(
		NewSMTPProbeMessageSent("probe_message_sent",
			"Indicates if the message was sent successfully",
			[]string{}))
	exporter.Register(
		NewSMTPProbeTLSVersion("probe_tls_version_info",
			"Contains the TLS version used",
			[]string{"version"}))
	exporter.Register(
		NewSMTPProbeIsTLS("probe_smtp_tls",
			"Indicates if TLS was used",
			[]string{}))
	exporter.Register(
		NewSMTPProbeTLSCertExpire("probe_tls_cert_expire",
			"Returns the TLS cert expire in unixtime",
			[]string{}))
	exporter.Register(
		NewSMTPProbeTLSInformation("probe_tls_info",
			"Contains certificate information",
			[]string{"fingerprint_sha256"}))
	exporter.Register(
		NewSMTPProbeStatusCode("probe_smtp_status_code",
			"Response smtp status code",
			[]string{}))
	exporter.Register(
		NewSMTPProbeEnhancedStatusCode("probe_smtp_enhanced_status_code",
			"Response smtp enhanced status code",
			[]string{}))
	exporter.Register(
		NewSMTPProbeSuccess("probe_success",
			"Displays whether or not the probe was a success",
			[]string{}))
	exporter.Register(
		NewSMTPProbeDuration("probe_duration_seconds",
			"Returns how long the probe took to complete in seconds",
			[]string{}))
	exporter.Register(
		NewSMTPProbeDNSLookupTime("probe_dns_lookup_time_seconds",
			"Returns the time taken for probe dns lookup in seconds",
			[]string{}))
	exporter.Register(
		NewSMTPProbeIPProtocol("probe_ip_protocol",
			"Specifies whether probe ip protocol IP4 or IP6",
			[]string{}))
	exporter.Register(
		NewSMTPProbeIPAddrHash("probe_ip_addr_hash",
			"Specifies the hash of IP address. It's useful to detect if the IP address changes.",
			[]string{}))
}

// 创建指标实例
func NewSMTPProbeMessageSent(fqname, help string, labels []string) *SMTPProbeMessageSent {
	return &SMTPProbeMessageSent{NewMetrics(fqname, help, labels)}
}

func NewSMTPProbeTLSVersion(fqname, help string, labels []string) *SMTPProbeTLSVersion {
	return &SMTPProbeTLSVersion{NewMetrics(fqname, help, labels)}
}

func NewSMTPProbeIsTLS(fqname, help string, labels []string) *SMTPProbeIsTLS {
	return &SMTPProbeIsTLS{NewMetrics(fqname, help, labels)}
}

func NewSMTPProbeTLSCertExpire(fqname, help string, labels []string) *SMTPProbeTLSCertExpire {
	return &SMTPProbeTLSCertExpire{NewMetrics(fqname, help, labels)}
}

func NewSMTPProbeTLSInformation(fqname, help string, labels []string) *SMTPProbeTLSInformation {
	return &SMTPProbeTLSInformation{NewMetrics(fqname, help, labels)}
}

func NewSMTPProbeStatusCode(fqname, help string, labels []string) *SMTPProbeStatusCode {
	return &SMTPProbeStatusCode{NewMetrics(fqname, help, labels)}
}

func NewSMTPProbeEnhancedStatusCode(fqname, help string, labels []string) *SMTPProbeEnhancedStatusCode {
	return &SMTPProbeEnhancedStatusCode{NewMetrics(fqname, help, labels)}
}

func NewSMTPProbeSuccess(fqname, help string, labels []string) *SMTPProbeSuccess {
	return &SMTPProbeSuccess{NewMetrics(fqname, help, labels)}
}

func NewSMTPProbeDuration(fqname, help string, labels []string) *SMTPProbeDuration {
	return &SMTPProbeDuration{NewMetrics(fqname, help, labels)}
}

func NewSMTPProbeDNSLookupTime(fqname, help string, labels []string) *SMTPProbeDNSLookupTime {
	return &SMTPProbeDNSLookupTime{NewMetrics(fqname, help, labels)}
}

func NewSMTPProbeIPProtocol(fqname, help string, labels []string) *SMTPProbeIPProtocol {
	return &SMTPProbeIPProtocol{NewMetrics(fqname, help, labels)}
}

func NewSMTPProbeIPAddrHash(fqname, help string, labels []string) *SMTPProbeIPAddrHash {
	return &SMTPProbeIPAddrHash{NewMetrics(fqname, help, labels)}
}

// 收集指标
func (m *SMTPProbeMessageSent) Collect(ch chan<- prometheus.Metric) {
	// 在收集过程中实现SMTP探测逻辑
	_, messageSent, _, _, _, _, _, _, _, _, _, _ := smtpProbeWithCache()

	// 设置指标值
	if messageSent {
		m.baseMetrics.collect(ch, 1, []string{})
	} else {
		m.baseMetrics.collect(ch, 0, []string{})
	}
}

func (m *SMTPProbeTLSVersion) Collect(ch chan<- prometheus.Metric) {
	_, _, tlsVersion, _, _, _, _, _, _, _, _, _ := smtpProbeWithCache()

	if tlsVersion != "" {
		m.baseMetrics.collect(ch, 1, []string{tlsVersion})
	}
}

func (m *SMTPProbeIsTLS) Collect(ch chan<- prometheus.Metric) {
	_, _, _, isTLS, _, _, _, _, _, _, _, _ := smtpProbeWithCache()

	if isTLS {
		m.baseMetrics.collect(ch, 1, []string{})
	} else {
		m.baseMetrics.collect(ch, 0, []string{})
	}
}

func (m *SMTPProbeTLSCertExpire) Collect(ch chan<- prometheus.Metric) {
	_, _, _, _, tlsCertExpire, _, _, _, _, _, _, _ := smtpProbeWithCache()

	if tlsCertExpire > 0 {
		m.baseMetrics.collect(ch, float64(tlsCertExpire), []string{})
	}
}

func (m *SMTPProbeTLSInformation) Collect(ch chan<- prometheus.Metric) {
	_, _, _, _, _, tlsInfo, _, _, _, _, _, _ := smtpProbeWithCache()

	if tlsInfo != "" {
		m.baseMetrics.collect(ch, 1, []string{tlsInfo})
	}
}

func (m *SMTPProbeStatusCode) Collect(ch chan<- prometheus.Metric) {
	_, _, _, _, _, _, statusCode, _, _, _, _, _ := smtpProbeWithCache()

	m.baseMetrics.collect(ch, float64(statusCode), []string{})
}

func (m *SMTPProbeEnhancedStatusCode) Collect(ch chan<- prometheus.Metric) {
	_, _, _, _, _, _, _, enhancedStatusCode, _, _, _, _ := smtpProbeWithCache()

	m.baseMetrics.collect(ch, float64(enhancedStatusCode), []string{})
}

func (m *SMTPProbeSuccess) Collect(ch chan<- prometheus.Metric) {
	probeSuccess, _, _, _, _, _, _, _, _, _, _, _ := smtpProbeWithCache()

	if probeSuccess {
		m.baseMetrics.collect(ch, 1, []string{})
	} else {
		m.baseMetrics.collect(ch, 0, []string{})
	}
}

func (m *SMTPProbeDuration) Collect(ch chan<- prometheus.Metric) {
	_, _, _, _, _, _, _, _, duration, _, _, _ := smtpProbeWithCache()

	m.baseMetrics.collect(ch, duration, []string{})
}

func (m *SMTPProbeDNSLookupTime) Collect(ch chan<- prometheus.Metric) {
	_, _, _, _, _, _, _, _, _, dnsLookupTime, _, _ := smtpProbeWithCache()

	m.baseMetrics.collect(ch, dnsLookupTime, []string{})
}

func (m *SMTPProbeIPProtocol) Collect(ch chan<- prometheus.Metric) {
	_, _, _, _, _, _, _, _, _, _, ipProtocol, _ := smtpProbeWithCache()

	m.baseMetrics.collect(ch, ipProtocol, []string{})
}

func (m *SMTPProbeIPAddrHash) Collect(ch chan<- prometheus.Metric) {
	_, _, _, _, _, _, _, _, _, _, _, ipHash := smtpProbeWithCache()

	m.baseMetrics.collect(ch, ipHash, []string{})
}

// SMTP探测函数
func smtpProbeWithCache() (bool, bool, string, bool, int64, string, int, int, float64, float64, float64, float64) {
	probeResultMutex.Lock()
	defer probeResultMutex.Unlock()

	// 检查缓存是否有效
	if time.Since(lastProbeResult.timestamp) < probeResultCacheDuration {
		// 使用缓存的结果
		return lastProbeResult.probeResult,
			lastProbeResult.messageSent,
			lastProbeResult.tlsVersion,
			lastProbeResult.isTLS,
			lastProbeResult.tlsCertExpire,
			lastProbeResult.tlsInfo,
			lastProbeResult.statusCode,
			lastProbeResult.enhancedCode,
			lastProbeResult.duration,
			lastProbeResult.dnsLookupTime,
			lastProbeResult.ipProtocol,
			lastProbeResult.ipAddrHash
	}

	// 从设置的目标地址获取
	target := lastProbeResult.target
	if target == "" {
		// 如果没有设置目标地址，使用默认值
		target = "localhost:25"
		logrus.Warn("No target specified, using default target: ", target)
	}

	// 使用设置的模块名，如果没有设置，使用默认值
	moduleName := lastProbeResult.moduleName
	if moduleName == "" {
		moduleName = "default"
	}

	// 获取模块配置
	module, ok := sc.GetModule(moduleName)
	if !ok {
		logrus.Errorf("Module %s not found in configuration, using default module", moduleName)
		module, ok = sc.GetModule("default")
		if !ok {
			logrus.Error("Default module not found in configuration")
			return false, false, "", false, 0, "", -1, -1, 0, 0, 0, 0
		}
	}

	// 初始化返回值
	probeSuccess := false
	messageSent := false
	tlsVersion := ""
	isTLS := false
	tlsCertExpire := int64(0)
	tlsInfo := ""
	statusCode := -1
	statusCodeEnhanced := -1
	duration := 0.0
	dnsLookupTime := 0.0
	var ipProtocol float64 = 0
	var ipAddrHash float64 = 0

	// 创建日志记录器
	logger := log.NewNopLogger()

	// 开始计时
	start := time.Now()

	// 解析目标地址
	targetHost, targetPort, err := net.SplitHostPort(target)
	if err != nil {
		_ = level.Error(logger).Log("msg", "Could not parse target", "err", err)
		return probeSuccess, messageSent, tlsVersion, isTLS, tlsCertExpire, tlsInfo, statusCode, statusCodeEnhanced, duration, dnsLookupTime, ipProtocol, ipAddrHash
	}

	// 选择IP协议
	ip, lookupTime, err := chooseProtocol(context.Background(), module.SMTP.IPProtocol, module.SMTP.IPProtocolFallback, targetHost)
	dnsLookupTime = lookupTime
	if err != nil {
		_ = level.Error(logger).Log("msg", "Error resolving address", "err", err)
		return probeSuccess, messageSent, tlsVersion, isTLS, tlsCertExpire, tlsInfo, statusCode, statusCodeEnhanced, duration, dnsLookupTime, ipProtocol, ipAddrHash
	}

	if ip != nil {
		ipAddrHash = ipHashFunc(ip.IP)
		if ip.IP.To4() != nil {
			ipProtocol = 4
		} else {
			ipProtocol = 6
		}
	}

	// 设置TLS服务器名称
	if len(module.SMTP.TLSConfig.ServerName) == 0 {
		module.SMTP.TLSConfig.ServerName = targetHost
	}

	// 创建SMTP客户端
	c, err := newSMTPClient(context.Background(), module.SMTP, ip, targetPort)
	if err != nil {
		_ = level.Error(logger).Log("msg", "Error creating SMTP client", "err", err)
		return probeSuccess, messageSent, tlsVersion, isTLS, tlsCertExpire, tlsInfo, statusCode, statusCodeEnhanced, duration, dnsLookupTime, ipProtocol, ipAddrHash
	}

	defer func() {
		_ = c.Quit()
	}()

	// 如果使用TLS，获取TLS信息
	if module.SMTP.TLS != "no" {
		state, _ := c.TLSConnectionState()
		isTLS = true
		tlsVersion = getTLSVersion(&state)
		tlsCertExpire = getCertExpiry(&state).Unix()
		tlsInfo = getFingerprintSMTP(&state)
	}

	// 认证
	if len(module.SMTP.Auth.Username) > 0 {
		auth := sasl.NewPlainClient("", module.SMTP.Auth.Username, string(module.SMTP.Auth.Password))
		if err = c.Auth(auth); err != nil {
			smtpErr, _ := err.(*smtp.SMTPError)
			statusCode = smtpErr.Code
			statusCodeEnhanced = smtpErr.EnhancedCode[0]*100 + smtpErr.EnhancedCode[1]*10 + smtpErr.EnhancedCode[2]

			// 检查是否为有效状态码
			for _, validStatusCode := range module.SMTP.ValidStatusCodes {
				if validStatusCode == smtpErr.Code {
					probeSuccess = true
				}
			}

			duration = time.Since(start).Seconds()

			// 更新缓存
			updateProbeResultCache(target, moduleName, probeSuccess, messageSent, tlsVersion, isTLS, tlsCertExpire, tlsInfo, statusCode, statusCodeEnhanced, duration, dnsLookupTime, ipProtocol, ipAddrHash)

			return probeSuccess, messageSent, tlsVersion, isTLS, tlsCertExpire, tlsInfo, statusCode, statusCodeEnhanced, duration, dnsLookupTime, ipProtocol, ipAddrHash
		}
	}

	// MAIL FROM
	if len(module.SMTP.MailFrom) == 0 {
		_ = level.Error(logger).Log("msg", "MAIL FROM in module configuration is empty")
		duration = time.Since(start).Seconds()

		// 更新缓存
		updateProbeResultCache(target, moduleName, probeSuccess, messageSent, tlsVersion, isTLS, tlsCertExpire, tlsInfo, statusCode, statusCodeEnhanced, duration, dnsLookupTime, ipProtocol, ipAddrHash)

		return probeSuccess, messageSent, tlsVersion, isTLS, tlsCertExpire, tlsInfo, statusCode, statusCodeEnhanced, duration, dnsLookupTime, ipProtocol, ipAddrHash
	}

	if err = c.Mail(module.SMTP.MailFrom, nil); err != nil {
		smtpErr, _ := err.(*smtp.SMTPError)
		statusCode = smtpErr.Code
		statusCodeEnhanced = smtpErr.EnhancedCode[0]*100 + smtpErr.EnhancedCode[1]*10 + smtpErr.EnhancedCode[2]

		// 检查是否为有效状态码
		for _, validStatusCode := range module.SMTP.ValidStatusCodes {
			if validStatusCode == smtpErr.Code {
				probeSuccess = true
			}
		}

		duration = time.Since(start).Seconds()

		// 更新缓存
		updateProbeResultCache(target, moduleName, probeSuccess, messageSent, tlsVersion, isTLS, tlsCertExpire, tlsInfo, statusCode, statusCodeEnhanced, duration, dnsLookupTime, ipProtocol, ipAddrHash)

		return probeSuccess, messageSent, tlsVersion, isTLS, tlsCertExpire, tlsInfo, statusCode, statusCodeEnhanced, duration, dnsLookupTime, ipProtocol, ipAddrHash
	}

	// RCPT TO
	if len(module.SMTP.MailTo) == 0 {
		_ = level.Error(logger).Log("msg", "RCPT TO in module configuration is empty")
		duration = time.Since(start).Seconds()

		// 更新缓存
		updateProbeResultCache(target, moduleName, probeSuccess, messageSent, tlsVersion, isTLS, tlsCertExpire, tlsInfo, statusCode, statusCodeEnhanced, duration, dnsLookupTime, ipProtocol, ipAddrHash)

		return probeSuccess, messageSent, tlsVersion, isTLS, tlsCertExpire, tlsInfo, statusCode, statusCodeEnhanced, duration, dnsLookupTime, ipProtocol, ipAddrHash
	}

	if err = c.Rcpt(module.SMTP.MailTo, nil); err != nil {
		smtpErr, _ := err.(*smtp.SMTPError)
		statusCode = smtpErr.Code
		statusCodeEnhanced = smtpErr.EnhancedCode[0]*100 + smtpErr.EnhancedCode[1]*10 + smtpErr.EnhancedCode[2]

		// 检查是否为有效状态码
		for _, validStatusCode := range module.SMTP.ValidStatusCodes {
			if validStatusCode == smtpErr.Code {
				probeSuccess = true
			}
		}

		duration = time.Since(start).Seconds()

		// 更新缓存
		updateProbeResultCache(target, moduleName, probeSuccess, messageSent, tlsVersion, isTLS, tlsCertExpire, tlsInfo, statusCode, statusCodeEnhanced, duration, dnsLookupTime, ipProtocol, ipAddrHash)

		return probeSuccess, messageSent, tlsVersion, isTLS, tlsCertExpire, tlsInfo, statusCode, statusCodeEnhanced, duration, dnsLookupTime, ipProtocol, ipAddrHash
	}

	// DATA
	w, err := c.Data()
	if err != nil {
		smtpErr, _ := err.(*smtp.SMTPError)
		statusCode = smtpErr.Code
		statusCodeEnhanced = smtpErr.EnhancedCode[0]*100 + smtpErr.EnhancedCode[1]*10 + smtpErr.EnhancedCode[2]

		// 检查是否为有效状态码
		for _, validStatusCode := range module.SMTP.ValidStatusCodes {
			if validStatusCode == smtpErr.Code {
				probeSuccess = true
			}
		}

		duration = time.Since(start).Seconds()

		// 更新缓存
		updateProbeResultCache(target, moduleName, probeSuccess, messageSent, tlsVersion, isTLS, tlsCertExpire, tlsInfo, statusCode, statusCodeEnhanced, duration, dnsLookupTime, ipProtocol, ipAddrHash)

		return probeSuccess, messageSent, tlsVersion, isTLS, tlsCertExpire, tlsInfo, statusCode, statusCodeEnhanced, duration, dnsLookupTime, ipProtocol, ipAddrHash
	}

	// 构建邮件内容
	body := module.SMTP.Body
	if len(body) == 0 {
		body = "This is a test mail sent by the smtp_exporter"
	}

	headers := make(map[string]string)
	for k, v := range module.SMTP.Headers {
		headers[strings.Title(strings.ToLower(k))] = v
	}

	_, ok = headers["Subject"]
	if !ok {
		headers["Subject"] = "[smtp_exporter]"
	}

	headers["Message-ID"] = generateMessageID()
	headers["Date"] = time.Now().Format("Mon, _2 Jan 2006 15:04:05 -0700")
	headers["Subject"] = fmt.Sprintf("%s %s", headers["Subject"], uuid.New().String())

	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	if _, err = w.Write([]byte(message)); err != nil {
		_ = level.Error(logger).Log("msg", "Error writing message buffer", "err", err)
		duration = time.Since(start).Seconds()

		// 更新缓存
		updateProbeResultCache(target, moduleName, probeSuccess, messageSent, tlsVersion, isTLS, tlsCertExpire, tlsInfo, statusCode, statusCodeEnhanced, duration, dnsLookupTime, ipProtocol, ipAddrHash)

		return probeSuccess, messageSent, tlsVersion, isTLS, tlsCertExpire, tlsInfo, statusCode, statusCodeEnhanced, duration, dnsLookupTime, ipProtocol, ipAddrHash
	}

	if err = w.Close(); err != nil {
		_ = level.Error(logger).Log("msg", "Error closing message buffer", "err", err)
		duration = time.Since(start).Seconds()

		// 更新缓存
		updateProbeResultCache(target, moduleName, probeSuccess, messageSent, tlsVersion, isTLS, tlsCertExpire, tlsInfo, statusCode, statusCodeEnhanced, duration, dnsLookupTime, ipProtocol, ipAddrHash)

		return probeSuccess, messageSent, tlsVersion, isTLS, tlsCertExpire, tlsInfo, statusCode, statusCodeEnhanced, duration, dnsLookupTime, ipProtocol, ipAddrHash
	}

	messageSent = true
	probeSuccess = true

	// go-smtp库在成功发送时不提供状态码，设置默认值
	statusCode = 221
	statusCodeEnhanced = 200

	duration = time.Since(start).Seconds()

	// 更新缓存
	updateProbeResultCache(target, moduleName, probeSuccess, messageSent, tlsVersion, isTLS, tlsCertExpire, tlsInfo, statusCode, statusCodeEnhanced, duration, dnsLookupTime, ipProtocol, ipAddrHash)

	return probeSuccess, messageSent, tlsVersion, isTLS, tlsCertExpire, tlsInfo, statusCode, statusCodeEnhanced, duration, dnsLookupTime, ipProtocol, ipAddrHash
}

// 更新探测结果缓存
func updateProbeResultCache(target, moduleName string, probeSuccess, messageSent bool, tlsVersion string, isTLS bool, tlsCertExpire int64, tlsInfo string, statusCode, enhancedCode int, duration, dnsLookupTime, ipProtocol, ipAddrHash float64) {
	lastProbeResult.target = target
	lastProbeResult.moduleName = moduleName
	lastProbeResult.probeResult = probeSuccess
	lastProbeResult.messageSent = messageSent
	lastProbeResult.tlsVersion = tlsVersion
	lastProbeResult.isTLS = isTLS
	lastProbeResult.tlsCertExpire = tlsCertExpire
	lastProbeResult.tlsInfo = tlsInfo
	lastProbeResult.statusCode = statusCode
	lastProbeResult.enhancedCode = enhancedCode
	lastProbeResult.duration = duration
	lastProbeResult.dnsLookupTime = dnsLookupTime
	lastProbeResult.ipProtocol = ipProtocol
	lastProbeResult.ipAddrHash = ipAddrHash
	lastProbeResult.timestamp = time.Now()
}

var protocolToGauge = map[string]float64{
	"ip4": 4,
	"ip6": 6,
}

func chooseProtocol(ctx context.Context, IPProtocol string, fallbackIPProtocol bool, target string) (ip *net.IPAddr, lookupTime float64, returnerr error) {
	var fallbackProtocol string

	if IPProtocol == "ip6" {
		fallbackProtocol = "ip4"
	} else {
		fallbackProtocol = "ip6"
	}

	resolveStart := time.Now()

	defer func() {
		lookupTime = time.Since(resolveStart).Seconds()
	}()

	resolver := &net.Resolver{}

	if ips, err := resolver.LookupIP(ctx, IPProtocol, target); err == nil {
		ip = &net.IPAddr{IP: ips[0]}
		return
	} else if !fallbackIPProtocol {
		returnerr = fmt.Errorf("unable to find ip; no fallback: %s", err)
		return
	}

	ips, err := resolver.LookupIP(ctx, fallbackProtocol, target)
	if err != nil {
		returnerr = fmt.Errorf("unable to find ip; exhausted fallback: %s", err)
		return
	}
	ip = &net.IPAddr{IP: ips[0]}

	return
}

func ipHashFunc(ip net.IP) float64 {
	h := fnv.New32a()
	_, _ = h.Write(ip)
	return float64(h.Sum32())
}

func generateMessageID() string {
	t := time.Now().UnixNano()
	pid := os.Getpid()
	rint, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		panic(fmt.Sprintf("error generating MessageID: %s", err))
	}
	h, err := os.Hostname()
	// If we can't get the hostname, we'll use localhost
	if err != nil {
		h = "localhost.localdomain"
	}
	msgid := fmt.Sprintf("<%d.%d.%d@%s>", t, pid, rint, h)
	return msgid
}

func getTLSVersion(state *tls.ConnectionState) string {
	switch state.Version {
	case tls.VersionSSL30:
		return "SSL 3.0"
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return "unkown"
	}
}

func getCertExpiry(state *tls.ConnectionState) time.Time {
	return state.PeerCertificates[0].NotAfter
}

func getFingerprintSMTP(state *tls.ConnectionState) string {
	cert := state.PeerCertificates[0]
	fingerprint := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(fingerprint[:])
}

func newTLSConfig(promTLSConfig *pconfig.TLSConfig) (*tls.Config, error) {
	tlsConfig, err := pconfig.NewTLSConfig(promTLSConfig)
	if err != nil {
		return nil, err
	}
	return tlsConfig, nil
}

func ehlo(c *smtp.Client, ehloName string) error {
	err := c.Hello(ehloName)
	if err != nil {
		return err
	}
	return nil
}

func newSMTPClient(ctx context.Context, smtpProbe config.SMTPProbe, ip *net.IPAddr, port string) (c *smtp.Client, err error) {
	var dialProtocol, dialTarget string
	if ip.IP.To4() == nil {
		dialProtocol = "tcp6"
	} else {
		dialProtocol = "tcp4"
	}

	dialTarget = net.JoinHostPort(ip.String(), port)

	if strings.EqualFold(smtpProbe.TLS, "no") ||
		strings.EqualFold(smtpProbe.TLS, "starttls") {

		var d net.Dialer
		conn, err := d.DialContext(ctx, dialProtocol, dialTarget)
		if err != nil {
			return nil, err
		}

		deadline, _ := ctx.Deadline()
		if err = conn.SetDeadline(deadline); err != nil {
			return nil, err
		}

		if strings.EqualFold(smtpProbe.TLS, "starttls") {
			// 使用NewClientStartTLS创建带STARTTLS的客户端
			tlsConfig, err := newTLSConfig(&smtpProbe.TLSConfig)
			if err != nil {
				return nil, err
			}
			c, err = smtp.NewClientStartTLS(conn, tlsConfig)
			if err != nil {
				return nil, err
			}
		} else {
			// 普通客户端
			c = smtp.NewClient(conn)
		}

		if err = ehlo(c, smtpProbe.EHLO); err != nil {
			return nil, err
		}
	}

	if strings.EqualFold(smtpProbe.TLS, "tls") {
		// TLS配置需要从配置文件中读取
		tlsConfig, err := newTLSConfig(&smtpProbe.TLSConfig)
		if err != nil {
			return nil, err
		}

		var d tls.Dialer
		d.Config = tlsConfig
		conn, err := d.DialContext(ctx, dialProtocol, dialTarget)
		if err != nil {
			return nil, fmt.Errorf("could not connect to target: %s", err)
		}

		deadline, _ := ctx.Deadline()
		if err = conn.SetDeadline(deadline); err != nil {
			return nil, err
		}

		c = smtp.NewClient(conn)

		if err = ehlo(c, smtpProbe.EHLO); err != nil {
			return nil, err
		}
	}

	if c == nil {
		return nil, errors.New("could not create SMTP client")
	}
	return c, nil
}

// 导出SmtpProbeInternal函数，使其可以从server包中调用
// 实际实现在metrics.go文件中
// func SmtpProbeInternal(ctx context.Context, target string, module *config.SMTPModule) bool {
// 	// 实现在metrics.go中
// 	return false  // 这只是一个存根，实际实现在metrics.go文件中
// }
// Part 2 commit for smtp_exporter/internal/metrics/smtp.go

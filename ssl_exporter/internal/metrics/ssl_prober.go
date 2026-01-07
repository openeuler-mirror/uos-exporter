package metrics

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ocsp"
)

// TLSConfig 结构体，用于配置 TLS 连接
type TLSConfig struct {
	CAFile             string
	CertFile           string
	KeyFile            string
	ServerName         string
	InsecureSkipVerify bool
	Renegotiation      int
}

// Module 配置探针
type Module struct {
	Prober     string
	Target     string
	Timeout    time.Duration
	TLSConfig  TLSConfig
	TCP        TCPProbe
	HTTPS      HTTPSProbe
}

// TCPProbe 配置 TCP 探针
type TCPProbe struct {
	StartTLS string
}

// HTTPSProbe 配置 HTTPS 探针
type HTTPSProbe struct {
	ProxyURL *url.URL
}

// ProbeResult 存储探针结果
type ProbeResult struct {
	Success         bool
	Prober          string
	Certificates    []*x509.Certificate
	VerifiedChains  [][]*x509.Certificate
	TLSVersion      uint16
	OCSPResponse    []byte
	ConnectionState tls.ConnectionState
}

// newTLSConfig 创建新的 TLS 配置
func newTLSConfig(target string, cfg *TLSConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		// #nosec G402
		InsecureSkipVerify: cfg.InsecureSkipVerify,
		Renegotiation:      tls.RenegotiationSupport(cfg.Renegotiation),
	}

	if cfg.ServerName == "" && target != "" {
		targetAddress, _, err := net.SplitHostPort(target)
		if err != nil {
			return nil, err
		}
		tlsConfig.ServerName = targetAddress
	} else {
		tlsConfig.ServerName = cfg.ServerName
	}

	return tlsConfig, nil
}

// SSLProber 探针实例
var SSLProber *sslProber

// 初始化探针
func init() {
	SSLProber = NewSSLProber()
}

// sslProber 是 SSL 探针的实现
type sslProber struct{}

// NewSSLProber 创建新的 SSL 探针
func NewSSLProber() *sslProber {
	return &sslProber{}
}

// ProbeTCP 执行 TCP 探针
func (p *sslProber) ProbeTCP(ctx context.Context, target string, module Module) (*ProbeResult, error) {
	tlsConfig, err := newTLSConfig(target, &module.TLSConfig)
	if err != nil {
		return nil, err
	}

	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", target)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	deadline, _ := ctx.Deadline()
	if err := conn.SetDeadline(deadline); err != nil {
		return nil, fmt.Errorf("Error setting deadline")
	}

	if module.TCP.StartTLS != "" {
		err = startTLS(conn, module.TCP.StartTLS)
		if err != nil {
			return nil, err
		}
	}

	tlsConn := tls.Client(conn, tlsConfig)
	defer tlsConn.Close()

	if err := tlsConn.Handshake(); err != nil {
		return nil, err
	}

	state := tlsConn.ConnectionState()

	result := &ProbeResult{
		Success:         true,
		Prober:          "tcp",
		Certificates:    state.PeerCertificates,
		VerifiedChains:  state.VerifiedChains,
		TLSVersion:      state.Version,
		OCSPResponse:    state.OCSPResponse,
		ConnectionState: state,
	}

	return result, nil
}

// ProbeHTTPS 执行 HTTPS 探针
func (p *sslProber) ProbeHTTPS(ctx context.Context, target string, module Module) (*ProbeResult, error) {
	targetURL, err := url.Parse(fmt.Sprintf("https://%s", target))
	if err != nil {
		return nil, err
	}

	tlsConfig, err := newTLSConfig(target, &module.TLSConfig)
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	if module.HTTPS.ProxyURL != nil {
		transport.Proxy = http.ProxyURL(module.HTTPS.ProxyURL)
	}

	client := &http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: module.Timeout,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", targetURL.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 确保读取完整的响应体，但我们不需要内容
	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.TLS == nil {
		return nil, fmt.Errorf("No TLS connection state")
	}

	result := &ProbeResult{
		Success:         true,
		Prober:          "https",
		Certificates:    resp.TLS.PeerCertificates,
		VerifiedChains:  resp.TLS.VerifiedChains,
		TLSVersion:      resp.TLS.Version,
		OCSPResponse:    resp.TLS.OCSPResponse,
		ConnectionState: *resp.TLS,
	}

	return result, nil
}

// CollectMetrics 收集探针结果的指标
func CollectMetrics(result *ProbeResult) {
	// 这里实现指标收集逻辑
	if result == nil {
		return
	}

	// 设置 probe_success 指标
	metrics.probeSuccessValue = result.Success

	// 设置 prober 指标
	metrics.proberTypeValue = result.Prober

	// 设置 TLS 版本指标
	tlsVersionValue := ""
	switch result.TLSVersion {
	case tls.VersionTLS10:
		tlsVersionValue = "TLS 1.0"
	case tls.VersionTLS11:
		tlsVersionValue = "TLS 1.1"
	case tls.VersionTLS12:
		tlsVersionValue = "TLS 1.2"
	case tls.VersionTLS13:
		tlsVersionValue = "TLS 1.3"
	default:
		tlsVersionValue = "unknown"
	}
	metrics.tlsVersionInfo = map[string]float64{tlsVersionValue: 1}

	// 处理证书指标
	if len(result.Certificates) > 0 {
		certs := uniq(result.Certificates)
		metrics.certNotAfterValues = make(map[string]float64)
		metrics.certNotBeforeValues = make(map[string]float64)

		for _, cert := range certs {
			labels := labelValues(cert)
			labelKey := joinLabels(labels)

			if !cert.NotAfter.IsZero() {
				metrics.certNotAfterValues[labelKey] = float64(cert.NotAfter.Unix())
			}

			if !cert.NotBefore.IsZero() {
				metrics.certNotBeforeValues[labelKey] = float64(cert.NotBefore.Unix())
			}
		}
	}

	// 处理验证链指标
	if len(result.VerifiedChains) > 0 {
		metrics.verifiedNotAfterValues = make(map[string]float64)
		metrics.verifiedNotBeforeValues = make(map[string]float64)

		for i, chain := range result.VerifiedChains {
			chain = uniq(chain)
			for _, cert := range chain {
				chainNo := strconv.Itoa(i)
				labels := append([]string{chainNo}, labelValues(cert)...)
				labelKey := joinLabels(labels)

				if !cert.NotAfter.IsZero() {
					metrics.verifiedNotAfterValues[labelKey] = float64(cert.NotAfter.Unix())
				}

				if !cert.NotBefore.IsZero() {
					metrics.verifiedNotBeforeValues[labelKey] = float64(cert.NotBefore.Unix())
				}
			}
		}
	}

	// 处理 OCSP 响应指标
	if len(result.OCSPResponse) > 0 {
		metrics.ocspResponseStapledValue = 1

		resp, err := ocsp.ParseResponse(result.OCSPResponse, nil)
		if err == nil {
			metrics.ocspResponseStatusValue = float64(resp.Status)
			metrics.ocspResponseProducedAtValue = float64(resp.ProducedAt.Unix())
			metrics.ocspResponseThisUpdateValue = float64(resp.ThisUpdate.Unix())
			metrics.ocspResponseNextUpdateValue = float64(resp.NextUpdate.Unix())
			metrics.ocspResponseRevokedAtValue = float64(resp.RevokedAt.Unix())
		}
	}
}

// 全局指标数据存储
var metrics = struct {
	probeSuccessValue             bool
	proberTypeValue               string
	tlsVersionInfo                map[string]float64
	certNotAfterValues            map[string]float64
	certNotBeforeValues           map[string]float64
	verifiedNotAfterValues        map[string]float64
	verifiedNotBeforeValues       map[string]float64
	ocspResponseStapledValue      float64
	ocspResponseStatusValue       float64
	ocspResponseProducedAtValue   float64
	ocspResponseThisUpdateValue   float64
	ocspResponseNextUpdateValue   float64
	ocspResponseRevokedAtValue    float64
}{
	probeSuccessValue:             false,
	proberTypeValue:               "",
	tlsVersionInfo:                make(map[string]float64),
	certNotAfterValues:            make(map[string]float64),
	certNotBeforeValues:           make(map[string]float64),
	verifiedNotAfterValues:        make(map[string]float64),
	verifiedNotBeforeValues:       make(map[string]float64),
	ocspResponseStapledValue:      0,
	ocspResponseStatusValue:       0,
	ocspResponseProducedAtValue:   0,
	ocspResponseThisUpdateValue:   0,
	ocspResponseNextUpdateValue:   0,
	ocspResponseRevokedAtValue:    0,
}

// 重写 Collect 方法，让其能够获取全局存储的指标数据
func (s *SSLProbeSuccess) Collect(ch chan<- prometheus.Metric) {
	value := 0.0
	if metrics.probeSuccessValue {
		value = 1.0
	}
	s.baseMetrics.collect(ch, value, []string{})
}

func (s *SSLProberType) Collect(ch chan<- prometheus.Metric) {
	if metrics.proberTypeValue != "" {
		s.baseMetrics.collect(ch, 1.0, []string{metrics.proberTypeValue})
	}
}

func (s *SSLTLSVersion) Collect(ch chan<- prometheus.Metric) {
	for version, value := range metrics.tlsVersionInfo {
		s.baseMetrics.collect(ch, value, []string{version})
	}
}

func (s *SSLCertNotAfter) Collect(ch chan<- prometheus.Metric) {
	for labelKey, value := range metrics.certNotAfterValues {
		s.baseMetrics.collect(ch, value, splitLabels(labelKey))
	}
}

func (s *SSLCertNotBefore) Collect(ch chan<- prometheus.Metric) {
	for labelKey, value := range metrics.certNotBeforeValues {
		s.baseMetrics.collect(ch, value, splitLabels(labelKey))
	}
}

func (s *SSLVerifiedCertNotAfter) Collect(ch chan<- prometheus.Metric) {
	for labelKey, value := range metrics.verifiedNotAfterValues {
		s.baseMetrics.collect(ch, value, splitLabels(labelKey))
	}
}

func (s *SSLVerifiedCertNotBefore) Collect(ch chan<- prometheus.Metric) {
	for labelKey, value := range metrics.verifiedNotBeforeValues {
		s.baseMetrics.collect(ch, value, splitLabels(labelKey))
	}
}

func (s *SSLOCSPResponseStapled) Collect(ch chan<- prometheus.Metric) {
	s.baseMetrics.collect(ch, metrics.ocspResponseStapledValue, []string{})
}

func (s *SSLOCSPResponseStatus) Collect(ch chan<- prometheus.Metric) {
	s.baseMetrics.collect(ch, metrics.ocspResponseStatusValue, []string{})
}

func (s *SSLOCSPResponseProducedAt) Collect(ch chan<- prometheus.Metric) {
	s.baseMetrics.collect(ch, metrics.ocspResponseProducedAtValue, []string{})
}

func (s *SSLOCSPResponseThisUpdate) Collect(ch chan<- prometheus.Metric) {
	s.baseMetrics.collect(ch, metrics.ocspResponseThisUpdateValue, []string{})
}

func (s *SSLOCSPResponseNextUpdate) Collect(ch chan<- prometheus.Metric) {
	s.baseMetrics.collect(ch, metrics.ocspResponseNextUpdateValue, []string{})
}

func (s *SSLOCSPResponseRevokedAt) Collect(ch chan<- prometheus.Metric) {
	s.baseMetrics.collect(ch, metrics.ocspResponseRevokedAtValue, []string{})
}

// 辅助函数，用于处理标签
func joinLabels(labels []string) string {
	var result bytes.Buffer
	for i, label := range labels {
		if i > 0 {
			result.WriteString("|")
		}
		result.WriteString(label)
	}
	return result.String()
}

func splitLabels(labelKey string) []string {
	return regexp.MustCompile(`\|`).Split(labelKey, -1)
}

// startTLS 实现 STARTTLS 命令
func startTLS(conn net.Conn, proto string) error {
	var err error

	qr, ok := startTLSqueryResponses[proto]
	if !ok {
		return fmt.Errorf("STARTTLS is not supported for %s", proto)
	}

	scanner := bufio.NewScanner(conn)
	for _, qr := range qr {
		if qr.expect != "" {
			var match bool
			for scanner.Scan() {
				logrus.Debugf("read line: %s", scanner.Text())
				match, err = regexp.Match(qr.expect, scanner.Bytes())
				if err != nil {
					return err
				}
				if match {
					logrus.Debugf("regex: %s matched: %s", qr.expect, scanner.Text())
					break
				}
			}
			if scanner.Err() != nil {
				return scanner.Err()
			}
			if !match {
				return fmt.Errorf("regex: %s didn't match: %s", qr.expect, scanner.Text())
			}
		}
		if len(qr.expectBytes) > 0 {
			buffer := make([]byte, len(qr.expectBytes))
			_, err = io.ReadFull(conn, buffer)
			if err != nil {
				return nil
			}
			logrus.Debugf("read bytes: %x", buffer)
			if bytes.Compare(buffer, qr.expectBytes) != 0 {
				return fmt.Errorf("read bytes %x didn't match with expected bytes %x", buffer, qr.expectBytes)
			} else {
				logrus.Debugf("expected bytes %x matched with read bytes %x", qr.expectBytes, buffer)
			}
		}
		if qr.send != "" {
			logrus.Debugf("sending line: %s", qr.send)
			if _, err := fmt.Fprintf(conn, "%s\r\n", qr.send); err != nil {
				return err
			}
		}
		if len(qr.sendBytes) > 0 {
			logrus.Debugf("sending bytes: %x", qr.sendBytes)
			if _, err = conn.Write(qr.sendBytes); err != nil {
				return err
			}
		}
	}
	return nil
}

// startTLSqueryResponses 定义了各种协议的 STARTTLS 交互
var startTLSqueryResponses = map[string][]queryResponse{
	"smtp": {
		{
			expect: "^220",
		},
		{
			send: "EHLO prober",
		},
		{
			expect: "^250(-| )STARTTLS",
		},
		{
			send: "STARTTLS",
		},
		{
			expect: "^220",
		},
	},
	"ftp": {
		{
			expect: "^220",
		},
		{
			send: "AUTH TLS",
		},
		{
			expect: "^234",
		},
	},
	"imap": {
		{
			expect: "OK",
		},
		{
			send: ". CAPABILITY",
		},
		{
			expect: "STARTTLS",
		},
		{
			expect: "OK",
		},
		{
			send: ". STARTTLS",
		},
		{
			expect: "OK",
		},
	},
	"postgres": {
		{
			sendBytes: []byte{0x00, 0x00, 0x00, 0x08, 0x04, 0xd2, 0x16, 0x2f},
		},
		{
			expectBytes: []byte{0x53},
		},
	},
	"pop3": {
		{
			expect: "OK",
		},
		{
			send: "STLS",
		},
		{
			expect: "OK",
		},
	},
}

// queryResponse 定义了请求响应交互
type queryResponse struct {
	expect      string
	send        string
	sendBytes   []byte
	expectBytes []byte
}

// 辅助函数，用于生成证书标签值
func labelValues(cert *x509.Certificate) []string {
	return []string{
		cert.SerialNumber.String(),
		cert.Issuer.CommonName,
		cert.Subject.CommonName,
		dnsNames(cert),
		ipAddresses(cert),
		emailAddresses(cert),
		organizationalUnits(cert),
	}
}

func dnsNames(cert *x509.Certificate) string {
	if len(cert.DNSNames) == 0 {
		return ""
	}
	return strings.Join(cert.DNSNames, ",")
}

func emailAddresses(cert *x509.Certificate) string {
	if len(cert.EmailAddresses) == 0 {
		return ""
	}
	return strings.Join(cert.EmailAddresses, ",")
}

func ipAddresses(cert *x509.Certificate) string {
	if len(cert.IPAddresses) == 0 {
		return ""
	}
	var ips []string
	for _, ip := range cert.IPAddresses {
		ips = append(ips, ip.String())
	}
	return strings.Join(ips, ",")
}

func organizationalUnits(cert *x509.Certificate) string {
	if len(cert.Subject.OrganizationalUnit) == 0 {
		return ""
	}
	return strings.Join(cert.Subject.OrganizationalUnit, ",")
}

// 辅助函数，用于处理证书唯一性
func uniq(certs []*x509.Certificate) []*x509.Certificate {
	r := []*x509.Certificate{}

	for _, c := range certs {
		if !contains(r, c) {
			r = append(r, c)
		}
	}

	return r
}

func contains(certs []*x509.Certificate, cert *x509.Certificate) bool {
	for _, c := range certs {
		if (c.SerialNumber.String() == cert.SerialNumber.String()) && (c.Issuer.CommonName == cert.Issuer.CommonName) {
			return true
		}
	}
	return false
}

func decodeCertificates(data []byte) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate
	for block, rest := pem.Decode(data); block != nil; block, rest = pem.Decode(rest) {
		if block.Type == "CERTIFICATE" || block.Type == "TRUSTED CERTIFICATE" {
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return certs, err
			}
			if !contains(certs, cert) {
				certs = append(certs, cert)
			}
		}
	}

	return certs, nil
} 

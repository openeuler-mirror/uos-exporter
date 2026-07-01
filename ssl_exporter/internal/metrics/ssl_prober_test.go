package metrics

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"net/url"
	"testing"
	"time"
)

// 创建模拟证书用于测试
func createMockCert() *x509.Certificate {
	return &x509.Certificate{
		SerialNumber: big.NewInt(123456),
		Subject: pkix.Name{
			CommonName:         "test.example.com",
			OrganizationalUnit: []string{"Test Unit"},
		},
		Issuer: pkix.Name{
			CommonName: "Test CA",
		},
		NotBefore:      time.Now().Add(-24 * time.Hour),
		NotAfter:       time.Now().Add(365 * 24 * time.Hour),
		DNSNames:       []string{"test.example.com", "www.test.example.com"},
		IPAddresses:    []net.IP{net.ParseIP("192.168.1.1")},
		EmailAddresses: []string{"test@example.com"},
	}
}

// 测试TLS配置生成函数
func TestNewTLSConfig(t *testing.T) {
	tests := []struct {
		name      string
		target    string
		tlsConfig *TLSConfig
		wantErr   bool
	}{
		{
			name:      "基本配置测试",
			target:    "example.com:443",
			tlsConfig: &TLSConfig{},
			wantErr:   false,
		},
		{
			name:      "带ServerName的配置测试",
			target:    "example.com:443",
			tlsConfig: &TLSConfig{ServerName: "custom.example.com"},
			wantErr:   false,
		},
		{
			name:      "InsecureSkipVerify配置测试",
			target:    "example.com:443",
			tlsConfig: &TLSConfig{InsecureSkipVerify: true},
			wantErr:   false,
		},
		{
			name:      "不合法目标格式测试",
			target:    "example.com", // 缺少端口
			tlsConfig: &TLSConfig{},
			wantErr:   true,
		},
		{
			name:      "Renegotiation设置测试",
			target:    "example.com:443",
			tlsConfig: &TLSConfig{Renegotiation: 1}, // RenegotiateOnceAsClient
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newTLSConfig(tt.target, tt.tlsConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("newTLSConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("newTLSConfig() returned nil but wanted a config")
			}
			if !tt.wantErr {
				// 检查ServerName设置
				if tt.tlsConfig.ServerName != "" {
					if got.ServerName != tt.tlsConfig.ServerName {
						t.Errorf("newTLSConfig() ServerName = %v, want %v", got.ServerName, tt.tlsConfig.ServerName)
					}
				} else if got.ServerName == "" {
					t.Error("newTLSConfig() ServerName is empty")
				}
				// 检查InsecureSkipVerify设置
				if got.InsecureSkipVerify != tt.tlsConfig.InsecureSkipVerify {
					t.Errorf("newTLSConfig() InsecureSkipVerify = %v, want %v", got.InsecureSkipVerify, tt.tlsConfig.InsecureSkipVerify)
				}
				// 检查Renegotiation设置
				if got.Renegotiation != tls.RenegotiationSupport(tt.tlsConfig.Renegotiation) {
					t.Errorf("newTLSConfig() Renegotiation = %v, want %v", got.Renegotiation, tls.RenegotiationSupport(tt.tlsConfig.Renegotiation))
				}
			}
		})
	}
}

// 测试SSL探针实例化
func TestNewSSLProber(t *testing.T) {
	prober := NewSSLProber()
	if prober == nil {
		t.Error("NewSSLProber() returned nil")
	}
}

// 测试标签辅助函数
func TestLabelFunctions(t *testing.T) {
	// 测试生成标签值
	cert := createMockCert()
	labels := labelValues(cert)
	if len(labels) != 7 {
		t.Errorf("labelValues() returned %d labels, want 7", len(labels))
	}

	// 测试DNS名称拼接
	dnsNamesStr := dnsNames(cert)
	if dnsNamesStr != "test.example.com,www.test.example.com" {
		t.Errorf("dnsNames() = %v, want %v", dnsNamesStr, "test.example.com,www.test.example.com")
	}

	// 测试IP地址拼接
	ipAddressesStr := ipAddresses(cert)
	if ipAddressesStr != "192.168.1.1" {
		t.Errorf("ipAddresses() = %v, want %v", ipAddressesStr, "192.168.1.1")
	}

	// 测试Email地址拼接
	emailAddressesStr := emailAddresses(cert)
	if emailAddressesStr != "test@example.com" {
		t.Errorf("emailAddresses() = %v, want %v", emailAddressesStr, "test@example.com")
	}

	// 测试组织单位拼接
	ouStr := organizationalUnits(cert)
	if ouStr != "Test Unit" {
		t.Errorf("organizationalUnits() = %v, want %v", ouStr, "Test Unit")
	}

	// 测试空值情况
	emptyCert := &x509.Certificate{
		SerialNumber: big.NewInt(123456),
		Subject: pkix.Name{
			CommonName: "test.example.com",
		},
		Issuer: pkix.Name{
			CommonName: "Test CA",
		},
	}

	if dnsNames(emptyCert) != "" {
		t.Errorf("dnsNames() for empty cert = %v, want %v", dnsNames(emptyCert), "")
	}
	if ipAddresses(emptyCert) != "" {
		t.Errorf("ipAddresses() for empty cert = %v, want %v", ipAddresses(emptyCert), "")
	}
	if emailAddresses(emptyCert) != "" {
		t.Errorf("emailAddresses() for empty cert = %v, want %v", emailAddresses(emptyCert), "")
	}
	if organizationalUnits(emptyCert) != "" {
		t.Errorf("organizationalUnits() for empty cert = %v, want %v", organizationalUnits(emptyCert), "")
	}
}

// 测试标签连接和分割函数
func TestJoinSplitLabels(t *testing.T) {
	labels := []string{"label1", "label2", "label3"}
	joined := joinLabels(labels)
	if joined != "label1|label2|label3" {
		t.Errorf("joinLabels() = %v, want %v", joined, "label1|label2|label3")
	}

	split := splitLabels(joined)
	if len(split) != 3 {
		t.Errorf("splitLabels() returned %d labels, want 3", len(split))
	}
	for i, label := range labels {
		if split[i] != label {
			t.Errorf("splitLabels()[%d] = %v, want %v", i, split[i], label)
		}
	}

	// 测试边缘情况
	emptyLabels := []string{}
	emptyJoined := joinLabels(emptyLabels)
	if emptyJoined != "" {
		t.Errorf("joinLabels() for empty labels = %v, want %v", emptyJoined, "")
	}

	emptySplit := splitLabels("")
	if len(emptySplit) != 1 || emptySplit[0] != "" {
		t.Errorf("splitLabels() for empty string = %v, want [\"\"]", emptySplit)
	}
}

// 测试证书唯一性函数
func TestCertificateUniqueness(t *testing.T) {
	cert1 := createMockCert()
	cert2 := createMockCert()
	cert2.SerialNumber = big.NewInt(789012) // 不同的序列号

	certs := []*x509.Certificate{cert1, cert2, cert1} // 重复的cert1
	uniqueCerts := uniq(certs)

	if len(uniqueCerts) != 2 {
		t.Errorf("uniq() returned %d certs, want 2", len(uniqueCerts))
	}

	// 检查contains函数
	if !contains(uniqueCerts, cert1) {
		t.Error("contains() returned false for cert1, want true")
	}
	if !contains(uniqueCerts, cert2) {
		t.Error("contains() returned false for cert2, want true")
	}

	// 创建新证书但序列号和颁发者相同
	cert3 := createMockCert()
	cert3.Subject.CommonName = "different.example.com"
	if !contains(uniqueCerts, cert3) {
		t.Error("contains() returned false for cert with same serial and issuer, want true")
	}

	// 完全不同的证书
	cert4 := createMockCert()
	cert4.SerialNumber = big.NewInt(111222)
	cert4.Issuer.CommonName = "Different CA"
	if contains(uniqueCerts, cert4) {
		t.Error("contains() returned true for completely different cert, want false")
	}
}

// 测试证书解码函数
func TestDecodeCertificates(t *testing.T) {
	// 此测试需要实际的PEM编码证书数据，我们只测试基本错误情况
	invalidData := []byte("not a valid PEM certificate")
	certs, err := decodeCertificates(invalidData)
	if err != nil {
		t.Errorf("decodeCertificates() returned error for invalid data: %v", err)
	}
	if len(certs) != 0 {
		t.Errorf("decodeCertificates() returned %d certs for invalid data, want 0", len(certs))
	}
}

// 测试CollectMetrics函数基本功能
func TestCollectMetrics(t *testing.T) {
	// 空结果测试
	CollectMetrics(nil)

	// 成功结果测试
	result := &ProbeResult{
		Success: true,
		Prober:  "tcp",
		TLSVersion: tls.VersionTLS12,
	}
	CollectMetrics(result)
	if !metrics.probeSuccessValue {
		t.Error("CollectMetrics() did not set probeSuccessValue")
	}
	if metrics.proberTypeValue != "tcp" {
		t.Errorf("CollectMetrics() set proberTypeValue = %v, want %v", metrics.proberTypeValue, "tcp")
	}

	// 失败结果测试
	failResult := &ProbeResult{
		Success: false,
		Prober:  "https",
	}
	CollectMetrics(failResult)
	if metrics.probeSuccessValue {
		t.Error("CollectMetrics() did not set probeSuccessValue to false")
	}
	if metrics.proberTypeValue != "https" {
		t.Errorf("CollectMetrics() set proberTypeValue = %v, want %v", metrics.proberTypeValue, "https")
	}
}

// 模拟TCP连接
type mockConn struct {
	net.Conn
	readData []byte
	writeData bytes.Buffer
}

func (m *mockConn) Read(b []byte) (n int, err error) {
	if len(m.readData) == 0 {
		return 0, nil
	}
	n = copy(b, m.readData)
	m.readData = m.readData[n:]
	return n, nil
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	return m.writeData.Write(b)
}

func (m *mockConn) Close() error {
	return nil
}

func (m *mockConn) SetDeadline(t time.Time) error {
	return nil
}

// 具体的TCP和HTTPS探针测试可能需要更复杂的模拟或依赖注入
// 这里我们只测试一些简单的直接调用情况

// 测试探测模块结构
func TestModuleStructs(t *testing.T) {
	module := Module{
		Prober:  "tcp",
		Target:  "example.com:443",
		Timeout: 10 * time.Second,
		TLSConfig: TLSConfig{
			InsecureSkipVerify: true,
		},
		TCP: TCPProbe{
			StartTLS: "smtp",
		},
		HTTPS: HTTPSProbe{
			ProxyURL: &url.URL{
				Scheme: "http",
				Host:   "proxy.example.com:8080",
			},
		},
	}

	// 验证结构字段
	if module.Prober != "tcp" {
		t.Errorf("Module.Prober = %v, want %v", module.Prober, "tcp")
	}
	if module.Target != "example.com:443" {
		t.Errorf("Module.Target = %v, want %v", module.Target, "example.com:443")
	}
	if module.Timeout != 10*time.Second {
		t.Errorf("Module.Timeout = %v, want %v", module.Timeout, 10*time.Second)
	}
	if !module.TLSConfig.InsecureSkipVerify {
		t.Error("Module.TLSConfig.InsecureSkipVerify = false, want true")
	}
	if module.TCP.StartTLS != "smtp" {
		t.Errorf("Module.TCP.StartTLS = %v, want %v", module.TCP.StartTLS, "smtp")
	}
	if module.HTTPS.ProxyURL.Host != "proxy.example.com:8080" {
		t.Errorf("Module.HTTPS.ProxyURL.Host = %v, want %v", module.HTTPS.ProxyURL.Host, "proxy.example.com:8080")
	}
}

// 测试模拟的探针调用 - 更复杂的验证通常需要完整的集成测试
func TestSSLProberMockCall(t *testing.T) {
	prober := NewSSLProber()
	if prober == nil {
		t.Fatal("NewSSLProber() returned nil")
	}

	// 基本结构验证
	ctx := context.Background()
	module := Module{
		Prober: "tcp",
		Target: "example.com:443",
		TLSConfig: TLSConfig{},
	}

	// 这些会失败，因为它们需要实际的网络连接
	// 我们只验证函数调用不会导致panic
	_, err := prober.ProbeTCP(ctx, module.Target, module)
	if err == nil {
		// 在实际环境中这几乎肯定会失败，除非正好能连接到本地测试服务器
		t.Log("ProbeTCP() unexpectedly succeeded - might be running with actual network")
	}

	_, err = prober.ProbeHTTPS(ctx, module.Target, module)
	if err == nil {
		// 在实际环境中这几乎肯定会失败，除非正好能连接到本地测试服务器
		t.Log("ProbeHTTPS() unexpectedly succeeded - might be running with actual network")
	}
} 
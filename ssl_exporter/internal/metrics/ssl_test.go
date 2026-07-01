package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"testing"
)

func TestNewSSLCertNotAfter(t *testing.T) {
	tests := []struct {
		name   string
		fqname string
		help   string
		labels []string
	}{
		{
			name:   "基本创建测试",
			fqname: "ssl_cert_not_after",
			help:   "NotAfter expressed as a Unix Epoch Time",
			labels: []string{"serial_no", "issuer_cn", "cn", "dnsnames", "ips", "emails", "ou"},
		},
		{
			name:   "空标签测试",
			fqname: "ssl_cert_not_after",
			help:   "NotAfter expressed as a Unix Epoch Time",
			labels: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSSLCertNotAfter(tt.fqname, tt.help, tt.labels)
			if got == nil {
				t.Error("NewSSLCertNotAfter() returned nil")
			}
			if got.baseMetrics == nil {
				t.Error("NewSSLCertNotAfter() returned metrics with nil baseMetrics")
			}
		})
	}
}

func TestNewSSLCertNotBefore(t *testing.T) {
	tests := []struct {
		name   string
		fqname string
		help   string
		labels []string
	}{
		{
			name:   "基本创建测试",
			fqname: "ssl_cert_not_before",
			help:   "NotBefore expressed as a Unix Epoch Time",
			labels: []string{"serial_no", "issuer_cn", "cn", "dnsnames", "ips", "emails", "ou"},
		},
		{
			name:   "空标签测试",
			fqname: "ssl_cert_not_before",
			help:   "NotBefore expressed as a Unix Epoch Time",
			labels: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSSLCertNotBefore(tt.fqname, tt.help, tt.labels)
			if got == nil {
				t.Error("NewSSLCertNotBefore() returned nil")
			}
			if got.baseMetrics == nil {
				t.Error("NewSSLCertNotBefore() returned metrics with nil baseMetrics")
			}
		})
	}
}

func TestNewSSLVerifiedCertNotAfter(t *testing.T) {
	tests := []struct {
		name   string
		fqname string
		help   string
		labels []string
	}{
		{
			name:   "基本创建测试",
			fqname: "ssl_verified_cert_not_after",
			help:   "NotAfter expressed as a Unix Epoch Time",
			labels: []string{"chain_no", "serial_no", "issuer_cn", "cn", "dnsnames", "ips", "emails", "ou"},
		},
		{
			name:   "空标签测试",
			fqname: "ssl_verified_cert_not_after",
			help:   "NotAfter expressed as a Unix Epoch Time",
			labels: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSSLVerifiedCertNotAfter(tt.fqname, tt.help, tt.labels)
			if got == nil {
				t.Error("NewSSLVerifiedCertNotAfter() returned nil")
			}
			if got.baseMetrics == nil {
				t.Error("NewSSLVerifiedCertNotAfter() returned metrics with nil baseMetrics")
			}
		})
	}
}

func TestNewSSLVerifiedCertNotBefore(t *testing.T) {
	tests := []struct {
		name   string
		fqname string
		help   string
		labels []string
	}{
		{
			name:   "基本创建测试",
			fqname: "ssl_verified_cert_not_before",
			help:   "NotBefore expressed as a Unix Epoch Time",
			labels: []string{"chain_no", "serial_no", "issuer_cn", "cn", "dnsnames", "ips", "emails", "ou"},
		},
		{
			name:   "空标签测试",
			fqname: "ssl_verified_cert_not_before",
			help:   "NotBefore expressed as a Unix Epoch Time",
			labels: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSSLVerifiedCertNotBefore(tt.fqname, tt.help, tt.labels)
			if got == nil {
				t.Error("NewSSLVerifiedCertNotBefore() returned nil")
			}
			if got.baseMetrics == nil {
				t.Error("NewSSLVerifiedCertNotBefore() returned metrics with nil baseMetrics")
			}
		})
	}
}

func TestNewSSLTLSVersion(t *testing.T) {
	tests := []struct {
		name   string
		fqname string
		help   string
		labels []string
	}{
		{
			name:   "基本创建测试",
			fqname: "ssl_tls_version_info",
			help:   "The TLS version used",
			labels: []string{"version"},
		},
		{
			name:   "空标签测试",
			fqname: "ssl_tls_version_info",
			help:   "The TLS version used",
			labels: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSSLTLSVersion(tt.fqname, tt.help, tt.labels)
			if got == nil {
				t.Error("NewSSLTLSVersion() returned nil")
			}
			if got.baseMetrics == nil {
				t.Error("NewSSLTLSVersion() returned metrics with nil baseMetrics")
			}
		})
	}
}

func TestNewSSLOCSPResponseStapled(t *testing.T) {
	tests := []struct {
		name   string
		fqname string
		help   string
		labels []string
	}{
		{
			name:   "基本创建测试",
			fqname: "ssl_ocsp_response_stapled",
			help:   "If the connection state contains a stapled OCSP response",
			labels: []string{},
		},
		{
			name:   "额外标签测试",
			fqname: "ssl_ocsp_response_stapled",
			help:   "If the connection state contains a stapled OCSP response",
			labels: []string{"extra"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSSLOCSPResponseStapled(tt.fqname, tt.help, tt.labels)
			if got == nil {
				t.Error("NewSSLOCSPResponseStapled() returned nil")
			}
			if got.baseMetrics == nil {
				t.Error("NewSSLOCSPResponseStapled() returned metrics with nil baseMetrics")
			}
		})
	}
}

func TestNewSSLOCSPResponseStatus(t *testing.T) {
	tests := []struct {
		name   string
		fqname string
		help   string
		labels []string
	}{
		{
			name:   "基本创建测试",
			fqname: "ssl_ocsp_response_status",
			help:   "The status in the OCSP response 0=Good 1=Revoked 2=Unknown",
			labels: []string{},
		},
		{
			name:   "额外标签测试",
			fqname: "ssl_ocsp_response_status",
			help:   "The status in the OCSP response 0=Good 1=Revoked 2=Unknown",
			labels: []string{"extra"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSSLOCSPResponseStatus(tt.fqname, tt.help, tt.labels)
			if got == nil {
				t.Error("NewSSLOCSPResponseStatus() returned nil")
			}
			if got.baseMetrics == nil {
				t.Error("NewSSLOCSPResponseStatus() returned metrics with nil baseMetrics")
			}
		})
	}
}

func TestNewSSLOCSPResponseProducedAt(t *testing.T) {
	tests := []struct {
		name   string
		fqname string
		help   string
		labels []string
	}{
		{
			name:   "基本创建测试",
			fqname: "ssl_ocsp_response_produced_at",
			help:   "The producedAt value in the OCSP response, expressed as a Unix Epoch Time",
			labels: []string{},
		},
		{
			name:   "额外标签测试",
			fqname: "ssl_ocsp_response_produced_at",
			help:   "The producedAt value in the OCSP response, expressed as a Unix Epoch Time",
			labels: []string{"extra"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSSLOCSPResponseProducedAt(tt.fqname, tt.help, tt.labels)
			if got == nil {
				t.Error("NewSSLOCSPResponseProducedAt() returned nil")
			}
			if got.baseMetrics == nil {
				t.Error("NewSSLOCSPResponseProducedAt() returned metrics with nil baseMetrics")
			}
		})
	}
}

func TestNewSSLOCSPResponseThisUpdate(t *testing.T) {
	tests := []struct {
		name   string
		fqname string
		help   string
		labels []string
	}{
		{
			name:   "基本创建测试",
			fqname: "ssl_ocsp_response_this_update",
			help:   "The thisUpdate value in the OCSP response, expressed as a Unix Epoch Time",
			labels: []string{},
		},
		{
			name:   "额外标签测试",
			fqname: "ssl_ocsp_response_this_update",
			help:   "The thisUpdate value in the OCSP response, expressed as a Unix Epoch Time",
			labels: []string{"extra"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSSLOCSPResponseThisUpdate(tt.fqname, tt.help, tt.labels)
			if got == nil {
				t.Error("NewSSLOCSPResponseThisUpdate() returned nil")
			}
			if got.baseMetrics == nil {
				t.Error("NewSSLOCSPResponseThisUpdate() returned metrics with nil baseMetrics")
			}
		})
	}
}

func TestNewSSLOCSPResponseNextUpdate(t *testing.T) {
	tests := []struct {
		name   string
		fqname string
		help   string
		labels []string
	}{
		{
			name:   "基本创建测试",
			fqname: "ssl_ocsp_response_next_update",
			help:   "The nextUpdate value in the OCSP response, expressed as a Unix Epoch Time",
			labels: []string{},
		},
		{
			name:   "额外标签测试",
			fqname: "ssl_ocsp_response_next_update",
			help:   "The nextUpdate value in the OCSP response, expressed as a Unix Epoch Time",
			labels: []string{"extra"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSSLOCSPResponseNextUpdate(tt.fqname, tt.help, tt.labels)
			if got == nil {
				t.Error("NewSSLOCSPResponseNextUpdate() returned nil")
			}
			if got.baseMetrics == nil {
				t.Error("NewSSLOCSPResponseNextUpdate() returned metrics with nil baseMetrics")
			}
		})
	}
}

func TestNewSSLOCSPResponseRevokedAt(t *testing.T) {
	tests := []struct {
		name   string
		fqname string
		help   string
		labels []string
	}{
		{
			name:   "基本创建测试",
			fqname: "ssl_ocsp_response_revoked_at",
			help:   "The revocationTime value in the OCSP response, expressed as a Unix Epoch Time",
			labels: []string{},
		},
		{
			name:   "额外标签测试",
			fqname: "ssl_ocsp_response_revoked_at",
			help:   "The revocationTime value in the OCSP response, expressed as a Unix Epoch Time",
			labels: []string{"extra"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSSLOCSPResponseRevokedAt(tt.fqname, tt.help, tt.labels)
			if got == nil {
				t.Error("NewSSLOCSPResponseRevokedAt() returned nil")
			}
			if got.baseMetrics == nil {
				t.Error("NewSSLOCSPResponseRevokedAt() returned metrics with nil baseMetrics")
			}
		})
	}
}

func TestNewSSLProbeSuccess(t *testing.T) {
	tests := []struct {
		name   string
		fqname string
		help   string
		labels []string
	}{
		{
			name:   "基本创建测试",
			fqname: "ssl_probe_success",
			help:   "If the probe was a success",
			labels: []string{},
		},
		{
			name:   "额外标签测试",
			fqname: "ssl_probe_success",
			help:   "If the probe was a success",
			labels: []string{"extra"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSSLProbeSuccess(tt.fqname, tt.help, tt.labels)
			if got == nil {
				t.Error("NewSSLProbeSuccess() returned nil")
			}
			if got.baseMetrics == nil {
				t.Error("NewSSLProbeSuccess() returned metrics with nil baseMetrics")
			}
		})
	}
}

func TestNewSSLProberType(t *testing.T) {
	tests := []struct {
		name   string
		fqname string
		help   string
		labels []string
	}{
		{
			name:   "基本创建测试",
			fqname: "ssl_prober",
			help:   "The prober used by the exporter to connect to the target",
			labels: []string{"prober"},
		},
		{
			name:   "额外标签测试",
			fqname: "ssl_prober",
			help:   "The prober used by the exporter to connect to the target",
			labels: []string{"prober", "extra"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSSLProberType(tt.fqname, tt.help, tt.labels)
			if got == nil {
				t.Error("NewSSLProberType() returned nil")
			}
			if got.baseMetrics == nil {
				t.Error("NewSSLProberType() returned metrics with nil baseMetrics")
			}
		})
	}
}

// 测试指标收集器的集合方法
func TestSSLMetricsCollectors(t *testing.T) {
	// 为所有指标创建实例进行收集测试
	metricsList := []struct {
		name string
		baseMetric *baseMetrics
	}{
		{
			name: "SSLCertNotAfter",
			baseMetric: NewSSLCertNotAfter("ssl_cert_not_after", "test", []string{"serial_no", "issuer_cn", "cn", "dnsnames", "ips", "emails", "ou"}).baseMetrics,
		},
		{
			name: "SSLCertNotBefore",
			baseMetric: NewSSLCertNotBefore("ssl_cert_not_before", "test", []string{"serial_no", "issuer_cn", "cn", "dnsnames", "ips", "emails", "ou"}).baseMetrics,
		},
		{
			name: "SSLVerifiedCertNotAfter",
			baseMetric: NewSSLVerifiedCertNotAfter("ssl_verified_cert_not_after", "test", []string{"chain_no", "serial_no", "issuer_cn", "cn", "dnsnames", "ips", "emails", "ou"}).baseMetrics,
		},
		{
			name: "SSLVerifiedCertNotBefore",
			baseMetric: NewSSLVerifiedCertNotBefore("ssl_verified_cert_not_before", "test", []string{"chain_no", "serial_no", "issuer_cn", "cn", "dnsnames", "ips", "emails", "ou"}).baseMetrics,
		},
		{
			name: "SSLTLSVersion",
			baseMetric: NewSSLTLSVersion("ssl_tls_version_info", "test", []string{"version"}).baseMetrics,
		},
		{
			name: "SSLOCSPResponseStapled",
			baseMetric: NewSSLOCSPResponseStapled("ssl_ocsp_response_stapled", "test", []string{}).baseMetrics,
		},
		{
			name: "SSLOCSPResponseStatus",
			baseMetric: NewSSLOCSPResponseStatus("ssl_ocsp_response_status", "test", []string{}).baseMetrics,
		},
		{
			name: "SSLOCSPResponseProducedAt",
			baseMetric: NewSSLOCSPResponseProducedAt("ssl_ocsp_response_produced_at", "test", []string{}).baseMetrics,
		},
		{
			name: "SSLOCSPResponseThisUpdate",
			baseMetric: NewSSLOCSPResponseThisUpdate("ssl_ocsp_response_this_update", "test", []string{}).baseMetrics,
		},
		{
			name: "SSLOCSPResponseNextUpdate",
			baseMetric: NewSSLOCSPResponseNextUpdate("ssl_ocsp_response_next_update", "test", []string{}).baseMetrics,
		},
		{
			name: "SSLOCSPResponseRevokedAt",
			baseMetric: NewSSLOCSPResponseRevokedAt("ssl_ocsp_response_revoked_at", "test", []string{}).baseMetrics,
		},
		{
			name: "SSLProbeSuccess",
			baseMetric: NewSSLProbeSuccess("ssl_probe_success", "test", []string{}).baseMetrics,
		},
		{
			name: "SSLProberType",
			baseMetric: NewSSLProberType("ssl_prober", "test", []string{"prober"}).baseMetrics,
		},
	}

	// 为每个指标测试collect方法
	for _, m := range metricsList {
		t.Run(m.name, func(t *testing.T) {
			ch := make(chan prometheus.Metric, 5)
			
			// 全局模拟数据
			metrics.probeSuccessValue = true
			metrics.proberTypeValue = "tcp"
			metrics.tlsVersionInfo = map[string]float64{"TLS 1.2": 1}
			metrics.certNotAfterValues = map[string]float64{"1|CN=test|test.com|test.com||": 1600000000}
			metrics.certNotBeforeValues = map[string]float64{"1|CN=test|test.com|test.com||": 1500000000}
			metrics.verifiedNotAfterValues = map[string]float64{"0|1|CN=test|test.com|test.com||": 1600000000}
			metrics.verifiedNotBeforeValues = map[string]float64{"0|1|CN=test|test.com|test.com||": 1500000000}
			metrics.ocspResponseStapledValue = 1
			metrics.ocspResponseStatusValue = 0
			metrics.ocspResponseProducedAtValue = 1590000000
			metrics.ocspResponseThisUpdateValue = 1590000000
			metrics.ocspResponseNextUpdateValue = 1600000000
			metrics.ocspResponseRevokedAtValue = 0

			// 测试基础收集方法
			testLabels := make([]string, len(m.baseMetric.labels))
			m.baseMetric.collect(ch, 1.0, testLabels)
			
			// 检查是否有指标生成
			select {
			case metric := <-ch:
				if metric == nil {
					t.Error("baseMetrics.collect() sent nil to the channel")
				}
			default:
				t.Error("baseMetrics.collect() didn't send any metrics to the channel")
			}
		})
	}
}

// 测试标签初始化
func TestSSLMetricsNamespace(t *testing.T) {
	if namespace != "ssl" {
		t.Errorf("namespace = %v, want %v", namespace, "ssl")
	}
} 
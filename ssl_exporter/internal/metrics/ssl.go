package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"ssl_exporter/internal/exporter"
)

const (
	namespace = "ssl"
)

func init() {
	exporter.Register(
		NewSSLCertNotAfter(prometheus.BuildFQName(namespace, "", "cert_not_after"),
			"NotAfter expressed as a Unix Epoch Time",
			[]string{"serial_no", "issuer_cn", "cn", "dnsnames", "ips", "emails", "ou"}))

	exporter.Register(
		NewSSLCertNotBefore(prometheus.BuildFQName(namespace, "", "cert_not_before"),
			"NotBefore expressed as a Unix Epoch Time",
			[]string{"serial_no", "issuer_cn", "cn", "dnsnames", "ips", "emails", "ou"}))

	exporter.Register(
		NewSSLVerifiedCertNotAfter(prometheus.BuildFQName(namespace, "", "verified_cert_not_after"),
			"NotAfter expressed as a Unix Epoch Time",
			[]string{"chain_no", "serial_no", "issuer_cn", "cn", "dnsnames", "ips", "emails", "ou"}))

	exporter.Register(
		NewSSLVerifiedCertNotBefore(prometheus.BuildFQName(namespace, "", "verified_cert_not_before"),
			"NotBefore expressed as a Unix Epoch Time",
			[]string{"chain_no", "serial_no", "issuer_cn", "cn", "dnsnames", "ips", "emails", "ou"}))

	exporter.Register(
		NewSSLTLSVersion(prometheus.BuildFQName(namespace, "", "tls_version_info"),
			"The TLS version used",
			[]string{"version"}))

	exporter.Register(
		NewSSLOCSPResponseStapled(prometheus.BuildFQName(namespace, "", "ocsp_response_stapled"),
			"If the connection state contains a stapled OCSP response",
			[]string{}))

	exporter.Register(
		NewSSLOCSPResponseStatus(prometheus.BuildFQName(namespace, "", "ocsp_response_status"),
			"The status in the OCSP response 0=Good 1=Revoked 2=Unknown",
			[]string{}))

	exporter.Register(
		NewSSLOCSPResponseProducedAt(prometheus.BuildFQName(namespace, "", "ocsp_response_produced_at"),
			"The producedAt value in the OCSP response, expressed as a Unix Epoch Time",
			[]string{}))

	exporter.Register(
		NewSSLOCSPResponseThisUpdate(prometheus.BuildFQName(namespace, "", "ocsp_response_this_update"),
			"The thisUpdate value in the OCSP response, expressed as a Unix Epoch Time",
			[]string{}))

	exporter.Register(
		NewSSLOCSPResponseNextUpdate(prometheus.BuildFQName(namespace, "", "ocsp_response_next_update"),
			"The nextUpdate value in the OCSP response, expressed as a Unix Epoch Time",
			[]string{}))

	exporter.Register(
		NewSSLOCSPResponseRevokedAt(prometheus.BuildFQName(namespace, "", "ocsp_response_revoked_at"),
			"The revocationTime value in the OCSP response, expressed as a Unix Epoch Time",
			[]string{}))

	exporter.Register(
		NewSSLProbeSuccess(prometheus.BuildFQName(namespace, "", "probe_success"),
			"If the probe was a success",
			[]string{}))

	exporter.Register(
		NewSSLProberType(prometheus.BuildFQName(namespace, "", "prober"),
			"The prober used by the exporter to connect to the target",
			[]string{"prober"}))
}

// SSL Cert Not After
type SSLCertNotAfter struct {
	*baseMetrics
}

func NewSSLCertNotAfter(fqname, help string, labels []string) *SSLCertNotAfter {
	return &SSLCertNotAfter{NewMetrics(fqname, help, labels)}
}

// SSL Cert Not Before
type SSLCertNotBefore struct {
	*baseMetrics
}

func NewSSLCertNotBefore(fqname, help string, labels []string) *SSLCertNotBefore {
	return &SSLCertNotBefore{NewMetrics(fqname, help, labels)}
}

// SSL Verified Cert Not After
type SSLVerifiedCertNotAfter struct {
	*baseMetrics
}

func NewSSLVerifiedCertNotAfter(fqname, help string, labels []string) *SSLVerifiedCertNotAfter {
	return &SSLVerifiedCertNotAfter{NewMetrics(fqname, help, labels)}
}

// SSL Verified Cert Not Before
type SSLVerifiedCertNotBefore struct {
	*baseMetrics
}

func NewSSLVerifiedCertNotBefore(fqname, help string, labels []string) *SSLVerifiedCertNotBefore {
	return &SSLVerifiedCertNotBefore{NewMetrics(fqname, help, labels)}
}

// SSL TLS Version
type SSLTLSVersion struct {
	*baseMetrics
}

func NewSSLTLSVersion(fqname, help string, labels []string) *SSLTLSVersion {
	return &SSLTLSVersion{NewMetrics(fqname, help, labels)}
}

// SSL OCSP Response Stapled
type SSLOCSPResponseStapled struct {
	*baseMetrics
}

func NewSSLOCSPResponseStapled(fqname, help string, labels []string) *SSLOCSPResponseStapled {
	return &SSLOCSPResponseStapled{NewMetrics(fqname, help, labels)}
}

// SSL OCSP Response Status
type SSLOCSPResponseStatus struct {
	*baseMetrics
}

func NewSSLOCSPResponseStatus(fqname, help string, labels []string) *SSLOCSPResponseStatus {
	return &SSLOCSPResponseStatus{NewMetrics(fqname, help, labels)}
}

// SSL OCSP Response ProducedAt
type SSLOCSPResponseProducedAt struct {
	*baseMetrics
}

func NewSSLOCSPResponseProducedAt(fqname, help string, labels []string) *SSLOCSPResponseProducedAt {
	return &SSLOCSPResponseProducedAt{NewMetrics(fqname, help, labels)}
}

// SSL OCSP Response ThisUpdate
type SSLOCSPResponseThisUpdate struct {
	*baseMetrics
}

func NewSSLOCSPResponseThisUpdate(fqname, help string, labels []string) *SSLOCSPResponseThisUpdate {
	return &SSLOCSPResponseThisUpdate{NewMetrics(fqname, help, labels)}
}

// SSL OCSP Response NextUpdate
type SSLOCSPResponseNextUpdate struct {
	*baseMetrics
}

func NewSSLOCSPResponseNextUpdate(fqname, help string, labels []string) *SSLOCSPResponseNextUpdate {
	return &SSLOCSPResponseNextUpdate{NewMetrics(fqname, help, labels)}
}

// SSL OCSP Response RevokedAt
type SSLOCSPResponseRevokedAt struct {
	*baseMetrics
}

func NewSSLOCSPResponseRevokedAt(fqname, help string, labels []string) *SSLOCSPResponseRevokedAt {
	return &SSLOCSPResponseRevokedAt{NewMetrics(fqname, help, labels)}
}

// SSL Probe Success
type SSLProbeSuccess struct {
	*baseMetrics
}

func NewSSLProbeSuccess(fqname, help string, labels []string) *SSLProbeSuccess {
	return &SSLProbeSuccess{NewMetrics(fqname, help, labels)}
}

// SSL Prober Type
type SSLProberType struct {
	*baseMetrics
}

func NewSSLProberType(fqname, help string, labels []string) *SSLProberType {
	return &SSLProberType{NewMetrics(fqname, help, labels)}
} 
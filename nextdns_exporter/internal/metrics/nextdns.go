package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"nextdns_exporter/internal/api"
)

// LoggerAdapter adapts logrus.Logger to api.Logger interface
type LoggerAdapter struct {
	*logrus.Logger
}

func (l *LoggerAdapter) Error(msg string, keysAndValues ...interface{}) {
	l.Logger.Errorln(msg, keysAndValues)
}

type NextDNSMetrics struct {
	baseMetrics

	// Totaled metrics
	totalQueries        *prometheus.Desc
	totalAllowedQueries *prometheus.Desc
	totalBlockedQueries *prometheus.Desc

	// Detailed metrics
	blockedQueries     *prometheus.Desc
	deviceQueries      *prometheus.Desc
	protocolQueries    *prometheus.Desc
	typeQueries        *prometheus.Desc
	ipVersionQueries   *prometheus.Desc
	dnssecQueries      *prometheus.Desc
	encryptedQueries   *prometheus.Desc
	destinationQueries *prometheus.Desc

	profile string
	apiKey  string
	client  api.Client
	logger  *logrus.Logger
}

func NewNextDNSMetrics(profile, apiKey string) *NextDNSMetrics {
	logger := logrus.New()
	m := &NextDNSMetrics{
		profile: profile,
		apiKey:  apiKey,
		logger:  logger,
	}

	// Initialize base metrics
	m.baseMetrics.init("nextdns")

	// Initialize client with logger adapter
	m.client = api.NewClient(api.BaseURL, profile, apiKey, &LoggerAdapter{logger})

	// Initialize totaled metrics
	m.totalQueries = prometheus.NewDesc(
		prometheus.BuildFQName("nextdns", "queries", "total"),
		"Total number of queries.",
		[]string{"profile"}, nil,
	)
	m.totalAllowedQueries = prometheus.NewDesc(
		prometheus.BuildFQName("nextdns", "allowed_queries", "total"),
		"Total number of allowed queries.",
		[]string{"profile"}, nil,
	)
	m.totalBlockedQueries = prometheus.NewDesc(
		prometheus.BuildFQName("nextdns", "blocked_queries", "total"),
		"Total number of blocked queries.",
		[]string{"profile"}, nil,
	)

	// Initialize detailed metrics
	m.blockedQueries = prometheus.NewDesc(
		prometheus.BuildFQName("nextdns", "blocked", "queries"),
		"Number of blocked queries per domain.",
		[]string{"profile", "domain", "root", "tracker"}, nil,
	)
	m.deviceQueries = prometheus.NewDesc(
		prometheus.BuildFQName("nextdns", "device", "queries"),
		"Number of queries per device.",
		[]string{"profile", "id", "name", "model", "local_ip"}, nil,
	)
	m.protocolQueries = prometheus.NewDesc(
		prometheus.BuildFQName("nextdns", "protocol", "queries"),
		"Number of queries per protocol.",
		[]string{"profile", "protocol"}, nil,
	)
	m.typeQueries = prometheus.NewDesc(
		prometheus.BuildFQName("nextdns", "type", "queries"),
		"Number of queries per type.",
		[]string{"profile", "type", "name"}, nil,
	)
	m.ipVersionQueries = prometheus.NewDesc(
		prometheus.BuildFQName("nextdns", "ip_version", "queries"),
		"Number of queries per IP version.",
		[]string{"profile", "version"}, nil,
	)
	m.dnssecQueries = prometheus.NewDesc(
		prometheus.BuildFQName("nextdns", "dnssec", "queries"),
		"Number of DNSSEC and non-DNSSEC queries.",
		[]string{"profile", "validated"}, nil,
	)
	m.encryptedQueries = prometheus.NewDesc(
		prometheus.BuildFQName("nextdns", "encrypted", "queries"),
		"Number of encrypted and unencrypted queries.",
		[]string{"profile", "encrypted"}, nil,
	)
	m.destinationQueries = prometheus.NewDesc(
		prometheus.BuildFQName("nextdns", "destination", "queries"),
		"Number of queries per geographic destination.",
		[]string{"profile", "code", "name"}, nil,
	)

	return m
}

func (m *NextDNSMetrics) Describe(ch chan<- *prometheus.Desc) {
	// Describe base metrics
	m.baseMetrics.Describe(ch)

	// Describe totaled metrics
	ch <- m.totalQueries
	ch <- m.totalAllowedQueries
	ch <- m.totalBlockedQueries

	// Describe detailed metrics
	ch <- m.blockedQueries
	ch <- m.deviceQueries
	ch <- m.protocolQueries
	ch <- m.typeQueries
	ch <- m.ipVersionQueries
	ch <- m.dnssecQueries
	ch <- m.encryptedQueries
	ch <- m.destinationQueries
}

func (m *NextDNSMetrics) Collect(ch chan<- prometheus.Metric) {
	// Collect base metrics
	m.baseMetrics.Collect(ch)

	// Collect totaled metrics
	status, err := m.client.CollectStatus()
	if err != nil {
		m.logger.Error("error collecting status data", "error", err)
		// 即使状态收集失败，也尝试收集其他指标
	} else {
		ch <- prometheus.MustNewConstMetric(m.totalQueries, prometheus.GaugeValue, status.TotalQueries, m.profile)
		ch <- prometheus.MustNewConstMetric(m.totalAllowedQueries, prometheus.GaugeValue, status.AllowedQueries, m.profile)
		ch <- prometheus.MustNewConstMetric(m.totalBlockedQueries, prometheus.GaugeValue, status.BlockedQueries, m.profile)
	}

	// Collect detailed metrics
	domains, err := m.client.CollectDomains()
	if err != nil {
		m.logger.Error("error collecting domains data", "error", err)
		// 继续收集其他指标
	} else if domains != nil && len(domains.BlockedDomains) > 0 {
		for _, domain := range domains.BlockedDomains {
			ch <- prometheus.MustNewConstMetric(
				m.blockedQueries,
				prometheus.GaugeValue,
				domain.Queries, m.profile, domain.Domain, domain.Root, domain.Tracker,
			)
		}
	}

	devices, err := m.client.CollectDevices()
	if err != nil {
		m.logger.Error("error collecting devices data", "error", err)
		// 继续收集其他指标
	} else if devices != nil && len(devices.Devices) > 0 {
		for _, device := range devices.Devices {
			ch <- prometheus.MustNewConstMetric(
				m.deviceQueries,
				prometheus.GaugeValue,
				device.Queries, m.profile, device.ID, device.Name, device.Model, device.LocalIP,
			)
		}
	}

	protocols, err := m.client.CollectProtocols()
	if err != nil {
		m.logger.Error("error collecting protocols data", "error", err)
		// 继续收集其他指标
	} else if protocols != nil && len(protocols.Protocols) > 0 {
		for _, protocol := range protocols.Protocols {
			ch <- prometheus.MustNewConstMetric(
				m.protocolQueries,
				prometheus.GaugeValue,
				protocol.Queries, m.profile, protocol.Protocol,
			)
		}
	}

	queryTypes, err := m.client.CollectQueryTypes()
	if err != nil {
		m.logger.Error("error collecting query types data", "error", err)
		// 继续收集其他指标
	} else if queryTypes != nil && len(queryTypes.QueryTypes) > 0 {
		for _, queryType := range queryTypes.QueryTypes {
			ch <- prometheus.MustNewConstMetric(
				m.typeQueries,
				prometheus.GaugeValue,
				queryType.Queries, m.profile, queryType.Type, queryType.Name,
			)
		}
	}

	ipVersions, err := m.client.CollectIPVersions()
	if err != nil {
		m.logger.Error("error collecting IP versions data", "error", err)
		// 继续收集其他指标
	} else if ipVersions != nil && len(ipVersions.IPVersions) > 0 {
		for _, ipVersion := range ipVersions.IPVersions {
			ch <- prometheus.MustNewConstMetric(
				m.ipVersionQueries,
				prometheus.GaugeValue,
				ipVersion.Queries, m.profile, ipVersion.Version,
			)
		}
	}

	dnssec, err := m.client.CollectDNSSEC()
	if err != nil {
		m.logger.Error("error collecting DNSSEC data", "error", err)
		// 继续收集其他指标
	} else if dnssec != nil && len(dnssec.Data) > 0 {
		for _, data := range dnssec.Data {
			ch <- prometheus.MustNewConstMetric(
				m.dnssecQueries,
				prometheus.GaugeValue,
				data.Queries, m.profile, data.Validated,
			)
		}
	}

	encryption, err := m.client.CollectEncryption()
	if err != nil {
		m.logger.Error("error collecting encryption data", "error", err)
		// 继续收集其他指标
	} else if encryption != nil && len(encryption.Data) > 0 {
		for _, data := range encryption.Data {
			ch <- prometheus.MustNewConstMetric(
				m.encryptedQueries,
				prometheus.GaugeValue,
				data.Queries, m.profile, data.Encrypted,
			)
		}
	}

	destinations, err := m.client.CollectDestinations()
	if err != nil {
		m.logger.Error("error collecting destinations data", "error", err)
		// 继续收集其他指标
	} else if destinations != nil && len(destinations.Destinations) > 0 {
		for _, destination := range destinations.Destinations {
			ch <- prometheus.MustNewConstMetric(
				m.destinationQueries,
				prometheus.GaugeValue,
				destination.Queries, m.profile, destination.Code, destination.Name,
			)
		}
	}
} 
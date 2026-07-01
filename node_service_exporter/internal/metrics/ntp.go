//go:build !nontp
// +build !nontp

package metrics

import (
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"
	"node_service_exporter/internal/exporter"

	"github.com/beevik/ntp"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	exporter.Register(NewNTPCollectorWrapper())
}

const (
	hour24       = 24 * time.Hour // `time` does not export `Day` as Day != 24h because of DST
	ntpSubsystem = "ntp"
)

var (
	leapMidnight      time.Time
	leapMidnightMutex = &sync.Mutex{}
)

// NTPCollectorWrapper wraps the old collector to work with new framework
type NTPCollectorWrapper struct {
	collector *NTPCollector
}

func NewNTPCollectorWrapper() *NTPCollectorWrapper {
	collector, err := NewNTPCollector(nil, nil)
	if err != nil {
		return nil
	}
	return &NTPCollectorWrapper{
		collector: collector,
	}
}

func (n *NTPCollectorWrapper) Collect(ch chan<- prometheus.Metric) {
	if n.collector != nil {
		if err := n.collector.Collect(ch); err != nil {
			fmt.Printf("Error collecting metrics: %v\n", err)
		}
	}
}

// NTPConfig holds NTP collector configuration
type NTPConfig struct {
	Server             string
	ServerPort         int
	ProtocolVersion    int
	ServerIsLocal      bool
	IPTTL              int
	MaxDistance        time.Duration
	OffsetTolerance    time.Duration
}

// DefaultNTPConfig returns default NTP configuration
func DefaultNTPConfig() *NTPConfig {
	return &NTPConfig{
		Server:             "127.0.0.1",
		ServerPort:         123,
		ProtocolVersion:    4,
		ServerIsLocal:      false,
		IPTTL:              1,
		MaxDistance:        3466080000, // 3.46608s in nanoseconds
		OffsetTolerance:    1000000,    // 1ms in nanoseconds
	}
}

// NTPCollector collects NTP-related metrics
type NTPCollector struct {
	stratum, leap, rtt, offset, reftime, rootDelay, rootDispersion, sanity typedDesc
	config                                                                 *NTPConfig
	logger                                                                 *slog.Logger
}

// NewNTPCollector creates a new NTP collector
func NewNTPCollector(logger *slog.Logger, config *NTPConfig) (*NTPCollector, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if config == nil {
		config = DefaultNTPConfig()
	}

	// Validate configuration
	ipaddr := net.ParseIP(config.Server)
	if !config.ServerIsLocal && (ipaddr == nil || !ipaddr.IsLoopback()) {
		return nil, fmt.Errorf("only IP address of local NTP server is valid for NTP server")
	}

	if config.ProtocolVersion < 2 || config.ProtocolVersion > 4 {
		return nil, fmt.Errorf("invalid NTP protocol version %d; must be 2, 3, or 4", config.ProtocolVersion)
	}

	if config.OffsetTolerance < 0 {
		return nil, fmt.Errorf("offset tolerance must be non-negative")
	}

	if config.ServerPort < 1 || config.ServerPort > 65535 {
		return nil, fmt.Errorf("invalid NTP port number %d; must be between 1 and 65535 inclusive", config.ServerPort)
	}

	logger.Warn("NTP collector is deprecated and may be removed in future versions.")

	return &NTPCollector{
		stratum: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, ntpSubsystem, "stratum"),
				"NTPD stratum.",
				nil,
				nil,
			),
			valueType: prometheus.GaugeValue,
		},
		leap: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, ntpSubsystem, "leap"),
				"NTPD leap second indicator, 2 bits.",
				nil,
				nil,
			),
			valueType: prometheus.GaugeValue,
		},
		rtt: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, ntpSubsystem, "rtt_seconds"),
				"RTT to NTPD.",
				nil,
				nil,
			),
			valueType: prometheus.GaugeValue,
		},
		offset: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, ntpSubsystem, "offset_seconds"),
				"ClockOffset between NTP and local clock.",
				nil,
				nil,
			),
			valueType: prometheus.GaugeValue,
		},
		reftime: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, ntpSubsystem, "reference_timestamp_seconds"),
				"NTPD ReferenceTime, UNIX timestamp.",
				nil,
				nil,
			),
			valueType: prometheus.GaugeValue,
		},
		rootDelay: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, ntpSubsystem, "root_delay_seconds"),
				"NTPD RootDelay.",
				nil,
				nil,
			),
			valueType: prometheus.GaugeValue,
		},
		rootDispersion: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, ntpSubsystem, "root_dispersion_seconds"),
				"NTPD RootDispersion.",
				nil,
				nil,
			),
			valueType: prometheus.GaugeValue,
		},
		sanity: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, ntpSubsystem, "sanity"),
				"NTPD sanity according to RFC5905 heuristics and configured limits.",
				nil,
				nil,
			),
			valueType: prometheus.GaugeValue,
		},
		config: config,
		logger: logger,
	}, nil
}

// Collect implements the Collector interface
func (c *NTPCollector) Collect(ch chan<- prometheus.Metric) error {
	if c == nil {
		return fmt.Errorf("NTPCollector is nil")
	}

	resp, err := ntp.QueryWithOptions(c.config.Server, ntp.QueryOptions{
		Version: c.config.ProtocolVersion,
		TTL:     c.config.IPTTL,
		Timeout: time.Second, // default `ntpdate` timeout
		Port:    c.config.ServerPort,
	})
	if err != nil {
		return fmt.Errorf("couldn't get SNTP reply from %s: %w", c.config.Server, err)
	}

	ch <- c.stratum.mustNewConstMetric(float64(resp.Stratum))
	ch <- c.leap.mustNewConstMetric(float64(resp.Leap))
	ch <- c.rtt.mustNewConstMetric(resp.RTT.Seconds())
	ch <- c.offset.mustNewConstMetric(resp.ClockOffset.Seconds())

	if resp.ReferenceTime.Unix() > 0 {
		// Go Zero is   0001-01-01 00:00:00 UTC
		// NTP Zero is  1900-01-01 00:00:00 UTC
		// UNIX Zero is 1970-01-01 00:00:00 UTC
		// so let's keep ALL ancient `reftime` values as zero
		ch <- c.reftime.mustNewConstMetric(float64(resp.ReferenceTime.UnixNano()) / 1e9)
	} else {
		ch <- c.reftime.mustNewConstMetric(0)
	}

	ch <- c.rootDelay.mustNewConstMetric(resp.RootDelay.Seconds())
	ch <- c.rootDispersion.mustNewConstMetric(resp.RootDispersion.Seconds())

	// Here is SNTP packet sanity check that is exposed to move burden of
	// configuration from node_exporter user to the developer.
	maxerr := time.Duration(c.config.OffsetTolerance)
	leapMidnightMutex.Lock()
	if resp.Leap == ntp.LeapAddSecond || resp.Leap == ntp.LeapDelSecond {
		// state of leapMidnight is cached as leap flag is dropped right after midnight
		leapMidnight = resp.Time.Truncate(hour24).Add(hour24)
	}
	if leapMidnight.Add(-hour24).Before(resp.Time) && resp.Time.Before(leapMidnight.Add(hour24)) {
		// tolerate leap smearing
		maxerr += time.Second
	}
	leapMidnightMutex.Unlock()

	maxDistance := time.Duration(c.config.MaxDistance)
	if resp.Validate() == nil && resp.RootDistance <= maxDistance && resp.MinError <= maxerr {
		ch <- c.sanity.mustNewConstMetric(1)
	} else {
		ch <- c.sanity.mustNewConstMetric(0)
	}

	return nil
} 

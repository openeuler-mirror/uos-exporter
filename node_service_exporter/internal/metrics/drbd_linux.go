//go:build !nodrbd
// +build !nodrbd

package metrics

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"node_service_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	exporter.Register(NewDRBDCollectorWrapper())
}

// DRBDCollectorWrapper wraps the old collector to work with new framework
type DRBDCollectorWrapper struct {
	collector *DRBDCollector
}

func NewDRBDCollectorWrapper() *DRBDCollectorWrapper {
	collector, err := NewDRBDCollector(nil)
	if err != nil {
		return nil
	}
	return &DRBDCollectorWrapper{
		collector: collector,
	}
}

func (d *DRBDCollectorWrapper) Collect(ch chan<- prometheus.Metric) {
	if d.collector != nil {
		if err := d.collector.Collect(ch); err != nil {
			fmt.Printf("Error collecting metrics: %v\n", err)
		}
	}
}

// DRBDNumericalMetric represents a numerical metric from /proc/drbd
type DRBDNumericalMetric struct {
	*baseMetrics
	valueType  prometheus.ValueType
	multiplier float64
}

// newDRBDNumericalMetric creates a new DRBD numerical metric
func newDRBDNumericalMetric(name, help string, valueType prometheus.ValueType, multiplier float64) *DRBDNumericalMetric {
	return &DRBDNumericalMetric{
		baseMetrics: &baseMetrics{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "drbd", name),
				help,
				[]string{"device"},
				nil,
			),
		},
		valueType:  valueType,
		multiplier: multiplier,
	}
}

// Collect implements the Collect method for DRBDNumericalMetric
func (d *DRBDNumericalMetric) Collect(ch chan<- prometheus.Metric, value float64, device string) {
	ch <- prometheus.MustNewConstMetric(
		d.desc,
		d.valueType,
		value*d.multiplier,
		device,
	)
}

// DRBDStringPairMetric represents a string pair metric from /proc/drbd
type DRBDStringPairMetric struct {
	*baseMetrics
	valueOK string
}

// newDRBDStringPairMetric creates a new DRBD string pair metric
func newDRBDStringPairMetric(name, help, valueOK string) *DRBDStringPairMetric {
	return &DRBDStringPairMetric{
		baseMetrics: &baseMetrics{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "drbd", name),
				help,
				[]string{"device", "node"},
				nil,
			),
		},
		valueOK: valueOK,
	}
}

// isOkay checks if the value matches the expected OK value
func (d *DRBDStringPairMetric) isOkay(v string) float64 {
	if v == d.valueOK {
		return 1
	}
	return 0
}

// Collect implements the Collect method for DRBDStringPairMetric
func (d *DRBDStringPairMetric) Collect(ch chan<- prometheus.Metric, localValue, remoteValue, device string) {
	ch <- prometheus.MustNewConstMetric(
		d.desc,
		prometheus.GaugeValue,
		d.isOkay(localValue),
		device,
		"local",
	)
	ch <- prometheus.MustNewConstMetric(
		d.desc,
		prometheus.GaugeValue,
		d.isOkay(remoteValue),
		device,
		"remote",
	)
}

// DRBDConnectedMetric represents the DRBD connection state metric
type DRBDConnectedMetric struct {
	*baseMetrics
}

// newDRBDConnectedMetric creates a new DRBD connected metric
func newDRBDConnectedMetric() *DRBDConnectedMetric {
	return &DRBDConnectedMetric{
		baseMetrics: &baseMetrics{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "drbd", "connected"),
				"Whether DRBD is connected to the peer.",
				[]string{"device"},
				nil,
			),
		},
	}
}

// Collect implements the Collect method for DRBDConnectedMetric
func (d *DRBDConnectedMetric) Collect(ch chan<- prometheus.Metric, connected float64, device string) {
	ch <- prometheus.MustNewConstMetric(
		d.desc,
		prometheus.GaugeValue,
		connected,
		device,
	)
}

// DRBDCollector collects DRBD metrics
type DRBDCollector struct {
	numerical  map[string]*DRBDNumericalMetric
	stringPair map[string]*DRBDStringPairMetric
	connected  *DRBDConnectedMetric
	logger     *slog.Logger
}

// NewDRBDCollector creates a new DRBD collector
func NewDRBDCollector(logger *slog.Logger) (*DRBDCollector, error) {
	if logger == nil {
		logger = slog.Default()
	}

	return &DRBDCollector{
		numerical: map[string]*DRBDNumericalMetric{
			"ns": newDRBDNumericalMetric(
				"network_sent_bytes_total",
				"Total number of bytes sent via the network.",
				prometheus.CounterValue,
				1024,
			),
			"nr": newDRBDNumericalMetric(
				"network_received_bytes_total",
				"Total number of bytes received via the network.",
				prometheus.CounterValue,
				1,
			),
			"dw": newDRBDNumericalMetric(
				"disk_written_bytes_total",
				"Net data written on local hard disk; in bytes.",
				prometheus.CounterValue,
				1024,
			),
			"dr": newDRBDNumericalMetric(
				"disk_read_bytes_total",
				"Net data read from local hard disk; in bytes.",
				prometheus.CounterValue,
				1024,
			),
			"al": newDRBDNumericalMetric(
				"activitylog_writes_total",
				"Number of updates of the activity log area of the meta data.",
				prometheus.CounterValue,
				1,
			),
			"bm": newDRBDNumericalMetric(
				"bitmap_writes_total",
				"Number of updates of the bitmap area of the meta data.",
				prometheus.CounterValue,
				1,
			),
			"lo": newDRBDNumericalMetric(
				"local_pending",
				"Number of open requests to the local I/O sub-system.",
				prometheus.GaugeValue,
				1,
			),
			"pe": newDRBDNumericalMetric(
				"remote_pending",
				"Number of requests sent to the peer, but that have not yet been answered by the latter.",
				prometheus.GaugeValue,
				1,
			),
			"ua": newDRBDNumericalMetric(
				"remote_unacknowledged",
				"Number of requests received by the peer via the network connection, but that have not yet been answered.",
				prometheus.GaugeValue,
				1,
			),
			"ap": newDRBDNumericalMetric(
				"application_pending",
				"Number of block I/O requests forwarded to DRBD, but not yet answered by DRBD.",
				prometheus.GaugeValue,
				1,
			),
			"ep": newDRBDNumericalMetric(
				"epochs",
				"Number of Epochs currently on the fly.",
				prometheus.GaugeValue,
				1,
			),
			"oos": newDRBDNumericalMetric(
				"out_of_sync_bytes",
				"Amount of data known to be out of sync; in bytes.",
				prometheus.GaugeValue,
				1024,
			),
		},

		stringPair: map[string]*DRBDStringPairMetric{
			"ro": newDRBDStringPairMetric(
				"node_role_is_primary",
				"Whether the role of the node is in the primary state.",
				"Primary",
			),
			"ds": newDRBDStringPairMetric(
				"disk_state_is_up_to_date",
				"Whether the disk of the node is up to date.",
				"UpToDate",
			),
		},

		connected: newDRBDConnectedMetric(),
		logger:    logger,
	}, nil
}

// Collect implements the Collector interface
func (c *DRBDCollector) Collect(ch chan<- prometheus.Metric) error {
	if c == nil {
		return fmt.Errorf("DRBDCollector is nil")
	}

	statsFile := "/proc/drbd"
	file, err := os.Open(statsFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			c.logger.Debug("DRBD stats file does not exist, skipping", "file", statsFile, "err", err)
			return nil
		}
		return fmt.Errorf("failed to open DRBD stats file %s: %w", statsFile, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanWords)
	device := "unknown"

	for scanner.Scan() {
		field := scanner.Text()
		if field == "" {
			continue
		}

		kv := strings.Split(field, ":")
		if len(kv) != 2 {
			c.logger.Debug("skipping invalid key:value pair", "field", field)
			continue
		}

		key, value := kv[0], kv[1]

		// Check for new DRBD device
		if id, err := strconv.ParseUint(key, 10, 64); err == nil && value == "" {
			device = fmt.Sprintf("drbd%d", id)
			continue
		}

		// Handle numerical metrics
		if m, ok := c.numerical[key]; ok {
			v, err := strconv.ParseFloat(value, 64)
			if err != nil {
				c.logger.Debug("failed to parse numerical value", "key", key, "value", value, "err", err)
				continue
			}
			m.Collect(ch, v, device)
			continue
		}

		// Handle string pair metrics
		if m, ok := c.stringPair[key]; ok {
			values := strings.Split(value, "/")
			if len(values) != 2 {
				c.logger.Debug("invalid string pair format", "key", key, "value", value)
				continue
			}
			m.Collect(ch, values[0], values[1], device)
			continue
		}

		// Handle connection state
		if key == "cs" {
			var connected float64
			if value == "Connected" {
				connected = 1
			}
			c.connected.Collect(ch, connected, device)
			continue
		}

		c.logger.Debug("unhandled key-value pair", "key", key, "value", value)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error scanning DRBD stats: %w", err)
	}

	return nil
} 

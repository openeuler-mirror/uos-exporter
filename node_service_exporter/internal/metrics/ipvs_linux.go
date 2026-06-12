//go:build !noipvs && linux
// +build !noipvs,linux

package metrics

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"node_service_exporter/internal/exporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
)

const (
	ipvsLabelLocalAddress  = "local_address"
	ipvsLabelLocalPort     = "local_port"
	ipvsLabelRemoteAddress = "remote_address"
	ipvsLabelRemotePort    = "remote_port"
	ipvsLabelProto         = "proto"
	ipvsLabelLocalMark     = "local_mark"
)

var (
	fullIpvsBackendLabels = []string{
		ipvsLabelLocalAddress,
		ipvsLabelLocalPort,
		ipvsLabelRemoteAddress,
		ipvsLabelRemotePort,
		ipvsLabelProto,
		ipvsLabelLocalMark,
	}
	// Default to all labels for compatibility
	ipvsBackendLabels = fullIpvsBackendLabels
)

type ipvsBackendStatus struct {
	ActiveConn uint64
	InactConn  uint64
	Weight     uint64
}

func init() {
	exporter.Register(NewIPVSConnections())
	exporter.Register(NewIPVSIncomingPackets())
	exporter.Register(NewIPVSOutgoingPackets())
	exporter.Register(NewIPVSIncomingBytes())
	exporter.Register(NewIPVSOutgoingBytes())
	exporter.Register(NewIPVSBackendConnectionsActive())
	exporter.Register(NewIPVSBackendConnectionsInactive())
	exporter.Register(NewIPVSBackendWeight())
}

var ipvsFS procfs.FS
var ipvsInitialized bool

func initIPVSFS() error {
	if !ipvsInitialized {
		fs, err := procfs.NewFS("/proc")
		if err != nil {
			return fmt.Errorf("failed to open procfs: %w", err)
		}
		ipvsFS = fs
		ipvsInitialized = true
	}
	return nil
}

// IPVS Connections
type IPVSConnections struct {
	*baseMetrics
}

func NewIPVSConnections() *IPVSConnections {
	return &IPVSConnections{
		NewMetrics("node_ipvs_connections_total",
			"The total number of connections made.",
			nil),
	}
}

func (i *IPVSConnections) Collect(ch chan<- prometheus.Metric) {
	if err := initIPVSFS(); err != nil {
		return
	}

	ipvsStats, err := ipvsFS.IPVSStats()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	i.baseMetrics.collectCounter(ch, float64(ipvsStats.Connections), nil)
}

// IPVS Incoming Packets
type IPVSIncomingPackets struct {
	*baseMetrics
}

func NewIPVSIncomingPackets() *IPVSIncomingPackets {
	return &IPVSIncomingPackets{
		NewMetrics("node_ipvs_incoming_packets_total",
			"The total number of incoming packets.",
			nil),
	}
}

func (i *IPVSIncomingPackets) Collect(ch chan<- prometheus.Metric) {
	if err := initIPVSFS(); err != nil {
		return
	}

	ipvsStats, err := ipvsFS.IPVSStats()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	i.baseMetrics.collectCounter(ch, float64(ipvsStats.IncomingPackets), nil)
}

// IPVS Outgoing Packets
type IPVSOutgoingPackets struct {
	*baseMetrics
}

func NewIPVSOutgoingPackets() *IPVSOutgoingPackets {
	return &IPVSOutgoingPackets{
		NewMetrics("node_ipvs_outgoing_packets_total",
			"The total number of outgoing packets.",
			nil),
	}
}

func (i *IPVSOutgoingPackets) Collect(ch chan<- prometheus.Metric) {
	if err := initIPVSFS(); err != nil {
		return
	}

	ipvsStats, err := ipvsFS.IPVSStats()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	i.baseMetrics.collectCounter(ch, float64(ipvsStats.OutgoingPackets), nil)
}

// IPVS Incoming Bytes
type IPVSIncomingBytes struct {
	*baseMetrics
}

func NewIPVSIncomingBytes() *IPVSIncomingBytes {
	return &IPVSIncomingBytes{
		NewMetrics("node_ipvs_incoming_bytes_total",
			"The total amount of incoming data.",
			nil),
	}
}

func (i *IPVSIncomingBytes) Collect(ch chan<- prometheus.Metric) {
	if err := initIPVSFS(); err != nil {
		return
	}

	ipvsStats, err := ipvsFS.IPVSStats()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	i.baseMetrics.collectCounter(ch, float64(ipvsStats.IncomingBytes), nil)
}

// IPVS Outgoing Bytes
type IPVSOutgoingBytes struct {
	*baseMetrics
}

func NewIPVSOutgoingBytes() *IPVSOutgoingBytes {
	return &IPVSOutgoingBytes{
		NewMetrics("node_ipvs_outgoing_bytes_total",
			"The total amount of outgoing data.",
			nil),
	}
}

func (i *IPVSOutgoingBytes) Collect(ch chan<- prometheus.Metric) {
	if err := initIPVSFS(); err != nil {
		return
	}

	ipvsStats, err := ipvsFS.IPVSStats()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	i.baseMetrics.collectCounter(ch, float64(ipvsStats.OutgoingBytes), nil)
}

// IPVS Backend Connections Active
type IPVSBackendConnectionsActive struct {
	*baseMetrics
}

func NewIPVSBackendConnectionsActive() *IPVSBackendConnectionsActive {
	return &IPVSBackendConnectionsActive{
		NewMetrics("node_ipvs_backend_connections_active",
			"The current active connections by local and remote address.",
			ipvsBackendLabels),
	}
}

func (i *IPVSBackendConnectionsActive) Collect(ch chan<- prometheus.Metric) {
	if err := initIPVSFS(); err != nil {
		return
	}

	backendStats, err := ipvsFS.IPVSBackendStatus()
	if err != nil {
		return
	}

	sums, labelValues := processIPVSBackends(backendStats, ipvsBackendLabels)
	for key, status := range sums {
		kv := labelValues[key]
		i.baseMetrics.collect(ch, float64(status.ActiveConn), kv)
	}
}

// IPVS Backend Connections Inactive
type IPVSBackendConnectionsInactive struct {
	*baseMetrics
}

func NewIPVSBackendConnectionsInactive() *IPVSBackendConnectionsInactive {
	return &IPVSBackendConnectionsInactive{
		NewMetrics("node_ipvs_backend_connections_inactive",
			"The current inactive connections by local and remote address.",
			ipvsBackendLabels),
	}
}

func (i *IPVSBackendConnectionsInactive) Collect(ch chan<- prometheus.Metric) {
	if err := initIPVSFS(); err != nil {
		return
	}

	backendStats, err := ipvsFS.IPVSBackendStatus()
	if err != nil {
		return
	}

	sums, labelValues := processIPVSBackends(backendStats, ipvsBackendLabels)
	for key, status := range sums {
		kv := labelValues[key]
		i.baseMetrics.collect(ch, float64(status.InactConn), kv)
	}
}

// IPVS Backend Weight
type IPVSBackendWeight struct {
	*baseMetrics
}

func NewIPVSBackendWeight() *IPVSBackendWeight {
	return &IPVSBackendWeight{
		NewMetrics("node_ipvs_backend_weight",
			"The current backend weight by local and remote address.",
			ipvsBackendLabels),
	}
}

func (i *IPVSBackendWeight) Collect(ch chan<- prometheus.Metric) {
	if err := initIPVSFS(); err != nil {
		return
	}

	backendStats, err := ipvsFS.IPVSBackendStatus()
	if err != nil {
		return
	}

	sums, labelValues := processIPVSBackends(backendStats, ipvsBackendLabels)
	for key, status := range sums {
		kv := labelValues[key]
		i.baseMetrics.collect(ch, float64(status.Weight), kv)
	}
}

// Helper function to process IPVS backend statistics
func processIPVSBackends(backendStats []procfs.IPVSBackendStatus, backendLabels []string) (map[string]ipvsBackendStatus, map[string][]string) {
	sums := map[string]ipvsBackendStatus{}
	labelValues := map[string][]string{}
	
	for _, backend := range backendStats {
		localAddress := ""
		if backend.LocalAddress.String() != "<nil>" {
			localAddress = backend.LocalAddress.String()
		}
		
		kv := make([]string, len(backendLabels))
		for i, label := range backendLabels {
			var labelValue string
			switch label {
			case ipvsLabelLocalAddress:
				labelValue = localAddress
			case ipvsLabelLocalPort:
				labelValue = strconv.FormatUint(uint64(backend.LocalPort), 10)
			case ipvsLabelRemoteAddress:
				labelValue = backend.RemoteAddress.String()
			case ipvsLabelRemotePort:
				labelValue = strconv.FormatUint(uint64(backend.RemotePort), 10)
			case ipvsLabelProto:
				labelValue = backend.Proto
			case ipvsLabelLocalMark:
				labelValue = backend.LocalMark
			}
			kv[i] = labelValue
		}
		
		key := strings.Join(kv, "-")
		status := sums[key]
		status.ActiveConn += backend.ActiveConn
		status.InactConn += backend.InactConn
		status.Weight += backend.Weight
		sums[key] = status
		labelValues[key] = kv
	}
	
	return sums, labelValues
}

// Helper function to parse IPVS labels (for future extensibility)
func parseIpvsLabels(labelString string) ([]string, error) {
	labels := strings.Split(labelString, ",")
	labelSet := make(map[string]bool, len(labels))
	results := make([]string, 0, len(labels))
	
	for _, label := range labels {
		if label != "" {
			labelSet[label] = true
		}
	}

	for _, label := range fullIpvsBackendLabels {
		if labelSet[label] {
			results = append(results, label)
		}
		delete(labelSet, label)
	}

	if len(labelSet) > 0 {
		keys := make([]string, 0, len(labelSet))
		for label := range labelSet {
			keys = append(keys, label)
		}
		sort.Strings(keys)
		return nil, fmt.Errorf("unknown IPVS backend labels: %q", strings.Join(keys, ", "))
	}

	return results, nil
} 
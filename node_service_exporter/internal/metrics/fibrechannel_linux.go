//go:build !nofibrechannel && linux
// +build !nofibrechannel,linux

package metrics

import (
	"fmt"
	"os"
	"node_service_exporter/internal/exporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs/sysfs"
)

const maxUint64 = ^uint64(0)

func init() {
	exporter.Register(NewFibreChannelInfo())
	exporter.Register(NewFibreChannelDumpedFrames())
	exporter.Register(NewFibreChannelLossOfSignal())
	exporter.Register(NewFibreChannelLossOfSync())
	exporter.Register(NewFibreChannelRxFrames())
	exporter.Register(NewFibreChannelErrorFrames())
	exporter.Register(NewFibreChannelInvalidTxWords())
	exporter.Register(NewFibreChannelSecondsSinceLastReset())
	exporter.Register(NewFibreChannelTxWords())
	exporter.Register(NewFibreChannelInvalidCRC())
	exporter.Register(NewFibreChannelNos())
	exporter.Register(NewFibreChannelFcpPacketAborts())
	exporter.Register(NewFibreChannelRxWords())
	exporter.Register(NewFibreChannelTxFrames())
	exporter.Register(NewFibreChannelLinkFailure())
}

var fibrechannelFS sysfs.FS
var fibrechannelInitialized bool

func initFibreChannelFS() error {
	if !fibrechannelInitialized {
		fs, err := sysfs.NewFS("/sys")
		if err != nil {
			return fmt.Errorf("failed to open sysfs: %w", err)
		}
		fibrechannelFS = fs
		fibrechannelInitialized = true
	}
	return nil
}

// Helper function to check if counter is implemented (not maxUint64)
func isCounterImplemented(value uint64) bool {
	return value != maxUint64
}

// FibreChannel Info
type FibreChannelInfo struct {
	*baseMetrics
}

func NewFibreChannelInfo() *FibreChannelInfo {
	return &FibreChannelInfo{
		NewMetrics("node_fibrechannel_info",
			"Non-numeric data from /sys/class/fc_host/<host>, value is always 1.",
			[]string{"fc_host", "speed", "port_state", "port_type", "port_id", "port_name", "fabric_name", "symbolic_name", "supported_classes", "supported_speeds", "dev_loss_tmo"}),
	}
}

func (f *FibreChannelInfo) Collect(ch chan<- prometheus.Metric) {
	if err := initFibreChannelFS(); err != nil {
		return
	}

	hosts, err := fibrechannelFS.FibreChannelClass()
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		return
	}

	for _, host := range hosts {
		// Helper function to safely dereference string pointers
		safeDereference := func(ptrs ...*string) []string {
			result := make([]string, len(ptrs))
			for i, ptr := range ptrs {
				if ptr != nil {
					result[i] = *ptr
				} else {
					result[i] = ""
				}
			}
			return result
		}

		labels := safeDereference(
			host.Name,
			host.Speed,
			host.PortState,
			host.PortType,
			host.PortID,
			host.PortName,
			host.FabricName,
			host.SymbolicName,
			host.SupportedClasses,
			host.SupportedSpeeds,
			host.DevLossTMO,
		)

		f.baseMetrics.collect(ch, 1.0, labels)
	}
}

// FibreChannel Dumped Frames
type FibreChannelDumpedFrames struct {
	*baseMetrics
}

func NewFibreChannelDumpedFrames() *FibreChannelDumpedFrames {
	return &FibreChannelDumpedFrames{
		NewMetrics("node_fibrechannel_dumped_frames_total",
			"Number of dumped frames",
			[]string{"fc_host"}),
	}
}

func (f *FibreChannelDumpedFrames) Collect(ch chan<- prometheus.Metric) {
	if err := initFibreChannelFS(); err != nil {
		return
	}

	hosts, err := fibrechannelFS.FibreChannelClass()
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		return
	}

	for _, host := range hosts {
		if host.Name != nil && host.Counters.DumpedFrames != nil {
			if isCounterImplemented(*host.Counters.DumpedFrames) {
				f.baseMetrics.collectCounter(ch, float64(*host.Counters.DumpedFrames), []string{*host.Name})
			}
		}
	}
}

// FibreChannel Loss of Signal
type FibreChannelLossOfSignal struct {
	*baseMetrics
}

func NewFibreChannelLossOfSignal() *FibreChannelLossOfSignal {
	return &FibreChannelLossOfSignal{
		NewMetrics("node_fibrechannel_loss_of_signal_total",
			"Number of times signal has been lost",
			[]string{"fc_host"}),
	}
}

func (f *FibreChannelLossOfSignal) Collect(ch chan<- prometheus.Metric) {
	if err := initFibreChannelFS(); err != nil {
		return
	}

	hosts, err := fibrechannelFS.FibreChannelClass()
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		return
	}

	for _, host := range hosts {
		if host.Name != nil && host.Counters.LossOfSignalCount != nil {
			if isCounterImplemented(*host.Counters.LossOfSignalCount) {
				f.baseMetrics.collectCounter(ch, float64(*host.Counters.LossOfSignalCount), []string{*host.Name})
			}
		}
	}
}

// FibreChannel Loss of Sync
type FibreChannelLossOfSync struct {
	*baseMetrics
}

func NewFibreChannelLossOfSync() *FibreChannelLossOfSync {
	return &FibreChannelLossOfSync{
		NewMetrics("node_fibrechannel_loss_of_sync_total",
			"Number of failures on either bit or transmission word boundaries",
			[]string{"fc_host"}),
	}
}

func (f *FibreChannelLossOfSync) Collect(ch chan<- prometheus.Metric) {
	if err := initFibreChannelFS(); err != nil {
		return
	}

	hosts, err := fibrechannelFS.FibreChannelClass()
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		return
	}

	for _, host := range hosts {
		if host.Name != nil && host.Counters.LossOfSyncCount != nil {
			if isCounterImplemented(*host.Counters.LossOfSyncCount) {
				f.baseMetrics.collectCounter(ch, float64(*host.Counters.LossOfSyncCount), []string{*host.Name})
			}
		}
	}
}

// FibreChannel RX Frames
type FibreChannelRxFrames struct {
	*baseMetrics
}

func NewFibreChannelRxFrames() *FibreChannelRxFrames {
	return &FibreChannelRxFrames{
		NewMetrics("node_fibrechannel_rx_frames_total",
			"Number of frames received",
			[]string{"fc_host"}),
	}
}

func (f *FibreChannelRxFrames) Collect(ch chan<- prometheus.Metric) {
	if err := initFibreChannelFS(); err != nil {
		return
	}

	hosts, err := fibrechannelFS.FibreChannelClass()
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		return
	}

	for _, host := range hosts {
		if host.Name != nil && host.Counters.RXFrames != nil {
			if isCounterImplemented(*host.Counters.RXFrames) {
				f.baseMetrics.collectCounter(ch, float64(*host.Counters.RXFrames), []string{*host.Name})
			}
		}
	}
}

// FibreChannel Error Frames
type FibreChannelErrorFrames struct {
	*baseMetrics
}

func NewFibreChannelErrorFrames() *FibreChannelErrorFrames {
	return &FibreChannelErrorFrames{
		NewMetrics("node_fibrechannel_error_frames_total",
			"Number of errors in frames",
			[]string{"fc_host"}),
	}
}

func (f *FibreChannelErrorFrames) Collect(ch chan<- prometheus.Metric) {
	if err := initFibreChannelFS(); err != nil {
		return
	}

	hosts, err := fibrechannelFS.FibreChannelClass()
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		return
	}

	for _, host := range hosts {
		if host.Name != nil && host.Counters.ErrorFrames != nil {
			if isCounterImplemented(*host.Counters.ErrorFrames) {
				f.baseMetrics.collectCounter(ch, float64(*host.Counters.ErrorFrames), []string{*host.Name})
			}
		}
	}
}

// FibreChannel Invalid TX Words
type FibreChannelInvalidTxWords struct {
	*baseMetrics
}

func NewFibreChannelInvalidTxWords() *FibreChannelInvalidTxWords {
	return &FibreChannelInvalidTxWords{
		NewMetrics("node_fibrechannel_invalid_tx_words_total",
			"Number of invalid words transmitted by host port",
			[]string{"fc_host"}),
	}
}

func (f *FibreChannelInvalidTxWords) Collect(ch chan<- prometheus.Metric) {
	if err := initFibreChannelFS(); err != nil {
		return
	}

	hosts, err := fibrechannelFS.FibreChannelClass()
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		return
	}

	for _, host := range hosts {
		if host.Name != nil && host.Counters.InvalidTXWordCount != nil {
			if isCounterImplemented(*host.Counters.InvalidTXWordCount) {
				f.baseMetrics.collectCounter(ch, float64(*host.Counters.InvalidTXWordCount), []string{*host.Name})
			}
		}
	}
}

// FibreChannel Seconds Since Last Reset
type FibreChannelSecondsSinceLastReset struct {
	*baseMetrics
}

func NewFibreChannelSecondsSinceLastReset() *FibreChannelSecondsSinceLastReset {
	return &FibreChannelSecondsSinceLastReset{
		NewMetrics("node_fibrechannel_seconds_since_last_reset_total",
			"Number of seconds since last host port reset",
			[]string{"fc_host"}),
	}
}

func (f *FibreChannelSecondsSinceLastReset) Collect(ch chan<- prometheus.Metric) {
	if err := initFibreChannelFS(); err != nil {
		return
	}

	hosts, err := fibrechannelFS.FibreChannelClass()
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		return
	}

	for _, host := range hosts {
		if host.Name != nil && host.Counters.SecondsSinceLastReset != nil {
			if isCounterImplemented(*host.Counters.SecondsSinceLastReset) {
				f.baseMetrics.collectCounter(ch, float64(*host.Counters.SecondsSinceLastReset), []string{*host.Name})
			}
		}
	}
}

// FibreChannel TX Words
type FibreChannelTxWords struct {
	*baseMetrics
}

func NewFibreChannelTxWords() *FibreChannelTxWords {
	return &FibreChannelTxWords{
		NewMetrics("node_fibrechannel_tx_words_total",
			"Number of words transmitted by host port",
			[]string{"fc_host"}),
	}
}

func (f *FibreChannelTxWords) Collect(ch chan<- prometheus.Metric) {
	if err := initFibreChannelFS(); err != nil {
		return
	}

	hosts, err := fibrechannelFS.FibreChannelClass()
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		return
	}

	for _, host := range hosts {
		if host.Name != nil && host.Counters.TXWords != nil {
			if isCounterImplemented(*host.Counters.TXWords) {
				f.baseMetrics.collectCounter(ch, float64(*host.Counters.TXWords), []string{*host.Name})
			}
		}
	}
}

// FibreChannel Invalid CRC
type FibreChannelInvalidCRC struct {
	*baseMetrics
}

func NewFibreChannelInvalidCRC() *FibreChannelInvalidCRC {
	return &FibreChannelInvalidCRC{
		NewMetrics("node_fibrechannel_invalid_crc_total",
			"Invalid Cyclic Redundancy Check count",
			[]string{"fc_host"}),
	}
}

func (f *FibreChannelInvalidCRC) Collect(ch chan<- prometheus.Metric) {
	if err := initFibreChannelFS(); err != nil {
		return
	}

	hosts, err := fibrechannelFS.FibreChannelClass()
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		return
	}

	for _, host := range hosts {
		if host.Name != nil && host.Counters.InvalidCRCCount != nil {
			if isCounterImplemented(*host.Counters.InvalidCRCCount) {
				f.baseMetrics.collectCounter(ch, float64(*host.Counters.InvalidCRCCount), []string{*host.Name})
			}
		}
	}
}

// FibreChannel NOS
type FibreChannelNos struct {
	*baseMetrics
}

func NewFibreChannelNos() *FibreChannelNos {
	return &FibreChannelNos{
		NewMetrics("node_fibrechannel_nos_total",
			"Number Not_Operational Primitive Sequence received by host port",
			[]string{"fc_host"}),
	}
}

func (f *FibreChannelNos) Collect(ch chan<- prometheus.Metric) {
	if err := initFibreChannelFS(); err != nil {
		return
	}

	hosts, err := fibrechannelFS.FibreChannelClass()
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		return
	}

	for _, host := range hosts {
		if host.Name != nil && host.Counters.NosCount != nil {
			if isCounterImplemented(*host.Counters.NosCount) {
				f.baseMetrics.collectCounter(ch, float64(*host.Counters.NosCount), []string{*host.Name})
			}
		}
	}
}

// FibreChannel FCP Packet Aborts
type FibreChannelFcpPacketAborts struct {
	*baseMetrics
}

func NewFibreChannelFcpPacketAborts() *FibreChannelFcpPacketAborts {
	return &FibreChannelFcpPacketAborts{
		NewMetrics("node_fibrechannel_fcp_packet_aborts_total",
			"Number of aborted packets",
			[]string{"fc_host"}),
	}
}

func (f *FibreChannelFcpPacketAborts) Collect(ch chan<- prometheus.Metric) {
	if err := initFibreChannelFS(); err != nil {
		return
	}

	hosts, err := fibrechannelFS.FibreChannelClass()
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		return
	}

	for _, host := range hosts {
		if host.Name != nil && host.Counters.FCPPacketAborts != nil {
			if isCounterImplemented(*host.Counters.FCPPacketAborts) {
				f.baseMetrics.collectCounter(ch, float64(*host.Counters.FCPPacketAborts), []string{*host.Name})
			}
		}
	}
}

// FibreChannel RX Words
type FibreChannelRxWords struct {
	*baseMetrics
}

func NewFibreChannelRxWords() *FibreChannelRxWords {
	return &FibreChannelRxWords{
		NewMetrics("node_fibrechannel_rx_words_total",
			"Number of words received by host port",
			[]string{"fc_host"}),
	}
}

func (f *FibreChannelRxWords) Collect(ch chan<- prometheus.Metric) {
	if err := initFibreChannelFS(); err != nil {
		return
	}

	hosts, err := fibrechannelFS.FibreChannelClass()
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		return
	}

	for _, host := range hosts {
		if host.Name != nil && host.Counters.RXWords != nil {
			if isCounterImplemented(*host.Counters.RXWords) {
				f.baseMetrics.collectCounter(ch, float64(*host.Counters.RXWords), []string{*host.Name})
			}
		}
	}
}

// FibreChannel TX Frames
type FibreChannelTxFrames struct {
	*baseMetrics
}

func NewFibreChannelTxFrames() *FibreChannelTxFrames {
	return &FibreChannelTxFrames{
		NewMetrics("node_fibrechannel_tx_frames_total",
			"Number of frames transmitted by host port",
			[]string{"fc_host"}),
	}
}

func (f *FibreChannelTxFrames) Collect(ch chan<- prometheus.Metric) {
	if err := initFibreChannelFS(); err != nil {
		return
	}

	hosts, err := fibrechannelFS.FibreChannelClass()
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		return
	}

	for _, host := range hosts {
		if host.Name != nil && host.Counters.TXFrames != nil {
			if isCounterImplemented(*host.Counters.TXFrames) {
				f.baseMetrics.collectCounter(ch, float64(*host.Counters.TXFrames), []string{*host.Name})
			}
		}
	}
}

// FibreChannel Link Failure
type FibreChannelLinkFailure struct {
	*baseMetrics
}

func NewFibreChannelLinkFailure() *FibreChannelLinkFailure {
	return &FibreChannelLinkFailure{
		NewMetrics("node_fibrechannel_link_failure_total",
			"Number of times the host port link has failed",
			[]string{"fc_host"}),
	}
}

func (f *FibreChannelLinkFailure) Collect(ch chan<- prometheus.Metric) {
	if err := initFibreChannelFS(); err != nil {
		return
	}

	hosts, err := fibrechannelFS.FibreChannelClass()
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		return
	}

	for _, host := range hosts {
		if host.Name != nil && host.Counters.LinkFailureCount != nil {
			if isCounterImplemented(*host.Counters.LinkFailureCount) {
				f.baseMetrics.collectCounter(ch, float64(*host.Counters.LinkFailureCount), []string{*host.Name})
			}
		}
	}
} 
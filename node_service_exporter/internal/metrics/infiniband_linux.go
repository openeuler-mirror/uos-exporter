//go:build !noinfiniband && linux
// +build !noinfiniband,linux

package metrics

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"node_service_exporter/internal/exporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs/sysfs"
)

func init() {
	exporter.Register(NewInfiniBandInfo())
	exporter.Register(NewInfiniBandStateID())
	exporter.Register(NewInfiniBandPhysicalStateID())
	exporter.Register(NewInfiniBandRateBytes())
	exporter.Register(NewInfiniBandLegacyMulticastPacketsReceived())
	exporter.Register(NewInfiniBandLegacyMulticastPacketsTransmitted())
	exporter.Register(NewInfiniBandLegacyDataReceived())
	exporter.Register(NewInfiniBandLegacyPacketsReceived())
	exporter.Register(NewInfiniBandLegacyUnicastPacketsReceived())
	exporter.Register(NewInfiniBandLegacyUnicastPacketsTransmitted())
	exporter.Register(NewInfiniBandLegacyDataTransmitted())
	exporter.Register(NewInfiniBandLegacyPacketsTransmitted())
	exporter.Register(NewInfiniBandExcessiveBufferOverrunErrors())
	exporter.Register(NewInfiniBandLinkDowned())
	exporter.Register(NewInfiniBandLinkErrorRecovery())
	exporter.Register(NewInfiniBandLocalLinkIntegrityErrors())
	exporter.Register(NewInfiniBandMulticastPacketsReceived())
	exporter.Register(NewInfiniBandMulticastPacketsTransmitted())
	exporter.Register(NewInfiniBandPortConstraintErrorsReceived())
	exporter.Register(NewInfiniBandPortConstraintErrorsTransmitted())
	exporter.Register(NewInfiniBandPortDataReceived())
	exporter.Register(NewInfiniBandPortDataTransmitted())
	exporter.Register(NewInfiniBandPortDiscardsReceived())
	exporter.Register(NewInfiniBandPortDiscardsTransmitted())
	exporter.Register(NewInfiniBandPortErrorsReceived())
	exporter.Register(NewInfiniBandPortPacketsReceived())
	exporter.Register(NewInfiniBandPortPacketsTransmitted())
	exporter.Register(NewInfiniBandPortTransmitWait())
	exporter.Register(NewInfiniBandUnicastPacketsReceived())
	exporter.Register(NewInfiniBandUnicastPacketsTransmitted())
	exporter.Register(NewInfiniBandPortReceiveRemotePhysicalErrors())
	exporter.Register(NewInfiniBandPortReceiveSwitchRelayErrors())
	exporter.Register(NewInfiniBandSymbolError())
	exporter.Register(NewInfiniBandVL15Dropped())
}

var infinibandFS sysfs.FS
var infinibandInitialized bool

func initInfiniBandFS() error {
	if !infinibandInitialized {
		fs, err := sysfs.NewFS("/sys")
		if err != nil {
			return fmt.Errorf("failed to open sysfs: %w", err)
		}
		infinibandFS = fs
		infinibandInitialized = true
	}
	return nil
}

// Helper function to safely get counter value
func getCounterValue(counter *uint64) float64 {
	if counter != nil {
		return float64(*counter)
	}
	return 0
}

// InfiniBand Info
type InfiniBandInfo struct {
	*baseMetrics
}

func NewInfiniBandInfo() *InfiniBandInfo {
	return &InfiniBandInfo{
		NewMetrics("node_infiniband_info",
			"Non-numeric data from /sys/class/infiniband/<device>, value is always 1.",
			[]string{"device", "board_id", "firmware_version", "hca_type"}),
	}
}

func (i *InfiniBandInfo) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil {
		return
	}

	devices, err := infinibandFS.InfiniBandClass()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	for _, device := range devices {
		i.baseMetrics.collect(ch, 1.0, []string{device.Name, device.BoardID, device.FirmwareVersion, device.HCAType})
	}
}

// InfiniBand State ID
type InfiniBandStateID struct {
	*baseMetrics
}

func NewInfiniBandStateID() *InfiniBandStateID {
	return &InfiniBandStateID{
		NewMetrics("node_infiniband_state_id",
			"State of the InfiniBand port (0: no change, 1: down, 2: init, 3: armed, 4: active, 5: act defer)",
			[]string{"device", "port"}),
	}
}

func (i *InfiniBandStateID) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil {
		return
	}

	devices, err := infinibandFS.InfiniBandClass()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	for _, device := range devices {
		for _, port := range device.Ports {
			portStr := strconv.FormatUint(uint64(port.Port), 10)
			i.baseMetrics.collect(ch, float64(port.StateID), []string{port.Name, portStr})
		}
	}
}

// InfiniBand Physical State ID
type InfiniBandPhysicalStateID struct {
	*baseMetrics
}

func NewInfiniBandPhysicalStateID() *InfiniBandPhysicalStateID {
	return &InfiniBandPhysicalStateID{
		NewMetrics("node_infiniband_physical_state_id",
			"Physical state of the InfiniBand port (0: no change, 1: sleep, 2: polling, 3: disable, 4: shift, 5: link up, 6: link error recover, 7: phytest)",
			[]string{"device", "port"}),
	}
}

func (i *InfiniBandPhysicalStateID) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil {
		return
	}

	devices, err := infinibandFS.InfiniBandClass()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	for _, device := range devices {
		for _, port := range device.Ports {
			portStr := strconv.FormatUint(uint64(port.Port), 10)
			i.baseMetrics.collect(ch, float64(port.PhysStateID), []string{port.Name, portStr})
		}
	}
}

// InfiniBand Rate Bytes Per Second
type InfiniBandRateBytes struct {
	*baseMetrics
}

func NewInfiniBandRateBytes() *InfiniBandRateBytes {
	return &InfiniBandRateBytes{
		NewMetrics("node_infiniband_rate_bytes_per_second",
			"Maximum signal transfer rate",
			[]string{"device", "port"}),
	}
}

func (i *InfiniBandRateBytes) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil {
		return
	}

	devices, err := infinibandFS.InfiniBandClass()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	for _, device := range devices {
		for _, port := range device.Ports {
			portStr := strconv.FormatUint(uint64(port.Port), 10)
			i.baseMetrics.collect(ch, float64(port.Rate), []string{port.Name, portStr})
		}
	}
}

// InfiniBand Legacy Multicast Packets Received
type InfiniBandLegacyMulticastPacketsReceived struct {
	*baseMetrics
}

func NewInfiniBandLegacyMulticastPacketsReceived() *InfiniBandLegacyMulticastPacketsReceived {
	return &InfiniBandLegacyMulticastPacketsReceived{
		NewMetrics("node_infiniband_legacy_multicast_packets_received_total",
			"Number of multicast packets received",
			[]string{"device", "port"}),
	}
}

func (i *InfiniBandLegacyMulticastPacketsReceived) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil {
		return
	}

	devices, err := infinibandFS.InfiniBandClass()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	for _, device := range devices {
		for _, port := range device.Ports {
			portStr := strconv.FormatUint(uint64(port.Port), 10)
			if port.Counters.LegacyPortMulticastRcvPackets != nil {
				i.baseMetrics.collectCounter(ch, getCounterValue(port.Counters.LegacyPortMulticastRcvPackets), []string{port.Name, portStr})
			}
		}
	}
}

// InfiniBand Legacy Multicast Packets Transmitted
type InfiniBandLegacyMulticastPacketsTransmitted struct {
	*baseMetrics
}

func NewInfiniBandLegacyMulticastPacketsTransmitted() *InfiniBandLegacyMulticastPacketsTransmitted {
	return &InfiniBandLegacyMulticastPacketsTransmitted{
		NewMetrics("node_infiniband_legacy_multicast_packets_transmitted_total",
			"Number of multicast packets transmitted",
			[]string{"device", "port"}),
	}
}

func (i *InfiniBandLegacyMulticastPacketsTransmitted) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil {
		return
	}

	devices, err := infinibandFS.InfiniBandClass()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	for _, device := range devices {
		for _, port := range device.Ports {
			portStr := strconv.FormatUint(uint64(port.Port), 10)
			if port.Counters.LegacyPortMulticastXmitPackets != nil {
				i.baseMetrics.collectCounter(ch, getCounterValue(port.Counters.LegacyPortMulticastXmitPackets), []string{port.Name, portStr})
			}
		}
	}
}

// InfiniBand Legacy Data Received
type InfiniBandLegacyDataReceived struct {
	*baseMetrics
}

func NewInfiniBandLegacyDataReceived() *InfiniBandLegacyDataReceived {
	return &InfiniBandLegacyDataReceived{
		NewMetrics("node_infiniband_legacy_data_received_bytes_total",
			"Number of data octets received on all links",
			[]string{"device", "port"}),
	}
}

func (i *InfiniBandLegacyDataReceived) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil {
		return
	}

	devices, err := infinibandFS.InfiniBandClass()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	for _, device := range devices {
		for _, port := range device.Ports {
			portStr := strconv.FormatUint(uint64(port.Port), 10)
			if port.Counters.LegacyPortRcvData64 != nil {
				i.baseMetrics.collectCounter(ch, getCounterValue(port.Counters.LegacyPortRcvData64), []string{port.Name, portStr})
			}
		}
	}
}

// InfiniBand Legacy Packets Received
type InfiniBandLegacyPacketsReceived struct {
	*baseMetrics
}

func NewInfiniBandLegacyPacketsReceived() *InfiniBandLegacyPacketsReceived {
	return &InfiniBandLegacyPacketsReceived{
		NewMetrics("node_infiniband_legacy_packets_received_total",
			"Number of data packets received on all links",
			[]string{"device", "port"}),
	}
}

func (i *InfiniBandLegacyPacketsReceived) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil {
		return
	}

	devices, err := infinibandFS.InfiniBandClass()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	for _, device := range devices {
		for _, port := range device.Ports {
			portStr := strconv.FormatUint(uint64(port.Port), 10)
			if port.Counters.LegacyPortRcvPackets64 != nil {
				i.baseMetrics.collectCounter(ch, getCounterValue(port.Counters.LegacyPortRcvPackets64), []string{port.Name, portStr})
			}
		}
	}
}

// InfiniBand Legacy Unicast Packets Received
type InfiniBandLegacyUnicastPacketsReceived struct {
	*baseMetrics
}

func NewInfiniBandLegacyUnicastPacketsReceived() *InfiniBandLegacyUnicastPacketsReceived {
	return &InfiniBandLegacyUnicastPacketsReceived{
		NewMetrics("node_infiniband_legacy_unicast_packets_received_total",
			"Number of unicast packets received",
			[]string{"device", "port"}),
	}
}

func (i *InfiniBandLegacyUnicastPacketsReceived) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil {
		return
	}

	devices, err := infinibandFS.InfiniBandClass()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	for _, device := range devices {
		for _, port := range device.Ports {
			portStr := strconv.FormatUint(uint64(port.Port), 10)
			if port.Counters.LegacyPortUnicastRcvPackets != nil {
				i.baseMetrics.collectCounter(ch, getCounterValue(port.Counters.LegacyPortUnicastRcvPackets), []string{port.Name, portStr})
			}
		}
	}
}

// InfiniBand Legacy Unicast Packets Transmitted
type InfiniBandLegacyUnicastPacketsTransmitted struct {
	*baseMetrics
}

func NewInfiniBandLegacyUnicastPacketsTransmitted() *InfiniBandLegacyUnicastPacketsTransmitted {
	return &InfiniBandLegacyUnicastPacketsTransmitted{
		NewMetrics("node_infiniband_legacy_unicast_packets_transmitted_total",
			"Number of unicast packets transmitted",
			[]string{"device", "port"}),
	}
}

func (i *InfiniBandLegacyUnicastPacketsTransmitted) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil {
		return
	}

	devices, err := infinibandFS.InfiniBandClass()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	for _, device := range devices {
		for _, port := range device.Ports {
			portStr := strconv.FormatUint(uint64(port.Port), 10)
			if port.Counters.LegacyPortUnicastXmitPackets != nil {
				i.baseMetrics.collectCounter(ch, getCounterValue(port.Counters.LegacyPortUnicastXmitPackets), []string{port.Name, portStr})
			}
		}
	}
}

// InfiniBand Legacy Data Transmitted
type InfiniBandLegacyDataTransmitted struct {
	*baseMetrics
}

func NewInfiniBandLegacyDataTransmitted() *InfiniBandLegacyDataTransmitted {
	return &InfiniBandLegacyDataTransmitted{
		NewMetrics("node_infiniband_legacy_data_transmitted_bytes_total",
			"Number of data octets transmitted on all links",
			[]string{"device", "port"}),
	}
}

func (i *InfiniBandLegacyDataTransmitted) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil {
		return
	}

	devices, err := infinibandFS.InfiniBandClass()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	for _, device := range devices {
		for _, port := range device.Ports {
			portStr := strconv.FormatUint(uint64(port.Port), 10)
			if port.Counters.LegacyPortXmitData64 != nil {
				i.baseMetrics.collectCounter(ch, getCounterValue(port.Counters.LegacyPortXmitData64), []string{port.Name, portStr})
			}
		}
	}
}

// InfiniBand Legacy Packets Transmitted
type InfiniBandLegacyPacketsTransmitted struct {
	*baseMetrics
}

func NewInfiniBandLegacyPacketsTransmitted() *InfiniBandLegacyPacketsTransmitted {
	return &InfiniBandLegacyPacketsTransmitted{
		NewMetrics("node_infiniband_legacy_packets_transmitted_total",
			"Number of data packets received on all links",
			[]string{"device", "port"}),
	}
}

func (i *InfiniBandLegacyPacketsTransmitted) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil {
		return
	}

	devices, err := infinibandFS.InfiniBandClass()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	for _, device := range devices {
		for _, port := range device.Ports {
			portStr := strconv.FormatUint(uint64(port.Port), 10)
			if port.Counters.LegacyPortXmitPackets64 != nil {
				i.baseMetrics.collectCounter(ch, getCounterValue(port.Counters.LegacyPortXmitPackets64), []string{port.Name, portStr})
			}
		}
	}
}

// Error and reliability metrics
type InfiniBandExcessiveBufferOverrunErrors struct {
	*baseMetrics
}

func NewInfiniBandExcessiveBufferOverrunErrors() *InfiniBandExcessiveBufferOverrunErrors {
	return &InfiniBandExcessiveBufferOverrunErrors{
		NewMetrics("node_infiniband_excessive_buffer_overrun_errors_total",
			"Number of times that OverrunErrors consecutive flow control update periods occurred, each having at least one overrun error.",
			[]string{"device", "port"}),
	}
}

func (i *InfiniBandExcessiveBufferOverrunErrors) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil {
		return
	}

	devices, err := infinibandFS.InfiniBandClass()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	for _, device := range devices {
		for _, port := range device.Ports {
			portStr := strconv.FormatUint(uint64(port.Port), 10)
			if port.Counters.ExcessiveBufferOverrunErrors != nil {
				i.baseMetrics.collectCounter(ch, getCounterValue(port.Counters.ExcessiveBufferOverrunErrors), []string{port.Name, portStr})
			}
		}
	}
}

// Link status metrics
type InfiniBandLinkDowned struct {
	*baseMetrics
}

func NewInfiniBandLinkDowned() *InfiniBandLinkDowned {
	return &InfiniBandLinkDowned{
		NewMetrics("node_infiniband_link_downed_total",
			"Number of times the link failed to recover from an error state and went down",
			[]string{"device", "port"}),
	}
}

func (i *InfiniBandLinkDowned) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil {
		return
	}

	devices, err := infinibandFS.InfiniBandClass()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	for _, device := range devices {
		for _, port := range device.Ports {
			portStr := strconv.FormatUint(uint64(port.Port), 10)
			if port.Counters.LinkDowned != nil {
				i.baseMetrics.collectCounter(ch, getCounterValue(port.Counters.LinkDowned), []string{port.Name, portStr})
			}
		}
	}
}

type InfiniBandLinkErrorRecovery struct {
	*baseMetrics
}

func NewInfiniBandLinkErrorRecovery() *InfiniBandLinkErrorRecovery {
	return &InfiniBandLinkErrorRecovery{
		NewMetrics("node_infiniband_link_error_recovery_total",
			"Number of times the link successfully recovered from an error state",
			[]string{"device", "port"}),
	}
}

func (i *InfiniBandLinkErrorRecovery) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil {
		return
	}

	devices, err := infinibandFS.InfiniBandClass()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	for _, device := range devices {
		for _, port := range device.Ports {
			portStr := strconv.FormatUint(uint64(port.Port), 10)
			if port.Counters.LinkErrorRecovery != nil {
				i.baseMetrics.collectCounter(ch, getCounterValue(port.Counters.LinkErrorRecovery), []string{port.Name, portStr})
			}
		}
	}
}

// Due to file length limits, I'll provide the structure for remaining collectors
// Each following the same pattern: struct definition, constructor, and Collect method

// Implement remaining collectors with proper structure pattern...

// InfiniBand Local Link Integrity Errors
type InfiniBandLocalLinkIntegrityErrors struct {
	*baseMetrics
}

func NewInfiniBandLocalLinkIntegrityErrors() *InfiniBandLocalLinkIntegrityErrors {
	return &InfiniBandLocalLinkIntegrityErrors{
		NewMetrics("node_infiniband_local_link_integrity_errors_total",
			"Number of times that the count of local physical errors exceeded the threshold specified by LocalPhyErrors.",
			[]string{"device", "port"}),
	}
}

func (i *InfiniBandLocalLinkIntegrityErrors) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil {
		return
	}

	devices, err := infinibandFS.InfiniBandClass()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	for _, device := range devices {
		for _, port := range device.Ports {
			portStr := strconv.FormatUint(uint64(port.Port), 10)
			if port.Counters.LocalLinkIntegrityErrors != nil {
				i.baseMetrics.collectCounter(ch, getCounterValue(port.Counters.LocalLinkIntegrityErrors), []string{port.Name, portStr})
			}
		}
	}
}

// InfiniBand Multicast Packets Received
type InfiniBandMulticastPacketsReceived struct {
	*baseMetrics
}

func NewInfiniBandMulticastPacketsReceived() *InfiniBandMulticastPacketsReceived {
	return &InfiniBandMulticastPacketsReceived{
		NewMetrics("node_infiniband_multicast_packets_received_total",
			"Number of multicast packets received (including errors)",
			[]string{"device", "port"}),
	}
}

func (i *InfiniBandMulticastPacketsReceived) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil {
		return
	}

	devices, err := infinibandFS.InfiniBandClass()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	for _, device := range devices {
		for _, port := range device.Ports {
			portStr := strconv.FormatUint(uint64(port.Port), 10)
			if port.Counters.MulticastRcvPackets != nil {
				i.baseMetrics.collectCounter(ch, getCounterValue(port.Counters.MulticastRcvPackets), []string{port.Name, portStr})
			}
		}
	}
}

// InfiniBand Multicast Packets Transmitted
type InfiniBandMulticastPacketsTransmitted struct {
	*baseMetrics
}

func NewInfiniBandMulticastPacketsTransmitted() *InfiniBandMulticastPacketsTransmitted {
	return &InfiniBandMulticastPacketsTransmitted{
		NewMetrics("node_infiniband_multicast_packets_transmitted_total",
			"Number of multicast packets transmitted (including errors)",
			[]string{"device", "port"}),
	}
}

func (i *InfiniBandMulticastPacketsTransmitted) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil {
		return
	}

	devices, err := infinibandFS.InfiniBandClass()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	for _, device := range devices {
		for _, port := range device.Ports {
			portStr := strconv.FormatUint(uint64(port.Port), 10)
			if port.Counters.MulticastXmitPackets != nil {
				i.baseMetrics.collectCounter(ch, getCounterValue(port.Counters.MulticastXmitPackets), []string{port.Name, portStr})
			}
		}
	}
}

// InfiniBand Port Constraint Errors Received
type InfiniBandPortConstraintErrorsReceived struct {
	*baseMetrics
}

func NewInfiniBandPortConstraintErrorsReceived() *InfiniBandPortConstraintErrorsReceived {
	return &InfiniBandPortConstraintErrorsReceived{
		NewMetrics("node_infiniband_port_constraint_errors_received_total",
			"Number of packets received on the switch physical port that are discarded",
			[]string{"device", "port"}),
	}
}

func (i *InfiniBandPortConstraintErrorsReceived) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil {
		return
	}

	devices, err := infinibandFS.InfiniBandClass()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	for _, device := range devices {
		for _, port := range device.Ports {
			portStr := strconv.FormatUint(uint64(port.Port), 10)
			if port.Counters.PortRcvConstraintErrors != nil {
				i.baseMetrics.collectCounter(ch, getCounterValue(port.Counters.PortRcvConstraintErrors), []string{port.Name, portStr})
			}
		}
	}
}

// InfiniBand Port Constraint Errors Transmitted
type InfiniBandPortConstraintErrorsTransmitted struct {
	*baseMetrics
}

func NewInfiniBandPortConstraintErrorsTransmitted() *InfiniBandPortConstraintErrorsTransmitted {
	return &InfiniBandPortConstraintErrorsTransmitted{
		NewMetrics("node_infiniband_port_constraint_errors_transmitted_total",
			"Number of packets not transmitted from the switch physical port",
			[]string{"device", "port"}),
	}
}

func (i *InfiniBandPortConstraintErrorsTransmitted) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil {
		return
	}

	devices, err := infinibandFS.InfiniBandClass()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	for _, device := range devices {
		for _, port := range device.Ports {
			portStr := strconv.FormatUint(uint64(port.Port), 10)
			if port.Counters.PortXmitConstraintErrors != nil {
				i.baseMetrics.collectCounter(ch, getCounterValue(port.Counters.PortXmitConstraintErrors), []string{port.Name, portStr})
			}
		}
	}
}

// InfiniBand Port Data Received
type InfiniBandPortDataReceived struct {
	*baseMetrics
}

func NewInfiniBandPortDataReceived() *InfiniBandPortDataReceived {
	return &InfiniBandPortDataReceived{
		NewMetrics("node_infiniband_port_data_received_bytes_total",
			"Number of data octets received on all links",
			[]string{"device", "port"}),
	}
}

func (i *InfiniBandPortDataReceived) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil {
		return
	}

	devices, err := infinibandFS.InfiniBandClass()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	for _, device := range devices {
		for _, port := range device.Ports {
			portStr := strconv.FormatUint(uint64(port.Port), 10)
			if port.Counters.PortRcvData != nil {
				i.baseMetrics.collectCounter(ch, getCounterValue(port.Counters.PortRcvData), []string{port.Name, portStr})
			}
		}
	}
}

// InfiniBand Port Data Transmitted
type InfiniBandPortDataTransmitted struct {
	*baseMetrics
}

func NewInfiniBandPortDataTransmitted() *InfiniBandPortDataTransmitted {
	return &InfiniBandPortDataTransmitted{
		NewMetrics("node_infiniband_port_data_transmitted_bytes_total",
			"Number of data octets transmitted on all links",
			[]string{"device", "port"}),
	}
}

func (i *InfiniBandPortDataTransmitted) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil {
		return
	}

	devices, err := infinibandFS.InfiniBandClass()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	for _, device := range devices {
		for _, port := range device.Ports {
			portStr := strconv.FormatUint(uint64(port.Port), 10)
			if port.Counters.PortXmitData != nil {
				i.baseMetrics.collectCounter(ch, getCounterValue(port.Counters.PortXmitData), []string{port.Name, portStr})
			}
		}
	}
}

// For the remaining collectors, I'll implement them with the same pattern but in a more compact form
// Each one follows: struct definition, constructor, and Collect method

// Port Discards and Errors
type InfiniBandPortDiscardsReceived struct{ *baseMetrics }
func NewInfiniBandPortDiscardsReceived() *InfiniBandPortDiscardsReceived {
	return &InfiniBandPortDiscardsReceived{NewMetrics("node_infiniband_port_discards_received_total", "Number of inbound packets discarded by the port because the port is down or congested", []string{"device", "port"})}
}
func (i *InfiniBandPortDiscardsReceived) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil { return }
	devices, err := infinibandFS.InfiniBandClass()
	if err != nil { if errors.Is(err, os.ErrNotExist) { return }; return }
	for _, device := range devices {
		for _, port := range device.Ports {
			portStr := strconv.FormatUint(uint64(port.Port), 10)
			if port.Counters.PortRcvDiscards != nil {
				i.baseMetrics.collectCounter(ch, getCounterValue(port.Counters.PortRcvDiscards), []string{port.Name, portStr})
			}
		}
	}
}

type InfiniBandPortDiscardsTransmitted struct{ *baseMetrics }
func NewInfiniBandPortDiscardsTransmitted() *InfiniBandPortDiscardsTransmitted {
	return &InfiniBandPortDiscardsTransmitted{NewMetrics("node_infiniband_port_discards_transmitted_total", "Number of outbound packets discarded by the port because the port is down or congested", []string{"device", "port"})}
}
func (i *InfiniBandPortDiscardsTransmitted) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil { return }
	devices, err := infinibandFS.InfiniBandClass()
	if err != nil { if errors.Is(err, os.ErrNotExist) { return }; return }
	for _, device := range devices {
		for _, port := range device.Ports {
			portStr := strconv.FormatUint(uint64(port.Port), 10)
			if port.Counters.PortXmitDiscards != nil {
				i.baseMetrics.collectCounter(ch, getCounterValue(port.Counters.PortXmitDiscards), []string{port.Name, portStr})
			}
		}
	}
}

type InfiniBandPortErrorsReceived struct{ *baseMetrics }
func NewInfiniBandPortErrorsReceived() *InfiniBandPortErrorsReceived {
	return &InfiniBandPortErrorsReceived{NewMetrics("node_infiniband_port_errors_received_total", "Number of packets containing an error that were received on this port", []string{"device", "port"})}
}
func (i *InfiniBandPortErrorsReceived) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil { return }
	devices, err := infinibandFS.InfiniBandClass()
	if err != nil { if errors.Is(err, os.ErrNotExist) { return }; return }
	for _, device := range devices {
		for _, port := range device.Ports {
			portStr := strconv.FormatUint(uint64(port.Port), 10)
			if port.Counters.PortRcvErrors != nil {
				i.baseMetrics.collectCounter(ch, getCounterValue(port.Counters.PortRcvErrors), []string{port.Name, portStr})
			}
		}
	}
}

type InfiniBandPortPacketsReceived struct{ *baseMetrics }
func NewInfiniBandPortPacketsReceived() *InfiniBandPortPacketsReceived {
	return &InfiniBandPortPacketsReceived{NewMetrics("node_infiniband_port_packets_received_total", "Number of packets received on all VLs by this port (including errors)", []string{"device", "port"})}
}
func (i *InfiniBandPortPacketsReceived) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil { return }
	devices, err := infinibandFS.InfiniBandClass()
	if err != nil { if errors.Is(err, os.ErrNotExist) { return }; return }
	for _, device := range devices {
		for _, port := range device.Ports {
			portStr := strconv.FormatUint(uint64(port.Port), 10)
			if port.Counters.PortRcvPackets != nil {
				i.baseMetrics.collectCounter(ch, getCounterValue(port.Counters.PortRcvPackets), []string{port.Name, portStr})
			}
		}
	}
}

type InfiniBandPortPacketsTransmitted struct{ *baseMetrics }
func NewInfiniBandPortPacketsTransmitted() *InfiniBandPortPacketsTransmitted {
	return &InfiniBandPortPacketsTransmitted{NewMetrics("node_infiniband_port_packets_transmitted_total", "Number of packets transmitted on all VLs from this port (including errors)", []string{"device", "port"})}
}
func (i *InfiniBandPortPacketsTransmitted) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil { return }
	devices, err := infinibandFS.InfiniBandClass()
	if err != nil { if errors.Is(err, os.ErrNotExist) { return }; return }
	for _, device := range devices {
		for _, port := range device.Ports {
			portStr := strconv.FormatUint(uint64(port.Port), 10)
			if port.Counters.PortXmitPackets != nil {
				i.baseMetrics.collectCounter(ch, getCounterValue(port.Counters.PortXmitPackets), []string{port.Name, portStr})
			}
		}
	}
}

type InfiniBandPortTransmitWait struct{ *baseMetrics }
func NewInfiniBandPortTransmitWait() *InfiniBandPortTransmitWait {
	return &InfiniBandPortTransmitWait{NewMetrics("node_infiniband_port_transmit_wait_total", "Number of ticks during which the port had data to transmit but no data was sent during the entire tick", []string{"device", "port"})}
}
func (i *InfiniBandPortTransmitWait) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil { return }
	devices, err := infinibandFS.InfiniBandClass()
	if err != nil { if errors.Is(err, os.ErrNotExist) { return }; return }
	for _, device := range devices {
		for _, port := range device.Ports {
			portStr := strconv.FormatUint(uint64(port.Port), 10)
			if port.Counters.PortXmitWait != nil {
				i.baseMetrics.collectCounter(ch, getCounterValue(port.Counters.PortXmitWait), []string{port.Name, portStr})
			}
		}
	}
}

// Unicast Packets
type InfiniBandUnicastPacketsReceived struct{ *baseMetrics }
func NewInfiniBandUnicastPacketsReceived() *InfiniBandUnicastPacketsReceived {
	return &InfiniBandUnicastPacketsReceived{NewMetrics("node_infiniband_unicast_packets_received_total", "Number of unicast packets received (including errors)", []string{"device", "port"})}
}
func (i *InfiniBandUnicastPacketsReceived) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil { return }
	devices, err := infinibandFS.InfiniBandClass()
	if err != nil { if errors.Is(err, os.ErrNotExist) { return }; return }
	for _, device := range devices {
		for _, port := range device.Ports {
			portStr := strconv.FormatUint(uint64(port.Port), 10)
			if port.Counters.UnicastRcvPackets != nil {
				i.baseMetrics.collectCounter(ch, getCounterValue(port.Counters.UnicastRcvPackets), []string{port.Name, portStr})
			}
		}
	}
}

type InfiniBandUnicastPacketsTransmitted struct{ *baseMetrics }
func NewInfiniBandUnicastPacketsTransmitted() *InfiniBandUnicastPacketsTransmitted {
	return &InfiniBandUnicastPacketsTransmitted{NewMetrics("node_infiniband_unicast_packets_transmitted_total", "Number of unicast packets transmitted (including errors)", []string{"device", "port"})}
}
func (i *InfiniBandUnicastPacketsTransmitted) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil { return }
	devices, err := infinibandFS.InfiniBandClass()
	if err != nil { if errors.Is(err, os.ErrNotExist) { return }; return }
	for _, device := range devices {
		for _, port := range device.Ports {
			portStr := strconv.FormatUint(uint64(port.Port), 10)
			if port.Counters.UnicastXmitPackets != nil {
				i.baseMetrics.collectCounter(ch, getCounterValue(port.Counters.UnicastXmitPackets), []string{port.Name, portStr})
			}
		}
	}
}

// Remaining error counters
type InfiniBandPortReceiveRemotePhysicalErrors struct{ *baseMetrics }
func NewInfiniBandPortReceiveRemotePhysicalErrors() *InfiniBandPortReceiveRemotePhysicalErrors {
	return &InfiniBandPortReceiveRemotePhysicalErrors{NewMetrics("node_infiniband_port_receive_remote_physical_errors_total", "Number of packets marked with the EBP (End of Bad Packet) delimiter received on the port.", []string{"device", "port"})}
}
func (i *InfiniBandPortReceiveRemotePhysicalErrors) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil { return }
	devices, err := infinibandFS.InfiniBandClass()
	if err != nil { if errors.Is(err, os.ErrNotExist) { return }; return }
	for _, device := range devices {
		for _, port := range device.Ports {
			portStr := strconv.FormatUint(uint64(port.Port), 10)
			if port.Counters.PortRcvRemotePhysicalErrors != nil {
				i.baseMetrics.collectCounter(ch, getCounterValue(port.Counters.PortRcvRemotePhysicalErrors), []string{port.Name, portStr})
			}
		}
	}
}

type InfiniBandPortReceiveSwitchRelayErrors struct{ *baseMetrics }
func NewInfiniBandPortReceiveSwitchRelayErrors() *InfiniBandPortReceiveSwitchRelayErrors {
	return &InfiniBandPortReceiveSwitchRelayErrors{NewMetrics("node_infiniband_port_receive_switch_relay_errors_total", "Number of packets that could not be forwarded by the switch.", []string{"device", "port"})}
}
func (i *InfiniBandPortReceiveSwitchRelayErrors) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil { return }
	devices, err := infinibandFS.InfiniBandClass()
	if err != nil { if errors.Is(err, os.ErrNotExist) { return }; return }
	for _, device := range devices {
		for _, port := range device.Ports {
			portStr := strconv.FormatUint(uint64(port.Port), 10)
			if port.Counters.PortRcvSwitchRelayErrors != nil {
				i.baseMetrics.collectCounter(ch, getCounterValue(port.Counters.PortRcvSwitchRelayErrors), []string{port.Name, portStr})
			}
		}
	}
}

type InfiniBandSymbolError struct{ *baseMetrics }
func NewInfiniBandSymbolError() *InfiniBandSymbolError {
	return &InfiniBandSymbolError{NewMetrics("node_infiniband_symbol_error_total", "Number of minor link errors detected on one or more physical lanes.", []string{"device", "port"})}
}
func (i *InfiniBandSymbolError) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil { return }
	devices, err := infinibandFS.InfiniBandClass()
	if err != nil { if errors.Is(err, os.ErrNotExist) { return }; return }
	for _, device := range devices {
		for _, port := range device.Ports {
			portStr := strconv.FormatUint(uint64(port.Port), 10)
			if port.Counters.SymbolError != nil {
				i.baseMetrics.collectCounter(ch, getCounterValue(port.Counters.SymbolError), []string{port.Name, portStr})
			}
		}
	}
}

type InfiniBandVL15Dropped struct{ *baseMetrics }
func NewInfiniBandVL15Dropped() *InfiniBandVL15Dropped {
	return &InfiniBandVL15Dropped{NewMetrics("node_infiniband_vl15_dropped_total", "Number of incoming VL15 packets dropped due to resource limitations.", []string{"device", "port"})}
}
func (i *InfiniBandVL15Dropped) Collect(ch chan<- prometheus.Metric) {
	if err := initInfiniBandFS(); err != nil { return }
	devices, err := infinibandFS.InfiniBandClass()
	if err != nil { if errors.Is(err, os.ErrNotExist) { return }; return }
	for _, device := range devices {
		for _, port := range device.Ports {
			portStr := strconv.FormatUint(uint64(port.Port), 10)
			if port.Counters.VL15Dropped != nil {
				i.baseMetrics.collectCounter(ch, getCounterValue(port.Counters.VL15Dropped), []string{port.Name, portStr})
			}
		}
	}
} 
//go:build !nonfs && linux
// +build !nonfs,linux

package metrics

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"node_service_exporter/internal/exporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs/nfs"
)

func init() {
	exporter.Register(NewNFSNetworkPackets())
	exporter.Register(NewNFSNetworkConnections())
	exporter.Register(NewNFSRPCOperations())
	exporter.Register(NewNFSRPCRetransmissions())
	exporter.Register(NewNFSRPCAuthRefreshes())
	exporter.Register(NewNFSProcedures())
}

var nfsFS nfs.FS
var nfsInitialized bool

func initNFSFS() error {
	if !nfsInitialized {
		fs, err := nfs.NewFS("/proc")
		if err != nil {
			return fmt.Errorf("failed to open procfs: %w", err)
		}
		nfsFS = fs
		nfsInitialized = true
	}
	return nil
}

// NFS Network Packets
type NFSNetworkPackets struct {
	*baseMetrics
}

func NewNFSNetworkPackets() *NFSNetworkPackets {
	return &NFSNetworkPackets{
		NewMetrics("node_nfs_packets_total",
			"Total NFSd network packets (sent+received) by protocol type.",
			[]string{"protocol"}),
	}
}

func (n *NFSNetworkPackets) Collect(ch chan<- prometheus.Metric) {
	if err := initNFSFS(); err != nil {
		return
	}

	stats, err := nfsFS.ClientRPCStats()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	n.baseMetrics.collectCounter(ch, float64(stats.Network.UDPCount), []string{"udp"})
	n.baseMetrics.collectCounter(ch, float64(stats.Network.TCPCount), []string{"tcp"})
}

// NFS Network Connections
type NFSNetworkConnections struct {
	*baseMetrics
}

func NewNFSNetworkConnections() *NFSNetworkConnections {
	return &NFSNetworkConnections{
		NewMetrics("node_nfs_connections_total",
			"Total number of NFSd TCP connections.",
			[]string{}),
	}
}

func (n *NFSNetworkConnections) Collect(ch chan<- prometheus.Metric) {
	if err := initNFSFS(); err != nil {
		return
	}

	stats, err := nfsFS.ClientRPCStats()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	n.baseMetrics.collectCounter(ch, float64(stats.Network.TCPConnect), []string{})
}

// NFS RPC Operations
type NFSRPCOperations struct {
	*baseMetrics
}

func NewNFSRPCOperations() *NFSRPCOperations {
	return &NFSRPCOperations{
		NewMetrics("node_nfs_rpcs_total",
			"Total number of RPCs performed.",
			[]string{}),
	}
}

func (n *NFSRPCOperations) Collect(ch chan<- prometheus.Metric) {
	if err := initNFSFS(); err != nil {
		return
	}

	stats, err := nfsFS.ClientRPCStats()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	n.baseMetrics.collectCounter(ch, float64(stats.ClientRPC.RPCCount), []string{})
}

// NFS RPC Retransmissions
type NFSRPCRetransmissions struct {
	*baseMetrics
}

func NewNFSRPCRetransmissions() *NFSRPCRetransmissions {
	return &NFSRPCRetransmissions{
		NewMetrics("node_nfs_rpc_retransmissions_total",
			"Number of RPC transmissions performed.",
			[]string{}),
	}
}

func (n *NFSRPCRetransmissions) Collect(ch chan<- prometheus.Metric) {
	if err := initNFSFS(); err != nil {
		return
	}

	stats, err := nfsFS.ClientRPCStats()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	n.baseMetrics.collectCounter(ch, float64(stats.ClientRPC.Retransmissions), []string{})
}

// NFS RPC Authentication Refreshes
type NFSRPCAuthRefreshes struct {
	*baseMetrics
}

func NewNFSRPCAuthRefreshes() *NFSRPCAuthRefreshes {
	return &NFSRPCAuthRefreshes{
		NewMetrics("node_nfs_rpc_authentication_refreshes_total",
			"Number of RPC authentication refreshes performed.",
			[]string{}),
	}
}

func (n *NFSRPCAuthRefreshes) Collect(ch chan<- prometheus.Metric) {
	if err := initNFSFS(); err != nil {
		return
	}

	stats, err := nfsFS.ClientRPCStats()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	n.baseMetrics.collectCounter(ch, float64(stats.ClientRPC.AuthRefreshes), []string{})
}

// NFS Procedures
type NFSProcedures struct {
	*baseMetrics
}

func NewNFSProcedures() *NFSProcedures {
	return &NFSProcedures{
		NewMetrics("node_nfs_requests_total",
			"Number of NFS procedures invoked.",
			[]string{"proto", "method"}),
	}
}

func (n *NFSProcedures) Collect(ch chan<- prometheus.Metric) {
	if err := initNFSFS(); err != nil {
		return
	}

	stats, err := nfsFS.ClientRPCStats()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	// NFSv2 Stats
	const proto2 = "2"
	v2 := reflect.ValueOf(&stats.V2Stats).Elem()
	for i := 0; i < v2.NumField(); i++ {
		field := v2.Field(i)
		n.baseMetrics.collectCounter(ch, float64(field.Uint()), []string{proto2, v2.Type().Field(i).Name})
	}

	// NFSv3 Stats
	const proto3 = "3"
	v3 := reflect.ValueOf(&stats.V3Stats).Elem()
	for i := 0; i < v3.NumField(); i++ {
		field := v3.Field(i)
		n.baseMetrics.collectCounter(ch, float64(field.Uint()), []string{proto3, v3.Type().Field(i).Name})
	}

	// NFSv4 Stats
	const proto4 = "4"
	v4 := reflect.ValueOf(&stats.ClientV4Stats).Elem()
	for i := 0; i < v4.NumField(); i++ {
		field := v4.Field(i)
		n.baseMetrics.collectCounter(ch, float64(field.Uint()), []string{proto4, v4.Type().Field(i).Name})
	}
} 
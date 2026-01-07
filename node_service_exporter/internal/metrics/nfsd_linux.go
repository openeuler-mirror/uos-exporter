//go:build !nonfsd && linux
// +build !nonfsd,linux

package metrics

import (
	"errors"
	"fmt"
	"os"
	"node_service_exporter/internal/exporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs/nfs"
)

func init() {
	exporter.Register(NewNFSdReplyCacheHits())
	exporter.Register(NewNFSdReplyCacheMisses())
	exporter.Register(NewNFSdReplyCacheNoCache())
	exporter.Register(NewNFSdFileHandlesStale())
	exporter.Register(NewNFSdDiskBytesRead())
	exporter.Register(NewNFSdDiskBytesWritten())
	exporter.Register(NewNFSdServerThreads())
	exporter.Register(NewNFSdReadAheadCacheSize())
	exporter.Register(NewNFSdReadAheadCacheNotFound())
	exporter.Register(NewNFSdNetworkPackets())
	exporter.Register(NewNFSdNetworkConnections())
	exporter.Register(NewNFSdRPCErrors())
	exporter.Register(NewNFSdServerRPCs())
	exporter.Register(NewNFSdRequests())
}

var nfsdFS nfs.FS
var nfsdInitialized bool

func initNFSdFS() error {
	if !nfsdInitialized {
		fs, err := nfs.NewFS("/proc")
		if err != nil {
			return fmt.Errorf("failed to open procfs: %w", err)
		}
		nfsdFS = fs
		nfsdInitialized = true
	}
	return nil
}

// NFSd Reply Cache Hits
type NFSdReplyCacheHits struct {
	*baseMetrics
}

func NewNFSdReplyCacheHits() *NFSdReplyCacheHits {
	return &NFSdReplyCacheHits{
		NewMetrics("node_nfsd_reply_cache_hits_total",
			"Total number of NFSd Reply Cache hits (client lost server response).",
			[]string{}),
	}
}

func (n *NFSdReplyCacheHits) Collect(ch chan<- prometheus.Metric) {
	if err := initNFSdFS(); err != nil {
		return
	}

	stats, err := nfsdFS.ServerRPCStats()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	n.baseMetrics.collectCounter(ch, float64(stats.ReplyCache.Hits), []string{})
}

// NFSd Reply Cache Misses
type NFSdReplyCacheMisses struct {
	*baseMetrics
}

func NewNFSdReplyCacheMisses() *NFSdReplyCacheMisses {
	return &NFSdReplyCacheMisses{
		NewMetrics("node_nfsd_reply_cache_misses_total",
			"Total number of NFSd Reply Cache an operation that requires caching (idempotent).",
			[]string{}),
	}
}

func (n *NFSdReplyCacheMisses) Collect(ch chan<- prometheus.Metric) {
	if err := initNFSdFS(); err != nil {
		return
	}

	stats, err := nfsdFS.ServerRPCStats()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	n.baseMetrics.collectCounter(ch, float64(stats.ReplyCache.Misses), []string{})
}

// NFSd Reply Cache NoCache
type NFSdReplyCacheNoCache struct {
	*baseMetrics
}

func NewNFSdReplyCacheNoCache() *NFSdReplyCacheNoCache {
	return &NFSdReplyCacheNoCache{
		NewMetrics("node_nfsd_reply_cache_nocache_total",
			"Total number of NFSd Reply Cache non-idempotent operations (rename/delete/…).",
			[]string{}),
	}
}

func (n *NFSdReplyCacheNoCache) Collect(ch chan<- prometheus.Metric) {
	if err := initNFSdFS(); err != nil {
		return
	}

	stats, err := nfsdFS.ServerRPCStats()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	n.baseMetrics.collectCounter(ch, float64(stats.ReplyCache.NoCache), []string{})
}

// NFSd File Handles Stale
type NFSdFileHandlesStale struct {
	*baseMetrics
}

func NewNFSdFileHandlesStale() *NFSdFileHandlesStale {
	return &NFSdFileHandlesStale{
		NewMetrics("node_nfsd_file_handles_stale_total",
			"Total number of NFSd stale file handles",
			[]string{}),
	}
}

func (n *NFSdFileHandlesStale) Collect(ch chan<- prometheus.Metric) {
	if err := initNFSdFS(); err != nil {
		return
	}

	stats, err := nfsdFS.ServerRPCStats()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	n.baseMetrics.collectCounter(ch, float64(stats.FileHandles.Stale), []string{})
}

// NFSd Disk Bytes Read
type NFSdDiskBytesRead struct {
	*baseMetrics
}

func NewNFSdDiskBytesRead() *NFSdDiskBytesRead {
	return &NFSdDiskBytesRead{
		NewMetrics("node_nfsd_disk_bytes_read_total",
			"Total NFSd bytes read.",
			[]string{}),
	}
}

func (n *NFSdDiskBytesRead) Collect(ch chan<- prometheus.Metric) {
	if err := initNFSdFS(); err != nil {
		return
	}

	stats, err := nfsdFS.ServerRPCStats()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	n.baseMetrics.collectCounter(ch, float64(stats.InputOutput.Read), []string{})
}

// NFSd Disk Bytes Written
type NFSdDiskBytesWritten struct {
	*baseMetrics
}

func NewNFSdDiskBytesWritten() *NFSdDiskBytesWritten {
	return &NFSdDiskBytesWritten{
		NewMetrics("node_nfsd_disk_bytes_written_total",
			"Total NFSd bytes written.",
			[]string{}),
	}
}

func (n *NFSdDiskBytesWritten) Collect(ch chan<- prometheus.Metric) {
	if err := initNFSdFS(); err != nil {
		return
	}

	stats, err := nfsdFS.ServerRPCStats()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	n.baseMetrics.collectCounter(ch, float64(stats.InputOutput.Write), []string{})
}

// NFSd Server Threads
type NFSdServerThreads struct {
	*baseMetrics
}

func NewNFSdServerThreads() *NFSdServerThreads {
	return &NFSdServerThreads{
		NewMetrics("node_nfsd_server_threads",
			"Total number of NFSd kernel threads that are running.",
			[]string{}),
	}
}

func (n *NFSdServerThreads) Collect(ch chan<- prometheus.Metric) {
	if err := initNFSdFS(); err != nil {
		return
	}

	stats, err := nfsdFS.ServerRPCStats()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	n.baseMetrics.collect(ch, float64(stats.Threads.Threads), []string{})
}

// NFSd Read Ahead Cache Size
type NFSdReadAheadCacheSize struct {
	*baseMetrics
}

func NewNFSdReadAheadCacheSize() *NFSdReadAheadCacheSize {
	return &NFSdReadAheadCacheSize{
		NewMetrics("node_nfsd_read_ahead_cache_size_blocks",
			"How large the read ahead cache is in blocks.",
			[]string{}),
	}
}

func (n *NFSdReadAheadCacheSize) Collect(ch chan<- prometheus.Metric) {
	if err := initNFSdFS(); err != nil {
		return
	}

	stats, err := nfsdFS.ServerRPCStats()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	n.baseMetrics.collect(ch, float64(stats.ReadAheadCache.CacheSize), []string{})
}

// NFSd Read Ahead Cache Not Found
type NFSdReadAheadCacheNotFound struct {
	*baseMetrics
}

func NewNFSdReadAheadCacheNotFound() *NFSdReadAheadCacheNotFound {
	return &NFSdReadAheadCacheNotFound{
		NewMetrics("node_nfsd_read_ahead_cache_not_found_total",
			"Total number of NFSd read ahead cache not found.",
			[]string{}),
	}
}

func (n *NFSdReadAheadCacheNotFound) Collect(ch chan<- prometheus.Metric) {
	if err := initNFSdFS(); err != nil {
		return
	}

	stats, err := nfsdFS.ServerRPCStats()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	n.baseMetrics.collectCounter(ch, float64(stats.ReadAheadCache.NotFound), []string{})
}

// NFSd Network Packets
type NFSdNetworkPackets struct {
	*baseMetrics
}

func NewNFSdNetworkPackets() *NFSdNetworkPackets {
	return &NFSdNetworkPackets{
		NewMetrics("node_nfsd_packets_total",
			"Total NFSd network packets (sent+received) by protocol type.",
			[]string{"proto"}),
	}
}

func (n *NFSdNetworkPackets) Collect(ch chan<- prometheus.Metric) {
	if err := initNFSdFS(); err != nil {
		return
	}

	stats, err := nfsdFS.ServerRPCStats()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	n.baseMetrics.collectCounter(ch, float64(stats.Network.UDPCount), []string{"udp"})
	n.baseMetrics.collectCounter(ch, float64(stats.Network.TCPCount), []string{"tcp"})
}

// NFSd Network Connections
type NFSdNetworkConnections struct {
	*baseMetrics
}

func NewNFSdNetworkConnections() *NFSdNetworkConnections {
	return &NFSdNetworkConnections{
		NewMetrics("node_nfsd_connections_total",
			"Total number of NFSd TCP connections.",
			[]string{}),
	}
}

func (n *NFSdNetworkConnections) Collect(ch chan<- prometheus.Metric) {
	if err := initNFSdFS(); err != nil {
		return
	}

	stats, err := nfsdFS.ServerRPCStats()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	n.baseMetrics.collectCounter(ch, float64(stats.Network.TCPConnect), []string{})
}

// NFSd RPC Errors
type NFSdRPCErrors struct {
	*baseMetrics
}

func NewNFSdRPCErrors() *NFSdRPCErrors {
	return &NFSdRPCErrors{
		NewMetrics("node_nfsd_rpc_errors_total",
			"Total number of NFSd RPC errors by error type.",
			[]string{"error"}),
	}
}

func (n *NFSdRPCErrors) Collect(ch chan<- prometheus.Metric) {
	if err := initNFSdFS(); err != nil {
		return
	}

	stats, err := nfsdFS.ServerRPCStats()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	n.baseMetrics.collectCounter(ch, float64(stats.ServerRPC.BadFmt), []string{"fmt"})
	n.baseMetrics.collectCounter(ch, float64(stats.ServerRPC.BadAuth), []string{"auth"})
	n.baseMetrics.collectCounter(ch, float64(stats.ServerRPC.BadcInt), []string{"cInt"})
}

// NFSd Server RPCs
type NFSdServerRPCs struct {
	*baseMetrics
}

func NewNFSdServerRPCs() *NFSdServerRPCs {
	return &NFSdServerRPCs{
		NewMetrics("node_nfsd_server_rpcs_total",
			"Total number of NFSd RPCs.",
			[]string{}),
	}
}

func (n *NFSdServerRPCs) Collect(ch chan<- prometheus.Metric) {
	if err := initNFSdFS(); err != nil {
		return
	}

	stats, err := nfsdFS.ServerRPCStats()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	n.baseMetrics.collectCounter(ch, float64(stats.ServerRPC.RPCCount), []string{})
}

// NFSd Requests
type NFSdRequests struct {
	*baseMetrics
}

func NewNFSdRequests() *NFSdRequests {
	return &NFSdRequests{
		NewMetrics("node_nfsd_requests_total",
			"Total number NFSd Requests by method and protocol.",
			[]string{"proto", "method"}),
	}
}

func (n *NFSdRequests) Collect(ch chan<- prometheus.Metric) {
	if err := initNFSdFS(); err != nil {
		return
	}

	stats, err := nfsdFS.ServerRPCStats()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	// NFSv2 Stats
	n.updateNFSdRequestsv2Stats(ch, &stats.V2Stats)
	// NFSv3 Stats
	n.updateNFSdRequestsv3Stats(ch, &stats.V3Stats)
	// NFSv4 Stats
	n.updateNFSdRequestsv4Stats(ch, &stats.V4Ops)
	// Special NFSv4 operation
	n.baseMetrics.collectCounter(ch, float64(stats.WdelegGetattr), []string{"4", "WdelegGetattr"})
}

func (n *NFSdRequests) updateNFSdRequestsv2Stats(ch chan<- prometheus.Metric, s *nfs.V2Stats) {
	const proto = "2"
	n.baseMetrics.collectCounter(ch, float64(s.GetAttr), []string{proto, "GetAttr"})
	n.baseMetrics.collectCounter(ch, float64(s.SetAttr), []string{proto, "SetAttr"})
	n.baseMetrics.collectCounter(ch, float64(s.Root), []string{proto, "Root"})
	n.baseMetrics.collectCounter(ch, float64(s.Lookup), []string{proto, "Lookup"})
	n.baseMetrics.collectCounter(ch, float64(s.ReadLink), []string{proto, "ReadLink"})
	n.baseMetrics.collectCounter(ch, float64(s.Read), []string{proto, "Read"})
	n.baseMetrics.collectCounter(ch, float64(s.WrCache), []string{proto, "WrCache"})
	n.baseMetrics.collectCounter(ch, float64(s.Write), []string{proto, "Write"})
	n.baseMetrics.collectCounter(ch, float64(s.Create), []string{proto, "Create"})
	n.baseMetrics.collectCounter(ch, float64(s.Remove), []string{proto, "Remove"})
	n.baseMetrics.collectCounter(ch, float64(s.Rename), []string{proto, "Rename"})
	n.baseMetrics.collectCounter(ch, float64(s.Link), []string{proto, "Link"})
	n.baseMetrics.collectCounter(ch, float64(s.SymLink), []string{proto, "SymLink"})
	n.baseMetrics.collectCounter(ch, float64(s.MkDir), []string{proto, "MkDir"})
	n.baseMetrics.collectCounter(ch, float64(s.RmDir), []string{proto, "RmDir"})
	n.baseMetrics.collectCounter(ch, float64(s.ReadDir), []string{proto, "ReadDir"})
	n.baseMetrics.collectCounter(ch, float64(s.FsStat), []string{proto, "FsStat"})
}

func (n *NFSdRequests) updateNFSdRequestsv3Stats(ch chan<- prometheus.Metric, s *nfs.V3Stats) {
	const proto = "3"
	n.baseMetrics.collectCounter(ch, float64(s.GetAttr), []string{proto, "GetAttr"})
	n.baseMetrics.collectCounter(ch, float64(s.SetAttr), []string{proto, "SetAttr"})
	n.baseMetrics.collectCounter(ch, float64(s.Lookup), []string{proto, "Lookup"})
	n.baseMetrics.collectCounter(ch, float64(s.Access), []string{proto, "Access"})
	n.baseMetrics.collectCounter(ch, float64(s.ReadLink), []string{proto, "ReadLink"})
	n.baseMetrics.collectCounter(ch, float64(s.Read), []string{proto, "Read"})
	n.baseMetrics.collectCounter(ch, float64(s.Write), []string{proto, "Write"})
	n.baseMetrics.collectCounter(ch, float64(s.Create), []string{proto, "Create"})
	n.baseMetrics.collectCounter(ch, float64(s.MkDir), []string{proto, "MkDir"})
	n.baseMetrics.collectCounter(ch, float64(s.SymLink), []string{proto, "SymLink"})
	n.baseMetrics.collectCounter(ch, float64(s.MkNod), []string{proto, "MkNod"})
	n.baseMetrics.collectCounter(ch, float64(s.Remove), []string{proto, "Remove"})
	n.baseMetrics.collectCounter(ch, float64(s.RmDir), []string{proto, "RmDir"})
	n.baseMetrics.collectCounter(ch, float64(s.Rename), []string{proto, "Rename"})
	n.baseMetrics.collectCounter(ch, float64(s.Link), []string{proto, "Link"})
	n.baseMetrics.collectCounter(ch, float64(s.ReadDir), []string{proto, "ReadDir"})
	n.baseMetrics.collectCounter(ch, float64(s.ReadDirPlus), []string{proto, "ReadDirPlus"})
	n.baseMetrics.collectCounter(ch, float64(s.FsStat), []string{proto, "FsStat"})
	n.baseMetrics.collectCounter(ch, float64(s.FsInfo), []string{proto, "FsInfo"})
	n.baseMetrics.collectCounter(ch, float64(s.PathConf), []string{proto, "PathConf"})
	n.baseMetrics.collectCounter(ch, float64(s.Commit), []string{proto, "Commit"})
}

func (n *NFSdRequests) updateNFSdRequestsv4Stats(ch chan<- prometheus.Metric, s *nfs.V4Ops) {
	const proto = "4"
	n.baseMetrics.collectCounter(ch, float64(s.Access), []string{proto, "Access"})
	n.baseMetrics.collectCounter(ch, float64(s.Close), []string{proto, "Close"})
	n.baseMetrics.collectCounter(ch, float64(s.Commit), []string{proto, "Commit"})
	n.baseMetrics.collectCounter(ch, float64(s.Create), []string{proto, "Create"})
	n.baseMetrics.collectCounter(ch, float64(s.DelegPurge), []string{proto, "DelegPurge"})
	n.baseMetrics.collectCounter(ch, float64(s.DelegReturn), []string{proto, "DelegReturn"})
	n.baseMetrics.collectCounter(ch, float64(s.GetAttr), []string{proto, "GetAttr"})
	n.baseMetrics.collectCounter(ch, float64(s.GetFH), []string{proto, "GetFH"})
	n.baseMetrics.collectCounter(ch, float64(s.Link), []string{proto, "Link"})
	n.baseMetrics.collectCounter(ch, float64(s.Lock), []string{proto, "Lock"})
	n.baseMetrics.collectCounter(ch, float64(s.Lockt), []string{proto, "Lockt"})
	n.baseMetrics.collectCounter(ch, float64(s.Locku), []string{proto, "Locku"})
	n.baseMetrics.collectCounter(ch, float64(s.Lookup), []string{proto, "Lookup"})
	n.baseMetrics.collectCounter(ch, float64(s.LookupRoot), []string{proto, "LookupRoot"})
	n.baseMetrics.collectCounter(ch, float64(s.Nverify), []string{proto, "Nverify"})
	n.baseMetrics.collectCounter(ch, float64(s.Open), []string{proto, "Open"})
	n.baseMetrics.collectCounter(ch, float64(s.OpenAttr), []string{proto, "OpenAttr"})
	n.baseMetrics.collectCounter(ch, float64(s.OpenConfirm), []string{proto, "OpenConfirm"})
	n.baseMetrics.collectCounter(ch, float64(s.OpenDgrd), []string{proto, "OpenDgrd"})
	n.baseMetrics.collectCounter(ch, float64(s.PutFH), []string{proto, "PutFH"})
	n.baseMetrics.collectCounter(ch, float64(s.Read), []string{proto, "Read"})
	n.baseMetrics.collectCounter(ch, float64(s.ReadDir), []string{proto, "ReadDir"})
	n.baseMetrics.collectCounter(ch, float64(s.ReadLink), []string{proto, "ReadLink"})
	n.baseMetrics.collectCounter(ch, float64(s.Remove), []string{proto, "Remove"})
	n.baseMetrics.collectCounter(ch, float64(s.Rename), []string{proto, "Rename"})
	n.baseMetrics.collectCounter(ch, float64(s.Renew), []string{proto, "Renew"})
	n.baseMetrics.collectCounter(ch, float64(s.RestoreFH), []string{proto, "RestoreFH"})
	n.baseMetrics.collectCounter(ch, float64(s.SaveFH), []string{proto, "SaveFH"})
	n.baseMetrics.collectCounter(ch, float64(s.SecInfo), []string{proto, "SecInfo"})
	n.baseMetrics.collectCounter(ch, float64(s.SetAttr), []string{proto, "SetAttr"})
	n.baseMetrics.collectCounter(ch, float64(s.SetClientID), []string{proto, "SetClientID"})
	n.baseMetrics.collectCounter(ch, float64(s.SetClientIDConfirm), []string{proto, "SetClientIDConfirm"})
	n.baseMetrics.collectCounter(ch, float64(s.Verify), []string{proto, "Verify"})
	n.baseMetrics.collectCounter(ch, float64(s.Write), []string{proto, "Write"})
	n.baseMetrics.collectCounter(ch, float64(s.RelLockOwner), []string{proto, "RelLockOwner"})
} 
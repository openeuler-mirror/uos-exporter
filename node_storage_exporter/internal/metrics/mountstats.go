package metrics

import (
	"log/slog"

	"node_storage_exporter/internal/exporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
)

var (
	// 64-bit float mantissa: https://en.wikipedia.org/wiki/Double-precision_floating-point_format
	float64Mantissa uint64 = 9007199254740992
)

func init() {
	exporter.Register(NewMountStatsCollector())
}

// used to uniquely identify an NFS mount to prevent duplicates
type nfsDeviceIdentifier struct {
	Device       string
	Protocol     string
	MountAddress string
}

type MountStatsCollector struct {
	// General statistics
	NFSAgeSecondsTotal *prometheus.Desc

	// Byte statistics
	NFSReadBytesTotal        *prometheus.Desc
	NFSWriteBytesTotal       *prometheus.Desc
	NFSDirectReadBytesTotal  *prometheus.Desc
	NFSDirectWriteBytesTotal *prometheus.Desc
	NFSTotalReadBytesTotal   *prometheus.Desc
	NFSTotalWriteBytesTotal  *prometheus.Desc
	NFSReadPagesTotal        *prometheus.Desc
	NFSWritePagesTotal       *prometheus.Desc

	// Per-operation statistics
	NFSOperationsRequestsTotal            *prometheus.Desc
	NFSOperationsTransmissionsTotal       *prometheus.Desc
	NFSOperationsMajorTimeoutsTotal       *prometheus.Desc
	NFSOperationsSentBytesTotal           *prometheus.Desc
	NFSOperationsReceivedBytesTotal       *prometheus.Desc
	NFSOperationsQueueTimeSecondsTotal    *prometheus.Desc
	NFSOperationsResponseTimeSecondsTotal *prometheus.Desc
	NFSOperationsRequestTimeSecondsTotal  *prometheus.Desc

	// Transport statistics
	NFSTransportBindTotal              *prometheus.Desc
	NFSTransportConnectTotal           *prometheus.Desc
	NFSTransportIdleTimeSeconds        *prometheus.Desc
	NFSTransportSendsTotal             *prometheus.Desc
	NFSTransportReceivesTotal          *prometheus.Desc
	NFSTransportBadTransactionIDsTotal *prometheus.Desc
	NFSTransportBacklogQueueTotal      *prometheus.Desc
	NFSTransportMaximumRPCSlots        *prometheus.Desc
	NFSTransportSendingQueueTotal      *prometheus.Desc
	NFSTransportPendingQueueTotal      *prometheus.Desc

	// Event statistics
	NFSEventInodeRevalidateTotal     *prometheus.Desc
	NFSEventDnodeRevalidateTotal     *prometheus.Desc
	NFSEventDataInvalidateTotal      *prometheus.Desc
	NFSEventAttributeInvalidateTotal *prometheus.Desc
	NFSEventVFSOpenTotal             *prometheus.Desc
	NFSEventVFSLookupTotal           *prometheus.Desc
	NFSEventVFSAccessTotal           *prometheus.Desc
	NFSEventVFSUpdatePageTotal       *prometheus.Desc
	NFSEventVFSReadPageTotal         *prometheus.Desc
	NFSEventVFSReadPagesTotal        *prometheus.Desc
	NFSEventVFSWritePageTotal        *prometheus.Desc
	NFSEventVFSWritePagesTotal       *prometheus.Desc
	NFSEventVFSGetdentsTotal         *prometheus.Desc
	NFSEventVFSSetattrTotal          *prometheus.Desc
	NFSEventVFSFlushTotal            *prometheus.Desc
	NFSEventVFSFsyncTotal            *prometheus.Desc
	NFSEventVFSLockTotal             *prometheus.Desc
	NFSEventVFSFileReleaseTotal      *prometheus.Desc
	NFSEventTruncationTotal          *prometheus.Desc
	NFSEventWriteExtensionTotal      *prometheus.Desc
	NFSEventSillyRenameTotal         *prometheus.Desc
	NFSEventShortReadTotal           *prometheus.Desc
	NFSEventShortWriteTotal          *prometheus.Desc
	NFSEventJukeboxDelayTotal        *prometheus.Desc
	NFSEventPNFSReadTotal            *prometheus.Desc
	NFSEventPNFSWriteTotal           *prometheus.Desc

	proc   procfs.Proc
	logger *slog.Logger
}

func NewMountStatsCollector() *MountStatsCollector {
	logger := slog.Default()

	fs, err := procfs.NewFS("/proc")
	if err != nil {
		logger.Debug("failed to open procfs for mountstats", "error", err)
		return &MountStatsCollector{
			logger: logger,
		}
	}

	proc, err := fs.Self()
	if err != nil {
		logger.Debug("failed to open /proc/self for mountstats", "error", err)
		return &MountStatsCollector{
			logger: logger,
		}
	}

	const subsystem = "mountstats_nfs"
	var (
		labels   = []string{"export", "protocol", "mountaddr"}
		opLabels = []string{"export", "protocol", "mountaddr", "operation"}
	)

	return &MountStatsCollector{
		NFSAgeSecondsTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "age_seconds_total"),
			"The age of the NFS mount in seconds.",
			labels,
			nil,
		),

		NFSReadBytesTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "read_bytes_total"),
			"Number of bytes read using the read() syscall.",
			labels,
			nil,
		),

		NFSWriteBytesTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "write_bytes_total"),
			"Number of bytes written using the write() syscall.",
			labels,
			nil,
		),

		NFSDirectReadBytesTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "direct_read_bytes_total"),
			"Number of bytes read using the read() syscall in O_DIRECT mode.",
			labels,
			nil,
		),

		NFSDirectWriteBytesTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "direct_write_bytes_total"),
			"Number of bytes written using the write() syscall in O_DIRECT mode.",
			labels,
			nil,
		),

		NFSTotalReadBytesTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "total_read_bytes_total"),
			"Number of bytes read from the NFS server, in total.",
			labels,
			nil,
		),

		NFSTotalWriteBytesTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "total_write_bytes_total"),
			"Number of bytes written to the NFS server, in total.",
			labels,
			nil,
		),

		NFSReadPagesTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "read_pages_total"),
			"Number of pages read directly via mmap()'d files.",
			labels,
			nil,
		),

		NFSWritePagesTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "write_pages_total"),
			"Number of pages written directly via mmap()'d files.",
			labels,
			nil,
		),

		NFSTransportBindTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "transport_bind_total"),
			"Number of times the client has had to establish a connection from scratch to the NFS server.",
			labels,
			nil,
		),

		NFSTransportConnectTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "transport_connect_total"),
			"Number of times the client has made a TCP connection to the NFS server.",
			labels,
			nil,
		),

		NFSTransportIdleTimeSeconds: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "transport_idle_time_seconds"),
			"Duration since the NFS mount last saw any RPC traffic, in seconds.",
			labels,
			nil,
		),

		NFSTransportSendsTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "transport_sends_total"),
			"Number of RPC requests for this mount sent to the NFS server.",
			labels,
			nil,
		),

		NFSTransportReceivesTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "transport_receives_total"),
			"Number of RPC responses for this mount received from the NFS server.",
			labels,
			nil,
		),

		NFSTransportBadTransactionIDsTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "transport_bad_transaction_ids_total"),
			"Number of times the NFS server sent a response with a transaction ID unknown to this client.",
			labels,
			nil,
		),

		NFSTransportBacklogQueueTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "transport_backlog_queue_total"),
			"Total number of items added to the RPC backlog queue.",
			labels,
			nil,
		),

		NFSTransportMaximumRPCSlots: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "transport_maximum_rpc_slots"),
			"Maximum number of simultaneously active RPC requests ever used.",
			labels,
			nil,
		),

		NFSTransportSendingQueueTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "transport_sending_queue_total"),
			"Total number of items added to the RPC transmission sending queue.",
			labels,
			nil,
		),

		NFSTransportPendingQueueTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "transport_pending_queue_total"),
			"Total number of items added to the RPC transmission pending queue.",
			labels,
			nil,
		),

		NFSOperationsRequestsTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "operations_requests_total"),
			"Number of requests performed for a given operation.",
			opLabels,
			nil,
		),

		NFSOperationsTransmissionsTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "operations_transmissions_total"),
			"Number of times an actual RPC request has been transmitted for a given operation.",
			opLabels,
			nil,
		),

		NFSOperationsMajorTimeoutsTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "operations_major_timeouts_total"),
			"Number of times a request has had a major timeout for a given operation.",
			opLabels,
			nil,
		),

		NFSOperationsSentBytesTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "operations_sent_bytes_total"),
			"Number of bytes sent for a given operation, including RPC headers and payload.",
			opLabels,
			nil,
		),

		NFSOperationsReceivedBytesTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "operations_received_bytes_total"),
			"Number of bytes received for a given operation, including RPC headers and payload.",
			opLabels,
			nil,
		),

		NFSOperationsQueueTimeSecondsTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "operations_queue_time_seconds_total"),
			"Duration all requests spent queued for transmission for a given operation before they were sent, in seconds.",
			opLabels,
			nil,
		),

		NFSOperationsResponseTimeSecondsTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "operations_response_time_seconds_total"),
			"Duration all requests took to get a reply back after a request for a given operation was transmitted, in seconds.",
			opLabels,
			nil,
		),

		NFSOperationsRequestTimeSecondsTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "operations_request_time_seconds_total"),
			"Duration all requests took from when a request was enqueued to when it was completely handled for a given operation, in seconds.",
			opLabels,
			nil,
		),

		NFSEventInodeRevalidateTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "event_inode_revalidate_total"),
			"Number of times cached inode attributes are re-validated from the server.",
			labels,
			nil,
		),

		NFSEventDnodeRevalidateTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "event_dnode_revalidate_total"),
			"Number of times cached dentry nodes are re-validated from the server.",
			labels,
			nil,
		),

		NFSEventDataInvalidateTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "event_data_invalidate_total"),
			"Number of times an inode cache is cleared.",
			labels,
			nil,
		),

		NFSEventAttributeInvalidateTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "event_attribute_invalidate_total"),
			"Number of times cached inode attributes are invalidated.",
			labels,
			nil,
		),

		NFSEventVFSOpenTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "event_vfs_open_total"),
			"Number of times cached inode attributes are invalidated.",
			labels,
			nil,
		),

		NFSEventVFSLookupTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "event_vfs_lookup_total"),
			"Number of times a directory lookup has occurred.",
			labels,
			nil,
		),

		NFSEventVFSAccessTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "event_vfs_access_total"),
			"Number of times permissions have been checked.",
			labels,
			nil,
		),

		NFSEventVFSUpdatePageTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "event_vfs_update_page_total"),
			"Number of updates (and potential writes) to pages.",
			labels,
			nil,
		),

		NFSEventVFSReadPageTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "event_vfs_read_page_total"),
			"Number of pages read directly via mmap()'d files.",
			labels,
			nil,
		),

		NFSEventVFSReadPagesTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "event_vfs_read_pages_total"),
			"Number of times a group of pages have been read.",
			labels,
			nil,
		),

		NFSEventVFSWritePageTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "event_vfs_write_page_total"),
			"Number of pages written directly via mmap()'d files.",
			labels,
			nil,
		),

		NFSEventVFSWritePagesTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "event_vfs_write_pages_total"),
			"Number of times a group of pages have been written.",
			labels,
			nil,
		),

		NFSEventVFSGetdentsTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "event_vfs_getdents_total"),
			"Number of times directory entries have been read with getdents().",
			labels,
			nil,
		),

		NFSEventVFSSetattrTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "event_vfs_setattr_total"),
			"Number of times directory entries have been read with getdents().",
			labels,
			nil,
		),

		NFSEventVFSFlushTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "event_vfs_flush_total"),
			"Number of pending writes that have been forcefully flushed to the server.",
			labels,
			nil,
		),

		NFSEventVFSFsyncTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "event_vfs_fsync_total"),
			"Number of times fsync() has been called on directories and files.",
			labels,
			nil,
		),

		NFSEventVFSLockTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "event_vfs_lock_total"),
			"Number of times locking has been attempted on a file.",
			labels,
			nil,
		),

		NFSEventVFSFileReleaseTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "event_vfs_file_release_total"),
			"Number of times files have been closed and released.",
			labels,
			nil,
		),

		NFSEventTruncationTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "event_truncation_total"),
			"Number of times files have been truncated.",
			labels,
			nil,
		),

		NFSEventWriteExtensionTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "event_write_extension_total"),
			"Number of times a file has been grown due to writes beyond its existing end.",
			labels,
			nil,
		),

		NFSEventSillyRenameTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "event_silly_rename_total"),
			"Number of times a file was removed while still open by another process.",
			labels,
			nil,
		),

		NFSEventShortReadTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "event_short_read_total"),
			"Number of times the NFS server gave less data than expected while reading.",
			labels,
			nil,
		),

		NFSEventShortWriteTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "event_short_write_total"),
			"Number of times the NFS server wrote less data than expected while writing.",
			labels,
			nil,
		),

		NFSEventJukeboxDelayTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "event_jukebox_delay_total"),
			"Number of times the NFS server indicated EJUKEBOX; retrieving data from offline storage.",
			labels,
			nil,
		),

		NFSEventPNFSReadTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "event_pnfs_read_total"),
			"Number of NFS v4.1+ pNFS reads.",
			labels,
			nil,
		),

		NFSEventPNFSWriteTotal: prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, "event_pnfs_write_total"),
			"Number of NFS v4.1+ pNFS writes.",
			labels,
			nil,
		),

		proc:   proc,
		logger: logger,
	}
}

func (c *MountStatsCollector) Collect(ch chan<- prometheus.Metric) {
	if c.proc.PID == 0 {
		c.logger.Debug("No valid proc available for mountstats")
		return
	}

	mounts, err := c.proc.MountStats()
	if err != nil {
		c.logger.Debug("Error parsing mount stats", "error", err)
		return
	}

	// Deduplicate NFS mounts by device, protocol, and mount address.
	// This is needed because the same NFS mount may be present multiple
	// times in /proc/self/mountstats if it is bind mounted multiple times.
	seenNFSMounts := make(map[nfsDeviceIdentifier]bool)

	for _, mount := range mounts {
		if mount.Stats == nil {
			c.logger.Debug("Mount has no stats", "mount", mount.Device)
			continue
		}

		nfsStats, ok := mount.Stats.(*procfs.MountStatsNFS)
		if !ok {
			continue
		}

		// Handle Transport as an array
		for _, transport := range nfsStats.Transport {
			protocol := transport.Protocol
			mountAddress := ""
			
			// Try to get mount address from opts if available
			if addr, exists := nfsStats.Opts["addr"]; exists {
				mountAddress = addr
			}

			mountIdentifier := nfsDeviceIdentifier{
				Device:       mount.Device,
				Protocol:     protocol,
				MountAddress: mountAddress,
			}

			if seenNFSMounts[mountIdentifier] {
				c.logger.Debug("Skipping duplicate NFS mount", "device", mount.Device, "protocol", protocol, "mountaddr", mountAddress)
				continue
			}
			seenNFSMounts[mountIdentifier] = true

			c.updateNFSStats(ch, nfsStats, mount.Device, protocol, mountAddress)
		}
	}
}

func (c *MountStatsCollector) updateNFSStats(ch chan<- prometheus.Metric, s *procfs.MountStatsNFS, export, protocol, mountAddress string) {
	labelValues := []string{export, protocol, mountAddress}

	ch <- prometheus.MustNewConstMetric(c.NFSAgeSecondsTotal, prometheus.CounterValue, s.Age.Seconds(), labelValues...)

	ch <- prometheus.MustNewConstMetric(c.NFSReadBytesTotal, prometheus.CounterValue, float64(s.Bytes.Read), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.NFSWriteBytesTotal, prometheus.CounterValue, float64(s.Bytes.Write), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.NFSDirectReadBytesTotal, prometheus.CounterValue, float64(s.Bytes.DirectRead), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.NFSDirectWriteBytesTotal, prometheus.CounterValue, float64(s.Bytes.DirectWrite), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.NFSTotalReadBytesTotal, prometheus.CounterValue, float64(s.Bytes.ReadTotal), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.NFSTotalWriteBytesTotal, prometheus.CounterValue, float64(s.Bytes.WriteTotal), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.NFSReadPagesTotal, prometheus.CounterValue, float64(s.Bytes.ReadPages), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.NFSWritePagesTotal, prometheus.CounterValue, float64(s.Bytes.WritePages), labelValues...)

	// Transport statistics - iterate through all transport entries
	for _, transport := range s.Transport {
		ch <- prometheus.MustNewConstMetric(c.NFSTransportBindTotal, prometheus.CounterValue, float64(transport.Bind), labelValues...)
		ch <- prometheus.MustNewConstMetric(c.NFSTransportConnectTotal, prometheus.CounterValue, float64(transport.Connect), labelValues...)
		ch <- prometheus.MustNewConstMetric(c.NFSTransportIdleTimeSeconds, prometheus.GaugeValue, float64(transport.IdleTimeSeconds), labelValues...)
		ch <- prometheus.MustNewConstMetric(c.NFSTransportSendsTotal, prometheus.CounterValue, float64(transport.Sends), labelValues...)
		ch <- prometheus.MustNewConstMetric(c.NFSTransportReceivesTotal, prometheus.CounterValue, float64(transport.Receives), labelValues...)
		ch <- prometheus.MustNewConstMetric(c.NFSTransportBadTransactionIDsTotal, prometheus.CounterValue, float64(transport.BadTransactionIDs), labelValues...)
		ch <- prometheus.MustNewConstMetric(c.NFSTransportBacklogQueueTotal, prometheus.CounterValue, float64(transport.CumulativeBacklog), labelValues...)
		ch <- prometheus.MustNewConstMetric(c.NFSTransportMaximumRPCSlots, prometheus.GaugeValue, float64(transport.MaximumRPCSlotsUsed), labelValues...)
		ch <- prometheus.MustNewConstMetric(c.NFSTransportSendingQueueTotal, prometheus.CounterValue, float64(transport.CumulativeSendingQueue), labelValues...)
		ch <- prometheus.MustNewConstMetric(c.NFSTransportPendingQueueTotal, prometheus.CounterValue, float64(transport.CumulativePendingQueue), labelValues...)
	}

	for _, op := range s.Operations {
		opLabelValues := append(labelValues, op.Operation)

		ch <- prometheus.MustNewConstMetric(c.NFSOperationsRequestsTotal, prometheus.CounterValue, float64(op.Requests), opLabelValues...)
		ch <- prometheus.MustNewConstMetric(c.NFSOperationsTransmissionsTotal, prometheus.CounterValue, float64(op.Transmissions), opLabelValues...)
		ch <- prometheus.MustNewConstMetric(c.NFSOperationsMajorTimeoutsTotal, prometheus.CounterValue, float64(op.MajorTimeouts), opLabelValues...)
		ch <- prometheus.MustNewConstMetric(c.NFSOperationsSentBytesTotal, prometheus.CounterValue, float64(op.BytesSent), opLabelValues...)
		ch <- prometheus.MustNewConstMetric(c.NFSOperationsReceivedBytesTotal, prometheus.CounterValue, float64(op.BytesReceived), opLabelValues...)

		// Convert time values from milliseconds to seconds.
		ch <- prometheus.MustNewConstMetric(c.NFSOperationsQueueTimeSecondsTotal, prometheus.CounterValue, float64(op.CumulativeQueueMilliseconds)/1000, opLabelValues...)
		ch <- prometheus.MustNewConstMetric(c.NFSOperationsResponseTimeSecondsTotal, prometheus.CounterValue, float64(op.CumulativeTotalResponseMilliseconds)/1000, opLabelValues...)
		ch <- prometheus.MustNewConstMetric(c.NFSOperationsRequestTimeSecondsTotal, prometheus.CounterValue, float64(op.CumulativeTotalRequestMilliseconds)/1000, opLabelValues...)
	}

	ch <- prometheus.MustNewConstMetric(c.NFSEventInodeRevalidateTotal, prometheus.CounterValue, float64(s.Events.InodeRevalidate), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.NFSEventDnodeRevalidateTotal, prometheus.CounterValue, float64(s.Events.DnodeRevalidate), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.NFSEventDataInvalidateTotal, prometheus.CounterValue, float64(s.Events.DataInvalidate), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.NFSEventAttributeInvalidateTotal, prometheus.CounterValue, float64(s.Events.AttributeInvalidate), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.NFSEventVFSOpenTotal, prometheus.CounterValue, float64(s.Events.VFSOpen), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.NFSEventVFSLookupTotal, prometheus.CounterValue, float64(s.Events.VFSLookup), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.NFSEventVFSAccessTotal, prometheus.CounterValue, float64(s.Events.VFSAccess), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.NFSEventVFSUpdatePageTotal, prometheus.CounterValue, float64(s.Events.VFSUpdatePage), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.NFSEventVFSReadPageTotal, prometheus.CounterValue, float64(s.Events.VFSReadPage), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.NFSEventVFSReadPagesTotal, prometheus.CounterValue, float64(s.Events.VFSReadPages), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.NFSEventVFSWritePageTotal, prometheus.CounterValue, float64(s.Events.VFSWritePage), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.NFSEventVFSWritePagesTotal, prometheus.CounterValue, float64(s.Events.VFSWritePages), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.NFSEventVFSGetdentsTotal, prometheus.CounterValue, float64(s.Events.VFSGetdents), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.NFSEventVFSSetattrTotal, prometheus.CounterValue, float64(s.Events.VFSSetattr), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.NFSEventVFSFlushTotal, prometheus.CounterValue, float64(s.Events.VFSFlush), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.NFSEventVFSFsyncTotal, prometheus.CounterValue, float64(s.Events.VFSFsync), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.NFSEventVFSLockTotal, prometheus.CounterValue, float64(s.Events.VFSLock), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.NFSEventVFSFileReleaseTotal, prometheus.CounterValue, float64(s.Events.VFSFileRelease), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.NFSEventTruncationTotal, prometheus.CounterValue, float64(s.Events.Truncation), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.NFSEventWriteExtensionTotal, prometheus.CounterValue, float64(s.Events.WriteExtension), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.NFSEventSillyRenameTotal, prometheus.CounterValue, float64(s.Events.SillyRename), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.NFSEventShortReadTotal, prometheus.CounterValue, float64(s.Events.ShortRead), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.NFSEventShortWriteTotal, prometheus.CounterValue, float64(s.Events.ShortWrite), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.NFSEventJukeboxDelayTotal, prometheus.CounterValue, float64(s.Events.JukeboxDelay), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.NFSEventPNFSReadTotal, prometheus.CounterValue, float64(s.Events.PNFSRead), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.NFSEventPNFSWriteTotal, prometheus.CounterValue, float64(s.Events.PNFSWrite), labelValues...)
}

func (c *MountStatsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.NFSAgeSecondsTotal
	ch <- c.NFSReadBytesTotal
	ch <- c.NFSWriteBytesTotal
	ch <- c.NFSDirectReadBytesTotal
	ch <- c.NFSDirectWriteBytesTotal
	ch <- c.NFSTotalReadBytesTotal
	ch <- c.NFSTotalWriteBytesTotal
	ch <- c.NFSReadPagesTotal
	ch <- c.NFSWritePagesTotal
	ch <- c.NFSOperationsRequestsTotal
	ch <- c.NFSOperationsTransmissionsTotal
	ch <- c.NFSOperationsMajorTimeoutsTotal
	ch <- c.NFSOperationsSentBytesTotal
	ch <- c.NFSOperationsReceivedBytesTotal
	ch <- c.NFSOperationsQueueTimeSecondsTotal
	ch <- c.NFSOperationsResponseTimeSecondsTotal
	ch <- c.NFSOperationsRequestTimeSecondsTotal
	ch <- c.NFSTransportBindTotal
	ch <- c.NFSTransportConnectTotal
	ch <- c.NFSTransportIdleTimeSeconds
	ch <- c.NFSTransportSendsTotal
	ch <- c.NFSTransportReceivesTotal
	ch <- c.NFSTransportBadTransactionIDsTotal
	ch <- c.NFSTransportBacklogQueueTotal
	ch <- c.NFSTransportMaximumRPCSlots
	ch <- c.NFSTransportSendingQueueTotal
	ch <- c.NFSTransportPendingQueueTotal
	ch <- c.NFSEventInodeRevalidateTotal
	ch <- c.NFSEventDnodeRevalidateTotal
	ch <- c.NFSEventDataInvalidateTotal
	ch <- c.NFSEventAttributeInvalidateTotal
	ch <- c.NFSEventVFSOpenTotal
	ch <- c.NFSEventVFSLookupTotal
	ch <- c.NFSEventVFSAccessTotal
	ch <- c.NFSEventVFSUpdatePageTotal
	ch <- c.NFSEventVFSReadPageTotal
	ch <- c.NFSEventVFSReadPagesTotal
	ch <- c.NFSEventVFSWritePageTotal
	ch <- c.NFSEventVFSWritePagesTotal
	ch <- c.NFSEventVFSGetdentsTotal
	ch <- c.NFSEventVFSSetattrTotal
	ch <- c.NFSEventVFSFlushTotal
	ch <- c.NFSEventVFSFsyncTotal
	ch <- c.NFSEventVFSLockTotal
	ch <- c.NFSEventVFSFileReleaseTotal
	ch <- c.NFSEventTruncationTotal
	ch <- c.NFSEventWriteExtensionTotal
	ch <- c.NFSEventSillyRenameTotal
	ch <- c.NFSEventShortReadTotal
	ch <- c.NFSEventShortWriteTotal
	ch <- c.NFSEventJukeboxDelayTotal
	ch <- c.NFSEventPNFSReadTotal
	ch <- c.NFSEventPNFSWriteTotal
} 
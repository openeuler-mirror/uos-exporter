package metrics

import (
	"log/slog"

	"node_storage_exporter/internal/exporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs/xfs"
)

func init() {
	exporter.Register(NewXFSCollector())
}

type XFSCollector struct {
	fs     xfs.FS
	logger *slog.Logger
}

func NewXFSCollector() *XFSCollector {
	logger := slog.Default()
	
	fs, err := xfs.NewFS("/proc", "/sys")
	if err != nil {
		logger.Debug("Failed to open XFS filesystem", "error", err)
		return &XFSCollector{
			logger: logger,
		}
	}

	return &XFSCollector{
		fs:     fs,
		logger: logger,
	}
}

func (c *XFSCollector) Collect(ch chan<- prometheus.Metric) {
	if c.fs == (xfs.FS{}) {
		c.logger.Debug("XFS filesystem not available")
		return
	}

	stats, err := c.fs.SysStats()
	if err != nil {
		c.logger.Debug("Failed to retrieve XFS stats", "error", err)
		return
	}

	for _, s := range stats {
		c.updateXFSStats(ch, s)
	}
}

func (c *XFSCollector) updateXFSStats(ch chan<- prometheus.Metric, s *xfs.Stats) {
	const subsystem = "xfs"
	labels := []string{"device"}

	// Metric names and descriptions are sourced from:
	// http://xfs.org/index.php/Runtime_Stats.
	metrics := []struct {
		name  string
		desc  string
		value float64
	}{
		{
			name:  "extent_allocation_extents_allocated_total",
			desc:  "Number of extents allocated for a filesystem.",
			value: float64(s.ExtentAllocation.ExtentsAllocated),
		},
		{
			name:  "extent_allocation_blocks_allocated_total",
			desc:  "Number of blocks allocated for a filesystem.",
			value: float64(s.ExtentAllocation.BlocksAllocated),
		},
		{
			name:  "extent_allocation_extents_freed_total",
			desc:  "Number of extents freed for a filesystem.",
			value: float64(s.ExtentAllocation.ExtentsFreed),
		},
		{
			name:  "extent_allocation_blocks_freed_total",
			desc:  "Number of blocks freed for a filesystem.",
			value: float64(s.ExtentAllocation.BlocksFreed),
		},
		{
			name:  "allocation_btree_lookups_total",
			desc:  "Number of allocation B-tree lookups for a filesystem.",
			value: float64(s.AllocationBTree.Lookups),
		},
		{
			name:  "allocation_btree_compares_total",
			desc:  "Number of allocation B-tree compares for a filesystem.",
			value: float64(s.AllocationBTree.Compares),
		},
		{
			name:  "allocation_btree_records_inserted_total",
			desc:  "Number of allocation B-tree records inserted for a filesystem.",
			value: float64(s.AllocationBTree.RecordsInserted),
		},
		{
			name:  "allocation_btree_records_deleted_total",
			desc:  "Number of allocation B-tree records deleted for a filesystem.",
			value: float64(s.AllocationBTree.RecordsDeleted),
		},
		{
			name:  "block_mapping_reads_total",
			desc:  "Number of block map for read operations for a filesystem.",
			value: float64(s.BlockMapping.Reads),
		},
		{
			name:  "block_mapping_writes_total",
			desc:  "Number of block map for write operations for a filesystem.",
			value: float64(s.BlockMapping.Writes),
		},
		{
			name:  "block_mapping_unmaps_total",
			desc:  "Number of block unmaps (deletes) for a filesystem.",
			value: float64(s.BlockMapping.Unmaps),
		},
		{
			name:  "block_mapping_extent_list_insertions_total",
			desc:  "Number of extent list insertions for a filesystem.",
			value: float64(s.BlockMapping.ExtentListInsertions),
		},
		{
			name:  "block_mapping_extent_list_deletions_total",
			desc:  "Number of extent list deletions for a filesystem.",
			value: float64(s.BlockMapping.ExtentListDeletions),
		},
		{
			name:  "block_mapping_extent_list_lookups_total",
			desc:  "Number of extent list lookups for a filesystem.",
			value: float64(s.BlockMapping.ExtentListLookups),
		},
		{
			name:  "block_mapping_extent_list_compares_total",
			desc:  "Number of extent list compares for a filesystem.",
			value: float64(s.BlockMapping.ExtentListCompares),
		},
		{
			name:  "block_map_btree_lookups_total",
			desc:  "Number of block map B-tree lookups for a filesystem.",
			value: float64(s.BlockMapBTree.Lookups),
		},
		{
			name:  "block_map_btree_compares_total",
			desc:  "Number of block map B-tree compares for a filesystem.",
			value: float64(s.BlockMapBTree.Compares),
		},
		{
			name:  "block_map_btree_records_inserted_total",
			desc:  "Number of block map B-tree records inserted for a filesystem.",
			value: float64(s.BlockMapBTree.RecordsInserted),
		},
		{
			name:  "block_map_btree_records_deleted_total",
			desc:  "Number of block map B-tree records deleted for a filesystem.",
			value: float64(s.BlockMapBTree.RecordsDeleted),
		},
		{
			name:  "directory_operation_lookup_total",
			desc:  "Number of file name directory lookups which miss the operating systems directory name lookup cache.",
			value: float64(s.DirectoryOperation.Lookups),
		},
		{
			name:  "directory_operation_create_total",
			desc:  "Number of times a new directory entry was created for a filesystem.",
			value: float64(s.DirectoryOperation.Creates),
		},
		{
			name:  "directory_operation_remove_total",
			desc:  "Number of times an existing directory entry was created for a filesystem.",
			value: float64(s.DirectoryOperation.Removes),
		},
		{
			name:  "directory_operation_getdents_total",
			desc:  "Number of times the directory getdents operation was performed for a filesystem.",
			value: float64(s.DirectoryOperation.Getdents),
		},
		{
			name:  "inode_operation_attempts_total",
			desc:  "Number of times the OS looked for an XFS inode in the inode cache.",
			value: float64(s.InodeOperation.Attempts),
		},
		{
			name:  "inode_operation_found_total",
			desc:  "Number of times the OS looked for and found an XFS inode in the inode cache.",
			value: float64(s.InodeOperation.Found),
		},
		{
			name:  "inode_operation_recycled_total",
			desc:  "Number of times the OS found an XFS inode in the cache, but could not use it as it was being recycled.",
			value: float64(s.InodeOperation.Recycle),
		},
		{
			name:  "inode_operation_missed_total",
			desc:  "Number of times the OS looked for an XFS inode in the cache, but did not find it.",
			value: float64(s.InodeOperation.Missed),
		},
		{
			name:  "inode_operation_duplicates_total",
			desc:  "Number of times the OS tried to add a missing XFS inode to the inode cache, but found it had already been added by another process.",
			value: float64(s.InodeOperation.Duplicate),
		},
		{
			name:  "inode_operation_reclaims_total",
			desc:  "Number of times the OS reclaimed an XFS inode from the inode cache to free memory for another purpose.",
			value: float64(s.InodeOperation.Reclaims),
		},
		{
			name:  "inode_operation_attribute_changes_total",
			desc:  "Number of times the OS explicitly changed the attributes of an XFS inode.",
			value: float64(s.InodeOperation.AttributeChange),
		},
		{
			name:  "read_calls_total",
			desc:  "Number of read(2) system calls made to files in a filesystem.",
			value: float64(s.ReadWrite.Read),
		},
		{
			name:  "write_calls_total",
			desc:  "Number of write(2) system calls made to files in a filesystem.",
			value: float64(s.ReadWrite.Write),
		},
		{
			name:  "vnode_active_total",
			desc:  "Number of vnodes not on free lists for a filesystem.",
			value: float64(s.Vnode.Active),
		},
		{
			name:  "vnode_allocate_total",
			desc:  "Number of times vn_alloc called for a filesystem.",
			value: float64(s.Vnode.Allocate),
		},
		{
			name:  "vnode_get_total",
			desc:  "Number of times vn_get called for a filesystem.",
			value: float64(s.Vnode.Get),
		},
		{
			name:  "vnode_hold_total",
			desc:  "Number of times vn_hold called for a filesystem.",
			value: float64(s.Vnode.Hold),
		},
		{
			name:  "vnode_release_total",
			desc:  "Number of times vn_rele called for a filesystem.",
			value: float64(s.Vnode.Release),
		},
		{
			name:  "vnode_reclaim_total",
			desc:  "Number of times vn_reclaim called for a filesystem.",
			value: float64(s.Vnode.Reclaim),
		},
		{
			name:  "vnode_remove_total",
			desc:  "Number of times vn_remove called for a filesystem.",
			value: float64(s.Vnode.Remove),
		},
	}

	for _, m := range metrics {
		desc := prometheus.NewDesc(
			prometheus.BuildFQName("node", subsystem, m.name),
			m.desc,
			labels,
			nil,
		)

		ch <- prometheus.MustNewConstMetric(
			desc,
			prometheus.CounterValue,
			m.value,
			s.Name,
		)
	}
}

func (c *XFSCollector) Describe(ch chan<- *prometheus.Desc) {
	// XFS collector creates dynamic descriptors
} 
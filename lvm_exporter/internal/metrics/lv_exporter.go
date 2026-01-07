package metrics

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"
)

type LvExporter struct {
	cache_dirty_blocks          *prometheus.Desc
	cache_read_hits             *prometheus.Desc
	cache_read_misses           *prometheus.Desc
	cache_total_blocks          *prometheus.Desc
	cache_used_blocks           *prometheus.Desc
	cache_write_hits            *prometheus.Desc
	cache_write_misses          *prometheus.Desc
	copy_percent                *prometheus.Desc
	data_percent                *prometheus.Desc
	integritymismatches         *prometheus.Desc
	lv_active_exclusively       *prometheus.Desc
	lv_active_locally           *prometheus.Desc
	lv_active_remotely          *prometheus.Desc
	lv_allocation_locked        *prometheus.Desc
	lv_check_needed             *prometheus.Desc
	lv_converting               *prometheus.Desc
	lv_device_open              *prometheus.Desc
	lv_fixed_minor              *prometheus.Desc
	lv_historical               *prometheus.Desc
	lv_image_synced             *prometheus.Desc
	lv_inactive_table           *prometheus.Desc
	lv_initial_image_sync       *prometheus.Desc
	lv_kernel_major             *prometheus.Desc
	lv_kernel_minor             *prometheus.Desc
	lv_live_table               *prometheus.Desc
	lv_major                    *prometheus.Desc
	lv_merge_failed             *prometheus.Desc
	lv_merging                  *prometheus.Desc
	lv_metadata_size            *prometheus.Desc
	lv_minor                    *prometheus.Desc
	lv_read_ahead               *prometheus.Desc
	lv_size                     *prometheus.Desc
	lv_skip_activation          *prometheus.Desc
	lv_snapshot_invalid         *prometheus.Desc
	lv_suspended                *prometheus.Desc
	lv_time                     *prometheus.Desc
	lv_time_removed             *prometheus.Desc
	metadata_percent            *prometheus.Desc
	origin_size                 *prometheus.Desc
	raid_max_recovery_rate      *prometheus.Desc
	raid_min_recovery_rate      *prometheus.Desc
	raid_mismatch_count         *prometheus.Desc
	raid_write_behind           *prometheus.Desc
	raidintegrityblocksize      *prometheus.Desc
	seg_count                   *prometheus.Desc
	snap_percent                *prometheus.Desc
	sync_percent                *prometheus.Desc
	vdo_saving_percent          *prometheus.Desc
	vdo_used_size               *prometheus.Desc
	writecache_error            *prometheus.Desc
	writecache_free_blocks      *prometheus.Desc
	writecache_total_blocks     *prometheus.Desc
	writecache_writeback_blocks *prometheus.Desc
}

var lvnamespace = "lv"

func NewLvExporter() *LvExporter {
	cache_dirty_blocks := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_cache_dirty_blocks"),
		"Dirty cache blocks", []string{"lv_uuid"}, nil,
	)

	cache_read_hits := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_cache_read_hits"),
		"Cache read hits", []string{"lv_uuid"}, nil,
	)

	cache_read_misses := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_cache_read_misses"),
		"Cache read misses", []string{"lv_uuid"}, nil,
	)

	cache_total_blocks := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_cache_total_blocks"),
		"Total cache blocks", []string{"lv_uuid"}, nil,
	)

	cache_used_blocks := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_cache_used_blocks"),
		"Used cache blocks", []string{"lv_uuid"}, nil,
	)

	cache_write_hits := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_cache_write_hits"),
		"Cache write hits", []string{"lv_uuid"}, nil,
	)

	cache_write_misses := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_cache_write_misses"),
		"Cache write misses", []string{"lv_uuid"}, nil,
	)

	copy_percent := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_copy_percent"),
		"For Cache, RAID, mirrors and pvmove, current percentage in-sync", []string{"lv_uuid"}, nil,
	)

	data_percent := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_data_percent"),
		"For snapshot, cache and thin pools and volumes, the percentage full if LV is active", []string{"lv_uuid"}, nil,
	)

	integritymismatches := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_integritymismatches"),
		"The number of integrity mismatches", []string{"lv_uuid"}, nil,
	)

	lv_active_exclusively := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_active_exclusively"),
		"Set if the LV is active exclusively", []string{"lv_uuid"}, nil,
	)

	lv_active_locally := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_active_locally"),
		"Set if the LV is active locally", []string{"lv_uuid"}, nil,
	)

	lv_active_remotely := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_active_remotely"),
		"Set if the LV is active remotely", []string{"lv_uuid"}, nil,
	)

	lv_allocation_locked := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_allocation_locked"),
		"Set if LV is locked against allocation changes", []string{"lv_uuid"}, nil,
	)

	lv_check_needed := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_check_needed"),
		"For thin pools and cache volumes, whether metadata check is needed", []string{"lv_uuid"}, nil,
	)

	lv_converting := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_converting"),
		"Set if LV is being converted", []string{"lv_uuid"}, nil,
	)

	lv_device_open := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_device_open"),
		"Set if LV device is open", []string{"lv_uuid"}, nil,
	)

	lv_fixed_minor := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_fixed_minor"),
		"Set if LV has fixed minor number assigned", []string{"lv_uuid"}, nil,
	)

	lv_historical := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_historical"),
		"Set if the LV is historical", []string{"lv_uuid"}, nil,
	)

	lv_image_synced := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_image_synced"),
		"Set if mirror/RAID image is synchronized", []string{"lv_uuid"}, nil,
	)

	lv_inactive_table := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_inactive_table"),
		"Set if LV has inactive table present", []string{"lv_uuid"}, nil,
	)

	lv_initial_image_sync := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_initial_image_sync"),
		"Set if mirror/RAID images underwent initial resynchronization", []string{"lv_uuid"}, nil,
	)

	lv_kernel_major := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_kernel_major"),
		"Currently assigned major number or -1 if LV is not active", []string{"lv_uuid"}, nil,
	)

	lv_kernel_minor := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_kernel_minor"),
		"Currently assigned minor number or -1 if LV is not active", []string{"lv_uuid"}, nil,
	)

	lv_live_table := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_live_table"),
		"Set if LV has live table present", []string{"lv_uuid"}, nil,
	)

	lv_major := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_major"),
		"Persistent major number or -1 if not persistent", []string{"lv_uuid"}, nil,
	)

	lv_merge_failed := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_merge_failed"),
		"Set if snapshot merge failed", []string{"lv_uuid"}, nil,
	)

	lv_merging := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_merging"),
		"Set if snapshot LV is being merged to origin", []string{"lv_uuid"}, nil,
	)

	lv_metadata_size := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_metadata_size_bytes"),
		"For thin and cache pools, the size of the LV that holds the metadata", []string{"lv_uuid"}, nil,
	)

	lv_minor := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_minor"),
		"Persistent minor number or -1 if not persistent", []string{"lv_uuid"}, nil,
	)

	lv_read_ahead := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_read_ahead_bytes"),
		"Read ahead setting", []string{"lv_uuid"}, nil,
	)

	lv_size := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_size_bytes"),
		"Size of LV", []string{"lv_uuid"}, nil,
	)

	lv_skip_activation := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_skip_activation"),
		"Set if LV is skipped on activation", []string{"lv_uuid"}, nil,
	)

	lv_snapshot_invalid := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_snapshot_invalid"),
		"Set if snapshot LV is invalid", []string{"lv_uuid"}, nil,
	)

	lv_suspended := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_suspended"),
		"Set if LV is suspended", []string{"lv_uuid"}, nil,
	)

	lv_time := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_time"),
		"Creation time of the LV, if known", []string{"lv_uuid"}, nil,
	)

	metadata_percent := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_metadata_percent"),
		"For cache and thin pools, the percentage of metadata full if LV is active", []string{"lv_uuid"}, nil,
	)

	lv_time_removed := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_time_removed"),
		"Set if LV is suspended", []string{"lv_uuid"}, nil,
	)

	origin_size := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_origin_size_bytes"),
		"For snapshots, the size of the origin device of this LV", []string{"lv_uuid"}, nil,
	)

	raid_max_recovery_rate := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_raid_max_recovery_rate"),
		"For RAID1, the maximum recovery I/O load in kiB/sec/disk", []string{"lv_uuid"}, nil,
	)

	raid_min_recovery_rate := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_raid_min_recovery_rate"),
		"For RAID1, the minimum recovery I/O load in kiB/sec/disk", []string{"lv_uuid"}, nil,
	)

	raid_mismatch_count := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_raid_mismatch_count"),
		"For RAID, number of mismatches found or repaired", []string{"lv_uuid"}, nil,
	)

	raid_write_behind := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_raid_write_behind"),
		"For RAID1, the number of outstanding writes allowed to writemostly devices", []string{"lv_uuid"}, nil,
	)

	raidintegrityblocksize := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_raidintegrityblocksize"),
		"The integrity block size", []string{"lv_uuid"}, nil,
	)

	seg_count := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_seg_count"),
		"Number of segments in LV", []string{"lv_uuid"}, nil,
	)

	snap_percent := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_snap_percent"),
		"For snapshots, the percentage full if LV is active", []string{"lv_uuid"}, nil,
	)

	sync_percent := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_sync_percent"),
		"For Cache, RAID, mirrors and pvmove, current percentage in-sync", []string{"lv_uuid"}, nil,
	)

	vdo_saving_percent := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_vdo_saving_percent"),
		"For vdo pools, percentage of saved space", []string{"lv_uuid"}, nil,
	)

	vdo_used_size := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_vdo_used_size_bytes"),
		"For vdo pools, currently used space", []string{"lv_uuid"}, nil,
	)

	writecache_error := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_writecache_error"),
		"Total writecache errors", []string{"lv_uuid"}, nil,
	)

	writecache_free_blocks := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_writecache_free_blocks"),
		"Total writecache free blocks", []string{"lv_uuid"}, nil,
	)

	writecache_total_blocks := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_writecache_total_blocks"),
		"Total writecache blocks", []string{"lv_uuid"}, nil,
	)

	writecache_writeback_blocks := prometheus.NewDesc(
		prometheus.BuildFQName(lvnamespace, "", "lv_writecache_writeback_blocks"),
		"Total writecache writeback blocks", []string{"lv_uuid"}, nil,
	)

	return &LvExporter{
		cache_dirty_blocks:          cache_dirty_blocks,
		cache_read_hits:             cache_read_hits,
		cache_read_misses:           cache_read_misses,
		cache_total_blocks:          cache_total_blocks,
		cache_used_blocks:           cache_used_blocks,
		cache_write_hits:            cache_write_hits,
		cache_write_misses:          cache_write_misses,
		copy_percent:                copy_percent,
		data_percent:                data_percent,
		integritymismatches:         integritymismatches,
		lv_active_exclusively:       lv_active_exclusively,
		lv_active_locally:           lv_active_locally,
		lv_active_remotely:          lv_active_remotely,
		lv_allocation_locked:        lv_allocation_locked,
		lv_check_needed:             lv_check_needed,
		lv_converting:               lv_converting,
		lv_device_open:              lv_device_open,
		lv_fixed_minor:              lv_fixed_minor,
		lv_historical:               lv_historical,
		lv_image_synced:             lv_image_synced,
		lv_inactive_table:           lv_inactive_table,
		lv_initial_image_sync:       lv_initial_image_sync,
		lv_kernel_major:             lv_kernel_major,
		lv_kernel_minor:             lv_kernel_minor,
		lv_live_table:               lv_live_table,
		lv_major:                    lv_major,
		lv_merge_failed:             lv_merge_failed,
		lv_merging:                  lv_merging,
		lv_metadata_size:            lv_metadata_size,
		lv_minor:                    lv_minor,
		lv_read_ahead:               lv_read_ahead,
		lv_size:                     lv_size,
		lv_skip_activation:          lv_skip_activation,
		lv_snapshot_invalid:         lv_snapshot_invalid,
		lv_suspended:                lv_suspended,
		lv_time:                     lv_time,
		lv_time_removed:             lv_time_removed,
		metadata_percent:            metadata_percent,
		origin_size:                 origin_size,
		raid_max_recovery_rate:      raid_max_recovery_rate,
		raid_min_recovery_rate:      raid_min_recovery_rate,
		raid_mismatch_count:         raid_mismatch_count,
		raid_write_behind:           raid_write_behind,
		raidintegrityblocksize:      raidintegrityblocksize,
		seg_count:                   seg_count,
		snap_percent:                snap_percent,
		sync_percent:                sync_percent,
		vdo_saving_percent:          vdo_saving_percent,
		vdo_used_size:               vdo_used_size,
		writecache_error:            writecache_error,
		writecache_free_blocks:      writecache_free_blocks,
		writecache_total_blocks:     writecache_total_blocks,
		writecache_writeback_blocks: writecache_writeback_blocks,
	}
}

func (e *LvExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.cache_dirty_blocks
	ch <- e.cache_read_hits
	ch <- e.cache_read_misses
	ch <- e.cache_total_blocks
	ch <- e.cache_used_blocks
	ch <- e.cache_write_hits
	ch <- e.cache_write_misses
	ch <- e.copy_percent
	ch <- e.data_percent
	ch <- e.integritymismatches
	ch <- e.lv_active_exclusively
	ch <- e.lv_active_locally
	ch <- e.lv_active_remotely
	ch <- e.lv_allocation_locked
	ch <- e.lv_check_needed
	ch <- e.lv_converting
	ch <- e.lv_device_open
	ch <- e.lv_fixed_minor
	ch <- e.lv_historical
	ch <- e.lv_image_synced
	ch <- e.lv_inactive_table
	ch <- e.lv_initial_image_sync
	ch <- e.lv_kernel_major
	ch <- e.lv_kernel_minor
	ch <- e.lv_live_table
	ch <- e.lv_major
	ch <- e.lv_merge_failed
	ch <- e.lv_merging
	ch <- e.lv_metadata_size
	ch <- e.lv_minor
	ch <- e.lv_read_ahead
	ch <- e.lv_size
	ch <- e.lv_skip_activation
	ch <- e.lv_snapshot_invalid
	ch <- e.lv_suspended
	ch <- e.lv_time
	ch <- e.lv_time_removed
	ch <- e.metadata_percent
	ch <- e.origin_size
	ch <- e.raid_max_recovery_rate
	ch <- e.raid_min_recovery_rate
	ch <- e.raid_mismatch_count
	ch <- e.raid_write_behind
	ch <- e.raidintegrityblocksize
	ch <- e.seg_count
	ch <- e.snap_percent
	ch <- e.sync_percent
	ch <- e.vdo_saving_percent
	ch <- e.vdo_used_size
	ch <- e.writecache_error
	ch <- e.writecache_free_blocks
	ch <- e.writecache_total_blocks
	ch <- e.writecache_writeback_blocks
}

func (e *LvExporter) Collect(ch chan<- prometheus.Metric) {
	log.Println("run here Collect")

	e.LvCollect(ch)
}

func (e *LvExporter) LvCollect(ch chan<- prometheus.Metric) {
	report, err := GetLvmReport()
	if err != nil {
		log.Println("Error get JSON:", err)
	}
	lvs, err := GetLvInfo(report)
	if err != nil {
		log.Println("Error get PvInfo:", err)
	}
	for _, lv := range lvs {
		cache_dirty_blocks := parseString(lv.Cache_dirty_blocks)
		ch <- prometheus.MustNewConstMetric(
			e.cache_dirty_blocks, prometheus.GaugeValue, cache_dirty_blocks, lv.Lv_uuid)

		cache_read_hits := parseString(lv.Cache_read_hits)
		ch <- prometheus.MustNewConstMetric(
			e.cache_read_hits, prometheus.GaugeValue, cache_read_hits, lv.Lv_uuid)

		cache_read_misses := parseString(lv.Cache_read_misses)
		ch <- prometheus.MustNewConstMetric(
			e.cache_read_misses, prometheus.GaugeValue, cache_read_misses, lv.Lv_uuid)

		cache_total_blocks := parseString(lv.Cache_total_blocks)
		ch <- prometheus.MustNewConstMetric(
			e.cache_total_blocks, prometheus.GaugeValue, cache_total_blocks, lv.Lv_uuid)

		cache_used_blocks := parseString(lv.Cache_used_blocks)
		ch <- prometheus.MustNewConstMetric(
			e.cache_used_blocks, prometheus.GaugeValue, cache_used_blocks, lv.Lv_uuid)

		cache_write_hits := parseString(lv.Cache_write_hits)
		ch <- prometheus.MustNewConstMetric(
			e.cache_write_hits, prometheus.GaugeValue, cache_write_hits, lv.Lv_uuid)

		cache_write_misses := parseString(lv.Cache_write_misses)
		ch <- prometheus.MustNewConstMetric(
			e.cache_write_misses, prometheus.GaugeValue, cache_write_misses, lv.Lv_uuid)

		copy_percent := parseString(lv.Copy_percent)
		ch <- prometheus.MustNewConstMetric(
			e.copy_percent, prometheus.GaugeValue, copy_percent, lv.Lv_uuid)

		data_percent := parseString(lv.Data_percent)
		ch <- prometheus.MustNewConstMetric(
			e.data_percent, prometheus.GaugeValue, data_percent, lv.Lv_uuid)

		integritymismatches := parseString(lv.Integritymismatches)
		ch <- prometheus.MustNewConstMetric(
			e.integritymismatches, prometheus.GaugeValue, integritymismatches, lv.Lv_uuid)

		lv_active_exclusively := 0.0
		switch lv.Lv_active_exclusively {
		case "active exclusively":
			lv_active_exclusively = 1.0
		case "":
			lv_active_exclusively = 0.0
		default:
			lv_active_exclusively = 1.0
		}
		ch <- prometheus.MustNewConstMetric(
			e.lv_active_exclusively, prometheus.GaugeValue, lv_active_exclusively, lv.Lv_uuid)

		lv_active_locally := 0.0
		switch lv.Lv_active_locally {
		case "active locally":
			lv_active_locally = 1.0
		case "":
			lv_active_locally = 0.0
		default:
			lv_active_locally = 1.0
		}
		ch <- prometheus.MustNewConstMetric(
			e.lv_active_locally, prometheus.GaugeValue, lv_active_locally, lv.Lv_uuid)

		lv_active_remotely := 0.0
		switch lv.Lv_active_remotely {
		case "active remotely":
			lv_active_remotely = 1.0
		case "":
			lv_active_remotely = 0.0
		default:
			lv_active_remotely = 1.0
		}
		ch <- prometheus.MustNewConstMetric(
			e.lv_active_remotely, prometheus.GaugeValue, lv_active_remotely, lv.Lv_uuid)

		lv_allocation_locked := parseString(lv.Lv_allocation_locked)
		ch <- prometheus.MustNewConstMetric(
			e.lv_allocation_locked, prometheus.GaugeValue, lv_allocation_locked, lv.Lv_uuid)

		lv_check_needed := 0.0
		switch lv.Lv_check_needed {
		case "unknown":
			lv_check_needed = 0.0
		case "":
			lv_check_needed = 1.0
		default:
			lv_check_needed = 0.0
		}
		ch <- prometheus.MustNewConstMetric(
			e.lv_check_needed, prometheus.GaugeValue, lv_check_needed, lv.Lv_uuid)

		lv_converting := parseString(lv.Lv_converting)
		ch <- prometheus.MustNewConstMetric(
			e.lv_converting, prometheus.GaugeValue, lv_converting, lv.Lv_uuid)

		lv_device_open := 0.0
		switch lv.Lv_device_open {
		case "open":
			lv_check_needed = 1.0
		case "":
			lv_check_needed = 0.0
		default:
			lv_check_needed = 0.0
		}
		ch <- prometheus.MustNewConstMetric(
			e.lv_device_open, prometheus.GaugeValue, lv_device_open, lv.Lv_uuid)

		lv_fixed_minor := parseString(lv.Lv_fixed_minor)
		ch <- prometheus.MustNewConstMetric(
			e.lv_fixed_minor, prometheus.GaugeValue, lv_fixed_minor, lv.Lv_uuid)

		lv_historical := parseString(lv.Lv_historical)
		ch <- prometheus.MustNewConstMetric(
			e.lv_historical, prometheus.GaugeValue, lv_historical, lv.Lv_uuid)

		lv_image_synced := parseString(lv.Lv_image_synced)
		ch <- prometheus.MustNewConstMetric(
			e.lv_image_synced, prometheus.GaugeValue, lv_image_synced, lv.Lv_uuid)

		lv_inactive_table := parseString(lv.Lv_inactive_table)
		ch <- prometheus.MustNewConstMetric(
			e.lv_inactive_table, prometheus.GaugeValue, lv_inactive_table, lv.Lv_uuid)

		lv_initial_image_sync := parseString(lv.Lv_initial_image_sync)
		ch <- prometheus.MustNewConstMetric(
			e.lv_initial_image_sync, prometheus.GaugeValue, lv_initial_image_sync, lv.Lv_uuid)

		lv_kernel_major := parseString(lv.Lv_kernel_major)
		ch <- prometheus.MustNewConstMetric(
			e.lv_kernel_major, prometheus.GaugeValue, lv_kernel_major, lv.Lv_uuid)

		lv_kernel_minor := parseString(lv.Lv_kernel_minor)
		ch <- prometheus.MustNewConstMetric(
			e.lv_kernel_minor, prometheus.GaugeValue, lv_kernel_minor, lv.Lv_uuid)

		lv_live_table := 0.0
		switch lv.Lv_live_table {
		case "live table present":
			lv_live_table = 1.0
		case "":
			lv_live_table = 0.0
		default:
			lv_live_table = 0.0
		}
		ch <- prometheus.MustNewConstMetric(
			e.lv_live_table, prometheus.GaugeValue, lv_live_table, lv.Lv_uuid)

		lv_major := parseString(lv.Lv_major)
		ch <- prometheus.MustNewConstMetric(
			e.lv_major, prometheus.GaugeValue, lv_major, lv.Lv_uuid)

		lv_merge_failed := 0.0
		switch lv.Lv_merge_failed {
		case "unknown":
			lv_merge_failed = 0.0
		case "":
			lv_merge_failed = 0.0
		default:
			lv_merge_failed = 0.0
		}
		ch <- prometheus.MustNewConstMetric(
			e.lv_merge_failed, prometheus.GaugeValue, lv_merge_failed, lv.Lv_uuid)

		lv_merging := parseString(lv.Lv_merging)
		ch <- prometheus.MustNewConstMetric(
			e.lv_merging, prometheus.GaugeValue, lv_merging, lv.Lv_uuid)

		lv_metadata_size := parseString(lv.Lv_metadata_size)
		ch <- prometheus.MustNewConstMetric(
			e.lv_metadata_size, prometheus.GaugeValue, lv_metadata_size, lv.Lv_uuid)

		lv_minor := parseString(lv.Lv_minor)
		ch <- prometheus.MustNewConstMetric(
			e.lv_minor, prometheus.GaugeValue, lv_minor, lv.Lv_uuid)

		lv_read_ahead := 0.0
		switch lv.Lv_read_ahead {
		case "auto":
			lv_read_ahead = -1.0
		case "":
			lv_read_ahead = 0.0
		default:
			lv_read_ahead = parseString(lv.Lv_merge_failed)
		}
		ch <- prometheus.MustNewConstMetric(
			e.lv_read_ahead, prometheus.GaugeValue, lv_read_ahead, lv.Lv_uuid)

		lv_size := parseSize(lv.Lv_size)
		ch <- prometheus.MustNewConstMetric(
			e.lv_size, prometheus.GaugeValue, lv_size, lv.Lv_uuid)

		lv_skip_activation := 0.0
		switch lv.Lv_skip_activation {
		case "skip":
			lv_skip_activation = -1.0
		case "":
			lv_skip_activation = 0.0
		default:
			lv_skip_activation = 0.0
		}
		ch <- prometheus.MustNewConstMetric(
			e.lv_skip_activation, prometheus.GaugeValue, lv_skip_activation, lv.Lv_uuid)

		lv_snapshot_invalid := 0.0
		switch lv.Lv_snapshot_invalid {
		case "unknown":
			lv_snapshot_invalid = 0.0
		case "":
			lv_snapshot_invalid = 1.0
		default:
			lv_snapshot_invalid = 0.0
		}
		ch <- prometheus.MustNewConstMetric(
			e.lv_snapshot_invalid, prometheus.GaugeValue, lv_snapshot_invalid, lv.Lv_uuid)

		lv_suspended := parseString(lv.Lv_suspended)
		ch <- prometheus.MustNewConstMetric(
			e.lv_suspended, prometheus.GaugeValue, lv_suspended, lv.Lv_uuid)

		lv_time := parseTimeStamp(lv.Lv_time)
		ch <- prometheus.MustNewConstMetric(
			e.lv_time, prometheus.GaugeValue, lv_time, lv.Lv_uuid)

		lv_time_removed := parseTimeStamp(lv.Lv_time_removed)
		ch <- prometheus.MustNewConstMetric(
			e.lv_time_removed, prometheus.GaugeValue, lv_time_removed, lv.Lv_uuid)

		metadata_percent := parseString(lv.Metadata_percent)
		ch <- prometheus.MustNewConstMetric(
			e.metadata_percent, prometheus.GaugeValue, metadata_percent, lv.Lv_uuid)

		origin_size := parseString(lv.Origin_size)
		ch <- prometheus.MustNewConstMetric(
			e.origin_size, prometheus.GaugeValue, origin_size, lv.Lv_uuid)

		raid_max_recovery_rate := parseString(lv.Raid_max_recovery_rate)
		ch <- prometheus.MustNewConstMetric(
			e.raid_max_recovery_rate, prometheus.GaugeValue, raid_max_recovery_rate, lv.Lv_uuid)

		raid_min_recovery_rate := parseString(lv.Raid_min_recovery_rate)
		ch <- prometheus.MustNewConstMetric(
			e.raid_min_recovery_rate, prometheus.GaugeValue, raid_min_recovery_rate, lv.Lv_uuid)

		raid_mismatch_count := parseString(lv.Raid_mismatch_count)
		ch <- prometheus.MustNewConstMetric(
			e.raid_mismatch_count, prometheus.GaugeValue, raid_mismatch_count, lv.Lv_uuid)

		raid_write_behind := parseString(lv.Raid_write_behind)
		ch <- prometheus.MustNewConstMetric(
			e.raid_write_behind, prometheus.GaugeValue, raid_write_behind, lv.Lv_uuid)

		raidintegrityblocksize := parseString(lv.Raidintegrityblocksize)
		ch <- prometheus.MustNewConstMetric(
			e.raidintegrityblocksize, prometheus.GaugeValue, raidintegrityblocksize, lv.Lv_uuid)

		seg_count := parseString(lv.Seg_count)
		ch <- prometheus.MustNewConstMetric(
			e.seg_count, prometheus.GaugeValue, seg_count, lv.Lv_uuid)

		snap_percent := parseString(lv.Snap_percent)
		ch <- prometheus.MustNewConstMetric(
			e.snap_percent, prometheus.GaugeValue, snap_percent, lv.Lv_uuid)

		sync_percent := parseString(lv.Sync_percent)
		ch <- prometheus.MustNewConstMetric(
			e.sync_percent, prometheus.GaugeValue, sync_percent, lv.Lv_uuid)

		// ch <- e.vdo_saving_percent
		// ch <- e.vdo_used_size
		vdo_saving_percent := parseString(lv.Vdo_saving_percent)
		ch <- prometheus.MustNewConstMetric(
			e.vdo_saving_percent, prometheus.GaugeValue, vdo_saving_percent, lv.Lv_uuid)

		vdo_used_size := parseString(lv.Vdo_used_size)
		ch <- prometheus.MustNewConstMetric(
			e.vdo_used_size, prometheus.GaugeValue, vdo_used_size, lv.Lv_uuid)

		writecache_error := parseString(lv.Writecache_error)
		ch <- prometheus.MustNewConstMetric(
			e.writecache_error, prometheus.GaugeValue, writecache_error, lv.Lv_uuid)

		writecache_free_blocks := parseString(lv.Writecache_free_blocks)
		ch <- prometheus.MustNewConstMetric(
			e.writecache_free_blocks, prometheus.GaugeValue, writecache_free_blocks, lv.Lv_uuid)

		writecache_total_blocks := parseString(lv.Writecache_total_blocks)
		ch <- prometheus.MustNewConstMetric(
			e.writecache_total_blocks, prometheus.GaugeValue, writecache_total_blocks, lv.Lv_uuid)

		writecache_writeback_blocks := parseString(lv.Writecache_writeback_blocks)
		ch <- prometheus.MustNewConstMetric(
			e.writecache_writeback_blocks, prometheus.GaugeValue, writecache_writeback_blocks, lv.Lv_uuid)
	}

}

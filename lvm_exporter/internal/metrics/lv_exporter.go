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


// TODO: implement functions

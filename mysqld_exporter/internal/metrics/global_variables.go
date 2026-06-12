package metrics

import (
	"database/sql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"mysqld_exporter/internal/exporter"
	"mysqld_exporter/internal/mysql"
	"regexp"
	"strconv"
	"strings"
)

const (
	globalVariables       = "global_variables"
	globalVariablesQuery  = `SHOW GLOBAL VARIABLES`
	globalVariablesResult = `
*************************** 598. row ***************************
Variable_name: sync_relay_log_info
        Value: 10000
*************************** 599. row ***************************
Variable_name: sync_source_info
        Value: 10000
*************************** 600. row ***************************
Variable_name: system_time_zone
        Value: CST
*************************** 601. row ***************************
Variable_name: table_definition_cache
        Value: 2000
*************************** 602. row ***************************
Variable_name: table_encryption_privilege_check
        Value: OFF
*************************** 603. row ***************************
Variable_name: table_open_cache
        Value: 4000
*************************** 604. row ***************************
Variable_name: table_open_cache_instances
        Value: 16
*************************** 605. row ***************************
Variable_name: tablespace_definition_cache
        Value: 256
*************************** 606. row ***************************
Variable_name: temptable_max_mmap
        Value: 1073741824
*************************** 607. row ***************************
Variable_name: temptable_max_ram
        Value: 1073741824
*************************** 608. row ***************************
Variable_name: temptable_use_mmap
        Value: ON
*************************** 609. row ***************************
Variable_name: terminology_use_previous
        Value: NONE
*************************** 610. row ***************************
Variable_name: thread_cache_size
        Value: 9
*************************** 611. row ***************************
Variable_name: thread_handling
        Value: one-thread-per-connection
*************************** 612. row ***************************
Variable_name: thread_stack
        Value: 1048576
*************************** 613. row ***************************
Variable_name: time_zone
        Value: SYSTEM
*************************** 614. row ***************************
Variable_name: tls_ciphersuites
        Value: 
*************************** 615. row ***************************
Variable_name: tls_version
        Value: TLSv1.2,TLSv1.3
*************************** 616. row ***************************
Variable_name: tmp_table_size
        Value: 16777216
*************************** 617. row ***************************
Variable_name: tmpdir
        Value: /var/tmp
*************************** 618. row ***************************
Variable_name: transaction_alloc_block_size
        Value: 8192
*************************** 619. row ***************************
Variable_name: transaction_isolation
        Value: REPEATABLE-READ
*************************** 620. row ***************************
Variable_name: transaction_prealloc_size
        Value: 4096
*************************** 621. row ***************************
Variable_name: transaction_read_only
        Value: OFF
*************************** 622. row ***************************
Variable_name: transaction_write_set_extraction
        Value: XXHASH64
*************************** 623. row ***************************
Variable_name: unique_checks
        Value: ON
*************************** 624. row ***************************
Variable_name: updatable_views_with_limit
        Value: YES
*************************** 625. row ***************************
Variable_name: version
        Value: 8.0.40
*************************** 626. row ***************************
Variable_name: version_comment
        Value: Source distribution
*************************** 627. row ***************************
Variable_name: version_compile_machine
        Value: x86_64
*************************** 628. row ***************************
Variable_name: version_compile_os
        Value: Linux
*************************** 629. row ***************************
Variable_name: version_compile_zlib
        Value: 1.3.1
*************************** 630. row ***************************
Variable_name: wait_timeout
        Value: 28800
*************************** 631. row ***************************
Variable_name: windowing_use_high_precision
        Value: ON
*************************** 632. row ***************************
Variable_name: xa_detach_on_prepare
        Value: ON
`
)

type ScrapeGlobalVariables struct {
	instance mysql.Instance
	rocksdb_access_hint_on_compaction_start
	rocksdb_advise_random_on_open
	rocksdb_allow_concurrent_memtable_write
	rocksdb_allow_mmap_reads
	rocksdb_allow_mmap_writes
	rocksdb_block_cache_size
	rocksdb_block_restart_interval
	rocksdb_block_size_deviation
	rocksdb_block_size
	rocksdb_bulk_load_size
	rocksdb_bulk_load
	rocksdb_bytes_per_sync
	rocksdb_cache_index_and_filter_blocks
	rocksdb_checksums_pct
	rocksdb_collect_sst_properties
	rocksdb_commit_in_the_middle
	rocksdb_compaction_readahead_size
	rocksdb_compaction_sequential_deletes_count_sd
	rocksdb_compaction_sequential_deletes_file_size
	rocksdb_compaction_sequential_deletes_window
	rocksdb_compaction_sequential_deletes
	rocksdb_create_if_missing
	rocksdb_create_missing_column_families
	rocksdb_db_write_buffer_size
	rocksdb_deadlock_detect
	rocksdb_debug_optimizer_no_zero_cardinality
	rocksdb_delayed_write_rate
	rocksdb_delete_obsolete_files_period_micros
	rocksdb_enable_bulk_load_api
	rocksdb_enable_thread_tracking
	rocksdb_enable_write_thread_adaptive_yield
	rocksdb_error_if_exists
	rocksdb_flush_log_at_trx_commit
	rocksdb_flush_memtable_on_analyze
	rocksdb_force_compute_memtable_stats
	rocksdb_force_flush_memtable_now
	rocksdb_force_index_records_in_range
	rocksdb_hash_index_allow_collision
	rocksdb_keep_log_file_num
	rocksdb_lock_scanned_rows
	rocksdb_lock_wait_timeout
	rocksdb_log_file_time_to_roll
	rocksdb_manifest_preallocation_size
	rocksdb_max_open_files
	rocksdb_max_row_locks
	rocksdb_max_subcompactions
	rocksdb_max_total_wal_size
	rocksdb_merge_buf_size
	rocksdb_merge_combine_read_size
	rocksdb_new_table_reader_for_compaction_inputs
	rocksdb_no_block_cache
	rocksdb_paranoid_checks
	rocksdb_pause_background_work
	rocksdb_perf_context_level
	rocksdb_persistent_cache_size_mb
	rocksdb_pin_l0_filter_and_index_blocks_in_cache
	rocksdb_print_snapshot_conflict_queries
	rocksdb_rate_limiter_bytes_per_sec
	rocksdb_records_in_range
	rocksdb_seconds_between_stat_computes
	rocksdb_signal_drop_index_thread
	rocksdb_skip_bloom_filter_on_read
	rocksdb_skip_fill_cache
	rocksdb_stats_dump_period_sec
	rocksdb_store_row_debug_checksums
	rocksdb_strict_collation_check
	rocksdb_table_cache_numshardbits
	rocksdb_use_adaptive_mutex
	rocksdb_use_direct_reads
	rocksdb_use_fsync
	rocksdb_validate_tables
	rocksdb_verify_row_debug_checksums
	rocksdb_wal_bytes_per_sync
	rocksdb_wal_recovery_mode
	rocksdb_wal_size_limit_mb
	rocksdb_wal_ttl_seconds
	rocksdb_whole_key_filtering
	rocksdb_write_disable_wal
	rocksdb_write_ignore_missing_column_families
}

func init() {
	exporter.Register(
		NewScrapeGlobalVariables())
}
func NewScrapeGlobalVariables() *ScrapeGlobalVariables {
	return &ScrapeGlobalVariables{
		//instance:                                        instance,
		rocksdb_access_hint_on_compaction_start:         *Newrocksdb_access_hint_on_compaction_start(),
		rocksdb_advise_random_on_open:                   *Newrocksdb_advise_random_on_open(),
		rocksdb_allow_concurrent_memtable_write:         *Newrocksdb_allow_concurrent_memtable_write(),
		rocksdb_allow_mmap_reads:                        *Newrocksdb_allow_mmap_reads(),
		rocksdb_allow_mmap_writes:                       *Newrocksdb_allow_mmap_writes(),
		rocksdb_block_cache_size:                        *Newrocksdb_block_cache_size(),
		rocksdb_block_restart_interval:                  *Newrocksdb_block_restart_interval(),
		rocksdb_block_size_deviation:                    *Newrocksdb_block_size_deviation(),
		rocksdb_block_size:                              *Newrocksdb_block_size(),
		rocksdb_bulk_load_size:                          *Newrocksdb_bulk_load_size(),
		rocksdb_bulk_load:                               *Newrocksdb_bulk_load(),
		rocksdb_bytes_per_sync:                          *Newrocksdb_bytes_per_sync(),
		rocksdb_cache_index_and_filter_blocks:           *Newrocksdb_cache_index_and_filter_blocks(),
		rocksdb_checksums_pct:                           *Newrocksdb_checksums_pct(),
		rocksdb_collect_sst_properties:                  *Newrocksdb_collect_sst_properties(),
		rocksdb_commit_in_the_middle:                    *Newrocksdb_commit_in_the_middle(),
		rocksdb_compaction_readahead_size:               *Newrocksdb_compaction_readahead_size(),
		rocksdb_compaction_sequential_deletes_count_sd:  *Newrocksdb_compaction_sequential_deletes_count_sd(),
		rocksdb_compaction_sequential_deletes_file_size: *Newrocksdb_compaction_sequential_deletes_file_size(),
		rocksdb_compaction_sequential_deletes_window:    *Newrocksdb_compaction_sequential_deletes_window(),
		rocksdb_compaction_sequential_deletes:           *Newrocksdb_compaction_sequential_deletes(),
		rocksdb_create_if_missing:                       *Newrocksdb_create_if_missing(),
		rocksdb_create_missing_column_families:          *Newrocksdb_create_missing_column_families(),
		rocksdb_db_write_buffer_size:                    *Newrocksdb_db_write_buffer_size(),
		rocksdb_deadlock_detect:                         *Newrocksdb_deadlock_detect(),
		rocksdb_debug_optimizer_no_zero_cardinality:     *Newrocksdb_debug_optimizer_no_zero_cardinality(),
		rocksdb_delayed_write_rate:                      *Newrocksdb_delayed_write_rate(),
		rocksdb_delete_obsolete_files_period_micros:     *Newrocksdb_delete_obsolete_files_period_micros(),
		rocksdb_enable_bulk_load_api:                    *Newrocksdb_enable_bulk_load_api(),
		rocksdb_enable_thread_tracking:                  *Newrocksdb_enable_thread_tracking(),
		rocksdb_enable_write_thread_adaptive_yield:      *Newrocksdb_enable_write_thread_adaptive_yield(),
		rocksdb_error_if_exists:                         *Newrocksdb_error_if_exists(),
		rocksdb_flush_log_at_trx_commit:                 *Newrocksdb_flush_log_at_trx_commit(),
		rocksdb_flush_memtable_on_analyze:               *Newrocksdb_flush_memtable_on_analyze(),
		rocksdb_force_compute_memtable_stats:            *Newrocksdb_force_compute_memtable_stats(),
		rocksdb_force_flush_memtable_now:                *Newrocksdb_force_flush_memtable_now(),
		rocksdb_force_index_records_in_range:            *Newrocksdb_force_index_records_in_range(),
		rocksdb_hash_index_allow_collision:              *Newrocksdb_hash_index_allow_collision(),
		rocksdb_keep_log_file_num:                       *Newrocksdb_keep_log_file_num(),
		rocksdb_lock_scanned_rows:                       *Newrocksdb_lock_scanned_rows(),
		rocksdb_lock_wait_timeout:                       *Newrocksdb_lock_wait_timeout(),
		rocksdb_log_file_time_to_roll:                   *Newrocksdb_log_file_time_to_roll(),
		rocksdb_manifest_preallocation_size:             *Newrocksdb_manifest_preallocation_size(),
		rocksdb_max_open_files:                          *Newrocksdb_max_open_files(),
		rocksdb_max_row_locks:                           *Newrocksdb_max_row_locks(),
		rocksdb_max_subcompactions:                      *Newrocksdb_max_subcompactions(),
		rocksdb_max_total_wal_size:                      *Newrocksdb_max_total_wal_size(),
		rocksdb_merge_buf_size:                          *Newrocksdb_merge_buf_size(),
		rocksdb_merge_combine_read_size:                 *Newrocksdb_merge_combine_read_size(),
		rocksdb_new_table_reader_for_compaction_inputs:  *Newrocksdb_new_table_reader_for_compaction_inputs(),
		rocksdb_no_block_cache:                          *Newrocksdb_no_block_cache(),
		rocksdb_paranoid_checks:                         *Newrocksdb_paranoid_checks(),
		rocksdb_pause_background_work:                   *Newrocksdb_pause_background_work(),
		rocksdb_perf_context_level:                      *Newrocksdb_perf_context_level(),
		rocksdb_persistent_cache_size_mb:                *Newrocksdb_persistent_cache_size_mb(),
		rocksdb_pin_l0_filter_and_index_blocks_in_cache: *Newrocksdb_pin_l0_filter_and_index_blocks_in_cache(),
		rocksdb_print_snapshot_conflict_queries:         *Newrocksdb_print_snapshot_conflict_queries(),
		rocksdb_rate_limiter_bytes_per_sec:              *Newrocksdb_rate_limiter_bytes_per_sec(),
		rocksdb_records_in_range:                        *Newrocksdb_records_in_range(),
		rocksdb_seconds_between_stat_computes:           *Newrocksdb_seconds_between_stat_computes(),
		rocksdb_signal_drop_index_thread:                *Newrocksdb_signal_drop_index_thread(),
		rocksdb_skip_bloom_filter_on_read:               *Newrocksdb_skip_bloom_filter_on_read(),
		rocksdb_skip_fill_cache:                         *Newrocksdb_skip_fill_cache(),
		rocksdb_stats_dump_period_sec:                   *Newrocksdb_stats_dump_period_sec(),
		rocksdb_store_row_debug_checksums:               *Newrocksdb_store_row_debug_checksums(),
		rocksdb_strict_collation_check:                  *Newrocksdb_strict_collation_check(),
		rocksdb_table_cache_numshardbits:                *Newrocksdb_table_cache_numshardbits(),
		rocksdb_use_adaptive_mutex:                      *Newrocksdb_use_adaptive_mutex(),
		rocksdb_use_direct_reads:                        *Newrocksdb_use_direct_reads(),
		rocksdb_use_fsync:                               *Newrocksdb_use_fsync(),
		rocksdb_validate_tables:                         *Newrocksdb_validate_tables(),
		rocksdb_verify_row_debug_checksums:              *Newrocksdb_verify_row_debug_checksums(),
		rocksdb_wal_bytes_per_sync:                      *Newrocksdb_wal_bytes_per_sync(),
		rocksdb_wal_recovery_mode:                       *Newrocksdb_wal_recovery_mode(),
		rocksdb_wal_size_limit_mb:                       *Newrocksdb_wal_size_limit_mb(),
		rocksdb_wal_ttl_seconds:                         *Newrocksdb_wal_ttl_seconds(),
		rocksdb_whole_key_filtering:                     *Newrocksdb_whole_key_filtering(),
		rocksdb_write_disable_wal:                       *Newrocksdb_write_disable_wal(),
		rocksdb_write_ignore_missing_column_families:    *Newrocksdb_write_ignore_missing_column_families(),
	}
}
func (qd ScrapeGlobalVariables) Collect(ch chan<- prometheus.Metric) {
	qd.instance = *GetInstance()

	if err := qd.instance.Ping(); err != nil {
		logrus.Errorf("ping mysql instance error: %s", err)
		return
	}
	db := instance.GetDB()
	rows, err := db.Query(globalVariablesQuery)
	if err != nil {
		logrus.Error(err)
		return
	}
	defer rows.Close()
	var key string
	var val sql.RawBytes

	for rows.Next() {
		err = rows.Scan(&key, &val)
		if err != nil {
			logrus.Error(err)
			continue
		}
		key = validPrometheusName(key)
		if floatVal, ok := parseStatus(val); ok {
			if key == "rocksdb_access_hint_on_compaction_start" {
				qd.rocksdb_access_hint_on_compaction_start.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_advise_random_on_open" {
				qd.rocksdb_advise_random_on_open.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_allow_concurrent_memtable_write" {
				qd.rocksdb_allow_concurrent_memtable_write.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_allow_mmap_reads" {
				qd.rocksdb_allow_mmap_reads.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_allow_mmap_writes" {
				qd.rocksdb_allow_mmap_writes.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_block_cache_size" {
				qd.rocksdb_block_cache_size.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_block_restart_interval" {
				qd.rocksdb_block_restart_interval.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_block_size_deviation" {
				qd.rocksdb_block_size_deviation.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_block_size" {
				qd.rocksdb_block_size.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_bulk_load_size" {
				qd.rocksdb_bulk_load_size.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_bulk_load" {
				qd.rocksdb_bulk_load.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_bytes_per_sync" {
				qd.rocksdb_bytes_per_sync.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_cache_index_and_filter_blocks" {
				qd.rocksdb_cache_index_and_filter_blocks.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_checksums_pct" {
				qd.rocksdb_checksums_pct.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_collect_sst_properties" {
				qd.rocksdb_collect_sst_properties.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_commit_in_the_middle" {
				qd.rocksdb_commit_in_the_middle.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_compaction_readahead_size" {
				qd.rocksdb_compaction_readahead_size.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_compaction_sequential_deletes_count_sd" {
				qd.rocksdb_compaction_sequential_deletes_count_sd.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_compaction_sequential_deletes_file_size" {
				qd.rocksdb_compaction_sequential_deletes_file_size.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_compaction_sequential_deletes_window" {
				qd.rocksdb_compaction_sequential_deletes_window.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_compaction_sequential_deletes" {
				qd.rocksdb_compaction_sequential_deletes.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_create_if_missing" {
				qd.rocksdb_create_if_missing.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_create_missing_column_families" {
				qd.rocksdb_create_missing_column_families.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_db_write_buffer_size" {
				qd.rocksdb_db_write_buffer_size.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_deadlock_detect" {
				qd.rocksdb_deadlock_detect.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_debug_optimizer_no_zero_cardinality" {
				qd.rocksdb_debug_optimizer_no_zero_cardinality.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_delayed_write_rate" {
				qd.rocksdb_delayed_write_rate.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_delete_obsolete_files_period_micros" {
				qd.rocksdb_delete_obsolete_files_period_micros.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_enable_bulk_load_api" {
				qd.rocksdb_enable_bulk_load_api.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_enable_thread_tracking" {
				qd.rocksdb_enable_thread_tracking.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_enable_write_thread_adaptive_yield" {
				qd.rocksdb_enable_write_thread_adaptive_yield.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_error_if_exists" {
				qd.rocksdb_error_if_exists.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_flush_log_at_trx_commit" {
				qd.rocksdb_flush_log_at_trx_commit.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_flush_memtable_on_analyze" {
				qd.rocksdb_flush_memtable_on_analyze.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_force_compute_memtable_stats" {
				qd.rocksdb_force_compute_memtable_stats.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_force_flush_memtable_now" {
				qd.rocksdb_force_flush_memtable_now.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_force_index_records_in_range" {
				qd.rocksdb_force_index_records_in_range.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_hash_index_allow_collision" {
				qd.rocksdb_hash_index_allow_collision.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_keep_log_file_num" {
				qd.rocksdb_keep_log_file_num.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_lock_scanned_rows" {
				qd.rocksdb_lock_scanned_rows.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_lock_wait_timeout" {
				qd.rocksdb_lock_wait_timeout.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_log_file_time_to_roll" {
				qd.rocksdb_log_file_time_to_roll.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_manifest_preallocation_size" {
				qd.rocksdb_manifest_preallocation_size.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_max_open_files" {
				qd.rocksdb_max_open_files.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_max_row_locks" {
				qd.rocksdb_max_row_locks.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_max_subcompactions" {
				qd.rocksdb_max_subcompactions.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_max_total_wal_size" {
				qd.rocksdb_max_total_wal_size.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_merge_buf_size" {
				qd.rocksdb_merge_buf_size.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_merge_combine_read_size" {
				qd.rocksdb_merge_combine_read_size.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_new_table_reader_for_compaction_inputs" {
				qd.rocksdb_new_table_reader_for_compaction_inputs.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_no_block_cache" {
				qd.rocksdb_no_block_cache.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_paranoid_checks" {
				qd.rocksdb_paranoid_checks.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_pause_background_work" {
				qd.rocksdb_pause_background_work.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_perf_context_level" {
				qd.rocksdb_perf_context_level.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_persistent_cache_size_mb" {
				qd.rocksdb_persistent_cache_size_mb.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_pin_l0_filter_and_index_blocks_in_cache" {
				qd.rocksdb_pin_l0_filter_and_index_blocks_in_cache.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_print_snapshot_conflict_queries" {
				qd.rocksdb_print_snapshot_conflict_queries.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_rate_limiter_bytes_per_sec" {
				qd.rocksdb_rate_limiter_bytes_per_sec.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_records_in_range" {
				qd.rocksdb_records_in_range.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_seconds_between_stat_computes" {
				qd.rocksdb_seconds_between_stat_computes.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_signal_drop_index_thread" {
				qd.rocksdb_signal_drop_index_thread.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_skip_bloom_filter_on_read" {
				qd.rocksdb_skip_bloom_filter_on_read.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_skip_fill_cache" {
				qd.rocksdb_skip_fill_cache.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_stats_dump_period_sec" {
				qd.rocksdb_stats_dump_period_sec.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_store_row_debug_checksums" {
				qd.rocksdb_store_row_debug_checksums.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_strict_collation_check" {
				qd.rocksdb_strict_collation_check.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_table_cache_numshardbits" {
				qd.rocksdb_table_cache_numshardbits.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_use_adaptive_mutex" {
				qd.rocksdb_use_adaptive_mutex.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_use_direct_reads" {
				qd.rocksdb_use_direct_reads.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_use_fsync" {
				qd.rocksdb_use_fsync.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_validate_tables" {
				qd.rocksdb_validate_tables.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_verify_row_debug_checksums" {
				qd.rocksdb_verify_row_debug_checksums.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_wal_bytes_per_sync" {
				qd.rocksdb_wal_bytes_per_sync.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_wal_recovery_mode" {
				qd.rocksdb_wal_recovery_mode.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_wal_size_limit_mb" {
				qd.rocksdb_wal_size_limit_mb.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_wal_ttl_seconds" {
				qd.rocksdb_wal_ttl_seconds.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_whole_key_filtering" {
				qd.rocksdb_whole_key_filtering.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_write_disable_wal" {
				qd.rocksdb_write_disable_wal.Collect(ch,
					floatVal,
					nil)
				continue
			}
			if key == "rocksdb_write_ignore_missing_column_families" {
				qd.rocksdb_write_ignore_missing_column_families.Collect(ch,
					floatVal,
					nil)
				continue
			}
		}

	}

}

type rocksdb_access_hint_on_compaction_start struct {
	*baseMetrics
}

func Newrocksdb_access_hint_on_compaction_start() *rocksdb_access_hint_on_compaction_start {
	return &rocksdb_access_hint_on_compaction_start{
		NewMetrics(
			"rocksdb_access_hint_on_compaction_start",
			"File access pattern once a compaction is started, applied to all input files of a compaction.",
			nil)}
}
func (qd *rocksdb_access_hint_on_compaction_start) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_advise_random_on_open struct {
	*baseMetrics
}

func Newrocksdb_advise_random_on_open() *rocksdb_advise_random_on_open {
	return &rocksdb_advise_random_on_open{
		NewMetrics(
			"rocksdb_advise_random_on_open",
			"Hint of random access to the filesystem when a data file is opened.",
			nil)}
}
func (qd *rocksdb_advise_random_on_open) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_allow_concurrent_memtable_write struct {
	*baseMetrics
}

func Newrocksdb_allow_concurrent_memtable_write() *rocksdb_allow_concurrent_memtable_write {
	return &rocksdb_allow_concurrent_memtable_write{
		NewMetrics(
			"rocksdb_allow_concurrent_memtable_write",
			"Allow multi-writers to update memtables in parallel.",
			nil)}
}
func (qd *rocksdb_allow_concurrent_memtable_write) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_allow_mmap_reads struct {
	*baseMetrics
}

func Newrocksdb_allow_mmap_reads() *rocksdb_allow_mmap_reads {
	return &rocksdb_allow_mmap_reads{
		NewMetrics(
			"rocksdb_allow_mmap_reads",
			"Allow the OS to mmap a data file for reads.",
			nil)}
}
func (qd *rocksdb_allow_mmap_reads) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_allow_mmap_writes struct {
	*baseMetrics
}

func Newrocksdb_allow_mmap_writes() *rocksdb_allow_mmap_writes {
	return &rocksdb_allow_mmap_writes{
		NewMetrics(
			"rocksdb_allow_mmap_writes",
			"Allow the OS to mmap a data file for writes.",
			nil)}
}
func (qd *rocksdb_allow_mmap_writes) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_block_cache_size struct {
	*baseMetrics
}

func Newrocksdb_block_cache_size() *rocksdb_block_cache_size {
	return &rocksdb_block_cache_size{
		NewMetrics(
			"rocksdb_block_cache_size",
			"Size of the LRU block cache in RocksDB. This memory is reserved for the block cache, which is in addition to any filesystem caching that may occur.",
			nil)}
}
func (qd *rocksdb_block_cache_size) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_block_restart_interval struct {
	*baseMetrics
}

func Newrocksdb_block_restart_interval() *rocksdb_block_restart_interval {
	return &rocksdb_block_restart_interval{
		NewMetrics(
			"rocksdb_block_restart_interval",
			"Number of keys for each set of delta encoded data.",
			nil)}
}
func (qd *rocksdb_block_restart_interval) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_block_size_deviation struct {
	*baseMetrics
}

func Newrocksdb_block_size_deviation() *rocksdb_block_size_deviation {
	return &rocksdb_block_size_deviation{
		NewMetrics(
			"rocksdb_block_size_deviation",
			"If the percentage of free space in the current data block (size specified in rocksdb-block-size) is less than this amount, close the block (and write record to new block).",
			nil)}
}
func (qd *rocksdb_block_size_deviation) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_block_size struct {
	*baseMetrics
}

func Newrocksdb_block_size() *rocksdb_block_size {
	return &rocksdb_block_size{
		NewMetrics(
			"rocksdb_block_size",
			"Size of the data block for reading sst files.",
			nil)}
}
func (qd *rocksdb_block_size) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_bulk_load_size struct {
	*baseMetrics
}

func Newrocksdb_bulk_load_size() *rocksdb_bulk_load_size {
	return &rocksdb_bulk_load_size{
		NewMetrics(
			"rocksdb_bulk_load_size",
			"Sets the number of keys to accumulate before committing them to the storage engine during bulk loading.",
			nil)}
}
func (qd *rocksdb_bulk_load_size) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_bulk_load struct {
	*baseMetrics
}

func Newrocksdb_bulk_load() *rocksdb_bulk_load {
	return &rocksdb_bulk_load{
		NewMetrics(
			"rocksdb_bulk_load",
			"When set, MyRocks will ignore checking keys for uniqueness or acquiring locks during transactions. This option should only be used when the application is certain there are no row conflicts, such as when setting up a new MyRocks instance from an existing MySQL dump.",
			nil)}
}
func (qd *rocksdb_bulk_load) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_bytes_per_sync struct {
	*baseMetrics
}

func Newrocksdb_bytes_per_sync() *rocksdb_bytes_per_sync {
	return &rocksdb_bytes_per_sync{
		NewMetrics(
			"rocksdb_bytes_per_sync",
			"Enables the OS to sync out file writes as data files are created.",
			nil)}
}
func (qd *rocksdb_bytes_per_sync) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_cache_index_and_filter_blocks struct {
	*baseMetrics
}

func Newrocksdb_cache_index_and_filter_blocks() *rocksdb_cache_index_and_filter_blocks {
	return &rocksdb_cache_index_and_filter_blocks{
		NewMetrics(
			"rocksdb_cache_index_and_filter_blocks",
			"Requests RocksDB to use the block cache for caching the index and bloomfilter data blocks from each data file. If this is not set, RocksDB will allocate additional memory to maintain these data blocks.",
			nil)}
}
func (qd *rocksdb_cache_index_and_filter_blocks) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_checksums_pct struct {
	*baseMetrics
}

func Newrocksdb_checksums_pct() *rocksdb_checksums_pct {
	return &rocksdb_checksums_pct{
		NewMetrics(
			"rocksdb_checksums_pct",
			"Sets the percentage of rows to calculate and set MyRocks checksums.",
			nil)}
}
func (qd *rocksdb_checksums_pct) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_collect_sst_properties struct {
	*baseMetrics
}

func Newrocksdb_collect_sst_properties() *rocksdb_collect_sst_properties {
	return &rocksdb_collect_sst_properties{
		NewMetrics(
			"rocksdb_collect_sst_properties",
			"Enables collecting statistics of each data file for improving optimizer behavior.",
			nil)}
}
func (qd *rocksdb_collect_sst_properties) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_commit_in_the_middle struct {
	*baseMetrics
}

func Newrocksdb_commit_in_the_middle() *rocksdb_commit_in_the_middle {
	return &rocksdb_commit_in_the_middle{
		NewMetrics(
			"rocksdb_commit_in_the_middle",
			"Commit rows implicitly every rocksdb-bulk-load-size, during bulk load/insert/update/deletes.",
			nil)}
}
func (qd *rocksdb_commit_in_the_middle) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_compaction_readahead_size struct {
	*baseMetrics
}

func Newrocksdb_compaction_readahead_size() *rocksdb_compaction_readahead_size {
	return &rocksdb_compaction_readahead_size{
		NewMetrics(
			"rocksdb_compaction_readahead_size",
			"When non-zero, bigger reads are performed during compaction. Useful if running RocksDB on spinning disks, compaction will do sequential instead of random reads.",
			nil)}
}
func (qd *rocksdb_compaction_readahead_size) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_compaction_sequential_deletes_count_sd struct {
	*baseMetrics
}

func Newrocksdb_compaction_sequential_deletes_count_sd() *rocksdb_compaction_sequential_deletes_count_sd {
	return &rocksdb_compaction_sequential_deletes_count_sd{
		NewMetrics(
			"rocksdb_compaction_sequential_deletes_count_sd",
			"If enabled, factor in single deletes as part of rocksdb-compaction-sequential-deletes.",
			nil)}
}
func (qd *rocksdb_compaction_sequential_deletes_count_sd) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_compaction_sequential_deletes_file_size struct {
	*baseMetrics
}

func Newrocksdb_compaction_sequential_deletes_file_size() *rocksdb_compaction_sequential_deletes_file_size {
	return &rocksdb_compaction_sequential_deletes_file_size{
		NewMetrics(
			"rocksdb_compaction_sequential_deletes_file_size",
			"Threshold to trigger compaction if the number of sequential keys that are all delete markers exceed this value. While this compaction helps reduce request latency by removing delete markers, it can increase write rates of RocksDB.",
			nil)}
}
func (qd *rocksdb_compaction_sequential_deletes_file_size) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_compaction_sequential_deletes_window struct {
	*baseMetrics
}

func Newrocksdb_compaction_sequential_deletes_window() *rocksdb_compaction_sequential_deletes_window {
	return &rocksdb_compaction_sequential_deletes_window{
		NewMetrics(
			"rocksdb_compaction_sequential_deletes_window",
			"Threshold to trigger compaction if, within a sliding window of keys, there exists this parameter's number of delete marker.",
			nil)}
}
func (qd *rocksdb_compaction_sequential_deletes_window) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_compaction_sequential_deletes struct {
	*baseMetrics
}

func Newrocksdb_compaction_sequential_deletes() *rocksdb_compaction_sequential_deletes {
	return &rocksdb_compaction_sequential_deletes{
		NewMetrics(
			"rocksdb_compaction_sequential_deletes",
			"Enables triggering of compaction when the number of delete markers in a data file exceeds a certain threshold. Depending on workload patterns, RocksDB can potentially maintain large numbers of delete markers and increase latency of all queries.",
			nil)}
}
func (qd *rocksdb_compaction_sequential_deletes) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_create_if_missing struct {
	*baseMetrics
}

func Newrocksdb_create_if_missing() *rocksdb_create_if_missing {
	return &rocksdb_create_if_missing{
		NewMetrics(
			"rocksdb_create_if_missing",
			"Allows creating the RocksDB database if it does not exist.",
			nil)}
}
func (qd *rocksdb_create_if_missing) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_create_missing_column_families struct {
	*baseMetrics
}

func Newrocksdb_create_missing_column_families() *rocksdb_create_missing_column_families {
	return &rocksdb_create_missing_column_families{
		NewMetrics(
			"rocksdb_create_missing_column_families",
			"Allows creating new column families if they did not exist.",
			nil)}
}
func (qd *rocksdb_create_missing_column_families) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_db_write_buffer_size struct {
	*baseMetrics
}

func Newrocksdb_db_write_buffer_size() *rocksdb_db_write_buffer_size {
	return &rocksdb_db_write_buffer_size{
		NewMetrics(
			"rocksdb_db_write_buffer_size",
			"Size of the memtable used to store writes within RocksDB. This is the size per column family. Once this size is reached, a flush of the memtable to persistent media occurs.",
			nil)}
}
func (qd *rocksdb_db_write_buffer_size) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_deadlock_detect struct {
	*baseMetrics
}

func Newrocksdb_deadlock_detect() *rocksdb_deadlock_detect {
	return &rocksdb_deadlock_detect{
		NewMetrics(
			"rocksdb_deadlock_detect",
			"Enables deadlock detection in RocksDB.",
			nil)}
}
func (qd *rocksdb_deadlock_detect) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_debug_optimizer_no_zero_cardinality struct {
	*baseMetrics
}

func Newrocksdb_debug_optimizer_no_zero_cardinality() *rocksdb_debug_optimizer_no_zero_cardinality {
	return &rocksdb_debug_optimizer_no_zero_cardinality{
		NewMetrics(
			"rocksdb_debug_optimizer_no_zero_cardinality",
			"Test only to prevent MyRocks from calculating cardinality.",
			nil)}
}
func (qd *rocksdb_debug_optimizer_no_zero_cardinality) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_delayed_write_rate struct {
	*baseMetrics
}

func Newrocksdb_delayed_write_rate() *rocksdb_delayed_write_rate {
	return &rocksdb_delayed_write_rate{
		NewMetrics(
			"rocksdb_delayed_write_rate",
			"When RocksDB hits the soft limits/thresholds for writes, such as soft_pending_compaction_bytes_limit being hit, or level0_slowdown_writes_trigger being hit, RocksDB will slow the write rate down to the value of this parameter as bytes/second.",
			nil)}
}
func (qd *rocksdb_delayed_write_rate) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_delete_obsolete_files_period_micros struct {
	*baseMetrics
}

func Newrocksdb_delete_obsolete_files_period_micros() *rocksdb_delete_obsolete_files_period_micros {
	return &rocksdb_delete_obsolete_files_period_micros{
		NewMetrics(
			"rocksdb_delete_obsolete_files_period_micros",
			"The periodicity of when obsolete files get deleted, but does not affect files removed through compaction.",
			nil)}
}
func (qd *rocksdb_delete_obsolete_files_period_micros) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_enable_bulk_load_api struct {
	*baseMetrics
}

func Newrocksdb_enable_bulk_load_api() *rocksdb_enable_bulk_load_api {
	return &rocksdb_enable_bulk_load_api{
		NewMetrics(
			"rocksdb_enable_bulk_load_api",
			"Enables using the SSTFileWriter feature in RocksDB, which bypasses the memtable, but this requires keys to be inserted into the table in either ascending or descending order. If disabled, bulk loading uses the normal write path via the memtable and does not keys to be inserted in any order.",
			nil)}
}
func (qd *rocksdb_enable_bulk_load_api) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_enable_thread_tracking struct {
	*baseMetrics
}

func Newrocksdb_enable_thread_tracking() *rocksdb_enable_thread_tracking {
	return &rocksdb_enable_thread_tracking{
		NewMetrics(
			"rocksdb_enable_thread_tracking",
			"Set to allow RocksDB to track the status of threads accessing the database.",
			nil)}
}
func (qd *rocksdb_enable_thread_tracking) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_enable_write_thread_adaptive_yield struct {
	*baseMetrics
}

func Newrocksdb_enable_write_thread_adaptive_yield() *rocksdb_enable_write_thread_adaptive_yield {
	return &rocksdb_enable_write_thread_adaptive_yield{
		NewMetrics(
			"rocksdb_enable_write_thread_adaptive_yield",
			"Set to allow RocksDB write batch group leader to wait up to the max time allowed before blocking on a mutex, allowing an increase in throughput for concurrent workloads.",
			nil)}
}
func (qd *rocksdb_enable_write_thread_adaptive_yield) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_error_if_exists struct {
	*baseMetrics
}

func Newrocksdb_error_if_exists() *rocksdb_error_if_exists {
	return &rocksdb_error_if_exists{
		NewMetrics(
			"rocksdb_error_if_exists",
			"If set, reports an error if an existing database already exists.",
			nil)}
}
func (qd *rocksdb_error_if_exists) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_flush_log_at_trx_commit struct {
	*baseMetrics
}

func Newrocksdb_flush_log_at_trx_commit() *rocksdb_flush_log_at_trx_commit {
	return &rocksdb_flush_log_at_trx_commit{
		NewMetrics(
			"rocksdb_flush_log_at_trx_commit",
			"Sync'ing on transaction commit similar to innodb-flush-log-at-trx-commit: 0 - never sync, 1 - always sync, 2 - sync based on a timer controlled via rocksdb-background-sync",
			nil)}
}
func (qd *rocksdb_flush_log_at_trx_commit) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_flush_memtable_on_analyze struct {
	*baseMetrics
}

func Newrocksdb_flush_memtable_on_analyze() *rocksdb_flush_memtable_on_analyze {
	return &rocksdb_flush_memtable_on_analyze{
		NewMetrics(
			"rocksdb_flush_memtable_on_analyze",
			"When analyze table is run, determines of the memtable should be flushed so that data in the memtable is also used for calculating stats.",
			nil)}
}
func (qd *rocksdb_flush_memtable_on_analyze) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_force_compute_memtable_stats struct {
	*baseMetrics
}

func Newrocksdb_force_compute_memtable_stats() *rocksdb_force_compute_memtable_stats {
	return &rocksdb_force_compute_memtable_stats{
		NewMetrics(
			"rocksdb_force_compute_memtable_stats",
			"When enabled, also include data in the memtables for index statistics calculations used by the query optimizer. Greater accuracy, but requires more cpu.",
			nil)}
}
func (qd *rocksdb_force_compute_memtable_stats) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_force_flush_memtable_now struct {
	*baseMetrics
}

func Newrocksdb_force_flush_memtable_now() *rocksdb_force_flush_memtable_now {
	return &rocksdb_force_flush_memtable_now{
		NewMetrics(
			"rocksdb_force_flush_memtable_now",
			"Triggers MyRocks to flush the memtables out to the data files.",
			nil)}
}
func (qd *rocksdb_force_flush_memtable_now) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_force_index_records_in_range struct {
	*baseMetrics
}

func Newrocksdb_force_index_records_in_range() *rocksdb_force_index_records_in_range {
	return &rocksdb_force_index_records_in_range{
		NewMetrics(
			"rocksdb_force_index_records_in_range",
			"When force index is used, a non-zero value here will be used as the number of rows to be returned to the query optimizer when trying to determine the estimated number of rows.",
			nil)}
}
func (qd *rocksdb_force_index_records_in_range) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_hash_index_allow_collision struct {
	*baseMetrics
}

func Newrocksdb_hash_index_allow_collision() *rocksdb_hash_index_allow_collision {
	return &rocksdb_hash_index_allow_collision{
		NewMetrics(
			"rocksdb_hash_index_allow_collision",
			"Enables RocksDB to allow hashes to collide (uses less memory). Otherwise, the full prefix is stored to prevent hash collisions.",
			nil)}
}
func (qd *rocksdb_hash_index_allow_collision) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_keep_log_file_num struct {
	*baseMetrics
}

func Newrocksdb_keep_log_file_num() *rocksdb_keep_log_file_num {
	return &rocksdb_keep_log_file_num{
		NewMetrics(
			"rocksdb_keep_log_file_num",
			"Sets the maximum number of info LOG files to keep around.",
			nil)}
}
func (qd *rocksdb_keep_log_file_num) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_lock_scanned_rows struct {
	*baseMetrics
}

func Newrocksdb_lock_scanned_rows() *rocksdb_lock_scanned_rows {
	return &rocksdb_lock_scanned_rows{
		NewMetrics(
			"rocksdb_lock_scanned_rows",
			"If enabled, rows that are scanned during UPDATE remain locked even if they have not been updated.",
			nil)}
}
func (qd *rocksdb_lock_scanned_rows) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_lock_wait_timeout struct {
	*baseMetrics
}

func Newrocksdb_lock_wait_timeout() *rocksdb_lock_wait_timeout {
	return &rocksdb_lock_wait_timeout{
		NewMetrics(
			"rocksdb_lock_wait_timeout",
			"Sets the number of seconds MyRocks will wait to acquire a row lock before aborting the request.",
			nil)}
}
func (qd *rocksdb_lock_wait_timeout) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_log_file_time_to_roll struct {
	*baseMetrics
}

func Newrocksdb_log_file_time_to_roll() *rocksdb_log_file_time_to_roll {
	return &rocksdb_log_file_time_to_roll{
		NewMetrics(
			"rocksdb_log_file_time_to_roll",
			"Sets the number of seconds a info LOG file captures before rolling to a new LOG file.",
			nil)}
}
func (qd *rocksdb_log_file_time_to_roll) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_manifest_preallocation_size struct {
	*baseMetrics
}

func Newrocksdb_manifest_preallocation_size() *rocksdb_manifest_preallocation_size {
	return &rocksdb_manifest_preallocation_size{
		NewMetrics(
			"rocksdb_manifest_preallocation_size",
			"Sets the number of bytes to preallocate for the MANIFEST file in RocksDB and reduce possible random I/O on XFS. MANIFEST files are used to store information about column families, levels, active files, etc.",
			nil)}
}
func (qd *rocksdb_manifest_preallocation_size) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_max_open_files struct {
	*baseMetrics
}

func Newrocksdb_max_open_files() *rocksdb_max_open_files {
	return &rocksdb_max_open_files{
		NewMetrics(
			"rocksdb_max_open_files",
			"Sets a limit on the maximum number of file handles opened by RocksDB.",
			nil)}
}
func (qd *rocksdb_max_open_files) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_max_row_locks struct {
	*baseMetrics
}

func Newrocksdb_max_row_locks() *rocksdb_max_row_locks {
	return &rocksdb_max_row_locks{
		NewMetrics(
			"rocksdb_max_row_locks",
			"Sets a limit on the maximum number of row locks held by a transaction before failing it.",
			nil)}
}
func (qd *rocksdb_max_row_locks) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_max_subcompactions struct {
	*baseMetrics
}

func Newrocksdb_max_subcompactions() *rocksdb_max_subcompactions {
	return &rocksdb_max_subcompactions{
		NewMetrics(
			"rocksdb_max_subcompactions",
			"For each compaction job, the maximum threads that will work on it simultaneously (i.e. subcompactions). A value of 1 means no subcompactions.",
			nil)}
}
func (qd *rocksdb_max_subcompactions) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_max_total_wal_size struct {
	*baseMetrics
}

func Newrocksdb_max_total_wal_size() *rocksdb_max_total_wal_size {
	return &rocksdb_max_total_wal_size{
		NewMetrics(
			"rocksdb_max_total_wal_size",
			"Sets a limit on the maximum size of WAL files kept around. Once this limit is hit, RocksDB will force the flushing of memtables to reduce the size of WAL files.",
			nil)}
}
func (qd *rocksdb_max_total_wal_size) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_merge_buf_size struct {
	*baseMetrics
}

func Newrocksdb_merge_buf_size() *rocksdb_merge_buf_size {
	return &rocksdb_merge_buf_size{
		NewMetrics(
			"rocksdb_merge_buf_size",
			"Size (in bytes) of the merge buffers used to accumulate data during secondary key creation. During secondary key creation the data, we avoid updating the new indexes through the memtable and L0 by writing new entries directly to the lowest level in the database. This requires the values to be sorted so we use a merge/sort algorithm. This setting controls how large the merge buffers are. The default is 64Mb.",
			nil)}
}
func (qd *rocksdb_merge_buf_size) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_merge_combine_read_size struct {
	*baseMetrics
}

func Newrocksdb_merge_combine_read_size() *rocksdb_merge_combine_read_size {
	return &rocksdb_merge_combine_read_size{
		NewMetrics(
			"rocksdb_merge_combine_read_size",
			"Size (in bytes) of the merge combine buffer used in the merge/sort algorithm as described in rocksdb-merge-buf-size.",
			nil)}
}
func (qd *rocksdb_merge_combine_read_size) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_new_table_reader_for_compaction_inputs struct {
	*baseMetrics
}

func Newrocksdb_new_table_reader_for_compaction_inputs() *rocksdb_new_table_reader_for_compaction_inputs {
	return &rocksdb_new_table_reader_for_compaction_inputs{
		NewMetrics(
			"rocksdb_new_table_reader_for_compaction_inputs",
			"Indicates whether RocksDB should create a new file descriptor and table reader for each compaction input. Doing so may use more memory but may allow pre-fetch options to be specified for compaction input files without impacting table readers used for user queries.",
			nil)}
}
func (qd *rocksdb_new_table_reader_for_compaction_inputs) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_no_block_cache struct {
	*baseMetrics
}

func Newrocksdb_no_block_cache() *rocksdb_no_block_cache {
	return &rocksdb_no_block_cache{
		NewMetrics(
			"rocksdb_no_block_cache",
			"Disables using the block cache for a column family.",
			nil)}
}
func (qd *rocksdb_no_block_cache) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_paranoid_checks struct {
	*baseMetrics
}

func Newrocksdb_paranoid_checks() *rocksdb_paranoid_checks {
	return &rocksdb_paranoid_checks{
		NewMetrics(
			"rocksdb_paranoid_checks",
			"Forces RocksDB to re-read a data file that was just created to verify correctness.",
			nil)}
}
func (qd *rocksdb_paranoid_checks) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_pause_background_work struct {
	*baseMetrics
}

func Newrocksdb_pause_background_work() *rocksdb_pause_background_work {
	return &rocksdb_pause_background_work{
		NewMetrics(
			"rocksdb_pause_background_work",
			"Test only to start and stop all background compactions within RocksDB.",
			nil)}
}
func (qd *rocksdb_pause_background_work) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_perf_context_level struct {
	*baseMetrics
}

func Newrocksdb_perf_context_level() *rocksdb_perf_context_level {
	return &rocksdb_perf_context_level{
		NewMetrics(
			"rocksdb_perf_context_level",
			"Sets the level of information to capture via the perf context plugins.",
			nil)}
}
func (qd *rocksdb_perf_context_level) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_persistent_cache_size_mb struct {
	*baseMetrics
}

func Newrocksdb_persistent_cache_size_mb() *rocksdb_persistent_cache_size_mb {
	return &rocksdb_persistent_cache_size_mb{
		NewMetrics(
			"rocksdb_persistent_cache_size_mb",
			"The size (in Mb) to allocate to the RocksDB persistent cache if desired.",
			nil)}
}
func (qd *rocksdb_persistent_cache_size_mb) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_pin_l0_filter_and_index_blocks_in_cache struct {
	*baseMetrics
}

func Newrocksdb_pin_l0_filter_and_index_blocks_in_cache() *rocksdb_pin_l0_filter_and_index_blocks_in_cache {
	return &rocksdb_pin_l0_filter_and_index_blocks_in_cache{
		NewMetrics(
			"rocksdb_pin_l0_filter_and_index_blocks_in_cache",
			"If rocksdb-cache-index-and-filter-blocks is true then this controls whether RocksDB 'pins' the filter and index blocks in the cache.",
			nil)}
}
func (qd *rocksdb_pin_l0_filter_and_index_blocks_in_cache) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_print_snapshot_conflict_queries struct {
	*baseMetrics
}

func Newrocksdb_print_snapshot_conflict_queries() *rocksdb_print_snapshot_conflict_queries {
	return &rocksdb_print_snapshot_conflict_queries{
		NewMetrics(
			"rocksdb_print_snapshot_conflict_queries",
			"If this is true, MyRocks will log queries that generate snapshot conflicts into the .err log.",
			nil)}
}
func (qd *rocksdb_print_snapshot_conflict_queries) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_rate_limiter_bytes_per_sec struct {
	*baseMetrics
}

func Newrocksdb_rate_limiter_bytes_per_sec() *rocksdb_rate_limiter_bytes_per_sec {
	return &rocksdb_rate_limiter_bytes_per_sec{
		NewMetrics(
			"rocksdb_rate_limiter_bytes_per_sec",
			"Controls the rate at which RocksDB is allowed to write to media via memtable flushes and compaction.",
			nil)}
}
func (qd *rocksdb_rate_limiter_bytes_per_sec) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_records_in_range struct {
	*baseMetrics
}

func Newrocksdb_records_in_range() *rocksdb_records_in_range {
	return &rocksdb_records_in_range{
		NewMetrics(
			"rocksdb_records_in_range",
			"Test only to override the value returned by records-in-range.",
			nil)}
}
func (qd *rocksdb_records_in_range) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_seconds_between_stat_computes struct {
	*baseMetrics
}

func Newrocksdb_seconds_between_stat_computes() *rocksdb_seconds_between_stat_computes {
	return &rocksdb_seconds_between_stat_computes{
		NewMetrics(
			"rocksdb_seconds_between_stat_computes",
			"Sets the number of seconds between recomputation of table statistics for the optimizer.",
			nil)}
}
func (qd *rocksdb_seconds_between_stat_computes) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_signal_drop_index_thread struct {
	*baseMetrics
}

func Newrocksdb_signal_drop_index_thread() *rocksdb_signal_drop_index_thread {
	return &rocksdb_signal_drop_index_thread{
		NewMetrics(
			"rocksdb_signal_drop_index_thread",
			"Test only to signal the MyRocks drop index thread.",
			nil)}
}
func (qd *rocksdb_signal_drop_index_thread) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_skip_bloom_filter_on_read struct {
	*baseMetrics
}

func Newrocksdb_skip_bloom_filter_on_read() *rocksdb_skip_bloom_filter_on_read {
	return &rocksdb_skip_bloom_filter_on_read{
		NewMetrics(
			"rocksdb_skip_bloom_filter_on_read",
			"Indicates whether the bloom filters should be skipped on reads.",
			nil)}
}
func (qd *rocksdb_skip_bloom_filter_on_read) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_skip_fill_cache struct {
	*baseMetrics
}

func Newrocksdb_skip_fill_cache() *rocksdb_skip_fill_cache {
	return &rocksdb_skip_fill_cache{
		NewMetrics(
			"rocksdb_skip_fill_cache",
			"Requests MyRocks to skip caching data on read requests.",
			nil)}
}
func (qd *rocksdb_skip_fill_cache) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_stats_dump_period_sec struct {
	*baseMetrics
}

func Newrocksdb_stats_dump_period_sec() *rocksdb_stats_dump_period_sec {
	return &rocksdb_stats_dump_period_sec{
		NewMetrics(
			"rocksdb_stats_dump_period_sec",
			"Sets the number of seconds to perform a RocksDB stats dump to the info LOG files.",
			nil)}
}
func (qd *rocksdb_stats_dump_period_sec) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_store_row_debug_checksums struct {
	*baseMetrics
}

func Newrocksdb_store_row_debug_checksums() *rocksdb_store_row_debug_checksums {
	return &rocksdb_store_row_debug_checksums{
		NewMetrics(
			"rocksdb_store_row_debug_checksums",
			"Include checksums when writing index/table records.",
			nil)}
}
func (qd *rocksdb_store_row_debug_checksums) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_strict_collation_check struct {
	*baseMetrics
}

func Newrocksdb_strict_collation_check() *rocksdb_strict_collation_check {
	return &rocksdb_strict_collation_check{
		NewMetrics(
			"rocksdb_strict_collation_check",
			"Enables MyRocks to check and verify table indexes have the proper collation settings.",
			nil)}
}
func (qd *rocksdb_strict_collation_check) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_table_cache_numshardbits struct {
	*baseMetrics
}

func Newrocksdb_table_cache_numshardbits() *rocksdb_table_cache_numshardbits {
	return &rocksdb_table_cache_numshardbits{
		NewMetrics(
			"rocksdb_table_cache_numshardbits",
			"Sets the number of table caches within RocksDB.",
			nil)}
}
func (qd *rocksdb_table_cache_numshardbits) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_use_adaptive_mutex struct {
	*baseMetrics
}

func Newrocksdb_use_adaptive_mutex() *rocksdb_use_adaptive_mutex {
	return &rocksdb_use_adaptive_mutex{
		NewMetrics(
			"rocksdb_use_adaptive_mutex",
			"Enables adaptive mutexes in RocksDB which spins in user space before resorting to the kernel.",
			nil)}
}
func (qd *rocksdb_use_adaptive_mutex) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_use_direct_reads struct {
	*baseMetrics
}

func Newrocksdb_use_direct_reads() *rocksdb_use_direct_reads {
	return &rocksdb_use_direct_reads{
		NewMetrics(
			"rocksdb_use_direct_reads",
			"Enable direct IO when opening a file for read/write. This means that data will not be cached or buffered.",
			nil)}
}
func (qd *rocksdb_use_direct_reads) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_use_fsync struct {
	*baseMetrics
}

func Newrocksdb_use_fsync() *rocksdb_use_fsync {
	return &rocksdb_use_fsync{
		NewMetrics(
			"rocksdb_use_fsync",
			"Requires RocksDB to use fsync instead of fdatasync when requesting a sync of a data file.",
			nil)}
}
func (qd *rocksdb_use_fsync) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_validate_tables struct {
	*baseMetrics
}

func Newrocksdb_validate_tables() *rocksdb_validate_tables {
	return &rocksdb_validate_tables{
		NewMetrics(
			"rocksdb_validate_tables",
			"Requires MyRocks to verify all of MySQL's .frm files match tables stored in RocksDB.",
			nil)}
}
func (qd *rocksdb_validate_tables) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_verify_row_debug_checksums struct {
	*baseMetrics
}

func Newrocksdb_verify_row_debug_checksums() *rocksdb_verify_row_debug_checksums {
	return &rocksdb_verify_row_debug_checksums{
		NewMetrics(
			"rocksdb_verify_row_debug_checksums",
			"Verify checksums when reading index/table records.",
			nil)}
}
func (qd *rocksdb_verify_row_debug_checksums) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_wal_bytes_per_sync struct {
	*baseMetrics
}

func Newrocksdb_wal_bytes_per_sync() *rocksdb_wal_bytes_per_sync {
	return &rocksdb_wal_bytes_per_sync{
		NewMetrics(
			"rocksdb_wal_bytes_per_sync",
			"Controls the rate at which RocksDB writes out WAL file data.",
			nil)}
}
func (qd *rocksdb_wal_bytes_per_sync) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_wal_recovery_mode struct {
	*baseMetrics
}

func Newrocksdb_wal_recovery_mode() *rocksdb_wal_recovery_mode {
	return &rocksdb_wal_recovery_mode{
		NewMetrics(
			"rocksdb_wal_recovery_mode",
			"Sets RocksDB's level of tolerance when recovering the WAL files after a system crash.",
			nil)}
}
func (qd *rocksdb_wal_recovery_mode) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_wal_size_limit_mb struct {
	*baseMetrics
}

func Newrocksdb_wal_size_limit_mb() *rocksdb_wal_size_limit_mb {
	return &rocksdb_wal_size_limit_mb{
		NewMetrics(
			"rocksdb_wal_size_limit_mb",
			"Maximum size the RocksDB WAL is allow to grow to. When this size is exceeded rocksdb attempts to flush sufficient memtables to allow for the deletion of the oldest log.",
			nil)}
}
func (qd *rocksdb_wal_size_limit_mb) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_wal_ttl_seconds struct {
	*baseMetrics
}

func Newrocksdb_wal_ttl_seconds() *rocksdb_wal_ttl_seconds {
	return &rocksdb_wal_ttl_seconds{
		NewMetrics(
			"rocksdb_wal_ttl_seconds",
			"No WAL file older than this value should exist.",
			nil)}
}
func (qd *rocksdb_wal_ttl_seconds) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_whole_key_filtering struct {
	*baseMetrics
}

func Newrocksdb_whole_key_filtering() *rocksdb_whole_key_filtering {
	return &rocksdb_whole_key_filtering{
		NewMetrics(
			"rocksdb_whole_key_filtering",
			"Enables the bloomfilter to use the whole key for filtering instead of just the prefix. In order for this to be efficient, lookups should use the whole key for matching.",
			nil)}
}
func (qd *rocksdb_whole_key_filtering) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_write_disable_wal struct {
	*baseMetrics
}

func Newrocksdb_write_disable_wal() *rocksdb_write_disable_wal {
	return &rocksdb_write_disable_wal{
		NewMetrics(
			"rocksdb_write_disable_wal",
			"Disables logging data to the WAL files. Useful for bulk loading.",
			nil)}
}
func (qd *rocksdb_write_disable_wal) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type rocksdb_write_ignore_missing_column_families struct {
	*baseMetrics
}

func Newrocksdb_write_ignore_missing_column_families() *rocksdb_write_ignore_missing_column_families {
	return &rocksdb_write_ignore_missing_column_families{
		NewMetrics(
			"rocksdb_write_ignore_missing_column_families",
			"If 1, then writes to column families that do not exist is ignored by RocksDB.",
			nil)}
}
func (qd *rocksdb_write_ignore_missing_column_families) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

func parseWsrepProviderOptions(opts string) float64 {
	var val float64
	r, _ := regexp.Compile(`gcache.size = (\d+)([MG]?);`)
	data := r.FindStringSubmatch(opts)
	if data == nil {
		return 0
	}

	val, _ = strconv.ParseFloat(data[1], 64)
	switch data[2] {
	case "M":
		val = val * 1024 * 1024
	case "G":
		val = val * 1024 * 1024 * 1024
	}

	return val
}

func validPrometheusName(s string) string {
	nameRe := regexp.MustCompile("([^a-zA-Z0-9_])")
	s = nameRe.ReplaceAllString(s, "_")
	s = strings.ToLower(s)
	return s
}

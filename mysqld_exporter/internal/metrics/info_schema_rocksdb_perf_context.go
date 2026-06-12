package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"mysqld_exporter/internal/exporter"
	"mysqld_exporter/internal/mysql"
)

const (
	rocksdbPerfContextQuery = `
                SELECT
                  TABLE_SCHEMA, 
                  TABLE_NAME, 
                  ifnull(PARTITION_NAME, ''), 
                  STAT_TYPE,
                  VALUE
                  FROM information_schema.ROCKSDB_PERF_CONTEXT
                `
)

func init() {
	exporter.Register(
		NewScrapeRocksDBPerfContext())
}

type ScrapeRocksDBPerfContext struct {
	instance mysql.Instance
	infoSchema_rocksdb_perf_context_user_key_comparison_count
	infoSchema_rocksdb_perf_context_block_cache_hit_count
	infoSchema_rocksdb_perf_context_block_read_count
	infoSchema_rocksdb_perf_context_block_read_byte
	infoSchema_rocksdb_perf_context_get_read_bytes
	infoSchema_rocksdb_perf_context_multiget_read_bytes
	infoSchema_rocksdb_perf_context_iter_read_bytes
	infoSchema_rocksdb_perf_context_internal_key_skipped_count
	infoSchema_rocksdb_perf_context_internal_delete_skipped_count
	infoSchema_rocksdb_perf_context_internal_recent_skipped_count
	infoSchema_rocksdb_perf_context_internal_merge_count
	infoSchema_rocksdb_perf_context_get_from_memtable_count
	infoSchema_rocksdb_perf_context_seek_on_memtable_count
	infoSchema_rocksdb_perf_context_next_on_memtable_count
	infoSchema_rocksdb_perf_context_prev_on_memtable_count
	infoSchema_rocksdb_perf_context_seek_child_seek_count
	infoSchema_rocksdb_perf_context_bloom_memtable_hit_count
	infoSchema_rocksdb_perf_context_bloom_memtable_miss_count
	infoSchema_rocksdb_perf_context_bloom_sst_hit_count
	infoSchema_rocksdb_perf_context_bloom_sst_miss_count
	infoSchema_rocksdb_perf_context_key_lock_wait_count
	infoSchema_rocksdb_perf_context_io_bytes_written
	infoSchema_rocksdb_perf_context_io_bytes_read
}

func (qd ScrapeRocksDBPerfContext) Collect(ch chan<- prometheus.Metric) {
	qd.instance = *GetInstance()

	if err := qd.instance.Ping(); err != nil {
		logrus.Errorf("ping mysql instance error: %s", err)
		return
	}
	db := instance.GetDB()
	informationSchemaInnodbCmpMemRows, err := db.Query(rocksdbPerfContextQuery)
	if err != nil {
		logrus.Debugf("failed to query mysql instance information_schema.ROCKSDB_PERF_CONTEXT: %s",
			err)
		return
	}
	defer informationSchemaInnodbCmpMemRows.Close()
	var (
		schema string
		table  string
		part   string
		stat   string
		value  float64
	)
	for informationSchemaInnodbCmpMemRows.Next() {
		if err := informationSchemaInnodbCmpMemRows.Scan(
			&schema,
			&table,
			&part,
			&stat,
			&value,
		); err != nil {
			logrus.Errorf("failed to scan mysql instance information_schema.ROCKSDB_PERF_CONTEXT: %s",
				err)
			return
		}
		if stat == "USER_KEY_COMPARISON_COUNT" {
			qd.infoSchema_rocksdb_perf_context_user_key_comparison_count.Collect(ch,
				float64(value),
				[]string{
					schema,
					table,
					part,
				})
		}
		if stat == "BLOCK_CACHE_HIT_COUNT" {
			qd.infoSchema_rocksdb_perf_context_block_cache_hit_count.Collect(ch,
				float64(value),
				[]string{
					schema,
					table,
					part,
				})
		}
		if stat == "BLOCK_READ_COUNT" {
			qd.infoSchema_rocksdb_perf_context_block_read_count.Collect(ch,
				float64(value),
				[]string{
					schema,
					table,
					part,
				})
		}
		if stat == "BLOCK_READ_BYTE" {
			qd.infoSchema_rocksdb_perf_context_block_read_byte.Collect(ch,
				float64(value),
				[]string{
					schema,
					table,
					part,
				})
		}
		if stat == "GET_READ_BYTES" {
			qd.infoSchema_rocksdb_perf_context_get_read_bytes.Collect(ch,
				float64(value),
				[]string{
					schema,
					table,
					part,
				})
		}
		if stat == "MULTIGET_READ_BYTES" {
			qd.infoSchema_rocksdb_perf_context_multiget_read_bytes.Collect(ch,
				float64(value),
				[]string{
					schema,
					table,
					part,
				})
		}
		if stat == "ITER_READ_BYTES" {
			qd.infoSchema_rocksdb_perf_context_iter_read_bytes.Collect(ch,
				float64(value),
				[]string{
					schema,
					table,
					part,
				})
		}
		if stat == "INTERNAL_KEY_SKIPPED_COUNT" {
			qd.infoSchema_rocksdb_perf_context_internal_key_skipped_count.Collect(ch,
				float64(value),
				[]string{
					schema,
					table,
					part,
				})
		}
		if stat == "INTERNAL_DELETE_SKIPPED_COUNT" {
			qd.infoSchema_rocksdb_perf_context_internal_delete_skipped_count.Collect(ch,
				float64(value),
				[]string{
					schema,
					table,
					part,
				})
		}
		if stat == "INTERNAL_RECENT_SKIPPED_COUNT" {
			qd.infoSchema_rocksdb_perf_context_internal_recent_skipped_count.Collect(ch,
				float64(value),
				[]string{
					schema,
					table,
					part,
				})
		}
		if stat == "INTERNAL_MERGE_COUNT" {
			qd.infoSchema_rocksdb_perf_context_internal_merge_count.Collect(ch,
				float64(value),
				[]string{
					schema,
					table,
					part,
				})
		}
		if stat == "GET_FROM_MEMTABLE_COUNT" {
			qd.infoSchema_rocksdb_perf_context_get_from_memtable_count.Collect(ch,
				float64(value),
				[]string{
					schema,
					table,
					part,
				})
		}
		if stat == "SEEK_ON_MEMTABLE_COUNT" {
			qd.infoSchema_rocksdb_perf_context_seek_on_memtable_count.Collect(ch,
				float64(value),
				[]string{
					schema,
					table,
					part,
				})
		}
		if stat == "NEXT_ON_MEMTABLE_COUNT" {
			qd.infoSchema_rocksdb_perf_context_next_on_memtable_count.Collect(ch,
				float64(value),
				[]string{
					schema,
					table,
					part,
				})
		}
		if stat == "PREV_ON_MEMTABLE_COUNT" {
			qd.infoSchema_rocksdb_perf_context_prev_on_memtable_count.Collect(ch,
				float64(value),
				[]string{
					schema,
					table,
					part,
				})
		}
		if stat == "SEEK_CHILD_SEEK_COUNT" {
			qd.infoSchema_rocksdb_perf_context_seek_child_seek_count.Collect(ch,
				float64(value),
				[]string{
					schema,
					table,
					part,
				})
		}
		if stat == "BLOOM_MEMTABLE_HIT_COUNT" {
			qd.infoSchema_rocksdb_perf_context_bloom_memtable_hit_count.Collect(ch,
				float64(value),
				[]string{
					schema,
					table,
					part,
				})
		}
		if stat == "BLOOM_MEMTABLE_MISS_COUNT" {
			qd.infoSchema_rocksdb_perf_context_bloom_memtable_miss_count.Collect(ch,
				float64(value),
				[]string{
					schema,
					table,
					part,
				})
		}
		if stat == "BLOOM_SST_HIT_COUNT" {
			qd.infoSchema_rocksdb_perf_context_bloom_sst_hit_count.Collect(ch,
				float64(value),
				[]string{
					schema,
					table,
					part,
				})
		}
		if stat == "BLOOM_SST_MISS_COUNT" {
			qd.infoSchema_rocksdb_perf_context_bloom_sst_miss_count.Collect(ch,
				float64(value),
				[]string{
					schema,
					table,
					part,
				})
		}
		if stat == "KEY_LOCK_WAIT_COUNT" {
			qd.infoSchema_rocksdb_perf_context_key_lock_wait_count.Collect(ch,
				float64(value),
				[]string{
					schema,
					table,
					part,
				})
		}
		if stat == "IO_BYTES_WRITTEN" {
			qd.infoSchema_rocksdb_perf_context_io_bytes_written.Collect(ch,
				float64(value),
				[]string{
					schema,
					table,
					part,
				})
		}
		if stat == "IO_BYTES_READ" {
			qd.infoSchema_rocksdb_perf_context_io_bytes_read.Collect(ch,
				float64(value),
				[]string{
					schema,
					table,
					part,
				})
		}
	}
}

func NewScrapeRocksDBPerfContext() *ScrapeRocksDBPerfContext {
	return &ScrapeRocksDBPerfContext{
		//instance: instance,
		infoSchema_rocksdb_perf_context_user_key_comparison_count:     *NewinfoSchemarocksdb_perf_context_user_key_comparison_count(),
		infoSchema_rocksdb_perf_context_block_cache_hit_count:         *NewinfoSchemarocksdb_perf_context_block_cache_hit_count(),
		infoSchema_rocksdb_perf_context_block_read_count:              *NewinfoSchemarocksdb_perf_context_block_read_count(),
		infoSchema_rocksdb_perf_context_block_read_byte:               *NewinfoSchemarocksdb_perf_context_block_read_byte(),
		infoSchema_rocksdb_perf_context_get_read_bytes:                *NewinfoSchemarocksdb_perf_context_get_read_bytes(),
		infoSchema_rocksdb_perf_context_multiget_read_bytes:           *NewinfoSchemarocksdb_perf_context_multiget_read_bytes(),
		infoSchema_rocksdb_perf_context_iter_read_bytes:               *NewinfoSchemarocksdb_perf_context_iter_read_bytes(),
		infoSchema_rocksdb_perf_context_internal_key_skipped_count:    *NewinfoSchemarocksdb_perf_context_internal_key_skipped_count(),
		infoSchema_rocksdb_perf_context_internal_delete_skipped_count: *NewinfoSchema_rocksdb_perf_context_internal_delete_skipped_count(),
		infoSchema_rocksdb_perf_context_internal_recent_skipped_count: *NewinfoSchema_rocksdb_perf_context_internal_recent_skipped_count(),
		infoSchema_rocksdb_perf_context_internal_merge_count:          *NewinfoSchema_rocksdb_perf_context_internal_merge_count(),
		infoSchema_rocksdb_perf_context_get_from_memtable_count:       *NewinfoSchema_rocksdb_perf_context_get_from_memtable_count(),
		infoSchema_rocksdb_perf_context_seek_on_memtable_count:        *NewinfoSchema_rocksdb_perf_context_seek_on_memtable_count(),
		infoSchema_rocksdb_perf_context_next_on_memtable_count:        *NewinfoSchema_rocksdb_perf_context_next_on_memtable_count(),
		infoSchema_rocksdb_perf_context_prev_on_memtable_count:        *NewinfoSchema_rocksdb_perf_context_prev_on_memtable_count(),
		infoSchema_rocksdb_perf_context_seek_child_seek_count:         *NewinfoSchema_rocksdb_perf_context_seek_child_seek_count(),
		infoSchema_rocksdb_perf_context_bloom_memtable_hit_count:      *NewinfoSchema_rocksdb_perf_context_bloom_memtable_hit_count(),
		infoSchema_rocksdb_perf_context_bloom_memtable_miss_count:     *NewinfoSchema_rocksdb_perf_context_bloom_memtable_miss_count(),
		infoSchema_rocksdb_perf_context_bloom_sst_hit_count:           *NewinfoSchema_rocksdb_perf_context_bloom_sst_hit_count(),
		infoSchema_rocksdb_perf_context_bloom_sst_miss_count:          *NewinfoSchema_rocksdb_perf_context_bloom_sst_miss_count(),
		infoSchema_rocksdb_perf_context_key_lock_wait_count:           *NewinfoSchema_rocksdb_perf_context_key_lock_wait_count(),
		infoSchema_rocksdb_perf_context_io_bytes_written:              *NewinfoSchema_rocksdb_perf_context_io_bytes_written(),
		infoSchema_rocksdb_perf_context_io_bytes_read:                 *NewinfoSchema_rocksdb_perf_context_io_bytes_read(),
	}
}

type infoSchema_rocksdb_perf_context_user_key_comparison_count struct {
	*baseMetrics
}

func NewinfoSchemarocksdb_perf_context_user_key_comparison_count() *infoSchema_rocksdb_perf_context_user_key_comparison_count {
	return &infoSchema_rocksdb_perf_context_user_key_comparison_count{
		NewMetrics(
			"info_schema_rocksdb_perf_context_user_key_comparison_count",
			"Total number of user key comparisons performed in binary search.",
			[]string{
				"schema",
				"table",
				"part"})}
}

func (qd *infoSchema_rocksdb_perf_context_user_key_comparison_count) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchema_rocksdb_perf_context_block_cache_hit_count struct {
	*baseMetrics
}

func NewinfoSchemarocksdb_perf_context_block_cache_hit_count() *infoSchema_rocksdb_perf_context_block_cache_hit_count {
	return &infoSchema_rocksdb_perf_context_block_cache_hit_count{
		NewMetrics(
			"rocksdb_perf_context_block_cache_hit_count",
			"Total number of times cache entry was found in block cache.",
			[]string{
				"schema",
				"table",
				"part"})}
}
func (qd *infoSchema_rocksdb_perf_context_block_cache_hit_count) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchema_rocksdb_perf_context_block_read_count struct {
	*baseMetrics
}

func NewinfoSchemarocksdb_perf_context_block_read_count() *infoSchema_rocksdb_perf_context_block_read_count {
	return &infoSchema_rocksdb_perf_context_block_read_count{
		NewMetrics(
			"rocksdb_perf_context_block_read_count",
			"Total number of times a block has been read.",
			[]string{
				"schema",
				"table",
				"part"})}
}
func (qd *infoSchema_rocksdb_perf_context_block_read_count) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchema_rocksdb_perf_context_block_read_byte struct {
	*baseMetrics
}

func NewinfoSchemarocksdb_perf_context_block_read_byte() *infoSchema_rocksdb_perf_context_block_read_byte {
	return &infoSchema_rocksdb_perf_context_block_read_byte{
		NewMetrics(
			"rocksdb_perf_context_block_read_byte",
			"Total number of bytes read from block cache and/or disk.",
			[]string{
				"schema",
				"table",
				"part"})}
}

func (qd *infoSchema_rocksdb_perf_context_block_read_byte) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchema_rocksdb_perf_context_get_read_bytes struct {
	*baseMetrics
}

func NewinfoSchemarocksdb_perf_context_get_read_bytes() *infoSchema_rocksdb_perf_context_get_read_bytes {
	return &infoSchema_rocksdb_perf_context_get_read_bytes{
		NewMetrics(
			"rocksdb_perf_context_get_read_bytes",
			"Total number of bytes read during get operations.",
			[]string{
				"schema",
				"table",
				"part"})}
}
func (qd *infoSchema_rocksdb_perf_context_get_read_bytes) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchema_rocksdb_perf_context_multiget_read_bytes struct {
	*baseMetrics
}

func NewinfoSchemarocksdb_perf_context_multiget_read_bytes() *infoSchema_rocksdb_perf_context_multiget_read_bytes {
	return &infoSchema_rocksdb_perf_context_multiget_read_bytes{
		NewMetrics(
			"rocksdb_perf_context_multiget_read_bytes",
			"Total number of bytes read during multiget operations.",
			[]string{
				"schema",
				"table",
				"part"})}
}
func (qd *infoSchema_rocksdb_perf_context_multiget_read_bytes) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchema_rocksdb_perf_context_iter_read_bytes struct {
	*baseMetrics
}

func NewinfoSchemarocksdb_perf_context_iter_read_bytes() *infoSchema_rocksdb_perf_context_iter_read_bytes {
	return &infoSchema_rocksdb_perf_context_iter_read_bytes{
		NewMetrics(
			"rocksdb_perf_context_iter_read_bytes",
			"Total number of bytes read during iterator operations.",
			[]string{
				"schema",
				"table",
				"part"})}
}
func (qd *infoSchema_rocksdb_perf_context_iter_read_bytes) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchema_rocksdb_perf_context_internal_key_skipped_count struct {
	*baseMetrics
}

func NewinfoSchemarocksdb_perf_context_internal_key_skipped_count() *infoSchema_rocksdb_perf_context_internal_key_skipped_count {
	return &infoSchema_rocksdb_perf_context_internal_key_skipped_count{
		NewMetrics(
			"rocksdb_perf_context_internal_key_skipped_count",
			"Total number of internal keys skipped during iteration.",
			[]string{
				"schema",
				"table",
				"part"})}

}

func (qd *infoSchema_rocksdb_perf_context_internal_key_skipped_count) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchema_rocksdb_perf_context_internal_delete_skipped_count struct {
	*baseMetrics
}

func NewinfoSchema_rocksdb_perf_context_internal_delete_skipped_count() *infoSchema_rocksdb_perf_context_internal_delete_skipped_count {
	return &infoSchema_rocksdb_perf_context_internal_delete_skipped_count{
		NewMetrics(
			"rocksdb_perf_context_internal_delete_skipped_count",
			"Count of internal delete operations that were skipped.",
			[]string{
				"schema",
				"table",
				"part"})}

}

func (qd *infoSchema_rocksdb_perf_context_internal_delete_skipped_count) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchema_rocksdb_perf_context_internal_recent_skipped_count struct {
	*baseMetrics
}

func NewinfoSchema_rocksdb_perf_context_internal_recent_skipped_count() *infoSchema_rocksdb_perf_context_internal_recent_skipped_count {
	return &infoSchema_rocksdb_perf_context_internal_recent_skipped_count{
		NewMetrics(
			"rocksdb_perf_context_internal_recent_skipped_count",
			"Count of recently skipped internal operations.",
			[]string{
				"schema",
				"table",
				"part"})}

}

func (qd *infoSchema_rocksdb_perf_context_internal_recent_skipped_count) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchema_rocksdb_perf_context_internal_merge_count struct {
	*baseMetrics
}

func NewinfoSchema_rocksdb_perf_context_internal_merge_count() *infoSchema_rocksdb_perf_context_internal_merge_count {
	return &infoSchema_rocksdb_perf_context_internal_merge_count{
		NewMetrics(
			"rocksdb_perf_context_internal_merge_count",
			"Count of internal merge operations.",
			[]string{
				"schema",
				"table",
				"part"})}

}

func (qd *infoSchema_rocksdb_perf_context_internal_merge_count) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchema_rocksdb_perf_context_get_from_memtable_count struct {
	*baseMetrics
}

func NewinfoSchema_rocksdb_perf_context_get_from_memtable_count() *infoSchema_rocksdb_perf_context_get_from_memtable_count {
	return &infoSchema_rocksdb_perf_context_get_from_memtable_count{
		NewMetrics(
			"rocksdb_perf_context_get_from_memtable_count",
			"Count of get operations that found entries in memtables.",
			[]string{
				"schema",
				"table",
				"part"})}
}

func (qd *infoSchema_rocksdb_perf_context_get_from_memtable_count) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchema_rocksdb_perf_context_seek_on_memtable_count struct {
	*baseMetrics
}

func NewinfoSchema_rocksdb_perf_context_seek_on_memtable_count() *infoSchema_rocksdb_perf_context_seek_on_memtable_count {
	return &infoSchema_rocksdb_perf_context_seek_on_memtable_count{
		NewMetrics(
			"rocksdb_perf_context_seek_on_memtable_count",
			"Count of seek operations that found entries in memtables.",
			[]string{
				"schema",
				"table",
				"part"})}
}
func (qd *infoSchema_rocksdb_perf_context_seek_on_memtable_count) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchema_rocksdb_perf_context_next_on_memtable_count struct {
	*baseMetrics
}

func NewinfoSchema_rocksdb_perf_context_next_on_memtable_count() *infoSchema_rocksdb_perf_context_next_on_memtable_count {
	return &infoSchema_rocksdb_perf_context_next_on_memtable_count{
		NewMetrics(
			"rocksdb_perf_context_next_on_memtable_count",
			"Count of next operations that found entries in memtables.",
			[]string{
				"schema",
				"table",
				"part"})}
}
func (qd *infoSchema_rocksdb_perf_context_next_on_memtable_count) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchema_rocksdb_perf_context_prev_on_memtable_count struct {
	*baseMetrics
}

func NewinfoSchema_rocksdb_perf_context_prev_on_memtable_count() *infoSchema_rocksdb_perf_context_prev_on_memtable_count {
	return &infoSchema_rocksdb_perf_context_prev_on_memtable_count{
		NewMetrics(
			"rocksdb_perf_context_prev_on_memtable_count",
			"Count of prev operations that found entries in memtables.",
			[]string{
				"schema",
				"table",
				"part"})}

}

func (qd *infoSchema_rocksdb_perf_context_prev_on_memtable_count) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchema_rocksdb_perf_context_seek_child_seek_count struct {
	*baseMetrics
}

func NewinfoSchema_rocksdb_perf_context_seek_child_seek_count() *infoSchema_rocksdb_perf_context_seek_child_seek_count {
	return &infoSchema_rocksdb_perf_context_seek_child_seek_count{
		NewMetrics(
			"rocksdb_perf_context_seek_child_seek_count",
			"Count of seek operations that found entries in memtables.",
			[]string{
				"schema",
				"table",
				"part"})}
}

func (qd *infoSchema_rocksdb_perf_context_seek_child_seek_count) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchema_rocksdb_perf_context_bloom_memtable_hit_count struct {
	*baseMetrics
}

func NewinfoSchema_rocksdb_perf_context_bloom_memtable_hit_count() *infoSchema_rocksdb_perf_context_bloom_memtable_hit_count {
	return &infoSchema_rocksdb_perf_context_bloom_memtable_hit_count{
		NewMetrics(
			"rocksdb_perf_context_bloom_memtable_hit_count",
			"Count of bloom filter hits in memtables.",
			[]string{
				"schema",
				"table",
				"part"})}
}

func (qd *infoSchema_rocksdb_perf_context_bloom_memtable_hit_count) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchema_rocksdb_perf_context_bloom_memtable_miss_count struct {
	*baseMetrics
}

func NewinfoSchema_rocksdb_perf_context_bloom_memtable_miss_count() *infoSchema_rocksdb_perf_context_bloom_memtable_miss_count {
	return &infoSchema_rocksdb_perf_context_bloom_memtable_miss_count{
		NewMetrics(
			"rocksdb_perf_context_bloom_memtable_miss_count",
			"Count of bloom filter misses in memtables.",
			[]string{
				"schema",
				"table",
				"part"})}
}

func (qd *infoSchema_rocksdb_perf_context_bloom_memtable_miss_count) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchema_rocksdb_perf_context_bloom_sst_hit_count struct {
	*baseMetrics
}

func NewinfoSchema_rocksdb_perf_context_bloom_sst_hit_count() *infoSchema_rocksdb_perf_context_bloom_sst_hit_count {
	return &infoSchema_rocksdb_perf_context_bloom_sst_hit_count{
		NewMetrics(
			"rocksdb_perf_context_bloom_sst_hit_count",
			"Count of bloom filter hits in SST files.",
			[]string{
				"schema",
				"table",
				"part"})}
}

func (qd *infoSchema_rocksdb_perf_context_bloom_sst_hit_count) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchema_rocksdb_perf_context_bloom_sst_miss_count struct {
	*baseMetrics
}

func NewinfoSchema_rocksdb_perf_context_bloom_sst_miss_count() *infoSchema_rocksdb_perf_context_bloom_sst_miss_count {
	return &infoSchema_rocksdb_perf_context_bloom_sst_miss_count{
		NewMetrics(
			"rocksdb_perf_context_bloom_sst_miss_count",
			"Count of bloom filter misses in SST files.",
			[]string{
				"schema",
				"table",
				"part"})}
}

func (qd *infoSchema_rocksdb_perf_context_bloom_sst_miss_count) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchema_rocksdb_perf_context_key_lock_wait_count struct {
	*baseMetrics
}

func NewinfoSchema_rocksdb_perf_context_key_lock_wait_count() *infoSchema_rocksdb_perf_context_key_lock_wait_count {
	return &infoSchema_rocksdb_perf_context_key_lock_wait_count{
		NewMetrics(
			"rocksdb_perf_context_key_lock_wait_count",
			"Count of lock wait.",
			[]string{
				"schema",
				"table",
				"part"})}
}
func (qd *infoSchema_rocksdb_perf_context_key_lock_wait_count) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchema_rocksdb_perf_context_io_bytes_written struct {
	*baseMetrics
}

func NewinfoSchema_rocksdb_perf_context_io_bytes_written() *infoSchema_rocksdb_perf_context_io_bytes_written {
	return &infoSchema_rocksdb_perf_context_io_bytes_written{
		NewMetrics(
			"rocksdb_perf_context_io_bytes_written",
			"Count of bytes written to storage.",
			[]string{
				"schema",
				"table",
				"part"})}
}
func (qd *infoSchema_rocksdb_perf_context_io_bytes_written) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchema_rocksdb_perf_context_io_bytes_read struct {
	*baseMetrics
}

func NewinfoSchema_rocksdb_perf_context_io_bytes_read() *infoSchema_rocksdb_perf_context_io_bytes_read {
	return &infoSchema_rocksdb_perf_context_io_bytes_read{
		NewMetrics(
			"rocksdb_perf_context_io_bytes_read",
			"Count of bytes read from storage.",
			[]string{
				"schema",
				"table",
				"part"})}
}
func (qd *infoSchema_rocksdb_perf_context_io_bytes_read) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

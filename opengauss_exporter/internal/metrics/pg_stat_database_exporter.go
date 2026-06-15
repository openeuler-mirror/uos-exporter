package metrics

import (
	"database/sql"

	"opengauss_exporter/internal/collector"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type PgStatDatabaseExporter struct {
	db       *sql.DB
	instance string
	uri      string
	metrics  *pgStatDatabaseMetrics
}

type pgStatDatabaseMetrics struct {
	// 连接统计指标
	numBackendsMetric *baseMetrics

	// 事务统计指标
	xactCommitMetric   *baseMetrics
	xactRollbackMetric *baseMetrics

	// 磁盘I/O统计指标
	blksReadMetric *baseMetrics
	blksHitMetric  *baseMetrics

	// 元组操作统计指标
	tupReturnedMetric *baseMetrics
	tupFetchedMetric  *baseMetrics
	tupInsertedMetric *baseMetrics
	tupUpdatedMetric  *baseMetrics
	tupDeletedMetric  *baseMetrics

	// 冲突统计指标
	conflictsMetric *baseMetrics

	// 临时文件统计指标
	tempFilesMetric *baseMetrics
	tempBytesMetric *baseMetrics

	// 死锁统计指标
	deadlocksMetric *baseMetrics

	// 统计重置时间指标
	statsResetMetric *baseMetrics
}

func newPgStatDatabaseMetrics() *pgStatDatabaseMetrics {
	return &pgStatDatabaseMetrics{
		numBackendsMetric: NewMetrics(
			"pg_stat_database_numbackends",
			"Number of backends currently connected to this database",
			[]string{"instance", "uri", "database"},
		),
		xactCommitMetric: NewMetrics(
			"pg_stat_database_xact_commit_total",
			"Number of transactions in this database that have been committed",
			[]string{"instance", "uri", "database"},
		),
		xactRollbackMetric: NewMetrics(
			"pg_stat_database_xact_rollback_total",
			"Number of transactions in this database that have been rolled back",
			[]string{"instance", "uri", "database"},
		),
		blksReadMetric: NewMetrics(
			"pg_stat_database_blks_read_total",
			"Number of disk blocks read in this database",
			[]string{"instance", "uri", "database"},
		),
		blksHitMetric: NewMetrics(
			"pg_stat_database_blks_hit_total",
			"Number of times disk blocks were found already in the buffer cache",
			[]string{"instance", "uri", "database"},
		),
		tupReturnedMetric: NewMetrics(
			"pg_stat_database_tup_returned_total",
			"Number of rows returned by queries in this database",
			[]string{"instance", "uri", "database"},
		),
		tupFetchedMetric: NewMetrics(
			"pg_stat_database_tup_fetched_total",
			"Number of rows fetched by queries in this database",
			[]string{"instance", "uri", "database"},
		),
		tupInsertedMetric: NewMetrics(
			"pg_stat_database_tup_inserted_total",
			"Number of rows inserted by queries in this database",
			[]string{"instance", "uri", "database"},
		),
		tupUpdatedMetric: NewMetrics(
			"pg_stat_database_tup_updated_total",
			"Number of rows updated by queries in this database",
			[]string{"instance", "uri", "database"},
		),
		tupDeletedMetric: NewMetrics(
			"pg_stat_database_tup_deleted_total",
			"Number of rows deleted by queries in this database",
			[]string{"instance", "uri", "database"},
		),
		conflictsMetric: NewMetrics(
			"pg_stat_database_conflicts_total",
			"Number of queries canceled due to conflicts with recovery in this database",
			[]string{"instance", "uri", "database"},
		),
		tempFilesMetric: NewMetrics(
			"pg_stat_database_temp_files_total",
			"Number of temporary files created by queries in this database",
			[]string{"instance", "uri", "database"},
		),
		tempBytesMetric: NewMetrics(
			"pg_stat_database_temp_bytes_total",
			"Total amount of data written to temporary files by queries in this database",
			[]string{"instance", "uri", "database"},
		),
		deadlocksMetric: NewMetrics(
			"pg_stat_database_deadlocks_total",
			"Number of deadlocks detected in this database",
			[]string{"instance", "uri", "database"},
		),
		statsResetMetric: NewMetrics(
			"pg_stat_database_stats_reset_time",
			"Time at which these statistics were last reset for this database",
			[]string{"instance", "uri", "database"},
		),
	}
}

// NewPgStatDatabaseExporter 创建一个新的 PgStatDatabase Exporter
func NewPgStatDatabaseExporter(db *sql.DB, instance, uri string) *PgStatDatabaseExporter {
	return &PgStatDatabaseExporter{
		db:       db,
		instance: instance,
		uri:      uri,
		metrics:  newPgStatDatabaseMetrics(),
	}
}

// Describe implements Prometheus Collector interface
func (e *PgStatDatabaseExporter) Describe(descs chan<- *prometheus.Desc) {
	// This is a no-op implementation as we use ConstMetrics
}

// Collect implements Prometheus Collector interface and Metric interface
func (e *PgStatDatabaseExporter) Collect(ch chan<- prometheus.Metric) {
	baseLabels := []string{e.instance, e.uri}

	collection, err := collector.ScrapePgStatDatabase(e.db)
	if err != nil {
		logrus.Debugf("Failed to scrape pg_stat_database: %v", err)
		return
	}

	// 为每个数据库导出指标
	for dbName, dbStat := range collection.Databases {
		labels := append(baseLabels, dbName)

		// 导出连接统计指标
		e.metrics.numBackendsMetric.collect(
			ch,
			float64(dbStat.NumBackends),
			labels,
		)

		// 导出事务统计指标
		e.metrics.xactCommitMetric.collect(
			ch,
			float64(dbStat.XactCommit),
			labels,
		)
		e.metrics.xactRollbackMetric.collect(
			ch,
			float64(dbStat.XactRollback),
			labels,
		)

		// 导出磁盘I/O统计指标
		e.metrics.blksReadMetric.collect(
			ch,
			float64(dbStat.BlksRead),
			labels,
		)
		e.metrics.blksHitMetric.collect(
			ch,
			float64(dbStat.BlksHit),
			labels,
		)

		// 导出元组操作统计指标
		e.metrics.tupReturnedMetric.collect(
			ch,
			float64(dbStat.TupReturned),
			labels,
		)
		e.metrics.tupFetchedMetric.collect(
			ch,
			float64(dbStat.TupFetched),
			labels,
		)
		e.metrics.tupInsertedMetric.collect(
			ch,
			float64(dbStat.TupInserted),
			labels,
		)
		e.metrics.tupUpdatedMetric.collect(
			ch,
			float64(dbStat.TupUpdated),
			labels,
		)
		e.metrics.tupDeletedMetric.collect(
			ch,
			float64(dbStat.TupDeleted),
			labels,
		)

		// 导出冲突统计指标
		e.metrics.conflictsMetric.collect(
			ch,
			float64(dbStat.Conflicts),
			labels,
		)

		// 导出临时文件统计指标
		e.metrics.tempFilesMetric.collect(
			ch,
			float64(dbStat.TempFiles),
			labels,
		)
		e.metrics.tempBytesMetric.collect(
			ch,
			float64(dbStat.TempBytes),
			labels,
		)

		// 导出死锁统计指标
		e.metrics.deadlocksMetric.collect(
			ch,
			float64(dbStat.Deadlocks),
			labels,
		)

		// 导出统计重置时间指标
		if dbStat.StatsReset != nil {
			e.metrics.statsResetMetric.collect(
				ch,
				float64(dbStat.StatsReset.Unix()),
				labels,
			)
		} else {
			e.metrics.statsResetMetric.collect(
				ch,
				0,
				labels,
			)
		}
	}
}

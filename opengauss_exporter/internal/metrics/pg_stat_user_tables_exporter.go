package metrics

import (
	"database/sql"

	"opengauss_exporter/internal/collector"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type PgStatUserTablesExporter struct {
	db       *sql.DB
	instance string
	uri      string
	metrics  *pgStatUserTablesMetrics
}

type pgStatUserTablesMetrics struct {
	// 扫描统计指标
	seqScanMetric     *baseMetrics
	seqTupReadMetric  *baseMetrics
	idxScanMetric     *baseMetrics
	idxTupFetchMetric *baseMetrics

	// 元组变更统计指标
	nTupInsMetric    *baseMetrics
	nTupUpdMetric    *baseMetrics
	nTupDelMetric    *baseMetrics
	nTupHotUpdMetric *baseMetrics

	// 元组状态统计指标
	nLiveTupMetric         *baseMetrics
	nDeadTupMetric         *baseMetrics
	nModSinceAnalyzeMetric *baseMetrics

	// VACUUM统计指标
	vacuumCountMetric     *baseMetrics
	autovacuumCountMetric *baseMetrics
	lastVacuumMetric      *baseMetrics
	lastAutovacuumMetric  *baseMetrics

	// ANALYZE统计指标
	analyzeCountMetric     *baseMetrics
	autoanalyzeCountMetric *baseMetrics
	lastAnalyzeMetric      *baseMetrics
	lastAutoanalyzeMetric  *baseMetrics
}

func newPgStatUserTablesMetrics() *pgStatUserTablesMetrics {
	return &pgStatUserTablesMetrics{
		seqScanMetric: NewMetrics(
			"pg_stat_user_tables_seq_scan_total",
			"Number of sequential scans initiated on this table",
			[]string{"instance", "uri", "schema", "table"},
		),
		seqTupReadMetric: NewMetrics(
			"pg_stat_user_tables_seq_tup_read_total",
			"Number of live rows fetched by sequential scans",
			[]string{"instance", "uri", "schema", "table"},
		),
		idxScanMetric: NewMetrics(
			"pg_stat_user_tables_idx_scan_total",
			"Number of index scans initiated on this table",
			[]string{"instance", "uri", "schema", "table"},
		),
		idxTupFetchMetric: NewMetrics(
			"pg_stat_user_tables_idx_tup_fetch_total",
			"Number of live rows fetched by index scans",
			[]string{"instance", "uri", "schema", "table"},
		),
		nTupInsMetric: NewMetrics(
			"pg_stat_user_tables_n_tup_ins_total",
			"Number of rows inserted",
			[]string{"instance", "uri", "schema", "table"},
		),
		nTupUpdMetric: NewMetrics(
			"pg_stat_user_tables_n_tup_upd_total",
			"Number of rows updated",
			[]string{"instance", "uri", "schema", "table"},
		),
		nTupDelMetric: NewMetrics(
			"pg_stat_user_tables_n_tup_del_total",
			"Number of rows deleted",
			[]string{"instance", "uri", "schema", "table"},
		),
		nTupHotUpdMetric: NewMetrics(
			"pg_stat_user_tables_n_tup_hot_upd_total",
			"Number of rows HOT updated",
			[]string{"instance", "uri", "schema", "table"},
		),
		nLiveTupMetric: NewMetrics(
			"pg_stat_user_tables_n_live_tup",
			"Estimated number of live rows",
			[]string{"instance", "uri", "schema", "table"},
		),
		nDeadTupMetric: NewMetrics(
			"pg_stat_user_tables_n_dead_tup",
			"Estimated number of dead rows",
			[]string{"instance", "uri", "schema", "table"},
		),
		nModSinceAnalyzeMetric: NewMetrics(
			"pg_stat_user_tables_n_mod_since_analyze",
			"Estimated number of rows modified since this table was last analyzed",
			[]string{"instance", "uri", "schema", "table"},
		),
		vacuumCountMetric: NewMetrics(
			"pg_stat_user_tables_vacuum_count_total",
			"Number of times this table has been manually vacuumed",
			[]string{"instance", "uri", "schema", "table"},
		),
		autovacuumCountMetric: NewMetrics(
			"pg_stat_user_tables_autovacuum_count_total",
			"Number of times this table has been vacuumed by the autovacuum daemon",
			[]string{"instance", "uri", "schema", "table"},
		),
		lastVacuumMetric: NewMetrics(
			"pg_stat_user_tables_last_vacuum_time",
			"Last time at which this table was manually vacuumed",
			[]string{"instance", "uri", "schema", "table"},
		),
		lastAutovacuumMetric: NewMetrics(
			"pg_stat_user_tables_last_autovacuum_time",
			"Last time at which this table was vacuumed by the autovacuum daemon",
			[]string{"instance", "uri", "schema", "table"},
		),
		analyzeCountMetric: NewMetrics(
			"pg_stat_user_tables_analyze_count_total",
			"Number of times this table has been manually analyzed",
			[]string{"instance", "uri", "schema", "table"},
		),
		autoanalyzeCountMetric: NewMetrics(
			"pg_stat_user_tables_autoanalyze_count_total",
			"Number of times this table has been analyzed by the autovacuum daemon",
			[]string{"instance", "uri", "schema", "table"},
		),
		lastAnalyzeMetric: NewMetrics(
			"pg_stat_user_tables_last_analyze_time",
			"Last time at which this table was manually analyzed",
			[]string{"instance", "uri", "schema", "table"},
		),
		lastAutoanalyzeMetric: NewMetrics(
			"pg_stat_user_tables_last_autoanalyze_time",
			"Last time at which this table was analyzed by the autovacuum daemon",
			[]string{"instance", "uri", "schema", "table"},
		),
	}
}

// NewPgStatUserTablesExporter 创建一个新的 PgStatUserTables Exporter
func NewPgStatUserTablesExporter(db *sql.DB, instance, uri string) *PgStatUserTablesExporter {
	return &PgStatUserTablesExporter{
		db:       db,
		instance: instance,
		uri:      uri,
		metrics:  newPgStatUserTablesMetrics(),
	}
}

// Describe implements Prometheus Collector interface
func (e *PgStatUserTablesExporter) Describe(descs chan<- *prometheus.Desc) {
	// This is a no-op implementation as we use ConstMetrics
}

// Collect implements Prometheus Collector interface and Metric interface
func (e *PgStatUserTablesExporter) Collect(ch chan<- prometheus.Metric) {
	baseLabels := []string{e.instance, e.uri}

	collection, err := collector.ScrapePgStatUserTables(e.db)
	if err != nil {
		logrus.Debugf("Failed to scrape pg_stat_user_tables: %v", err)
		return
	}

	if len(collection.Tables) == 0 {
		logrus.Debug("No user tables found in pg_stat_user_tables")
		return
	}

	logrus.Debugf("Exporting metrics for %d user tables", len(collection.Tables))
	var metricsCount int

	// 为每个表导出指标
	for _, tableStat := range collection.Tables {
		labels := append(baseLabels, tableStat.SchemaName, tableStat.TableName)

		// 导出扫描统计指标
		e.metrics.seqScanMetric.collect(
			ch,
			float64(tableStat.SeqScan),
			labels,
		)
		e.metrics.seqTupReadMetric.collect(
			ch,
			float64(tableStat.SeqTupRead),
			labels,
		)
		e.metrics.idxScanMetric.collect(
			ch,
			float64(tableStat.IdxScan),
			labels,
		)
		e.metrics.idxTupFetchMetric.collect(
			ch,
			float64(tableStat.IdxTupFetch),
			labels,
		)

		// 导出元组变更统计指标
		e.metrics.nTupInsMetric.collect(
			ch,
			float64(tableStat.NTupIns),
			labels,
		)
		e.metrics.nTupUpdMetric.collect(
			ch,
			float64(tableStat.NTupUpd),
			labels,
		)
		e.metrics.nTupDelMetric.collect(
			ch,
			float64(tableStat.NTupDel),
			labels,
		)
		e.metrics.nTupHotUpdMetric.collect(
			ch,
			float64(tableStat.NTupHotUpd),
			labels,
		)

		// 导出元组状态统计指标
		e.metrics.nLiveTupMetric.collect(
			ch,
			float64(tableStat.NLiveTup),
			labels,
		)
		e.metrics.nDeadTupMetric.collect(
			ch,
			float64(tableStat.NDeadTup),
			labels,
		)
		e.metrics.nModSinceAnalyzeMetric.collect(
			ch,
			float64(tableStat.NModSinceAnalyze),
			labels,
		)

		// 导出VACUUM统计指标
		e.metrics.vacuumCountMetric.collect(
			ch,
			float64(tableStat.VacuumCount),
			labels,
		)
		e.metrics.autovacuumCountMetric.collect(
			ch,
			float64(tableStat.AutovacuumCount),
			labels,
		)
		if tableStat.LastVacuum != nil {
			e.metrics.lastVacuumMetric.collect(
				ch,
				float64(tableStat.LastVacuum.Unix()),
				labels,
			)
		} else {
			e.metrics.lastVacuumMetric.collect(
				ch,
				0,
				labels,
			)
		}
		if tableStat.LastAutovacuum != nil {
			e.metrics.lastAutovacuumMetric.collect(
				ch,
				float64(tableStat.LastAutovacuum.Unix()),
				labels,
			)
		} else {
			e.metrics.lastAutovacuumMetric.collect(
				ch,
				0,
				labels,
			)
		}

		// 导出ANALYZE统计指标
		e.metrics.analyzeCountMetric.collect(
			ch,
			float64(tableStat.AnalyzeCount),
			labels,
		)
		e.metrics.autoanalyzeCountMetric.collect(
			ch,
			float64(tableStat.AutoanalyzeCount),
			labels,
		)
		if tableStat.LastAnalyze != nil {
			e.metrics.lastAnalyzeMetric.collect(
				ch,
				float64(tableStat.LastAnalyze.Unix()),
				labels,
			)
		} else {
			e.metrics.lastAnalyzeMetric.collect(
				ch,
				0,
				labels,
			)
		}
		if tableStat.LastAutoanalyze != nil {
			e.metrics.lastAutoanalyzeMetric.collect(
				ch,
				float64(tableStat.LastAutoanalyze.Unix()),
				labels,
			)
		} else {
			e.metrics.lastAutoanalyzeMetric.collect(
				ch,
				0,
				labels,
			)
		}

		metricsCount += 18 // 每个表导出18个指标
	}

	logrus.Debugf("Successfully exported %d pg_stat_user_tables metrics for %d tables", metricsCount, len(collection.Tables))
}

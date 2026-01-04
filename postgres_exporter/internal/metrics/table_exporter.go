package metrics

import (
	"database/sql"

	"postgres_exporter/internal/collector"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type PostgreSQLTableExporter struct {
	metrics *tableMetrics
	db      *sql.DB
	uri     string
	dbName  string
}

type tableMetrics struct {
	seqScansMetric    *baseMetrics
	seqTupReadMetric  *baseMetrics
	idxScansMetric    *baseMetrics
	idxTupFetchMetric *baseMetrics

	insertsMetric    *baseMetrics
	updatesMetric    *baseMetrics
	deletesMetric    *baseMetrics
	hotUpdatesMetric *baseMetrics

	liveRowsMetric        *baseMetrics
	deadRowsMetric        *baseMetrics
	modSinceAnalyzeMetric *baseMetrics
	insSinceVacuumMetric  *baseMetrics

	lastVacuumTimestampMetric      *baseMetrics
	lastAutoVacuumTimestampMetric  *baseMetrics
	lastAnalyzeTimestampMetric     *baseMetrics
	lastAutoAnalyzeTimestampMetric *baseMetrics

	vacuumCountMetric      *baseMetrics
	autoVacuumCountMetric  *baseMetrics
	analyzeCountMetric     *baseMetrics
	autoAnalyzeCountMetric *baseMetrics
}

func newTableMetrics() *tableMetrics {
	return &tableMetrics{
		seqScansMetric: NewMetrics(
			"pg_table_seq_scans_total",
			"The number of sequential scans performed on this table.",
			[]string{"instance", "uri", "db", "schema", "table"},
		),
		seqTupReadMetric: NewMetrics(
			"pg_table_seq_tup_read_total",
			"The number of rows fetched via sequential scans.",
			[]string{"instance", "uri", "db", "schema", "table"},
		),
		idxScansMetric: NewMetrics(
			"pg_table_idx_scans_total",
			"The number of index scans performed on this table.",
			[]string{"instance", "uri", "db", "schema", "table"},
		),
		idxTupFetchMetric: NewMetrics(
			"pg_table_idx_tup_fetch_total",
			"The number of rows fetched via index scans.",
			[]string{"instance", "uri", "db", "schema", "table"},
		),

		insertsMetric: NewMetrics(
			"pg_table_n_tup_ins_total",
			"The number of inserted tuples.",
			[]string{"instance", "uri", "db", "schema", "table"},
		),
		updatesMetric: NewMetrics(
			"pg_table_n_tup_upd_total",
			"The number of updated tuples.",
			[]string{"instance", "uri", "db", "schema", "table"},
		),
		deletesMetric: NewMetrics(
			"pg_table_n_tup_del_total",
			"The number of deleted tuples.",
			[]string{"instance", "uri", "db", "schema", "table"},
		),
		hotUpdatesMetric: NewMetrics(
			"pg_table_n_tup_hot_upd_total",
			"The number of HOT updates.",
			[]string{"instance", "uri", "db", "schema", "table"},
		),

		liveRowsMetric: NewMetrics(
			"pg_table_rows_live",
			"The number of live tuples in the table.",
			[]string{"instance", "uri", "db", "schema", "table"},
		),
		deadRowsMetric: NewMetrics(
			"pg_table_rows_dead",
			"The number of dead tuples in the table.",
			[]string{"instance", "uri", "db", "schema", "table"},
		),
		modSinceAnalyzeMetric: NewMetrics(
			"pg_table_n_mod_since_analyze",
			"The number of rows modified since last analyze.",
			[]string{"instance", "uri", "db", "schema", "table"},
		),
		insSinceVacuumMetric: NewMetrics(
			"pg_table_n_ins_since_vacuum",
			"The number of rows inserted since last vacuum.",
			[]string{"instance", "uri", "db", "schema", "table"},
		),
		lastVacuumTimestampMetric: NewMetrics(
			"pg_table_last_vacuum_timestamp",
			"The timestamp of the last manual vacuum.",
			[]string{"instance", "uri", "db", "schema", "table"},
		),
		lastAutoVacuumTimestampMetric: NewMetrics(
			"pg_table_last_autovacuum_timestamp",
			"The timestamp of the last autovacuum.",
			[]string{"instance", "uri", "db", "schema", "table"},
		),
		lastAnalyzeTimestampMetric: NewMetrics(
			"pg_table_last_analyze_timestamp",
			"The timestamp of the last manual analyze.",
			[]string{"instance", "uri", "db", "schema", "table"},
		),
		lastAutoAnalyzeTimestampMetric: NewMetrics(
			"pg_table_last_autoanalyze_timestamp",
			"The timestamp of the last auto-analyze.",
			[]string{"instance", "uri", "db", "schema", "table"},
		),
		vacuumCountMetric: NewMetrics(
			"pg_table_vacuum_count",
			"The number of manual vacuums executed.",
			[]string{"instance", "uri", "db", "schema", "table"},
		),
		autoVacuumCountMetric: NewMetrics(
			"pg_table_autovacuum_count",
			"The number of autovacuum operations.",
			[]string{"instance", "uri", "db", "schema", "table"},
		),
		analyzeCountMetric: NewMetrics(
			"pg_table_analyze_count",
			"The number of manual analyzes.",
			[]string{"instance", "uri", "db", "schema", "table"},
		),
		autoAnalyzeCountMetric: NewMetrics(
			"pg_table_autoanalyze_count",
			"The number of autoanalyzes.",
			[]string{"instance", "uri", "db", "schema", "table"},
		),
	}
}

// NewPostgreSQLTableExporter 创建一个新的 Table Exporter
func NewPostgreSQLTableExporter(db *sql.DB, uri, dbName string) *PostgreSQLTableExporter {
	return &PostgreSQLTableExporter{
		metrics: newTableMetrics(),
		db:      db,
		uri:     uri,
		dbName:  dbName,
	}
}

// Describe implements Prometheus Collector interface

// Collect implements Prometheus Collector interface
func (e *PostgreSQLTableExporter) collect(ch chan<- prometheus.Metric) {
	tableStats, err := collector.ScrapePostgreSQLTables(e.db)
	if err != nil || len(tableStats.Tables) == 0 {
		logrus.Debugf("Failed to scrape PostgreSQL table stats: %v", err)
		logrus.Debugf("Table stats: %v", tableStats)
		return
	}

	labelsBase := []string{"local", e.uri, e.dbName}

	for _, table := range tableStats.Tables {
		labels := append(labelsBase, table.Schema, table.Name)

		e.metrics.seqScansMetric.collect(
			ch,
			float64(table.SeqScans),
			labels,
		)
		e.metrics.seqTupReadMetric.collect(
			ch,
			float64(table.SeqTupRead),
			labels,
		)
		e.metrics.idxScansMetric.collect(
			ch,
			float64(table.IdxScans),
			labels,
		)
		e.metrics.idxTupFetchMetric.collect(
			ch,
			float64(table.IdxTupFetch),
			labels,
		)

		e.metrics.insertsMetric.collect(
			ch,
			float64(table.Inserts),
			labels,
		)
		e.metrics.updatesMetric.collect(
			ch,
			float64(table.Updates),
			labels,
		)
		e.metrics.deletesMetric.collect(
			ch,
			float64(table.Deletes),
			labels,
		)
		e.metrics.hotUpdatesMetric.collect(
			ch,
			float64(table.HotUpdates),
			labels,
		)

		e.metrics.liveRowsMetric.collect(
			ch,
			float64(table.LiveTuples),
			labels,
		)
		e.metrics.deadRowsMetric.collect(
			ch,
			float64(table.DeadTuples),
			labels,
		)
		e.metrics.modSinceAnalyzeMetric.collect(
			ch,
			float64(table.NModSinceAnalyze),
			labels,
		)
		e.metrics.insSinceVacuumMetric.collect(
			ch,
			float64(table.NInsSinceVacuum),
			labels,
		)

		e.metrics.vacuumCountMetric.collect(
			ch,
			float64(table.VacuumCount),
			labels,
		)
		e.metrics.autoVacuumCountMetric.collect(
			ch,
			float64(table.AutoVacuumCount),
			labels,
		)
		e.metrics.analyzeCountMetric.collect(
			ch,
			float64(table.AnalyzeCount),
			labels,
		)
		e.metrics.autoAnalyzeCountMetric.collect(
			ch,
			float64(table.AutoAnalyzeCount),
			labels,
		)

		// 上报时间戳
		if table.LastVacuum != nil {
			e.metrics.lastVacuumTimestampMetric.collect(
				ch,
				float64(table.LastVacuum.Unix()),
				labels,
			)
		}
		if table.LastAutoVacuum != nil {
			e.metrics.lastAutoVacuumTimestampMetric.collect(
				ch,
				float64(table.LastAutoVacuum.Unix()),
				labels,
			)
		}
		if table.LastAnalyze != nil {
			e.metrics.lastAnalyzeTimestampMetric.collect(
				ch,
				float64(table.LastAnalyze.Unix()),
				labels,
			)
		}
		if table.LastAutoAnalyze != nil {
			e.metrics.lastAutoAnalyzeTimestampMetric.collect(
				ch,
				float64(table.LastAutoAnalyze.Unix()),
				labels,
			)
		}
	}
}
// Part 2 commit for postgres_exporter/internal/metrics/table_exporter.go

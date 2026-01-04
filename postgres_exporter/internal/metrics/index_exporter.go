package metrics

import (
	"database/sql"

	"postgres_exporter/internal/collector"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type PostgreSQLIndexExporter struct {
	metrics *indexMetrics
	db      *sql.DB
	uri     string
	dbName  string
}

type indexMetrics struct {
	sizeBytesMetric             *baseMetrics
	scanCountMetric             *baseMetrics
	tupReadMetric               *baseMetrics
	tupFetchMetric              *baseMetrics
	isUniqueMetric              *baseMetrics
	typeMetric                  *baseMetrics
	lastAnalyzedTimestampMetric *baseMetrics
}

func newIndexMetrics() *indexMetrics {
	return &indexMetrics{
		sizeBytesMetric: NewMetrics(
			"pg_index_size_bytes",
			"The size of the index in bytes.",
			[]string{"instance", "uri", "db", "schema", "table", "index"},
		),
		scanCountMetric: NewMetrics(
			"pg_index_scans_total",
			"The number of scans on this index.",
			[]string{"instance", "uri", "db", "schema", "table", "index"},
		),
		tupReadMetric: NewMetrics(
			"pg_index_tup_read_total",
			"The number of tuples read via this index.",
			[]string{"instance", "uri", "db", "schema", "table", "index"},
		),
		tupFetchMetric: NewMetrics(
			"pg_index_tup_fetch_total",
			"The number of tuples fetched via this index.",
			[]string{"instance", "uri", "db", "schema", "table", "index"},
		),
		isUniqueMetric: NewMetrics(
			"pg_index_unique",
			"Gauge indicating if this index is unique (1) or not (0).",
			[]string{"instance", "uri", "db", "schema", "table", "index"},
		),
		typeMetric: NewMetrics(
			"pg_index_type",
			"Gauge indicating the type of the index.",
			[]string{"instance", "uri", "db", "schema", "table", "index", "type"},
		),
		lastAnalyzedTimestampMetric: NewMetrics(
			"pg_index_last_analyzed_timestamp",
			"The timestamp of the last analyze (manual or auto).",
			[]string{"instance", "uri", "db", "schema", "table", "index"},
		),
	}
}

// NewPostgreSQLIndexExporter 创建一个新的 Index Exporter
func NewPostgreSQLIndexExporter(db *sql.DB, uri, dbName string) *PostgreSQLIndexExporter {
	return &PostgreSQLIndexExporter{
		metrics: newIndexMetrics(),
		db:      db,
		uri:     uri,
		dbName:  dbName,
	}
}

// Describe implements Prometheus Collector interface

// Collect implements Prometheus Collector interface
func (e *PostgreSQLIndexExporter) collect(ch chan<- prometheus.Metric) {
	indexStats, err := collector.ScrapePostgreSQLIndexes(e.db)
	if err != nil || len(indexStats.Indices) == 0 {
		logrus.Debugf("Failed to scrape PostgreSQL index stats: %v", err)
		logrus.Debugf("Index stats: %v", indexStats)
		return
	}

	labelsBase := []string{"", e.uri, e.dbName, "public"} // schema 可以动态获取

	for _, idx := range indexStats.Indices {
		labels := append(labelsBase, idx.Table, idx.Name)

		e.metrics.sizeBytesMetric.collect(
			ch,
			float64(idx.SizeBytes),
			labels,
		)
		e.metrics.scanCountMetric.collect(
			ch,
			float64(idx.ScanCount),
			labels,
		)
		e.metrics.tupReadMetric.collect(
			ch,
			float64(idx.TupRead),
			labels,
		)
		e.metrics.tupFetchMetric.collect(
			ch,
			float64(idx.TupFetch),
			labels,
		)
		e.metrics.isUniqueMetric.collect(
			ch,
			boolToFloat64(idx.Unique),
			labels,
		)
		e.metrics.typeMetric.collect(
			ch,
			1, append(labels, idx.IndexType))

		if idx.LastAnalyzedTime != nil {
			e.metrics.lastAnalyzedTimestampMetric.collect(
				ch,
				float64(idx.LastAnalyzedTime.Unix()),
				labels,
			)
		}
	}
}

// 辅助函数：bool -> float64
func boolToFloat64(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
// Part 2 commit for postgres_exporter/internal/metrics/index_exporter.go

package metrics

import (
	"database/sql"

	"postgres_exporter/internal/collector"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type PostgreSQLBloatExporter struct {
	metrics *bloatMetrics
	db      *sql.DB
	uri     string
	dbName  string
}

type bloatMetrics struct {
	tableBloatRatioMetric *baseMetrics
	tableWastedMBMetric   *baseMetrics
	indexBloatRatioMetric *baseMetrics
	indexWastedMBMetric   *baseMetrics
}

func newBloatMetrics() *bloatMetrics {
	return &bloatMetrics{
		tableBloatRatioMetric: NewMetrics(
			"pg_table_bloat_ratio",
			"The ratio of actual size to ideal size for a table.",
			[]string{"instance", "uri", "db", "schema", "table"},
		),
		tableWastedMBMetric: NewMetrics(
			"pg_table_bloat_wasted_mb",
			"The amount of space wasted by table bloat in MB.",
			[]string{"instance", "uri", "db", "schema", "table"},
		),
		indexBloatRatioMetric: NewMetrics(
			"pg_index_bloat_ratio",
			"The ratio of actual size to ideal size for an index.",
			[]string{"instance", "uri", "db", "schema", "table", "index"},
		),
		indexWastedMBMetric: NewMetrics(
			"pg_index_bloat_wasted_mb",
			"The amount of space wasted by index bloat in MB.",
			[]string{"instance", "uri", "db", "schema", "table", "index"},
		),
	}
}

// NewPostgreSQLBloatExporter 创建一个新的 Bloat Exporter
func NewPostgreSQLBloatExporter(db *sql.DB, uri, dbName string) *PostgreSQLBloatExporter {
	return &PostgreSQLBloatExporter{
		metrics: newBloatMetrics(),
		db:      db,
		uri:     uri,
		dbName:  dbName,
	}
}

// Collect implements Prometheus Collector interface
func (e *PostgreSQLBloatExporter) collect(ch chan<- prometheus.Metric) {
	bloatStats, err := collector.ScrapePostgreSQLBloat(e.db)
	if err != nil {
		logrus.Debugf("Failed to scrape PostgreSQL bloat stats: %v", err)
		logrus.Debugf("Bloat stats: %v", bloatStats)
		return
	}

	labelsBase := []string{e.uri, e.dbName}

	// 上报表膨胀数据
	for _, table := range bloatStats.Tables {
		labels := append(labelsBase, table.Schema, table.Table)

		e.metrics.tableBloatRatioMetric.collect(
			ch,
			table.BloatRatio,
			labels,
		)
		e.metrics.tableWastedMBMetric.collect(
			ch,
			table.WastedMB,
			labels,
		)
	}

	// 上报索引膨胀数据
	for _, index := range bloatStats.Indices {
		labels := append(labelsBase, index.Schema, index.Table, index.Index)

		e.metrics.indexBloatRatioMetric.collect(
			ch,
			index.BloatRatio,
			labels,
		)
		e.metrics.indexWastedMBMetric.collect(
			ch,
			index.WastedMB,
			labels,
		)
	}
}
// Part 2 commit for postgres_exporter/internal/metrics/bloat_exporter.go

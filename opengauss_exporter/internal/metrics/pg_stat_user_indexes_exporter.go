package metrics

import (
	"database/sql"

	"opengauss_exporter/internal/collector"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type PgStatUserIndexesExporter struct {
	db       *sql.DB
	instance string
	uri      string
	metrics  *pgStatUserIndexesMetrics
}

type pgStatUserIndexesMetrics struct {
	idxScanMetric     *baseMetrics
	idxTupReadMetric  *baseMetrics
	idxTupFetchMetric *baseMetrics
}

func newPgStatUserIndexesMetrics() *pgStatUserIndexesMetrics {
	return &pgStatUserIndexesMetrics{
		idxScanMetric: NewMetrics(
			"pg_stat_user_indexes_idx_scan_total",
			"Number of index scans initiated on this index",
			[]string{"instance", "uri", "schema", "table", "index"},
		),
		idxTupReadMetric: NewMetrics(
			"pg_stat_user_indexes_idx_tup_read_total",
			"Number of index entries returned by scans on this index",
			[]string{"instance", "uri", "schema", "table", "index"},
		),
		idxTupFetchMetric: NewMetrics(
			"pg_stat_user_indexes_idx_tup_fetch_total",
			"Number of live table rows fetched by simple index scans using this index",
			[]string{"instance", "uri", "schema", "table", "index"},
		),
	}
}

func NewPgStatUserIndexesExporter(db *sql.DB, instance, uri string) *PgStatUserIndexesExporter {
	return &PgStatUserIndexesExporter{
		db:       db,
		instance: instance,
		uri:      uri,
		metrics:  newPgStatUserIndexesMetrics(),
	}
}

func (e *PgStatUserIndexesExporter) Describe(descs chan<- *prometheus.Desc) {}

func (e *PgStatUserIndexesExporter) Collect(ch chan<- prometheus.Metric) {
	baseLabels := []string{e.instance, e.uri}

	collection, err := collector.ScrapePgStatUserIndexes(e.db)
	if err != nil {
		logrus.Debugf("Failed to scrape pg_stat_user_indexes: %v", err)
		return
	}

	for _, indexStat := range collection.Indexes {
		labels := append(baseLabels, indexStat.SchemaName, indexStat.TableName, indexStat.IndexName)

		e.metrics.idxScanMetric.collect(ch, float64(indexStat.IdxScan), labels)
		e.metrics.idxTupReadMetric.collect(ch, float64(indexStat.IdxTupRead), labels)
		e.metrics.idxTupFetchMetric.collect(ch, float64(indexStat.IdxTupFetch), labels)
	}
}

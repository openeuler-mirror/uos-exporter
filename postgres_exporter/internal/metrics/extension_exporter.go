package metrics

import (
	"database/sql"

	"postgres_exporter/internal/collector"

	"github.com/prometheus/client_golang/prometheus"
)

type PostgreSQLExtensionExporter struct {
	metrics *extensionMetrics
	db      *sql.DB
	uri     string
	dbName  string
}

type extensionMetrics struct {
	installedMetric  *baseMetrics
	versionMetric    *baseMetrics
	schemaMetric     *baseMetrics
	requiredMetric   *baseMetrics
	totalCountMetric *baseMetrics
}

func newExtensionMetrics() *extensionMetrics {
	return &extensionMetrics{
		installedMetric: NewMetrics(
			"pg_extension_installed",
			"Gauge indicating if the extension is installed (1) or not (0).",
			[]string{"instance", "uri", "db", "extension"},
		),
		versionMetric: NewMetrics(
			"pg_extension_version",
			"Gauge indicating the version of the extension.",
			[]string{"instance", "uri", "db", "extension", "version"},
		),
		schemaMetric: NewMetrics(
			"pg_extension_schema",
			"Gauge indicating the schema where the extension was installed.",
			[]string{"instance", "uri", "db", "extension", "schema"},
		),
		requiredMetric: NewMetrics(
			"pg_extension_required",
			"Gauge indicating if this extension is required by other extensions (1) or not (0).",
			[]string{"instance", "uri", "db", "extension"},
		),
		totalCountMetric: NewMetrics(
			"pg_extensions_total",
			"The total number of extensions installed in this database.",
			[]string{"instance", "uri", "db"},
		),
	}
}

// NewPostgreSQLExtensionExporter 创建一个新的 Extension Exporter
func NewPostgreSQLExtensionExporter(db *sql.DB, uri, dbName string) *PostgreSQLExtensionExporter {
	return &PostgreSQLExtensionExporter{
		metrics: newExtensionMetrics(),
		db:      db,
		uri:     uri,
		dbName:  dbName,
	}
}

// Describe implements Prometheus Collector interface

// Collect implements Prometheus Collector interface
func (e *PostgreSQLExtensionExporter) collect(ch chan<- prometheus.Metric) {
	extStats, err := collector.ScrapePostgreSQLExtensions(e.db)
	if err != nil || len(extStats.Extensions) == 0 {
		return
	}

	labelsBase := []string{"local", e.uri, e.dbName}

	// 上报总扩展数
	e.metrics.totalCountMetric.collect(
		ch,
		float64(len(extStats.Extensions)),
		labelsBase,
	)

	// 遍历每个扩展上报指标
	for _, ext := range extStats.Extensions {
		labels := append(labelsBase, ext.Name)

		e.metrics.installedMetric.collect(
			ch,
			1,
			labels,
		)
		e.metrics.versionMetric.collect(
			ch,
			1, append(labels, ext.Version))
		e.metrics.schemaMetric.collect(
			ch,
			1, append(labels, ext.Schema))
		e.metrics.requiredMetric.collect(
			ch,
			boolToFloat64(ext.Required),
			labels,
		)
	}
}

package metrics

import (
	"database/sql"

	"postgres_exporter/internal/collector"

	"github.com/prometheus/client_golang/prometheus"
)

type PostgreSQLSchemaExporter struct {
	metrics *schemaMetrics
	db      *sql.DB
	uri     string
	dbName  string
}

type schemaMetrics struct {
	schemaCountMetric           *baseMetrics
	schemaSizeBytesMetric       *baseMetrics
	schemaOwnerMetric           *baseMetrics
	tableCountMetric            *baseMetrics
	viewCountMetric             *baseMetrics
	indexCountMetric            *baseMetrics
	lastModifiedTimestampMetric *baseMetrics
}

func newSchemaMetrics() *schemaMetrics {
	return &schemaMetrics{
		schemaCountMetric: NewMetrics(
			"pg_schemas_total",
			"The total number of schemas in the database.",
			[]string{"instance", "uri", "db"},
		),
		schemaSizeBytesMetric: NewMetrics(
			"pg_schema_size_bytes",
			"The total size of all objects in this schema (bytes).",
			[]string{"instance", "uri", "db", "schema"},
		),
		schemaOwnerMetric: NewMetrics(
			"pg_schema_owner",
			"Gauge indicating the owner of the schema.",
			[]string{"instance", "uri", "db", "schema", "owner"},
		),
		tableCountMetric: NewMetrics(
			"pg_schema_table_count",
			"The number of tables in this schema.",
			[]string{"instance", "uri", "db", "schema"},
		),
		viewCountMetric: NewMetrics(
			"pg_schema_view_count",
			"The number of views in this schema.",
			[]string{"instance", "uri", "db", "schema"},
		),
		indexCountMetric: NewMetrics(
			"pg_schema_index_count",
			"The number of indexes in this schema.",
			[]string{"instance", "uri", "db", "schema"},
		),
		lastModifiedTimestampMetric: NewMetrics(
			"pg_schema_last_modified_timestamp",
			"The timestamp when the schema was last modified.",
			[]string{"instance", "uri", "db", "schema"},
		),
	}
}

// NewPostgreSQLSchemaExporter 创建一个新的 Schema Exporter
func NewPostgreSQLSchemaExporter(db *sql.DB, uri, dbName string) *PostgreSQLSchemaExporter {
	return &PostgreSQLSchemaExporter{
		metrics: newSchemaMetrics(),
		db:      db,
		uri:     uri,
		dbName:  dbName,
	}
}

// Describe implements Prometheus Collector interface

// Collect implements Prometheus Collector interface
func (e *PostgreSQLSchemaExporter) collect(ch chan<- prometheus.Metric) {
	schemaStats, err := collector.ScrapePostgreSQLSchemas(e.db)
	if err != nil || len(schemaStats.Schemas) == 0 {
		return
	}

	labelsBase := []string{"", e.uri, e.dbName}

	// 上报总 Schema 数量
	e.metrics.schemaCountMetric.collect(
		ch,
		float64(len(schemaStats.Schemas)),
		labelsBase,
	)

	// 遍历每个 Schema 上报指标
	for _, schema := range schemaStats.Schemas {
		labels := append(labelsBase, schema.Name)

		e.metrics.schemaSizeBytesMetric.collect(
			ch,
			float64(schema.SizeBytes),
			labels,
		)
		e.metrics.schemaOwnerMetric.collect(
			ch,
			1, append(labels, schema.Owner))
		e.metrics.tableCountMetric.collect(
			ch,
			float64(schema.TableCount),
			labels,
		)
		e.metrics.viewCountMetric.collect(
			ch,
			float64(schema.ViewCount),
			labels,
		)
		e.metrics.indexCountMetric.collect(
			ch,
			float64(schema.IndexCount),
			labels,
		)
		e.metrics.lastModifiedTimestampMetric.collect(
			ch,
			float64(schema.LastModified.Unix()),
			labels,
		)
	}
}
// Part 2 commit for postgres_exporter/internal/metrics/schema_exporter.go

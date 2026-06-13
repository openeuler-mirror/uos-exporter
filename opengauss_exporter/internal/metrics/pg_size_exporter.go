package metrics

import (
	"database/sql"

	"opengauss_exporter/internal/collector"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type PgSizeExporter struct {
	db       *sql.DB
	instance string
	uri      string
	metrics  *pgSizeMetrics
}

type pgSizeMetrics struct {
	// 数据库大小指标
	databaseSizeMetric      *baseMetrics
	totalDatabaseSizeMetric *baseMetrics

	// 表大小指标
	tableSizeMetric      *baseMetrics
	tableTotalSizeMetric *baseMetrics
	totalTableSizeMetric *baseMetrics

	// 表空间大小指标
	tablespaceSizeMetric *baseMetrics
}

func newPgSizeMetrics() *pgSizeMetrics {
	return &pgSizeMetrics{
		databaseSizeMetric: NewMetrics(
			"pg_database_size_bytes",
			"Size of database in bytes",
			[]string{"instance", "uri", "database"},
		),
		totalDatabaseSizeMetric: NewMetrics(
			"pg_database_size_total_bytes",
			"Total size of all databases in bytes",
			[]string{"instance", "uri"},
		),
		tableSizeMetric: NewMetrics(
			"pg_table_size_bytes",
			"Size of table in bytes (excluding indexes)",
			[]string{"instance", "uri", "schema", "table"},
		),
		tableTotalSizeMetric: NewMetrics(
			"pg_table_total_size_bytes",
			"Total size of table in bytes (including indexes)",
			[]string{"instance", "uri", "schema", "table"},
		),
		totalTableSizeMetric: NewMetrics(
			"pg_table_size_total_bytes",
			"Total size of all tables in bytes",
			[]string{"instance", "uri"},
		),
		tablespaceSizeMetric: NewMetrics(
			"pg_tablespace_size_bytes",
			"Size of tablespace in bytes",
			[]string{"instance", "uri", "tablespace"},
		),
	}
}

// NewPgSizeExporter 创建一个新的 PgSize Exporter
func NewPgSizeExporter(db *sql.DB, instance, uri string) *PgSizeExporter {
	return &PgSizeExporter{
		db:       db,
		instance: instance,
		uri:      uri,
		metrics:  newPgSizeMetrics(),
	}
}

// Describe implements Prometheus Collector interface
func (e *PgSizeExporter) Describe(descs chan<- *prometheus.Desc) {
	// This is a no-op implementation as we use ConstMetrics
}

// Collect implements Prometheus Collector interface and Metric interface
func (e *PgSizeExporter) Collect(ch chan<- prometheus.Metric) {
	baseLabels := []string{e.instance, e.uri}

	sizeStats, err := collector.ScrapePgSizeStats(e.db)
	if err != nil {
		logrus.Debugf("Failed to scrape pg size stats: %v", err)
		return
	}

	var metricsCount int

	// 导出数据库大小指标
	for _, dbSize := range sizeStats.DatabaseSizes {
		e.metrics.databaseSizeMetric.collect(
			ch,
			float64(dbSize.Size),
			append(baseLabels, dbSize.DatName),
		)
		metricsCount++
	}

	// 导出总数据库大小指标
	e.metrics.totalDatabaseSizeMetric.collect(
		ch,
		float64(sizeStats.TotalDatabaseSize),
		baseLabels,
	)
	metricsCount++

	// 导出表大小指标
	for _, tableSize := range sizeStats.TableSizes {
		labels := append(baseLabels, tableSize.SchemaName, tableSize.TableName)

		e.metrics.tableSizeMetric.collect(
			ch,
			float64(tableSize.Size),
			labels,
		)
		e.metrics.tableTotalSizeMetric.collect(
			ch,
			float64(tableSize.TotalSize),
			labels,
		)
		metricsCount += 2
	}

	// 导出总表大小指标
	e.metrics.totalTableSizeMetric.collect(
		ch,
		float64(sizeStats.TotalTableSize),
		baseLabels,
	)
	metricsCount++

	// 导出表空间大小指标
	for _, tablespaceSize := range sizeStats.TablespaceSizes {
		e.metrics.tablespaceSizeMetric.collect(
			ch,
			float64(tablespaceSize.Size),
			append(baseLabels, tablespaceSize.TablespaceName),
		)
		metricsCount++
	}

	logrus.Debugf("Successfully exported %d size metrics (%d databases: %d MB, %d tables: %d MB, %d tablespaces)",
		metricsCount, len(sizeStats.DatabaseSizes), sizeStats.TotalDatabaseSize/(1024*1024),
		len(sizeStats.TableSizes), sizeStats.TotalTableSize/(1024*1024), len(sizeStats.TablespaceSizes))
}

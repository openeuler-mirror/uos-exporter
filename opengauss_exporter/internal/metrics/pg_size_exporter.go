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


// TODO: implement functions

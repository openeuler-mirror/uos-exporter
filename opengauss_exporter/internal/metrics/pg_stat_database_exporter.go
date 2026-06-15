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


// TODO: implement functions

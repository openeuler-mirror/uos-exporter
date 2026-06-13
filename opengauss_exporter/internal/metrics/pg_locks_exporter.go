package metrics

import (
	"database/sql"

	"opengauss_exporter/internal/collector"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type PgLocksExporter struct {
	db       *sql.DB
	instance string
	uri      string
	metrics  *pgLocksMetrics
}

type pgLocksMetrics struct {
	// 按类型分组的锁统计
	locksByTypeMetric *baseMetrics

	// 按模式分组的锁统计
	locksByModeMetric *baseMetrics

	// 按状态分组的锁统计
	locksByStateMetric *baseMetrics

	// 按数据库分组的锁统计
	locksByDatabaseMetric *baseMetrics

	// 总体锁统计
	waitingLocksMetric *baseMetrics
	grantedLocksMetric *baseMetrics
	totalLocksMetric   *baseMetrics
}


// TODO: implement functions

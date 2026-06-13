package metrics

import (
	"database/sql"

	"opengauss_exporter/internal/collector"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type PgStatActivityExporter struct {
	db       *sql.DB
	instance string
	uri      string
	metrics  *pgStatActivityMetrics
}

type pgStatActivityMetrics struct {
	// 按状态分组的连接数指标
	activeConnectionsMetric            *baseMetrics
	idleConnectionsMetric              *baseMetrics
	idleInTransactionConnectionsMetric *baseMetrics
	waitingConnectionsMetric           *baseMetrics
	otherConnectionsMetric             *baseMetrics
	totalConnectionsMetric             *baseMetrics

	// 按数据库分组的连接数指标
	connectionsByDatabaseMetric *baseMetrics

	// 按用户分组的连接数指标
	connectionsByUserMetric *baseMetrics

	// 等待事件统计指标
	waitEventStatsMetric *baseMetrics

	// 长时间运行查询指标
	longRunningQueriesMetric  *baseMetrics
	oldestQueryDurationMetric *baseMetrics
}


// TODO: implement functions

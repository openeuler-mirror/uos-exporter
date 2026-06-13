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

func newPgStatActivityMetrics() *pgStatActivityMetrics {
	return &pgStatActivityMetrics{
		activeConnectionsMetric: NewMetrics(
			"pg_stat_activity_connections_active",
			"Number of active connections from pg_stat_activity",
			[]string{"instance", "uri"},
		),
		idleConnectionsMetric: NewMetrics(
			"pg_stat_activity_connections_idle",
			"Number of idle connections from pg_stat_activity",
			[]string{"instance", "uri"},
		),
		idleInTransactionConnectionsMetric: NewMetrics(
			"pg_stat_activity_connections_idle_in_transaction",
			"Number of idle in transaction connections from pg_stat_activity",
			[]string{"instance", "uri"},
		),
		waitingConnectionsMetric: NewMetrics(
			"pg_stat_activity_connections_waiting",
			"Number of waiting connections from pg_stat_activity",
			[]string{"instance", "uri"},
		),
		otherConnectionsMetric: NewMetrics(
			"pg_stat_activity_connections_other",
			"Number of other state connections from pg_stat_activity",
			[]string{"instance", "uri"},
		),
		totalConnectionsMetric: NewMetrics(
			"pg_stat_activity_connections_total",
			"Total number of connections from pg_stat_activity",
			[]string{"instance", "uri"},
		),
		connectionsByDatabaseMetric: NewMetrics(
			"pg_stat_activity_connections_by_database",
			"Number of connections by database from pg_stat_activity",
			[]string{"instance", "uri", "database"},
		),
		connectionsByUserMetric: NewMetrics(
			"pg_stat_activity_connections_by_user",
			"Number of connections by user from pg_stat_activity",
			[]string{"instance", "uri", "user"},
		),
		waitEventStatsMetric: NewMetrics(
			"pg_stat_activity_wait_events",
			"Number of connections by wait event type from pg_stat_activity",
			[]string{"instance", "uri", "wait_event_type"},
		),
		longRunningQueriesMetric: NewMetrics(
			"pg_stat_activity_long_running_queries",
			"Number of long running queries (>5 minutes) from pg_stat_activity",
			[]string{"instance", "uri"},
		),
		oldestQueryDurationMetric: NewMetrics(
			"pg_stat_activity_oldest_query_duration_seconds",
			"Duration of the oldest active query in seconds from pg_stat_activity",
			[]string{"instance", "uri"},
		),
	}
}

// NewPgStatActivityExporter 创建一个新的 PgStatActivity Exporter
func NewPgStatActivityExporter(db *sql.DB, instance, uri string) *PgStatActivityExporter {
	return &PgStatActivityExporter{
		db:       db,
		instance: instance,
		uri:      uri,
		metrics:  newPgStatActivityMetrics(),
	}
}

// Describe implements Prometheus Collector interface
func (e *PgStatActivityExporter) Describe(descs chan<- *prometheus.Desc) {
	// This is a no-op implementation as we use ConstMetrics
}

// Collect implements Prometheus Collector interface and Metric interface
func (e *PgStatActivityExporter) Collect(ch chan<- prometheus.Metric) {
	labels := []string{e.instance, e.uri}

	activity, err := collector.ScrapePgStatActivity(e.db)
	if err != nil {
		logrus.Debugf("Failed to scrape pg_stat_activity: %v", err)
		return
	}

	// 导出按状态分组的连接数指标
	e.metrics.activeConnectionsMetric.collect(
		ch,
		float64(activity.ActiveConnections),
		labels,
	)
	e.metrics.idleConnectionsMetric.collect(
		ch,
		float64(activity.IdleConnections),
		labels,
	)
	e.metrics.idleInTransactionConnectionsMetric.collect(
		ch,
		float64(activity.IdleInTransactionConnections),
		labels,
	)
	e.metrics.waitingConnectionsMetric.collect(
		ch,
		float64(activity.WaitingConnections),
		labels,
	)
	e.metrics.otherConnectionsMetric.collect(
		ch,
		float64(activity.OtherConnections),
		labels,
	)
	e.metrics.totalConnectionsMetric.collect(
		ch,
		float64(activity.TotalConnections),
		labels,
	)

	// 导出按数据库分组的连接数指标
	for database, count := range activity.ConnectionsByDatabase {
		e.metrics.connectionsByDatabaseMetric.collect(
			ch,
			float64(count),
			append(labels, database),
		)
	}

	// 导出按用户分组的连接数指标
	for user, count := range activity.ConnectionsByUser {
		e.metrics.connectionsByUserMetric.collect(
			ch,
			float64(count),
			append(labels, user),
		)
	}

	// 导出等待事件统计指标
	for waitEventType, count := range activity.WaitEventStats {
		e.metrics.waitEventStatsMetric.collect(
			ch,
			float64(count),
			append(labels, waitEventType),
		)
	}

	// 导出长时间运行查询指标
	e.metrics.longRunningQueriesMetric.collect(
		ch,
		float64(activity.LongRunningQueries),
		labels,
	)
	e.metrics.oldestQueryDurationMetric.collect(
		ch,
		activity.OldestQueryDuration,
		labels,
	)
}

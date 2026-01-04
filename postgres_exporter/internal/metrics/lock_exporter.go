package metrics

import (
	"database/sql"
	"fmt"

	"postgres_exporter/internal/collector"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type PostgreSQLLockExporter struct {
	metrics *lockMetrics
	db      *sql.DB
	uri     string
	dbName  string
}

type lockMetrics struct {
	lockCountMetric           *baseMetrics
	lockWaitingCountMetric    *baseMetrics
	lockDurationSecondsMetric *baseMetrics
	lockQueryStateMetric      *baseMetrics
}

func newLockMetrics() *lockMetrics {
	return &lockMetrics{
		lockCountMetric: NewMetrics(
			"pg_locks_count",
			"Gauge indicating if a lock is held (1) or not (0).",
			[]string{"instance", "uri", "db", "schema", "table", "mode", "pid"},
		),
		lockWaitingCountMetric: NewMetrics(
			"pg_locks_waiting_count",
			"Gauge indicating if a lock is currently waiting (1) or not (0).",
			[]string{"instance", "uri", "db", "schema", "table", "mode", "pid"},
		),
		lockDurationSecondsMetric: NewMetrics(
			"pg_locks_query_duration_seconds",
			"The duration of the current query holding the lock.",
			[]string{"instance", "uri", "db", "schema", "table", "mode", "pid"},
		),
		lockQueryStateMetric: NewMetrics(
			"pg_locks_query_state",
			"Gauge indicating the state of the query (active=1, idle=2, waiting=3).",
			[]string{"instance", "uri", "db", "schema", "table", "mode", "pid"},
		),
	}
}

// NewPostgreSQLLockExporter 创建一个新的 Lock Exporter
func NewPostgreSQLLockExporter(db *sql.DB, uri, dbName string) *PostgreSQLLockExporter {
	return &PostgreSQLLockExporter{
		metrics: newLockMetrics(),
		db:      db,
		uri:     uri,
		dbName:  dbName,
	}
}

// Describe implements Prometheus Collector interface

// Collect implements Prometheus Collector interface
func (e *PostgreSQLLockExporter) collect(ch chan<- prometheus.Metric) {
	lockStats, err := collector.ScrapePostgreSQLLocks(e.db)
	if err != nil || len(lockStats.Locks) == 0 {
		logrus.Debugf("Failed to scrape PostgreSQL lock stats: %v", err)
		logrus.Debugf("Lock stats: %v", lockStats)
		return
	}

	labelsBase := []string{e.uri, e.dbName}

	for _, lock := range lockStats.Locks {
		labels := append(labelsBase, lock.Schema, lock.Table, lock.Mode, fmt.Sprintf("%d", lock.PID))

		// 上报锁持有状态
		e.metrics.lockCountMetric.collect(
			ch,
			1,
			labels,
		)

		// 上报是否在等待锁
		e.metrics.lockWaitingCountMetric.collect(
			ch,
			boolToFloat64(lock.Waited),
			labels,
		)

		// 上报查询持续时间
		e.metrics.lockDurationSecondsMetric.collect(
			ch,
			float64(lock.Duration.Seconds()),
			labels,
		)

		// 上报查询状态（active/idle/waiting）
		stateVal := getStateValue(lock.State)
		e.metrics.lockQueryStateMetric.collect(
			ch,
			float64(stateVal),
			labels,
		)
	}
}

// 辅助函数：将查询状态映射为数值
func getStateValue(state string) float64 {
	switch state {
	case "active":
		return 1.0
	case "idle":
		return 2.0
	case "idle in transaction", "idle in transaction (waiting)", "waiting":
		return 3.0
	default:
		return 0.0
	}
}
// Part 2 commit for postgres_exporter/internal/metrics/lock_exporter.go

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

func newPgLocksMetrics() *pgLocksMetrics {
	return &pgLocksMetrics{
		locksByTypeMetric: NewMetrics(
			"pg_locks_by_type",
			"Number of locks by lock type",
			[]string{"instance", "uri", "locktype"},
		),
		locksByModeMetric: NewMetrics(
			"pg_locks_by_mode",
			"Number of locks by lock mode",
			[]string{"instance", "uri", "mode"},
		),
		locksByStateMetric: NewMetrics(
			"pg_locks_by_state",
			"Number of locks by lock state",
			[]string{"instance", "uri", "state"},
		),
		locksByDatabaseMetric: NewMetrics(
			"pg_locks_by_database",
			"Number of locks by database",
			[]string{"instance", "uri", "database"},
		),
		waitingLocksMetric: NewMetrics(
			"pg_locks_waiting_total",
			"Total number of waiting locks",
			[]string{"instance", "uri"},
		),
		grantedLocksMetric: NewMetrics(
			"pg_locks_granted_total",
			"Total number of granted locks",
			[]string{"instance", "uri"},
		),
		totalLocksMetric: NewMetrics(
			"pg_locks_total",
			"Total number of locks",
			[]string{"instance", "uri"},
		),
	}
}

// NewPgLocksExporter 创建一个新的 PgLocks Exporter
func NewPgLocksExporter(db *sql.DB, instance, uri string) *PgLocksExporter {
	return &PgLocksExporter{
		db:       db,
		instance: instance,
		uri:      uri,
		metrics:  newPgLocksMetrics(),
	}
}

// Describe implements Prometheus Collector interface
func (e *PgLocksExporter) Describe(descs chan<- *prometheus.Desc) {
	// This is a no-op implementation as we use ConstMetrics
}

// Collect implements Prometheus Collector interface and Metric interface
func (e *PgLocksExporter) Collect(ch chan<- prometheus.Metric) {
	baseLabels := []string{e.instance, e.uri}

	locksStat, err := collector.ScrapePgLocks(e.db)
	if err != nil {
		logrus.Debugf("Failed to scrape pg_locks: %v", err)
		return
	}

	var metricsCount int

	// 导出按锁类型分组的指标
	for locktype, count := range locksStat.LocksByType {
		e.metrics.locksByTypeMetric.collect(
			ch,
			float64(count),
			append(baseLabels, locktype),
		)
		metricsCount++
	}

	// 导出按锁模式分组的指标
	for mode, count := range locksStat.LocksByMode {
		e.metrics.locksByModeMetric.collect(
			ch,
			float64(count),
			append(baseLabels, mode),
		)
		metricsCount++
	}

	// 导出按锁状态分组的指标
	for state, count := range locksStat.LocksByState {
		e.metrics.locksByStateMetric.collect(
			ch,
			float64(count),
			append(baseLabels, state),
		)
		metricsCount++
	}

	// 导出按数据库分组的指标
	for database, count := range locksStat.LocksByDatabase {
		e.metrics.locksByDatabaseMetric.collect(
			ch,
			float64(count),
			append(baseLabels, database),
		)
		metricsCount++
	}

	// 导出总体锁统计指标
	e.metrics.waitingLocksMetric.collect(
		ch,
		float64(locksStat.WaitingLocks),
		baseLabels,
	)
	e.metrics.grantedLocksMetric.collect(
		ch,
		float64(locksStat.GrantedLocks),
		baseLabels,
	)
	e.metrics.totalLocksMetric.collect(
		ch,
		float64(locksStat.TotalLocks),
		baseLabels,
	)
	metricsCount += 3

	logrus.Debugf("Successfully exported %d pg_locks metrics (total: %d locks, waiting: %d, granted: %d)",
		metricsCount, locksStat.TotalLocks, locksStat.WaitingLocks, locksStat.GrantedLocks)
}

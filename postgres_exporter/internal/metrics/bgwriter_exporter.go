package metrics

import (
	"database/sql"
	"time"

	"postgres_exporter/internal/collector"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type PostgreSQLBGWriterExporter struct {
	metrics *bgwriterMetrics
	db      *sql.DB
	uri     string
	dbName  string
}

type bgwriterMetrics struct {
	checkpointsTotalMetric           *baseMetrics
	checkpointsScheduledMetric       *baseMetrics
	checkpointsRequestedMetric       *baseMetrics
	checkpointWriteTimeSecondsMetric *baseMetrics
	checkpointSyncTimeSecondsMetric  *baseMetrics
	buffersCheckpointMetric          *baseMetrics
	buffersCleanMetric               *baseMetrics
	buffersBackendMetric             *baseMetrics
	buffersAllocatedMetric           *baseMetrics
	maxWaitTimeMillisMetric          *baseMetrics
	avgWaitTimeMillisMetric          *baseMetrics
	statsLastResetTimestampMetric    *baseMetrics
}

func newBGWriterMetrics() *bgwriterMetrics {
	return &bgwriterMetrics{
		checkpointsTotalMetric: NewMetrics(
			"pg_bgwriter_checkpoints_total",
			"The total number of checkpoints (scheduled + requested).",
			[]string{"instance", "uri", "db"},
		),
		checkpointsScheduledMetric: NewMetrics(
			"pg_bgwriter_checkpoints_scheduled_total",
			"The number of scheduled checkpoints.",
			[]string{"instance", "uri", "db"},
		),
		checkpointsRequestedMetric: NewMetrics(
			"pg_bgwriter_checkpoints_requested_total",
			"The number of requested checkpoints.",
			[]string{"instance", "uri", "db"},
		),
		checkpointWriteTimeSecondsMetric: NewMetrics(
			"pg_bgwriter_checkpoint_write_time_seconds",
			"The time spent writing buffers during checkpoints in seconds.",
			[]string{"instance", "uri", "db"},
		),
		checkpointSyncTimeSecondsMetric: NewMetrics(
			"pg_bgwriter_checkpoint_sync_time_seconds",
			"The time spent syncing buffers during checkpoints in seconds.",
			[]string{"instance", "uri", "db"},
		),
		buffersCheckpointMetric: NewMetrics(
			"pg_bgwriter_buffers_checkpoint_total",
			"The number of buffers written during checkpoints.",
			[]string{"instance", "uri", "db"},
		),
		buffersCleanMetric: NewMetrics(
			"pg_bgwriter_buffers_clean_total",
			"The number of buffers written by the background writer.",
			[]string{"instance", "uri", "db"},
		),
		buffersBackendMetric: NewMetrics(
			"pg_bgwriter_buffers_backend_total",
			"The number of buffers written directly by backends.",
			[]string{"instance", "uri", "db"},
		),
		buffersAllocatedMetric: NewMetrics(
			"pg_bgwriter_buffers_allocated_total",
			"The number of buffers allocated.",
			[]string{"instance", "uri", "db"},
		),
		maxWaitTimeMillisMetric: NewMetrics(
			"pg_bgwriter_max_wait_time_millis",
			"The maximum wait time before writing a dirty buffer.",
			[]string{"instance", "uri", "db"},
		),
		avgWaitTimeMillisMetric: NewMetrics(
			"pg_bgwriter_avg_wait_time_millis",
			"The average wait time before writing a dirty buffer.",
			[]string{"instance", "uri", "db"},
		),
		statsLastResetTimestampMetric: NewMetrics(
			"pg_bgwriter_stats_last_reset_timestamp",
			"The timestamp when bgwriter statistics were last reset.",
			[]string{"instance", "uri", "db"},
		),
	}
}

// NewPostgreSQLBGWriterExporter 创建一个新的 BGWriter Exporter
func NewPostgreSQLBGWriterExporter(db *sql.DB, uri, dbName string) *PostgreSQLBGWriterExporter {
	return &PostgreSQLBGWriterExporter{
		metrics: newBGWriterMetrics(),
		db:      db,
		uri:     uri,
		dbName:  dbName,
	}
}

// Describe implements Prometheus Collector interface

// Collect implements Prometheus Collector interface
func (e *PostgreSQLBGWriterExporter) collect(ch chan<- prometheus.Metric) {
	// bgwriterStats, _ := collector.ScrapePostgreSQLBGWriter(e.db)
	bgwriterStats, err := collector.ScrapePostgreSQLBGWriter(e.db)
	if err != nil || bgwriterStats == nil || bgwriterStats.BGWriter == nil {
		logrus.Debugf("Failed to scrape PostgreSQL BGWriter stats: %v", err)
		logrus.Debugf("BGWriter stats: %v", bgwriterStats)
		return
	}

	labelsBase := []string{e.dbName, e.uri, bgwriterStats.BGWriter.Database}

	bg := bgwriterStats.BGWriter

	// 上报 Checkpoint 总数
	e.metrics.checkpointsScheduledMetric.collect(
		ch,
		float64(bg.CheckpointsScheduled),
		labelsBase,
	)
	e.metrics.checkpointsRequestedMetric.collect(
		ch,
		float64(bg.CheckpointsRequested),
		labelsBase,
	)
	e.metrics.checkpointsTotalMetric.collect(
		ch,
		float64(bg.CheckpointsScheduled+bg.CheckpointsRequested),
		labelsBase,
	)

	// 上报 Checkpoint 时间
	e.metrics.checkpointWriteTimeSecondsMetric.collect(
		ch,
		bg.CheckpointWriteTime/1000,
		labelsBase,
	)
	e.metrics.checkpointSyncTimeSecondsMetric.collect(
		ch,
		bg.CheckpointSyncTime/1000,
		labelsBase,
	)

	// 缓冲区写入统计
	e.metrics.buffersCheckpointMetric.collect(
		ch,
		float64(bg.BuffersCheckpoint),
		labelsBase,
	)
	e.metrics.buffersCleanMetric.collect(
		ch,
		float64(bg.BuffersClean),
		labelsBase,
	)
	e.metrics.buffersBackendMetric.collect(
		ch,
		float64(bg.BuffersBackend),
		labelsBase,
	)
	e.metrics.buffersAllocatedMetric.collect(
		ch,
		float64(bg.BuffersAllocated),
		labelsBase,
	)

	// 等待时间
	e.metrics.maxWaitTimeMillisMetric.collect(
		ch,
		bg.MaxWaitTime,
		labelsBase,
	)
	e.metrics.avgWaitTimeMillisMetric.collect(
		ch,
		bg.AvgWaitTime,
		labelsBase,
	)

	// 统计重置时间
	if bg.StatsReset != "" {
		t, _ := time.Parse(time.RFC3339, bg.StatsReset)
		e.metrics.statsLastResetTimestampMetric.collect(
			ch,
			float64(t.Unix()),
			labelsBase,
		)
	}
}

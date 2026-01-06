package metrics

import (
	"database/sql"

	"opengauss_exporter/internal/collector"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type PgStatBgwriterExporter struct {
	db       *sql.DB
	instance string
	uri      string
	metrics  *pgStatBgwriterMetrics
}

type pgStatBgwriterMetrics struct {
	// 检查点指标
	checkpointsTimedMetric    *baseMetrics
	checkpointsReqMetric      *baseMetrics
	checkpointWriteTimeMetric *baseMetrics
	checkpointSyncTimeMetric  *baseMetrics

	// 缓冲区指标
	buffersCheckpointMetric   *baseMetrics
	buffersCleanMetric        *baseMetrics
	maxwrittenCleanMetric     *baseMetrics
	buffersBackendMetric      *baseMetrics
	buffersBackendFsyncMetric *baseMetrics
	buffersAllocMetric        *baseMetrics

	// 统计重置时间指标
	statsResetMetric *baseMetrics
}

func newPgStatBgwriterMetrics() *pgStatBgwriterMetrics {
	return &pgStatBgwriterMetrics{
		checkpointsTimedMetric: NewMetrics(
			"pg_stat_bgwriter_checkpoints_timed_total",
			"Number of scheduled checkpoints that have been performed",
			[]string{"instance", "uri"},
		),
		checkpointsReqMetric: NewMetrics(
			"pg_stat_bgwriter_checkpoints_req_total",
			"Number of requested checkpoints that have been performed",
			[]string{"instance", "uri"},
		),
		checkpointWriteTimeMetric: NewMetrics(
			"pg_stat_bgwriter_checkpoint_write_time_ms",
			"Total amount of time that has been spent in the portion of checkpoint processing where files are written to disk, in milliseconds",
			[]string{"instance", "uri"},
		),
		checkpointSyncTimeMetric: NewMetrics(
			"pg_stat_bgwriter_checkpoint_sync_time_ms",
			"Total amount of time that has been spent in the portion of checkpoint processing where files are synchronized to disk, in milliseconds",
			[]string{"instance", "uri"},
		),
		buffersCheckpointMetric: NewMetrics(
			"pg_stat_bgwriter_buffers_checkpoint_total",
			"Number of buffers written during checkpoints",
			[]string{"instance", "uri"},
		),
		buffersCleanMetric: NewMetrics(
			"pg_stat_bgwriter_buffers_clean_total",
			"Number of buffers written by the background writer",
			[]string{"instance", "uri"},
		),
		maxwrittenCleanMetric: NewMetrics(
			"pg_stat_bgwriter_maxwritten_clean_total",
			"Number of times the background writer stopped a cleaning scan because it had written too many buffers",
			[]string{"instance", "uri"},
		),
		buffersBackendMetric: NewMetrics(
			"pg_stat_bgwriter_buffers_backend_total",
			"Number of buffers written directly by a backend",
			[]string{"instance", "uri"},
		),
		buffersBackendFsyncMetric: NewMetrics(
			"pg_stat_bgwriter_buffers_backend_fsync_total",
			"Number of times a backend had to execute its own fsync call",
			[]string{"instance", "uri"},
		),
		buffersAllocMetric: NewMetrics(
			"pg_stat_bgwriter_buffers_alloc_total",
			"Number of buffers allocated",
			[]string{"instance", "uri"},
		),
		statsResetMetric: NewMetrics(
			"pg_stat_bgwriter_stats_reset_time",
			"Time at which these statistics were last reset",
			[]string{"instance", "uri"},
		),
	}
}

// NewPgStatBgwriterExporter 创建一个新的 PgStatBgwriter Exporter
func NewPgStatBgwriterExporter(db *sql.DB, instance, uri string) *PgStatBgwriterExporter {
	return &PgStatBgwriterExporter{
		db:       db,
		instance: instance,
		uri:      uri,
		metrics:  newPgStatBgwriterMetrics(),
	}
}

// Describe implements Prometheus Collector interface
func (e *PgStatBgwriterExporter) Describe(descs chan<- *prometheus.Desc) {
	// This is a no-op implementation as we use ConstMetrics
}

// Collect implements Prometheus Collector interface and Metric interface
func (e *PgStatBgwriterExporter) Collect(ch chan<- prometheus.Metric) {
	labels := []string{e.instance, e.uri}

	bgwriter, err := collector.ScrapePgStatBgwriter(e.db)
	if err != nil {
		logrus.Debugf("Failed to scrape pg_stat_bgwriter: %v", err)
		return
	}

	// 导出检查点指标
	e.metrics.checkpointsTimedMetric.collect(
		ch,
		float64(bgwriter.CheckpointsTimed),
		labels,
	)
	e.metrics.checkpointsReqMetric.collect(
		ch,
		float64(bgwriter.CheckpointsReq),
		labels,
	)
	e.metrics.checkpointWriteTimeMetric.collect(
		ch,
		bgwriter.CheckpointWriteTime,
		labels,
	)
	e.metrics.checkpointSyncTimeMetric.collect(
		ch,
		bgwriter.CheckpointSyncTime,
		labels,
	)

	// 导出缓冲区指标
	e.metrics.buffersCheckpointMetric.collect(
		ch,
		float64(bgwriter.BuffersCheckpoint),
		labels,
	)
	e.metrics.buffersCleanMetric.collect(
		ch,
		float64(bgwriter.BuffersClean),
		labels,
	)
	e.metrics.maxwrittenCleanMetric.collect(
		ch,
		float64(bgwriter.MaxwrittenClean),
		labels,
	)
	e.metrics.buffersBackendMetric.collect(
		ch,
		float64(bgwriter.BuffersBackend),
		labels,
	)
	e.metrics.buffersBackendFsyncMetric.collect(
		ch,
		float64(bgwriter.BuffersBackendFsync),
		labels,
	)
	e.metrics.buffersAllocMetric.collect(
		ch,
		float64(bgwriter.BuffersAlloc),
		labels,
	)

	// 导出统计重置时间指标
	if bgwriter.StatsReset != nil {
		e.metrics.statsResetMetric.collect(
			ch,
			float64(bgwriter.StatsReset.Unix()),
			labels,
		)
	} else {
		e.metrics.statsResetMetric.collect(
			ch,
			0,
			labels,
		)
	}
}

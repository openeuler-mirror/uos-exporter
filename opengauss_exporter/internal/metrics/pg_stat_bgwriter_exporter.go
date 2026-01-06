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


// TODO: implement functions

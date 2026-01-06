package metrics

import (
	"database/sql"

	"opengauss_exporter/internal/collector"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type PgStatUserTablesExporter struct {
	db       *sql.DB
	instance string
	uri      string
	metrics  *pgStatUserTablesMetrics
}

type pgStatUserTablesMetrics struct {
	// 扫描统计指标
	seqScanMetric     *baseMetrics
	seqTupReadMetric  *baseMetrics
	idxScanMetric     *baseMetrics
	idxTupFetchMetric *baseMetrics

	// 元组变更统计指标
	nTupInsMetric    *baseMetrics
	nTupUpdMetric    *baseMetrics
	nTupDelMetric    *baseMetrics
	nTupHotUpdMetric *baseMetrics

	// 元组状态统计指标
	nLiveTupMetric         *baseMetrics
	nDeadTupMetric         *baseMetrics
	nModSinceAnalyzeMetric *baseMetrics

	// VACUUM统计指标
	vacuumCountMetric     *baseMetrics
	autovacuumCountMetric *baseMetrics
	lastVacuumMetric      *baseMetrics
	lastAutovacuumMetric  *baseMetrics

	// ANALYZE统计指标
	analyzeCountMetric     *baseMetrics
	autoanalyzeCountMetric *baseMetrics
	lastAnalyzeMetric      *baseMetrics
	lastAutoanalyzeMetric  *baseMetrics
}


// TODO: implement functions

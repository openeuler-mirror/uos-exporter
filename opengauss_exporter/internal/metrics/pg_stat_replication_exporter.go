package metrics

import (
	"database/sql"

	"opengauss_exporter/internal/collector"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type PgStatReplicationExporter struct {
	db       *sql.DB
	instance string
	uri      string
	metrics  *pgStatReplicationMetrics
}

type pgStatReplicationMetrics struct {
	writeLagMetric      *baseMetrics
	flushLagMetric      *baseMetrics
	replayLagMetric     *baseMetrics
	totalReplicasMetric *baseMetrics
	backendStartMetric  *baseMetrics
}

func newPgStatReplicationMetrics() *pgStatReplicationMetrics {
	return &pgStatReplicationMetrics{
		writeLagMetric: NewMetrics(
			"pg_stat_replication_write_lag_seconds",
			"Time elapsed between flushing recent WAL locally and receiving notification that this standby server has written it",
			[]string{"instance", "uri", "application_name", "client_addr", "state"},
		),
		flushLagMetric: NewMetrics(
			"pg_stat_replication_flush_lag_seconds",
			"Time elapsed between flushing recent WAL locally and receiving notification that this standby server has written and flushed it",
			[]string{"instance", "uri", "application_name", "client_addr", "state"},
		),
		replayLagMetric: NewMetrics(
			"pg_stat_replication_replay_lag_seconds",
			"Time elapsed between flushing recent WAL locally and receiving notification that this standby server has written, flushed and applied it",
			[]string{"instance", "uri", "application_name", "client_addr", "state"},
		),
		totalReplicasMetric: NewMetrics(
			"pg_stat_replication_total_replicas",
			"Total number of replication connections",
			[]string{"instance", "uri"},
		),
		backendStartMetric: NewMetrics(
			"pg_stat_replication_backend_start_time",
			"Time when this process was started",
			[]string{"instance", "uri", "application_name", "client_addr"},
		),
	}
}

func NewPgStatReplicationExporter(db *sql.DB, instance, uri string) *PgStatReplicationExporter {
	return &PgStatReplicationExporter{
		db:       db,
		instance: instance,
		uri:      uri,
		metrics:  newPgStatReplicationMetrics(),
	}
}

func (e *PgStatReplicationExporter) Describe(descs chan<- *prometheus.Desc) {}

func (e *PgStatReplicationExporter) Collect(ch chan<- prometheus.Metric) {
	baseLabels := []string{e.instance, e.uri}

	collection, err := collector.ScrapePgStatReplication(e.db)
	if err != nil {
		logrus.Debugf("Failed to scrape pg_stat_replication: %v", err)
		return
	}

	// 导出总复制连接数
	e.metrics.totalReplicasMetric.collect(ch, float64(collection.TotalReplicas), baseLabels)

	// 为每个复制连接导出指标
	for _, replication := range collection.Replications {
		labels := append(baseLabels, replication.ApplicationName, replication.ClientAddr, replication.State)
		backendLabels := append(baseLabels, replication.ApplicationName, replication.ClientAddr)

		e.metrics.writeLagMetric.collect(ch, replication.WriteLag, labels)
		e.metrics.flushLagMetric.collect(ch, replication.FlushLag, labels)
		e.metrics.replayLagMetric.collect(ch, replication.ReplayLag, labels)

		if replication.BackendStart != nil {
			e.metrics.backendStartMetric.collect(ch, float64(replication.BackendStart.Unix()), backendLabels)
		} else {
			e.metrics.backendStartMetric.collect(ch, 0, backendLabels)
		}
	}
}

package metrics

import (
	"database/sql"

	"postgres_exporter/internal/collector"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type PostgreSQLReplicationExporter struct {
	metrics *replicationMetrics
	db      *sql.DB
	uri     string
	dbName  string
}

type replicationMetrics struct {
	upMetric                   *baseMetrics
	isSyncStandbyMetric        *baseMetrics
	replicaStateMetric         *baseMetrics
	replicaClientAddressMetric *baseMetrics
	writeLagSecondsMetric      *baseMetrics
	flushLagSecondsMetric      *baseMetrics
	replayLagSecondsMetric     *baseMetrics
	bytesSentMetric            *baseMetrics
	writeLocationMetric        *baseMetrics
	flushLocationMetric        *baseMetrics
	replayLocationMetric       *baseMetrics
	latestWALFileMetric        *baseMetrics
	lastUpdatedTimestampMetric *baseMetrics
}

func newReplicationMetrics() *replicationMetrics {
	return &replicationMetrics{
		upMetric: NewMetrics(
			"pg_replication_up",
			"Gauge indicating if a standby is connected (1) or not (0).",
			[]string{"instance", "uri", "db", "standby"},
		),
		isSyncStandbyMetric: NewMetrics(
			"pg_replication_is_sync_standby",
			"Gauge indicating if the standby is in synchronous mode.",
			[]string{"instance", "uri", "db", "standby"},
		),
		replicaStateMetric: NewMetrics(
			"pg_replication_state",
			"Gauge indicating the current replication state.",
			[]string{"instance", "uri", "db", "standby"},
		),
		replicaClientAddressMetric: NewMetrics(
			"pg_replication_client_address",
			"Gauge indicating the client address of the standby.",
			[]string{"instance", "uri", "db", "standby", "address"},
		),
		writeLagSecondsMetric: NewMetrics(
			"pg_replication_write_lag_seconds",
			"The lag between primary and standby for write.",
			[]string{"instance", "uri", "db", "standby"},
		),
		flushLagSecondsMetric: NewMetrics(
			"pg_replication_flush_lag_seconds",
			"The lag between primary and standby for flush.",
			[]string{"instance", "uri", "db", "standby"},
		),
		replayLagSecondsMetric: NewMetrics(
			"pg_replication_replay_lag_seconds",
			"The lag between primary and standby for replay.",
			[]string{"instance", "uri", "db", "standby"},
		),
		bytesSentMetric: NewMetrics(
			"pg_replication_bytes_sent_total",
			"The total number of bytes sent to the standby.",
			[]string{"instance", "uri", "db", "standby"},
		),
		writeLocationMetric: NewMetrics(
			"pg_replication_write_location_bytes",
			"The current LSN write location in bytes.",
			[]string{"instance", "uri", "db", "standby"},
		),
		flushLocationMetric: NewMetrics(
			"pg_replication_flush_location_bytes",
			"The current LSN flush location in bytes.",
			[]string{"instance", "uri", "db", "standby"},
		),
		replayLocationMetric: NewMetrics(
			"pg_replication_replay_location_bytes",
			"The current LSN replay location in bytes.",
			[]string{"instance", "uri", "db", "standby"},
		),
		latestWALFileMetric: NewMetrics(
			"pg_replication_latest_wal_file",
			"Gauge indicating the latest WAL file sent.",
			[]string{"instance", "uri", "db", "standby", "wal_file"},
		),
		lastUpdatedTimestampMetric: NewMetrics(
			"pg_replication_last_updated_timestamp",
			"The timestamp when replication was last updated.",
			[]string{"instance", "uri", "db", "standby"},
		),
	}
}

// NewPostgreSQLReplicationExporter 创建一个新的 Replication Exporter
func NewPostgreSQLReplicationExporter(db *sql.DB, uri, dbName string) *PostgreSQLReplicationExporter {
	return &PostgreSQLReplicationExporter{
		metrics: newReplicationMetrics(),
		db:      db,
		uri:     uri,
		dbName:  dbName,
	}
}

// Describe implements Prometheus Collector interface

// Collect implements Prometheus Collector interface
func (e *PostgreSQLReplicationExporter) collect(ch chan<- prometheus.Metric) {
	replicationStats, err := collector.ScrapePostgreSQLReplication(e.db)
	if err != nil || len(replicationStats.Replicas) == 0 {
		logrus.Debugf("Failed to scrape PostgreSQL replication stats: %v", err)
		logrus.Debugf("Replication stats: %v", replicationStats)
		return
	}

	labelsBase := []string{e.uri, e.dbName}

	for _, replica := range replicationStats.Replicas {
		labels := append(labelsBase, replica.ApplicationName)
		addressLabels := append(labels, replica.ClientAddress)
		walLabels := append(labels, replica.LatestWALSent)

		// 上报是否在线
		e.metrics.upMetric.collect(
			ch,
			1,
			labels,
		)

		// 是否是同步备机
		e.metrics.isSyncStandbyMetric.collect(
			ch,
			boolToFloat64(replica.IsSyncStandby),
			labels,
		)

		// 复制状态
		e.metrics.replicaStateMetric.collect(
			ch,
			1, append(labels, replica.State))

		// 客户端地址
		e.metrics.replicaClientAddressMetric.collect(
			ch,
			1, addressLabels)

		// 延迟指标
		e.metrics.writeLagSecondsMetric.collect(
			ch,
			replica.WriteLag,
			labels,
		)
		e.metrics.flushLagSecondsMetric.collect(
			ch,
			replica.FlushLag,
			labels,
		)
		e.metrics.replayLagSecondsMetric.collect(
			ch,
			replica.ReplayLag,
			labels,
		)

		// 发送的字节数
		e.metrics.bytesSentMetric.collect(
			ch,
			float64(replica.BytesSent),
			labels,
		)

		// LSN 位置
		e.metrics.writeLocationMetric.collect(
			ch,
			float64(replica.WriteLocation),
			labels,
		)
		e.metrics.flushLocationMetric.collect(
			ch,
			float64(replica.FlushLocation),
			labels,
		)
		e.metrics.replayLocationMetric.collect(
			ch,
			float64(replica.ReplayLocation),
			labels,
		)

		// 最新 WAL 文件
		e.metrics.latestWALFileMetric.collect(
			ch,
			1, walLabels)

		// 最后更新时间戳
		e.metrics.lastUpdatedTimestampMetric.collect(
			ch,
			float64(replica.LastUpdated.Unix()),
			labels,
		)
	}
}
// Part 2 commit for postgres_exporter/internal/metrics/replication_exporter.go

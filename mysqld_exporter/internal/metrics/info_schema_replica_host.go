package metrics

import (
	CMySQL "github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"mysqld_exporter/internal/exporter"
	"mysqld_exporter/internal/mysql"
)

const (
	replicaHostQuery = `
	  SELECT SERVER_ID
		   , if(SESSION_ID='MASTER_SESSION_ID','writer','reader') AS ROLE
		   , CPU
		   , MASTER_SLAVE_LATENCY_IN_MICROSECONDS
		   , REPLICA_LAG_IN_MILLISECONDS
		   , LOG_STREAM_SPEED_IN_KiB_PER_SECOND
		   , CURRENT_REPLAY_LATENCY_IN_MICROSECONDS
		FROM information_schema.replica_host_status
	`
)

func init() {
	exporter.Register(
		NewScrapeReplicaHost())
}

type ScrapeReplicaHost struct {
	instance mysql.Instance
	infoSchemaReplicaHostCpuDesc
	infoSchemaReplicaHostReplicaLatencyDesc
	infoSchemaReplicaHostLagDesc
	infoSchemaReplicaHostLogStreamSpeedDesc
	infoSchemaReplicaHostReplayLatencyDesc
}

func NewScrapeReplicaHost() *ScrapeReplicaHost {
	return &ScrapeReplicaHost{
		//instance:                                instance,
		infoSchemaReplicaHostCpuDesc:            *NewinfoSchemaReplicaHostCpuDesc(),
		infoSchemaReplicaHostReplicaLatencyDesc: *NewinfoSchemaReplicaHostReplicaLatencyDesc(),
		infoSchemaReplicaHostLagDesc:            *NewinfoSchemaReplicaHostLagDesc(),
		infoSchemaReplicaHostLogStreamSpeedDesc: *NewinfoSchemaReplicaHostLogStreamSpeedDesc(),
		infoSchemaReplicaHostReplayLatencyDesc:  *NewinfoSchemaReplicaHostReplayLatencyDesc(),
	}
}

func (qd ScrapeReplicaHost) Collect(ch chan<- prometheus.Metric) {
	qd.instance = *GetInstance()

	if err := qd.instance.Ping(); err != nil {
		logrus.Errorf("ping mysql instance error: %s", err)
		return
	}
	db := instance.GetDB()
	replicaHostRows, err := db.Query(replicaHostQuery)
	if err != nil {
		if mysqlErr, ok := err.(*CMySQL.MySQLError); ok { // Now the error number is accessible directly
			// Check for error 1109: Unknown table
			if mysqlErr.Number == 1109 {
				logrus.Debug("information_schema.replica_host_status is not available.")
				return
			}
		}
		logrus.Debugf("failed to query mysql instance information_schema.ROCKSDB_PERF_CONTEXT: %s",
			err)
		return
	}
	defer replicaHostRows.Close()
	var (
		serverId       string
		role           string
		cpu            float64
		replicaLatency uint64
		replicaLag     float64
		logStreamSpeed float64
		replayLatency  uint64
	)
	for replicaHostRows.Next() {
		if err := replicaHostRows.Scan(
			&serverId,
			&role,
			&cpu,
			&replicaLatency,
			&replicaLag,
			&logStreamSpeed,
			&replayLatency,
		); err != nil {
			//return err
			logrus.Error(err)
			return
		}
		qd.infoSchemaReplicaHostCpuDesc.Collect(ch,
			float64(cpu),
			[]string{
				serverId,
				role,
			})
		qd.infoSchemaReplicaHostReplicaLatencyDesc.Collect(ch,
			float64(replicaLatency)*0.000001,
			[]string{
				serverId,
				role,
			})

		qd.infoSchemaReplicaHostLagDesc.Collect(ch,
			replicaLag*0.001,
			[]string{
				serverId,
				role,
			})

		qd.infoSchemaReplicaHostLogStreamSpeedDesc.Collect(ch,
			logStreamSpeed,
			[]string{
				serverId,
				role,
			})

		qd.infoSchemaReplicaHostReplayLatencyDesc.Collect(ch,
			float64(replayLatency)*0.000001,
			[]string{
				serverId,
				role,
			})
	}
}

type infoSchemaReplicaHostCpuDesc struct {
	*baseMetrics
}

func NewinfoSchemaReplicaHostCpuDesc() *infoSchemaReplicaHostCpuDesc {
	return &infoSchemaReplicaHostCpuDesc{
		NewMetrics(
			"info_schema_replica_host_cpu_percent",
			"The CPU usage as a percentage.",
			[]string{
				"server_id",
				"role"})}
}

func (qd *infoSchemaReplicaHostCpuDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaReplicaHostReplicaLatencyDesc struct {
	*baseMetrics
}

func NewinfoSchemaReplicaHostReplicaLatencyDesc() *infoSchemaReplicaHostReplicaLatencyDesc {
	return &infoSchemaReplicaHostReplicaLatencyDesc{
		NewMetrics(
			"info_schema_replica_host_replica_latency_seconds",
			"The source-replica latency in seconds.",
			[]string{
				"server_id",
				"role"})}
}

func (qd *infoSchemaReplicaHostReplicaLatencyDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaReplicaHostLagDesc struct {
	*baseMetrics
}

func NewinfoSchemaReplicaHostLagDesc() *infoSchemaReplicaHostLagDesc {
	return &infoSchemaReplicaHostLagDesc{
		NewMetrics(
			"info_schema_replica_host_lag_seconds",
			"The replica lag in seconds.",
			[]string{
				"server_id",
				"role"})}
}
func (qd *infoSchemaReplicaHostLagDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaReplicaHostLogStreamSpeedDesc struct {
	*baseMetrics
}

func NewinfoSchemaReplicaHostLogStreamSpeedDesc() *infoSchemaReplicaHostLogStreamSpeedDesc {
	return &infoSchemaReplicaHostLogStreamSpeedDesc{
		NewMetrics(
			"info_schema_replica_host_log_stream_speed",
			"The log stream speed in kilobytes per second.",
			[]string{
				"server_id",
				"role"})}
}

func (qd *infoSchemaReplicaHostLogStreamSpeedDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaReplicaHostReplayLatencyDesc struct {
	*baseMetrics
}

func NewinfoSchemaReplicaHostReplayLatencyDesc() *infoSchemaReplicaHostReplayLatencyDesc {
	return &infoSchemaReplicaHostReplayLatencyDesc{
		NewMetrics(
			"info_schema_replica_host_replay_latency_seconds",
			"The replay latency in seconds.",
			[]string{
				"server_id",
				"role"})}
}

func (qd *infoSchemaReplicaHostReplayLatencyDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

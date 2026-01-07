package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"mysqld_exporter/internal/exporter"
	"mysqld_exporter/internal/mysql"
	"time"
)

const (
	perfReplicationApplierStatsByWorkerQuery = `
	SELECT 
	    CHANNEL_NAME,
		WORKER_ID,
		LAST_APPLIED_TRANSACTION_ORIGINAL_COMMIT_TIMESTAMP,
		LAST_APPLIED_TRANSACTION_IMMEDIATE_COMMIT_TIMESTAMP,
		LAST_APPLIED_TRANSACTION_START_APPLY_TIMESTAMP,
		LAST_APPLIED_TRANSACTION_END_APPLY_TIMESTAMP,
		APPLYING_TRANSACTION_ORIGINAL_COMMIT_TIMESTAMP,
		APPLYING_TRANSACTION_IMMEDIATE_COMMIT_TIMESTAMP, 
	  	APPLYING_TRANSACTION_START_APPLY_TIMESTAMP
    FROM performance_schema.replication_applier_status_by_worker
	`
	timeLayout = "2006-01-02 15:04:05.000000"
)

func init() {
	exporter.Register(
		NewScrapePerfReplicationApplierStatsByWorker())
}

type ScrapePerfReplicationApplierStatsByWorker struct {
	instance mysql.Instance
	performanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionOriginalCommitSecondDesc
	performanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionImmediateCommitSecondDesc
	performanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionStartApplySecondDesc
	performanceSchemaReplicationApplierStatsByWorkerApplyingTransactionOriginalCommitSecondDesc
	performanceSchemaReplicationApplierStatsByWorkerApplyingTransactionImmediateCommitSecondDesc
	performanceSchemaReplicationApplierStatsByWorkerApplyingTransactionStartApplySecondDesc
	performanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionEndApplySecondDesc
}

func NewScrapePerfReplicationApplierStatsByWorker() *ScrapePerfReplicationApplierStatsByWorker {
	return &ScrapePerfReplicationApplierStatsByWorker{
		//instance: instance,
		performanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionOriginalCommitSecondDesc:  *newperformanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionOriginalCommitSecondDesc(),
		performanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionImmediateCommitSecondDesc: *newperformanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionImmediateCommitSecondDesc(),
		performanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionStartApplySecondDesc:      *newperformanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionStartApplySecondDesc(),
		performanceSchemaReplicationApplierStatsByWorkerApplyingTransactionOriginalCommitSecondDesc:     *newperformanceSchemaReplicationApplierStatsByWorkerApplyingTransactionOriginalCommitSecondDesc(),
		performanceSchemaReplicationApplierStatsByWorkerApplyingTransactionImmediateCommitSecondDesc:    *newperformanceSchemaReplicationApplierStatsByWorkerApplyingTransactionImmediateCommitSecondDesc(),
		performanceSchemaReplicationApplierStatsByWorkerApplyingTransactionStartApplySecondDesc:         *newperformanceSchemaReplicationApplierStatsByWorkerApplyingTransactionStartApplySecondDesc(),
		performanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionEndApplySecondDesc:        *newperformanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionEndApplySecondDesc(),
	}

}

func (qd ScrapePerfReplicationApplierStatsByWorker) Collect(ch chan<- prometheus.Metric) {
	qd.instance = *GetInstance()

	if err := qd.instance.Ping(); err != nil {
		logrus.Errorf("ping mysql instance error: %s", err)
		return
	}
	db := instance.GetDB()
	rows, err := db.Query(perfReplicationApplierStatsByWorkerQuery)
	if err != nil {
		logrus.Error(err)
		return
	}
	defer rows.Close()
	var (
		channelName                                  string
		workerId                                     string
		lastAppliedTransactionOriginalCommit         string
		lastAppliedTransactionImmediateCommit        string
		lastAppliedTransactionStartApply             string
		lastAppliedTransactionEndApply               string
		applyingTransactionOriginalCommit            string
		applyingTransactionImmediateCommit           string
		applyingTransactionStartApply                string
		lastAppliedTransactionOriginalCommitSeconds  float64
		lastAppliedTransactionImmediateCommitSeconds float64
		lastAppliedTransactionStartApplySeconds      float64
		lastAppliedTransactionEndApplySeconds        float64
		applyingTransactionOriginalCommitSeconds     float64
		applyingTransactionImmediateCommitSeconds    float64
		applyingTransactionStartApplySeconds         float64
	)
	for rows.Next() {
		err := rows.Scan(
			&channelName,
			&workerId,
			&lastAppliedTransactionOriginalCommit,
			&lastAppliedTransactionImmediateCommit,
			&lastAppliedTransactionStartApply,
			&lastAppliedTransactionEndApply,
			&applyingTransactionOriginalCommit,
			&applyingTransactionImmediateCommit,
			&applyingTransactionStartApply,
		)
		if err != nil {
			logrus.Error(err)
			return
		}
		lastAppliedTransactionOriginalCommitTime, err := time.Parse(timeLayout, lastAppliedTransactionOriginalCommit)
		if err != nil {
			lastAppliedTransactionOriginalCommitTime = time.Time{}
		}
		if !lastAppliedTransactionOriginalCommitTime.IsZero() {
			lastAppliedTransactionOriginalCommitSeconds = float64(lastAppliedTransactionOriginalCommitTime.UnixNano()) / 1e9
		} else {
			lastAppliedTransactionOriginalCommitSeconds = 0
		}
		qd.performanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionOriginalCommitSecondDesc.
			collect(
				ch,
				lastAppliedTransactionOriginalCommitSeconds,
				[]string{
					channelName,
					workerId})
		lastAppliedTransactionStartApplyTime, err := time.Parse(timeLayout, lastAppliedTransactionStartApply)
		if err != nil {
			lastAppliedTransactionStartApplyTime = time.Time{}
		}
		if !lastAppliedTransactionStartApplyTime.IsZero() {
			lastAppliedTransactionStartApplySeconds = float64(lastAppliedTransactionStartApplyTime.UnixNano()) / 1e9
		} else {
			lastAppliedTransactionStartApplySeconds = 0
		}
		qd.performanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionStartApplySecondDesc.
			collect(
				ch,
				lastAppliedTransactionStartApplySeconds,
				[]string{
					channelName,
					workerId})
		lastAppliedTransactionEndApplyTime, err := time.Parse(timeLayout, lastAppliedTransactionEndApply)
		if err != nil {
			lastAppliedTransactionEndApplyTime = time.Time{}
		}
		if !lastAppliedTransactionEndApplyTime.IsZero() {
			lastAppliedTransactionEndApplySeconds = float64(lastAppliedTransactionEndApplyTime.UnixNano()) / 1e9
		} else {
			lastAppliedTransactionEndApplySeconds = 0
		}
		qd.performanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionEndApplySecondDesc.
			collect(
				ch,
				lastAppliedTransactionEndApplySeconds,
				[]string{
					channelName,
					workerId})
		applyingTransactionOriginalCommitTime, err := time.Parse(timeLayout, applyingTransactionOriginalCommit)
		if err != nil {
			applyingTransactionOriginalCommitTime = time.Time{}
		}
		if !applyingTransactionOriginalCommitTime.IsZero() {
			applyingTransactionOriginalCommitSeconds = float64(applyingTransactionOriginalCommitTime.UnixNano()) / 1e9
		} else {
			applyingTransactionOriginalCommitSeconds = 0
		}
		qd.performanceSchemaReplicationApplierStatsByWorkerApplyingTransactionOriginalCommitSecondDesc.
			collect(
				ch,
				applyingTransactionOriginalCommitSeconds,
				[]string{
					channelName,
					workerId})
		applyingTransactionImmediateCommitTime, err := time.Parse(timeLayout, applyingTransactionImmediateCommit)
		if err != nil {
			applyingTransactionImmediateCommitTime = time.Time{}
		}
		if !applyingTransactionImmediateCommitTime.IsZero() {
			applyingTransactionImmediateCommitSeconds = float64(applyingTransactionImmediateCommitTime.UnixNano()) / 1e9
		} else {
			applyingTransactionImmediateCommitSeconds = 0
		}
		qd.performanceSchemaReplicationApplierStatsByWorkerApplyingTransactionImmediateCommitSecondDesc.
			collect(
				ch,
				applyingTransactionImmediateCommitSeconds,
				[]string{
					channelName,
					workerId})
		applyingTransactionStartApplyTime, err := time.Parse(timeLayout, applyingTransactionStartApply)
		if err != nil {
			applyingTransactionStartApplyTime = time.Time{}
		}
		if !applyingTransactionStartApplyTime.IsZero() {
			applyingTransactionStartApplySeconds = float64(applyingTransactionStartApplyTime.UnixNano()) / 1e9
		} else {
			applyingTransactionStartApplySeconds = 0
		}
		qd.performanceSchemaReplicationApplierStatsByWorkerApplyingTransactionStartApplySecondDesc.
			collect(
				ch,
				applyingTransactionStartApplySeconds,
				[]string{
					channelName,
					workerId})

		lastAppliedTransactionImmediateCommitTime, err := time.Parse(timeLayout, lastAppliedTransactionImmediateCommit)
		if err != nil {
			lastAppliedTransactionImmediateCommitTime = time.Time{}
		}
		if !lastAppliedTransactionImmediateCommitTime.IsZero() {
			lastAppliedTransactionImmediateCommitSeconds = float64(lastAppliedTransactionImmediateCommitTime.UnixNano()) / 1e9
		} else {
			lastAppliedTransactionImmediateCommitSeconds = 0
		}
		qd.performanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionImmediateCommitSecondDesc.
			collect(
				ch,
				lastAppliedTransactionImmediateCommitSeconds,
				[]string{
					channelName,
					workerId})
	}
}

type performanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionOriginalCommitSecondDesc struct {
	*baseMetrics
}

func newperformanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionOriginalCommitSecondDesc() *performanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionOriginalCommitSecondDesc {
	return &performanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionOriginalCommitSecondDesc{
		NewMetrics(
			"perf_schema_last_applied_transaction_original_commit_timestamp_seconds",
			"A timestamp shows when the last transaction applied "+
				"by this worker was committed on the original master.",
			[]string{
				"channel_name",
				"member_id"})}
}

func (qd *performanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionOriginalCommitSecondDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionImmediateCommitSecondDesc struct {
	*baseMetrics
}

func newperformanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionImmediateCommitSecondDesc() *performanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionImmediateCommitSecondDesc {
	return &performanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionImmediateCommitSecondDesc{
		NewMetrics(
			"perf_schema_last_applied_transaction_immediate_commit_timestamp_seconds",
			"A timestamp shows when the last transaction applied "+
				"by this worker was committed on the immediate master.",
			[]string{
				"channel_name",
				"member_id"})}
}

func (qd *performanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionImmediateCommitSecondDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionStartApplySecondDesc struct {
	*baseMetrics
}

func newperformanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionStartApplySecondDesc() *performanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionStartApplySecondDesc {
	return &performanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionStartApplySecondDesc{
		NewMetrics(
			"perf_schema_last_applied_transaction_start_apply_timestamp_seconds",
			"A timestamp shows when the last transaction applied "+
				"by this worker was started applying.",
			[]string{
				"channel_name",
				"member_id"})}
}

func (qd *performanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionStartApplySecondDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionEndApplySecondDesc struct {
	*baseMetrics
}

func newperformanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionEndApplySecondDesc() *performanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionEndApplySecondDesc {
	return &performanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionEndApplySecondDesc{
		NewMetrics(
			"perf_schema_last_applied_transaction_end_apply_timestamp_seconds",
			"A timestamp shows when the last transaction applied "+
				"by this worker was finished applying.",
			[]string{
				"channel_name",
				"member_id"})}
}

func (qd *performanceSchemaReplicationApplierStatsByWorkerLastAppliedTransactionEndApplySecondDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaReplicationApplierStatsByWorkerApplyingTransactionOriginalCommitSecondDesc struct {
	*baseMetrics
}

func newperformanceSchemaReplicationApplierStatsByWorkerApplyingTransactionOriginalCommitSecondDesc() *performanceSchemaReplicationApplierStatsByWorkerApplyingTransactionOriginalCommitSecondDesc {
	return &performanceSchemaReplicationApplierStatsByWorkerApplyingTransactionOriginalCommitSecondDesc{
		NewMetrics(
			"perf_schema_applying_transaction_original_commit_timestamp_seconds",
			"A timestamp shows when the last transaction applied "+
				"by this worker was committed on the original master.",
			[]string{
				"channel_name",
				"member_id"})}
}

func (qd *performanceSchemaReplicationApplierStatsByWorkerApplyingTransactionOriginalCommitSecondDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaReplicationApplierStatsByWorkerApplyingTransactionImmediateCommitSecondDesc struct {
	*baseMetrics
}

func newperformanceSchemaReplicationApplierStatsByWorkerApplyingTransactionImmediateCommitSecondDesc() *performanceSchemaReplicationApplierStatsByWorkerApplyingTransactionImmediateCommitSecondDesc {
	return &performanceSchemaReplicationApplierStatsByWorkerApplyingTransactionImmediateCommitSecondDesc{
		NewMetrics(
			"perf_schema_applying_transaction_immediate_commit_timestamp_seconds",
			"A timestamp shows when the last transaction applied "+
				"by this worker was committed on the immediate master.",
			[]string{
				"channel_name",
				"member_id"})}
}

func (qd *performanceSchemaReplicationApplierStatsByWorkerApplyingTransactionImmediateCommitSecondDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaReplicationApplierStatsByWorkerApplyingTransactionStartApplySecondDesc struct {
	*baseMetrics
}

func newperformanceSchemaReplicationApplierStatsByWorkerApplyingTransactionStartApplySecondDesc() *performanceSchemaReplicationApplierStatsByWorkerApplyingTransactionStartApplySecondDesc {
	return &performanceSchemaReplicationApplierStatsByWorkerApplyingTransactionStartApplySecondDesc{
		NewMetrics(
			"perf_schema_applying_transaction_start_apply_timestamp_seconds",
			"A timestamp shows when the last transaction applied "+
				"by this worker was started applying.",
			[]string{
				"channel_name",
				"member_id"})}
}

func (qd *performanceSchemaReplicationApplierStatsByWorkerApplyingTransactionStartApplySecondDesc) Collect(
	ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

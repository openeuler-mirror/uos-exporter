package metrics

import (
	"database/sql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"mysqld_exporter/internal/exporter"
	"mysqld_exporter/internal/mysql"
	"strconv"
)

const (
	perfReplicationGroupMemberStatsQuery = `
	SELECT * FROM performance_schema.replication_group_member_stats WHERE MEMBER_ID=@@server_uuid
`
)

func init() {
	exporter.Register(
		NewScrapePerfReplicationGroupMemberStats())
}

type ScrapePerfReplicationGroupMemberStats struct {
	instance mysql.Instance
	performanceSchemaTransactionsInQueue
	performanceSchemaTransactionsCheckedTotal
	performanceSchemaTransactionsDetectedTotal
	performanceSchemaTransactionsRowsValidatingTotal
	performanceSchemaTransactionsRemoteInApplierQueue
	performanceSchemaTransactionsRemoteAppliedTotal
	performanceSchemaTransactionsLocalProposedTotal
	performanceSchemaTransactionsLocalRollbackTotal
	//performanceSchemaTransactionsRemoteInApplierQueue
}

func NewScrapePerfReplicationGroupMemberStats() *ScrapePerfReplicationGroupMemberStats {
	return &ScrapePerfReplicationGroupMemberStats{
		//instance:                                          instance,
		performanceSchemaTransactionsInQueue:              *newperformanceSchemaTransactionsInQueue(),
		performanceSchemaTransactionsCheckedTotal:         *newperformanceSchemaTransactionsCheckedTotal(),
		performanceSchemaTransactionsDetectedTotal:        *newperformanceSchemaTransactionsDetectedTotal(),
		performanceSchemaTransactionsRowsValidatingTotal:  *newperformanceSchemaTransactionsRowsValidatingTotal(),
		performanceSchemaTransactionsRemoteInApplierQueue: *newperformanceSchemaTransactionsRemoteInApplierQueue(),
		performanceSchemaTransactionsRemoteAppliedTotal:   *newperformanceSchemaTransactionsRemoteAppliedTotal(),
		performanceSchemaTransactionsLocalProposedTotal:   *newperformanceSchemaTransactionsLocalProposedTotal(),
		performanceSchemaTransactionsLocalRollbackTotal:   *newperformanceSchemaTransactionsLocalRollbackTotal(),
		//performanceSchemaTransactionsRowsValidatingTotal:  *newperformanceSchemaTransactionsRowsValidatingTotal(),
	}

}

func (qd ScrapePerfReplicationGroupMemberStats) Collect(ch chan<- prometheus.Metric) {
	qd.instance = *GetInstance()

	logrus.Info("Start collecting ScrapePerfReplicationGroupMemberStats metrics")
	db := instance.GetDB()
	rows, err := db.Query(perfReplicationGroupMemberStatsQuery)
	if err != nil {
		logrus.Error(err)
		return
	}
	defer rows.Close()
	var columnNames []string
	if columnNames, err = rows.Columns(); err != nil {
		logrus.Error(err)
		return
	}
	var scanArgs = make([]interface{}, len(columnNames))
	for i := range scanArgs {
		scanArgs[i] = &sql.RawBytes{}
	}
	for rows.Next() {
		err := rows.Scan(scanArgs...)
		if err != nil {
			logrus.Error(err)
			return
		}
		for i, columnName := range columnNames {

			if columnName == "COUNT_TRANSACTIONS_IN_QUEUE" {
				value, err := strconv.ParseFloat(string(*scanArgs[i].(*sql.RawBytes)), 64)
				if err != nil {
					logrus.Error(err)
					return
				}
				qd.performanceSchemaTransactionsInQueue.Collect(ch,
					value,
					[]string{})
			} else if columnName == "COUNT_TRANSACTIONS_CHECKED" {
				value, err := strconv.ParseFloat(string(*scanArgs[i].(*sql.RawBytes)), 64)
				if err != nil {
					logrus.Error(err)
					return
				}
				qd.performanceSchemaTransactionsCheckedTotal.Collect(ch,
					value,
					[]string{})
			} else if columnName == "COUNT_TRANSACTIONS_DETECTED" {
				value, err := strconv.ParseFloat(string(*scanArgs[i].(*sql.RawBytes)), 64)
				if err != nil {
					logrus.Error(err)
					return
				}
				qd.performanceSchemaTransactionsCheckedTotal.Collect(ch,
					value,
					[]string{})
			} else if columnName == "COUNT_TRANSACTIONS_ROWS_VALIDATING" {
				value, err := strconv.ParseFloat(string(*scanArgs[i].(*sql.RawBytes)), 64)
				if err != nil {
					logrus.Error(err)
					return
				}
				qd.performanceSchemaTransactionsRemoteAppliedTotal.Collect(ch,
					value,
					[]string{})
			} else if columnName == "COUNT_TRANSACTIONS_REMOTE_IN_APPLIER_QUEUE" {
				value, err := strconv.ParseFloat(string(*scanArgs[i].(*sql.RawBytes)), 64)
				if err != nil {
					logrus.Error(err)
					return
				}
				qd.performanceSchemaTransactionsRemoteInApplierQueue.Collect(ch,
					value,
					[]string{})
			} else if columnName == "COUNT_TRANSACTIONS_REMOTE_APPLIED" {
				value, err := strconv.ParseFloat(string(*scanArgs[i].(*sql.RawBytes)), 64)
				if err != nil {
					logrus.Error(err)
					return
				}
				qd.performanceSchemaTransactionsRemoteAppliedTotal.Collect(ch,
					value,
					[]string{})
			} else if columnName == "COUNT_TRANSACTIONS_LOCAL_PROPOSED" {
				value, err := strconv.ParseFloat(string(*scanArgs[i].(*sql.RawBytes)), 64)
				if err != nil {
					logrus.Error(err)
					return
				}
				qd.performanceSchemaTransactionsLocalProposedTotal.Collect(ch,
					value,
					[]string{})
			} else if columnName == "COUNT_TRANSACTIONS_LOCAL_ROLLBACK" {
				value, err := strconv.ParseFloat(string(*scanArgs[i].(*sql.RawBytes)), 64)
				if err != nil {
					logrus.Error(err)
					return
				}
				qd.performanceSchemaTransactionsLocalRollbackTotal.Collect(ch,
					value,
					[]string{})
			}

		}
	}
}

type performanceSchemaTransactionsInQueue struct {
	*baseMetrics
}

func newperformanceSchemaTransactionsInQueue() *performanceSchemaTransactionsInQueue {
	return &performanceSchemaTransactionsInQueue{
		NewMetrics(
			"perf_schema_transactions_in_queue",
			"The number of transactions in the queue pending conflict detection checks.",
			nil)}
}

func (qd *performanceSchemaTransactionsInQueue) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaTransactionsCheckedTotal struct {
	*baseMetrics
}

func newperformanceSchemaTransactionsCheckedTotal() *performanceSchemaTransactionsCheckedTotal {
	return &performanceSchemaTransactionsCheckedTotal{
		NewMetrics(
			"perf_schema_transactions_checked_total",
			"The number of transactions checked for conflicts.",
			nil)}
}
func (qd *performanceSchemaTransactionsCheckedTotal) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaTransactionsDetectedTotal struct {
	*baseMetrics
}

func newperformanceSchemaTransactionsDetectedTotal() *performanceSchemaTransactionsDetectedTotal {
	return &performanceSchemaTransactionsDetectedTotal{
		NewMetrics(
			"perf_schema_transactions_detected_total",
			"The number of transactions detected as conflicting.",
			nil)}
}
func (qd *performanceSchemaTransactionsDetectedTotal) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaTransactionsRowsValidatingTotal struct {
	*baseMetrics
}

func newperformanceSchemaTransactionsRowsValidatingTotal() *performanceSchemaTransactionsRowsValidatingTotal {
	return &performanceSchemaTransactionsRowsValidatingTotal{
		NewMetrics(
			"perf_schema_transactions_rows_validating_total",
			"Number of transaction rows which can be used "+
				"for certification, but have not been garbage collected.",
			nil)}
}

func (qd *performanceSchemaTransactionsRowsValidatingTotal) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaTransactionsRemoteInApplierQueue struct {
	*baseMetrics
}

func newperformanceSchemaTransactionsRemoteInApplierQueue() *performanceSchemaTransactionsRemoteInApplierQueue {
	return &performanceSchemaTransactionsRemoteInApplierQueue{
		NewMetrics(
			"perf_schema_transactions_remote_in_applier_queue",
			"The number of transactions that this member has received"+
				" from the replication group which are waiting to be applied.",
			nil)}
}
func (qd *performanceSchemaTransactionsRemoteInApplierQueue) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaTransactionsRemoteAppliedTotal struct {
	*baseMetrics
}

func newperformanceSchemaTransactionsRemoteAppliedTotal() *performanceSchemaTransactionsRemoteAppliedTotal {
	return &performanceSchemaTransactionsRemoteAppliedTotal{
		NewMetrics(
			"perf_schema_transactions_remote_applied_total",
			"Number of transactions this member has received from the group and applied.",
			nil)}
}
func (qd *performanceSchemaTransactionsRemoteAppliedTotal) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaTransactionsLocalProposedTotal struct {
	*baseMetrics
}

func newperformanceSchemaTransactionsLocalProposedTotal() *performanceSchemaTransactionsLocalProposedTotal {
	return &performanceSchemaTransactionsLocalProposedTotal{
		NewMetrics(
			"perf_schema_transactions_local_proposed_total",
			"Number of transactions which originated on this member and were sent to the group.",
			nil)}
}
func (qd *performanceSchemaTransactionsLocalProposedTotal) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaTransactionsLocalRollbackTotal struct {
	*baseMetrics
}

func newperformanceSchemaTransactionsLocalRollbackTotal() *performanceSchemaTransactionsLocalRollbackTotal {
	return &performanceSchemaTransactionsLocalRollbackTotal{
		NewMetrics(
			"perf_schema_transactions_local_rollback_total",
			"Number of transactions which originated on this member and were rolled back by the group.",
			nil)}
}
func (qd *performanceSchemaTransactionsLocalRollbackTotal) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

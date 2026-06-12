package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"mysqld_exporter/internal/exporter"
	"mysqld_exporter/internal/mysql"
)

const (
	userstatCheckQuery = `SHOW GLOBAL VARIABLES WHERE Variable_Name='userstat'
		OR Variable_Name='userstat_running'`
	userStatQuery  = `SELECT * FROM information_schema.user_statistics`
	userStatResult = `
`
)

func init() {
	exporter.Register(
		NewScrapeUserStat())
}

type ScrapeUserStat struct {
	instance mysql.Instance
	info_schema_user_statistics_total_connections
	info_schema_user_statistics_concurrent_connections
	info_schema_user_statistics_connected_time_seconds_total
	info_schema_user_statistics_busy_seconds_total
	info_schema_user_statistics_cpu_time_seconds_total
	info_schema_user_statistics_bytes_received_total
	info_schema_user_statistics_bytes_sent_total
	info_schema_user_statistics_binlog_bytes_written_total
	info_schema_user_statistics_rows_read_total
	info_schema_user_statistics_rows_sent_total
	info_schema_user_statistics_rows_deleted_total
	info_schema_user_statistics_rows_inserted_total
	info_schema_user_statistics_rows_fetched_total
	info_schema_user_statistics_rows_updated_total
	info_schema_user_statistics_table_rows_read_total
	info_schema_user_statistics_select_commands_total
	info_schema_user_statistics_update_commands_total
	info_schema_user_statistics_other_commands_total
	info_schema_user_statistics_commit_transactions_total
	info_schema_user_statistics_rollback_transactions_total
	info_schema_user_statistics_denied_connections_total
	info_schema_user_statistics_lost_connections_total
	info_schema_user_statistics_access_denied_total
	info_schema_user_statistics_empty_queries_total
	info_schema_user_statistics_total_ssl_connections_total
}

func NewScrapeUserStat() *ScrapeUserStat {
	return &ScrapeUserStat{
		//instance: instance,
		info_schema_user_statistics_total_connections:            *Newinfo_schema_user_statistics_total_connections(),
		info_schema_user_statistics_concurrent_connections:       *Newinfo_schema_user_statistics_concurrent_connections(),
		info_schema_user_statistics_connected_time_seconds_total: *Newinfo_schema_user_statistics_connected_time_seconds_total(),
		info_schema_user_statistics_busy_seconds_total:           *Newinfo_schema_user_statistics_busy_seconds_total(),
		info_schema_user_statistics_cpu_time_seconds_total:       *Newinfo_schema_user_statistics_cpu_time_seconds_total(),
		info_schema_user_statistics_bytes_received_total:         *Newinfo_schema_user_statistics_bytes_received_total(),
		info_schema_user_statistics_bytes_sent_total:             *Newinfo_schema_user_statistics_bytes_sent_total(),
		info_schema_user_statistics_binlog_bytes_written_total:   *Newinfo_schema_user_statistics_binlog_bytes_written_total(),
		info_schema_user_statistics_rows_read_total:              *Newinfo_schema_user_statistics_rows_read_total(),
		info_schema_user_statistics_rows_sent_total:              *Newinfo_schema_user_statistics_rows_sent_total(),
		info_schema_user_statistics_rows_deleted_total:           *Newinfo_schema_user_statistics_rows_deleted_total(),
		info_schema_user_statistics_rows_inserted_total:          *Newinfo_schema_user_statistics_rows_inserted_total(),
		info_schema_user_statistics_rows_fetched_total:           *Newinfo_schema_user_statistics_rows_fetched_total(),
		info_schema_user_statistics_rows_updated_total:           *Newinfo_schema_user_statistics_rows_updated_total(),
		info_schema_user_statistics_table_rows_read_total:        *Newinfo_schema_user_statistics_table_rows_read_total(),
		info_schema_user_statistics_select_commands_total:        *Newinfo_schema_user_statistics_select_commands_total(),
		info_schema_user_statistics_update_commands_total:        *Newinfo_schema_user_statistics_update_commands_total(),
		info_schema_user_statistics_other_commands_total:         *Newinfo_schema_user_statistics_other_commands_total(),
		info_schema_user_statistics_commit_transactions_total:    *Newinfo_schema_user_statistics_commit_transactions_total(),
		info_schema_user_statistics_rollback_transactions_total:  *Newinfo_schema_user_statistics_rollback_transactions_total(),
		info_schema_user_statistics_denied_connections_total:     *Newinfo_schema_user_statistics_denied_connections_total(),
		info_schema_user_statistics_lost_connections_total:       *Newinfo_schema_user_statistics_lost_connections_total(),
		info_schema_user_statistics_access_denied_total:          *Newinfo_schema_user_statistics_access_denied_total(),
		info_schema_user_statistics_empty_queries_total:          *Newinfo_schema_user_statistics_empty_queries_total(),
		info_schema_user_statistics_total_ssl_connections_total:  *Newinfo_schema_user_statistics_total_ssl_connections_total(),
	}
}

func (qd ScrapeUserStat) Collect(ch chan<- prometheus.Metric) {
	qd.instance = *GetInstance()

	if err := qd.instance.Ping(); err != nil {
		logrus.Errorf("ping mysql instance error: %s", err)
		return
	}
	db := instance.GetDB()
	rows, err := db.Query(userstatCheckQuery)
	if err != nil {
		logrus.Debugf("failed check mysql instance userstat: %s",
			err)
		return
	}
	defer rows.Close()
	var (
		varName  string
		varValue string
	)
	err = rows.Scan(&varName, &varValue)
	if err != nil {
		logrus.Debugf("failed to scan mysql instance userstat: %s",
			err)
		return
	}
	if varValue == "OFF" {
		logrus.Debugf("mysql instance userstat is disabled")
		return
	}
	informationSchemaUserStatisticsRows, err := db.Query(userStatQuery)
	if err != nil {
		logrus.Errorf("query mysql instance userstat error: %s", err)
		return
	}
	defer informationSchemaUserStatisticsRows.Close()
	var columnNames []string
	columnNames, err = informationSchemaUserStatisticsRows.Columns()
	if err != nil {
		logrus.Errorf("get mysql instance userstat column names error: %s", err)
		return
	}
	var user string
	var userStatData = make([]float64, len(columnNames)-1)
	var userStatScanArgs = make([]interface{}, len(columnNames))
	userStatScanArgs[0] = &user
	for i := range userStatData {
		userStatScanArgs[i+1] = &userStatData[i]
	}
	for informationSchemaUserStatisticsRows.Next() {
		err = informationSchemaUserStatisticsRows.Scan(userStatScanArgs...)
		if err != nil {
			logrus.Errorf("scan mysql instance userstat error: %s", err)
			return
		}

		for idx, columnName := range columnNames[1:] {
			if columnName == "TOTAL_CONNECTIONS" {
				qd.info_schema_user_statistics_total_connections.Collect(
					ch,
					float64(userStatData[idx]),
					[]string{
						user})
				continue
			}
			if columnName == "CONCURRENT_CONNECTIONS" {
				qd.info_schema_user_statistics_concurrent_connections.Collect(
					ch,
					float64(userStatData[idx]),
					[]string{
						user})
				continue
			}

			if columnName == "CONNECTED_TIME" {
				qd.info_schema_user_statistics_connected_time_seconds_total.Collect(
					ch,
					float64(userStatData[idx]),
					[]string{
						user})
				continue
			}

			if columnName == "BUSY_TIME" {
				qd.info_schema_user_statistics_busy_seconds_total.Collect(
					ch,
					float64(userStatData[idx]),
					[]string{
						user})
				continue
			}

			if columnName == "CPU_TIME" {
				qd.info_schema_user_statistics_cpu_time_seconds_total.Collect(
					ch,
					float64(userStatData[idx]),
					[]string{
						user})
				continue
			}

			if columnName == "BYTES_RECEIVED" {
				qd.info_schema_user_statistics_bytes_received_total.Collect(
					ch,
					float64(userStatData[idx]),
					[]string{
						user})
				continue
			}

			if columnName == "BYTES_SENT" {
				qd.info_schema_user_statistics_bytes_sent_total.Collect(
					ch,
					float64(userStatData[idx]),
					[]string{
						user})
				continue
			}

			if columnName == "BINLOG_BYTES_WRITTEN" {
				qd.info_schema_user_statistics_binlog_bytes_written_total.Collect(
					ch,
					float64(userStatData[idx]),
					[]string{
						user})
				continue
			}
			if columnName == "ROWS_READ" {
				qd.info_schema_user_statistics_rows_read_total.Collect(
					ch,
					float64(userStatData[idx]),
					[]string{
						user})
				continue
			}

			if columnName == "ROWS_SENT" {
				qd.info_schema_user_statistics_rows_sent_total.Collect(
					ch,
					float64(userStatData[idx]),
					[]string{
						user})
				continue
			}

			if columnName == "ROWS_DELETED" {
				qd.info_schema_user_statistics_rows_deleted_total.Collect(
					ch,
					float64(userStatData[idx]),
					[]string{
						user})
				continue
			}

			if columnName == "ROWS_FETCHED" {
				qd.info_schema_user_statistics_rows_fetched_total.Collect(
					ch,
					float64(userStatData[idx]),
					[]string{
						user})
				continue
			}

			if columnName == "ROWS_UPDATED" {
				qd.info_schema_user_statistics_rows_updated_total.Collect(
					ch,
					float64(userStatData[idx]),
					[]string{
						user})
				continue
			}

			if columnName == "TABLE_ROWS_READ" {
				qd.info_schema_user_statistics_table_rows_read_total.Collect(
					ch,
					float64(userStatData[idx]),
					[]string{
						user})
				continue
			}

			if columnName == "SELECT_COMMANDS" {
				qd.info_schema_user_statistics_select_commands_total.Collect(
					ch,
					float64(userStatData[idx]),
					[]string{
						user})
				continue
			}

			if columnName == "UPDATE_COMMANDS" {
				qd.info_schema_user_statistics_update_commands_total.Collect(
					ch,
					float64(userStatData[idx]),
					[]string{
						user})
				continue
			}

			if columnName == "OTHER_COMMANDS" {
				qd.info_schema_user_statistics_other_commands_total.Collect(
					ch,
					float64(userStatData[idx]),
					[]string{
						user})
				continue
			}

			if columnName == "COMMIT_TRANSACTIONS" {
				qd.info_schema_user_statistics_commit_transactions_total.Collect(
					ch,
					float64(userStatData[idx]),
					[]string{
						user})
				continue
			}

			if columnName == "ROLLBACK_TRANSACTIONS" {
				qd.info_schema_user_statistics_rollback_transactions_total.Collect(
					ch,
					float64(userStatData[idx]),
					[]string{
						user})
				continue
			}

			if columnName == "DENIED_CONNECTIONS" {
				qd.info_schema_user_statistics_denied_connections_total.Collect(
					ch,
					float64(userStatData[idx]),
					[]string{
						user})
				continue
			}

			if columnName == "LOST_CONNECTIONS" {
				qd.info_schema_user_statistics_lost_connections_total.Collect(
					ch,
					float64(userStatData[idx]),
					[]string{
						user})
				continue
			}

			if columnName == "ACCESS_DENIED" {
				qd.info_schema_user_statistics_access_denied_total.Collect(
					ch,
					float64(userStatData[idx]),
					[]string{
						user})
				continue
			}

			if columnName == "EMPTY_QUERIES" {
				qd.info_schema_user_statistics_empty_queries_total.Collect(
					ch,
					float64(userStatData[idx]),
					[]string{
						user})
				continue
			}

			if columnName == "TOTAL_SSL_CONNECTIONS" {
				qd.info_schema_user_statistics_total_ssl_connections_total.Collect(
					ch,
					float64(userStatData[idx]),
					[]string{
						user})
				continue
			}

		}
	}
}

type info_schema_user_statistics_total_connections struct {
	*baseMetrics
}

func Newinfo_schema_user_statistics_total_connections() *info_schema_user_statistics_total_connections {
	return &info_schema_user_statistics_total_connections{
		NewMetrics(
			"info_schema_user_statistics_total_connections",
			"The number of connections created for this user.",
			[]string{
				"user"})}
}
func (qd *info_schema_user_statistics_total_connections) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type info_schema_user_statistics_concurrent_connections struct {
	*baseMetrics
}

func Newinfo_schema_user_statistics_concurrent_connections() *info_schema_user_statistics_concurrent_connections {
	return &info_schema_user_statistics_concurrent_connections{
		NewMetrics(
			"info_schema_user_statistics_concurrent_connections",
			"The number of concurrent connections for this user.",
			[]string{
				"user"})}
}

func (qd *info_schema_user_statistics_concurrent_connections) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type info_schema_user_statistics_connected_time_seconds_total struct {
	*baseMetrics
}

func Newinfo_schema_user_statistics_connected_time_seconds_total() *info_schema_user_statistics_connected_time_seconds_total {
	return &info_schema_user_statistics_connected_time_seconds_total{
		NewMetrics(
			"info_schema_user_statistics_connected_time_seconds_total",
			"The total number of seconds the user has been connected.",
			[]string{
				"user"})}
}

func (qd *info_schema_user_statistics_connected_time_seconds_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type info_schema_user_statistics_busy_seconds_total struct {
	*baseMetrics
}

func Newinfo_schema_user_statistics_busy_seconds_total() *info_schema_user_statistics_busy_seconds_total {
	return &info_schema_user_statistics_busy_seconds_total{
		NewMetrics(
			"info_schema_user_statistics_busy_seconds_total",
			"The total number of seconds the user was in some non-idle state.",
			[]string{
				"user"})}
}
func (qd *info_schema_user_statistics_busy_seconds_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type info_schema_user_statistics_cpu_time_seconds_total struct {
	*baseMetrics
}

func Newinfo_schema_user_statistics_cpu_time_seconds_total() *info_schema_user_statistics_cpu_time_seconds_total {
	return &info_schema_user_statistics_cpu_time_seconds_total{
		NewMetrics(
			"info_schema_user_statistics_cpu_time_seconds_total",
			"The total number of seconds the user was in some non-idle state.",
			[]string{
				"user"})}
}

func (qd *info_schema_user_statistics_cpu_time_seconds_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type info_schema_user_statistics_bytes_received_total struct {
	*baseMetrics
}

func Newinfo_schema_user_statistics_bytes_received_total() *info_schema_user_statistics_bytes_received_total {
	return &info_schema_user_statistics_bytes_received_total{
		NewMetrics(
			"info_schema_user_statistics_bytes_received_total",
			"The total number of bytes received by this user.",
			[]string{
				"user"})}
}

func (qd *info_schema_user_statistics_bytes_received_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type info_schema_user_statistics_bytes_sent_total struct {
	*baseMetrics
}

func Newinfo_schema_user_statistics_bytes_sent_total() *info_schema_user_statistics_bytes_sent_total {
	return &info_schema_user_statistics_bytes_sent_total{
		NewMetrics(
			"info_schema_user_statistics_bytes_sent_total",
			"The total number of bytes sent by this user.",
			[]string{
				"user"})}
}
func (qd *info_schema_user_statistics_bytes_sent_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type info_schema_user_statistics_binlog_bytes_written_total struct {
	*baseMetrics
}

func Newinfo_schema_user_statistics_binlog_bytes_written_total() *info_schema_user_statistics_binlog_bytes_written_total {
	return &info_schema_user_statistics_binlog_bytes_written_total{
		NewMetrics(
			"info_schema_user_statistics_binlog_bytes_written_total",
			"The total number of bytes written to the binary log by this user.",
			[]string{
				"user"})}
}
func (qd *info_schema_user_statistics_binlog_bytes_written_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type info_schema_user_statistics_rows_read_total struct {
	*baseMetrics
}

func Newinfo_schema_user_statistics_rows_read_total() *info_schema_user_statistics_rows_read_total {
	return &info_schema_user_statistics_rows_read_total{
		NewMetrics(
			"info_schema_user_statistics_rows_read_total",
			"The total number of rows read by this user.",
			[]string{
				"user"})}
}
func (qd *info_schema_user_statistics_rows_read_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type info_schema_user_statistics_rows_sent_total struct {
	*baseMetrics
}

func Newinfo_schema_user_statistics_rows_sent_total() *info_schema_user_statistics_rows_sent_total {
	return &info_schema_user_statistics_rows_sent_total{
		NewMetrics(
			"info_schema_user_statistics_rows_sent_total",
			"The total number of rows sent by this user.",
			[]string{
				"user"})}
}

func (qd *info_schema_user_statistics_rows_sent_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type info_schema_user_statistics_rows_deleted_total struct {
	*baseMetrics
}

func Newinfo_schema_user_statistics_rows_deleted_total() *info_schema_user_statistics_rows_deleted_total {
	return &info_schema_user_statistics_rows_deleted_total{
		NewMetrics(
			"info_schema_user_statistics_rows_deleted_total",
			"The total number of rows deleted by this user.",
			[]string{
				"user"})}
}
func (qd *info_schema_user_statistics_rows_deleted_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type info_schema_user_statistics_rows_inserted_total struct {
	*baseMetrics
}

func Newinfo_schema_user_statistics_rows_inserted_total() *info_schema_user_statistics_rows_inserted_total {
	return &info_schema_user_statistics_rows_inserted_total{
		NewMetrics(
			"info_schema_user_statistics_rows_inserted_total",
			"The total number of rows inserted by this user.",
			[]string{
				"user"})}
}

func (qd *info_schema_user_statistics_rows_inserted_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type info_schema_user_statistics_rows_fetched_total struct {
	*baseMetrics
}

func Newinfo_schema_user_statistics_rows_fetched_total() *info_schema_user_statistics_rows_fetched_total {
	return &info_schema_user_statistics_rows_fetched_total{
		NewMetrics(
			"info_schema_user_statistics_rows_fetched_total",
			"The total number of rows fetched by this user.",
			[]string{
				"user"})}
}

func (qd *info_schema_user_statistics_rows_fetched_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type info_schema_user_statistics_rows_updated_total struct {
	*baseMetrics
}

func Newinfo_schema_user_statistics_rows_updated_total() *info_schema_user_statistics_rows_updated_total {
	return &info_schema_user_statistics_rows_updated_total{
		NewMetrics(
			"info_schema_user_statistics_rows_updated_total",
			"The total number of rows updated by this user.",
			[]string{
				"user"})}
}
func (qd *info_schema_user_statistics_rows_updated_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type info_schema_user_statistics_table_rows_read_total struct {
	*baseMetrics
}

func Newinfo_schema_user_statistics_table_rows_read_total() *info_schema_user_statistics_table_rows_read_total {
	return &info_schema_user_statistics_table_rows_read_total{
		NewMetrics(
			"info_schema_user_statistics_table_rows_read_total",
			"The total number of rows read by this user from tables.",
			[]string{
				"user"})}
}
func (qd *info_schema_user_statistics_table_rows_read_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type info_schema_user_statistics_select_commands_total struct {
	*baseMetrics
}

func Newinfo_schema_user_statistics_select_commands_total() *info_schema_user_statistics_select_commands_total {
	return &info_schema_user_statistics_select_commands_total{
		NewMetrics(
			"info_schema_user_statistics_select_commands_total",
			"The total number of SELECT commands executed by this user.",
			[]string{
				"user"})}
}
func (qd *info_schema_user_statistics_select_commands_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type info_schema_user_statistics_update_commands_total struct {
	*baseMetrics
}

func Newinfo_schema_user_statistics_update_commands_total() *info_schema_user_statistics_update_commands_total {
	return &info_schema_user_statistics_update_commands_total{
		NewMetrics(
			"info_schema_user_statistics_update_commands_total",
			"The total number of UPDATE commands executed by this user.",
			[]string{
				"user"})}
}
func (qd *info_schema_user_statistics_update_commands_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type info_schema_user_statistics_other_commands_total struct {
	*baseMetrics
}

func Newinfo_schema_user_statistics_other_commands_total() *info_schema_user_statistics_other_commands_total {
	return &info_schema_user_statistics_other_commands_total{
		NewMetrics(
			"info_schema_user_statistics_other_commands_total",
			"The total number of other commands executed by this user.",
			[]string{
				"user"})}
}
func (qd *info_schema_user_statistics_other_commands_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type info_schema_user_statistics_commit_transactions_total struct {
	*baseMetrics
}

func Newinfo_schema_user_statistics_commit_transactions_total() *info_schema_user_statistics_commit_transactions_total {
	return &info_schema_user_statistics_commit_transactions_total{
		NewMetrics(
			"info_schema_user_statistics_commit_transactions_total",
			"The total number of COMMIT commands executed by this user.",
			[]string{
				"user"})}
}
func (qd *info_schema_user_statistics_commit_transactions_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type info_schema_user_statistics_rollback_transactions_total struct {
	*baseMetrics
}

func Newinfo_schema_user_statistics_rollback_transactions_total() *info_schema_user_statistics_rollback_transactions_total {
	return &info_schema_user_statistics_rollback_transactions_total{
		NewMetrics(
			"info_schema_user_statistics_rollback_transactions_total",
			"The total number of ROLLBACK commands executed by this user.",
			[]string{
				"user"})}
}
func (qd *info_schema_user_statistics_rollback_transactions_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type info_schema_user_statistics_denied_connections_total struct {
	*baseMetrics
}

func Newinfo_schema_user_statistics_denied_connections_total() *info_schema_user_statistics_denied_connections_total {
	return &info_schema_user_statistics_denied_connections_total{
		NewMetrics(
			"info_schema_user_statistics_denied_connections_total",
			"The total number of denied connections by this user.",
			[]string{
				"user"})}
}
func (qd *info_schema_user_statistics_denied_connections_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type info_schema_user_statistics_lost_connections_total struct {
	*baseMetrics
}

func Newinfo_schema_user_statistics_lost_connections_total() *info_schema_user_statistics_lost_connections_total {
	return &info_schema_user_statistics_lost_connections_total{
		NewMetrics(
			"info_schema_user_statistics_lost_connections_total",
			"The total number of lost connections by this user.",
			[]string{
				"user"})}
}

func (qd *info_schema_user_statistics_lost_connections_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type info_schema_user_statistics_access_denied_total struct {
	*baseMetrics
}

func Newinfo_schema_user_statistics_access_denied_total() *info_schema_user_statistics_access_denied_total {
	return &info_schema_user_statistics_access_denied_total{
		NewMetrics(
			"info_schema_user_statistics_access_denied_total",
			"The total number of access denied errors by this user.",
			[]string{
				"user"})}
}

func (qd *info_schema_user_statistics_access_denied_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type info_schema_user_statistics_empty_queries_total struct {
	*baseMetrics
}

func Newinfo_schema_user_statistics_empty_queries_total() *info_schema_user_statistics_empty_queries_total {
	return &info_schema_user_statistics_empty_queries_total{
		NewMetrics(
			"info_schema_user_statistics_empty_queries_total",
			"The total number of empty queries by this user.",
			[]string{
				"user"})}
}
func (qd *info_schema_user_statistics_empty_queries_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type info_schema_user_statistics_total_ssl_connections_total struct {
	*baseMetrics
}

func Newinfo_schema_user_statistics_total_ssl_connections_total() *info_schema_user_statistics_total_ssl_connections_total {
	return &info_schema_user_statistics_total_ssl_connections_total{
		NewMetrics(
			"info_schema_user_statistics_total_ssl_connections_total",
			"The total number of SSL connections by this user.",
			[]string{
				"user"})}
}

func (qd *info_schema_user_statistics_total_ssl_connections_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

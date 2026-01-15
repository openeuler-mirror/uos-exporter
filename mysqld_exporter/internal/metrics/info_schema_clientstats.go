package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"mysqld_exporter/internal/exporter"
	"mysqld_exporter/internal/mysql"
)

const (
	clientStatQuery = `SELECT * FROM information_schema.client_statistics`
)

type ScrapeClientStat struct {
	instance mysql.Instance
	infoSchemaclient_statistics_total_connections
	infoSchemaclient_statistics_concurrent_connections
	infoSchemaclient_statistics_connected_time_seconds_total
	infoSchemaclient_statistics_busy_time_seconds_total
	infoSchemaclient_statistics_cpu_time_seconds_total
	infoSchemaclient_statistics_bytes_received_total
	infoSchemaclient_statistics_bytes_sent_total
	infoSchemaclient_statistics_binlog_bytes_written_total
	infoSchemaclient_statistics_rows_read_total
	infoSchemaclient_statistics_rows_sent_total
	infoSchemaclient_statistics_rows_deleted_total
	infoSchemaclient_statistics_rows_inserted_total
	infoSchemaclient_statistics_rows_fetched_total
	infoSchemaclient_statistics_rows_updated_total
	infoSchemaclient_statistics_table_rows_read_total
	infoSchemaclient_statistics_select_commands_total
	infoSchemaclient_statistics_update_commands_total
	infoSchemaclient_statistics_other_commands_total
	infoSchemaclient_statistics_commit_transactions_total
	infoSchemaclient_statistics_rollback_transactions_total
	infoSchemaclient_statistics_denied_connections_total
	infoSchemaclient_statistics_lost_connections_total
	infoSchemaclient_statistics_access_denied_total
	infoSchemaclient_statistics_empty_queries_total
	infoSchemaclient_statistics_total_ssl_connections_total
	infoSchemaclient_statistics_max_statement_time_exceeded_total
}

func init() {
	exporter.Register(
		NewScrapeClientStat())
}
func NewScrapeClientStat() *ScrapeClientStat {
	return &ScrapeClientStat{
		//instance: instance,
		infoSchemaclient_statistics_total_connections:                 *NewinfoSchemaclient_statistics_total_connections(),
		infoSchemaclient_statistics_concurrent_connections:            *NewinfoSchemaclient_statistics_concurrent_connections(),
		infoSchemaclient_statistics_connected_time_seconds_total:      *NewinfoSchemaclient_statistics_connected_time_seconds_total(),
		infoSchemaclient_statistics_busy_time_seconds_total:           *NewinfoSchemaclient_statistics_busy_time_seconds_total(),
		infoSchemaclient_statistics_cpu_time_seconds_total:            *NewinfoSchemaclient_statistics_cpu_time_seconds_total(),
		infoSchemaclient_statistics_bytes_received_total:              *NewinfoSchemaclient_statistics_bytes_received_total(),
		infoSchemaclient_statistics_bytes_sent_total:                  *NewinfoSchemaclient_statistics_bytes_sent_total(),
		infoSchemaclient_statistics_binlog_bytes_written_total:        *NewinfoSchemaclient_statistics_binlog_bytes_written_total(),
		infoSchemaclient_statistics_rows_read_total:                   *NewinfoSchemaclient_statistics_rows_read_total(),
		infoSchemaclient_statistics_rows_sent_total:                   *NewinfoSchemaclient_statistics_rows_sent_total(),
		infoSchemaclient_statistics_rows_deleted_total:                *NewinfoSchemaclient_statistics_rows_deleted_total(),
		infoSchemaclient_statistics_rows_inserted_total:               *NewinfoSchemaclient_statistics_rows_inserted_total(),
		infoSchemaclient_statistics_rows_fetched_total:                *NewinfoSchemaclient_statistics_rows_fetched_total(),
		infoSchemaclient_statistics_rows_updated_total:                *NewinfoSchemaclient_statistics_rows_updated_total(),
		infoSchemaclient_statistics_table_rows_read_total:             *NewinfoSchemaclient_statistics_table_rows_read_total(),
		infoSchemaclient_statistics_select_commands_total:             *NewinfoSchemaclient_statistics_select_commands_total(),
		infoSchemaclient_statistics_update_commands_total:             *NewinfoSchemaclient_statistics_update_commands_total(),
		infoSchemaclient_statistics_other_commands_total:              *NewinfoSchemaclient_statistics_other_commands_total(),
		infoSchemaclient_statistics_commit_transactions_total:         *NewinfoSchemaclient_statistics_commit_transactions_total(),
		infoSchemaclient_statistics_rollback_transactions_total:       *NewinfoSchemainfoSchemaclient_statistics_rollback_transactions_total(),
		infoSchemaclient_statistics_denied_connections_total:          *NewinfoSchemaclient_statistics_denied_connections_total(),
		infoSchemaclient_statistics_lost_connections_total:            *NewinfoSchemaclient_statistics_lost_connections_total(),
		infoSchemaclient_statistics_access_denied_total:               *NewinfoSchemaclient_statistics_access_denied_total(),
		infoSchemaclient_statistics_empty_queries_total:               *NewinfoSchemaclient_statistics_empty_queries_total(),
		infoSchemaclient_statistics_total_ssl_connections_total:       *NewinfoSchemaclient_statistics_total_ssl_connections_total(),
		infoSchemaclient_statistics_max_statement_time_exceeded_total: *NewinfoSchemaclient_statistics_max_statement_time_exceeded_total(),
	}
}
func (qd ScrapeClientStat) Collect(ch chan<- prometheus.Metric) {
	qd.instance = *GetInstance()

	var (
		varName string
		varVal  string
	)
	if err := qd.instance.Ping(); err != nil {
		logrus.Errorf("ping mysql instance error: %s", err)
		return
	}
	db := instance.GetDB()
	err := db.QueryRow(userstatCheckQuery).Scan(&varName, &varVal)
	if err != nil {
		logrus.Debug("Detailed client stats are not available.")
		return
	}
	if varVal == "OFF" {
		logrus.Debug("MySQL variable is OFF.", "var", varName)
		return
	}

	informationSchemaClientStatisticsRows, err := db.Query(clientStatQuery)
	if err != nil {
		logrus.Error(err)
		return
	}
	defer informationSchemaClientStatisticsRows.Close()
	columnNames, err := informationSchemaClientStatisticsRows.Columns()
	if err != nil {
		logrus.Error(err)
		return
	}
	var (
		client             string
		clientStatData     = make([]float64, len(columnNames)-1)
		clientStatScanArgs = make([]interface{}, len(columnNames))
	)
	clientStatScanArgs[0] = &client
	for i := range clientStatData {
		clientStatScanArgs[i+1] = &clientStatData[i]
	}
	for informationSchemaClientStatisticsRows.Next() {
		err = informationSchemaClientStatisticsRows.Scan(clientStatScanArgs...)
		if err != nil {
			logrus.Error(err)
			return
		}

		for idx, columnName := range columnNames[1:] {
			if columnName == "TOTAL_CONNECTIONS" {
				qd.infoSchemaclient_statistics_total_connections.Collect(ch,
					float64(clientStatData[idx]),
					[]string{
						client,
					})
			} else if columnName == "CONCURRENT_CONNECTIONS" {
				qd.infoSchemaclient_statistics_concurrent_connections.Collect(ch,
					float64(clientStatData[idx]),
					[]string{
						client,
					})
			} else if columnName == "CONNECTED_TIME" {
				qd.infoSchemaclient_statistics_connected_time_seconds_total.Collect(ch,
					float64(clientStatData[idx]),
					[]string{
						client,
					})
			} else if columnName == "BUSY_TIME" {
				qd.infoSchemaclient_statistics_busy_time_seconds_total.Collect(ch,
					float64(clientStatData[idx]),
					[]string{
						client,
					})
			} else if columnName == "CPU_TIME" {
				qd.infoSchemaclient_statistics_cpu_time_seconds_total.Collect(ch,
					float64(clientStatData[idx]),
					[]string{
						client,
					})
			} else if columnName == "BYTES_RECEIVED" {
				qd.infoSchemaclient_statistics_bytes_received_total.Collect(ch,
					float64(clientStatData[idx]),
					[]string{
						client,
					})
			} else if columnName == "BYTES_SENT" {
				qd.infoSchemaclient_statistics_bytes_sent_total.Collect(ch,
					float64(clientStatData[idx]),
					[]string{
						client,
					})
			} else if columnName == "BINLOG_BYTES_WRITTEN" {
				qd.infoSchemaclient_statistics_binlog_bytes_written_total.Collect(ch,
					float64(clientStatData[idx]),
					[]string{
						client,
					})
			} else if columnName == "ROWS_READ" {
				qd.infoSchemaclient_statistics_rows_read_total.Collect(ch,
					float64(clientStatData[idx]),
					[]string{
						client,
					})
			} else if columnName == "ROWS_SENT" {
				qd.infoSchemaclient_statistics_rows_sent_total.Collect(ch,
					float64(clientStatData[idx]),
					[]string{
						client,
					})
			} else if columnName == "ROWS_DELETED" {
				qd.infoSchemaclient_statistics_rows_deleted_total.Collect(ch,
					float64(clientStatData[idx]),
					[]string{
						client,
					})
			} else if columnName == "ROWS_INSERTED" {
				qd.infoSchemaclient_statistics_rows_inserted_total.Collect(ch,
					float64(clientStatData[idx]),
					[]string{
						client,
					})
			} else if columnName == "ROWS_FETCHED" {
				qd.infoSchemaclient_statistics_rows_fetched_total.Collect(ch,
					float64(clientStatData[idx]),
					[]string{
						client,
					})
			} else if columnName == "ROWS_UPDATED" {
				qd.infoSchemaclient_statistics_rows_updated_total.Collect(ch,
					float64(clientStatData[idx]),
					[]string{
						client,
					})
			} else if columnName == "TABLE_ROWS_READ" {
				qd.infoSchemaclient_statistics_table_rows_read_total.Collect(ch,
					float64(clientStatData[idx]),
					[]string{
						client,
					})
			} else if columnName == "SELECT_COMMANDS" {
				qd.infoSchemaclient_statistics_select_commands_total.Collect(ch,
					float64(clientStatData[idx]),
					[]string{
						client,
					})
			} else if columnName == "UPDATE_COMMANDS" {
				qd.infoSchemaclient_statistics_update_commands_total.Collect(ch,
					float64(clientStatData[idx]),
					[]string{
						client,
					})
			} else if columnName == "OTHER_COMMANDS" {
				qd.infoSchemaclient_statistics_other_commands_total.Collect(ch,
					float64(clientStatData[idx]),
					[]string{
						client,
					})
			} else if columnName == "COMMIT_TRANSACTIONS" {
				qd.infoSchemaclient_statistics_commit_transactions_total.Collect(ch,
					float64(clientStatData[idx]),
					[]string{
						client,
					})
			} else if columnName == "ROLLBACK_TRANSACTIONS" {
				qd.infoSchemaclient_statistics_rollback_transactions_total.Collect(ch,
					float64(clientStatData[idx]),
					[]string{
						client,
					})
			} else if columnName == "DENIED_CONNECTIONS" {
				qd.infoSchemaclient_statistics_denied_connections_total.Collect(ch,
					float64(clientStatData[idx]),
					[]string{
						client,
					})
			} else if columnName == "LOST_CONNECTIONS" {
				qd.infoSchemaclient_statistics_lost_connections_total.Collect(ch,
					float64(clientStatData[idx]),
					[]string{
						client,
					})
			} else if columnName == "ACCESS_DENIED" {
				qd.infoSchemaclient_statistics_access_denied_total.Collect(ch,
					float64(clientStatData[idx]),
					[]string{
						client,
					})
			} else if columnName == "EMPTY_QUERIES" {
				qd.infoSchemaclient_statistics_empty_queries_total.Collect(ch,
					float64(clientStatData[idx]),
					[]string{
						client,
					})
			} else if columnName == "TOTAL_SSL_CONNECTIONS" {
				qd.infoSchemaclient_statistics_total_ssl_connections_total.Collect(ch,
					float64(clientStatData[idx]),
					[]string{
						client,
					})
			} else if columnName == "MAX_STATEMENT_TIME_EXCEEDED" {
				qd.infoSchemaclient_statistics_max_statement_time_exceeded_total.Collect(ch,
					float64(clientStatData[idx]),
					[]string{
						client,
					})
			}

		}
	}
}

type infoSchemaclient_statistics_total_connections struct {
	*baseMetrics
}

func NewinfoSchemaclient_statistics_total_connections() *infoSchemaclient_statistics_total_connections {
	return &infoSchemaclient_statistics_total_connections{
		NewMetrics(
			"info_schema_client_statistics_total_connections",
			"The number of connections created for this client.",
			[]string{
				"client",
			})}
}
func (qd *infoSchemaclient_statistics_total_connections) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaclient_statistics_concurrent_connections struct {
	*baseMetrics
}

func NewinfoSchemaclient_statistics_concurrent_connections() *infoSchemaclient_statistics_concurrent_connections {
	return &infoSchemaclient_statistics_concurrent_connections{
		NewMetrics(
			"info_schema_client_statistics_concurrent_connections",
			"The number of concurrent connections for this client.",
			[]string{
				"client",
			})}
}
func (qd *infoSchemaclient_statistics_concurrent_connections) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaclient_statistics_connected_time_seconds_total struct {
	*baseMetrics
}

func NewinfoSchemaclient_statistics_connected_time_seconds_total() *infoSchemaclient_statistics_connected_time_seconds_total {
	return &infoSchemaclient_statistics_connected_time_seconds_total{
		NewMetrics(
			"info_schema_client_statistics_connected_time_seconds_total",
			"The total number of seconds that the client has been connected.",
			[]string{
				"client",
			})}
}
func (qd *infoSchemaclient_statistics_connected_time_seconds_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaclient_statistics_busy_time_seconds_total struct {
	*baseMetrics
}

func NewinfoSchemaclient_statistics_busy_time_seconds_total() *infoSchemaclient_statistics_busy_time_seconds_total {
	return &infoSchemaclient_statistics_busy_time_seconds_total{
		NewMetrics(
			"info_schema_client_statistics_busy_time_seconds_total",
			"The total number of seconds that the client has spent executing queries.",
			[]string{
				"client",
			})}
}
func (qd *infoSchemaclient_statistics_busy_time_seconds_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaclient_statistics_cpu_time_seconds_total struct {
	*baseMetrics
}

func NewinfoSchemaclient_statistics_cpu_time_seconds_total() *infoSchemaclient_statistics_cpu_time_seconds_total {
	return &infoSchemaclient_statistics_cpu_time_seconds_total{
		NewMetrics(
			"info_schema_client_statistics_cpu_time_seconds_total",
			"The total number of seconds that the client has spent executing queries.",
			[]string{
				"client",
			})}
}
func (qd *infoSchemaclient_statistics_cpu_time_seconds_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaclient_statistics_bytes_received_total struct {
	*baseMetrics
}

func NewinfoSchemaclient_statistics_bytes_received_total() *infoSchemaclient_statistics_bytes_received_total {
	return &infoSchemaclient_statistics_bytes_received_total{
		NewMetrics(
			"info_schema_client_statistics_bytes_received_total",
			"The total number of bytes received by this client.",
			[]string{
				"client",
			})}
}
func (qd *infoSchemaclient_statistics_bytes_received_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaclient_statistics_bytes_sent_total struct {
	*baseMetrics
}

func NewinfoSchemaclient_statistics_bytes_sent_total() *infoSchemaclient_statistics_bytes_sent_total {
	return &infoSchemaclient_statistics_bytes_sent_total{
		NewMetrics(
			"info_schema_client_statistics_bytes_sent_total",
			"The total number of bytes sent by this client.",
			[]string{
				"client",
			})}
}
func (qd *infoSchemaclient_statistics_bytes_sent_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaclient_statistics_binlog_bytes_written_total struct {
	*baseMetrics
}

func NewinfoSchemaclient_statistics_binlog_bytes_written_total() *infoSchemaclient_statistics_binlog_bytes_written_total {
	return &infoSchemaclient_statistics_binlog_bytes_written_total{
		NewMetrics(
			"info_schema_client_statistics_binlog_bytes_written_total",
			"The total number of bytes written to the binary log by this client.",
			[]string{
				"client",
			})}
}
func (qd *infoSchemaclient_statistics_binlog_bytes_written_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaclient_statistics_rows_read_total struct {
	*baseMetrics
}

func NewinfoSchemaclient_statistics_rows_read_total() *infoSchemaclient_statistics_rows_read_total {
	return &infoSchemaclient_statistics_rows_read_total{
		NewMetrics(
			"info_schema_client_statistics_rows_read_total",
			"The total number of rows read by this client.",
			[]string{
				"client",
			})}
}
func (qd *infoSchemaclient_statistics_rows_read_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaclient_statistics_rows_sent_total struct {
	*baseMetrics
}

func NewinfoSchemaclient_statistics_rows_sent_total() *infoSchemaclient_statistics_rows_sent_total {
	return &infoSchemaclient_statistics_rows_sent_total{
		NewMetrics(
			"info_schema_client_statistics_rows_sent_total",
			"The total number of rows sent by this client.",
			[]string{
				"client",
			})}
}
func (qd *infoSchemaclient_statistics_rows_sent_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaclient_statistics_rows_deleted_total struct {
	*baseMetrics
}

func NewinfoSchemaclient_statistics_rows_deleted_total() *infoSchemaclient_statistics_rows_deleted_total {
	return &infoSchemaclient_statistics_rows_deleted_total{
		NewMetrics(
			"info_schema_client_statistics_rows_deleted_total",
			"The total number of rows deleted by this client.",
			[]string{
				"client",
			})}
}
func (qd *infoSchemaclient_statistics_rows_deleted_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaclient_statistics_rows_inserted_total struct {
	*baseMetrics
}

func NewinfoSchemaclient_statistics_rows_inserted_total() *infoSchemaclient_statistics_rows_inserted_total {
	return &infoSchemaclient_statistics_rows_inserted_total{
		NewMetrics(
			"info_schema_client_statistics_rows_inserted_total",
			"The total number of rows inserted by this client.",
			[]string{
				"client",
			})}
}
func (qd *infoSchemaclient_statistics_rows_inserted_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaclient_statistics_rows_fetched_total struct {
	*baseMetrics
}

func NewinfoSchemaclient_statistics_rows_fetched_total() *infoSchemaclient_statistics_rows_fetched_total {
	return &infoSchemaclient_statistics_rows_fetched_total{
		NewMetrics(
			"info_schema_client_statistics_rows_fetched_total",
			"The total number of rows fetched by this client.",
			[]string{
				"client",
			})}
}
func (qd *infoSchemaclient_statistics_rows_fetched_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaclient_statistics_rows_updated_total struct {
	*baseMetrics
}

func NewinfoSchemaclient_statistics_rows_updated_total() *infoSchemaclient_statistics_rows_updated_total {
	return &infoSchemaclient_statistics_rows_updated_total{
		NewMetrics(
			"info_schema_client_statistics_rows_updated_total",
			"The total number of rows updated by this client.",
			[]string{
				"client",
			})}
}
func (qd *infoSchemaclient_statistics_rows_updated_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaclient_statistics_table_rows_read_total struct {
	*baseMetrics
}

func NewinfoSchemaclient_statistics_table_rows_read_total() *infoSchemaclient_statistics_table_rows_read_total {
	return &infoSchemaclient_statistics_table_rows_read_total{
		NewMetrics(
			"info_schema_client_statistics_table_rows_read_total",
			"The total number of rows read by this client from tables.",
			[]string{
				"client",
			})}
}
func (qd *infoSchemaclient_statistics_table_rows_read_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaclient_statistics_select_commands_total struct {
	*baseMetrics
}

func NewinfoSchemaclient_statistics_select_commands_total() *infoSchemaclient_statistics_select_commands_total {
	return &infoSchemaclient_statistics_select_commands_total{
		NewMetrics(
			"info_schema_client_statistics_select_commands_total",
			"The total number of select commands executed by this client.",
			[]string{
				"client",
			})}
}
func (qd *infoSchemaclient_statistics_select_commands_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaclient_statistics_update_commands_total struct {
	*baseMetrics
}

func NewinfoSchemaclient_statistics_update_commands_total() *infoSchemaclient_statistics_update_commands_total {
	return &infoSchemaclient_statistics_update_commands_total{
		NewMetrics(
			"info_schema_client_statistics_update_commands_total",
			"The number of UPDATE commands executed from this client’s connections.",
			[]string{
				"client",
			})}
}
func (qd *infoSchemaclient_statistics_update_commands_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaclient_statistics_other_commands_total struct {
	*baseMetrics
}

func NewinfoSchemaclient_statistics_other_commands_total() *infoSchemaclient_statistics_other_commands_total {
	return &infoSchemaclient_statistics_other_commands_total{
		NewMetrics(
			"info_schema_client_statistics_other_commands_total",
			"The number of other commands executed from this client’s connections.",
			[]string{
				"client",
			})}
}
func (qd *infoSchemaclient_statistics_other_commands_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaclient_statistics_commit_transactions_total struct {
	*baseMetrics
}

func NewinfoSchemaclient_statistics_commit_transactions_total() *infoSchemaclient_statistics_commit_transactions_total {
	return &infoSchemaclient_statistics_commit_transactions_total{
		NewMetrics(
			"info_schema_client_statistics_commit_transactions_total",
			"The number of COMMIT commands issued by this client’s connections.",
			[]string{
				"client",
			})}
}
func (qd *infoSchemaclient_statistics_commit_transactions_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaclient_statistics_rollback_transactions_total struct {
	*baseMetrics
}

func NewinfoSchemainfoSchemaclient_statistics_rollback_transactions_total() *infoSchemaclient_statistics_rollback_transactions_total {
	return &infoSchemaclient_statistics_rollback_transactions_total{
		NewMetrics(
			"info_schema_client_statistics_rollback_transactions_total",
			"The number of COMMIT commands issued by this client’s connections.",
			[]string{
				"client",
			})}
}
func (qd *infoSchemaclient_statistics_rollback_transactions_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaclient_statistics_denied_connections_total struct {
	*baseMetrics
}

func NewinfoSchemaclient_statistics_denied_connections_total() *infoSchemaclient_statistics_denied_connections_total {
	return &infoSchemaclient_statistics_denied_connections_total{
		NewMetrics(
			"info_schema_client_statistics_denied_connections_total",
			"The number of connections denied to this client.",
			[]string{
				"client",
			})}
}
func (qd *infoSchemaclient_statistics_denied_connections_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaclient_statistics_lost_connections_total struct {
	*baseMetrics
}

func NewinfoSchemaclient_statistics_lost_connections_total() *infoSchemaclient_statistics_lost_connections_total {
	return &infoSchemaclient_statistics_lost_connections_total{
		NewMetrics(
			"info_schema_client_statistics_lost_connections_total",
			"The number of this client’s connections that were terminated uncleanly.",
			[]string{
				"client",
			})}
}
func (qd *infoSchemaclient_statistics_lost_connections_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaclient_statistics_access_denied_total struct {
	*baseMetrics
}

func NewinfoSchemaclient_statistics_access_denied_total() *infoSchemaclient_statistics_access_denied_total {
	return &infoSchemaclient_statistics_access_denied_total{
		NewMetrics(
			"info_schema_client_statistics_access_denied_total",
			"The number of this client’s connections that were terminated uncleanly.",
			[]string{
				"client",
			})}
}
func (qd *infoSchemaclient_statistics_access_denied_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaclient_statistics_empty_queries_total struct {
	*baseMetrics
}

func NewinfoSchemaclient_statistics_empty_queries_total() *infoSchemaclient_statistics_empty_queries_total {
	return &infoSchemaclient_statistics_empty_queries_total{
		NewMetrics(
			"info_schema_client_statistics_empty_queries_total",
			"The number of times this client’s connections sent empty queries to the server.",
			[]string{
				"client",
			})}
}
func (qd *infoSchemaclient_statistics_empty_queries_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaclient_statistics_total_ssl_connections_total struct {
	*baseMetrics
}

func NewinfoSchemaclient_statistics_total_ssl_connections_total() *infoSchemaclient_statistics_total_ssl_connections_total {
	return &infoSchemaclient_statistics_total_ssl_connections_total{
		NewMetrics(
			"info_schema_client_statistics_total_ssl_connections_total",
			"The number of times this client’s connections connected using SSL to the server.",
			[]string{
				"client",
			})}
}
func (qd *infoSchemaclient_statistics_total_ssl_connections_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaclient_statistics_max_statement_time_exceeded_total struct {
	*baseMetrics
}

func NewinfoSchemaclient_statistics_max_statement_time_exceeded_total() *infoSchemaclient_statistics_max_statement_time_exceeded_total {
	return &infoSchemaclient_statistics_max_statement_time_exceeded_total{
		NewMetrics(
			"info_schema_client_statistics_max_statement_time_exceeded_total",
			"The number of times a statement was aborted, because it was executed longer than its MAX_STATEMENT_TIME threshold.",
			[]string{
				"client",
			})}
}
func (qd *infoSchemaclient_statistics_max_statement_time_exceeded_total) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

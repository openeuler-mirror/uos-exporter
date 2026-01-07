package metrics

import (
	"database/sql"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"mysqld_exporter/internal/exporter"
	"mysqld_exporter/internal/mysql"
	"strings"
)

const (
	slaveStatus = "slave_status"
)

var slaveStatusQueries = [3]string{
	"SHOW ALL SLAVES STATUS",
	"SHOW SLAVE STATUS",
	"SHOW REPLICA STATUS"}
var slaveStatusQuerySuffixes = [3]string{
	" NONBLOCKING",
	" NOLOCK", ""}

func init() {
	exporter.Register(
		NewSlaveStatus())
}

type SlaveStatus struct {
	instance mysql.Instance
	*baseMetrics
}

func NewSlaveStatus() *SlaveStatus {
	return &SlaveStatus{
		//instance: instance,
		baseMetrics: NewMetrics(
			"mysql_slave_status",
			"Generic metric from SHOW SLAVE STATUS.",
			[]string{"master_host",
				"master_uuid",
				"channel_name",
				"connection_name",
				"slave_name"}),
	}
}

func (qd *SlaveStatus) Collect(ch chan<- prometheus.Metric) {
	qd.instance = *GetInstance()

	logrus.Info("Start collecting SlaveStatus metrics")
	var (
		slaveStatusRows *sql.Rows
		err             error
	)
	db := qd.instance.GetDB()
	for _, query := range slaveStatusQueries {
		slaveStatusRows, err = db.Query(query)
		if err != nil {
			for _, suffix := range slaveStatusQuerySuffixes {
				slaveStatusRows, err = db.Query(fmt.Sprint(query, suffix))
				if err == nil {
					break
				}
			}
		} else {
			break
		}
	}
	if err != nil {
		logrus.Error(err)
		return
	}
	defer slaveStatusRows.Close()

	slaveCols, err := slaveStatusRows.Columns()
	if err != nil {
		logrus.Error(err)
		return
	}
	for slaveStatusRows.Next() {
		scanArgs := make([]interface{}, len(slaveCols))
		for i := range scanArgs {
			scanArgs[i] = &sql.RawBytes{}
			if err := slaveStatusRows.Scan(scanArgs...); err != nil {
				logrus.Error(err)
				return
			}

			masterUUID := columnValue(
				scanArgs,
				slaveCols,
				"Master_UUID")
			if masterUUID == "" {
				masterUUID = columnValue(
					scanArgs,
					slaveCols,
					"Source_UUID")
			}
			masterHost := columnValue(
				scanArgs,
				slaveCols,
				"Master_Host")
			if masterHost == "" {
				masterHost = columnValue(
					scanArgs,
					slaveCols,
					"Source_Host")
			}
			channelName := columnValue(
				scanArgs,
				slaveCols,
				"Channel_Name")
			connectionName := columnValue(
				scanArgs,
				slaveCols,
				"Connection_name")
			for i, col := range slaveCols {
				if value, ok := parseStatus(
					*scanArgs[i].(*sql.RawBytes)); ok { // Silently skip unparsable values.
					qd.collect(
						ch,
						value,
						[]string{
							masterHost,
							masterUUID,
							channelName,
							connectionName,
							strings.ToLower(col)})

				}
			}
		}
	}

}

func columnIndex(slaveCols []string, colName string) int {
	for idx := range slaveCols {
		if slaveCols[idx] == colName {
			return idx
		}
	}
	return -1
}
func columnValue(scanArgs []interface{}, slaveCols []string, colName string) string {
	var columnIndex = columnIndex(slaveCols, colName)
	if columnIndex == -1 {
		return ""
	}
	return string(*scanArgs[columnIndex].(*sql.RawBytes))
}

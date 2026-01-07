package metrics

import (
	"database/sql"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"mysqld_exporter/internal/exporter"
	"mysqld_exporter/internal/mysql"
)

const (
	slavehosts        = "slave_hosts"
	slaveHostsQuery   = "SHOW SLAVE HOSTS"
	showReplicasQuery = "SHOW REPLICAS"
)

func init() {
	exporter.Register(
		NewSlaveHostsInfo())
}

type SlaveHostsInfo struct {
	instance mysql.Instance
	*baseMetrics
}

func NewSlaveHostsInfo() *SlaveHostsInfo {
	return &SlaveHostsInfo{
		//instance: instance,
		baseMetrics: NewMetrics(
			"mysql_slave_hosts_info",
			"Information about running slaves.",
			[]string{
				"server_id",
				"slave_host",
				"port",
				"master_id",
				"slave_uuid"}),
	}
}

func (s SlaveHostsInfo) Collect(ch chan<- prometheus.Metric) {
	var (
		slaveHostsRows *sql.Rows
		err            error
	)
	s.instance = *GetInstance()

	db := s.instance.GetDB()
	for _, query := range []string{slaveHostsQuery, showReplicasQuery} {
		slaveHostsRows, err = db.Query(query)
		if err == nil {
			break
		}
	}
	if err != nil {
		logrus.Warnf("Error collecting slave hosts info: %v", err)
		return
	}
	defer slaveHostsRows.Close()
	var serverId string
	var host string
	var port string
	var rrrOrMasterId string
	var slaveUuidOrMasterId string
	var masterId string
	var slaveUuid string

	columnNames, err := slaveHostsRows.Columns()
	if err != nil {
		logrus.Warnf("Error collecting slave hosts info: %v", err)
		return
	}
	for slaveHostsRows.Next() {
		if len(columnNames) == 5 {
			err = slaveHostsRows.Scan(&serverId, &host, &port, &rrrOrMasterId, &slaveUuidOrMasterId)
		} else {
			err = slaveHostsRows.Scan(&serverId, &host, &port, &rrrOrMasterId)
		}
		if err != nil {
			logrus.Warnf("Error collecting slave hosts info: %v", err)
			return
		}
		if len(columnNames) == 5 {
			if _, err = uuid.Parse(slaveUuidOrMasterId); err != nil {
				slaveUuid = ""
				masterId = slaveUuidOrMasterId
			} else {
				slaveUuid = slaveUuidOrMasterId
				masterId = rrrOrMasterId
			}
		} else {
			slaveUuid = ""
			masterId = rrrOrMasterId
		}
		s.collect(ch,
			1,
			[]string{
				serverId,
				host,
				port,
				masterId,
				slaveUuid})
	}

}

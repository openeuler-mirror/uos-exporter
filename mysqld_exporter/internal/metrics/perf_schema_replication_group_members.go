package metrics

import (
	"database/sql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"mysqld_exporter/internal/exporter"
	"mysqld_exporter/internal/mysql"
	"strings"
)

const (
	perfReplicationGroupMembersQuery = `
  SELECT * FROM performance_schema.replication_group_members
	`
)

func init() {
	exporter.Register(
		NewScrapePerfReplicationGroupMembers())
}

type ScrapePerfReplicationGroupMembers struct {
	instance mysql.Instance
	//*baseMetrics
}

func NewScrapePerfReplicationGroupMembers() *ScrapePerfReplicationGroupMembers {
	return &ScrapePerfReplicationGroupMembers{
		//instance: instance,
	}
}

func (s ScrapePerfReplicationGroupMembers) Collect(ch chan<- prometheus.Metric) {
	s.instance = *GetInstance()

	db := s.instance.GetDB()
	perfReplicationGroupMembersRows, err := db.Query(perfReplicationGroupMembersQuery)
	if err != nil {
		logrus.Error(err)
		return
	}
	defer perfReplicationGroupMembersRows.Close()
	var columnNames []string
	columnNames, err = perfReplicationGroupMembersRows.Columns()
	if err != nil {
		logrus.Error(err)
		return
	}
	var scanArgs = make([]interface{}, len(columnNames))
	for i := range scanArgs {
		scanArgs[i] = &sql.RawBytes{}
	}
	for perfReplicationGroupMembersRows.Next() {
		err := perfReplicationGroupMembersRows.Scan(scanArgs...)
		if err != nil {
			logrus.Error(err)
			return
		}
		var labelNames = make([]string, len(columnNames))
		var values = make([]string, len(columnNames))
		for i, columnName := range columnNames {
			labelNames[i] = strings.ToLower(columnName)
			values[i] = string(*scanArgs[i].(*sql.RawBytes))
		}
		var performanceSchemaReplicationGroupMembersMemberDesc = prometheus.NewDesc(
			prometheus.BuildFQName("mysql", "perf_schema", "replication_group_member_info"),
			"Information about the replication group member: "+
				"channel_name, member_id, member_host, member_port, member_state. "+
				"(member_role and member_version where available)",
			labelNames, nil,
		)

		ch <- prometheus.MustNewConstMetric(
			performanceSchemaReplicationGroupMembersMemberDesc,
			prometheus.GaugeValue,
			1, values...)
	}

}
// Part 2 commit for mysqld_exporter/internal/metrics/perf_schema_replication_group_members.go

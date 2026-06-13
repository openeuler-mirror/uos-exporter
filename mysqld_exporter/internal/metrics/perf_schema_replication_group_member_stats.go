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


// TODO: implement functions

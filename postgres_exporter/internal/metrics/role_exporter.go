package metrics

import (
	"database/sql"

	"postgres_exporter/internal/collector"

	"github.com/prometheus/client_golang/prometheus"
)

type PostgreSQLRoleExporter struct {
	metrics *roleMetrics
	db      *sql.DB
	uri     string
	dbName  string
}

type roleMetrics struct {
	countMetric               *baseMetrics
	superuserMetric           *baseMetrics
	createDBMetric            *baseMetrics
	createRoleMetric          *baseMetrics
	inheritMetric             *baseMetrics
	replicationMetric         *baseMetrics
	canLoginMetric            *baseMetrics
	connectionLimitMetric     *baseMetrics
	validUntilTimestampMetric *baseMetrics
	bypassRLSMetric           *baseMetrics
	isDefaultMetric           *baseMetrics
}

func newRoleMetrics() *roleMetrics {
	return &roleMetrics{
		countMetric: NewMetrics(
			"pg_role_count",
			"The total number of roles in the database.",
			[]string{"instance", "uri", "db"},
		),
		superuserMetric: NewMetrics(
			"pg_role_superuser",
			"Gauge indicating if this role is a superuser (1) or not (0).",
			[]string{"instance", "uri", "db", "role"},
		),
		createDBMetric: NewMetrics(
			"pg_role_createdb",
			"Gauge indicating if this role can create databases (1) or not (0).",
			[]string{"instance", "uri", "db", "role"},
		),
		createRoleMetric: NewMetrics(
			"pg_role_createrole",
			"Gauge indicating if this role can create roles (1) or not (0).",
			[]string{"instance", "uri", "db", "role"},
		),
		inheritMetric: NewMetrics(
			"pg_role_inherit",
			"Gauge indicating if this role inherits permissions (1) or not (0).",
			[]string{"instance", "uri", "db", "role"},
		),
		replicationMetric: NewMetrics(
			"pg_role_replication",
			"Gauge indicating if this role has replication privileges.",
			[]string{"instance", "uri", "db", "role"},
		),
		canLoginMetric: NewMetrics(
			"pg_role_can_login",
			"Gauge indicating if this role can login.",
			[]string{"instance", "uri", "db", "role"},
		),
		connectionLimitMetric: NewMetrics(
			"pg_role_connection_limit",
			"The maximum allowed connections for this role.",
			[]string{"instance", "uri", "db", "role"},
		),
		validUntilTimestampMetric: NewMetrics(
			"pg_role_valid_until_timestamp",
			"The timestamp when this role's password expires.",
			[]string{"instance", "uri", "db", "role"},
		),
		bypassRLSMetric: NewMetrics(
			"pg_role_bypass_rls",
			"Gauge indicating if this role can bypass row-level security.",
			[]string{"instance", "uri", "db", "role"},
		),
		isDefaultMetric: NewMetrics(
			"pg_role_is_default",
			"Gauge indicating if this is a default system role.",
			[]string{"instance", "uri", "db", "role"},
		),
	}
}

// NewPostgreSQLRoleExporter 创建一个新的 Role Exporter
func NewPostgreSQLRoleExporter(db *sql.DB, uri, dbName string) *PostgreSQLRoleExporter {
	return &PostgreSQLRoleExporter{
		metrics: newRoleMetrics(),
		db:      db,
		uri:     uri,
		dbName:  dbName,
	}
}

// Describe implements Prometheus Collector interface

// Collect implements Prometheus Collector interface
func (e *PostgreSQLRoleExporter) collect(ch chan<- prometheus.Metric) {
	roleStats, err := collector.ScrapePostgreSQLRoles(e.db)
	if err != nil || len(roleStats.Roles) == 0 {
		return
	}

	labelsBase := []string{"", e.uri, e.dbName}

	// 上报总角色数
	e.metrics.countMetric.collect(
		ch,
		float64(len(roleStats.Roles)),
		labelsBase,
	)

	// 遍历每个角色上报指标
	for _, role := range roleStats.Roles {
		labels := append(labelsBase, role.Name)

		e.metrics.superuserMetric.collect(
			ch,
			boolToFloat64(role.Superuser),
			labels,
		)
		e.metrics.createDBMetric.collect(
			ch,
			boolToFloat64(role.CreateDB),
			labels,
		)
		e.metrics.createRoleMetric.collect(
			ch,
			boolToFloat64(role.CreateUser),
			labels,
		)
		e.metrics.inheritMetric.collect(
			ch,
			boolToFloat64(role.Inherit),
			labels,
		)
		e.metrics.replicationMetric.collect(
			ch,
			boolToFloat64(role.Replication),
			labels,
		)
		e.metrics.canLoginMetric.collect(
			ch,
			boolToFloat64(role.CanLogin),
			labels,
		)
		e.metrics.connectionLimitMetric.collect(
			ch,
			float64(role.ConnectionLimit),
			labels,
		)
		e.metrics.isDefaultMetric.collect(
			ch,
			boolToFloat64(role.IsDefault),
			labels,
		)

		// 上报密码有效期（如果设置了）
		if role.ValidUntil != nil && !role.ValidUntil.IsZero() {
			e.metrics.validUntilTimestampMetric.collect(
				ch,
				float64(role.ValidUntil.Unix()),
				labels,
			)
		}
	}
}

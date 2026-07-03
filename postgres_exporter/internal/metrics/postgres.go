package metrics

import (
	"database/sql"

	"postgres_exporter/internal/exporter"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

func init() {
	exporter.Register(NewPostgreSQLExporter())
}

type PostgreSQLExporter struct {
	db                            *sql.DB
	PostgreSQLInfoExporter        *PostgreSQLInfoExporter
	PostgreSQLTableExporter       *PostgreSQLTableExporter
	PostgreSQLIndexExporter       *PostgreSQLIndexExporter
	PostgreSQLBloatExporter       *PostgreSQLBloatExporter
	PostgreSQLLockExporter        *PostgreSQLLockExporter
	PostgreSQLBGWriterExporter    *PostgreSQLBGWriterExporter
	PostgreSQLReplicationExporter *PostgreSQLReplicationExporter
	PostgreSQLExtensionExporter   *PostgreSQLExtensionExporter
	PostgreSQLRoleExporter        *PostgreSQLRoleExporter
	PostgreSQLSchemaExporter      *PostgreSQLSchemaExporter
}

func NewPostgreSQLExporter() *PostgreSQLExporter {
	cfg, err := exporter.LoadConfig()
	if err != nil {
		logrus.Errorf("Failed to load config: %v", err)
		return nil
	}
	instance := cfg.Instances[0] // 假设只处理第一个实例
	uri := instance.Connection.BuildDSN()
	lable := instance.Connection.BuildLable()

	db, err := sql.Open("pgx", uri)
	if err != nil {
		logrus.Errorf("Failed to open database: %v", err)
		return nil
	}
	p := &PostgreSQLExporter{
		db:                            db,
		PostgreSQLInfoExporter:        NewPostgreSQLInfoExporter(db, instance.Name, lable),
		PostgreSQLTableExporter:       NewPostgreSQLTableExporter(db, instance.Name, lable),
		PostgreSQLIndexExporter:       NewPostgreSQLIndexExporter(db, instance.Name, lable),
		PostgreSQLBloatExporter:       NewPostgreSQLBloatExporter(db, instance.Name, lable),
		PostgreSQLLockExporter:        NewPostgreSQLLockExporter(db, instance.Name, lable),
		PostgreSQLBGWriterExporter:    NewPostgreSQLBGWriterExporter(db, instance.Name, lable),
		PostgreSQLReplicationExporter: NewPostgreSQLReplicationExporter(db, instance.Name, lable),
		PostgreSQLExtensionExporter:   NewPostgreSQLExtensionExporter(db, instance.Name, lable),
		PostgreSQLRoleExporter:        NewPostgreSQLRoleExporter(db, instance.Name, lable),
		PostgreSQLSchemaExporter:      NewPostgreSQLSchemaExporter(db, instance.Name, lable),
	}

	return p
}

func (p *PostgreSQLExporter) Collect(ch chan<- prometheus.Metric) {
	p.PostgreSQLInfoExporter.collect(ch)
	p.PostgreSQLTableExporter.collect(ch)
	p.PostgreSQLIndexExporter.collect(ch)
	p.PostgreSQLBloatExporter.collect(ch)
	p.PostgreSQLLockExporter.collect(ch)
	p.PostgreSQLBGWriterExporter.collect(ch)
	p.PostgreSQLReplicationExporter.collect(ch)
	p.PostgreSQLExtensionExporter.collect(ch)
	p.PostgreSQLRoleExporter.collect(ch)
	p.PostgreSQLSchemaExporter.collect(ch)
}

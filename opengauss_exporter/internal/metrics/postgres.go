package metrics

import (
	"database/sql"
	"time"

	"opengauss_exporter/internal/exporter"

	// _ "github.com/jackc/pgx/v4/stdlib"
	// "gitee.com/opengauss/openGauss-connector-go-pq"
	_ "gitee.com/opengauss/openGauss-connector-go-pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

func init() {
	exporter.Register(NewOpenGaussExporter())
}

type OpenGaussExporter struct {
	db                        *sql.DB
	OpenGaussInfoExporter     *OpenGaussInfoExporter
	PgStatActivityExporter    *PgStatActivityExporter
	PgStatBgwriterExporter    *PgStatBgwriterExporter
	PgStatDatabaseExporter    *PgStatDatabaseExporter
	PgStatUserTablesExporter  *PgStatUserTablesExporter
	PgStatUserIndexesExporter *PgStatUserIndexesExporter
	PgStatReplicationExporter *PgStatReplicationExporter
	PgLocksExporter           *PgLocksExporter
	PgSizeExporter            *PgSizeExporter
}

func NewOpenGaussExporter() *OpenGaussExporter {
	logrus.Debug("Creating new OpenGaussExporter")
	cfg, err := exporter.LoadConfig()
	if err != nil {
		logrus.Errorf("Failed to load config: %v", err)
		return nil
	}
	instance := cfg.Instances[0] // 假设只处理第一个实例
	uri := instance.Connection.BuildDSN()
	lable := instance.Connection.BuildLable()

	logrus.Debugf("Connecting to OpenGauss instance: %s", instance.Name)
	db, err := sql.Open("opengauss", uri)
	if err != nil {
		logrus.Errorf("Failed to open database: %v", err)
		return nil
	}

	// 测试数据库连接
	if err := db.Ping(); err != nil {
		logrus.Errorf("Failed to ping database: %v", err)
		return nil
	}

	logrus.Debug("Successfully connected to database, initializing exporters")
	p := &OpenGaussExporter{
		db:                        db,
		OpenGaussInfoExporter:     NewOpenGaussInfoExporter(db, instance.Name, lable),
		PgStatActivityExporter:    NewPgStatActivityExporter(db, instance.Name, lable),
		PgStatBgwriterExporter:    NewPgStatBgwriterExporter(db, instance.Name, lable),
		PgStatDatabaseExporter:    NewPgStatDatabaseExporter(db, instance.Name, lable),
		PgStatUserTablesExporter:  NewPgStatUserTablesExporter(db, instance.Name, lable),
		PgStatUserIndexesExporter: NewPgStatUserIndexesExporter(db, instance.Name, lable),
		PgStatReplicationExporter: NewPgStatReplicationExporter(db, instance.Name, lable),
		PgLocksExporter:           NewPgLocksExporter(db, instance.Name, lable),
		PgSizeExporter:            NewPgSizeExporter(db, instance.Name, lable),
	}

	logrus.Debug("OpenGaussExporter initialized successfully")
	return p
}

func (p *OpenGaussExporter) Collect(ch chan<- prometheus.Metric) {
	startTime := time.Now()
	logrus.Debug("Starting OpenGauss metrics collection")

	// 收集info指标
	logrus.Debug("Collecting OpenGauss info metrics")
	infoStart := time.Now()
	p.OpenGaussInfoExporter.collect(ch)
	logrus.Debugf("Info metrics collection completed in %v", time.Since(infoStart))

	// 收集activity指标
	logrus.Debug("Collecting pg_stat_activity metrics")
	activityStart := time.Now()
	p.PgStatActivityExporter.Collect(ch)
	logrus.Debugf("Activity metrics collection completed in %v", time.Since(activityStart))

	// 收集bgwriter指标
	logrus.Debug("Collecting pg_stat_bgwriter metrics")
	bgwriterStart := time.Now()
	p.PgStatBgwriterExporter.Collect(ch)
	logrus.Debugf("Bgwriter metrics collection completed in %v", time.Since(bgwriterStart))

	// 收集database指标
	logrus.Debug("Collecting pg_stat_database metrics")
	databaseStart := time.Now()
	p.PgStatDatabaseExporter.Collect(ch)
	logrus.Debugf("Database metrics collection completed in %v", time.Since(databaseStart))

	// 收集user tables指标
	logrus.Debug("Collecting pg_stat_user_tables metrics")
	userTablesStart := time.Now()
	p.PgStatUserTablesExporter.Collect(ch)
	logrus.Debugf("User tables metrics collection completed in %v", time.Since(userTablesStart))

	// 收集user indexes指标
	logrus.Debug("Collecting pg_stat_user_indexes metrics")
	userIndexesStart := time.Now()
	p.PgStatUserIndexesExporter.Collect(ch)
	logrus.Debugf("User indexes metrics collection completed in %v", time.Since(userIndexesStart))

	// 收集replication指标
	logrus.Debug("Collecting pg_stat_replication metrics")
	replicationStart := time.Now()
	p.PgStatReplicationExporter.Collect(ch)
	logrus.Debugf("Replication metrics collection completed in %v", time.Since(replicationStart))

	// 收集locks指标
	logrus.Debug("Collecting pg_locks metrics")
	locksStart := time.Now()
	p.PgLocksExporter.Collect(ch)
	logrus.Debugf("Locks metrics collection completed in %v", time.Since(locksStart))

	// 收集size指标
	logrus.Debug("Collecting size metrics")
	sizeStart := time.Now()
	p.PgSizeExporter.Collect(ch)
	logrus.Debugf("Size metrics collection completed in %v", time.Since(sizeStart))

	totalDuration := time.Since(startTime)
	logrus.Debugf("OpenGauss metrics collection completed successfully in %v", totalDuration)

	// 如果收集时间过长，记录警告
	if totalDuration > 30*time.Second {
		logrus.Warnf("Metrics collection took longer than expected: %v", totalDuration)
	}
}

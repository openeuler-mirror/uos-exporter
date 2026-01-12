package collector

import (
	"context"
	"database/sql"
	"time"

	"opengauss_exporter/internal/model"

	"github.com/sirupsen/logrus"
)

// ScrapePgStatReplication 采集 pg_stat_replication 视图的复制统计数据
func ScrapePgStatReplication(db *sql.DB) (*model.PgStatReplicationCollection, error) {
	logrus.Debug("Starting pg_stat_replication collection")
	ctx := context.Background()
	collection := model.NewPgStatReplicationCollection()

	// 检查表是否存在
	checkTableQuery := `
		SELECT COUNT(*)
		FROM information_schema.tables 
		WHERE table_name = 'pg_stat_replication'`

	var tableExists int
	err := db.QueryRowContext(ctx, checkTableQuery).Scan(&tableExists)
	if err != nil || tableExists == 0 {
		logrus.Debug("pg_stat_replication table does not exist, returning empty collection")
		return collection, nil
	}

	logrus.Debug("Executing pg_stat_replication query")
	query := `
		SELECT 
			COALESCE(application_name, 'unknown') as application_name,
			COALESCE(client_addr::text, 'unknown') as client_addr,
			COALESCE(client_hostname, 'unknown') as client_hostname,
			COALESCE(state, 'unknown') as state,
			COALESCE(EXTRACT(EPOCH FROM write_lag), 0) as write_lag,
			COALESCE(EXTRACT(EPOCH FROM flush_lag), 0) as flush_lag,
			COALESCE(EXTRACT(EPOCH FROM replay_lag), 0) as replay_lag,
			COALESCE(sent_lsn::text, '') as sent_lsn,
			COALESCE(write_lsn::text, '') as write_lsn,
			COALESCE(flush_lsn::text, '') as flush_lsn,
			COALESCE(replay_lsn::text, '') as replay_lsn,
			backend_start
		FROM pg_stat_replication`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		logrus.Debugf("Failed to execute pg_stat_replication query: %v", err)
		return collection, nil
	}
	defer rows.Close()

	var totalReplicas int64
	for rows.Next() {
		replication := model.NewPgStatReplication()
		var backendStartStr sql.NullString

		err := rows.Scan(
			&replication.ApplicationName,
			&replication.ClientAddr,
			&replication.ClientHostname,
			&replication.State,
			&replication.WriteLag,
			&replication.FlushLag,
			&replication.ReplayLag,
			&replication.SentLsn,
			&replication.WriteLsn,
			&replication.FlushLsn,
			&replication.ReplayLsn,
			&backendStartStr,
		)
		if err != nil {
			logrus.Debugf("Failed to scan replication statistics row: %v", err)
			continue
		}

		// 解析后端启动时间
		if backendStartStr.Valid && backendStartStr.String != "" {
			if startTime, parseErr := time.Parse(time.RFC3339, backendStartStr.String); parseErr == nil {
				replication.BackendStart = &startTime
			} else {
				logrus.Debugf("Failed to parse backend start time: %v", parseErr)
			}
		}

		collection.Replications[replication.GetReplicationKey()] = replication
		totalReplicas++

		logrus.Debugf("Replication connection: app=%s, client=%s, state=%s, write_lag=%.3fs, flush_lag=%.3fs, replay_lag=%.3fs",
			replication.ApplicationName, replication.ClientAddr, replication.State,
			replication.WriteLag, replication.FlushLag, replication.ReplayLag)
	}

	collection.TotalReplicas = totalReplicas
	logrus.Debugf("Successfully collected pg_stat_replication data for %d replica connections", totalReplicas)
	return collection, nil
}

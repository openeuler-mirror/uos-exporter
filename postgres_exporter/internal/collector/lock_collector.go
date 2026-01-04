package collector

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"postgres_exporter/internal/model"
)

// ScrapePostgreSQLLocks 采集所有当前正在使用的锁信息
func ScrapePostgreSQLLocks(db *sql.DB) (*model.PostgreSQLLockStats, error) {
	ctx := context.Background()

	stats := &model.PostgreSQLLockStats{
		Locks: []*model.PostgreSQLLock{},
	}

	rows, err := db.QueryContext(ctx, `
        SELECT 
            COALESCE(t.schemaname, 'unknown') AS schemaname,
            t.relname AS tablename,
            l.mode,
            l.pid,
            a.query,
            a.state,
            a.query_start,
            EXTRACT(EPOCH FROM (NOW() - a.query_start))::int AS duration_seconds
        FROM pg_locks l
        LEFT JOIN pg_stat_all_tables t ON l.relation = t.relid
        JOIN pg_stat_activity a ON a.pid = l.pid
        WHERE l.locktype = 'relation'
          AND l.database IS NOT NULL
          AND t.schemaname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
    `)
	if err != nil {
		return stats, fmt.Errorf("failed to query locks: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var schema sql.NullString
		var name sql.NullString
		var mode sql.NullString
		var pid sql.NullInt64
		var query sql.NullString
		var state sql.NullString
		var queryStart sql.NullTime
		var durationSeconds sql.NullInt64

		err := rows.Scan(&schema, &name, &mode, &pid, &query, &state, &queryStart, &durationSeconds)
		if err != nil {
			continue
		}

		stats.Locks = append(stats.Locks, &model.PostgreSQLLock{
			Database: "your_db", // Exporter 注入
			Schema:   coalesceNullString(schema),
			Table:    coalesceNullString(name),
			Mode:     coalesceNullString(mode),
			PID:      coalesceNullInt64(pid),
			Query:    coalesceNullString(query),
			State:    coalesceNullString(state),
			Duration: time.Duration(coalesceNullInt64(durationSeconds)) * time.Second,
			Waited:   isWaitingState(coalesceNullString(state)),
		})
	}

	return stats, nil
}

// 辅助函数：判断是否是等待状态
func isWaitingState(state string) bool {
	return state == "idle in transaction (waiting)" || state == "waiting"
}

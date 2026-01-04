package collector

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"postgres_exporter/internal/model"
)

// ScrapePostgreSQLInfo 采集 PostgreSQL 的基本信息并返回结构体
func ScrapePostgreSQLInfo(db *sql.DB) (*model.PostgreSQLInfo, error) {
	ctx := context.Background()

	info := &model.PostgreSQLInfo{}

	// 检查是否在线
	if err := db.PingContext(ctx); err != nil {
		return info, err
	}
	info.Up = true

	// 获取版本号
	var version string
	err := db.QueryRow("SELECT version();").Scan(&version)
	if err == nil {
		info.Version = parseVersion(version)
	}

	// 获取数据库数量
	var dbCount sql.NullInt64
	err = db.QueryRow("SELECT COUNT(*) FROM pg_database WHERE NOT datistemplate;").Scan(&dbCount)
	if err == nil && dbCount.Valid {
		info.DatabaseCount = dbCount.Int64
	}

	// 当前连接数
	var connCount sql.NullInt64
	err = db.QueryRow("SELECT COUNT(*) FROM pg_stat_activity;").Scan(&connCount)
	if err == nil && connCount.Valid {
		info.ConnectionCurrent = connCount.Int64
	}

	// 最大连接数限制
	var maxConn sql.NullInt64
	err = db.QueryRow("SHOW max_connections;").Scan(&maxConn)
	if err == nil && maxConn.Valid {
		info.ConnectionMax = maxConn.Int64
	}

	// 后端状态统计
	rows, err := db.Query(`
        SELECT 
            CASE WHEN state IS NULL THEN 'waiting' ELSE state END AS state,
            COUNT(*)
        FROM pg_stat_activity
        GROUP BY state;
    `)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var state string
			var count int64
			if err := rows.Scan(&state, &count); err != nil {
				continue
			}

			switch state {
			case "active":
				info.ActiveBackends = count
			case "idle":
				info.IdleBackends = count
			case "waiting":
				info.WaitingBackends = count
			default:
				// 其他状态可忽略或单独处理
			}
		}
	}

	// 获取启动时间（估算为 postmaster 启动时间）
	var startTimeStr string
	err = db.QueryRow("SELECT pg_postmaster_start_time();").Scan(&startTimeStr)
	if err == nil {
		startTime, _ := time.Parse(time.RFC3339, startTimeStr)
		now := time.Now()
		info.UptimeSeconds = now.Sub(startTime).Seconds()
	} else {
		info.UptimeSeconds = 0
	}

	return info, nil
}

// 解析版本字符串，只保留主版本
func parseVersion(raw string) string {
	parts := strings.Split(raw, " ")
	if len(parts) >= 2 {
		return parts[1]
	}
	return raw
}

package collector

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"opengauss_exporter/internal/model"

	"github.com/sirupsen/logrus"
)

// ScrapeOpenGaussInfo 采集 OpenGauss 的基本信息并返回结构体
func ScrapeOpenGaussInfo(db *sql.DB) (*model.OpenGaussInfo, error) {
	logrus.Debug("Starting OpenGauss info collection")
	ctx := context.Background()

	info := &model.OpenGaussInfo{}

	// 检查是否在线
	if err := db.PingContext(ctx); err != nil {
		logrus.Debugf("Database ping failed: %v", err)
		return info, err
	}
	info.Up = true
	logrus.Debug("Database is up and responsive")

	// 获取版本号
	var version string
	err := db.QueryRow("SELECT version();").Scan(&version)
	if err == nil {
		info.Version = parseVersion(version)
		logrus.Debugf("Database version: %s", info.Version)
	} else {
		logrus.Debugf("Failed to get version: %v", err)
	}

	// 获取数据库数量
	var dbCount sql.NullInt64
	err = db.QueryRow("SELECT COUNT(*) FROM pg_database WHERE NOT datistemplate;").Scan(&dbCount)
	if err == nil && dbCount.Valid {
		info.DatabaseCount = dbCount.Int64
		logrus.Debugf("Database count: %d", info.DatabaseCount)
	} else {
		logrus.Debugf("Failed to get database count: %v", err)
	}

	// 当前连接数
	var connCount sql.NullInt64
	err = db.QueryRow("SELECT COUNT(*) FROM pg_stat_activity;").Scan(&connCount)
	if err == nil && connCount.Valid {
		info.ConnectionCurrent = connCount.Int64
		logrus.Debugf("Current connections: %d", info.ConnectionCurrent)
	} else {
		logrus.Debugf("Failed to get current connections: %v", err)
	}

	// 最大连接数限制
	var maxConn sql.NullInt64
	err = db.QueryRow("SHOW max_connections;").Scan(&maxConn)
	if err == nil && maxConn.Valid {
		info.ConnectionMax = maxConn.Int64
		logrus.Debugf("Max connections: %d", info.ConnectionMax)
	} else {
		logrus.Debugf("Failed to get max connections: %v", err)
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
		var statesFound int
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
			statesFound++
		}
		logrus.Debugf("Backend states collected: %d states (active=%d, idle=%d, waiting=%d)",
			statesFound, info.ActiveBackends, info.IdleBackends, info.WaitingBackends)
	} else {
		logrus.Debugf("Failed to get backend states: %v", err)
	}

	// 获取启动时间（估算为 postmaster 启动时间）
	var startTimeStr string
	err = db.QueryRow("SELECT pg_postmaster_start_time();").Scan(&startTimeStr)
	if err == nil {
		startTime, parseErr := time.Parse(time.RFC3339, startTimeStr)
		if parseErr == nil {
			now := time.Now()
			info.UptimeSeconds = now.Sub(startTime).Seconds()
			logrus.Debugf("Database uptime: %.0f seconds", info.UptimeSeconds)
		} else {
			logrus.Debugf("Failed to parse start time: %v", parseErr)
			info.UptimeSeconds = 0
		}
	} else {
		logrus.Debugf("Failed to get postmaster start time: %v", err)
		info.UptimeSeconds = 0
	}

	logrus.Debugf("Successfully collected OpenGauss info: version=%s, up=%t, databases=%d, current/max_conn=%d/%d",
		info.Version, info.Up, info.DatabaseCount, info.ConnectionCurrent, info.ConnectionMax)

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

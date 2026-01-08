package collector

import (
	"context"
	"database/sql"

	"opengauss_exporter/internal/model"

	"github.com/sirupsen/logrus"
)

// ScrapePgStatActivity 采集 pg_stat_activity 视图的指标数据
func ScrapePgStatActivity(db *sql.DB) (*model.PgStatActivity, error) {
	logrus.Debug("Starting pg_stat_activity collection")
	ctx := context.Background()
	activity := model.NewPgStatActivity()

	// 按状态分组统计连接数
	if err := collectConnectionsByState(ctx, db, activity); err != nil {
		logrus.Debugf("Failed to collect connections by state: %v", err)
		return activity, err
	}

	// 按数据库分组统计连接数
	if err := collectConnectionsByDatabase(ctx, db, activity); err != nil {
		logrus.Debugf("Failed to collect connections by database: %v", err)
		return activity, err
	}

	// 按用户分组统计连接数
	if err := collectConnectionsByUser(ctx, db, activity); err != nil {
		logrus.Debugf("Failed to collect connections by user: %v", err)
		return activity, err
	}

	// 等待事件统计
	if err := collectWaitEventStats(ctx, db, activity); err != nil {
		logrus.Debugf("Failed to collect wait event stats: %v", err)
		return activity, err
	}

	// 长时间运行查询统计
	if err := collectLongRunningQueries(ctx, db, activity); err != nil {
		logrus.Debugf("Failed to collect long running queries: %v", err)
		return activity, err
	}

	// 总连接数
	if err := collectTotalConnections(ctx, db, activity); err != nil {
		logrus.Debugf("Failed to collect total connections: %v", err)
		return activity, err
	}

	logrus.Debugf("Successfully collected activity stats: total=%d, active=%d, idle=%d, databases=%d, users=%d, long_queries=%d",
		activity.TotalConnections, activity.ActiveConnections, activity.IdleConnections,
		len(activity.ConnectionsByDatabase), len(activity.ConnectionsByUser), activity.LongRunningQueries)

	return activity, nil
}

// 按状态分组统计连接数
func collectConnectionsByState(ctx context.Context, db *sql.DB, activity *model.PgStatActivity) error {
	logrus.Debug("Collecting connections by state")
	query := `
		SELECT 
			COALESCE(state, 'unknown') AS state,
			COUNT(*) as count
		FROM pg_stat_activity
		WHERE pid != pg_backend_pid()  -- 排除当前连接
		GROUP BY state`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	var stateCount int
	for rows.Next() {
		var state string
		var count int64
		if err := rows.Scan(&state, &count); err != nil {
			continue
		}

		switch state {
		case "active":
			activity.ActiveConnections = count
		case "idle":
			activity.IdleConnections = count
		case "idle in transaction":
			activity.IdleInTransactionConnections = count
		case "waiting":
			activity.WaitingConnections = count
		default:
			activity.OtherConnections += count
		}
		stateCount++
	}

	logrus.Debugf("Collected %d connection states", stateCount)
	return nil
}

// 按数据库分组统计连接数
func collectConnectionsByDatabase(ctx context.Context, db *sql.DB, activity *model.PgStatActivity) error {
	logrus.Debug("Collecting connections by database")
	query := `
		SELECT 
			COALESCE(datname, 'unknown') AS database_name,
			COUNT(*) as count
		FROM pg_stat_activity
		WHERE pid != pg_backend_pid()
		GROUP BY datname`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var dbName string
		var count int64
		if err := rows.Scan(&dbName, &count); err != nil {
			continue
		}
		activity.ConnectionsByDatabase[dbName] = count
	}

	logrus.Debugf("Collected connections for %d databases", len(activity.ConnectionsByDatabase))
	return nil
}

// 按用户分组统计连接数
func collectConnectionsByUser(ctx context.Context, db *sql.DB, activity *model.PgStatActivity) error {
	logrus.Debug("Collecting connections by user")
	query := `
		SELECT 
			COALESCE(usename, 'unknown') AS username,
			COUNT(*) as count
		FROM pg_stat_activity
		WHERE pid != pg_backend_pid()
		GROUP BY usename`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var username string
		var count int64
		if err := rows.Scan(&username, &count); err != nil {
			continue
		}
		activity.ConnectionsByUser[username] = count
	}

	logrus.Debugf("Collected connections for %d users", len(activity.ConnectionsByUser))
	return nil
}

// 等待事件统计
func collectWaitEventStats(ctx context.Context, db *sql.DB, activity *model.PgStatActivity) error {
	logrus.Debug("Collecting wait event statistics")
	// 首先检查是否存在 wait_event_type 列（PostgreSQL 9.6+）
	checkColumnQuery := `
		SELECT COUNT(*)
		FROM information_schema.columns 
		WHERE table_name = 'pg_stat_activity' 
		  AND column_name = 'wait_event_type'`

	var columnExists int
	err := db.QueryRowContext(ctx, checkColumnQuery).Scan(&columnExists)
	if err != nil || columnExists == 0 {
		logrus.Debug("wait_event_type column does not exist, skipping wait event stats")
		// 如果列不存在，跳过等待事件统计
		// 为了保持指标一致性，添加一个默认的 'none' 类型统计
		activity.WaitEventStats["none"] = 0
		return nil
	}

	query := `
		SELECT 
			COALESCE(wait_event_type, 'none') AS wait_event_type,
			COUNT(*) as count
		FROM pg_stat_activity
		WHERE pid != pg_backend_pid()
		GROUP BY wait_event_type`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		logrus.Debugf("Wait event query failed: %v", err)
		// 如果查询失败，也跳过等待事件统计
		activity.WaitEventStats["none"] = 0
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		var waitEventType string
		var count int64
		if err := rows.Scan(&waitEventType, &count); err != nil {
			continue
		}
		activity.WaitEventStats[waitEventType] = count
	}

	logrus.Debugf("Collected %d wait event types", len(activity.WaitEventStats))
	return nil
}

// 长时间运行查询统计
func collectLongRunningQueries(ctx context.Context, db *sql.DB, activity *model.PgStatActivity) error {
	logrus.Debug("Collecting long running queries statistics")
	// 统计运行超过5分钟的查询
	longRunningQuery := `
		SELECT COUNT(*)
		FROM pg_stat_activity
		WHERE pid != pg_backend_pid()
		  AND state = 'active'
		  AND query_start < NOW() - INTERVAL '5 minute'`

	err := db.QueryRowContext(ctx, longRunningQuery).Scan(&activity.LongRunningQueries)
	if err != nil && err != sql.ErrNoRows {
		logrus.Debugf("Long running queries count failed: %v", err)
		// 如果查询失败，设置为0并继续
		activity.LongRunningQueries = 0
	}

	// 获取最长运行查询的时间
	oldestQueryQuery := `
		SELECT COALESCE(
			EXTRACT(EPOCH FROM (NOW() - MIN(query_start))), 0
		)
		FROM pg_stat_activity
		WHERE pid != pg_backend_pid()
		  AND state = 'active'
		  AND query_start IS NOT NULL`

	err = db.QueryRowContext(ctx, oldestQueryQuery).Scan(&activity.OldestQueryDuration)
	if err != nil && err != sql.ErrNoRows {
		logrus.Debugf("Oldest query duration failed: %v", err)
		// 如果查询失败，设置为0并继续
		activity.OldestQueryDuration = 0
	}

	logrus.Debugf("Long running queries: %d, oldest query duration: %.2f seconds",
		activity.LongRunningQueries, activity.OldestQueryDuration)
	return nil
}

// 总连接数统计
func collectTotalConnections(ctx context.Context, db *sql.DB, activity *model.PgStatActivity) error {
	logrus.Debug("Collecting total connections count")
	query := `
		SELECT COUNT(*)
		FROM pg_stat_activity
		WHERE pid != pg_backend_pid()`

	err := db.QueryRowContext(ctx, query).Scan(&activity.TotalConnections)
	if err != nil && err != sql.ErrNoRows {
		logrus.Debugf("Total connections count failed: %v", err)
		activity.TotalConnections = 0
	} else {
		logrus.Debugf("Total connections: %d", activity.TotalConnections)
	}

	return nil
}

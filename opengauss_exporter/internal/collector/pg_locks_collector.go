package collector

import (
	"context"
	"database/sql"

	"opengauss_exporter/internal/model"

	"github.com/sirupsen/logrus"
)

// ScrapePgLocks 采集 pg_locks 视图的锁统计数据
func ScrapePgLocks(db *sql.DB) (*model.PgLocksStat, error) {
	logrus.Debug("Starting pg_locks collection")
	ctx := context.Background()
	locksStat := model.NewPgLocksStat()

	// 检查pg_locks视图是否存在
	checkQuery := `
		SELECT COUNT(*)
		FROM information_schema.tables 
		WHERE table_name = 'pg_locks'`

	var tableExists int
	err := db.QueryRowContext(ctx, checkQuery).Scan(&tableExists)
	if err != nil || tableExists == 0 {
		logrus.Debug("pg_locks table does not exist, returning default values")
		// 如果表不存在，返回默认值
		return locksStat, nil
	}

	// 按锁类型统计
	if err := collectLocksByType(ctx, db, locksStat); err == nil {
		logrus.Debug("Successfully collected locks by type")
		// 继续收集其他统计
		_ = collectLocksByMode(ctx, db, locksStat)
		_ = collectLocksByState(ctx, db, locksStat)
		_ = collectLocksByDatabase(ctx, db, locksStat)
		_ = collectTotalLocks(ctx, db, locksStat)
	} else {
		logrus.Debugf("Failed to collect locks by type: %v", err)
	}

	logrus.Debugf("Successfully collected pg_locks data: %d types, %d modes, %d states, %d databases, %d total locks",
		len(locksStat.LocksByType), len(locksStat.LocksByMode),
		len(locksStat.LocksByState), len(locksStat.LocksByDatabase), locksStat.TotalLocks)

	return locksStat, nil
}

// collectLocksByType 按锁类型统计
func collectLocksByType(ctx context.Context, db *sql.DB, locksStat *model.PgLocksStat) error {
	logrus.Debug("Collecting locks by type")
	query := `
		SELECT 
			COALESCE(locktype, 'unknown') as locktype,
			COUNT(*) as count
		FROM pg_locks
		GROUP BY locktype`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		logrus.Debugf("Failed to query locks by type: %v", err)
		return err
	}
	defer rows.Close()

	var processedCount int
	for rows.Next() {
		var locktype string
		var count int64
		if err := rows.Scan(&locktype, &count); err != nil {
			logrus.Debugf("Failed to scan lock type row: %v", err)
			continue
		}
		locksStat.LocksByType[locktype] = count
		processedCount++
	}

	logrus.Debugf("Collected %d lock types", processedCount)
	return nil
}

// collectLocksByMode 按锁模式统计
func collectLocksByMode(ctx context.Context, db *sql.DB, locksStat *model.PgLocksStat) error {
	logrus.Debug("Collecting locks by mode")
	query := `
		SELECT 
			COALESCE(mode, 'unknown') as mode,
			COUNT(*) as count
		FROM pg_locks
		GROUP BY mode`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		logrus.Debugf("Failed to query locks by mode: %v", err)
		return err
	}
	defer rows.Close()

	var processedCount int
	for rows.Next() {
		var mode string
		var count int64
		if err := rows.Scan(&mode, &count); err != nil {
			logrus.Debugf("Failed to scan lock mode row: %v", err)
			continue
		}
		locksStat.LocksByMode[mode] = count
		processedCount++
	}

	logrus.Debugf("Collected %d lock modes", processedCount)
	return nil
}

// collectLocksByState 按锁状态统计
func collectLocksByState(ctx context.Context, db *sql.DB, locksStat *model.PgLocksStat) error {
	logrus.Debug("Collecting locks by state")
	query := `
		SELECT 
			CASE WHEN granted THEN 'granted' ELSE 'waiting' END as state,
			COUNT(*) as count
		FROM pg_locks
		GROUP BY granted`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		logrus.Debugf("Failed to query locks by state: %v", err)
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var state string
		var count int64
		if err := rows.Scan(&state, &count); err != nil {
			logrus.Debugf("Failed to scan lock state row: %v", err)
			continue
		}
		locksStat.LocksByState[state] = count

		// 同时更新等待锁和已授予锁的计数
		if state == "waiting" {
			locksStat.WaitingLocks = count
			logrus.Debugf("Found %d waiting locks", count)
		} else if state == "granted" {
			locksStat.GrantedLocks = count
			logrus.Debugf("Found %d granted locks", count)
		}
	}

	return nil
}

// collectLocksByDatabase 按数据库统计
func collectLocksByDatabase(ctx context.Context, db *sql.DB, locksStat *model.PgLocksStat) error {
	logrus.Debug("Collecting locks by database")
	query := `
		SELECT 
			COALESCE(d.datname, 'unknown') as database_name,
			COUNT(*) as count
		FROM pg_locks l
		LEFT JOIN pg_database d ON l.database = d.oid
		WHERE l.database IS NOT NULL
		GROUP BY d.datname`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		logrus.Debugf("Failed to query locks by database: %v", err)
		return err
	}
	defer rows.Close()

	var processedCount int
	for rows.Next() {
		var dbName string
		var count int64
		if err := rows.Scan(&dbName, &count); err != nil {
			logrus.Debugf("Failed to scan lock database row: %v", err)
			continue
		}
		locksStat.LocksByDatabase[dbName] = count
		processedCount++
	}

	logrus.Debugf("Collected locks for %d databases", processedCount)
	return nil
}

// collectTotalLocks 统计总锁数量
func collectTotalLocks(ctx context.Context, db *sql.DB, locksStat *model.PgLocksStat) error {
	logrus.Debug("Collecting total locks count")
	query := `SELECT COUNT(*) FROM pg_locks`

	err := db.QueryRowContext(ctx, query).Scan(&locksStat.TotalLocks)
	if err != nil {
		logrus.Debugf("Failed to get total locks count: %v", err)
		locksStat.TotalLocks = 0
	} else {
		logrus.Debugf("Total locks count: %d", locksStat.TotalLocks)
	}

	return err
}

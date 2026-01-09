package collector

import (
	"context"
	"database/sql"
	"time"

	"opengauss_exporter/internal/model"

	"github.com/sirupsen/logrus"
)

// ScrapePgStatDatabase 采集 pg_stat_database 视图的数据库统计数据
func ScrapePgStatDatabase(db *sql.DB) (*model.PgStatDatabaseCollection, error) {
	logrus.Debug("Starting pg_stat_database collection")
	ctx := context.Background()
	collection := model.NewPgStatDatabaseCollection()

	// 首先尝试完整查询
	if err := scrapePgStatDatabaseFull(ctx, db, collection); err != nil {
		logrus.Debugf("Full query failed, trying compatible query: %v", err)
		// 如果完整查询失败，尝试兼容性查询
		return scrapePgStatDatabaseCompatible(ctx, db, collection)
	}

	logrus.Debugf("Successfully collected pg_stat_database data for %d databases", len(collection.Databases))
	return collection, nil
}

// scrapePgStatDatabaseFull 完整查询pg_stat_database视图
func scrapePgStatDatabaseFull(ctx context.Context, db *sql.DB, collection *model.PgStatDatabaseCollection) error {
	logrus.Debug("Attempting full pg_stat_database query")
	query := `
		SELECT 
			COALESCE(datid, 0) as datid,
			COALESCE(datname, 'unknown') as datname,
			COALESCE(numbackends, 0) as numbackends,
			COALESCE(xact_commit, 0) as xact_commit,
			COALESCE(xact_rollback, 0) as xact_rollback,
			COALESCE(blks_read, 0) as blks_read,
			COALESCE(blks_hit, 0) as blks_hit,
			COALESCE(tup_returned, 0) as tup_returned,
			COALESCE(tup_fetched, 0) as tup_fetched,
			COALESCE(tup_inserted, 0) as tup_inserted,
			COALESCE(tup_updated, 0) as tup_updated,
			COALESCE(tup_deleted, 0) as tup_deleted,
			COALESCE(conflicts, 0) as conflicts,
			COALESCE(temp_files, 0) as temp_files,
			COALESCE(temp_bytes, 0) as temp_bytes,
			COALESCE(deadlocks, 0) as deadlocks,
			stats_reset
		FROM pg_stat_database
		WHERE datname IS NOT NULL 
		  AND datname NOT IN ('template0', 'template1')`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		logrus.Debugf("Full query execution failed: %v", err)
		return err
	}
	defer rows.Close()

	var processedCount int
	for rows.Next() {
		dbStat := model.NewPgStatDatabase()
		var statsResetStr sql.NullString

		err := rows.Scan(
			&dbStat.DatID,
			&dbStat.DatName,
			&dbStat.NumBackends,
			&dbStat.XactCommit,
			&dbStat.XactRollback,
			&dbStat.BlksRead,
			&dbStat.BlksHit,
			&dbStat.TupReturned,
			&dbStat.TupFetched,
			&dbStat.TupInserted,
			&dbStat.TupUpdated,
			&dbStat.TupDeleted,
			&dbStat.Conflicts,
			&dbStat.TempFiles,
			&dbStat.TempBytes,
			&dbStat.Deadlocks,
			&statsResetStr,
		)
		if err != nil {
			logrus.Debugf("Failed to scan database statistics row: %v", err)
			continue
		}

		// 解析统计重置时间
		if statsResetStr.Valid && statsResetStr.String != "" {
			if resetTime, parseErr := time.Parse(time.RFC3339, statsResetStr.String); parseErr == nil {
				dbStat.StatsReset = &resetTime
			} else {
				logrus.Debugf("Failed to parse stats_reset time for database %s: %v", dbStat.DatName, parseErr)
			}
		}

		collection.Databases[dbStat.DatName] = dbStat
		processedCount++
		logrus.Debugf("Database %s: backends=%d, commits=%d, rollbacks=%d",
			dbStat.DatName, dbStat.NumBackends, dbStat.XactCommit, dbStat.XactRollback)
	}

	logrus.Debugf("Full query processed %d databases successfully", processedCount)
	return nil
}

// scrapePgStatDatabaseCompatible 兼容性查询，适用于可能缺少某些字段的数据库版本
func scrapePgStatDatabaseCompatible(ctx context.Context, db *sql.DB, collection *model.PgStatDatabaseCollection) (*model.PgStatDatabaseCollection, error) {
	logrus.Debug("Attempting compatible pg_stat_database query")

	// 首先检查表是否存在
	checkTableQuery := `
		SELECT COUNT(*)
		FROM information_schema.tables 
		WHERE table_name = 'pg_stat_database'`

	var tableExists int
	err := db.QueryRowContext(ctx, checkTableQuery).Scan(&tableExists)
	if err != nil || tableExists == 0 {
		logrus.Debug("pg_stat_database table does not exist, returning empty collection")
		// 如果表不存在，返回空集合
		return collection, nil
	}

	// 获取基础字段（这些字段在大多数版本中都存在）
	basicQuery := `
		SELECT 
			COALESCE(datid, 0) as datid,
			COALESCE(datname, 'unknown') as datname,
			COALESCE(numbackends, 0) as numbackends,
			COALESCE(xact_commit, 0) as xact_commit,
			COALESCE(xact_rollback, 0) as xact_rollback,
			COALESCE(blks_read, 0) as blks_read,
			COALESCE(blks_hit, 0) as blks_hit,
			COALESCE(tup_returned, 0) as tup_returned,
			COALESCE(tup_fetched, 0) as tup_fetched,
			COALESCE(tup_inserted, 0) as tup_inserted,
			COALESCE(tup_updated, 0) as tup_updated,
			COALESCE(tup_deleted, 0) as tup_deleted
		FROM pg_stat_database
		WHERE datname IS NOT NULL 
		  AND datname NOT IN ('template0', 'template1')`

	rows, err := db.QueryContext(ctx, basicQuery)
	if err != nil {
		logrus.Debugf("Compatible query execution failed: %v", err)
		return collection, nil
	}
	defer rows.Close()

	var processedCount int
	for rows.Next() {
		dbStat := model.NewPgStatDatabase()

		err := rows.Scan(
			&dbStat.DatID,
			&dbStat.DatName,
			&dbStat.NumBackends,
			&dbStat.XactCommit,
			&dbStat.XactRollback,
			&dbStat.BlksRead,
			&dbStat.BlksHit,
			&dbStat.TupReturned,
			&dbStat.TupFetched,
			&dbStat.TupInserted,
			&dbStat.TupUpdated,
			&dbStat.TupDeleted,
		)
		if err != nil {
			logrus.Debugf("Failed to scan basic database statistics row: %v", err)
			continue
		}

		// 尝试获取可选字段
		_ = getOptionalDatabaseField(ctx, db, dbStat.DatName, "conflicts", &dbStat.Conflicts)
		_ = getOptionalDatabaseField(ctx, db, dbStat.DatName, "temp_files", &dbStat.TempFiles)
		_ = getOptionalDatabaseField(ctx, db, dbStat.DatName, "temp_bytes", &dbStat.TempBytes)
		_ = getOptionalDatabaseField(ctx, db, dbStat.DatName, "deadlocks", &dbStat.Deadlocks)

		collection.Databases[dbStat.DatName] = dbStat
		processedCount++
	}

	logrus.Debugf("Compatible query processed %d databases successfully", processedCount)
	return collection, nil
}

// getOptionalDatabaseField 获取可选字段值
func getOptionalDatabaseField(ctx context.Context, db *sql.DB, datname, fieldName string, target *int64) error {
	query := `SELECT COALESCE(` + fieldName + `, 0) FROM pg_stat_database WHERE datname = $1 LIMIT 1`
	err := db.QueryRowContext(ctx, query, datname).Scan(target)
	if err != nil {
		logrus.Debugf("Failed to get optional field %s for database %s: %v", fieldName, datname, err)
		*target = 0 // 如果字段不存在或查询失败，设置为0
	}
	return err
}

package collector

import (
	"context"
	"database/sql"
	"time"

	"opengauss_exporter/internal/model"

	"github.com/sirupsen/logrus"
)

// ScrapePgStatUserTables 采集 pg_stat_user_tables 视图的用户表统计数据
func ScrapePgStatUserTables(db *sql.DB) (*model.PgStatUserTablesCollection, error) {
	logrus.Debug("Starting pg_stat_user_tables collection")
	ctx := context.Background()
	collection := model.NewPgStatUserTablesCollection()

	// 首先尝试完整查询
	if err := scrapePgStatUserTablesFull(ctx, db, collection); err != nil {
		logrus.Debugf("Full query failed, trying compatible query: %v", err)
		// 如果完整查询失败，尝试兼容性查询
		return scrapePgStatUserTablesCompatible(ctx, db, collection)
	}

	logrus.Debugf("Successfully collected pg_stat_user_tables data for %d tables", len(collection.Tables))
	return collection, nil
}

// scrapePgStatUserTablesFull 完整查询pg_stat_user_tables视图
func scrapePgStatUserTablesFull(ctx context.Context, db *sql.DB, collection *model.PgStatUserTablesCollection) error {
	logrus.Debug("Attempting full pg_stat_user_tables query")
	query := `
		SELECT 
			COALESCE(schemaname, 'public') as schemaname,
			COALESCE(tablename, 'unknown') as tablename,
			COALESCE(relid, 0) as relid,
			COALESCE(seq_scan, 0) as seq_scan,
			COALESCE(seq_tup_read, 0) as seq_tup_read,
			COALESCE(idx_scan, 0) as idx_scan,
			COALESCE(idx_tup_fetch, 0) as idx_tup_fetch,
			COALESCE(n_tup_ins, 0) as n_tup_ins,
			COALESCE(n_tup_upd, 0) as n_tup_upd,
			COALESCE(n_tup_del, 0) as n_tup_del,
			COALESCE(n_tup_hot_upd, 0) as n_tup_hot_upd,
			COALESCE(n_live_tup, 0) as n_live_tup,
			COALESCE(n_dead_tup, 0) as n_dead_tup,
			COALESCE(n_mod_since_analyze, 0) as n_mod_since_analyze,
			COALESCE(vacuum_count, 0) as vacuum_count,
			COALESCE(autovacuum_count, 0) as autovacuum_count,
			last_vacuum,
			last_autovacuum,
			COALESCE(analyze_count, 0) as analyze_count,
			COALESCE(autoanalyze_count, 0) as autoanalyze_count,
			last_analyze,
			last_autoanalyze
		FROM pg_stat_user_tables`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		logrus.Debugf("Full query execution failed: %v", err)
		return err
	}
	defer rows.Close()

	var processedCount int
	for rows.Next() {
		tableStat := model.NewPgStatUserTable()
		var lastVacuum, lastAutovacuum, lastAnalyze, lastAutoanalyze sql.NullString

		err := rows.Scan(
			&tableStat.SchemaName,
			&tableStat.TableName,
			&tableStat.RelID,
			&tableStat.SeqScan,
			&tableStat.SeqTupRead,
			&tableStat.IdxScan,
			&tableStat.IdxTupFetch,
			&tableStat.NTupIns,
			&tableStat.NTupUpd,
			&tableStat.NTupDel,
			&tableStat.NTupHotUpd,
			&tableStat.NLiveTup,
			&tableStat.NDeadTup,
			&tableStat.NModSinceAnalyze,
			&tableStat.VacuumCount,
			&tableStat.AutovacuumCount,
			&lastVacuum,
			&lastAutovacuum,
			&tableStat.AnalyzeCount,
			&tableStat.AutoanalyzeCount,
			&lastAnalyze,
			&lastAutoanalyze,
		)
		if err != nil {
			logrus.Debugf("Failed to scan row for table: %v", err)
			continue
		}

		// 解析时间字段
		parseTimeField(lastVacuum, &tableStat.LastVacuum)
		parseTimeField(lastAutovacuum, &tableStat.LastAutovacuum)
		parseTimeField(lastAnalyze, &tableStat.LastAnalyze)
		parseTimeField(lastAutoanalyze, &tableStat.LastAutoanalyze)

		collection.Tables[tableStat.GetTableKey()] = tableStat
		processedCount++
	}

	logrus.Debugf("Full query processed %d user tables successfully", processedCount)
	return nil
}

// scrapePgStatUserTablesCompatible 兼容性查询，适用于可能缺少某些字段的数据库版本
func scrapePgStatUserTablesCompatible(ctx context.Context, db *sql.DB, collection *model.PgStatUserTablesCollection) (*model.PgStatUserTablesCollection, error) {
	logrus.Debug("Attempting compatible pg_stat_user_tables query")

	// 首先检查表是否存在
	checkTableQuery := `
		SELECT COUNT(*)
		FROM information_schema.tables 
		WHERE table_name = 'pg_stat_user_tables'`

	var tableExists int
	err := db.QueryRowContext(ctx, checkTableQuery).Scan(&tableExists)
	if err != nil || tableExists == 0 {
		logrus.Debug("pg_stat_user_tables table does not exist, returning empty collection")
		return collection, nil
	}

	// 获取基础字段（这些字段在大多数版本中都存在）
	basicQuery := `
		SELECT 
			COALESCE(schemaname, 'public') as schemaname,
			COALESCE(tablename, 'unknown') as tablename,
			COALESCE(relid, 0) as relid,
			COALESCE(seq_scan, 0) as seq_scan,
			COALESCE(seq_tup_read, 0) as seq_tup_read,
			COALESCE(idx_scan, 0) as idx_scan,
			COALESCE(idx_tup_fetch, 0) as idx_tup_fetch,
			COALESCE(n_tup_ins, 0) as n_tup_ins,
			COALESCE(n_tup_upd, 0) as n_tup_upd,
			COALESCE(n_tup_del, 0) as n_tup_del,
			COALESCE(n_live_tup, 0) as n_live_tup,
			COALESCE(n_dead_tup, 0) as n_dead_tup
		FROM pg_stat_user_tables`

	rows, err := db.QueryContext(ctx, basicQuery)
	if err != nil {
		logrus.Debugf("Compatible query execution failed: %v", err)
		return collection, nil
	}
	defer rows.Close()

	var processedCount int
	for rows.Next() {
		tableStat := model.NewPgStatUserTable()

		err := rows.Scan(
			&tableStat.SchemaName,
			&tableStat.TableName,
			&tableStat.RelID,
			&tableStat.SeqScan,
			&tableStat.SeqTupRead,
			&tableStat.IdxScan,
			&tableStat.IdxTupFetch,
			&tableStat.NTupIns,
			&tableStat.NTupUpd,
			&tableStat.NTupDel,
			&tableStat.NLiveTup,
			&tableStat.NDeadTup,
		)
		if err != nil {
			logrus.Debugf("Failed to scan basic row for table: %v", err)
			continue
		}

		// 尝试获取可选字段
		_ = getOptionalUserTableField(ctx, db, tableStat.SchemaName, tableStat.TableName, "n_tup_hot_upd", &tableStat.NTupHotUpd)
		_ = getOptionalUserTableField(ctx, db, tableStat.SchemaName, tableStat.TableName, "n_mod_since_analyze", &tableStat.NModSinceAnalyze)
		_ = getOptionalUserTableField(ctx, db, tableStat.SchemaName, tableStat.TableName, "vacuum_count", &tableStat.VacuumCount)
		_ = getOptionalUserTableField(ctx, db, tableStat.SchemaName, tableStat.TableName, "autovacuum_count", &tableStat.AutovacuumCount)
		_ = getOptionalUserTableField(ctx, db, tableStat.SchemaName, tableStat.TableName, "analyze_count", &tableStat.AnalyzeCount)
		_ = getOptionalUserTableField(ctx, db, tableStat.SchemaName, tableStat.TableName, "autoanalyze_count", &tableStat.AutoanalyzeCount)

		collection.Tables[tableStat.GetTableKey()] = tableStat
		processedCount++
	}

	logrus.Debugf("Compatible query processed %d user tables successfully", processedCount)
	return collection, nil
}

// getOptionalUserTableField 获取用户表的可选字段值
func getOptionalUserTableField(ctx context.Context, db *sql.DB, schemaname, tablename, fieldName string, target *int64) error {
	query := `SELECT COALESCE(` + fieldName + `, 0) FROM pg_stat_user_tables WHERE schemaname = $1 AND tablename = $2 LIMIT 1`
	err := db.QueryRowContext(ctx, query, schemaname, tablename).Scan(target)
	if err != nil {
		logrus.Debugf("Failed to get optional field %s for table %s.%s: %v", fieldName, schemaname, tablename, err)
		*target = 0 // 如果字段不存在或查询失败，设置为0
	}
	return err
}

// parseTimeField 解析时间字段
func parseTimeField(nullStr sql.NullString, target **time.Time) {
	if nullStr.Valid && nullStr.String != "" {
		if parsedTime, err := time.Parse(time.RFC3339, nullStr.String); err == nil {
			*target = &parsedTime
		} else {
			logrus.Debugf("Failed to parse time field: %v", err)
		}
	}
}

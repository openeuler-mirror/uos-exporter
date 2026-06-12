package collector

import (
	"context"
	"database/sql"

	"opengauss_exporter/internal/model"

	"github.com/sirupsen/logrus"
)

// ScrapePgStatUserIndexes 采集 pg_stat_user_indexes 视图的用户索引统计数据
func ScrapePgStatUserIndexes(db *sql.DB) (*model.PgStatUserIndexesCollection, error) {
	logrus.Debug("Starting pg_stat_user_indexes collection")
	ctx := context.Background()
	collection := model.NewPgStatUserIndexesCollection()

	// 检查表是否存在
	checkTableQuery := `
		SELECT COUNT(*)
		FROM information_schema.tables 
		WHERE table_name = 'pg_stat_user_indexes'`

	var tableExists int
	err := db.QueryRowContext(ctx, checkTableQuery).Scan(&tableExists)
	if err != nil || tableExists == 0 {
		logrus.Debug("pg_stat_user_indexes table does not exist, returning empty collection")
		return collection, nil
	}

	logrus.Debug("Executing pg_stat_user_indexes query")
	query := `
		SELECT 
			COALESCE(schemaname, 'public') as schemaname,
			COALESCE(tablename, 'unknown') as tablename,
			COALESCE(indexrelname, 'unknown') as indexrelname,
			COALESCE(indexrelid, 0) as indexrelid,
			COALESCE(idx_scan, 0) as idx_scan,
			COALESCE(idx_tup_read, 0) as idx_tup_read,
			COALESCE(idx_tup_fetch, 0) as idx_tup_fetch
		FROM pg_stat_user_indexes`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		logrus.Debugf("Failed to execute pg_stat_user_indexes query: %v", err)
		return collection, nil
	}
	defer rows.Close()

	var processedCount int
	for rows.Next() {
		indexStat := model.NewPgStatUserIndex()

		err := rows.Scan(
			&indexStat.SchemaName,
			&indexStat.TableName,
			&indexStat.IndexName,
			&indexStat.IndexID,
			&indexStat.IdxScan,
			&indexStat.IdxTupRead,
			&indexStat.IdxTupFetch,
		)
		if err != nil {
			logrus.Debugf("Failed to scan index statistics row: %v", err)
			continue
		}

		collection.Indexes[indexStat.GetIndexKey()] = indexStat
		processedCount++

		// 记录前5个索引的详细信息，避免日志过多
		if processedCount <= 5 {
			logrus.Debugf("Index %s.%s.%s - scans: %d, tup_read: %d, tup_fetch: %d",
				indexStat.SchemaName, indexStat.TableName, indexStat.IndexName,
				indexStat.IdxScan, indexStat.IdxTupRead, indexStat.IdxTupFetch)
		}
	}

	logrus.Debugf("Successfully collected pg_stat_user_indexes data for %d indexes", processedCount)
	return collection, nil
}

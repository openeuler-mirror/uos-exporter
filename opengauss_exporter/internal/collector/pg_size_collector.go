package collector

import (
	"context"
	"database/sql"

	"opengauss_exporter/internal/model"

	"github.com/sirupsen/logrus"
)

// ScrapePgSizeStats 采集数据库、表、表空间大小统计数据
func ScrapePgSizeStats(db *sql.DB) (*model.PgSizeStats, error) {
	logrus.Debug("Starting pg_size_stats collection")
	ctx := context.Background()
	sizeStats := model.NewPgSizeStats()

	// 收集数据库大小
	if err := collectDatabaseSizes(ctx, db, sizeStats); err == nil {
		logrus.Debugf("Successfully collected database sizes: %d databases", len(sizeStats.DatabaseSizes))
		// 收集表大小
		_ = collectTableSizes(ctx, db, sizeStats)
		// 收集表空间大小
		_ = collectTablespaceSizes(ctx, db, sizeStats)
	} else {
		logrus.Debugf("Failed to collect database sizes: %v", err)
	}

	logrus.Debugf("Successfully collected pg_size_stats: %d databases, %d tables, %d tablespaces, total DB size: %d bytes",
		len(sizeStats.DatabaseSizes), len(sizeStats.TableSizes), len(sizeStats.TablespaceSizes), sizeStats.TotalDatabaseSize)

	return sizeStats, nil
}

// collectDatabaseSizes 收集数据库大小统计
func collectDatabaseSizes(ctx context.Context, db *sql.DB, sizeStats *model.PgSizeStats) error {
	logrus.Debug("Collecting database sizes")
	query := `
		SELECT 
			datname,
			pg_database_size(datname) as size
		FROM pg_database
		WHERE datname IS NOT NULL 
		  AND datname NOT IN ('template0', 'template1')
		  AND datistemplate = false`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		logrus.Debugf("Failed to query database sizes: %v", err)
		return err
	}
	defer rows.Close()

	var totalSize int64
	var processedCount int
	for rows.Next() {
		var datname string
		var size int64
		if err := rows.Scan(&datname, &size); err != nil {
			logrus.Debugf("Failed to scan database size row: %v", err)
			continue
		}

		dbSize := &model.PgDatabaseSize{
			DatName: datname,
			Size:    size,
		}
		sizeStats.DatabaseSizes[datname] = dbSize
		totalSize += size
		processedCount++
		logrus.Debugf("Database %s size: %d bytes", datname, size)
	}

	sizeStats.TotalDatabaseSize = totalSize
	logrus.Debugf("Collected %d database sizes, total: %d bytes", processedCount, totalSize)
	return nil
}

// collectTableSizes 收集表大小统计
func collectTableSizes(ctx context.Context, db *sql.DB, sizeStats *model.PgSizeStats) error {
	logrus.Debug("Collecting table sizes")
	query := `
		SELECT 
			schemaname,
			tablename,
			pg_relation_size(schemaname||'.'||tablename) as size,
			pg_total_relation_size(schemaname||'.'||tablename) as total_size
		FROM pg_tables
		WHERE schemaname NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
		  AND schemaname NOT LIKE 'pg_temp_%'`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		logrus.Debugf("Failed to query table sizes: %v", err)
		return err
	}
	defer rows.Close()

	var totalTableSize int64
	var processedCount int
	for rows.Next() {
		var schemaname, tablename string
		var size, totalSize int64
		if err := rows.Scan(&schemaname, &tablename, &size, &totalSize); err != nil {
			logrus.Debugf("Failed to scan table size row: %v", err)
			continue
		}

		tableSize := &model.PgTableSize{
			SchemaName: schemaname,
			TableName:  tablename,
			Size:       size,
			TotalSize:  totalSize,
		}
		sizeStats.TableSizes[tableSize.GetTableKey()] = tableSize
		totalTableSize += totalSize
		processedCount++
		if processedCount <= 10 { // 只记录前10个表的详细信息，避免日志过多
			logrus.Debugf("Table %s.%s size: %d bytes (total: %d bytes)", schemaname, tablename, size, totalSize)
		}
	}

	sizeStats.TotalTableSize = totalTableSize
	logrus.Debugf("Collected %d table sizes, total: %d bytes", processedCount, totalTableSize)
	return nil
}

// collectTablespaceSizes 收集表空间大小统计
func collectTablespaceSizes(ctx context.Context, db *sql.DB, sizeStats *model.PgSizeStats) error {
	logrus.Debug("Collecting tablespace sizes")
	query := `
		SELECT 
			spcname,
			pg_tablespace_size(spcname) as size
		FROM pg_tablespace`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		logrus.Debugf("Failed to query tablespace sizes (may not be supported): %v", err)
		// 如果查询失败，可能是权限问题或版本不支持，继续执行
		return nil
	}
	defer rows.Close()

	var processedCount int
	for rows.Next() {
		var spcname string
		var size int64
		if err := rows.Scan(&spcname, &size); err != nil {
			logrus.Debugf("Failed to scan tablespace size row: %v", err)
			continue
		}

		tablespaceSize := &model.PgTablespaceSize{
			TablespaceName: spcname,
			Size:           size,
		}
		sizeStats.TablespaceSizes[spcname] = tablespaceSize
		processedCount++
		logrus.Debugf("Tablespace %s size: %d bytes", spcname, size)
	}

	logrus.Debugf("Collected %d tablespace sizes", processedCount)
	return nil
}

package collector

import (
	"context"
	"database/sql"
	"time"

	"opengauss_exporter/internal/model"

	"github.com/sirupsen/logrus"
)

// ScrapePgStatBgwriter 采集 pg_stat_bgwriter 视图的后台写进程统计数据
func ScrapePgStatBgwriter(db *sql.DB) (*model.PgStatBgwriter, error) {
	logrus.Debug("Starting pg_stat_bgwriter collection")
	ctx := context.Background()
	bgwriter := model.NewPgStatBgwriter()

	// 查询pg_stat_bgwriter视图
	query := `
		SELECT 
			COALESCE(checkpoints_timed, 0) as checkpoints_timed,
			COALESCE(checkpoints_req, 0) as checkpoints_req,
			COALESCE(checkpoint_write_time, 0) as checkpoint_write_time,
			COALESCE(checkpoint_sync_time, 0) as checkpoint_sync_time,
			COALESCE(buffers_checkpoint, 0) as buffers_checkpoint,
			COALESCE(buffers_clean, 0) as buffers_clean,
			COALESCE(maxwritten_clean, 0) as maxwritten_clean,
			COALESCE(buffers_backend, 0) as buffers_backend,
			COALESCE(buffers_backend_fsync, 0) as buffers_backend_fsync,
			COALESCE(buffers_alloc, 0) as buffers_alloc,
			stats_reset
		FROM pg_stat_bgwriter`

	var statsResetStr sql.NullString

	err := db.QueryRowContext(ctx, query).Scan(
		&bgwriter.CheckpointsTimed,
		&bgwriter.CheckpointsReq,
		&bgwriter.CheckpointWriteTime,
		&bgwriter.CheckpointSyncTime,
		&bgwriter.BuffersCheckpoint,
		&bgwriter.BuffersClean,
		&bgwriter.MaxwrittenClean,
		&bgwriter.BuffersBackend,
		&bgwriter.BuffersBackendFsync,
		&bgwriter.BuffersAlloc,
		&statsResetStr,
	)
	if err != nil {
		logrus.Debugf("Full bgwriter query failed, trying compatible query: %v", err)
		// 如果查询失败，可能是因为某些字段在OpenGauss中不存在
		// 尝试简化查询
		return scrapePgStatBgwriterCompatible(ctx, db, bgwriter)
	}

	// 解析统计重置时间
	if statsResetStr.Valid && statsResetStr.String != "" {
		if resetTime, parseErr := time.Parse(time.RFC3339, statsResetStr.String); parseErr == nil {
			bgwriter.StatsReset = &resetTime
		} else {
			logrus.Debugf("Failed to parse bgwriter stats_reset time: %v", parseErr)
		}
	}

	logrus.Debugf("Successfully collected bgwriter stats: checkpoints_timed=%d, checkpoints_req=%d, buffers_alloc=%d",
		bgwriter.CheckpointsTimed, bgwriter.CheckpointsReq, bgwriter.BuffersAlloc)
	return bgwriter, nil
}

// scrapePgStatBgwriterCompatible 兼容性查询，适用于可能缺少某些字段的数据库版本
func scrapePgStatBgwriterCompatible(ctx context.Context, db *sql.DB, bgwriter *model.PgStatBgwriter) (*model.PgStatBgwriter, error) {
	logrus.Debug("Attempting compatible bgwriter query")

	// 首先检查表是否存在
	checkTableQuery := `
		SELECT COUNT(*)
		FROM information_schema.tables 
		WHERE table_name = 'pg_stat_bgwriter'`

	var tableExists int
	err := db.QueryRowContext(ctx, checkTableQuery).Scan(&tableExists)
	if err != nil || tableExists == 0 {
		logrus.Debug("pg_stat_bgwriter table does not exist, returning default values")
		// 如果表不存在，返回默认值
		return bgwriter, nil
	}

	// 逐个检查和获取字段
	var fieldsFound int
	if err := getBgwriterField(ctx, db, "checkpoints_timed", &bgwriter.CheckpointsTimed); err == nil {
		fieldsFound++
		_ = getBgwriterField(ctx, db, "checkpoints_req", &bgwriter.CheckpointsReq)
		_ = getBgwriterField(ctx, db, "buffers_checkpoint", &bgwriter.BuffersCheckpoint)
		_ = getBgwriterField(ctx, db, "buffers_clean", &bgwriter.BuffersClean)
		_ = getBgwriterField(ctx, db, "maxwritten_clean", &bgwriter.MaxwrittenClean)
		_ = getBgwriterField(ctx, db, "buffers_backend", &bgwriter.BuffersBackend)
		_ = getBgwriterField(ctx, db, "buffers_alloc", &bgwriter.BuffersAlloc)

		// 获取时间字段
		_ = getBgwriterTimeField(ctx, db, "checkpoint_write_time", &bgwriter.CheckpointWriteTime)
		_ = getBgwriterTimeField(ctx, db, "checkpoint_sync_time", &bgwriter.CheckpointSyncTime)
		_ = getBgwriterField(ctx, db, "buffers_backend_fsync", &bgwriter.BuffersBackendFsync)
		fieldsFound += 9
	}

	logrus.Debugf("Compatible bgwriter query collected %d fields successfully", fieldsFound)
	return bgwriter, nil
}

// getBgwriterField 获取单个整型字段值
func getBgwriterField(ctx context.Context, db *sql.DB, fieldName string, target *int64) error {
	query := `SELECT COALESCE(` + fieldName + `, 0) FROM pg_stat_bgwriter LIMIT 1`
	err := db.QueryRowContext(ctx, query).Scan(target)
	if err != nil {
		logrus.Debugf("Failed to get bgwriter field %s: %v", fieldName, err)
		*target = 0
	}
	return err
}

// getBgwriterTimeField 获取单个时间字段值（转换为浮点数）
func getBgwriterTimeField(ctx context.Context, db *sql.DB, fieldName string, target *float64) error {
	query := `SELECT COALESCE(` + fieldName + `, 0) FROM pg_stat_bgwriter LIMIT 1`
	err := db.QueryRowContext(ctx, query).Scan(target)
	if err != nil {
		logrus.Debugf("Failed to get bgwriter time field %s: %v", fieldName, err)
		*target = 0
	}
	return err
}

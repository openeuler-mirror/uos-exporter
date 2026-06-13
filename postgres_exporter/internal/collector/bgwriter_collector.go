package collector

import (
	"context"
	"database/sql"
	"fmt"

	"postgres_exporter/internal/model"
)

// ScrapePostgreSQLBGWriter 采集后台写入器指标
func ScrapePostgreSQLBGWriter(db *sql.DB) (*model.PostgreSQLBGWriterStats, error) {
	ctx := context.Background()

	stats := &model.PostgreSQLBGWriter{
		CheckpointsScheduled: 0,
		CheckpointsRequested: 0,
		CheckpointWriteTime:  0,
		CheckpointSyncTime:   0,
		BuffersCheckpoint:    0,
		BuffersClean:         0,
		MaxWaitTime:          0,
		AvgWaitTime:          0,
		BuffersBackend:       0,
		BuffersAllocated:     0,
		StatsReset:           "",
	}

	row := db.QueryRowContext(ctx, `
        SELECT 
            checkpoints_timed,
            checkpoints_req,
            checkpoint_write_time,
            checkpoint_sync_time,
            buffers_checkpoint,
            buffers_clean,
            maxwritten_clean,
            buffers_backend,
            buffers_alloc,
            stats_reset
        FROM pg_stat_bgwriter;
    `)

	var resetTime sql.NullString
	var checkTimed sql.NullInt64
	var checkReq sql.NullInt64
	var writeTime sql.NullFloat64
	var syncTime sql.NullFloat64
	var bufsCheck sql.NullInt64
	var bufsClean sql.NullInt64
	var maxWritten sql.NullInt64
	var bufsBackend sql.NullInt64
	var bufsAlloc sql.NullInt64

	err := row.Scan(
		&checkTimed,
		&checkReq,
		&writeTime,
		&syncTime,
		&bufsCheck,
		&bufsClean,
		&maxWritten,
		&bufsBackend,
		&bufsAlloc,
		&resetTime,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan bgwriter stats: %v", err)
	}

	// 计算平均等待时间（假设基于 buffers_clean 和 checkpoint_write_time）
	var avgWait float64 = 0
	if bufsClean.Valid && bufsClean.Int64 > 0 {
		avgWait = writeTime.Float64 / float64(bufsClean.Int64)
	}

	stats.CheckpointsScheduled = coalesceNullInt64(checkTimed)
	stats.CheckpointsRequested = coalesceNullInt64(checkReq)
	stats.CheckpointWriteTime = coalesceNullFloat64(writeTime)
	stats.CheckpointSyncTime = coalesceNullFloat64(syncTime)
	stats.BuffersCheckpoint = coalesceNullInt64(bufsCheck)
	stats.BuffersClean = coalesceNullInt64(bufsClean)
	stats.MaxWaitTime = float64(coalesceNullInt64(maxWritten) * 10.0) // 假设每个 page 10ms 写出
	stats.AvgWaitTime = avgWait
	stats.BuffersBackend = coalesceNullInt64(bufsBackend)
	stats.BuffersAllocated = coalesceNullInt64(bufsAlloc)
	stats.StatsReset = coalesceNullString(resetTime)

	return &model.PostgreSQLBGWriterStats{
		BGWriter: stats,
	}, nil
}

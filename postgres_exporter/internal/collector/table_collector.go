package collector

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"postgres_exporter/internal/model"
)

// ScrapePostgreSQLTables 采集所有用户表的指标数据
func ScrapePostgreSQLTables(db *sql.DB) (*model.PostgreSQLTableStats, error) {
	ctx := context.Background()

	stats := &model.PostgreSQLTableStats{
		Tables: []*model.PostgreSQLTable{},
	}

	rows, err := db.QueryContext(ctx, `
        SELECT 
            relid,
            schemaname,
            relname,
            seq_scan,
            seq_tup_read,
            idx_scan,
            idx_tup_fetch,
            n_tup_ins,
            n_tup_upd,
            n_tup_del,
            n_tup_hot_upd,
            n_live_tup,
            n_dead_tup,
            n_mod_since_analyze,
            n_ins_since_vacuum,
            last_vacuum,
            last_autovacuum,
            last_analyze,
            last_autoanalyze,
            vacuum_count,
            autovacuum_count,
            analyze_count,
            autoanalyze_count
        FROM pg_stat_user_tables;
    `)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables stats: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var relID sql.NullInt64
		var schema sql.NullString
		var name sql.NullString

		var seqScan sql.NullInt64
		var seqTupRead sql.NullInt64
		var idxScan sql.NullInt64
		var idxTupFetch sql.NullInt64

		var inserts sql.NullInt64
		var updates sql.NullInt64
		var deletes sql.NullInt64
		var hotUpdates sql.NullInt64

		var liveTuples sql.NullInt64
		var deadTuples sql.NullInt64
		var nModSinceAnalyze sql.NullInt64
		var nInsSinceVacuum sql.NullInt64

		var lastVacuum sql.NullString
		var lastAutoVacuum sql.NullString
		var lastAnalyze sql.NullString
		var lastAutoAnalyze sql.NullString

		var vacuumCount sql.NullInt64
		var autoVacuumCount sql.NullInt64
		var analyzeCount sql.NullInt64
		var autoAnalyzeCount sql.NullInt64

		err := rows.Scan(
			&relID,
			&schema,
			&name,
			&seqScan,
			&seqTupRead,
			&idxScan,
			&idxTupFetch,
			&inserts,
			&updates,
			&deletes,
			&hotUpdates,
			&liveTuples,
			&deadTuples,
			&nModSinceAnalyze,
			&nInsSinceVacuum,
			&lastVacuum,
			&lastAutoVacuum,
			&lastAnalyze,
			&lastAutoAnalyze,
			&vacuumCount,
			&autoVacuumCount,
			&analyzeCount,
			&autoAnalyzeCount,
		)
		if err != nil {
			continue
		}

		table := &model.PostgreSQLTable{
			RelID:  coalesceNullInt64(relID),
			Schema: coalesceNullString(schema),
			Name:   coalesceNullString(name),

			SeqScans:    coalesceNullInt64(seqScan),
			SeqTupRead:  coalesceNullInt64(seqTupRead),
			IdxScans:    coalesceNullInt64(idxScan),
			IdxTupFetch: coalesceNullInt64(idxTupFetch),

			Inserts:    coalesceNullInt64(inserts),
			Updates:    coalesceNullInt64(updates),
			Deletes:    coalesceNullInt64(deletes),
			HotUpdates: coalesceNullInt64(hotUpdates),

			LiveTuples:       coalesceNullInt64(liveTuples),
			DeadTuples:       coalesceNullInt64(deadTuples),
			NModSinceAnalyze: coalesceNullInt64(nModSinceAnalyze),
			NInsSinceVacuum:  coalesceNullInt64(nInsSinceVacuum),

			VacuumCount:      coalesceNullInt64(vacuumCount),
			AutoVacuumCount:  coalesceNullInt64(autoVacuumCount),
			AnalyzeCount:     coalesceNullInt64(analyzeCount),
			AutoAnalyzeCount: coalesceNullInt64(autoAnalyzeCount),

			LastVacuum:      parseTime(coalesceNullString(lastVacuum)),
			LastAutoVacuum:  parseTime(coalesceNullString(lastAutoVacuum)),
			LastAnalyze:     parseTime(coalesceNullString(lastAnalyze)),
			LastAutoAnalyze: parseTime(coalesceNullString(lastAutoAnalyze)),
		}

		stats.Tables = append(stats.Tables, table)
	}

	return stats, nil
}

// 工具函数：处理 NullInt64
func coalesceNullInt64(val sql.NullInt64) int64 {
	if val.Valid {
		return val.Int64
	}
	return 0
}

// 工具函数：处理 NullString
func coalesceNullString(val sql.NullString) string {
	if val.Valid {
		return val.String
	}
	return ""
}

// 解析时间字符串为 time.Time
func parseTime(timeStr string) *time.Time {
	if timeStr == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02 15:04:05", timeStr)
	if err != nil {
		return nil
	}
	return &t
}

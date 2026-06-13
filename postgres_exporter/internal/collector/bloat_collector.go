package collector

import (
	"context"
	"database/sql"
	"fmt"

	"postgres_exporter/internal/model"
)

// ScrapePostgreSQLBloat 采集所有表和索引的膨胀信息（兼容 PG 15+）
func ScrapePostgreSQLBloat(db *sql.DB) (*model.PostgreSQLBloatStats, error) {
	ctx := context.Background()

	stats := &model.PostgreSQLBloatStats{
		Tables:  []*model.PostgreSQLTableBloat{},
		Indices: []*model.PostgreSQLIndexBloat{},
	}

	// --- 1. 表膨胀查询 ---
	tableRows, err := db.QueryContext(ctx, `
        WITH table_estimates AS (
            SELECT
                n.nspname AS schemaname,
                c.relname AS tablename,
                pg_table_size(c.oid) AS total_bytes,
                COALESCE(
                    (SELECT SUM(COALESCE(avg_width, a.attlen::numeric))
                     FROM pg_attribute a
                     LEFT JOIN pg_stats s ON a.attrelid = s.schemaname::regclass AND a.attname = s.attname
                     WHERE a.attrelid = c.oid AND a.attnum > 0 AND NOT a.attisdropped),
                    0
                ) AS avg_row_width,
                c.reltuples
            FROM pg_class c
            JOIN pg_namespace n ON n.oid = c.relnamespace
            WHERE c.relkind IN ('r', 'm') -- 普通表和物化视图
              AND n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
              AND c.reltuples > 0
        ),
        table_estimates_with_expected AS (
            SELECT
                schemaname,
                tablename,
                total_bytes,
                avg_row_width,
                reltuples,
                CASE WHEN avg_row_width > 0 THEN
                    ceil(reltuples / ((current_setting('block_size')::numeric - 24) / (avg_row_width + 24))) *
                    current_setting('block_size')::bigint
                ELSE 0
                END AS expected_bytes
            FROM table_estimates
        )
        SELECT 
            schemaname,
            tablename,
            total_bytes,
            expected_bytes,
            (total_bytes / greatest(expected_bytes, 1))::float8 AS bloat_ratio,
            (total_bytes - expected_bytes) / (1024 * 1024) AS wasted_mb
        FROM table_estimates_with_expected
        WHERE total_bytes > expected_bytes
        ORDER BY wasted_mb DESC;
    `)
	if err == nil {
		defer tableRows.Close()
		for tableRows.Next() {
			var schema sql.NullString
			var name sql.NullString
			var totalBytes sql.NullInt64
			var expectedBytes sql.NullInt64
			var bloatRatio sql.NullFloat64
			var wastedMB sql.NullFloat64

			err := tableRows.Scan(&schema, &name, &totalBytes, &expectedBytes, &bloatRatio, &wastedMB)
			if err != nil {
				continue
			}

			stats.Tables = append(stats.Tables, &model.PostgreSQLTableBloat{
				Database:      "your_db", // Exporter 注入
				Schema:        coalesceNullString(schema),
				Table:         coalesceNullString(name),
				TotalBytes:    coalesceNullInt64(totalBytes),
				ExpectedBytes: coalesceNullInt64(expectedBytes),
				BloatRatio:    coalesceNullFloat64(bloatRatio),
				WastedMB:      coalesceNullFloat64(wastedMB),
			})
		}
	} else {
		return stats, fmt.Errorf("failed to query table bloat: %v", err)
	}

	// --- 2. 索引膨胀查询 ---
	indexRows, err := db.QueryContext(ctx, `
        WITH index_estimates AS (
            SELECT
                c.oid,
                n.nspname AS schemaname,
                t.relname AS tablename,
                c.relname AS indexname,
                current_setting('block_size')::int AS blocksize,
                GREATEST(
                    ceil(pg_stat_user_indexes.idx_tup_read / least(0.1 * current_setting('block_size')::int, 1024)),
                    1
                ) * current_setting('block_size')::bigint AS expected_bytes
            FROM pg_class c
            JOIN pg_index i ON c.oid = i.indexrelid
            JOIN pg_class t ON i.indrelid = t.oid
            JOIN pg_namespace n ON n.oid = c.relnamespace
            JOIN pg_stat_user_indexes ON pg_stat_user_indexes.indexrelid = c.oid
            WHERE c.relkind = 'i' -- indexes
              AND n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
        ),
        index_bloat AS (
            SELECT 
                schemaname,
                tablename,
                indexname,
                pg_relation_size(oid) AS total_bytes,
                expected_bytes,
                (pg_relation_size(oid) / greatest(expected_bytes, 1))::float8 AS bloat_ratio
            FROM index_estimates
            WHERE pg_relation_size(oid) > expected_bytes
        )
        SELECT 
            schemaname,
            tablename,
            indexname,
            bloat_ratio,
            (total_bytes - expected_bytes) / (1024 * 1024) AS wasted_mb
        FROM index_bloat
        ORDER BY wasted_mb DESC;
    `)
	if err == nil {
		defer indexRows.Close()
		for indexRows.Next() {
			var schema sql.NullString
			var table sql.NullString
			var name sql.NullString
			var bloatRatio sql.NullFloat64
			var wastedMB sql.NullFloat64

			err := indexRows.Scan(&schema, &table, &name, &bloatRatio, &wastedMB)
			if err != nil {
				continue
			}

			stats.Indices = append(stats.Indices, &model.PostgreSQLIndexBloat{
				Database:   "your_db", // 由 Exporter 注入
				Schema:     coalesceNullString(schema),
				Table:      coalesceNullString(table),
				Index:      coalesceNullString(name),
				BloatRatio: coalesceNullFloat64(bloatRatio),
				WastedMB:   coalesceNullFloat64(wastedMB),
			})
		}
	} else {
		return stats, fmt.Errorf("failed to query index bloat: %v", err)
	}

	return stats, nil
}

// 工具函数：处理 NullFloat64
func coalesceNullFloat64(val sql.NullFloat64) float64 {
	if val.Valid {
		return val.Float64
	}
	return 0
}

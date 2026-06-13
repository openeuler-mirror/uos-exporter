package collector

import (
	"context"
	"database/sql"
	"fmt"

	"postgres_exporter/internal/model"
)

// ScrapePostgreSQLIndexes 使用你提供的 SQL 查询采集索引数据
func ScrapePostgreSQLIndexes(db *sql.DB) (*model.PostgreSQLIndexStats, error) {
	ctx := context.Background()

	stats := &model.PostgreSQLIndexStats{
		Indices: []*model.PostgreSQLIndex{},
	}

	rows, err := db.QueryContext(ctx, `
        SELECT 
            t.relname AS tablename,
            i.relname AS indexname,
            pg_relation_size(i.oid) AS size_bytes,
            s.idx_scan,
            s.idx_tup_read AS tuples_read,
            s.idx_tup_fetch AS tuples_fetched,
            (x.indisunique)::int AS is_unique,
            am.amname AS index_type,
            GREATEST(p.last_analyze, p.last_autoanalyze) AS last_analyzed_time
        FROM pg_index x
        JOIN pg_class i ON i.oid = x.indexrelid AND i.relkind = 'i'
        JOIN pg_class t ON t.oid = x.indrelid AND t.relkind = 'r'
        JOIN pg_stat_user_indexes s ON s.indexrelid = i.oid
        JOIN pg_am am ON am.oid = i.relam
        LEFT JOIN pg_stat_all_tables p ON p.relid = x.indrelid
        WHERE i.relkind = 'i';
    `)
	if err != nil {
		return nil, fmt.Errorf("failed to query index stats: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var table sql.NullString
		var name sql.NullString
		var sizeBytes sql.NullInt64
		var scanCount sql.NullInt64
		var tupRead sql.NullInt64
		var tupFetch sql.NullInt64
		var isUnique sql.NullInt64
		var indexType sql.NullString
		var lastAnalyzed sql.NullString

		err := rows.Scan(
			&table,
			&name,
			&sizeBytes,
			&scanCount,
			&tupRead,
			&tupFetch,
			&isUnique,
			&indexType,
			&lastAnalyzed,
		)
		if err != nil {
			continue
		}

		index := &model.PostgreSQLIndex{
			Table:            coalesceNullString(table),
			Name:             coalesceNullString(name),
			SizeBytes:        coalesceNullInt64(sizeBytes),
			ScanCount:        coalesceNullInt64(scanCount),
			TupRead:          coalesceNullInt64(tupRead),
			TupFetch:         coalesceNullInt64(tupFetch),
			Unique:           isUnique.Valid && (isUnique.Int64 == 1),
			IndexType:        coalesceNullString(indexType),
			LastAnalyzedTime: parseTime(coalesceNullString(lastAnalyzed)),
		}

		stats.Indices = append(stats.Indices, index)
	}

	return stats, nil
}

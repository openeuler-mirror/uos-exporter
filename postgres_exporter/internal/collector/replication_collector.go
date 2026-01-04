package collector

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"postgres_exporter/internal/model"
)

// ScrapePostgreSQLReplication 采集主从复制状态和延迟信息
func ScrapePostgreSQLReplication(db *sql.DB) (*model.PostgreSQLReplicationStats, error) {
	ctx := context.Background()

	stats := &model.PostgreSQLReplicationStats{
		Replicas: []*model.PostgreSQLReplication{},
	}

	rows, err := db.QueryContext(ctx, `
        SELECT 
            application_name,
            state,
            sync_state,
            client_addr,
            EXTRACT(EPOCH FROM write_lag)::float8,
            EXTRACT(EPOCH FROM flush_lag)::float8,
            EXTRACT(EPOCH FROM replay_lag)::float8,
            sent_lsn::pg_lsn - '0/00000000'::pg_lsn,
            write_lsn::pg_lsn - '0/00000000'::pg_lsn,
            flush_lsn::pg_lsn - '0/00000000'::pg_lsn,
            replay_lsn::pg_lsn - '0/00000000'::pg_lsn
        FROM pg_stat_replication;
    `)
	if err != nil {
		return stats, fmt.Errorf("failed to query replication stats: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var appName sql.NullString
		var state sql.NullString
		var syncState sql.NullString
		var clientAddr sql.NullString
		var writeLag sql.NullFloat64
		var flushLag sql.NullFloat64
		var replayLag sql.NullFloat64
		var bytesSent sql.NullInt64
		var writeLoc sql.NullInt64
		var flushLoc sql.NullInt64
		var replayLoc sql.NullInt64
		var latestWAL sql.NullString

		err := rows.Scan(
			&appName,
			&state,
			&syncState,
			&clientAddr,
			&writeLag,
			&flushLag,
			&replayLag,
			&bytesSent,
			&writeLoc,
			&flushLoc,
			&replayLoc,
			&latestWAL,
		)
		if err != nil {
			continue
		}

		replica := &model.PostgreSQLReplication{
			ApplicationName: coalesceNullString(appName),
			State:           coalesceNullString(state),
			SyncState:       coalesceNullString(syncState),
			ClientAddress:   coalesceNullString(clientAddr),
			WriteLag:        coalesceNullFloat64(writeLag),
			FlushLag:        coalesceNullFloat64(flushLag),
			ReplayLag:       coalesceNullFloat64(replayLag),
			BytesSent:       coalesceNullInt64(bytesSent),
			WriteLocation:   coalesceNullInt64(writeLoc),
			FlushLocation:   coalesceNullInt64(flushLoc),
			ReplayLocation:  coalesceNullInt64(replayLoc),
			LatestWALSent:   coalesceNullString(latestWAL),
			LastUpdated:     time.Now(),
			IsSyncStandby:   false,
		}

		if replica.SyncState == "sync" || replica.SyncState == "quorum" {
			replica.IsSyncStandby = true
		}

		stats.Replicas = append(stats.Replicas, replica)
	}

	return stats, nil
}

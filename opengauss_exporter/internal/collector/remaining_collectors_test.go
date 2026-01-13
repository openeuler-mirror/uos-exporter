package collector

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============= pg_stat_bgwriter tests =============

func TestScrapePgStatBgwriter_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock successful full query
	statsResetTime := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	mock.ExpectQuery("SELECT.*FROM pg_stat_bgwriter").
		WillReturnRows(sqlmock.NewRows([]string{
			"checkpoints_timed", "checkpoints_req", "checkpoint_write_time", "checkpoint_sync_time",
			"buffers_checkpoint", "buffers_clean", "maxwritten_clean", "buffers_backend",
			"buffers_backend_fsync", "buffers_alloc", "stats_reset",
		}).
			AddRow(100, 50, 1500.5, 200.3, 10000, 5000, 20, 8000, 10, 50000, statsResetTime))

	bgwriter, err := ScrapePgStatBgwriter(db)

	assert.NoError(t, err)
	assert.Equal(t, int64(100), bgwriter.CheckpointsTimed)
	assert.Equal(t, int64(50), bgwriter.CheckpointsReq)
	assert.Equal(t, 1500.5, bgwriter.CheckpointWriteTime)
	assert.Equal(t, 200.3, bgwriter.CheckpointSyncTime)
	assert.Equal(t, int64(10000), bgwriter.BuffersCheckpoint)
	assert.Equal(t, int64(5000), bgwriter.BuffersClean)
	assert.Equal(t, int64(20), bgwriter.MaxwrittenClean)
	assert.Equal(t, int64(8000), bgwriter.BuffersBackend)
	assert.Equal(t, int64(10), bgwriter.BuffersBackendFsync)
	assert.Equal(t, int64(50000), bgwriter.BuffersAlloc)
	assert.NotNil(t, bgwriter.StatsReset)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapePgStatBgwriter_CompatibleMode(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock failed full query
	mock.ExpectQuery("SELECT.*FROM pg_stat_bgwriter").
		WillReturnError(sql.ErrConnDone)

	// Mock table existence check
	mock.ExpectQuery("SELECT COUNT.*FROM information_schema.tables.*pg_stat_bgwriter").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(1))

	// Mock individual field queries
	mock.ExpectQuery("SELECT COALESCE.*checkpoints_timed.*FROM pg_stat_bgwriter").
		WillReturnRows(sqlmock.NewRows([]string{"checkpoints_timed"}).
			AddRow(100))
	mock.ExpectQuery("SELECT COALESCE.*checkpoints_req.*FROM pg_stat_bgwriter").
		WillReturnRows(sqlmock.NewRows([]string{"checkpoints_req"}).
			AddRow(50))
	mock.ExpectQuery("SELECT COALESCE.*buffers_checkpoint.*FROM pg_stat_bgwriter").
		WillReturnRows(sqlmock.NewRows([]string{"buffers_checkpoint"}).
			AddRow(10000))
	mock.ExpectQuery("SELECT COALESCE.*buffers_clean.*FROM pg_stat_bgwriter").
		WillReturnRows(sqlmock.NewRows([]string{"buffers_clean"}).
			AddRow(5000))
	mock.ExpectQuery("SELECT COALESCE.*maxwritten_clean.*FROM pg_stat_bgwriter").
		WillReturnRows(sqlmock.NewRows([]string{"maxwritten_clean"}).
			AddRow(20))
	mock.ExpectQuery("SELECT COALESCE.*buffers_backend.*FROM pg_stat_bgwriter").
		WillReturnRows(sqlmock.NewRows([]string{"buffers_backend"}).
			AddRow(8000))
	mock.ExpectQuery("SELECT COALESCE.*buffers_alloc.*FROM pg_stat_bgwriter").
		WillReturnRows(sqlmock.NewRows([]string{"buffers_alloc"}).
			AddRow(50000))
	mock.ExpectQuery("SELECT COALESCE.*checkpoint_write_time.*FROM pg_stat_bgwriter").
		WillReturnRows(sqlmock.NewRows([]string{"checkpoint_write_time"}).
			AddRow(1500.5))
	mock.ExpectQuery("SELECT COALESCE.*checkpoint_sync_time.*FROM pg_stat_bgwriter").
		WillReturnRows(sqlmock.NewRows([]string{"checkpoint_sync_time"}).
			AddRow(200.3))
	mock.ExpectQuery("SELECT COALESCE.*buffers_backend_fsync.*FROM pg_stat_bgwriter").
		WillReturnRows(sqlmock.NewRows([]string{"buffers_backend_fsync"}).
			AddRow(10))

	bgwriter, err := ScrapePgStatBgwriter(db)

	assert.NoError(t, err)
	assert.Equal(t, int64(100), bgwriter.CheckpointsTimed)
	assert.Equal(t, int64(50), bgwriter.CheckpointsReq)
	assert.Equal(t, 1500.5, bgwriter.CheckpointWriteTime)
	assert.Equal(t, 200.3, bgwriter.CheckpointSyncTime)
	assert.Equal(t, int64(10000), bgwriter.BuffersCheckpoint)
	assert.Equal(t, int64(50000), bgwriter.BuffersAlloc)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapePgStatBgwriter_TableNotExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock failed full query
	mock.ExpectQuery("SELECT.*FROM pg_stat_bgwriter").
		WillReturnError(sql.ErrConnDone)

	// Mock table existence check - table doesn't exist
	mock.ExpectQuery("SELECT COUNT.*FROM information_schema.tables.*pg_stat_bgwriter").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(0))

	bgwriter, err := ScrapePgStatBgwriter(db)

	assert.NoError(t, err)
	assert.Equal(t, int64(0), bgwriter.CheckpointsTimed)
	assert.Equal(t, int64(0), bgwriter.CheckpointsReq)
	assert.Equal(t, int64(0), bgwriter.BuffersAlloc)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// ============= pg_stat_user_indexes tests =============

func TestScrapePgStatUserIndexes_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock table existence check
	mock.ExpectQuery("SELECT COUNT.*FROM information_schema.tables.*pg_stat_user_indexes").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(1))

	// Mock successful query
	mock.ExpectQuery("SELECT.*FROM pg_stat_user_indexes").
		WillReturnRows(sqlmock.NewRows([]string{
			"schemaname", "tablename", "indexrelname", "indexrelid",
			"idx_scan", "idx_tup_read", "idx_tup_fetch",
		}).
			AddRow("public", "users", "users_pkey", 12345, 1000, 5000, 4500).
			AddRow("public", "users", "users_email_idx", 12346, 500, 2000, 1800).
			AddRow("public", "orders", "orders_pkey", 12347, 800, 3000, 2800))

	collection, err := ScrapePgStatUserIndexes(db)

	assert.NoError(t, err)
	assert.Equal(t, 3, len(collection.Indexes))

	usersPkeyKey := "public.users.users_pkey"
	usersPkey := collection.Indexes[usersPkeyKey]
	assert.NotNil(t, usersPkey)
	assert.Equal(t, "public", usersPkey.SchemaName)
	assert.Equal(t, "users", usersPkey.TableName)
	assert.Equal(t, "users_pkey", usersPkey.IndexName)
	assert.Equal(t, int64(12345), usersPkey.IndexID)
	assert.Equal(t, int64(1000), usersPkey.IdxScan)
	assert.Equal(t, int64(5000), usersPkey.IdxTupRead)
	assert.Equal(t, int64(4500), usersPkey.IdxTupFetch)

	usersEmailKey := "public.users.users_email_idx"
	usersEmail := collection.Indexes[usersEmailKey]
	assert.NotNil(t, usersEmail)
	assert.Equal(t, "users_email_idx", usersEmail.IndexName)
	assert.Equal(t, int64(500), usersEmail.IdxScan)

	ordersPkeyKey := "public.orders.orders_pkey"
	ordersPkey := collection.Indexes[ordersPkeyKey]
	assert.NotNil(t, ordersPkey)
	assert.Equal(t, "orders", ordersPkey.TableName)
	assert.Equal(t, int64(800), ordersPkey.IdxScan)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapePgStatUserIndexes_TableNotExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock table existence check - table doesn't exist
	mock.ExpectQuery("SELECT COUNT.*FROM information_schema.tables.*pg_stat_user_indexes").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(0))

	collection, err := ScrapePgStatUserIndexes(db)

	assert.NoError(t, err)
	assert.Equal(t, 0, len(collection.Indexes))

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapePgStatUserIndexes_EmptyResults(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock table existence check
	mock.ExpectQuery("SELECT COUNT.*FROM information_schema.tables.*pg_stat_user_indexes").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(1))

	// Mock empty query results
	mock.ExpectQuery("SELECT.*FROM pg_stat_user_indexes").
		WillReturnRows(sqlmock.NewRows([]string{
			"schemaname", "tablename", "indexrelname", "indexrelid",
			"idx_scan", "idx_tup_read", "idx_tup_fetch",
		}))

	collection, err := ScrapePgStatUserIndexes(db)

	assert.NoError(t, err)
	assert.Equal(t, 0, len(collection.Indexes))

	assert.NoError(t, mock.ExpectationsWereMet())
}

// ============= pg_stat_replication tests =============

func TestScrapePgStatReplication_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock table existence check
	mock.ExpectQuery("SELECT COUNT.*FROM information_schema.tables.*pg_stat_replication").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(1))

	// Mock successful query
	backendStartTime := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
	mock.ExpectQuery("SELECT.*FROM pg_stat_replication").
		WillReturnRows(sqlmock.NewRows([]string{
			"application_name", "client_addr", "client_hostname", "state",
			"write_lag", "flush_lag", "replay_lag",
			"sent_lsn", "write_lsn", "flush_lsn", "replay_lsn", "backend_start",
		}).
			AddRow("standby1", "192.168.1.100", "replica1.example.com", "streaming",
				1.5, 2.0, 2.5, "0/3000000", "0/2FF0000", "0/2FE0000", "0/2FD0000", backendStartTime).
			AddRow("standby2", "192.168.1.101", "replica2.example.com", "streaming",
				0.8, 1.2, 1.8, "0/3000000", "0/2FF8000", "0/2FF0000", "0/2FE0000", backendStartTime))

	collection, err := ScrapePgStatReplication(db)

	assert.NoError(t, err)
	assert.Equal(t, 2, len(collection.Replications))
	assert.Equal(t, int64(2), collection.TotalReplicas)

	// Check first replication
	standby1Key := "standby1:192.168.1.100"
	standby1, exists := collection.Replications[standby1Key]
	require.True(t, exists, "standby1 replication should exist")
	require.NotNil(t, standby1, "standby1 should not be nil")

	assert.Equal(t, "standby1", standby1.ApplicationName)
	assert.Equal(t, "192.168.1.100", standby1.ClientAddr)
	assert.Equal(t, "replica1.example.com", standby1.ClientHostname)
	assert.Equal(t, "streaming", standby1.State)
	assert.Equal(t, 1.5, standby1.WriteLag)
	assert.Equal(t, 2.0, standby1.FlushLag)
	assert.Equal(t, 2.5, standby1.ReplayLag)
	assert.Equal(t, "0/3000000", standby1.SentLsn)
	assert.Equal(t, "0/2FF0000", standby1.WriteLsn)
	assert.Equal(t, "0/2FE0000", standby1.FlushLsn)
	assert.Equal(t, "0/2FD0000", standby1.ReplayLsn)
	if standby1.BackendStart != nil {
		assert.True(t, standby1.BackendStart.Before(time.Now()))
	}

	// Check second replication
	standby2Key := "standby2:192.168.1.101"
	standby2, exists := collection.Replications[standby2Key]
	require.True(t, exists, "standby2 replication should exist")
	require.NotNil(t, standby2, "standby2 should not be nil")

	assert.Equal(t, "standby2", standby2.ApplicationName)
	assert.Equal(t, "192.168.1.101", standby2.ClientAddr)
	assert.Equal(t, 0.8, standby2.WriteLag)
	assert.Equal(t, 1.2, standby2.FlushLag)
	assert.Equal(t, 1.8, standby2.ReplayLag)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapePgStatReplication_TableNotExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock table existence check - table doesn't exist
	mock.ExpectQuery("SELECT COUNT.*FROM information_schema.tables.*pg_stat_replication").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(0))

	collection, err := ScrapePgStatReplication(db)

	assert.NoError(t, err)
	assert.Equal(t, 0, len(collection.Replications))
	assert.Equal(t, int64(0), collection.TotalReplicas)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapePgStatReplication_EmptyResults(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock table existence check
	mock.ExpectQuery("SELECT COUNT.*FROM information_schema.tables.*pg_stat_replication").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(1))

	// Mock empty query results
	mock.ExpectQuery("SELECT.*FROM pg_stat_replication").
		WillReturnRows(sqlmock.NewRows([]string{
			"application_name", "client_addr", "client_hostname", "state",
			"write_lag", "flush_lag", "replay_lag",
			"sent_lsn", "write_lsn", "flush_lsn", "replay_lsn", "backend_start",
		}))

	collection, err := ScrapePgStatReplication(db)

	assert.NoError(t, err)
	assert.Equal(t, 0, len(collection.Replications))
	assert.Equal(t, int64(0), collection.TotalReplicas)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapePgStatReplication_ScanErrors(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock table existence check
	mock.ExpectQuery("SELECT COUNT.*FROM information_schema.tables.*pg_stat_replication").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(1))

	// Mock query with scan errors
	mock.ExpectQuery("SELECT.*FROM pg_stat_replication").
		WillReturnRows(sqlmock.NewRows([]string{
			"application_name", "client_addr", "client_hostname", "state",
			"write_lag", "flush_lag", "replay_lag",
			"sent_lsn", "write_lsn", "flush_lsn", "replay_lsn", "backend_start",
		}).
			AddRow("standby1", "192.168.1.100", "replica1.example.com", "streaming",
				"invalid_lag", 2.0, 2.5, "0/3000000", "0/2FF0000", "0/2FE0000", "0/2FD0000", nil). // Invalid write_lag
			AddRow("standby2", "192.168.1.101", "replica2.example.com", "streaming",
				0.8, 1.2, 1.8, "0/3000000", "0/2FF8000", "0/2FF0000", "0/2FE0000", nil)) // Valid row

	collection, err := ScrapePgStatReplication(db)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(collection.Replications)) // Only valid row should be processed
	assert.Equal(t, int64(1), collection.TotalReplicas)

	standby2Key := "standby2:192.168.1.101"
	standby2 := collection.Replications[standby2Key]
	assert.NotNil(t, standby2)
	assert.Equal(t, "standby2", standby2.ApplicationName)
	assert.Equal(t, 0.8, standby2.WriteLag)

	assert.NoError(t, mock.ExpectationsWereMet())
}

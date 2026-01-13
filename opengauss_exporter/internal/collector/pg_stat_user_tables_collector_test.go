package collector

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScrapePgStatUserTables_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock successful full query
	lastVacuumTime := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	mock.ExpectQuery("SELECT.*FROM pg_stat_user_tables").
		WillReturnRows(sqlmock.NewRows([]string{
			"schemaname", "tablename", "relid", "seq_scan", "seq_tup_read", "idx_scan", "idx_tup_fetch",
			"n_tup_ins", "n_tup_upd", "n_tup_del", "n_tup_hot_upd", "n_live_tup", "n_dead_tup",
			"n_mod_since_analyze", "vacuum_count", "autovacuum_count", "last_vacuum", "last_autovacuum",
			"analyze_count", "autoanalyze_count", "last_analyze", "last_autoanalyze",
		}).
			AddRow("public", "users", 12345, 100, 5000, 200, 8000, 500, 100, 50, 75, 1000, 100, 25, 5, 10, lastVacuumTime, lastVacuumTime, 3, 8, lastVacuumTime, lastVacuumTime).
			AddRow("public", "orders", 12346, 50, 2500, 150, 6000, 200, 50, 25, 40, 500, 50, 15, 2, 5, nil, nil, 1, 4, nil, nil))

	collection, err := ScrapePgStatUserTables(db)

	assert.NoError(t, err)
	assert.Equal(t, 2, len(collection.Tables))

	// Check users table
	usersKey := "public.users"
	users := collection.Tables[usersKey]
	assert.NotNil(t, users)
	assert.Equal(t, "public", users.SchemaName)
	assert.Equal(t, "users", users.TableName)
	assert.Equal(t, int64(12345), users.RelID)
	assert.Equal(t, int64(100), users.SeqScan)
	assert.Equal(t, int64(5000), users.SeqTupRead)
	assert.Equal(t, int64(200), users.IdxScan)
	assert.Equal(t, int64(8000), users.IdxTupFetch)
	assert.Equal(t, int64(500), users.NTupIns)
	assert.Equal(t, int64(100), users.NTupUpd)
	assert.Equal(t, int64(50), users.NTupDel)
	assert.Equal(t, int64(75), users.NTupHotUpd)
	assert.Equal(t, int64(1000), users.NLiveTup)
	assert.Equal(t, int64(100), users.NDeadTup)
	assert.Equal(t, int64(25), users.NModSinceAnalyze)
	assert.Equal(t, int64(5), users.VacuumCount)
	assert.Equal(t, int64(10), users.AutovacuumCount)
	assert.NotNil(t, users.LastVacuum)
	assert.NotNil(t, users.LastAutovacuum)
	assert.Equal(t, int64(3), users.AnalyzeCount)
	assert.Equal(t, int64(8), users.AutoanalyzeCount)
	assert.NotNil(t, users.LastAnalyze)
	assert.NotNil(t, users.LastAutoanalyze)

	// Check orders table
	ordersKey := "public.orders"
	orders := collection.Tables[ordersKey]
	assert.NotNil(t, orders)
	assert.Equal(t, "public", orders.SchemaName)
	assert.Equal(t, "orders", orders.TableName)
	assert.Equal(t, int64(12346), orders.RelID)
	assert.Equal(t, int64(50), orders.SeqScan)
	assert.Nil(t, orders.LastVacuum) // Should be nil due to null
	assert.Nil(t, orders.LastAutovacuum)
	assert.Nil(t, orders.LastAnalyze)
	assert.Nil(t, orders.LastAutoanalyze)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapePgStatUserTables_CompatibleMode(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock failed full query
	mock.ExpectQuery("SELECT.*FROM pg_stat_user_tables").
		WillReturnError(sql.ErrConnDone)

	// Mock table existence check
	mock.ExpectQuery("SELECT COUNT.*FROM information_schema.tables.*pg_stat_user_tables").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(1))

	// Mock basic query for compatible mode
	mock.ExpectQuery("SELECT.*FROM pg_stat_user_tables").
		WillReturnRows(sqlmock.NewRows([]string{
			"schemaname", "tablename", "relid", "seq_scan", "seq_tup_read", "idx_scan", "idx_tup_fetch",
			"n_tup_ins", "n_tup_upd", "n_tup_del", "n_live_tup", "n_dead_tup",
		}).
			AddRow("public", "users", 12345, 100, 5000, 200, 8000, 500, 100, 50, 1000, 100))

	// Mock optional field queries with correct parameters
	mock.ExpectQuery("SELECT COALESCE\\(n_tup_hot_upd, 0\\) FROM pg_stat_user_tables WHERE schemaname = \\$1 AND tablename = \\$2 LIMIT 1").
		WithArgs("public", "users").
		WillReturnRows(sqlmock.NewRows([]string{"n_tup_hot_upd"}).
			AddRow(75))
	mock.ExpectQuery("SELECT COALESCE\\(n_mod_since_analyze, 0\\) FROM pg_stat_user_tables WHERE schemaname = \\$1 AND tablename = \\$2 LIMIT 1").
		WithArgs("public", "users").
		WillReturnRows(sqlmock.NewRows([]string{"n_mod_since_analyze"}).
			AddRow(25))
	mock.ExpectQuery("SELECT COALESCE\\(vacuum_count, 0\\) FROM pg_stat_user_tables WHERE schemaname = \\$1 AND tablename = \\$2 LIMIT 1").
		WithArgs("public", "users").
		WillReturnRows(sqlmock.NewRows([]string{"vacuum_count"}).
			AddRow(5))
	mock.ExpectQuery("SELECT COALESCE\\(autovacuum_count, 0\\) FROM pg_stat_user_tables WHERE schemaname = \\$1 AND tablename = \\$2 LIMIT 1").
		WithArgs("public", "users").
		WillReturnRows(sqlmock.NewRows([]string{"autovacuum_count"}).
			AddRow(10))
	mock.ExpectQuery("SELECT COALESCE\\(analyze_count, 0\\) FROM pg_stat_user_tables WHERE schemaname = \\$1 AND tablename = \\$2 LIMIT 1").
		WithArgs("public", "users").
		WillReturnRows(sqlmock.NewRows([]string{"analyze_count"}).
			AddRow(3))
	mock.ExpectQuery("SELECT COALESCE\\(autoanalyze_count, 0\\) FROM pg_stat_user_tables WHERE schemaname = \\$1 AND tablename = \\$2 LIMIT 1").
		WithArgs("public", "users").
		WillReturnRows(sqlmock.NewRows([]string{"autoanalyze_count"}).
			AddRow(8))

	collection, err := ScrapePgStatUserTables(db)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(collection.Tables))

	usersKey := "public.users"
	users := collection.Tables[usersKey]
	assert.NotNil(t, users)
	assert.Equal(t, "public", users.SchemaName)
	assert.Equal(t, "users", users.TableName)
	assert.Equal(t, int64(100), users.SeqScan)
	assert.Equal(t, int64(75), users.NTupHotUpd) // From optional field query
	assert.Equal(t, int64(25), users.NModSinceAnalyze)
	assert.Equal(t, int64(5), users.VacuumCount)
	assert.Equal(t, int64(10), users.AutovacuumCount)
	assert.Equal(t, int64(3), users.AnalyzeCount)
	assert.Equal(t, int64(8), users.AutoanalyzeCount)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapePgStatUserTables_TableNotExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock failed full query
	mock.ExpectQuery("SELECT.*FROM pg_stat_user_tables").
		WillReturnError(sql.ErrConnDone)

	// Mock table existence check - table doesn't exist
	mock.ExpectQuery("SELECT COUNT.*FROM information_schema.tables.*pg_stat_user_tables").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(0))

	collection, err := ScrapePgStatUserTables(db)

	assert.NoError(t, err)
	assert.Equal(t, 0, len(collection.Tables))

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapePgStatUserTables_EmptyResults(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock successful query with empty results
	mock.ExpectQuery("SELECT.*FROM pg_stat_user_tables").
		WillReturnRows(sqlmock.NewRows([]string{
			"schemaname", "tablename", "relid", "seq_scan", "seq_tup_read", "idx_scan", "idx_tup_fetch",
			"n_tup_ins", "n_tup_upd", "n_tup_del", "n_tup_hot_upd", "n_live_tup", "n_dead_tup",
			"n_mod_since_analyze", "vacuum_count", "autovacuum_count", "last_vacuum", "last_autovacuum",
			"analyze_count", "autoanalyze_count", "last_analyze", "last_autoanalyze",
		}))

	collection, err := ScrapePgStatUserTables(db)

	assert.NoError(t, err)
	assert.Equal(t, 0, len(collection.Tables))

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapePgStatUserTables_ScanErrors(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock query with scan errors
	mock.ExpectQuery("SELECT.*FROM pg_stat_user_tables").
		WillReturnRows(sqlmock.NewRows([]string{
			"schemaname", "tablename", "relid", "seq_scan", "seq_tup_read", "idx_scan", "idx_tup_fetch",
			"n_tup_ins", "n_tup_upd", "n_tup_del", "n_tup_hot_upd", "n_live_tup", "n_dead_tup",
			"n_mod_since_analyze", "vacuum_count", "autovacuum_count", "last_vacuum", "last_autovacuum",
			"analyze_count", "autoanalyze_count", "last_analyze", "last_autoanalyze",
		}).
			AddRow("public", "users", "invalid_relid", 100, 5000, 200, 8000, 500, 100, 50, 75, 1000, 100, 25, 5, 10, nil, nil, 3, 8, nil, nil). // Invalid relid
			AddRow("public", "orders", 12346, 50, 2500, 150, 6000, 200, 50, 25, 40, 500, 50, 15, 2, 5, nil, nil, 1, 4, nil, nil))               // Valid row

	collection, err := ScrapePgStatUserTables(db)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(collection.Tables)) // Only valid row should be processed

	ordersKey := "public.orders"
	orders := collection.Tables[ordersKey]
	assert.NotNil(t, orders)
	assert.Equal(t, "public", orders.SchemaName)
	assert.Equal(t, "orders", orders.TableName)
	assert.Equal(t, int64(12346), orders.RelID)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapePgStatUserTables_OptionalFieldFailures(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock failed full query
	mock.ExpectQuery("SELECT.*FROM pg_stat_user_tables").
		WillReturnError(sql.ErrConnDone)

	// Mock table existence check
	mock.ExpectQuery("SELECT COUNT.*FROM information_schema.tables.*pg_stat_user_tables").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(1))

	// Mock basic query for compatible mode
	mock.ExpectQuery("SELECT.*FROM pg_stat_user_tables").
		WillReturnRows(sqlmock.NewRows([]string{
			"schemaname", "tablename", "relid", "seq_scan", "seq_tup_read", "idx_scan", "idx_tup_fetch",
			"n_tup_ins", "n_tup_upd", "n_tup_del", "n_live_tup", "n_dead_tup",
		}).
			AddRow("public", "users", 12345, 100, 5000, 200, 8000, 500, 100, 50, 1000, 100))

	// Mock failed optional field queries
	mock.ExpectQuery("SELECT COALESCE\\(n_tup_hot_upd, 0\\) FROM pg_stat_user_tables WHERE schemaname = \\$1 AND tablename = \\$2 LIMIT 1").
		WithArgs("public", "users").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("SELECT COALESCE\\(n_mod_since_analyze, 0\\) FROM pg_stat_user_tables WHERE schemaname = \\$1 AND tablename = \\$2 LIMIT 1").
		WithArgs("public", "users").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("SELECT COALESCE\\(vacuum_count, 0\\) FROM pg_stat_user_tables WHERE schemaname = \\$1 AND tablename = \\$2 LIMIT 1").
		WithArgs("public", "users").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("SELECT COALESCE\\(autovacuum_count, 0\\) FROM pg_stat_user_tables WHERE schemaname = \\$1 AND tablename = \\$2 LIMIT 1").
		WithArgs("public", "users").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("SELECT COALESCE\\(analyze_count, 0\\) FROM pg_stat_user_tables WHERE schemaname = \\$1 AND tablename = \\$2 LIMIT 1").
		WithArgs("public", "users").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("SELECT COALESCE\\(autoanalyze_count, 0\\) FROM pg_stat_user_tables WHERE schemaname = \\$1 AND tablename = \\$2 LIMIT 1").
		WithArgs("public", "users").
		WillReturnError(sql.ErrNoRows)

	collection, err := ScrapePgStatUserTables(db)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(collection.Tables))

	usersKey := "public.users"
	users := collection.Tables[usersKey]
	assert.NotNil(t, users)
	assert.Equal(t, "public", users.SchemaName)
	assert.Equal(t, "users", users.TableName)
	assert.Equal(t, int64(100), users.SeqScan)
	// Optional fields should be 0 due to failures
	assert.Equal(t, int64(0), users.NTupHotUpd)
	assert.Equal(t, int64(0), users.NModSinceAnalyze)
	assert.Equal(t, int64(0), users.VacuumCount)
	assert.Equal(t, int64(0), users.AutovacuumCount)
	assert.Equal(t, int64(0), users.AnalyzeCount)
	assert.Equal(t, int64(0), users.AutoanalyzeCount)

	assert.NoError(t, mock.ExpectationsWereMet())
}

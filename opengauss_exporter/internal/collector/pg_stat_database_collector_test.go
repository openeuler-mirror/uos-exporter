package collector

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScrapePgStatDatabase_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock successful full query
	statsResetTime := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	mock.ExpectQuery("SELECT.*FROM pg_stat_database.*WHERE datname IS NOT NULL.*template0.*template1").
		WillReturnRows(sqlmock.NewRows([]string{
			"datid", "datname", "numbackends", "xact_commit", "xact_rollback",
			"blks_read", "blks_hit", "tup_returned", "tup_fetched", "tup_inserted",
			"tup_updated", "tup_deleted", "conflicts", "temp_files", "temp_bytes",
			"deadlocks", "stats_reset",
		}).
			AddRow(1, "testdb", 5, 1000, 10, 500, 4500, 10000, 2000, 100, 50, 20, 0, 5, 1024, 1, statsResetTime).
			AddRow(2, "postgres", 3, 800, 5, 300, 3700, 8000, 1500, 80, 40, 15, 1, 3, 512, 0, statsResetTime))

	collection, err := ScrapePgStatDatabase(db)

	assert.NoError(t, err)
	assert.Equal(t, 2, len(collection.Databases))

	testdb := collection.Databases["testdb"]
	assert.NotNil(t, testdb)
	assert.Equal(t, int64(1), testdb.DatID)
	assert.Equal(t, "testdb", testdb.DatName)
	assert.Equal(t, int64(5), testdb.NumBackends)
	assert.Equal(t, int64(1000), testdb.XactCommit)
	assert.Equal(t, int64(10), testdb.XactRollback)
	assert.Equal(t, int64(500), testdb.BlksRead)
	assert.Equal(t, int64(4500), testdb.BlksHit)
	assert.Equal(t, int64(10000), testdb.TupReturned)
	assert.Equal(t, int64(2000), testdb.TupFetched)
	assert.Equal(t, int64(100), testdb.TupInserted)
	assert.Equal(t, int64(50), testdb.TupUpdated)
	assert.Equal(t, int64(20), testdb.TupDeleted)
	assert.Equal(t, int64(0), testdb.Conflicts)
	assert.Equal(t, int64(5), testdb.TempFiles)
	assert.Equal(t, int64(1024), testdb.TempBytes)
	assert.Equal(t, int64(1), testdb.Deadlocks)
	assert.NotNil(t, testdb.StatsReset)

	postgres := collection.Databases["postgres"]
	assert.NotNil(t, postgres)
	assert.Equal(t, int64(2), postgres.DatID)
	assert.Equal(t, "postgres", postgres.DatName)
	assert.Equal(t, int64(3), postgres.NumBackends)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapePgStatDatabase_CompatibleMode(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock failed full query
	mock.ExpectQuery("SELECT.*FROM pg_stat_database.*WHERE datname IS NOT NULL.*template0.*template1").
		WillReturnError(sql.ErrConnDone)

	// Mock table existence check
	mock.ExpectQuery("SELECT COUNT.*FROM information_schema.tables.*pg_stat_database").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(1))

	// Mock basic query for compatible mode
	mock.ExpectQuery("SELECT.*FROM pg_stat_database.*WHERE datname IS NOT NULL.*template0.*template1").
		WillReturnRows(sqlmock.NewRows([]string{
			"datid", "datname", "numbackends", "xact_commit", "xact_rollback",
			"blks_read", "blks_hit", "tup_returned", "tup_fetched", "tup_inserted",
			"tup_updated", "tup_deleted",
		}).
			AddRow(1, "testdb", 5, 1000, 10, 500, 4500, 10000, 2000, 100, 50, 20))

	// Mock optional field queries for the single database
	mock.ExpectQuery("SELECT COALESCE\\(conflicts, 0\\) FROM pg_stat_database WHERE datname = \\$1 LIMIT 1").
		WithArgs("testdb").
		WillReturnRows(sqlmock.NewRows([]string{"conflicts"}).
			AddRow(2))
	mock.ExpectQuery("SELECT COALESCE\\(temp_files, 0\\) FROM pg_stat_database WHERE datname = \\$1 LIMIT 1").
		WithArgs("testdb").
		WillReturnRows(sqlmock.NewRows([]string{"temp_files"}).
			AddRow(3))
	mock.ExpectQuery("SELECT COALESCE\\(temp_bytes, 0\\) FROM pg_stat_database WHERE datname = \\$1 LIMIT 1").
		WithArgs("testdb").
		WillReturnRows(sqlmock.NewRows([]string{"temp_bytes"}).
			AddRow(1024))
	mock.ExpectQuery("SELECT COALESCE\\(deadlocks, 0\\) FROM pg_stat_database WHERE datname = \\$1 LIMIT 1").
		WithArgs("testdb").
		WillReturnRows(sqlmock.NewRows([]string{"deadlocks"}).
			AddRow(1))

	collection, err := ScrapePgStatDatabase(db)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(collection.Databases))

	testdb := collection.Databases["testdb"]
	assert.NotNil(t, testdb)
	assert.Equal(t, "testdb", testdb.DatName)
	assert.Equal(t, int64(5), testdb.NumBackends)
	assert.Equal(t, int64(1000), testdb.XactCommit)
	assert.Equal(t, int64(2), testdb.Conflicts)
	assert.Equal(t, int64(3), testdb.TempFiles)
	assert.Equal(t, int64(1024), testdb.TempBytes)
	assert.Equal(t, int64(1), testdb.Deadlocks)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapePgStatDatabase_TableNotExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock failed full query
	mock.ExpectQuery("SELECT.*FROM pg_stat_database.*WHERE datname IS NOT NULL.*template0.*template1").
		WillReturnError(sql.ErrConnDone)

	// Mock table existence check - table doesn't exist
	mock.ExpectQuery("SELECT COUNT.*FROM information_schema.tables.*pg_stat_database").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(0))

	collection, err := ScrapePgStatDatabase(db)

	assert.NoError(t, err)
	assert.Equal(t, 0, len(collection.Databases))

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapePgStatDatabase_CompatibleModeOptionalFieldFailures(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock failed full query
	mock.ExpectQuery("SELECT.*FROM pg_stat_database.*WHERE datname IS NOT NULL.*template0.*template1").
		WillReturnError(sql.ErrConnDone)

	// Mock table existence check
	mock.ExpectQuery("SELECT COUNT.*FROM information_schema.tables.*pg_stat_database").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(1))

	// Mock basic query for compatible mode
	mock.ExpectQuery("SELECT.*FROM pg_stat_database.*WHERE datname IS NOT NULL.*template0.*template1").
		WillReturnRows(sqlmock.NewRows([]string{
			"datid", "datname", "numbackends", "xact_commit", "xact_rollback",
			"blks_read", "blks_hit", "tup_returned", "tup_fetched", "tup_inserted",
			"tup_updated", "tup_deleted",
		}).
			AddRow(1, "testdb", 5, 1000, 10, 500, 4500, 10000, 2000, 100, 50, 20))

	// Mock failed optional field queries
	mock.ExpectQuery("SELECT COALESCE\\(conflicts, 0\\) FROM pg_stat_database WHERE datname = \\$1 LIMIT 1").
		WithArgs("testdb").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("SELECT COALESCE\\(temp_files, 0\\) FROM pg_stat_database WHERE datname = \\$1 LIMIT 1").
		WithArgs("testdb").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("SELECT COALESCE\\(temp_bytes, 0\\) FROM pg_stat_database WHERE datname = \\$1 LIMIT 1").
		WithArgs("testdb").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("SELECT COALESCE\\(deadlocks, 0\\) FROM pg_stat_database WHERE datname = \\$1 LIMIT 1").
		WithArgs("testdb").
		WillReturnError(sql.ErrNoRows)

	collection, err := ScrapePgStatDatabase(db)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(collection.Databases))

	testdb := collection.Databases["testdb"]
	assert.NotNil(t, testdb)
	assert.Equal(t, "testdb", testdb.DatName)
	assert.Equal(t, int64(5), testdb.NumBackends)
	assert.Equal(t, int64(1000), testdb.XactCommit)
	// Optional fields should be 0 due to failures
	assert.Equal(t, int64(0), testdb.Conflicts)
	assert.Equal(t, int64(0), testdb.TempFiles)
	assert.Equal(t, int64(0), testdb.TempBytes)
	assert.Equal(t, int64(0), testdb.Deadlocks)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapePgStatDatabase_EmptyResults(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock successful query with empty results
	mock.ExpectQuery("SELECT.*FROM pg_stat_database.*WHERE datname IS NOT NULL.*template0.*template1").
		WillReturnRows(sqlmock.NewRows([]string{
			"datid", "datname", "numbackends", "xact_commit", "xact_rollback",
			"blks_read", "blks_hit", "tup_returned", "tup_fetched", "tup_inserted",
			"tup_updated", "tup_deleted", "conflicts", "temp_files", "temp_bytes",
			"deadlocks", "stats_reset",
		}))

	collection, err := ScrapePgStatDatabase(db)

	assert.NoError(t, err)
	assert.Equal(t, 0, len(collection.Databases))

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapePgStatDatabase_NullStatsReset(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock successful query with null stats_reset
	mock.ExpectQuery("SELECT.*FROM pg_stat_database.*WHERE datname IS NOT NULL.*template0.*template1").
		WillReturnRows(sqlmock.NewRows([]string{
			"datid", "datname", "numbackends", "xact_commit", "xact_rollback",
			"blks_read", "blks_hit", "tup_returned", "tup_fetched", "tup_inserted",
			"tup_updated", "tup_deleted", "conflicts", "temp_files", "temp_bytes",
			"deadlocks", "stats_reset",
		}).
			AddRow(1, "testdb", 5, 1000, 10, 500, 4500, 10000, 2000, 100, 50, 20, 0, 5, 1024, 1, nil))

	collection, err := ScrapePgStatDatabase(db)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(collection.Databases))

	testdb := collection.Databases["testdb"]
	assert.NotNil(t, testdb)
	assert.Equal(t, "testdb", testdb.DatName)
	assert.Nil(t, testdb.StatsReset) // Should be nil due to null value

	assert.NoError(t, mock.ExpectationsWereMet())
}

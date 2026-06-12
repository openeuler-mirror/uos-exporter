package collector

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScrapePgLocks_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock table existence check
	mock.ExpectQuery("SELECT COUNT.*FROM information_schema.tables.*pg_locks").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(1))

	// Mock locks by type query
	mock.ExpectQuery("SELECT.*locktype.*COUNT.*FROM pg_locks.*GROUP BY locktype").
		WillReturnRows(sqlmock.NewRows([]string{"locktype", "count"}).
			AddRow("relation", 10).
			AddRow("tuple", 5).
			AddRow("advisory", 2))

	// Mock locks by mode query
	mock.ExpectQuery("SELECT.*mode.*COUNT.*FROM pg_locks.*GROUP BY mode").
		WillReturnRows(sqlmock.NewRows([]string{"mode", "count"}).
			AddRow("AccessShareLock", 8).
			AddRow("RowExclusiveLock", 6).
			AddRow("ExclusiveLock", 3))

	// Mock locks by state query
	mock.ExpectQuery("SELECT.*granted.*COUNT.*FROM pg_locks.*GROUP BY granted").
		WillReturnRows(sqlmock.NewRows([]string{"state", "count"}).
			AddRow("granted", 15).
			AddRow("waiting", 2))

	// Mock locks by database query
	mock.ExpectQuery("SELECT.*datname.*COUNT.*FROM pg_locks.*LEFT JOIN pg_database.*GROUP BY.*datname").
		WillReturnRows(sqlmock.NewRows([]string{"database_name", "count"}).
			AddRow("testdb", 10).
			AddRow("postgres", 7))

	// Mock total locks count query
	mock.ExpectQuery("SELECT COUNT.*FROM pg_locks").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(17))

	locksStat, err := ScrapePgLocks(db)

	assert.NoError(t, err)
	assert.Equal(t, int64(10), locksStat.LocksByType["relation"])
	assert.Equal(t, int64(5), locksStat.LocksByType["tuple"])
	assert.Equal(t, int64(2), locksStat.LocksByType["advisory"])
	assert.Equal(t, int64(8), locksStat.LocksByMode["AccessShareLock"])
	assert.Equal(t, int64(6), locksStat.LocksByMode["RowExclusiveLock"])
	assert.Equal(t, int64(3), locksStat.LocksByMode["ExclusiveLock"])
	assert.Equal(t, int64(15), locksStat.LocksByState["granted"])
	assert.Equal(t, int64(2), locksStat.LocksByState["waiting"])
	assert.Equal(t, int64(15), locksStat.GrantedLocks)
	assert.Equal(t, int64(2), locksStat.WaitingLocks)
	assert.Equal(t, int64(10), locksStat.LocksByDatabase["testdb"])
	assert.Equal(t, int64(7), locksStat.LocksByDatabase["postgres"])
	assert.Equal(t, int64(17), locksStat.TotalLocks)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapePgLocks_TableNotExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock table existence check - table doesn't exist
	mock.ExpectQuery("SELECT COUNT.*FROM information_schema.tables.*pg_locks").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(0))

	locksStat, err := ScrapePgLocks(db)

	assert.NoError(t, err)
	assert.Equal(t, 0, len(locksStat.LocksByType))
	assert.Equal(t, 0, len(locksStat.LocksByMode))
	assert.Equal(t, 0, len(locksStat.LocksByState))
	assert.Equal(t, 0, len(locksStat.LocksByDatabase))
	assert.Equal(t, int64(0), locksStat.GrantedLocks)
	assert.Equal(t, int64(0), locksStat.WaitingLocks)
	assert.Equal(t, int64(0), locksStat.TotalLocks)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapePgLocks_LocksByTypeFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock table existence check
	mock.ExpectQuery("SELECT COUNT.*FROM information_schema.tables.*pg_locks").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(1))

	// Mock failed locks by type query
	mock.ExpectQuery("SELECT.*locktype.*COUNT.*FROM pg_locks.*GROUP BY locktype").
		WillReturnError(sql.ErrConnDone)

	locksStat, err := ScrapePgLocks(db)

	assert.NoError(t, err)
	assert.Equal(t, 0, len(locksStat.LocksByType))
	assert.Equal(t, 0, len(locksStat.LocksByMode))
	assert.Equal(t, 0, len(locksStat.LocksByState))
	assert.Equal(t, 0, len(locksStat.LocksByDatabase))
	assert.Equal(t, int64(0), locksStat.GrantedLocks)
	assert.Equal(t, int64(0), locksStat.WaitingLocks)
	assert.Equal(t, int64(0), locksStat.TotalLocks)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapePgLocks_PartialSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock table existence check
	mock.ExpectQuery("SELECT COUNT.*FROM information_schema.tables.*pg_locks").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(1))

	// Mock successful locks by type query
	mock.ExpectQuery("SELECT.*locktype.*COUNT.*FROM pg_locks.*GROUP BY locktype").
		WillReturnRows(sqlmock.NewRows([]string{"locktype", "count"}).
			AddRow("relation", 5))

	// Mock successful locks by mode query
	mock.ExpectQuery("SELECT.*mode.*COUNT.*FROM pg_locks.*GROUP BY mode").
		WillReturnRows(sqlmock.NewRows([]string{"mode", "count"}).
			AddRow("AccessShareLock", 5))

	// Mock successful locks by state query
	mock.ExpectQuery("SELECT.*granted.*COUNT.*FROM pg_locks.*GROUP BY granted").
		WillReturnRows(sqlmock.NewRows([]string{"state", "count"}).
			AddRow("granted", 5))

	// Mock failed locks by database query
	mock.ExpectQuery("SELECT.*datname.*COUNT.*FROM pg_locks.*LEFT JOIN pg_database.*GROUP BY.*datname").
		WillReturnError(sql.ErrConnDone)

	// Mock successful total locks count query
	mock.ExpectQuery("SELECT COUNT.*FROM pg_locks").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(5))

	locksStat, err := ScrapePgLocks(db)

	assert.NoError(t, err)
	assert.Equal(t, int64(5), locksStat.LocksByType["relation"])
	assert.Equal(t, int64(5), locksStat.LocksByMode["AccessShareLock"])
	assert.Equal(t, int64(5), locksStat.LocksByState["granted"])
	assert.Equal(t, int64(5), locksStat.GrantedLocks)
	assert.Equal(t, int64(0), locksStat.WaitingLocks)
	assert.Equal(t, 0, len(locksStat.LocksByDatabase)) // Should be empty due to failure
	assert.Equal(t, int64(5), locksStat.TotalLocks)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapePgLocks_EmptyResults(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock table existence check
	mock.ExpectQuery("SELECT COUNT.*FROM information_schema.tables.*pg_locks").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(1))

	// Mock empty locks by type query
	mock.ExpectQuery("SELECT.*locktype.*COUNT.*FROM pg_locks.*GROUP BY locktype").
		WillReturnRows(sqlmock.NewRows([]string{"locktype", "count"}))

	// Mock empty locks by mode query
	mock.ExpectQuery("SELECT.*mode.*COUNT.*FROM pg_locks.*GROUP BY mode").
		WillReturnRows(sqlmock.NewRows([]string{"mode", "count"}))

	// Mock empty locks by state query
	mock.ExpectQuery("SELECT.*granted.*COUNT.*FROM pg_locks.*GROUP BY granted").
		WillReturnRows(sqlmock.NewRows([]string{"state", "count"}))

	// Mock empty locks by database query
	mock.ExpectQuery("SELECT.*datname.*COUNT.*FROM pg_locks.*LEFT JOIN pg_database.*GROUP BY.*datname").
		WillReturnRows(sqlmock.NewRows([]string{"database_name", "count"}))

	// Mock zero total locks count query
	mock.ExpectQuery("SELECT COUNT.*FROM pg_locks").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(0))

	locksStat, err := ScrapePgLocks(db)

	assert.NoError(t, err)
	assert.Equal(t, 0, len(locksStat.LocksByType))
	assert.Equal(t, 0, len(locksStat.LocksByMode))
	assert.Equal(t, 0, len(locksStat.LocksByState))
	assert.Equal(t, 0, len(locksStat.LocksByDatabase))
	assert.Equal(t, int64(0), locksStat.GrantedLocks)
	assert.Equal(t, int64(0), locksStat.WaitingLocks)
	assert.Equal(t, int64(0), locksStat.TotalLocks)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapePgLocks_ScanErrors(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock table existence check
	mock.ExpectQuery("SELECT COUNT.*FROM information_schema.tables.*pg_locks").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(1))

	// Mock locks by type query with scan error (wrong column type)
	mock.ExpectQuery("SELECT.*locktype.*COUNT.*FROM pg_locks.*GROUP BY locktype").
		WillReturnRows(sqlmock.NewRows([]string{"locktype", "count"}).
			AddRow("relation", "invalid_count").
			AddRow("tuple", 5))

	// Mock locks by mode query
	mock.ExpectQuery("SELECT.*mode.*COUNT.*FROM pg_locks.*GROUP BY mode").
		WillReturnRows(sqlmock.NewRows([]string{"mode", "count"}).
			AddRow("AccessShareLock", 5))

	// Mock locks by state query
	mock.ExpectQuery("SELECT.*granted.*COUNT.*FROM pg_locks.*GROUP BY granted").
		WillReturnRows(sqlmock.NewRows([]string{"state", "count"}).
			AddRow("granted", 5))

	// Mock locks by database query
	mock.ExpectQuery("SELECT.*datname.*COUNT.*FROM pg_locks.*LEFT JOIN pg_database.*GROUP BY.*datname").
		WillReturnRows(sqlmock.NewRows([]string{"database_name", "count"}).
			AddRow("testdb", 5))

	// Mock total locks count query
	mock.ExpectQuery("SELECT COUNT.*FROM pg_locks").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(5))

	locksStat, err := ScrapePgLocks(db)

	assert.NoError(t, err)
	// The row with scan error should be skipped
	assert.Equal(t, int64(5), locksStat.LocksByType["tuple"])
	assert.Equal(t, 1, len(locksStat.LocksByType)) // Only "tuple" should be present
	assert.Equal(t, int64(5), locksStat.LocksByMode["AccessShareLock"])
	assert.Equal(t, int64(5), locksStat.LocksByState["granted"])
	assert.Equal(t, int64(5), locksStat.LocksByDatabase["testdb"])
	assert.Equal(t, int64(5), locksStat.TotalLocks)

	assert.NoError(t, mock.ExpectationsWereMet())
}

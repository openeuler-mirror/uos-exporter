package collector

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScrapePgSizeStats_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock database sizes query
	mock.ExpectQuery("SELECT.*datname.*pg_database_size.*FROM pg_database.*WHERE.*datistemplate.*false").
		WillReturnRows(sqlmock.NewRows([]string{"datname", "size"}).
			AddRow("testdb", 1073741824).  // 1GB
			AddRow("postgres", 536870912)) // 512MB

	// Mock table sizes query
	mock.ExpectQuery("SELECT.*schemaname.*tablename.*pg_relation_size.*pg_total_relation_size.*FROM pg_tables.*WHERE schemaname NOT IN").
		WillReturnRows(sqlmock.NewRows([]string{"schemaname", "tablename", "size", "total_size"}).
			AddRow("public", "users", 104857600, 125829120). // 100MB / 120MB
			AddRow("public", "orders", 52428800, 62914560).  // 50MB / 60MB
			AddRow("app", "logs", 209715200, 251658240))     // 200MB / 240MB

	// Mock tablespace sizes query
	mock.ExpectQuery("SELECT.*spcname.*pg_tablespace_size.*FROM pg_tablespace").
		WillReturnRows(sqlmock.NewRows([]string{"spcname", "size"}).
			AddRow("pg_default", 2147483648). // 2GB
			AddRow("pg_global", 104857600))   // 100MB

	sizeStats, err := ScrapePgSizeStats(db)

	assert.NoError(t, err)

	// Check database sizes
	assert.Equal(t, 2, len(sizeStats.DatabaseSizes))
	assert.Equal(t, int64(1073741824), sizeStats.DatabaseSizes["testdb"].Size)
	assert.Equal(t, "testdb", sizeStats.DatabaseSizes["testdb"].DatName)
	assert.Equal(t, int64(536870912), sizeStats.DatabaseSizes["postgres"].Size)
	assert.Equal(t, "postgres", sizeStats.DatabaseSizes["postgres"].DatName)
	assert.Equal(t, int64(1610612736), sizeStats.TotalDatabaseSize) // 1GB + 512MB

	// Check table sizes
	assert.Equal(t, 3, len(sizeStats.TableSizes))
	usersKey := "public.users"
	assert.Equal(t, "public", sizeStats.TableSizes[usersKey].SchemaName)
	assert.Equal(t, "users", sizeStats.TableSizes[usersKey].TableName)
	assert.Equal(t, int64(104857600), sizeStats.TableSizes[usersKey].Size)
	assert.Equal(t, int64(125829120), sizeStats.TableSizes[usersKey].TotalSize)

	ordersKey := "public.orders"
	assert.Equal(t, int64(52428800), sizeStats.TableSizes[ordersKey].Size)
	assert.Equal(t, int64(62914560), sizeStats.TableSizes[ordersKey].TotalSize)

	logsKey := "app.logs"
	assert.Equal(t, int64(209715200), sizeStats.TableSizes[logsKey].Size)
	assert.Equal(t, int64(251658240), sizeStats.TableSizes[logsKey].TotalSize)

	expectedTotalTableSize := int64(125829120 + 62914560 + 251658240)
	assert.Equal(t, expectedTotalTableSize, sizeStats.TotalTableSize)

	// Check tablespace sizes
	assert.Equal(t, 2, len(sizeStats.TablespaceSizes))
	assert.Equal(t, int64(2147483648), sizeStats.TablespaceSizes["pg_default"].Size)
	assert.Equal(t, "pg_default", sizeStats.TablespaceSizes["pg_default"].TablespaceName)
	assert.Equal(t, int64(104857600), sizeStats.TablespaceSizes["pg_global"].Size)
	assert.Equal(t, "pg_global", sizeStats.TablespaceSizes["pg_global"].TablespaceName)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapePgSizeStats_DatabaseSizeFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock failed database sizes query
	mock.ExpectQuery("SELECT.*datname.*pg_database_size.*FROM pg_database.*WHERE.*datistemplate.*false").
		WillReturnError(sql.ErrConnDone)

	sizeStats, err := ScrapePgSizeStats(db)

	assert.NoError(t, err)
	assert.Equal(t, 0, len(sizeStats.DatabaseSizes))
	assert.Equal(t, 0, len(sizeStats.TableSizes))
	assert.Equal(t, 0, len(sizeStats.TablespaceSizes))
	assert.Equal(t, int64(0), sizeStats.TotalDatabaseSize)
	assert.Equal(t, int64(0), sizeStats.TotalTableSize)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapePgSizeStats_PartialFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock successful database sizes query
	mock.ExpectQuery("SELECT.*datname.*pg_database_size.*FROM pg_database.*WHERE.*datistemplate.*false").
		WillReturnRows(sqlmock.NewRows([]string{"datname", "size"}).
			AddRow("testdb", 1073741824))

	// Mock failed table sizes query
	mock.ExpectQuery("SELECT.*schemaname.*tablename.*pg_relation_size.*pg_total_relation_size.*FROM pg_tables.*WHERE schemaname NOT IN").
		WillReturnError(sql.ErrConnDone)

	// Mock failed tablespace sizes query
	mock.ExpectQuery("SELECT.*spcname.*pg_tablespace_size.*FROM pg_tablespace").
		WillReturnError(sql.ErrConnDone)

	sizeStats, err := ScrapePgSizeStats(db)

	assert.NoError(t, err)

	// Database sizes should be collected successfully
	assert.Equal(t, 1, len(sizeStats.DatabaseSizes))
	assert.Equal(t, int64(1073741824), sizeStats.DatabaseSizes["testdb"].Size)
	assert.Equal(t, int64(1073741824), sizeStats.TotalDatabaseSize)

	// Table and tablespace sizes should be empty due to failures
	assert.Equal(t, 0, len(sizeStats.TableSizes))
	assert.Equal(t, 0, len(sizeStats.TablespaceSizes))
	assert.Equal(t, int64(0), sizeStats.TotalTableSize)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapePgSizeStats_EmptyResults(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock empty database sizes query
	mock.ExpectQuery("SELECT.*datname.*pg_database_size.*FROM pg_database.*WHERE.*datistemplate.*false").
		WillReturnRows(sqlmock.NewRows([]string{"datname", "size"}))

	// Mock empty table sizes query
	mock.ExpectQuery("SELECT.*schemaname.*tablename.*pg_relation_size.*pg_total_relation_size.*FROM pg_tables.*WHERE schemaname NOT IN").
		WillReturnRows(sqlmock.NewRows([]string{"schemaname", "tablename", "size", "total_size"}))

	// Mock empty tablespace sizes query
	mock.ExpectQuery("SELECT.*spcname.*pg_tablespace_size.*FROM pg_tablespace").
		WillReturnRows(sqlmock.NewRows([]string{"spcname", "size"}))

	sizeStats, err := ScrapePgSizeStats(db)

	assert.NoError(t, err)
	assert.Equal(t, 0, len(sizeStats.DatabaseSizes))
	assert.Equal(t, 0, len(sizeStats.TableSizes))
	assert.Equal(t, 0, len(sizeStats.TablespaceSizes))
	assert.Equal(t, int64(0), sizeStats.TotalDatabaseSize)
	assert.Equal(t, int64(0), sizeStats.TotalTableSize)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapePgSizeStats_ScanErrors(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock database sizes query with scan error
	mock.ExpectQuery("SELECT.*datname.*pg_database_size.*FROM pg_database.*WHERE.*datistemplate.*false").
		WillReturnRows(sqlmock.NewRows([]string{"datname", "size"}).
			AddRow("testdb", "invalid_size"). // Invalid size type
			AddRow("postgres", 536870912))    // Valid row

	// Mock table sizes query with scan error
	mock.ExpectQuery("SELECT.*schemaname.*tablename.*pg_relation_size.*pg_total_relation_size.*FROM pg_tables.*WHERE schemaname NOT IN").
		WillReturnRows(sqlmock.NewRows([]string{"schemaname", "tablename", "size", "total_size"}).
			AddRow("public", "users", "invalid", "invalid"). // Invalid row
			AddRow("public", "orders", 52428800, 62914560))  // Valid row

	// Mock tablespace sizes query with scan error
	mock.ExpectQuery("SELECT.*spcname.*pg_tablespace_size.*FROM pg_tablespace").
		WillReturnRows(sqlmock.NewRows([]string{"spcname", "size"}).
			AddRow("pg_default", "invalid"). // Invalid size
			AddRow("pg_global", 104857600))  // Valid row

	sizeStats, err := ScrapePgSizeStats(db)

	assert.NoError(t, err)

	// Only valid rows should be processed
	assert.Equal(t, 1, len(sizeStats.DatabaseSizes))
	assert.Equal(t, int64(536870912), sizeStats.DatabaseSizes["postgres"].Size)
	assert.Equal(t, int64(536870912), sizeStats.TotalDatabaseSize)

	assert.Equal(t, 1, len(sizeStats.TableSizes))
	ordersKey := "public.orders"
	assert.Equal(t, int64(52428800), sizeStats.TableSizes[ordersKey].Size)
	assert.Equal(t, int64(62914560), sizeStats.TableSizes[ordersKey].TotalSize)
	assert.Equal(t, int64(62914560), sizeStats.TotalTableSize)

	assert.Equal(t, 1, len(sizeStats.TablespaceSizes))
	assert.Equal(t, int64(104857600), sizeStats.TablespaceSizes["pg_global"].Size)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapePgSizeStats_TablespaceSizeNotSupported(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock successful database sizes query
	mock.ExpectQuery("SELECT.*datname.*pg_database_size.*FROM pg_database.*WHERE.*datistemplate.*false").
		WillReturnRows(sqlmock.NewRows([]string{"datname", "size"}).
			AddRow("testdb", 1073741824))

	// Mock successful table sizes query
	mock.ExpectQuery("SELECT.*schemaname.*tablename.*pg_relation_size.*pg_total_relation_size.*FROM pg_tables.*WHERE schemaname NOT IN").
		WillReturnRows(sqlmock.NewRows([]string{"schemaname", "tablename", "size", "total_size"}).
			AddRow("public", "users", 104857600, 125829120))

	// Mock tablespace sizes query failure (not supported)
	mock.ExpectQuery("SELECT.*spcname.*pg_tablespace_size.*FROM pg_tablespace").
		WillReturnError(sql.ErrConnDone)

	sizeStats, err := ScrapePgSizeStats(db)

	assert.NoError(t, err)

	// Database and table sizes should be collected successfully
	assert.Equal(t, 1, len(sizeStats.DatabaseSizes))
	assert.Equal(t, int64(1073741824), sizeStats.TotalDatabaseSize)

	assert.Equal(t, 1, len(sizeStats.TableSizes))
	assert.Equal(t, int64(125829120), sizeStats.TotalTableSize)

	// Tablespace sizes should be empty due to failure (gracefully handled)
	assert.Equal(t, 0, len(sizeStats.TablespaceSizes))

	assert.NoError(t, mock.ExpectationsWereMet())
}

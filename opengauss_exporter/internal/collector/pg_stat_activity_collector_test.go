package collector

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScrapePgStatActivity_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock connections by state query
	mock.ExpectQuery("SELECT.*state.*COUNT.*FROM pg_stat_activity.*GROUP BY state").
		WillReturnRows(sqlmock.NewRows([]string{"state", "count"}).
			AddRow("active", 5).
			AddRow("idle", 3).
			AddRow("idle in transaction", 2).
			AddRow("waiting", 1))

	// Mock connections by database query
	mock.ExpectQuery("SELECT.*datname.*COUNT.*FROM pg_stat_activity.*GROUP BY datname").
		WillReturnRows(sqlmock.NewRows([]string{"database_name", "count"}).
			AddRow("testdb", 6).
			AddRow("postgres", 5))

	// Mock connections by user query
	mock.ExpectQuery("SELECT.*usename.*COUNT.*FROM pg_stat_activity.*GROUP BY usename").
		WillReturnRows(sqlmock.NewRows([]string{"username", "count"}).
			AddRow("user1", 7).
			AddRow("user2", 4))

	// Mock wait event type column check
	mock.ExpectQuery("SELECT COUNT.*FROM information_schema.columns.*wait_event_type").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(1))

	// Mock wait event stats query
	mock.ExpectQuery("SELECT.*wait_event_type.*COUNT.*FROM pg_stat_activity.*GROUP BY wait_event_type").
		WillReturnRows(sqlmock.NewRows([]string{"wait_event_type", "count"}).
			AddRow("Lock", 2).
			AddRow("IO", 1).
			AddRow("none", 8))

	// Mock long running queries count
	mock.ExpectQuery("SELECT COUNT.*FROM pg_stat_activity.*query_start.*5 minute").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(1))

	// Mock oldest query duration
	mock.ExpectQuery("SELECT COALESCE.*EXTRACT.*EPOCH.*MIN.*query_start").
		WillReturnRows(sqlmock.NewRows([]string{"duration"}).
			AddRow(300.5))

	// Mock total connections count
	mock.ExpectQuery("SELECT COUNT.*FROM pg_stat_activity").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(11))

	activity, err := ScrapePgStatActivity(db)

	assert.NoError(t, err)
	assert.Equal(t, int64(5), activity.ActiveConnections)
	assert.Equal(t, int64(3), activity.IdleConnections)
	assert.Equal(t, int64(2), activity.IdleInTransactionConnections)
	assert.Equal(t, int64(1), activity.WaitingConnections)
	assert.Equal(t, int64(6), activity.ConnectionsByDatabase["testdb"])
	assert.Equal(t, int64(5), activity.ConnectionsByDatabase["postgres"])
	assert.Equal(t, int64(7), activity.ConnectionsByUser["user1"])
	assert.Equal(t, int64(4), activity.ConnectionsByUser["user2"])
	assert.Equal(t, int64(2), activity.WaitEventStats["Lock"])
	assert.Equal(t, int64(1), activity.WaitEventStats["IO"])
	assert.Equal(t, int64(8), activity.WaitEventStats["none"])
	assert.Equal(t, int64(1), activity.LongRunningQueries)
	assert.Equal(t, 300.5, activity.OldestQueryDuration)
	assert.Equal(t, int64(11), activity.TotalConnections)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapePgStatActivity_NoWaitEventTypeColumn(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock connections by state query
	mock.ExpectQuery("SELECT.*state.*COUNT.*FROM pg_stat_activity.*GROUP BY state").
		WillReturnRows(sqlmock.NewRows([]string{"state", "count"}).
			AddRow("active", 2).
			AddRow("idle", 1))

	// Mock connections by database query
	mock.ExpectQuery("SELECT.*datname.*COUNT.*FROM pg_stat_activity.*GROUP BY datname").
		WillReturnRows(sqlmock.NewRows([]string{"database_name", "count"}).
			AddRow("testdb", 3))

	// Mock connections by user query
	mock.ExpectQuery("SELECT.*usename.*COUNT.*FROM pg_stat_activity.*GROUP BY usename").
		WillReturnRows(sqlmock.NewRows([]string{"username", "count"}).
			AddRow("user1", 3))

	// Mock wait event type column check - column doesn't exist
	mock.ExpectQuery("SELECT COUNT.*FROM information_schema.columns.*wait_event_type").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(0))

	// Mock long running queries count
	mock.ExpectQuery("SELECT COUNT.*FROM pg_stat_activity.*query_start.*5 minute").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(0))

	// Mock oldest query duration
	mock.ExpectQuery("SELECT COALESCE.*EXTRACT.*EPOCH.*MIN.*query_start").
		WillReturnRows(sqlmock.NewRows([]string{"duration"}).
			AddRow(0))

	// Mock total connections count
	mock.ExpectQuery("SELECT COUNT.*FROM pg_stat_activity").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(3))

	activity, err := ScrapePgStatActivity(db)

	assert.NoError(t, err)
	assert.Equal(t, int64(2), activity.ActiveConnections)
	assert.Equal(t, int64(1), activity.IdleConnections)
	assert.Equal(t, int64(3), activity.ConnectionsByDatabase["testdb"])
	assert.Equal(t, int64(3), activity.ConnectionsByUser["user1"])
	// Wait event stats should have default 'none' entry
	assert.Equal(t, int64(0), activity.WaitEventStats["none"])
	assert.Equal(t, int64(0), activity.LongRunningQueries)
	assert.Equal(t, float64(0), activity.OldestQueryDuration)
	assert.Equal(t, int64(3), activity.TotalConnections)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapePgStatActivity_QueryFailures(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock failed connections by state query
	mock.ExpectQuery("SELECT.*state.*COUNT.*FROM pg_stat_activity.*GROUP BY state").
		WillReturnError(sql.ErrConnDone)

	activity, err := ScrapePgStatActivity(db)

	assert.Error(t, err)
	assert.NotNil(t, activity)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapePgStatActivity_EmptyResults(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock empty connections by state query
	mock.ExpectQuery("SELECT.*state.*COUNT.*FROM pg_stat_activity.*GROUP BY state").
		WillReturnRows(sqlmock.NewRows([]string{"state", "count"}))

	// Mock empty connections by database query
	mock.ExpectQuery("SELECT.*datname.*COUNT.*FROM pg_stat_activity.*GROUP BY datname").
		WillReturnRows(sqlmock.NewRows([]string{"database_name", "count"}))

	// Mock empty connections by user query
	mock.ExpectQuery("SELECT.*usename.*COUNT.*FROM pg_stat_activity.*GROUP BY usename").
		WillReturnRows(sqlmock.NewRows([]string{"username", "count"}))

	// Mock wait event type column check
	mock.ExpectQuery("SELECT COUNT.*FROM information_schema.columns.*wait_event_type").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(1))

	// Mock empty wait event stats query
	mock.ExpectQuery("SELECT.*wait_event_type.*COUNT.*FROM pg_stat_activity.*GROUP BY wait_event_type").
		WillReturnRows(sqlmock.NewRows([]string{"wait_event_type", "count"}))

	// Mock long running queries count
	mock.ExpectQuery("SELECT COUNT.*FROM pg_stat_activity.*query_start.*5 minute").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(0))

	// Mock oldest query duration
	mock.ExpectQuery("SELECT COALESCE.*EXTRACT.*EPOCH.*MIN.*query_start").
		WillReturnRows(sqlmock.NewRows([]string{"duration"}).
			AddRow(0))

	// Mock total connections count
	mock.ExpectQuery("SELECT COUNT.*FROM pg_stat_activity").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(0))

	activity, err := ScrapePgStatActivity(db)

	assert.NoError(t, err)
	assert.Equal(t, int64(0), activity.ActiveConnections)
	assert.Equal(t, int64(0), activity.IdleConnections)
	assert.Equal(t, int64(0), activity.IdleInTransactionConnections)
	assert.Equal(t, int64(0), activity.WaitingConnections)
	assert.Equal(t, int64(0), activity.OtherConnections)
	assert.Equal(t, 0, len(activity.ConnectionsByDatabase))
	assert.Equal(t, 0, len(activity.ConnectionsByUser))
	assert.Equal(t, 0, len(activity.WaitEventStats))
	assert.Equal(t, int64(0), activity.LongRunningQueries)
	assert.Equal(t, float64(0), activity.OldestQueryDuration)
	assert.Equal(t, int64(0), activity.TotalConnections)

	assert.NoError(t, mock.ExpectationsWereMet())
}

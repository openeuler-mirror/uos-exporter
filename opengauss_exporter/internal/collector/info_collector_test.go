package collector

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScrapeOpenGaussInfo_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock version query
	mock.ExpectQuery("SELECT version\\(\\);").
		WillReturnRows(sqlmock.NewRows([]string{"version"}).
			AddRow("PostgreSQL 14.2 on x86_64-pc-linux-gnu"))

	// Mock database count query
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM pg_database WHERE NOT datistemplate;").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(3))

	// Mock current connections query
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM pg_stat_activity;").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(10))

	// Mock max connections query
	mock.ExpectQuery("SHOW max_connections;").
		WillReturnRows(sqlmock.NewRows([]string{"max_connections"}).
			AddRow(100))

	// Mock backend states query
	mock.ExpectQuery("SELECT.*FROM pg_stat_activity.*GROUP BY state").
		WillReturnRows(sqlmock.NewRows([]string{"state", "count"}).
			AddRow("active", 5).
			AddRow("idle", 3).
			AddRow("waiting", 2))

	// Mock postmaster start time query
	startTime := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	mock.ExpectQuery("SELECT pg_postmaster_start_time\\(\\);").
		WillReturnRows(sqlmock.NewRows([]string{"pg_postmaster_start_time"}).
			AddRow(startTime))

	info, err := ScrapeOpenGaussInfo(db)

	assert.NoError(t, err)
	assert.True(t, info.Up)
	assert.Equal(t, "14.2", info.Version)
	assert.Equal(t, int64(3), info.DatabaseCount)
	assert.Equal(t, int64(10), info.ConnectionCurrent)
	assert.Equal(t, int64(100), info.ConnectionMax)
	assert.Equal(t, int64(5), info.ActiveBackends)
	assert.Equal(t, int64(3), info.IdleBackends)
	assert.Equal(t, int64(2), info.WaitingBackends)
	assert.Greater(t, info.UptimeSeconds, float64(0))

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapeOpenGaussInfo_DatabaseDown(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer db.Close()

	// Mock ping failure
	mock.ExpectPing().WillReturnError(sql.ErrConnDone)

	info, err := ScrapeOpenGaussInfo(db)

	assert.Error(t, err)
	assert.False(t, info.Up)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapeOpenGaussInfo_PartialFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock successful version query
	mock.ExpectQuery("SELECT version\\(\\);").
		WillReturnRows(sqlmock.NewRows([]string{"version"}).
			AddRow("OpenGauss 3.0.0"))

	// Mock failed database count query
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM pg_database WHERE NOT datistemplate;").
		WillReturnError(sql.ErrNoRows)

	// Mock successful current connections query
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM pg_stat_activity;").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(5))

	// Mock failed max connections query
	mock.ExpectQuery("SHOW max_connections;").
		WillReturnError(sql.ErrNoRows)

	// Mock successful backend states query
	mock.ExpectQuery("SELECT.*FROM pg_stat_activity.*GROUP BY state").
		WillReturnRows(sqlmock.NewRows([]string{"state", "count"}).
			AddRow("active", 3).
			AddRow("idle", 2))

	// Mock failed postmaster start time query
	mock.ExpectQuery("SELECT pg_postmaster_start_time\\(\\);").
		WillReturnError(sql.ErrNoRows)

	info, err := ScrapeOpenGaussInfo(db)

	assert.NoError(t, err)
	assert.True(t, info.Up)
	assert.Equal(t, "3.0.0", info.Version)
	assert.Equal(t, int64(0), info.DatabaseCount) // Should be 0 due to failure
	assert.Equal(t, int64(5), info.ConnectionCurrent)
	assert.Equal(t, int64(0), info.ConnectionMax) // Should be 0 due to failure
	assert.Equal(t, int64(3), info.ActiveBackends)
	assert.Equal(t, int64(2), info.IdleBackends)
	assert.Equal(t, float64(0), info.UptimeSeconds) // Should be 0 due to failure

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestParseVersion(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"PostgreSQL 14.2 on x86_64-pc-linux-gnu", "14.2"},
		{"OpenGauss 3.0.0 compiled at 2022-01-01", "3.0.0"},
		{"SingleVersionString", "SingleVersionString"},
		{"", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := parseVersion(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestScrapeOpenGaussInfo_NullValues(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock version query with null
	mock.ExpectQuery("SELECT version\\(\\);").
		WillReturnRows(sqlmock.NewRows([]string{"version"}).
			AddRow(nil))

	// Mock database count query with null
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM pg_database WHERE NOT datistemplate;").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(nil))

	// Mock current connections query with null
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM pg_stat_activity;").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).
			AddRow(nil))

	// Mock max connections query with null
	mock.ExpectQuery("SHOW max_connections;").
		WillReturnRows(sqlmock.NewRows([]string{"max_connections"}).
			AddRow(nil))

	// Mock empty backend states query
	mock.ExpectQuery("SELECT.*FROM pg_stat_activity.*GROUP BY state").
		WillReturnRows(sqlmock.NewRows([]string{"state", "count"}))

	// Mock postmaster start time query with null
	mock.ExpectQuery("SELECT pg_postmaster_start_time\\(\\);").
		WillReturnRows(sqlmock.NewRows([]string{"pg_postmaster_start_time"}).
			AddRow(nil))

	info, err := ScrapeOpenGaussInfo(db)

	assert.NoError(t, err)
	assert.True(t, info.Up)
	assert.Equal(t, "", info.Version) // Should handle null version
	assert.Equal(t, int64(0), info.DatabaseCount)
	assert.Equal(t, int64(0), info.ConnectionCurrent)
	assert.Equal(t, int64(0), info.ConnectionMax)
	assert.Equal(t, int64(0), info.ActiveBackends)
	assert.Equal(t, int64(0), info.IdleBackends)
	assert.Equal(t, int64(0), info.WaitingBackends)
	assert.Equal(t, float64(0), info.UptimeSeconds)

	assert.NoError(t, mock.ExpectationsWereMet())
}

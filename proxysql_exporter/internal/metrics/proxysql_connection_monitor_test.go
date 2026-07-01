package metrics_test

import (
	"fmt"
	"time"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	initTestFunc "proxysql_exporter/internal/metrics" 
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestScrapeMySQLConnectionList_Success(t *testing.T) {

    db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
    require.NoError(t, err)
    defer db.Close()

    rows := sqlmock.NewRows([]string{"connection_count", "cli_host"}).
        AddRow(5.0, "host1").
        AddRow(3.0, "host2")
    
    exactSQL := `SELECT 
		COUNT(cli_host) as connection_count, cli_host 
		FROM stats_mysql_processlist 
		GROUP BY cli_host`
    mock.ExpectQuery(exactSQL).WillReturnRows(rows)

	ch := make(chan prometheus.Metric, 3) 
	err = initTestFunc.ScrapeMySQLConnectionList(db, ch)
	close(ch)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())

	
	var metrics []prometheus.Metric
	for m := range ch {
		metrics = append(metrics, m)
	}
	require.Len(t, metrics, 3, "Expected 3 metrics")

	expectedCounts := map[string]float64{
		"host1": 5.0,
		"host2": 3.0,
	}
	for _, m := range metrics[:2] {
		metric := &dto.Metric{}
		require.NoError(t, m.Write(metric))

		labels := metric.GetLabel()
		require.Len(t, labels, 1, "Expected one label")
		require.Equal(t, "client_host", labels[0].GetName())

		host := labels[0].GetValue()
		expected, ok := expectedCounts[host]
		require.True(t, ok, "Unexpected host %s", host)
		require.Equal(t, expected, metric.GetGauge().GetValue())
	}

	durationMetric := metrics[2]
	metric := &dto.Metric{}
	require.NoError(t, durationMetric.Write(metric))

	expectedDesc := prometheus.NewDesc(
		prometheus.BuildFQName("proxysql", "processlist", "scrape_duration_seconds"),
		"Time spent scraping connection data",
		nil, nil,
	)
	require.Equal(t, expectedDesc.String(), durationMetric.Desc().String())
	require.True(t, metric.GetGauge().GetValue() >= 0, "Expected non-negative duration")
}

func TestScrapeMySQLConnectionList_QueryError(t *testing.T) {
    db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
    require.NoError(t, err)
    defer db.Close()

    exactSQL := `SELECT 
		COUNT(cli_host) as connection_count, cli_host 
		FROM stats_mysql_processlist 
		GROUP BY cli_host`
    
    for i := 0; i < 3; i++ {
        mock.ExpectQuery(exactSQL).WillReturnError(fmt.Errorf("mock query error"))
    }

    ch := make(chan prometheus.Metric)
    err = initTestFunc.ScrapeMySQLConnectionList(db, ch)

    require.Error(t, err)
    require.Contains(t, err.Error(), "query failed after 3 retries")
    require.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapeMySQLConnectionList_InvalidColumns(t *testing.T) {
    db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
    require.NoError(t, err)
    defer db.Close()

    exactSQL := `SELECT 
		COUNT(cli_host) as connection_count, cli_host 
		FROM stats_mysql_processlist 
		GROUP BY cli_host`
    
    rows := sqlmock.NewRows([]string{"a", "b", "c"}).AddRow(1, "host", "extra")
    mock.ExpectQuery(exactSQL).WillReturnRows(rows)

    ch := make(chan prometheus.Metric)
    err = initTestFunc.ScrapeMySQLConnectionList(db, ch)

    require.Error(t, err)
    require.Contains(t, err.Error(), "invalid column count") 
    require.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapeMySQLConnectionList_EmptyHost(t *testing.T) {
    db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
    require.NoError(t, err)
    defer db.Close()

    exactSQL := strings.TrimSpace(`
        SELECT COUNT(cli_host) as connection_count, cli_host 
        FROM stats_mysql_processlist 
        GROUP BY cli_host
    `)

    rows := sqlmock.NewRows([]string{"connection_count", "cli_host"}).AddRow(2.0, "")
    mock.ExpectQuery(exactSQL).WillReturnRows(rows)

    ch := make(chan prometheus.Metric)
    err = initTestFunc.ScrapeMySQLConnectionList(db, ch)

    require.Error(t, err)
    require.Contains(t, err.Error(), "empty client host")
    require.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapeMySQLConnectionList_ScanError(t *testing.T) {
    db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
    require.NoError(t, err)
    defer db.Close()

    exactSQL := strings.ReplaceAll(`
        SELECT COUNT(cli_host) as connection_count, cli_host 
        FROM stats_mysql_processlist 
        GROUP BY cli_host
    `, "\n", " ")

    rows := sqlmock.NewRows([]string{"connection_count", "cli_host"}).
        AddRow("invalid", "host").
        RowError(0, errors.New("row scan failed"))
    mock.ExpectQuery(exactSQL).WillReturnRows(rows)

    ch := make(chan prometheus.Metric)
    err = initTestFunc.ScrapeMySQLConnectionList(db, ch)

    require.Error(t, err)
    require.Contains(t, err.Error(), "row scan failed")
    require.NoError(t, mock.ExpectationsWereMet())
}

func TestScrapeMySQLConnectionList_ContextTimeout(t *testing.T) {
    db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
    require.NoError(t, err)
    defer db.Close()

    exactSQL := strings.ReplaceAll(`
        SELECT COUNT(cli_host) as connection_count, 
        cli_host 
        FROM stats_mysql_processlist 
        GROUP BY cli_host
    `, "\n", " ")

    for i := 0; i < 3; i++ {
        mock.ExpectQuery(exactSQL).
            WillDelayFor(10 * time.Millisecond).
            WillReturnError(context.DeadlineExceeded)
    }

    ch := make(chan prometheus.Metric)
    err = initTestFunc.ScrapeMySQLConnectionList(db, ch)

    require.Error(t, err)
    require.Contains(t, err.Error(), "context deadline exceeded")
    require.NoError(t, mock.ExpectationsWereMet())
}

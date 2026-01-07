
package metrics_test

import (
	"testing"
	"errors"

	"github.com/DATA-DOG/go-sqlmock"
	initTestFunc "proxysql_exporter/internal/metrics" 
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestScrapeMySQLConnectionPool(t *testing.T) {
	t.Run("normal_case", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		ch := make(chan prometheus.Metric, 100)
		defer close(ch)

		rows := sqlmock.NewRows([]string{
			"hostgroup", 
			"srv_host", 
			"srv_port", 
			"status",
			"ConnUsed", 
			"ConnFree", 
			"ConnOK", 
			"ConnERR",
			"Queries", 
			"Bytes_data_sent", 
			"Bytes_data_recv", 
			"Latency_us",
		}).AddRow(
			"hg1", 
			"host1", 
			"3306", 
			"ONLINE",
			"10", 
			"5", 
			"100", 
			"2",
			"500", 
			"1024", 
			"2048", 
			"150",
		)

		mock.ExpectQuery(initTestFunc.ConnectionPoolQuery).WillReturnRows(rows)
		
		err = initTestFunc.ScrapeMySQLConnectionPool(db, ch)
		require.NoError(t, err)
		require.Equal(t, 10, len(ch)) // 9 metrics + duration
	})

	t.Run("query_error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		ch := make(chan prometheus.Metric, 10)
		defer close(ch)

		mock.ExpectQuery(initTestFunc.ConnectionPoolQuery).
			WillReturnError(errors.New("connection failed"))
		
		err = initTestFunc.ScrapeMySQLConnectionPool(db, ch)
		require.Error(t, err)
		require.Contains(t, err.Error(), "query failed")
	})

	t.Run("invalid_status_value", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		ch := make(chan prometheus.Metric, 10)
		defer close(ch)

		rows := sqlmock.NewRows([]string{
			"hostgroup", 
			"srv_host", 
			"srv_port", 
			"status",
			"ConnUsed", 
			"ConnFree", 
			"ConnOK", 
			"ConnERR",
			"Queries", 
			"Bytes_data_sent", 
			"Bytes_data_recv", 
			"Latency_us",
		}).AddRow(
			"hg1", 
			"host1", 
			"3306", 
			"INVALID_STATUS",
			"10", 
			"5", 
			"0", 
			"0", 
			"0", 
			"0", 
			"0", 
			"0", 
		)

		mock.ExpectQuery(initTestFunc.ConnectionPoolQuery).WillReturnRows(rows)
		err = initTestFunc.ScrapeMySQLConnectionPool(db, ch)
		require.Error(t, err)
		require.Contains(t, err.Error(), "parse status failed")
	})

	t.Run("numeric_parse_error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		ch := make(chan prometheus.Metric, 10)
		defer close(ch)

		rows := sqlmock.NewRows([]string{
			"hostgroup", 
			"srv_host", 
			"srv_port", 
			"status",
			"ConnUsed", 
			"ConnFree",
			"ConnOK", 
			"ConnERR",
			"Queries", 
			"Bytes_data_sent", 
			"Bytes_data_recv", 
			"Latency_us",
		}).AddRow(
			"hg1", 
			"host1", 
			"3306", 
			"ONLINE",
			"invalid", 
			"5", 
			"0", 
			"0", 
			"0", 
			"0", 
			"0", 
			"0",
		)

		mock.ExpectQuery(initTestFunc.ConnectionPoolQuery).WillReturnRows(rows)
		err = initTestFunc.ScrapeMySQLConnectionPool(db, ch)
		require.Error(t, err)
		require.Contains(t, err.Error(), "parse connused failed")
	})

	t.Run("empty_result_set", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		ch := make(chan prometheus.Metric, 10)
		defer close(ch)

		rows := sqlmock.NewRows([]string{
			"hostgroup", 
			"srv_host", 
			"srv_port", 
			"status",
			"ConnUsed", 
			"ConnFree",
		})

		mock.ExpectQuery(initTestFunc.ConnectionPoolQuery).WillReturnRows(rows)
		err = initTestFunc.ScrapeMySQLConnectionPool(db, ch)
		require.NoError(t, err)
		require.Equal(t, 1, len(ch)) // only duration metric
	})

	t.Run("row_scan_error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		ch := make(chan prometheus.Metric, 10)
		defer close(ch)

		rows := sqlmock.NewRows([]string{
			"hostgroup", 
			"srv_host", 
			"srv_port", 
			"status",
		}).AddRow(
			"hg1", 
		    "host1", 
			"3306", 
			"ONLINE").
			RowError(0, errors.New("scan error"))

		mock.ExpectQuery(initTestFunc.ConnectionPoolQuery).WillReturnRows(rows)
		err = initTestFunc.ScrapeMySQLConnectionPool(db, ch)
		require.Error(t, err)
		require.Contains(t, err.Error(), "scan error")
	})

	t.Run("columns_mismatch", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		ch := make(chan prometheus.Metric, 10)
		defer close(ch)

		rows := sqlmock.NewRows([]string{
			"hostgroup", 
			"srv_host", // missing columns
		}).AddRow(
			"hg1", 
			"host1",
		)

		mock.ExpectQuery(initTestFunc.ConnectionPoolQuery).WillReturnRows(rows)
		err = initTestFunc.ScrapeMySQLConnectionPool(db, ch)
		require.Error(t, err)
		require.Contains(t, err.Error(), "column count mismatch")
	})

	t.Run("metric_generation", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		ch := make(chan prometheus.Metric, 100)
		defer close(ch)

		rows := sqlmock.NewRows([]string{
			"hostgroup", 
			"srv_host", 
			"srv_port", 
			"status",
			"ConnUsed", 
			"ConnFree", 
			"ConnOK", 
			"ConnERR",
			"Queries", 
			"Bytes_data_sent", 
			"Bytes_data_recv", 
			"Latency_us",
		}).AddRow(
			"hg1", 
			"host1", 
			"3306", 
			"ONLINE",
			"10", 
			"5", 
			"100", 
			"2",
			"500", 
			"1024", 
			"2048", 
			"150",
		)

		mock.ExpectQuery(initTestFunc.ConnectionPoolQuery).WillReturnRows(rows)
		err = initTestFunc.ScrapeMySQLConnectionPool(db, ch)
		require.NoError(t, err)

		var metrics []prometheus.Metric
		for len(ch) > 0 {
			metrics = append(metrics, <-ch)
		}

		require.Equal(t, 10, len(metrics))
	})
}

func TestPoolCollector(t *testing.T) {
	collector := initTestFunc.NewPoolCollector()

	t.Run("build_metric_desc", func(t *testing.T) {
		desc := collector.BuildMetricDesc("status")
		require.Contains(t, desc.String(), "connection_pool_status")
	})

	t.Run("parse_valid_status", func(t *testing.T) {
		val, err := collector.ParseStatus("ONLINE")
		require.NoError(t, err)
		require.Equal(t, float64(1), val)
	})

	t.Run("parse_invalid_status", func(t *testing.T) {
		_, err := collector.ParseStatus("UNKNOWN")
		require.Error(t, err)
	})

	t.Run("process_row", func(t *testing.T) {
		scan := make([]interface{}, 12)
		for i := range scan {
			str := ""
			switch i {
			case 0: str = "hg1"
			case 1: str = "host1"
			case 2: str = "3306"
			case 3: str = "ONLINE"
			case 4: str = "10"
			case 5: str = "5"
			default: str = "0"
			}
			scan[i] = &str
		}

		record, err := collector.ProcessRow(scan)
		require.NoError(t, err)
		require.Equal(t, "hg1", record.Hostgroup)
		require.Equal(t, float64(10), record.Metrics["connused"])
	})
}

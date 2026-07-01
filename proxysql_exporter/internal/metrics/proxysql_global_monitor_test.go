
package metrics_test

import (
    "errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	initTestFunc "proxysql_exporter/internal/metrics" 
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestNewMetricCollectorMonitor(t *testing.T) {
	collector := initTestFunc.NewMetricCollectorMonitor()

	require.NotNil(t, collector)
    require.Len(t, collector.MetricsRegistry, 8)
    require.Equal(t, "active_transactions", 
        collector.MetricsRegistry["active_transactions"].Name)
    require.Equal(t, prometheus.GaugeValue,
        collector.MetricsRegistry["active_transactions"].ValueType)
}

func TestBuildMetricDesc(t *testing.T) {
	collector := initTestFunc.NewMetricCollectorMonitor()
	
	t.Run("known metric", func(t *testing.T) {
		desc := collector.BuildMetricDesc("active_transactions")
		require.Contains(t, desc.String(), "mysql_status_active_transactions")
	})

	t.Run("unknown metric", func(t *testing.T) {
		desc := collector.BuildMetricDesc("unknown_metric")
		require.Contains(t, desc.String(), "mysql_status_unknown_metric")
	})
}

func TestParseMetricValue(t *testing.T) {
	collector := initTestFunc.NewMetricCollectorMonitor()
	
	tests := []struct {
		name     string
		input    string
		expected float64
		err      bool
	}{
		{"valid integer", "123", 123, false},
		{"valid float", "12.34", 12.34, false},
		{"invalid string", "abc", 0, true},
		{"empty string", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := collector.ParseMetricValue(tt.input)
			if tt.err {
				require.Error(t, err)
			} else {
				require.Equal(t, tt.expected, val)
			}
		})
	}
}

func TestProcessMetricRow(t *testing.T) {
	collector := initTestFunc.NewMetricCollectorMonitor()
	ch := make(chan prometheus.Metric, 10)
	defer close(ch)

	t.Run("valid metric", func(t *testing.T) {
		err := collector.ProcessMetricRow("active_transactions", "10", ch)
		require.NoError(t, err)
		require.Len(t, ch, 1)
	})

	t.Run("invalid value", func(t *testing.T) {
		err := collector.ProcessMetricRow("active_transactions", "abc", ch)
		require.Error(t, err)
	})
}

func TestScrapeWithRetry(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	collector := initTestFunc.NewMetricCollectorMonitor()

	t.Run("success on first try", func(t *testing.T) {
		mock.ExpectQuery(initTestFunc.MySQLGlobalQueryMonitor).
			WillReturnRows(sqlmock.NewRows([]string{"Variable_Name", "Variable_Value"}))

		_, err := collector.ScrapeWithRetry(db)
		require.NoError(t, err)
	})

	t.Run("success after retry", func(t *testing.T) {
		mock.ExpectQuery(initTestFunc.MySQLGlobalQueryMonitor).
			WillReturnError(errors.New("temp error"))
		mock.ExpectQuery(initTestFunc.MySQLGlobalQueryMonitor).
			WillReturnRows(sqlmock.NewRows([]string{"Variable_Name", "Variable_Value"}))

		_, err := collector.ScrapeWithRetry(db)
		require.NoError(t, err)
	})

	t.Run("fail after max retries", func(t *testing.T) {
		for i := 0; i < initTestFunc.MaxRetries; i++ {
			mock.ExpectQuery(initTestFunc.MySQLGlobalQueryMonitor).
				WillReturnError(errors.New("persistent error"))
		}

		_, err := collector.ScrapeWithRetry(db)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed after 3 retries")
	})
}


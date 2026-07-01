package metrics

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	queryTimeout = 30 * time.Second
	MaxRetries = 3
)

type metricMonitor struct {
	Name      string
	ValueType prometheus.ValueType
	Help      string
}

type metricCollectorMonitor struct {
	MetricsRegistry map[string]*metricMonitor
	UnknownMetrics  prometheus.Counter
	queryStats struct {
		totalQueries   int
		failedQueries  int
		successQueries int
	}

	performanceMonitor struct {
		lastQueryDuration time.Duration
		maxQueryDuration  time.Duration
		minQueryDuration  time.Duration
	}
}

const MySQLGlobalQueryMonitor = "SELECT Variable_Name, Variable_Value FROM stats_mysql_global"

func NewMetricCollectorMonitor() *metricCollectorMonitor {
	return &metricCollectorMonitor{
		MetricsRegistry: map[string]*metricMonitor{
			"active_transactions":	{
				Name: "active_transactions",
				ValueType: prometheus.GaugeValue,
				Help: "Current number of active transactions",
			},
			"client_connections_aborted": {
				Name: "client_connections_aborted",
				ValueType: prometheus.CounterValue,
				Help: "Frontend connections aborted count",
			},
			"client_connections_connected": {
				Name: "client_connections_connected", 
				ValueType: prometheus.GaugeValue,
				Help: "Current frontend connections count",
			},
			"client_connections_created": {
				Name: "client_connections_created",
				ValueType: prometheus.CounterValue,
				Help: "Total frontend connections created",
			},
			"client_connections_non_idle": {
				Name: "client_connections_non_idle",
				ValueType: prometheus.GaugeValue,
				Help: "Non-idle client connections count",
			},
			"proxysql_uptime": {
				Name: "proxysql_uptime",
				ValueType: prometheus.CounterValue,
				Help: "Service uptime in seconds",
			},
			"questions": {
				Name: "questions",
				ValueType: prometheus.CounterValue,
				Help: "Total queries count",
			},
			"slow_queries": {
				Name: "slow_queries",
				ValueType: prometheus.CounterValue,
				Help: "Slow queries count",
			},
		},
		UnknownMetrics: prometheus.NewCounter(prometheus.CounterOpts{
            Name: "unknown_metrics_total",
            Help: "Total count of unknown metrics received",
        }),
	}
}

func (c *metricCollectorMonitor) BuildMetricDesc(metricName string) *prometheus.Desc {
	m := c.MetricsRegistry[metricName]
	if m == nil {
		m = &metricMonitor{
			Name:      metricName,
			ValueType: prometheus.UntypedValue,
			Help:      "Undocumented global metric",
		}
	}
	return prometheus.NewDesc(
		prometheus.BuildFQName(Namespace, "mysql_status", m.Name),
		m.Help,
		nil, nil,
	)
}

func (c *metricCollectorMonitor) ParseMetricValue(valueStr string) (float64, error) {
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return 0, fmt.Errorf("value parse error: %w", err)
	}
	return value, nil
}

func (c *metricCollectorMonitor) ProcessMetricRow(name, valueStr string, ch chan<- prometheus.Metric) error {
	if _, ok := c.MetricsRegistry[name]; !ok {
        ch <- c.UnknownMetrics
        return nil
    }

	value, err := c.ParseMetricValue(valueStr)
	if err != nil {
		return fmt.Errorf("metric %s %w", name, err)
	}

	metric := c.MetricsRegistry[name]
    ch <- prometheus.MustNewConstMetric(
        prometheus.NewDesc(
            strings.ToLower(name),
            metric.Help,
            nil, nil,
        ),
        metric.ValueType,
        value,
    )
    return nil
}

func (c *metricCollectorMonitor) ScrapeWithRetry(db *sql.DB) (*sql.Rows, error) {
	var rows *sql.Rows
	var err error

	for i := 0; i < MaxRetries; i++ {
		rows, err = db.Query(MySQLGlobalQueryMonitor)
		if err == nil {
			return rows, nil
		}
		time.Sleep(time.Second * time.Duration(i+1))
	}
	return nil, fmt.Errorf("query failed after %d retries: %w", MaxRetries, err)
}

func ScrapeMySQLGlobalNew(db *sql.DB, ch chan<- prometheus.Metric) error {
	collector := NewMetricCollectorMonitor()
	startTime := time.Now()

	rows, err := collector.ScrapeWithRetry(db)
	if err != nil {
		return fmt.Errorf("database query failed: %w", err)
	}
	defer rows.Close()

	var (
		metricCount int
		name, value string
	)

	for rows.Next() {
		if err := rows.Scan(&name, &value); err != nil {
			return fmt.Errorf("row scan failed: %w", err)
		}

		if err := collector.ProcessMetricRow(name, value, ch); err != nil {
			logger.Debug(err.Error())
			continue
		}
		metricCount++
	}

	queryDuration := time.Since(startTime)
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "scrape", "duration_seconds"),
			"Time spent scraping global metrics",
			nil, nil,
		),
		prometheus.GaugeValue,
		queryDuration.Seconds(),
	)

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "scrape", "metrics_count"),
			"Number of metrics processed",
			nil, nil,
		),
		prometheus.GaugeValue,
		float64(metricCount),
	)

	return rows.Err()
}

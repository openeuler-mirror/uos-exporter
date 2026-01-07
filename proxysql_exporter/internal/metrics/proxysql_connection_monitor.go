
package metrics

import (
	"database/sql"
	"fmt"
	// "strings"
	"time"
	"context"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	ConnectionListQuery = `SELECT 
		COUNT(cli_host) as connection_count, 
		cli_host 
		FROM stats_mysql_processlist 
		GROUP BY cli_host`
)

var (
	connectionMetricDef = &metric{
		name:      "client_connection_list",
		valueType: prometheus.GaugeValue,
		help:      "Total number of frontend connections per client host",
	}
)

type connectionRecord struct {
	host    string
	count   float64
}

type metricCollector struct {
	queryTimeout time.Duration
	maxRetries   int
}

func NewMetricCollector() *metricCollector {
	return &metricCollector{
		queryTimeout: 30 * time.Second,
		maxRetries:   3,
	}
}

func (c *metricCollector) buildMetricDesc() *prometheus.Desc {
	return prometheus.NewDesc(
		prometheus.BuildFQName(Namespace, "processlist", connectionMetricDef.name),
		connectionMetricDef.help,
		[]string{"client_host"},
		nil,
	)
}

func (c *metricCollector) validateColumns(columns []string) error {
	if len(columns) != 2 {
		return fmt.Errorf("invalid column count: expected 2, got %d", len(columns))
	}
	return nil
}

func (c *metricCollector) processRow(scan []interface{}, desc *prometheus.Desc, ch chan<- prometheus.Metric) error {
	record := connectionRecord{
		count: *scan[0].(*float64),
		host:  *scan[1].(*string),
	}

	if record.host == "" {
		return fmt.Errorf("empty client host detected")
	}

	ch <- prometheus.MustNewConstMetric(
		desc,
		connectionMetricDef.valueType,
		record.count,
		record.host,
	)
	return nil
}

func (c *metricCollector) scrapeWithRetry(db *sql.DB, query string) (*sql.Rows, error) {
	var rows *sql.Rows
	var err error

	for i := 0; i < c.maxRetries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), c.queryTimeout)
		defer cancel()

		rows, err = db.QueryContext(ctx, query)
		if err == nil {
			return rows, nil
		}
		time.Sleep(time.Second * time.Duration(i+1))
	}
	return nil, fmt.Errorf("query failed after %d retries: %w", c.maxRetries, err)
}

func ScrapeMySQLConnectionList(db *sql.DB, ch chan<- prometheus.Metric) error {
	collector := NewMetricCollector()
	desc := collector.buildMetricDesc()

	rows, err := collector.scrapeWithRetry(db, ConnectionListQuery)
	if err != nil {
		return fmt.Errorf("database query failed: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("get columns failed: %w", err)
	}

	if err := collector.validateColumns(columns); err != nil {
		return err
	}

	scan := make([]interface{}, len(columns))
	var (
		count  float64
		host   string
	)
	scan[0], scan[1] = &count, &host

	for rows.Next() {
		if err := rows.Scan(scan...); err != nil {
			return fmt.Errorf("row scan failed: %w", err)
		}

		if err := collector.processRow(scan, desc, ch); err != nil {
			return fmt.Errorf("process row failed: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows iteration error: %w", err)
	}
	startTime := time.Now()
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "processlist", "scrape_duration_seconds"),
			"Time spent scraping connection data",
			nil, nil,
		),
		prometheus.GaugeValue,
		float64(time.Since(startTime).Seconds()),
	)

	return nil
}

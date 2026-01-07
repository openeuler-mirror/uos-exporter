package metrics

import (
	"database/sql"
	"fmt"
	"strconv"
	// "strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	ConnectionPoolQuery = `SELECT 
		hostgroup, 
		srv_host, 
		srv_port, 
		status,
		ConnUsed,
		ConnFree,
		ConnOK,
		ConnERR,
		Queries,
		Bytes_data_sent,
		Bytes_data_recv,
		Latency_us 
		FROM stats_mysql_connection_pool`
)

type serverStatus int

const (
	connectStatusOnline serverStatus = iota + 1
	connectStatusShunned
	connectStatusOfflineSoft
	connectStatusOfflineHard
	connectStatusShunnedReplicationLag
)

var (
	statusMap = map[string]serverStatus{
		"ONLINE":                  connectStatusOnline,
		"SHUNNED":                 connectStatusShunned,
		"OFFLINE_SOFT":            connectStatusOfflineSoft,
		"OFFLINE_HARD":            connectStatusOfflineHard,
		"SHUNNED_REPLICATION_LAG": connectStatusShunnedReplicationLag,
	}

	connectionPoolMetrics = map[string]*metric{
		"status":          {"status", prometheus.GaugeValue, "Backend server status code"},
		"connused":        {"conn_used", prometheus.GaugeValue, "Active connections count"},
		"connfree":        {"conn_free", prometheus.GaugeValue, "Idle connections count"},
		"connok":          {"conn_ok", prometheus.CounterValue, "Successful connections count"},
		"connerr":         {"conn_err", prometheus.CounterValue, "Failed connections count"},
		"queries":         {"queries", prometheus.CounterValue, "Queries routed count"},
		"bytes_data_sent": {"bytes_data_sent", prometheus.CounterValue, "Data sent in bytes"},
		"bytes_data_recv": {"bytes_data_recv", prometheus.CounterValue, "Data received in bytes"},
		"latency_us":      {"latency_us", prometheus.GaugeValue, "Ping latency in microseconds"},
	}
)

type connectionPoolRecord struct {
	Hostgroup string
	Host      string
	Port      string
	Metrics   map[string]float64
}

type poolCollector struct {
	queryTimeout time.Duration
	maxRetries   int
}

func NewPoolCollector() *poolCollector {
	return &poolCollector{
		queryTimeout: 15 * time.Second,
		maxRetries:   3,
	}
}

func (c *poolCollector) BuildMetricDesc(metricName string) *prometheus.Desc {
	m := connectionPoolMetrics[metricName]
	if m == nil {
		m = &metric{
			name:      metricName,
			valueType: prometheus.UntypedValue,
			help:      "Undocumented connection pool metric",
		}
	}
	return prometheus.NewDesc(
		prometheus.BuildFQName(Namespace, "connection_pool", m.name),
		m.help,
		[]string{"hostgroup", "endpoint"},
		nil,
	)
}

func (c *poolCollector) ParseStatus(status string) (float64, error) {
	if val, exists := statusMap[status]; exists {
		return float64(val), nil
	}
	return 0, fmt.Errorf("invalid status value: %s", status)
}

func (c *poolCollector) ProcessRow(scan []interface{}) (*connectionPoolRecord, error) {
	
	if len(scan) < 12 { // 确保有足够的列
        return nil, fmt.Errorf("column count mismatch: expected at least 12, got %d", len(scan))
    }

	record := &connectionPoolRecord{
		Hostgroup: *scan[0].(*string),
		Host:      *scan[1].(*string),
		Port:      *scan[2].(*string),
		Metrics:   make(map[string]float64),
	}

	columns := []string{
		"status",
		"connused",
		"connfree",
		"connok",
		"connerr",
		"queries",
		"bytes_data_sent",
		"bytes_data_recv",
		"latency_us",
	}

	for i, col := range columns {
		valStr := *scan[i+3].(*string)
		var val float64
		var err error

		if col == "status" {
			val, err = c.ParseStatus(valStr)
		} else {
			val, err = strconv.ParseFloat(valStr, 64)
		}

		if err != nil {
			return nil, fmt.Errorf("parse %s failed: %w", col, err)
		}
		record.Metrics[col] = val
	}

	return record, nil
}

func ScrapeMySQLConnectionPool(db *sql.DB, ch chan<- prometheus.Metric) error {
	collector := NewPoolCollector()
	startTime := time.Now()

	rows, err := db.Query(ConnectionPoolQuery)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("get columns failed: %w", err)
	}

	scan := make([]interface{}, len(columns))
	for i := range scan {
		scan[i] = new(string)
	}

	for rows.Next() {
		if err := rows.Scan(scan...); err != nil {
			return fmt.Errorf("row scan failed: %w", err)
		}

		record, err := collector.ProcessRow(scan)
		if err != nil {
			return fmt.Errorf("process record failed: %w", err)
		}

		endpoint := fmt.Sprintf("%s:%s", record.Host, record.Port)
		for metricName, value := range record.Metrics {
			desc := collector.BuildMetricDesc(metricName)
			ch <- prometheus.MustNewConstMetric(
				desc,
				connectionPoolMetrics[metricName].valueType,
				value,
				record.Hostgroup, endpoint,
			)
		}
	}

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "scrape", "duration_seconds"),
			"Time spent scraping connection pool data",
			nil, nil,
		),
		prometheus.GaugeValue,
		time.Since(startTime).Seconds(),
	)

	return rows.Err()
}

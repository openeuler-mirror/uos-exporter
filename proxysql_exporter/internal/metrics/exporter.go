package metrics

import (
	"database/sql"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"context"
	"regexp"


	"github.com/prometheus/client_golang/prometheus"
)

const (
	Namespace              = "proxysql"
	DefaultTimeout         = 5 * time.Second
	MySQLDriverName        = "mysql"
	ConnectionPoolLabel    = "connection_pool"
	RuntimeServersLabel    = "runtime_servers"
	ProcessListLabel       = "processlist"
	MemoryMetricsLabel     = "memory_metrics"
	CommandCounterLabel    = "command_counter"
	GlobalStatusLabel      = "mysql_status"
)

type Exporter struct {
	dsn                              string
	scrapeMySQLGlobal                bool
	scrapeMySQLConnectionPool        bool
	scrapeMySQLConnectionList        bool
	scrapeDetailedMySQLProcessList   bool
	scrapeMySQLRuntimeServers        bool
	scrapeMemoryMetrics              bool
	scrapeMySQLCommandCounterMetrics bool
	scrapesTotal                     prometheus.Counter
	scrapeErrorsTotal                *prometheus.CounterVec
	lastScrapeError                  prometheus.Gauge
	lastScrapeDurationSeconds        prometheus.Gauge
	proxysqlUp                       prometheus.Gauge
}

func newCollectorExporter(
	dsn string,
	scrapeMySQLGlobal bool,
	scrapeMySQLConnectionPool bool,
	scrapeMySQLConnectionList bool,
	scrapeDetailedMySQLProcessList bool,
	scrapeMySQLRuntimeServers bool,
	scrapeMemoryMetrics bool,
	scrapeMySQLCommandCounterMetrics bool,
) *Exporter {
	return &Exporter{
		dsn:                              dsn,
		scrapeMySQLGlobal:                scrapeMySQLGlobal,
		scrapeMySQLConnectionPool:        scrapeMySQLConnectionPool,
		scrapeMySQLConnectionList:        scrapeMySQLConnectionList,
		scrapeDetailedMySQLProcessList:   scrapeDetailedMySQLProcessList,
		scrapeMySQLRuntimeServers:        scrapeMySQLRuntimeServers,
		scrapeMemoryMetrics:              scrapeMemoryMetrics,
		scrapeMySQLCommandCounterMetrics: scrapeMySQLCommandCounterMetrics,

		scrapesTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "exporter",
			Name:      "scrapes_total",
			Help:      "Total number of times ProxySQL was scraped for metrics.",
		}),
		scrapeErrorsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "exporter",
			Name:      "scrape_errors_total",
			Help:      "Total number of times an error occurred scraping a ProxySQL.",
		}, []string{"collector"}),
		lastScrapeError: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: "exporter",
			Name:      "last_scrape_error",
			Help:      "Whether the last scrape of metrics from ProxySQL resulted in an error (1 for error, 0 for success).",
		}),
		lastScrapeDurationSeconds: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: "exporter",
			Name:      "last_scrape_duration_seconds",
			Help:      "Duration of the last scrape of metrics from ProxySQL.",
		}),
		proxysqlUp: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "up",
			Help:      "Whether ProxySQL is up.",
		}),
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	metricCh := make(chan prometheus.Metric)
	doneCh := make(chan struct{})

	go func() {
		for m := range metricCh {
			ch <- m.Desc()
		}
		close(doneCh)
	}()

	e.Collect(metricCh)
	close(metricCh)
	<-doneCh
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.scrape(ch)

	e.scrapesTotal.Collect(ch)
	e.scrapeErrorsTotal.Collect(ch)
	e.lastScrapeError.Collect(ch)
	e.lastScrapeDurationSeconds.Collect(ch)
	e.proxysqlUp.Collect(ch)
}

func (e *Exporter) db() (*sql.DB, error) {
	db, err := sql.Open("mysql", e.dsn)
	if err == nil {
		err = db.Ping()
	}
	return db, err
}

func (e *Exporter) scrape(ch chan<- prometheus.Metric) {
	e.scrapesTotal.Inc()
	var err error
	defer func(begun time.Time) {
		e.lastScrapeDurationSeconds.Set(time.Since(begun).Seconds())
		if err == nil {
			e.lastScrapeError.Set(0)
		} else {
			e.lastScrapeError.Set(1)
		}
	}(time.Now())

	db, err := e.db()
	if db != nil {
		defer db.Close()
	}
	if err != nil {
		logger.Error("Error opening connection to ProxySQL", "error", err)
		e.proxysqlUp.Set(0)
		return
	}
	e.proxysqlUp.Set(1)

	if e.scrapeMySQLGlobal {
		if err = scrapeMySQLGlobal(db, ch); err != nil {
			logger.Error("Error scraping for collect.mysql_status:", "error", err)
			e.scrapeErrorsTotal.WithLabelValues("collect.mysql_status").Inc()
		}
	}
	if e.scrapeMySQLConnectionPool {
		if err = ScrapeMySQLConnectionPool(db, ch); err != nil {
			logger.Error("Error scraping for collect.mysql_connection_pool:", "error", err)
			e.scrapeErrorsTotal.WithLabelValues("collect.mysql_connection_pool").Inc()
		}
	}
	if e.scrapeMySQLConnectionList {
		if err = ScrapeMySQLConnectionList(db, ch); err != nil {
			logger.Error("Error scraping for collect.mysql_connection_list:", "error", err)
			e.scrapeErrorsTotal.WithLabelValues("collect.mysql_connection_list").Inc()
		}
	}
	if e.scrapeDetailedMySQLProcessList {
		if err = scrapeDetailedMySQLConnectionList(db, ch); err != nil {
			logger.Error("Error scraping for collect.stats_mysql_processlist", "error", err)
			e.scrapeErrorsTotal.WithLabelValues("collect.stats_mysql_processlist").Inc()
		}
	}
	if e.scrapeMySQLRuntimeServers {
		if err = scrapeMySQLRuntimeServers(db, ch); err != nil {
			logger.Error("Error scraping for collect.runtime_mysql_servers", "error", err)
			e.scrapeErrorsTotal.WithLabelValues("collect.runtime_mysql_servers").Inc()
		}
	}
	if e.scrapeMemoryMetrics {
		if err = scrapeMemoryMetrics(db, ch); err != nil {
			logger.Error("Error scraping for collect.stats_memory_metrics", "error", err)
			e.scrapeErrorsTotal.WithLabelValues("collect.stats_memory_metrics").Inc()
		}
	}
	if e.scrapeMySQLCommandCounterMetrics {
		if err = scrapeMySQLCommandCounterMetrics(db, ch); err != nil {
			logger.Error("Error scraping for collect.stats_command_counter_metrics", "error", err)
			e.scrapeErrorsTotal.WithLabelValues("collect.stats_command_counter_metrics").Inc()
		}
	}

	if err = scrapeProxySQLInfo(db, ch); err != nil {
		logger.Error("Error scraping for collect.proxysql_info", "error", err)
		e.scrapeErrorsTotal.WithLabelValues("collect.proxysql_info").Inc()
	}
}


type metric struct {
	name      string
	valueType prometheus.ValueType
	help      string
}

const mySQLGlobalQuery = "SELECT Variable_Name, Variable_Value FROM stats_mysql_global"

var mySQLGlobalMetrics = map[string]*metric{
	"active_transactions": {"active_transactions", prometheus.GaugeValue,
		"Current number of active transactions."},
	"client_connections_aborted": {"client_connections_aborted", prometheus.CounterValue,
		"Total number of frontend connections aborted due to invalid credential or max_connections reached."},
	"client_connections_connected": {"client_connections_connected", prometheus.GaugeValue,
		"Current number of frontend connections."},
	"client_connections_created": {"client_connections_created", prometheus.CounterValue,
		"Total number of frontend connections created so far."},
	"client_connections_non_idle": {"client_connections_non_idle", prometheus.GaugeValue,
		"Current number of client connections that are not idle."},
	"proxysql_uptime": {"proxysql_uptime", prometheus.CounterValue,
		"Uptime in seconds."},
	"questions": {"questions", prometheus.CounterValue,
		"Total number of queries sent from frontends."},
	"slow_queries": {"slow_queries", prometheus.CounterValue,
		"Total number of queries that ran for longer than the threshold in milliseconds defined in global variable mysql-long_query_time."},
}

func scrapeMySQLGlobal(db *sql.DB, ch chan<- prometheus.Metric) error {
    descCache := make(map[string]*prometheus.Desc)
    getOrCreateDesc := func(name string, m *metric) *prometheus.Desc {
        if desc, exists := descCache[name]; exists {
            return desc
        }
        desc := prometheus.NewDesc(
            prometheus.BuildFQName(Namespace, "mysql_status", name),
            m.help,
            nil, nil,
        )
        descCache[name] = desc
        return desc
    }

    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()

    rows, err := db.QueryContext(ctx, mySQLGlobalQuery)
    if err != nil {
        return fmt.Errorf("query failed: %w", err)
    }
    defer func() {
        if err := rows.Close(); err != nil {
            logger.Warn("rows close error", "error", err)
        }
    }()

    var (
        name, valueStr string
        processedCount int
    )
    for rows.Next() {
        if err := rows.Scan(&name, &valueStr); err != nil {
            logger.Error("row scan failed", "error", err)
            continue
        }

        value, err := strconv.ParseFloat(valueStr, 64)
        if err != nil {
            logger.Debug("parse metric value failed", 
                "metric", name, "value", valueStr, "error", err)
            continue
        }

        metricName := strings.ToLower(name)
        m := mySQLGlobalMetrics[metricName]
        if m == nil {
            m = &metric{
                name:      metricName,
                valueType: prometheus.UntypedValue,
                help:      "Undocumented metric",
            }
        }

        ch <- prometheus.MustNewConstMetric(
            getOrCreateDesc(metricName, m),
            m.valueType,
            value,
        )
        processedCount++
    }

    ch <- prometheus.MustNewConstMetric(
        prometheus.NewDesc(
            prometheus.BuildFQName(Namespace, "scrape", "processed_metrics_total"),
            "Number of metrics processed",
            nil, nil,
        ),
        prometheus.CounterValue,
        float64(processedCount),
    )

    return rows.Err()
}


const (
    detailedMySQLProcessListQuery = `
	SELECT 
        user, 
		db, 
		cli_host, 
		hostgroup, 
		COUNT(*) as count 
        FROM stats_mysql_processlist 
    GROUP BY 
		user, 
		db, 
		cli_host, 
		hostgroup`
)

var (
    processListLabels = []string{"user", "db", "client_host", "hostgroup"}
    processListMetric = &metric{
        name:      "detailed_client_connection_count",
        valueType: prometheus.GaugeValue,
        help:      "Number of client connections per user, db, host and hostgroup",
    }
)

type processListResult struct {
    user, db, clientHost, hostGroup string
    count                           float64
}

func newProcessListDesc() *prometheus.Desc {
    return prometheus.NewDesc(
        prometheus.BuildFQName(Namespace, "processlist", processListMetric.name),
        processListMetric.help,
        processListLabels,
        nil,
    )
}

func scrapeDetailedMySQLConnectionList(db *sql.DB, ch chan<- prometheus.Metric) error {
    startTime := time.Now()
    defer func() {
        ch <- prometheus.MustNewConstMetric(
            prometheus.NewDesc(
                prometheus.BuildFQName(Namespace, "processlist", "scrape_duration_seconds"),
                "Time spent scraping processlist data",
                nil, nil,
            ),
            prometheus.GaugeValue,
            time.Since(startTime).Seconds(),
        )
    }()

    rows, err := db.Query(detailedMySQLProcessListQuery)
    if err != nil {
        return fmt.Errorf("processlist query failed: %w", err)
    }
    defer rows.Close()

    desc := newProcessListDesc()
    for rows.Next() {
        var res processListResult
        if err := rows.Scan(
            &res.user, &res.db, &res.clientHost, &res.hostGroup, &res.count,
        ); err != nil {
            return fmt.Errorf("scan processlist row failed: %w", err)
        }

        ch <- prometheus.MustNewConstMetric(
            desc,
            processListMetric.valueType,
            res.count,
            res.user, res.db, res.clientHost, res.hostGroup,
        )
    }
    return rows.Err()
}

const mysqlCommandCounterQuery = `
    SELECT
        Command, 
		Total_Time_us, 
		Total_cnt, 
		cnt_100us, 
		cnt_500us, 
		cnt_1ms, 
		cnt_5ms, 
		cnt_10ms, 
		cnt_50ms,
        cnt_100ms, 
		cnt_500ms, 
		cnt_1s, 
		cnt_5s, 
		cnt_10s, 
		cnt_INFs
    FROM
        stats_mysql_commands_counters
    WHERE Command IN (
        'CREATE_TEMPORARY',
        'DELETE',
        'INSERT',
        'LOCK_TABLES',
        'SELECT',
        'SELECT_FOR_UPDATE',
        'UPDATE'
    )
`

var sInf = math.Inf(1)  

const (
    us100  = 0.1
    us500  = 0.5
    ms1    = 1
    ms5    = 5
    ms10   = 10
    ms50   = 50
    ms100  = 100
    ms500  = 500
    s1     = 1000
    s5     = 5000
    s10    = 10000
)

type latencyBuckets struct {
    us100, us500 uint64
    ms1, ms5, ms10, ms50, ms100, ms500 uint64
    s1, s5, s10, sInf uint64
}

type mysqlCommandCounterResult struct {
    Command     string
    TotalTimeUs float64
    TotalCnt    uint64
    latencyBuckets
}

func buildHistogramBuckets(res latencyBuckets) map[float64]uint64 {
    return map[float64]uint64{
        us100: res.us100,
        us500: res.us500 + res.us100,
        ms1:   res.ms1 + res.us500 + res.us100,
        ms5:   res.ms5 + res.ms1 + res.us500 + res.us100,
        ms10:  res.ms10 + res.ms5 + res.ms1 + res.us500 + res.us100,
        ms50:  res.ms50 + res.ms10 + res.ms5 + res.ms1 + res.us500 + res.us100,
        ms100: res.ms100 + res.ms50 + res.ms10 + res.ms5 + res.ms1 + res.us500 + res.us100,
        ms500: res.ms500 + res.ms100 + res.ms50 + res.ms10 + res.ms5 + res.ms1 + res.us500 + res.us100,
        s1:    res.s1 + res.ms500 + res.ms100 + res.ms50 + res.ms10 + res.ms5 + res.ms1 + res.us500 + res.us100,
        s5:    res.s5 + res.s1 + res.ms500 + res.ms100 + res.ms50 + res.ms10 + res.ms5 + res.ms1 + res.us500 + res.us100,
        s10:   res.s10 + res.s5 + res.s1 + res.ms500 + res.ms100 + res.ms50 + res.ms10 + res.ms5 + res.ms1 + res.us500 + res.us100,
        sInf:  res.sInf + res.s10 + res.s5 + res.s1 + res.ms500 + res.ms100 + res.ms50 + res.ms10 + res.ms5 + res.ms1 + res.us500 + res.us100,
    }
}

func scrapeMySQLCommandCounterMetrics(db *sql.DB, ch chan<- prometheus.Metric) error {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    rows, err := db.QueryContext(ctx, mysqlCommandCounterQuery)
    if err != nil {
        return fmt.Errorf("query failed: %w", err)
    }
    defer rows.Close()

    for rows.Next() {
        var res mysqlCommandCounterResult
        if err := rows.Scan(
            &res.Command, &res.TotalTimeUs, &res.TotalCnt,
            &res.us100, &res.us500, &res.ms1, &res.ms5, &res.ms10,
            &res.ms50, &res.ms100, &res.ms500, &res.s1, &res.s5, &res.s10, &res.sInf,
        ); err != nil {
            return fmt.Errorf("scan failed: %w", err)
        }

        desc := prometheus.NewDesc(
            prometheus.BuildFQName(Namespace, "mysql_command_counter", "latency_milliseconds"),
            "histogram over a commands latency in ms",
            []string{"command"},
            nil,
        )
        ch <- prometheus.MustNewConstHistogram(
            desc,
            res.TotalCnt,
            res.TotalTimeUs,
            buildHistogramBuckets(res.latencyBuckets),
            res.Command,
        )
    }

    if err := rows.Err(); err != nil {
        return fmt.Errorf("rows iteration failed: %w", err)
    }
    return nil
}


const mySQLruntimeServersQuery = "SELECT hostgroup_id, hostname, port, gtid_port, * FROM runtime_mysql_servers"

var mySQLruntimeServersMetrics = map[string]*metric{
	"status": {"status", prometheus.GaugeValue,
		"The status of the backend server (1 - ONLINE, 2 - SHUNNED, 3 - OFFLINE_SOFT, 4 - OFFLINE_HARD)."},

	"weight": {"weight", prometheus.GaugeValue,
		"The bigger the weight of a server relative to other weights, the higher the probability of the server to be chosen from a hostgroup."},
	
	"compression": {"compression", prometheus.GaugeValue,
		"If the value is 1, new connections to that server will use compression."},

	"max_connections": {"max_connections", prometheus.GaugeValue,
		"The maximum number of connections ProxySQL will open to this backend server."},

	"max_replication_lag": {"max_replication_lag", prometheus.GaugeValue,
		"If greater than 0, ProxySQL will regularly monitor replication lag and if it goes beyond such threshold it will temporary shun the host until replication catches up."},
	
	"use_ssl": {"use_ssl", prometheus.GaugeValue,
		"If set to 1, connections to the backend will use SSL."},

	"max_latency_ms": {"max_latency_ms", prometheus.GaugeValue,
		"Ping time."},
}

const (
    statusOnline     = 1
    statusShunned    = 2
    statusOfflineSoft = 3 
    statusOfflineHard = 4
)

var skipColumns = map[string]struct{}{
    "hostgroup_id": {},
    "hostname":     {},
    "port":         {},
    "gtid_port":    {},
}

type serverScanResult struct {
    hostgroupID string
    hostname    string
    port        string  
    gtidPort    string
    columns     []string
    values      []*string
}

func newServerScanResult(cols []string) *serverScanResult {
    s := &serverScanResult{
        columns: cols,
        values:  make([]*string, len(cols)),
    }
    for i := range s.values {
        s.values[i] = new(string)
    }
    return s
}

func convertStatus(status string) (float64, error) {
    switch status {
    	case "ONLINE":      
			return statusOnline, nil
    	case "SHUNNED":     
			return statusShunned, nil
    	case "OFFLINE_SOFT":
			return statusOfflineSoft, nil
    	case "OFFLINE_HARD":
			return statusOfflineHard, nil
    	default: 
			return 0, fmt.Errorf("invalid status: %s", status)
    }
}

func scrapeMySQLRuntimeServers(db *sql.DB, ch chan<- prometheus.Metric) error {
    rows, err := db.Query(mySQLruntimeServersQuery)
    if err != nil {
        return fmt.Errorf("query failed: %w", err)
    }
    defer rows.Close()

    cols, err := rows.Columns()
    if err != nil {
        return fmt.Errorf("get columns failed: %w", err)
    }

    result := newServerScanResult(cols)
    scanArgs := make([]interface{}, len(cols))
    scanArgs[0], 
	scanArgs[1], 
	scanArgs[2], 
	scanArgs[3] = 
        &result.hostgroupID, 
		&result.hostname, 
		&result.port, 
		&result.gtidPort

	strValues := make([]interface{}, len(result.values[4:]))
	for i, v := range result.values[4:] {
		strValues[i] = v
	}
	copy(scanArgs[4:], strValues)


    for rows.Next() {
        if err := rows.Scan(scanArgs...); err != nil {
            return fmt.Errorf("scan failed: %w", err)
        }

        for i, col := range cols {
            if _, ok := skipColumns[col]; ok {
                continue
            }

            valStr := *result.values[i]
            col = strings.ToLower(col)

            var val float64
            var err error
            if col == "status" {
                val, err = convertStatus(valStr)
            } else {
                val, err = strconv.ParseFloat(valStr, 64)
            }
            if err != nil {
                logger.Debug(fmt.Sprintf("column %s: %v", col, err))
                continue
            }

            m := mySQLruntimeServersMetrics[col]
            if m == nil {
                m = &metric{
                    name:      col,
                    valueType: prometheus.UntypedValue,
                    help:      "Undocumented runtime_mysql_servers metric",
                }
            }

            endpoint := fmt.Sprintf("%s:%s", result.hostname, result.port)
            desc := prometheus.NewDesc(
                prometheus.BuildFQName(Namespace, "runtime_servers", m.name),
                m.help,
                []string{"hostgroup", "endpoint", "gtid_port"},
                nil,
            )
            ch <- prometheus.MustNewConstMetric(desc, m.valueType, val,
                result.hostgroupID, endpoint, result.gtidPort)
        }
    }
    return rows.Err()
}

const memoryMetricsQuery = "select Variable_Name, Variable_Value  from stats_memory_metrics"

var memoryMetricsDefs = []struct {
    key       string
    valueType prometheus.ValueType
    help      string
}{
    {"jemalloc_allocated", 
		prometheus.GaugeValue, 
			"bytes allocated by the application"},

    {"jemalloc_active", 
		prometheus.GaugeValue, 
			"bytes in pages allocated by the application"},

    {"jemalloc_mapped", 
		prometheus.GaugeValue, 
			"bytes in extents mapped by the allocator"},

    {"jemalloc_metadata", 
		prometheus.GaugeValue, 
			"bytes dedicated to metadata"},

    {"jemalloc_resident", 
		prometheus.GaugeValue, 
			"bytes in physically resident data pages"},

    {"auth_memory", 
		prometheus.GaugeValue, 
			"memory for authentication credentials"},

    {"sqlite3_memory_bytes", 
		prometheus.GaugeValue, 
			"memory used by embedded SQLite"},

    {"query_digest_memory", 
		prometheus.GaugeValue, 
		"memory for query digest data"},
}

var memoryMetricsMap = func() map[string]*metric {
    m := make(map[string]*metric, len(memoryMetricsDefs))
    for _, def := range memoryMetricsDefs {
        m[def.key] = &metric{
            name:      def.key,
            valueType: def.valueType,
            help:      def.help,
        }
    }
    return m
}()

type memoryMetricsResult struct {
    name  string
    value float64
}

func newMetricDesc(name, help string) *prometheus.Desc {
    return prometheus.NewDesc(
        prometheus.BuildFQName(Namespace, "stats_memory", name),
        help,
        nil, nil,
    )
}

func scrapeMemoryMetrics(db *sql.DB, ch chan<- prometheus.Metric) error {
    rows, err := db.Query(memoryMetricsQuery)
    if err != nil {
        return fmt.Errorf("query failed: %w", err)
    }
    defer rows.Close()

    for rows.Next() {
        var res memoryMetricsResult
        if err := rows.Scan(&res.name, &res.value); err != nil {
            return fmt.Errorf("scan failed: %w", err)
        }

        key := strings.ToLower(res.name)
        m := memoryMetricsMap[key]
        if m == nil {
            m = &metric{
                name:      key,
                valueType: prometheus.UntypedValue,
                help:      "Undocumented memory metric: " + key,
            }
        }

        ch <- prometheus.MustNewConstMetric(
            newMetricDesc(m.name, m.help),
            m.valueType,
            res.value,
        )
    }
    return rows.Err()
}


type VersionCollector interface {
    Collect(context.Context, *sql.DB, chan<- prometheus.Metric) error
    Describe() *prometheus.Desc
}

type BaseVersionCollector struct {
    queryTemplate  string
    metricDesc     *prometheus.Desc
    timeout        time.Duration
    maxRetries     int
    sanitizer      VersionSanitizer
    logger         VersionLogger
}

func NewBaseVersionCollector() *BaseVersionCollector {
    return &BaseVersionCollector{
        queryTemplate: `
            SELECT variable_value 
            FROM global_variables 
            WHERE variable_name = ?`,
        metricDesc: prometheus.NewDesc(
            prometheus.BuildFQName(Namespace, "", "info"),
            "ProxySQL info",
            []string{"version"}, 
            nil,
        ),
        timeout:    30 * time.Second,
        maxRetries: 3,
        sanitizer:  NewDefaultVersionSanitizer(),
        logger:     NewVersionLogger(),
    }
}

type VersionSanitizer interface {
    Sanitize(string) string
    Validate(string) error
}

type DefaultVersionSanitizer struct {
    pattern *regexp.Regexp
}

func NewDefaultVersionSanitizer() *DefaultVersionSanitizer {
    return &DefaultVersionSanitizer{
        pattern: regexp.MustCompile(`^[a-zA-Z0-9\.\_\-]+$`),
    }
}

func (s *DefaultVersionSanitizer) Sanitize(v string) string {
    cleaned := strings.TrimSpace(v)
    cleaned = strings.ReplaceAll(cleaned, " ", "_")
    return strings.ToLower(cleaned)
}

func (s *DefaultVersionSanitizer) Validate(v string) error {
    if !s.pattern.MatchString(v) {
        return fmt.Errorf("invalid version format: %s", v)
    }
    return nil
}

type VersionLogger interface {
    Debug(msg string, fields map[string]interface{})
    Warn(msg string, fields map[string]interface{})
    Error(msg string, fields map[string]interface{})
}

type DefaultVersionLogger struct {

}

func NewVersionLogger() *DefaultVersionLogger {
    return &DefaultVersionLogger{}
}

func (l *DefaultVersionLogger) Debug(msg string, fields map[string]interface{}) {
    // logger.Debug(msg, fields)
	args := make([]interface{}, 0, len(fields)*2)
    for k, v := range fields {
        args = append(args, k, v)
    }
    logger.Debug(msg, args...)
}

func (l *DefaultVersionLogger) Warn(msg string, fields map[string]interface{}) {
    // logger.Warn(msg, fields)
	args := make([]interface{}, 0, len(fields)*2)
    for k, v := range fields {
        args = append(args, k, v)
    }
    logger.Warn(msg, args...)
}

func (l *DefaultVersionLogger) Error(msg string, fields map[string]interface{}) {
    // logger.Error(msg, fields)
	args := make([]interface{}, 0, len(fields)*2)
    for k, v := range fields {
        args = append(args, k, v)
    }
    logger.Error(msg, args...)
}

type VersionError struct {
    Code     ErrorCode
    Message  string
    Query    string
    Attempt  int
    Original error
}

type ErrorCode int

const (
    CodeQueryExecution ErrorCode = iota + 1
    CodeResultParsing
    CodeValidationFailed
    CodeMultipleRecords
    CodeNoResult
)

func (e *VersionError) Error() string {
    return fmt.Sprintf("[%d] %s (query: %s, attempt: %d)", 
        e.Code, e.Message, e.Query, e.Attempt)
}

func (e *VersionError) Unwrap() error {
    return e.Original
}

func (c *BaseVersionCollector) Collect(ctx context.Context, db *sql.DB, ch chan<- prometheus.Metric) error {
    var lastErr error
    
    for attempt := 1; attempt <= c.maxRetries; attempt++ {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            err := c.executeCollection(ctx, db, ch, attempt)
            if err == nil {
                return nil
            }
            lastErr = err
            
            c.logger.Warn("version collection retrying", map[string]interface{}{
                "attempt":    attempt,
                "max_retries": c.maxRetries,
                "error":      err.Error(),
            })
            
            time.Sleep(time.Duration(attempt) * time.Second)
        }
    }
    
    return fmt.Errorf("after %d attempts: %w", c.maxRetries, lastErr)
}

func (c *BaseVersionCollector) executeCollection(
    ctx context.Context, 
    db *sql.DB, 
    ch chan<- prometheus.Metric,
    attempt int,
) error {
    queryCtx, cancel := context.WithTimeout(ctx, c.timeout)
    defer cancel()

    stmt, err := db.PrepareContext(queryCtx, c.queryTemplate)
    if err != nil {
        return &VersionError{
            Code:     CodeQueryExecution,
            Message:  "prepare statement failed",
            Query:    c.queryTemplate,
            Attempt:  attempt,
            Original: err,
        }
    }
    defer stmt.Close()

    rows, err := stmt.QueryContext(queryCtx, "admin-version")
    if err != nil {
        return &VersionError{
            Code:     CodeQueryExecution,
            Message:  "query execution failed",
            Query:    c.queryTemplate,
            Attempt:  attempt,
            Original: err,
        }
    }
    defer rows.Close()

    var version string
    found := false
    
    for rows.Next() {
        if err := rows.Scan(&version); err != nil {
            return &VersionError{
                Code:     CodeResultParsing,
                Message:  "result parsing failed",
                Query:    c.queryTemplate,
                Attempt:  attempt,
                Original: err,
            }
        }
        
        if found {
            c.logger.Warn("multiple version records detected", map[string]interface{}{
                "query":    c.queryTemplate,
                "attempt":  attempt,
            })
            continue
        }
        
        cleanedVersion := c.sanitizer.Sanitize(version)
        if err := c.sanitizer.Validate(cleanedVersion); err != nil {
            return &VersionError{
                Code:     CodeValidationFailed,
                Message:  "version validation failed",
                Query:    c.queryTemplate,
                Attempt:  attempt,
                Original: err,
            }
        }
        
        ch <- prometheus.MustNewConstMetric(
            c.metricDesc,
            prometheus.GaugeValue,
            0,
            cleanedVersion,
        )
        found = true
    }

    if err := rows.Err(); err != nil {
        return &VersionError{
            Code:     CodeResultParsing,
            Message:  "result iteration failed",
            Query:    c.queryTemplate,
            Attempt:  attempt,
            Original: err,
        }
    }

    if !found {
        return &VersionError{
            Code:     CodeNoResult,
            Message:  "no version information found",
            Query:    c.queryTemplate,
            Attempt:  attempt,
        }
    }

    return nil
}

func scrapeProxySQLInfo(db *sql.DB, ch chan<- prometheus.Metric) error {
    collector := NewBaseVersionCollector()
    ctx := context.Background()
    return collector.Collect(ctx, db, ch)
}


// check interface
var _ prometheus.Collector = (*Exporter)(nil)

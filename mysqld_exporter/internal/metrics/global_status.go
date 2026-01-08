package metrics

import (
	"database/sql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"mysqld_exporter/internal/exporter"
	"mysqld_exporter/internal/mysql"
	"regexp"
)

const (
	globalStatusQuery  = `SHOW GLOBAL STATUS`
	globalStatusResult = `
Variable_name: Ssl_verify_depth
        Value: 0
*************************** 477. row ***************************
Variable_name: Ssl_verify_mode
        Value: 0
*************************** 478. row ***************************
Variable_name: Ssl_version
        Value: 
*************************** 479. row ***************************
Variable_name: Table_locks_immediate
        Value: 41
*************************** 480. row ***************************
Variable_name: Table_locks_waited
        Value: 0
*************************** 481. row ***************************
Variable_name: Table_open_cache_hits
        Value: 3692
*************************** 482. row ***************************
Variable_name: Table_open_cache_misses
        Value: 221
*************************** 483. row ***************************
Variable_name: Table_open_cache_overflows
        Value: 0
*************************** 484. row ***************************
Variable_name: Tc_log_max_pages_used
        Value: 0
*************************** 485. row ***************************
Variable_name: Tc_log_page_size
        Value: 0
*************************** 486. row ***************************
Variable_name: Tc_log_page_waits
        Value: 0
*************************** 487. row ***************************
Variable_name: Telemetry_traces_supported
        Value: ON
*************************** 488. row ***************************
Variable_name: Threads_cached
        Value: 1
*************************** 489. row ***************************
Variable_name: Threads_connected
        Value: 1
*************************** 490. row ***************************
Variable_name: Threads_created
        Value: 2
*************************** 491. row ***************************
Variable_name: Threads_running
        Value: 2
*************************** 492. row ***************************
Variable_name: Tls_library_version
        Value: OpenSSL 3.0.12 24 Oct 2023
*************************** 493. row ***************************
Variable_name: Uptime
        Value: 76845
*************************** 494. row ***************************
Variable_name: Uptime_since_flush_status
        Value: 76845

`
	globalStatus = "global_status"
)

var (
	globalStatusRE = regexp.MustCompile(`^(com|handler|connection_errors|innodb_buffer_pool_pages|innodb_rows|performance_schema)_(.*)$`)
)

type ScrapeGlobalStatus struct {
	instance mysql.Instance
	globalCommands
	globalHandler
	globalConnectionErrors
	globalBufferPoolPages
	globalBufferPoolDirtyPages
	globalBufferPoolPageChanges
	globalInnoDBRowOps
	globalPerformanceSchemaLost
}

func init() {
	exporter.Register(
		NewScrapeGlobalStatus())
}

func NewScrapeGlobalStatus() *ScrapeGlobalStatus {
	return &ScrapeGlobalStatus{
		//instance:                    instance,
		globalCommands:              *NewglobalCommands(),
		globalHandler:               *NewglobalHandler(),
		globalConnectionErrors:      *NewglobalConnectionErrors(),
		globalBufferPoolPages:       *NewglobalBufferPoolPages(),
		globalBufferPoolDirtyPages:  *NewglobalBufferPoolDirtyPages(),
		globalBufferPoolPageChanges: *NewglobalBufferPoolPageChanges(),
		globalInnoDBRowOps:          *NewglobalInnoDBRowOps(),
		globalPerformanceSchemaLost: *NewglobalPerformanceSchemaLost(),
	}
}

func (qd ScrapeGlobalStatus) Collect(ch chan<- prometheus.Metric) {
	qd.instance = *GetInstance()
	if err := qd.instance.Ping(); err != nil {
		logrus.Errorf("ping mysql instance error: %s", err)
		return
	}
	db := instance.GetDB()
	rows, err := db.Query(globalStatusQuery)
	if err != nil {
		logrus.Error(err)
		return
	}
	defer rows.Close()
	var key string
	var val sql.RawBytes

	for rows.Next() {
		err := rows.Scan(&key, &val)
		if err != nil {
			logrus.Error(err)
			return
		}
		if floatVal, ok := parseStatus(val); ok {
			key = validPrometheusName(key)
			match := globalStatusRE.FindStringSubmatch(key)
			if match == nil {
				continue
			}
			switch match[1] {
			case "com":
				qd.globalCommands.Collect(ch,
					floatVal,
					[]string{
						match[2],
					})

			case "handler":
				qd.globalHandler.Collect(ch,
					floatVal,
					[]string{
						match[2],
					})

			case "connection_errors":
				qd.globalConnectionErrors.Collect(ch,
					floatVal,
					[]string{
						match[2],
					})

			case "innodb_buffer_pool_pages":
				switch match[2] {
				case "data", "free", "misc", "old":
					qd.globalBufferPoolPages.Collect(ch,
						floatVal,
						[]string{
							match[2],
						})
				case "dirty":
					qd.globalBufferPoolDirtyPages.Collect(ch,
						floatVal,
						nil)
				case "total":
					continue
				default:
					qd.globalBufferPoolPageChanges.Collect(ch,
						floatVal,
						[]string{
							match[2],
						})
				}
			case "innodb_rows":
				qd.globalInnoDBRowOps.Collect(ch,
					floatVal,
					[]string{
						match[2],
					})

			case "performance_schema":
				qd.globalPerformanceSchemaLost.Collect(ch,
					floatVal,
					[]string{
						match[2],
					})
			}
		}
	}
}

type globalCommands struct {
	*baseMetrics
}

func NewglobalCommands() *globalCommands {
	return &globalCommands{
		NewMetrics(
			"global_status_commands_total",
			"Total number of executed MySQL commands.",
			[]string{
				"command",
			})}
}
func (qd *globalCommands) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type globalHandler struct {
	*baseMetrics
}

func NewglobalHandler() *globalHandler {
	return &globalHandler{
		NewMetrics(
			"global_status_handlers_total",
			"Total number of executed MySQL handlers.",
			[]string{
				"handler",
			})}
}
func (qd *globalHandler) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type globalConnectionErrors struct {
	*baseMetrics
}

func NewglobalConnectionErrors() *globalConnectionErrors {
	return &globalConnectionErrors{
		NewMetrics(
			"global_status_connection_errors_total",
			"Total number of MySQL connection errors.",
			[]string{
				"error",
			})}
}
func (qd *globalConnectionErrors) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type globalBufferPoolPages struct {
	*baseMetrics
}

func NewglobalBufferPoolPages() *globalBufferPoolPages {
	return &globalBufferPoolPages{
		NewMetrics(
			"global_status_buffer_pool_pages",
			"Innodb buffer pool pages by state.",
			[]string{
				"state",
			})}
}
func (qd *globalBufferPoolPages) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type globalBufferPoolDirtyPages struct {
	*baseMetrics
}

func NewglobalBufferPoolDirtyPages() *globalBufferPoolDirtyPages {
	return &globalBufferPoolDirtyPages{
		NewMetrics(
			"global_status_buffer_pool_dirty_pages",
			"Innodb buffer pool dirty pages.",
			nil)}
}
func (qd *globalBufferPoolDirtyPages) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type globalBufferPoolPageChanges struct {
	*baseMetrics
}

func NewglobalBufferPoolPageChanges() *globalBufferPoolPageChanges {
	return &globalBufferPoolPageChanges{
		NewMetrics(
			"global_status_buffer_pool_page_changes_total",
			"Innodb buffer pool page state changes.",
			[]string{
				"operation",
			})}
}
func (qd *globalBufferPoolPageChanges) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type globalInnoDBRowOps struct {
	*baseMetrics
}

func NewglobalInnoDBRowOps() *globalInnoDBRowOps {
	return &globalInnoDBRowOps{
		NewMetrics(
			"global_status_innodb_row_ops_total",
			"Total number of MySQL InnoDB row operations.",
			[]string{
				"operation",
			})}
}
func (qd *globalInnoDBRowOps) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type globalPerformanceSchemaLost struct {
	*baseMetrics
}

func NewglobalPerformanceSchemaLost() *globalPerformanceSchemaLost {
	return &globalPerformanceSchemaLost{
		NewMetrics(
			"global_status_performance_schema_lost_total",
			"Total number of MySQL instrumentations that could not be loaded or created due to memory constraints.",
			[]string{
				"instrumentation",
			})}
}
func (qd *globalPerformanceSchemaLost) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

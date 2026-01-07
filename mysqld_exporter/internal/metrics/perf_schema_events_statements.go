package metrics

import (
	"fmt"
	"github.com/Masterminds/semver/v3"
	"github.com/alecthomas/kingpin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"mysqld_exporter/internal/exporter"
	"mysqld_exporter/internal/mysql"
)

const (
	perfEventsStatementsQuery = `
	SELECT
	    ifnull(SCHEMA_NAME, 'NONE') as SCHEMA_NAME,
	    DIGEST,
	    LEFT(DIGEST_TEXT, %d) as DIGEST_TEXT,
	    COUNT_STAR,
	    SUM_TIMER_WAIT,
	    SUM_ERRORS,
	    SUM_WARNINGS,
	    SUM_ROWS_AFFECTED,
	    SUM_ROWS_SENT,
	    SUM_ROWS_EXAMINED,
	    SUM_CREATED_TMP_DISK_TABLES,
	    SUM_CREATED_TMP_TABLES,
	    SUM_SORT_MERGE_PASSES,
	    SUM_SORT_ROWS,
	    SUM_NO_INDEX_USED
	  FROM (
	    SELECT *
	    FROM performance_schema.events_statements_summary_by_digest
	    WHERE SCHEMA_NAME NOT IN ('mysql', 'performance_schema', 'information_schema')
	      AND LAST_SEEN > DATE_SUB(NOW(), INTERVAL %d SECOND)
	    ORDER BY LAST_SEEN DESC
	  )Q
	  GROUP BY
	    Q.SCHEMA_NAME,
	    Q.DIGEST,
	    Q.DIGEST_TEXT,
	    Q.COUNT_STAR,
	    Q.SUM_TIMER_WAIT,
	    Q.SUM_ERRORS,
	    Q.SUM_WARNINGS,
	    Q.SUM_ROWS_AFFECTED,
	    Q.SUM_ROWS_SENT,
	    Q.SUM_ROWS_EXAMINED,
	    Q.SUM_CREATED_TMP_DISK_TABLES,
	    Q.SUM_CREATED_TMP_TABLES,
	    Q.SUM_SORT_MERGE_PASSES,
	    Q.SUM_SORT_ROWS,
	    Q.SUM_NO_INDEX_USED
	  ORDER BY SUM_TIMER_WAIT DESC
	  LIMIT %d
	`
	perfEventsStatementsResult = `
   SCHEMA_NAME: NULL
        DIGEST: 1d71af84aa7e0dd94d128248786259636447819c7abb38d93986dd5f26eaffb2
    COUNT_STAR: 1
SUM_TIMER_WAIT: 1946601000
MIN_TIMER_WAIT: 1946601000
AVG_TIMER_WAIT: 1946601000
MAX_TIMER_WAIT: 1946601000
 SUM_LOCK_TIME: 10000000
    SUM_ERRORS: 0
 SUM_ROWS_SENT: 6
*************************** 41. row ***************************
   SCHEMA_NAME: NULL
        DIGEST: b589a24cd26f1989b32023956bdc889070b4533f708703f41bc85347b9a53956
    COUNT_STAR: 7
SUM_TIMER_WAIT: 30350755000
MIN_TIMER_WAIT: 4042843000
AVG_TIMER_WAIT: 4335822000
MAX_TIMER_WAIT: 4717462000
 SUM_LOCK_TIME: 18000000
    SUM_ERRORS: 0
 SUM_ROWS_SENT: 2807
*************************** 42. row ***************************
   SCHEMA_NAME: NULL
        DIGEST: edd434aaef4e700d07f0b7980a61b104e100c1d93e850fc260af98cc34a9b7a8
    COUNT_STAR: 2
SUM_TIMER_WAIT: 4322381000
MIN_TIMER_WAIT: 1686751000
AVG_TIMER_WAIT: 2161190000
MAX_TIMER_WAIT: 2635630000
 SUM_LOCK_TIME: 15000000
    SUM_ERRORS: 0
 SUM_ROWS_SENT: 78
*************************** 43. row ***************************
   SCHEMA_NAME: NULL
        DIGEST: 81f304f23c07a462c4453d3240908776088c81450c3c98d1cc01a42ec7c24b0f
    COUNT_STAR: 3
SUM_TIMER_WAIT: 13797955000
MIN_TIMER_WAIT: 1016635000
AVG_TIMER_WAIT: 4599318000
MAX_TIMER_WAIT: 11274185000
 SUM_LOCK_TIME: 6000000
    SUM_ERRORS: 0
 SUM_ROWS_SENT: 3
*************************** 44. row ***************************
   SCHEMA_NAME: NULL
        DIGEST: aefe18d1702362b120caeb37b4142209ad62139583774e0e3018adb7c3ba9440
    COUNT_STAR: 2
SUM_TIMER_WAIT: 11390385000
MIN_TIMER_WAIT: 1131795000
AVG_TIMER_WAIT: 5695192000
MAX_TIMER_WAIT: 10258590000
 SUM_LOCK_TIME: 4000000
    SUM_ERRORS: 0
 SUM_ROWS_SENT: 87


`
	perfEventsStatementsQueryMySQL = `
	SELECT
	    ifnull(SCHEMA_NAME, 'NONE') as SCHEMA_NAME,
	    DIGEST,
	    LEFT(DIGEST_TEXT, %d) as DIGEST_TEXT,
	    COUNT_STAR,
	    SUM_TIMER_WAIT,
	    SUM_LOCK_TIME,
	    SUM_CPU_TIME,
	    SUM_ERRORS,
	    SUM_WARNINGS,
	    SUM_ROWS_AFFECTED,
	    SUM_ROWS_SENT,
	    SUM_ROWS_EXAMINED,
	    SUM_CREATED_TMP_DISK_TABLES,
	    SUM_CREATED_TMP_TABLES,
	    SUM_SORT_MERGE_PASSES,
	    SUM_SORT_ROWS,
	    SUM_NO_INDEX_USED,
	    QUANTILE_95,
	    QUANTILE_99,
	    QUANTILE_999
	  FROM (
	    SELECT *
	    FROM performance_schema.events_statements_summary_by_digest
	    WHERE SCHEMA_NAME NOT IN ('mysql', 'performance_schema', 'information_schema')
	      AND LAST_SEEN > DATE_SUB(NOW(), INTERVAL %d SECOND)
	    ORDER BY LAST_SEEN DESC
	  )Q
	  GROUP BY
	    Q.SCHEMA_NAME,
	    Q.DIGEST,
	    Q.DIGEST_TEXT,
	    Q.COUNT_STAR,
	    Q.SUM_TIMER_WAIT,
	    Q.SUM_LOCK_TIME,
	    Q.SUM_CPU_TIME,
	    Q.SUM_ERRORS,
	    Q.SUM_WARNINGS,
	    Q.SUM_ROWS_AFFECTED,
	    Q.SUM_ROWS_SENT,
	    Q.SUM_ROWS_EXAMINED,
	    Q.SUM_CREATED_TMP_DISK_TABLES,
	    Q.SUM_CREATED_TMP_TABLES,
	    Q.SUM_SORT_MERGE_PASSES,
	    Q.SUM_SORT_ROWS,
	    Q.SUM_NO_INDEX_USED,
	    Q.QUANTILE_95,
	    Q.QUANTILE_99,
	    Q.QUANTILE_999
	  ORDER BY SUM_TIMER_WAIT DESC
	  LIMIT %d
	`
)

var (
	perfEventsStatementsLimit = kingpin.Flag(
		"collect.perf_schema.eventsstatements.limit",
		"Limit the number of events statements digests by response time",
	).Default("250").Int()
	perfEventsStatementsTimeLimit = kingpin.Flag(
		"collect.perf_schema.eventsstatements.timelimit",
		"Limit how old the 'last_seen' events statements can be, in seconds",
	).Default("86400").Int()
	perfEventsStatementsDigestTextLimit = kingpin.Flag(
		"collect.perf_schema.eventsstatements.digest_text_limit",
		"Maximum length of the normalized statement text",
	).Default("120").Int()
)

type ScrapePerfEventsStatements struct {
	instance mysql.Instance
	performanceSchemaEventsStatementsDesc
	performanceSchemaEventsStatementsTimeDesc
	performanceSchemaEventsStatementsLockTimeDesc
	performanceSchemaEventsStatementsCpuTimeDesc
	performanceSchemaEventsStatementsErrorsDesc
	performanceSchemaEventsStatementsWarningsDesc
	performanceSchemaEventsStatementsRowsAffectedDesc
	performanceSchemaEventsStatementsRowsSentDesc
	performanceSchemaEventsStatementsRowsExaminedDesc
	performanceSchemaEventsStatementsTmpTablesDesc
	performanceSchemaEventsStatementsTmpDiskTablesDesc
	performanceSchemaEventsStatementsSortMergePassesDesc
	performanceSchemaEventsStatementsSortRowsDesc
	performanceSchemaEventsStatementsNoIndexUsedDesc
	performanceSchemaEventsStatementsLatency
}

func init() {
	exporter.Register(
		NewScrapePerfEventsStatements())
}
func NewScrapePerfEventsStatements() *ScrapePerfEventsStatements {
	return &ScrapePerfEventsStatements{
		//instance:                                             instance,
		performanceSchemaEventsStatementsDesc:                *NewperformanceSchemaEventsStatementsDesc(),
		performanceSchemaEventsStatementsTimeDesc:            *NewperformanceSchemaEventsStatementsTimeDesc(),
		performanceSchemaEventsStatementsLockTimeDesc:        *NewperformanceSchemaEventsStatementsLockTimeDesc(),
		performanceSchemaEventsStatementsCpuTimeDesc:         *NewperformanceSchemaEventsStatementsCpuTimeDesc(),
		performanceSchemaEventsStatementsErrorsDesc:          *NewperformanceSchemaEventsStatementsErrorsDesc(),
		performanceSchemaEventsStatementsWarningsDesc:        *NewperformanceSchemaEventsStatementsWarningsDesc(),
		performanceSchemaEventsStatementsRowsAffectedDesc:    *NewperformanceSchemaEventsStatementsRowsAffectedDesc(),
		performanceSchemaEventsStatementsRowsSentDesc:        *NewperformanceSchemaEventsStatementsRowsSentDesc(),
		performanceSchemaEventsStatementsRowsExaminedDesc:    *NewperformanceSchemaEventsStatementsRowsExaminedDesc(),
		performanceSchemaEventsStatementsTmpTablesDesc:       *NewperformanceSchemaEventsStatementsTmpTablesDesc(),
		performanceSchemaEventsStatementsTmpDiskTablesDesc:   *NewperformanceSchemaEventsStatementsTmpDiskTablesDesc(),
		performanceSchemaEventsStatementsSortMergePassesDesc: *NewperformanceSchemaEventsStatementsSortMergePassesDesc(),
		performanceSchemaEventsStatementsSortRowsDesc:        *NewperformanceSchemaEventsStatementsSortRowsDesc(),
		performanceSchemaEventsStatementsNoIndexUsedDesc:     *NewperformanceSchemaEventsStatementsNoIndexUsedDesc(),
		performanceSchemaEventsStatementsLatency:             *NewperformanceSchemaEventsStatementsLatency(),
	}
}

func (qd ScrapePerfEventsStatements) Collect(ch chan<- prometheus.Metric) {
	qd.instance = *GetInstance()

	if err := qd.instance.Ping(); err != nil {
		logrus.Errorf("ping mysql instance error: %s", err)
		return
	}
	db := instance.GetDB()
	isMysql := instance.GetFlavor() == mysql.MySQL
	dbVersion := instance.GetVersion
	isNewDb := dbVersion().GreaterThan(semver.MustParse("8.0.28"))
	mysqlVersion8028 := isMysql && isNewDb
	perfQuery := perfEventsStatementsQuery
	if mysqlVersion8028 {
		perfQuery = perfEventsStatementsQueryMySQL
	}
	perfQuery = fmt.Sprintf(
		perfQuery,
		*perfEventsStatementsDigestTextLimit,
		*perfEventsStatementsTimeLimit,
		*perfEventsStatementsLimit,
	)
	perfSchemaEventsStatementsRows, err := db.Query(perfQuery)
	if err != nil {
		logrus.Error(err)
		return
	}
	defer perfSchemaEventsStatementsRows.Close()
	var (
		schemaName      string
		digest          string
		digestText      string
		count           uint64
		queryTime       uint64
		lockTime        uint64
		cpuTime         uint64
		errors          uint64
		warnings        uint64
		rowsAffected    uint64
		rowsSent        uint64
		rowsExamined    uint64
		tmpTables       uint64
		tmpDiskTables   uint64
		sortMergePasses uint64
		sortRows        uint64
		noIndexUsed     uint64
		quantile95      uint64
		quantile99      uint64
		quantile999     uint64
	)
	for perfSchemaEventsStatementsRows.Next() {
		var err error
		if mysqlVersion8028 {
			err = perfSchemaEventsStatementsRows.Scan(
				&schemaName, &digest, &digestText, &count, &queryTime, &lockTime, &cpuTime, &errors, &warnings, &rowsAffected, &rowsSent, &rowsExamined, &tmpDiskTables, &tmpTables, &sortMergePasses, &sortRows, &noIndexUsed, &quantile95, &quantile99, &quantile999,
			)
		} else {
			err = perfSchemaEventsStatementsRows.Scan(
				&schemaName, &digest, &digestText, &count, &queryTime, &errors, &warnings, &rowsAffected, &rowsSent, &rowsExamined, &tmpDiskTables, &tmpTables, &sortMergePasses, &sortRows, &noIndexUsed,
			)
		}
		if err != nil {
			logrus.Error(err)
			return
		}
		qd.performanceSchemaEventsStatementsErrorsDesc.Collect(
			ch,
			float64(count),
			[]string{
				schemaName,
				digest,
				digestText})
		qd.performanceSchemaEventsStatementsTimeDesc.Collect(
			ch,
			float64(queryTime)/picoSeconds,
			[]string{
				schemaName,
				digest,
				digestText})
		qd.performanceSchemaEventsStatementsLockTimeDesc.Collect(
			ch,
			float64(lockTime)/picoSeconds,
			[]string{
				schemaName,
				digest,
				digestText})

		qd.performanceSchemaEventsStatementsCpuTimeDesc.Collect(
			ch,
			float64(cpuTime)/picoSeconds,
			[]string{
				schemaName,
				digest,
				digestText})

		qd.performanceSchemaEventsStatementsErrorsDesc.Collect(
			ch,
			float64(errors),
			[]string{
				schemaName,
				digest,
				digestText})
		qd.performanceSchemaEventsStatementsWarningsDesc.Collect(
			ch,
			float64(warnings),
			[]string{
				schemaName,
				digest,
				digestText})

		qd.performanceSchemaEventsStatementsRowsAffectedDesc.Collect(
			ch,
			float64(rowsAffected),
			[]string{
				schemaName,
				digest,
				digestText})
		qd.performanceSchemaEventsStatementsRowsSentDesc.Collect(
			ch,
			float64(rowsSent),
			[]string{
				schemaName,
				digest,
				digestText})
		qd.performanceSchemaEventsStatementsRowsExaminedDesc.Collect(
			ch,
			float64(rowsExamined),
			[]string{
				schemaName,
				digest,
				digestText})

		qd.performanceSchemaEventsStatementsTmpTablesDesc.Collect(
			ch,
			float64(tmpTables),
			[]string{
				schemaName,
				digest,
				digestText})

		qd.performanceSchemaEventsStatementsTmpDiskTablesDesc.Collect(
			ch,
			float64(tmpDiskTables),
			[]string{
				schemaName,
				digest,
				digestText})

		qd.performanceSchemaEventsStatementsSortMergePassesDesc.Collect(
			ch,
			float64(sortMergePasses),
			[]string{
				schemaName,
				digest,
				digestText})

		qd.performanceSchemaEventsStatementsSortRowsDesc.Collect(
			ch,
			float64(sortRows),
			[]string{
				schemaName,
				digest,
				digestText})

		qd.performanceSchemaEventsStatementsNoIndexUsedDesc.Collect(
			ch,
			float64(noIndexUsed),
			[]string{
				schemaName,
				digest,
				digestText})

		qd.performanceSchemaEventsStatementsLatency.Collect(
			ch,
			float64(queryTime)/picoSeconds,
			[]string{
				schemaName,
				digest,
				digestText})
	}
}

type performanceSchemaEventsStatementsDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsDesc() *performanceSchemaEventsStatementsDesc {
	return &performanceSchemaEventsStatementsDesc{
		NewMetrics(
			"perf_schema_events_statements_total",
			"The total count of events statements by digest.",
			[]string{
				"schema",
				"digest",
				"digest_text"})}
}
func (qd *performanceSchemaEventsStatementsDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsTimeDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsTimeDesc() *performanceSchemaEventsStatementsTimeDesc {
	return &performanceSchemaEventsStatementsTimeDesc{
		NewMetrics(
			"perf_schema_events_statements_seconds_total",
			"The total time of events statements by digest.",
			[]string{
				"schema",
				"digest",
				"digest_text"})}
}
func (qd *performanceSchemaEventsStatementsTimeDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsLockTimeDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsLockTimeDesc() *performanceSchemaEventsStatementsLockTimeDesc {
	return &performanceSchemaEventsStatementsLockTimeDesc{
		NewMetrics(
			"perf_schema_events_statements_lock_time_seconds_total",
			"The total lock time of events statements by digest.",
			[]string{
				"schema",
				"digest",
				"digest_text"})}
}
func (qd *performanceSchemaEventsStatementsLockTimeDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsCpuTimeDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsCpuTimeDesc() *performanceSchemaEventsStatementsCpuTimeDesc {
	return &performanceSchemaEventsStatementsCpuTimeDesc{
		NewMetrics(
			"perf_schema_events_statements_cpu_time_seconds_total",
			"The total cpu time of events statements by digest.",
			[]string{
				"schema",
				"digest",
				"digest_text"})}
}
func (qd *performanceSchemaEventsStatementsCpuTimeDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsErrorsDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsErrorsDesc() *performanceSchemaEventsStatementsErrorsDesc {
	return &performanceSchemaEventsStatementsErrorsDesc{
		NewMetrics(
			"perf_schema_events_statements_errors_total",
			"The total errors of events statements by digest.",
			[]string{
				"schema",
				"digest",
				"digest_text"})}
}
func (qd *performanceSchemaEventsStatementsErrorsDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsWarningsDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsWarningsDesc() *performanceSchemaEventsStatementsWarningsDesc {
	return &performanceSchemaEventsStatementsWarningsDesc{
		NewMetrics(
			"perf_schema_events_statements_warnings_total",
			"The total warnings of events statements by digest.",
			[]string{
				"schema",
				"digest",
				"digest_text"})}
}
func (qd *performanceSchemaEventsStatementsWarningsDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsRowsAffectedDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsRowsAffectedDesc() *performanceSchemaEventsStatementsRowsAffectedDesc {
	return &performanceSchemaEventsStatementsRowsAffectedDesc{
		NewMetrics(
			"perf_schema_events_statements_rows_affected_total",
			"The total rows affected of events statements by digest.",
			[]string{
				"schema",
				"digest",
				"digest_text"})}
}

func (qd *performanceSchemaEventsStatementsRowsAffectedDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsRowsSentDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsRowsSentDesc() *performanceSchemaEventsStatementsRowsSentDesc {
	return &performanceSchemaEventsStatementsRowsSentDesc{
		NewMetrics(
			"perf_schema_events_statements_rows_sent_total",
			"The total rows sent of events statements by digest.",
			[]string{
				"schema",
				"digest",
				"digest_text"})}
}
func (qd *performanceSchemaEventsStatementsRowsSentDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsRowsExaminedDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsRowsExaminedDesc() *performanceSchemaEventsStatementsRowsExaminedDesc {
	return &performanceSchemaEventsStatementsRowsExaminedDesc{
		NewMetrics(
			"perf_schema_events_statements_rows_examined_total",
			"The total rows examined of events statements by digest.",
			[]string{
				"schema",
				"digest",
				"digest_text"})}
}
func (qd *performanceSchemaEventsStatementsRowsExaminedDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsTmpTablesDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsTmpTablesDesc() *performanceSchemaEventsStatementsTmpTablesDesc {
	return &performanceSchemaEventsStatementsTmpTablesDesc{
		NewMetrics(
			"perf_schema_events_statements_tmp_tables_total",
			"The total tmp tables of events statements by digest.",
			[]string{
				"schema",
				"digest",
				"digest_text"})}
}
func (qd *performanceSchemaEventsStatementsTmpTablesDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsTmpDiskTablesDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsTmpDiskTablesDesc() *performanceSchemaEventsStatementsTmpDiskTablesDesc {
	return &performanceSchemaEventsStatementsTmpDiskTablesDesc{
		NewMetrics(
			"perf_schema_events_statements_tmp_disk_tables_total",
			"The total tmp disk tables of events statements by digest.",
			[]string{
				"schema",
				"digest",
				"digest_text"})}
}
func (qd *performanceSchemaEventsStatementsTmpDiskTablesDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsSortMergePassesDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsSortMergePassesDesc() *performanceSchemaEventsStatementsSortMergePassesDesc {
	return &performanceSchemaEventsStatementsSortMergePassesDesc{
		NewMetrics(
			"perf_schema_events_statements_sort_merge_passes_total",
			"The total sort merge passes of events statements by digest.",
			[]string{
				"schema",
				"digest",
				"digest_text"})}
}
func (qd *performanceSchemaEventsStatementsSortMergePassesDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsSortRowsDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsSortRowsDesc() *performanceSchemaEventsStatementsSortRowsDesc {
	return &performanceSchemaEventsStatementsSortRowsDesc{
		NewMetrics(
			"perf_schema_events_statements_sort_rows_total",
			"The total sort rows of events statements by digest.",
			[]string{
				"schema",
				"digest",
				"digest_text"})}
}
func (qd *performanceSchemaEventsStatementsSortRowsDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsNoIndexUsedDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsNoIndexUsedDesc() *performanceSchemaEventsStatementsNoIndexUsedDesc {
	return &performanceSchemaEventsStatementsNoIndexUsedDesc{
		NewMetrics(
			"perf_schema_events_statements_no_index_used_total",
			"The total no index used of events statements by digest.",
			[]string{
				"schema",
				"digest",
				"digest_text"})}
}
func (qd *performanceSchemaEventsStatementsNoIndexUsedDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsLatency struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsLatency() *performanceSchemaEventsStatementsLatency {
	return &performanceSchemaEventsStatementsLatency{
		NewMetrics(
			"perf_schema_events_statements_latency_seconds_total",
			"The total latency of events statements by digest.",
			[]string{
				"schema",
				"digest",
				"digest_text"})}
}
func (qd *performanceSchemaEventsStatementsLatency) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

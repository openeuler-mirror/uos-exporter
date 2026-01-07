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


// TODO: implement functions

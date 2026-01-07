package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"mysqld_exporter/internal/exporter"
	"mysqld_exporter/internal/mysql"
)

const (
	perfEventsStatementsSumQuery = `
	SELECT
		SUM(COUNT_STAR)
		    AS SUM_COUNT_STAR,
		SUM(SUM_CREATED_TMP_DISK_TABLES)
		    AS SUM_SUM_CREATED_TMP_DISK_TABLES,
		SUM(SUM_CREATED_TMP_TABLES)
		    AS SUM_SUM_CREATED_TMP_TABLES,
		SUM(SUM_ERRORS) 
		    AS SUM_SUM_ERRORS,
		SUM(SUM_LOCK_TIME)
		    AS SUM_SUM_LOCK_TIME,
		SUM(SUM_NO_GOOD_INDEX_USED)
		    AS SUM_SUM_NO_GOOD_INDEX_USED,
		SUM(SUM_NO_INDEX_USED)
		    AS SUM_SUM_NO_INDEX_USED,
		SUM(SUM_ROWS_AFFECTED) 
		    AS SUM_SUM_ROWS_AFFECTED,
		SUM(SUM_ROWS_EXAMINED) 
		    AS SUM_SUM_ROWS_EXAMINED,
		SUM(SUM_ROWS_SENT)
		    AS SUM_SUM_ROWS_SENT,
		SUM(SUM_SELECT_FULL_JOIN)
		    AS SUM_SUM_SELECT_FULL_JOIN,
		SUM(SUM_SELECT_FULL_RANGE_JOIN) 
		    AS SUM_SUM_SELECT_FULL_RANGE_JOIN,
		SUM(SUM_SELECT_RANGE) 
		    AS SUM_SUM_SELECT_RANGE,
		SUM(SUM_SELECT_RANGE_CHECK) 
		    AS SUM_SUM_SELECT_RANGE_CHECK,
		SUM(SUM_SELECT_SCAN) 
		    AS SUM_SUM_SELECT_SCAN,
		SUM(SUM_SORT_MERGE_PASSES) 
		    AS SUM_SUM_SORT_MERGE_PASSES,
		SUM(SUM_SORT_RANGE) 
		    AS SUM_SUM_SORT_RANGE,
		SUM(SUM_SORT_ROWS)
		    AS SUM_SUM_SORT_ROWS,
		SUM(SUM_SORT_SCAN)
		    AS SUM_SUM_SORT_SCAN,
		SUM(SUM_TIMER_WAIT) 
		    AS SUM_SUM_TIMER_WAIT,
		SUM(SUM_WARNINGS) 
		    AS SUM_SUM_WARNINGS
	FROM performance_schema.events_statements_summary_by_digest;
	`
	perfEventsStatementsSumResult = `
MySQL [(none)]> desc performance_schema.events_statements_summary_by_digest;
+-----------------------------+-----------------+------+-----+---------+-------+
| Field                       | Type            | Null | Key | Default | Extra |
+-----------------------------+-----------------+------+-----+---------+-------+
| SCHEMA_NAME                 | varchar(64)     | YES  | MUL | NULL    |       |
| DIGEST                      | varchar(64)     | YES  |     | NULL    |       |
| DIGEST_TEXT                 | longtext        | YES  |     | NULL    |       |
| COUNT_STAR                  | bigint unsigned | NO   |     | NULL    |       |
| SUM_TIMER_WAIT              | bigint unsigned | NO   |     | NULL    |       |
| MIN_TIMER_WAIT              | bigint unsigned | NO   |     | NULL    |       |
| AVG_TIMER_WAIT              | bigint unsigned | NO   |     | NULL    |       |
| MAX_TIMER_WAIT              | bigint unsigned | NO   |     | NULL    |       |
| SUM_LOCK_TIME               | bigint unsigned | NO   |     | NULL    |       |
| SUM_ERRORS                  | bigint unsigned | NO   |     | NULL    |       |
| SUM_WARNINGS                | bigint unsigned | NO   |     | NULL    |       |
| SUM_ROWS_AFFECTED           | bigint unsigned | NO   |     | NULL    |       |
| SUM_ROWS_SENT               | bigint unsigned | NO   |     | NULL    |       |
| SUM_ROWS_EXAMINED           | bigint unsigned | NO   |     | NULL    |       |
| SUM_CREATED_TMP_DISK_TABLES | bigint unsigned | NO   |     | NULL    |       |
| SUM_CREATED_TMP_TABLES      | bigint unsigned | NO   |     | NULL    |       |
| SUM_SELECT_FULL_JOIN        | bigint unsigned | NO   |     | NULL    |       |
| SUM_SELECT_FULL_RANGE_JOIN  | bigint unsigned | NO   |     | NULL    |       |
| SUM_SELECT_RANGE            | bigint unsigned | NO   |     | NULL    |       |
| SUM_SELECT_RANGE_CHECK      | bigint unsigned | NO   |     | NULL    |       |
| SUM_SELECT_SCAN             | bigint unsigned | NO   |     | NULL    |       |
| SUM_SORT_MERGE_PASSES       | bigint unsigned | NO   |     | NULL    |       |
| SUM_SORT_RANGE              | bigint unsigned | NO   |     | NULL    |       |
| SUM_SORT_ROWS               | bigint unsigned | NO   |     | NULL    |       |
| SUM_SORT_SCAN               | bigint unsigned | NO   |     | NULL    |       |
| SUM_NO_INDEX_USED           | bigint unsigned | NO   |     | NULL    |       |
| SUM_NO_GOOD_INDEX_USED      | bigint unsigned | NO   |     | NULL    |       |
| SUM_CPU_TIME                | bigint unsigned | NO   |     | NULL    |       |
| MAX_CONTROLLED_MEMORY       | bigint unsigned | NO   |     | NULL    |       |
| MAX_TOTAL_MEMORY            | bigint unsigned | NO   |     | NULL    |       |
| COUNT_SECONDARY             | bigint unsigned | NO   |     | NULL    |       |
| FIRST_SEEN                  | timestamp(6)    | NO   |     | NULL    |       |
| LAST_SEEN                   | timestamp(6)    | NO   |     | NULL    |       |
| QUANTILE_95                 | bigint unsigned | NO   |     | NULL    |       |
| QUANTILE_99                 | bigint unsigned | NO   |     | NULL    |       |
| QUANTILE_999                | bigint unsigned | NO   |     | NULL    |       |
| QUERY_SAMPLE_TEXT           | longtext        | YES  |     | NULL    |       |
| QUERY_SAMPLE_SEEN           | timestamp(6)    | NO   |     | NULL    |       |
| QUERY_SAMPLE_TIMER_WAIT     | bigint unsigned | NO   |     | NULL    |       |
+-----------------------------+-----------------+------+-----+---------+-------+
39 rows in set (0.002 sec)
*************************** 1. row ***************************
                 SUM_COUNT_STAR: 123
SUM_SUM_CREATED_TMP_DISK_TABLES: 0
     SUM_SUM_CREATED_TMP_TABLES: 127
                 SUM_SUM_ERRORS: 3
              SUM_SUM_LOCK_TIME: 425000000
     SUM_SUM_NO_GOOD_INDEX_USED: 0
          SUM_SUM_NO_INDEX_USED: 65
          SUM_SUM_ROWS_AFFECTED: 0
          SUM_SUM_ROWS_EXAMINED: 9097
              SUM_SUM_ROWS_SENT: 4993
       SUM_SUM_SELECT_FULL_JOIN: 3
 SUM_SUM_SELECT_FULL_RANGE_JOIN: 0
           SUM_SUM_SELECT_RANGE: 0
     SUM_SUM_SELECT_RANGE_CHECK: 0
            SUM_SUM_SELECT_SCAN: 135
      SUM_SUM_SORT_MERGE_PASSES: 0
             SUM_SUM_SORT_RANGE: 0
              SUM_SUM_SORT_ROWS: 693
              SUM_SUM_SORT_SCAN: 88
             SUM_SUM_TIMER_WAIT: 310355086000
               SUM_SUM_WARNINGS: 13
1 row in set (0.001 sec)
`
)

type ScrapePerfEventsStatementsSum struct {
	instance mysql.Instance
	performanceSchemaEventsStatementsSumTotalDesc
	performanceSchemaEventsStatementsSumCreatedTmpDiskTablesDesc
	performanceSchemaEventsStatementsSumCreatedTmpTablesDesc
	performanceSchemaEventsStatementsSumErrorsDesc
	performanceSchemaEventsStatementsSumLockTimeDesc
	performanceSchemaEventsStatementsSumNoGoodIndexUsedDesc
	performanceSchemaEventsStatementsSumNoIndexUsedDesc
	performanceSchemaEventsStatementsSumRowsAffectedDesc
	performanceSchemaEventsStatementsSumRowsExaminedDesc
	performanceSchemaEventsStatementsSumRowsSentDesc
	performanceSchemaEventsStatementsSumSelectFullJoinDesc
	performanceSchemaEventsStatementsSumSelectFullRangeJoinDesc
	performanceSchemaEventsStatementsSumSelectRangeDesc
	performanceSchemaEventsStatementsSumSelectRangeCheckDesc
	performanceSchemaEventsStatementsSumSelectScanDesc
	performanceSchemaEventsStatementsSumSortMergePassesDesc
	performanceSchemaEventsStatementsSumSortRangeDesc
	performanceSchemaEventsStatementsSumSortRowsDesc
	performanceSchemaEventsStatementsSumSortScanDesc
	performanceSchemaEventsStatementsSumTimerWaitDesc
	performanceSchemaEventsStatementsSumWarningsDesc
}


// TODO: implement functions

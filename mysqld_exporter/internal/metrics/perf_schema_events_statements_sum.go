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

func init() {
	exporter.Register(
		NewScrapePerfEventsStatementsSum())
}
func NewScrapePerfEventsStatementsSum() *ScrapePerfEventsStatementsSum {
	return &ScrapePerfEventsStatementsSum{
		//instance: instance,
		performanceSchemaEventsStatementsSumTotalDesc:                *NewperformanceSchemaEventsStatementsSumTotalDesc(),
		performanceSchemaEventsStatementsSumCreatedTmpDiskTablesDesc: *NewperformanceSchemaEventsStatementsSumCreatedTmpDiskTablesDesc(),
		performanceSchemaEventsStatementsSumCreatedTmpTablesDesc:     *NewperformanceSchemaEventsStatementsSumCreatedTmpTablesDesc(),
		performanceSchemaEventsStatementsSumErrorsDesc:               *NewperformanceSchemaEventsStatementsSumErrorsDesc(),
		performanceSchemaEventsStatementsSumLockTimeDesc:             *NewperformanceSchemaEventsStatementsSumLockTimeDesc(),
		performanceSchemaEventsStatementsSumNoGoodIndexUsedDesc:      *NewperformanceSchemaEventsStatementsSumNoGoodIndexUsedDesc(),
		performanceSchemaEventsStatementsSumNoIndexUsedDesc:          *NewperformanceSchemaEventsStatementsSumNoIndexUsedDesc(),
		performanceSchemaEventsStatementsSumRowsAffectedDesc:         *NewperformanceSchemaEventsStatementsSumRowsAffectedDesc(),
		performanceSchemaEventsStatementsSumRowsExaminedDesc:         *NewperformanceSchemaEventsStatementsSumRowsExaminedDesc(),
		performanceSchemaEventsStatementsSumRowsSentDesc:             *NewperformanceSchemaEventsStatementsSumRowsSentDesc(),
		performanceSchemaEventsStatementsSumSelectFullJoinDesc:       *NewperformanceSchemaEventsStatementsSumSelectFullJoinDesc(),
		performanceSchemaEventsStatementsSumSelectFullRangeJoinDesc:  *NewperformanceSchemaEventsStatementsSumSelectFullRangeJoinDesc(),
		performanceSchemaEventsStatementsSumSelectRangeDesc:          *NewperformanceSchemaEventsStatementsSumSelectRangeDesc(),
		performanceSchemaEventsStatementsSumSelectRangeCheckDesc:     *NewperformanceSchemaEventsStatementsSumSelectRangeCheckDesc(),
		performanceSchemaEventsStatementsSumSelectScanDesc:           *NewperformanceSchemaEventsStatementsSumSelectScanDesc(),
		performanceSchemaEventsStatementsSumSortMergePassesDesc:      *NewperformanceSchemaEventsStatementsSumSortMergePassesDesc(),
		performanceSchemaEventsStatementsSumSortRangeDesc:            *NewperformanceSchemaEventsStatementsSumSortRangeDesc(),
		performanceSchemaEventsStatementsSumSortRowsDesc:             *NewperformanceSchemaEventsStatementsSumSortRowsDesc(),
		performanceSchemaEventsStatementsSumSortScanDesc:             *NewperformanceSchemaEventsStatementsSumSortScanDesc(),
		performanceSchemaEventsStatementsSumTimerWaitDesc:            *NewperformanceSchemaEventsStatementsSumTimerWaitDesc(),
		performanceSchemaEventsStatementsSumWarningsDesc:             *NewperformanceSchemaEventsStatementsSumWarningsDesc(),
	}
}

func (qd ScrapePerfEventsStatementsSum) Collect(ch chan<- prometheus.Metric) {
	qd.instance = *GetInstance()

	if err := qd.instance.Ping(); err != nil {
		logrus.Errorf("ping mysql instance error: %s", err)
		return
	}
	db := instance.GetDB()
	rows, err := db.Query(perfEventsStatementsSumQuery)
	if err != nil {
		logrus.Error(err)
		return
	}
	defer rows.Close()
	var (
		total                uint64
		createdTmpDiskTables uint64
		createdTmpTables     uint64
		errors               uint64
		lockTime             uint64
		noGoodIndexUsed      uint64
		noIndexUsed          uint64
		rowsAffected         uint64
		rowsExamined         uint64
		rowsSent             uint64
		selectFullJoin       uint64
		selectFullRangeJoin  uint64
		selectRange          uint64
		selectRangeCheck     uint64
		selectScan           uint64
		sortMergePasses      uint64
		sortRange            uint64
		sortRows             uint64
		sortScan             uint64
		timerWait            uint64
		warnings             uint64
	)
	for rows.Next() {
		err = rows.Scan(
			&total,
			&createdTmpDiskTables,
			&createdTmpTables,
			&errors,
			&lockTime,
			&noGoodIndexUsed,
			&noIndexUsed,
			&rowsAffected,
			&rowsExamined,
			&rowsSent,
			&selectFullJoin,
			&selectFullRangeJoin,
			&selectRange,
			&selectRangeCheck,
			&selectScan,
			&sortMergePasses,
			&sortRange,
			&sortRows,
			&sortScan,
			&timerWait,
			&warnings,
		)
		if err != nil {
			logrus.Error(err)
			return
		}
		qd.performanceSchemaEventsStatementsSumTotalDesc.Collect(
			ch,
			float64(total),
			nil)
		qd.performanceSchemaEventsStatementsSumCreatedTmpDiskTablesDesc.Collect(
			ch,
			float64(createdTmpDiskTables),
			nil)
		qd.performanceSchemaEventsStatementsSumCreatedTmpTablesDesc.Collect(
			ch,
			float64(createdTmpTables),
			nil)
		qd.performanceSchemaEventsStatementsSumErrorsDesc.Collect(
			ch,
			float64(errors),
			nil)
		qd.performanceSchemaEventsStatementsSumLockTimeDesc.Collect(
			ch,
			float64(lockTime),
			nil)
		qd.performanceSchemaEventsStatementsSumNoGoodIndexUsedDesc.Collect(
			ch,
			float64(noGoodIndexUsed),
			nil)
		qd.performanceSchemaEventsStatementsSumNoIndexUsedDesc.Collect(
			ch,
			float64(noIndexUsed),
			nil)
		qd.performanceSchemaEventsStatementsSumRowsAffectedDesc.Collect(
			ch,
			float64(rowsAffected),
			nil)
		qd.performanceSchemaEventsStatementsSumRowsExaminedDesc.Collect(
			ch,
			float64(rowsExamined),
			nil)
		qd.performanceSchemaEventsStatementsSumRowsSentDesc.Collect(
			ch,
			float64(rowsSent),
			nil)
		qd.performanceSchemaEventsStatementsSumSelectFullJoinDesc.Collect(
			ch,
			float64(selectFullJoin),
			nil)
		qd.performanceSchemaEventsStatementsSumSelectFullRangeJoinDesc.Collect(
			ch,
			float64(selectFullRangeJoin),
			nil)

		qd.performanceSchemaEventsStatementsSumSelectRangeDesc.Collect(
			ch,
			float64(selectRange),
			nil)
		qd.performanceSchemaEventsStatementsSumSelectRangeCheckDesc.Collect(
			ch,
			float64(selectRangeCheck),
			nil)
		qd.performanceSchemaEventsStatementsSumSelectScanDesc.Collect(
			ch,
			float64(selectScan),
			nil)
		qd.performanceSchemaEventsStatementsSumSortMergePassesDesc.Collect(
			ch,
			float64(sortMergePasses),
			nil)
		qd.performanceSchemaEventsStatementsSumSortRangeDesc.Collect(
			ch,
			float64(sortRange),
			nil)
		qd.performanceSchemaEventsStatementsSumSortRowsDesc.Collect(
			ch,
			float64(sortRows),
			nil)
		qd.performanceSchemaEventsStatementsSumSortScanDesc.Collect(
			ch,
			float64(sortScan),
			nil)
		qd.performanceSchemaEventsStatementsSumTimerWaitDesc.Collect(
			ch,
			float64(timerWait)/picoSeconds,
			nil)
		qd.performanceSchemaEventsStatementsSumWarningsDesc.Collect(
			ch,
			float64(warnings),
			nil)

	}
}

type performanceSchemaEventsStatementsSumTotalDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsSumTotalDesc() *performanceSchemaEventsStatementsSumTotalDesc {
	return &performanceSchemaEventsStatementsSumTotalDesc{
		NewMetrics(
			"perf_schema_events_statements_sum_total",
			"The total count of events statements.",
			nil)}
}
func (qd *performanceSchemaEventsStatementsSumTotalDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsSumCreatedTmpDiskTablesDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsSumCreatedTmpDiskTablesDesc() *performanceSchemaEventsStatementsSumCreatedTmpDiskTablesDesc {
	return &performanceSchemaEventsStatementsSumCreatedTmpDiskTablesDesc{
		NewMetrics(
			"perf_schema_events_statements_sum_created_tmp_disk_tables",
			"The number of on-disk temporary tables created.",
			nil)}
}
func (qd *performanceSchemaEventsStatementsSumCreatedTmpDiskTablesDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsSumCreatedTmpTablesDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsSumCreatedTmpTablesDesc() *performanceSchemaEventsStatementsSumCreatedTmpTablesDesc {
	return &performanceSchemaEventsStatementsSumCreatedTmpTablesDesc{
		NewMetrics(
			"perf_schema_events_statements_sum_created_tmp_tables",
			"The number of temporary tables created.",
			nil)}
}
func (qd *performanceSchemaEventsStatementsSumCreatedTmpTablesDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsSumErrorsDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsSumErrorsDesc() *performanceSchemaEventsStatementsSumErrorsDesc {
	return &performanceSchemaEventsStatementsSumErrorsDesc{
		NewMetrics(
			"perf_schema_events_statements_sum_errors",
			"Number of errors.",
			nil)}
}
func (qd *performanceSchemaEventsStatementsSumErrorsDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsSumLockTimeDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsSumLockTimeDesc() *performanceSchemaEventsStatementsSumLockTimeDesc {
	return &performanceSchemaEventsStatementsSumLockTimeDesc{
		NewMetrics(
			"perf_schema_events_statements_sum_lock_time",
			"Time in picoseconds spent waiting for locks.",
			nil)}
}
func (qd *performanceSchemaEventsStatementsSumLockTimeDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsSumNoGoodIndexUsedDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsSumNoGoodIndexUsedDesc() *performanceSchemaEventsStatementsSumNoGoodIndexUsedDesc {
	return &performanceSchemaEventsStatementsSumNoGoodIndexUsedDesc{
		NewMetrics(
			"perf_schema_events_statements_sum_no_good_index_used",
			"Number of times no good index was found.",
			nil)}
}
func (qd *performanceSchemaEventsStatementsSumNoGoodIndexUsedDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsSumNoIndexUsedDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsSumNoIndexUsedDesc() *performanceSchemaEventsStatementsSumNoIndexUsedDesc {
	return &performanceSchemaEventsStatementsSumNoIndexUsedDesc{
		NewMetrics(
			"perf_schema_events_statements_sum_no_index_used",
			"Number of times no index was found.",
			nil)}
}
func (qd *performanceSchemaEventsStatementsSumNoIndexUsedDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsSumRowsAffectedDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsSumRowsAffectedDesc() *performanceSchemaEventsStatementsSumRowsAffectedDesc {
	return &performanceSchemaEventsStatementsSumRowsAffectedDesc{
		NewMetrics(
			"perf_schema_events_statements_sum_rows_affected",
			"Number of rows affected by statements.",
			nil)}
}
func (qd *performanceSchemaEventsStatementsSumRowsAffectedDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsSumRowsExaminedDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsSumRowsExaminedDesc() *performanceSchemaEventsStatementsSumRowsExaminedDesc {
	return &performanceSchemaEventsStatementsSumRowsExaminedDesc{
		NewMetrics(
			"perf_schema_events_statements_sum_rows_examined",
			"Number of rows read during statements' execution.",
			nil)}
}
func (qd *performanceSchemaEventsStatementsSumRowsExaminedDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsSumRowsSentDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsSumRowsSentDesc() *performanceSchemaEventsStatementsSumRowsSentDesc {
	return &performanceSchemaEventsStatementsSumRowsSentDesc{
		NewMetrics(
			"perf_schema_events_statements_sum_rows_sent",
			"Number of rows returned.",
			nil)}
}
func (qd *performanceSchemaEventsStatementsSumRowsSentDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsSumSelectFullJoinDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsSumSelectFullJoinDesc() *performanceSchemaEventsStatementsSumSelectFullJoinDesc {
	return &performanceSchemaEventsStatementsSumSelectFullJoinDesc{
		NewMetrics(
			"perf_schema_events_statements_sum_select_full_join",
			"Number of joins performed by statements which did not use an index.",
			nil)}
}
func (qd *performanceSchemaEventsStatementsSumSelectFullJoinDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsSumSelectFullRangeJoinDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsSumSelectFullRangeJoinDesc() *performanceSchemaEventsStatementsSumSelectFullRangeJoinDesc {
	return &performanceSchemaEventsStatementsSumSelectFullRangeJoinDesc{
		NewMetrics(
			"perf_schema_events_statements_sum_select_full_range_join",
			"Number of joins performed by statements which used a range search of the first table.",
			nil)}
}
func (qd *performanceSchemaEventsStatementsSumSelectFullRangeJoinDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsSumSelectRangeDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsSumSelectRangeDesc() *performanceSchemaEventsStatementsSumSelectRangeDesc {
	return &performanceSchemaEventsStatementsSumSelectRangeDesc{
		NewMetrics(
			"perf_schema_events_statements_sum_select_range",
			"Number of joins performed by statements which used a range of the first table.",
			nil)}
}
func (qd *performanceSchemaEventsStatementsSumSelectRangeDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsSumSelectRangeCheckDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsSumSelectRangeCheckDesc() *performanceSchemaEventsStatementsSumSelectRangeCheckDesc {
	return &performanceSchemaEventsStatementsSumSelectRangeCheckDesc{
		NewMetrics(
			"perf_schema_events_statements_sum_select_range_check",
			"Number of joins without keys performed by statements that check for key usage after each row.",
			nil)}
}
func (qd *performanceSchemaEventsStatementsSumSelectRangeCheckDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsSumSelectScanDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsSumSelectScanDesc() *performanceSchemaEventsStatementsSumSelectScanDesc {
	return &performanceSchemaEventsStatementsSumSelectScanDesc{
		NewMetrics(
			"perf_schema_events_statements_sum_select_scan",
			"Number of joins performed by statements which used a full scan of the first table.",
			nil)}
}
func (qd *performanceSchemaEventsStatementsSumSelectScanDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsSumSortMergePassesDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsSumSortMergePassesDesc() *performanceSchemaEventsStatementsSumSortMergePassesDesc {
	return &performanceSchemaEventsStatementsSumSortMergePassesDesc{
		NewMetrics(
			"perf_schema_events_statements_sum_sort_merge_passes",
			"Number of merge passes by the sort algorithm performed by statements.",
			nil)}
}
func (qd *performanceSchemaEventsStatementsSumSortMergePassesDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsSumSortRangeDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsSumSortRangeDesc() *performanceSchemaEventsStatementsSumSortRangeDesc {
	return &performanceSchemaEventsStatementsSumSortRangeDesc{
		NewMetrics(
			"perf_schema_events_statements_sum_sort_range",
			"Number of sorts performed by statements which used a range.",
			nil)}
}
func (qd *performanceSchemaEventsStatementsSumSortRangeDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsSumSortRowsDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsSumSortRowsDesc() *performanceSchemaEventsStatementsSumSortRowsDesc {
	return &performanceSchemaEventsStatementsSumSortRowsDesc{
		NewMetrics(
			"perf_schema_events_statements_sum_sort_rows",
			"Number of rows sorted.",
			nil)}
}
func (qd *performanceSchemaEventsStatementsSumSortRowsDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsSumSortScanDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsSumSortScanDesc() *performanceSchemaEventsStatementsSumSortScanDesc {
	return &performanceSchemaEventsStatementsSumSortScanDesc{
		NewMetrics(
			"perf_schema_events_statements_sum_sort_scan",
			"Number of sorts performed by statements which used a full table scan.",
			nil)}
}
func (qd *performanceSchemaEventsStatementsSumSortScanDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsSumTimerWaitDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsSumTimerWaitDesc() *performanceSchemaEventsStatementsSumTimerWaitDesc {
	return &performanceSchemaEventsStatementsSumTimerWaitDesc{
		NewMetrics(
			"perf_schema_events_statements_sum_timer_wait",
			"Total wait time of the summarized events that are timed.",
			nil)}
}
func (qd *performanceSchemaEventsStatementsSumTimerWaitDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaEventsStatementsSumWarningsDesc struct {
	*baseMetrics
}

func NewperformanceSchemaEventsStatementsSumWarningsDesc() *performanceSchemaEventsStatementsSumWarningsDesc {
	return &performanceSchemaEventsStatementsSumWarningsDesc{
		NewMetrics(
			"perf_schema_events_statements_sum_warnings",
			"Number of warnings.",
			nil)}
}
func (qd *performanceSchemaEventsStatementsSumWarningsDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

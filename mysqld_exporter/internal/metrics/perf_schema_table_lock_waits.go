package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"mysqld_exporter/internal/exporter"
	"mysqld_exporter/internal/mysql"
)

const (
	picoSeconds = 1e12

	PerfSchemaTableLockWaits = "perf_schema_table_lock_waits"
	perfTableLockWaitsQuery  = `
	SELECT
	    OBJECT_SCHEMA,
	    OBJECT_NAME,
	    COUNT_READ_NORMAL,
	    COUNT_READ_WITH_SHARED_LOCKS,
	    COUNT_READ_HIGH_PRIORITY,
	    COUNT_READ_NO_INSERT,
	    COUNT_READ_EXTERNAL,
	    COUNT_WRITE_ALLOW_WRITE,
	    COUNT_WRITE_CONCURRENT_INSERT,
	    COUNT_WRITE_LOW_PRIORITY,
	    COUNT_WRITE_NORMAL,
	    COUNT_WRITE_EXTERNAL,
	    SUM_TIMER_READ_NORMAL,
	    SUM_TIMER_READ_WITH_SHARED_LOCKS,
	    SUM_TIMER_READ_HIGH_PRIORITY,
	    SUM_TIMER_READ_NO_INSERT,
	    SUM_TIMER_READ_EXTERNAL,
	    SUM_TIMER_WRITE_ALLOW_WRITE,
	    SUM_TIMER_WRITE_CONCURRENT_INSERT,
	    SUM_TIMER_WRITE_LOW_PRIORITY,
	    SUM_TIMER_WRITE_NORMAL,
	    SUM_TIMER_WRITE_EXTERNAL
	  FROM performance_schema.table_lock_waits_summary_by_table
	  WHERE OBJECT_SCHEMA NOT IN ('mysql', 'performance_schema', 'information_schema')
	`
	PerfSchemaTableLockWaitsResult = `*************************** 60. row ***************************
                    OBJECT_SCHEMA: mysql
                      OBJECT_NAME: replication_group_configuration_version
                COUNT_READ_NORMAL: 0
     COUNT_READ_WITH_SHARED_LOCKS: 0
         COUNT_READ_HIGH_PRIORITY: 0
             COUNT_READ_NO_INSERT: 0
              COUNT_READ_EXTERNAL: 0
          COUNT_WRITE_ALLOW_WRITE: 0
    COUNT_WRITE_CONCURRENT_INSERT: 0
         COUNT_WRITE_LOW_PRIORITY: 0
               COUNT_WRITE_NORMAL: 0
             COUNT_WRITE_EXTERNAL: 0
            SUM_TIMER_READ_NORMAL: 0
 SUM_TIMER_READ_WITH_SHARED_LOCKS: 0
     SUM_TIMER_READ_HIGH_PRIORITY: 0
         SUM_TIMER_READ_NO_INSERT: 0
          SUM_TIMER_READ_EXTERNAL: 0
      SUM_TIMER_WRITE_ALLOW_WRITE: 0
SUM_TIMER_WRITE_CONCURRENT_INSERT: 0
     SUM_TIMER_WRITE_LOW_PRIORITY: 0
           SUM_TIMER_WRITE_NORMAL: 0
         SUM_TIMER_WRITE_EXTERNAL: 0
*************************** 61. row ***************************
                    OBJECT_SCHEMA: mysql
                      OBJECT_NAME: replication_group_member_actions
                COUNT_READ_NORMAL: 0
     COUNT_READ_WITH_SHARED_LOCKS: 0
         COUNT_READ_HIGH_PRIORITY: 0
             COUNT_READ_NO_INSERT: 0
              COUNT_READ_EXTERNAL: 0
          COUNT_WRITE_ALLOW_WRITE: 0
    COUNT_WRITE_CONCURRENT_INSERT: 0
         COUNT_WRITE_LOW_PRIORITY: 0
               COUNT_WRITE_NORMAL: 0
             COUNT_WRITE_EXTERNAL: 0
            SUM_TIMER_READ_NORMAL: 0
 SUM_TIMER_READ_WITH_SHARED_LOCKS: 0
     SUM_TIMER_READ_HIGH_PRIORITY: 0
         SUM_TIMER_READ_NO_INSERT: 0
          SUM_TIMER_READ_EXTERNAL: 0
      SUM_TIMER_WRITE_ALLOW_WRITE: 0
SUM_TIMER_WRITE_CONCURRENT_INSERT: 0
     SUM_TIMER_WRITE_LOW_PRIORITY: 0
           SUM_TIMER_WRITE_NORMAL: 0
         SUM_TIMER_WRITE_EXTERNAL: 0
*************************** 62. row ***************************
                    OBJECT_SCHEMA: mysql
                      OBJECT_NAME: slow_log
                COUNT_READ_NORMAL: 0
     COUNT_READ_WITH_SHARED_LOCKS: 0
         COUNT_READ_HIGH_PRIORITY: 0
             COUNT_READ_NO_INSERT: 0
              COUNT_READ_EXTERNAL: 0
          COUNT_WRITE_ALLOW_WRITE: 0
    COUNT_WRITE_CONCURRENT_INSERT: 0
         COUNT_WRITE_LOW_PRIORITY: 0
               COUNT_WRITE_NORMAL: 0
             COUNT_WRITE_EXTERNAL: 0
            SUM_TIMER_READ_NORMAL: 0
 SUM_TIMER_READ_WITH_SHARED_LOCKS: 0
     SUM_TIMER_READ_HIGH_PRIORITY: 0
         SUM_TIMER_READ_NO_INSERT: 0
          SUM_TIMER_READ_EXTERNAL: 0
      SUM_TIMER_WRITE_ALLOW_WRITE: 0
SUM_TIMER_WRITE_CONCURRENT_INSERT: 0
     SUM_TIMER_WRITE_LOW_PRIORITY: 0
           SUM_TIMER_WRITE_NORMAL: 0
         SUM_TIMER_WRITE_EXTERNAL: 0
*************************** 63. row ***************************
                    OBJECT_SCHEMA: mysql
                      OBJECT_NAME: column_statistics
                COUNT_READ_NORMAL: 0
     COUNT_READ_WITH_SHARED_LOCKS: 0
         COUNT_READ_HIGH_PRIORITY: 0
             COUNT_READ_NO_INSERT: 0
              COUNT_READ_EXTERNAL: 0
          COUNT_WRITE_ALLOW_WRITE: 0
    COUNT_WRITE_CONCURRENT_INSERT: 0
         COUNT_WRITE_LOW_PRIORITY: 0
               COUNT_WRITE_NORMAL: 0
             COUNT_WRITE_EXTERNAL: 0
            SUM_TIMER_READ_NORMAL: 0
 SUM_TIMER_READ_WITH_SHARED_LOCKS: 0
     SUM_TIMER_READ_HIGH_PRIORITY: 0
         SUM_TIMER_READ_NO_INSERT: 0
          SUM_TIMER_READ_EXTERNAL: 0
      SUM_TIMER_WRITE_ALLOW_WRITE: 0
SUM_TIMER_WRITE_CONCURRENT_INSERT: 0
     SUM_TIMER_WRITE_LOW_PRIORITY: 0
           SUM_TIMER_WRITE_NORMAL: 0
         SUM_TIMER_WRITE_EXTERNAL: 0
63 rows in set (0.025 sec)
`
)

func init() {
	exporter.Register(
		NewScrapePerfTableLockWaits())
}

type SysScrapePerfTableLockWaits struct {
	instance mysql.Instance
	performanceSchemaSQLTableLockWaitsDesc
	performanceSchemaExternalTableLockWaitsDesc
	performanceSchemaSQLTableLockWaitsTimeDesc
	performanceSchemaExternalTableLockWaitsTimeDesc
}

func NewScrapePerfTableLockWaits() *SysScrapePerfTableLockWaits {
	return &SysScrapePerfTableLockWaits{
		//instance:                                        instance,
		performanceSchemaSQLTableLockWaitsDesc:          *newperformanceSchemaSQLTableLockWaitsDesc(),
		performanceSchemaExternalTableLockWaitsDesc:     *newperformanceSchemaExternalTableLockWaitsDesc(),
		performanceSchemaSQLTableLockWaitsTimeDesc:      *newperformanceSchemaSQLTableLockWaitsTimeDesc(),
		performanceSchemaExternalTableLockWaitsTimeDesc: *newperformanceSchemaExternalTableLockWaitsTimeDesc(),
	}
}
func (qd *SysScrapePerfTableLockWaits) Collect(ch chan<- prometheus.Metric) {
	qd.instance = *GetInstance()
	logrus.Info("Start collecting SysScrapePerfTableLockWaits metrics")
	db := instance.GetDB()
	rows, err := db.Query(perfTableLockWaitsQuery)
	if err != nil {
		logrus.Error(err)
		return
	}
	defer rows.Close()
	var (
		objectSchema               string
		objectName                 string
		countReadNormal            uint64
		countReadWithSharedLocks   uint64
		countReadHighPriority      uint64
		countReadNoInsert          uint64
		countReadExternal          uint64
		countWriteAllowWrite       uint64
		countWriteConcurrentInsert uint64
		countWriteLowPriority      uint64
		countWriteNormal           uint64
		countWriteExternal         uint64
		timeReadNormal             uint64
		timeReadWithSharedLocks    uint64
		timeReadHighPriority       uint64
		timeReadNoInsert           uint64
		timeReadExternal           uint64
		timeWriteAllowWrite        uint64
		timeWriteConcurrentInsert  uint64
		timeWriteLowPriority       uint64
		timeWriteNormal            uint64
		timeWriteExternal          uint64
	)
	for rows.Next() {
		err = rows.Scan(
			&objectSchema,
			&objectName,
			&countReadNormal,
			&countReadWithSharedLocks,
			&countReadHighPriority,
			&countReadNoInsert,
			&countReadExternal,
			&countWriteAllowWrite,
			&countWriteConcurrentInsert,
			&countWriteLowPriority,
			&countWriteNormal,
			&countWriteExternal,
			&timeReadNormal,
			&timeReadWithSharedLocks,
			&timeReadHighPriority,
			&timeReadNoInsert,
			&timeReadExternal,
			&timeWriteAllowWrite,
			&timeWriteConcurrentInsert,
			&timeWriteLowPriority,
			&timeWriteNormal,
			&timeWriteExternal)
		if err != nil {
			logrus.Error(err)
			return
		}
		qd.performanceSchemaSQLTableLockWaitsDesc.Collect(ch,
			float64(countReadNormal),
			[]string{
				objectSchema,
				objectName,
				"read_normal"})
		qd.performanceSchemaSQLTableLockWaitsDesc.Collect(ch,
			float64(countReadWithSharedLocks),
			[]string{
				objectSchema,
				objectName,
				"read_with_shared_locks"})
		qd.performanceSchemaSQLTableLockWaitsDesc.Collect(ch,
			float64(countReadHighPriority),
			[]string{
				objectSchema,
				objectName,
				"read_high_priority"})
		qd.performanceSchemaSQLTableLockWaitsDesc.Collect(ch,
			float64(countReadNoInsert),
			[]string{
				objectSchema,
				objectName,
				"read_no_insert"})
		qd.performanceSchemaSQLTableLockWaitsDesc.Collect(ch,
			float64(countWriteNormal),
			[]string{
				objectSchema,
				objectName,
				"write_normal"})
		qd.performanceSchemaSQLTableLockWaitsDesc.Collect(ch,
			float64(countWriteAllowWrite),
			[]string{
				objectSchema,
				objectName,
				"write_allow_write"})
		qd.performanceSchemaSQLTableLockWaitsDesc.Collect(ch,
			float64(countWriteConcurrentInsert),
			[]string{
				objectSchema,
				objectName,
				"write_concurrent_insert"})
		qd.performanceSchemaSQLTableLockWaitsDesc.Collect(ch,
			float64(countWriteLowPriority),
			[]string{
				objectSchema,
				objectName,
				"write_low_priority"})
		qd.performanceSchemaExternalTableLockWaitsDesc.Collect(ch,
			float64(countReadExternal),
			[]string{
				objectSchema,
				objectName,
				"read"})
		qd.performanceSchemaExternalTableLockWaitsDesc.Collect(ch,
			float64(countWriteExternal),
			[]string{
				objectSchema,
				objectName,
				"write"})
		qd.performanceSchemaSQLTableLockWaitsTimeDesc.Collect(ch,
			float64(timeReadNormal)/picoSeconds,
			[]string{
				objectSchema,
				objectName,
				"read_normal"})
		qd.performanceSchemaSQLTableLockWaitsTimeDesc.Collect(ch,
			float64(timeReadWithSharedLocks)/picoSeconds,
			[]string{
				objectSchema,
				objectName,
				"read_with_shared_locks"})

		qd.performanceSchemaSQLTableLockWaitsTimeDesc.Collect(ch,
			float64(timeReadHighPriority)/picoSeconds,
			[]string{
				objectSchema,
				objectName,
				"read_high_priority"})
		qd.performanceSchemaSQLTableLockWaitsTimeDesc.Collect(ch,
			float64(timeReadNoInsert)/picoSeconds,
			[]string{
				objectSchema,
				objectName,
				"read_no_insert"})
		qd.performanceSchemaSQLTableLockWaitsTimeDesc.Collect(ch,
			float64(timeWriteNormal)/picoSeconds,
			[]string{
				objectSchema,
				objectName,
				"write_normal"})
		qd.performanceSchemaSQLTableLockWaitsTimeDesc.Collect(ch,
			float64(timeWriteAllowWrite)/picoSeconds,
			[]string{
				objectSchema,
				objectName,
				"write_allow_write"})
		qd.performanceSchemaSQLTableLockWaitsTimeDesc.Collect(ch,
			float64(timeWriteConcurrentInsert)/picoSeconds,
			[]string{
				objectSchema,
				objectName,
				"write_concurrent_insert"})
		qd.performanceSchemaSQLTableLockWaitsTimeDesc.Collect(ch,
			float64(timeWriteLowPriority)/picoSeconds,
			[]string{
				objectSchema,
				objectName,
				"write_low_priority"})
		qd.performanceSchemaExternalTableLockWaitsTimeDesc.Collect(ch,
			float64(timeReadExternal)/picoSeconds,
			[]string{
				objectSchema,
				objectName,
				"read"})
		qd.performanceSchemaExternalTableLockWaitsTimeDesc.Collect(ch,
			float64(timeWriteExternal)/picoSeconds,
			[]string{
				objectSchema,
				objectName,
				"write"})

	}
}

type performanceSchemaSQLTableLockWaitsDesc struct {
	*baseMetrics
}

func newperformanceSchemaSQLTableLockWaitsDesc() *performanceSchemaSQLTableLockWaitsDesc {
	return &performanceSchemaSQLTableLockWaitsDesc{
		NewMetrics(
			"perf_schema_sql_lock_waits_total",
			"The total number of SQL lock wait events for each table and operation.",
			[]string{
				"schema",
				"name",
				"operation"})}
}

func (qd *performanceSchemaSQLTableLockWaitsDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaExternalTableLockWaitsDesc struct {
	*baseMetrics
}

func newperformanceSchemaExternalTableLockWaitsDesc() *performanceSchemaExternalTableLockWaitsDesc {
	return &performanceSchemaExternalTableLockWaitsDesc{
		NewMetrics(
			"perf_schema_external_lock_waits_total",
			"The total number of external lock wait events for each table and operation.",
			[]string{
				"schema",
				"name",
				"operation"})}
}
func (qd *performanceSchemaExternalTableLockWaitsDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaSQLTableLockWaitsTimeDesc struct {
	*baseMetrics
}

func newperformanceSchemaSQLTableLockWaitsTimeDesc() *performanceSchemaSQLTableLockWaitsTimeDesc {
	return &performanceSchemaSQLTableLockWaitsTimeDesc{
		NewMetrics(
			"perf_schema_sql_lock_waits_time_total",
			"The total time spent in SQL lock wait events for each table and operation.",
			[]string{
				"schema",
				"name",
				"operation"})}
}
func (qd *performanceSchemaSQLTableLockWaitsTimeDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaExternalTableLockWaitsTimeDesc struct {
	*baseMetrics
}

func newperformanceSchemaExternalTableLockWaitsTimeDesc() *performanceSchemaExternalTableLockWaitsTimeDesc {
	return &performanceSchemaExternalTableLockWaitsTimeDesc{
		NewMetrics(
			"perf_schema_external_lock_waits_time_total",
			"The total time spent in external lock wait events for each table and operation.",
			[]string{
				"schema",
				"name",
				"operation"})}
}
func (qd *performanceSchemaExternalTableLockWaitsTimeDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

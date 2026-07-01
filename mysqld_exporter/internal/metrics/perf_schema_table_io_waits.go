package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"mysqld_exporter/internal/exporter"
	"mysqld_exporter/internal/mysql"
)

const (
	perfTableIOWaitsQuery = `
	SELECT
	    OBJECT_SCHEMA, OBJECT_NAME,
	    COUNT_FETCH, COUNT_INSERT, COUNT_UPDATE, COUNT_DELETE,
	    SUM_TIMER_FETCH, SUM_TIMER_INSERT, SUM_TIMER_UPDATE, SUM_TIMER_DELETE
	  FROM performance_schema.table_io_waits_summary_by_table
	  WHERE OBJECT_SCHEMA NOT IN ('mysql', 'performance_schema')
	`
	perfTableIOWaitsResult = `*************************** 65. row ***************************
   OBJECT_SCHEMA: performance_schema
     OBJECT_NAME: events_statements_summary_by_user_by_event_name
     COUNT_FETCH: 0
    COUNT_INSERT: 0
    COUNT_UPDATE: 0
    COUNT_DELETE: 0
 SUM_TIMER_FETCH: 0
SUM_TIMER_INSERT: 0
SUM_TIMER_UPDATE: 0
SUM_TIMER_DELETE: 0
*************************** 66. row ***************************
   OBJECT_SCHEMA: performance_schema
     OBJECT_NAME: events_waits_summary_by_user_by_event_name
     COUNT_FETCH: 0
    COUNT_INSERT: 0
    COUNT_UPDATE: 0
    COUNT_DELETE: 0
 SUM_TIMER_FETCH: 0
SUM_TIMER_INSERT: 0
SUM_TIMER_UPDATE: 0
SUM_TIMER_DELETE: 0
*************************** 67. row ***************************
   OBJECT_SCHEMA: performance_schema
     OBJECT_NAME: memory_summary_by_user_by_event_name
     COUNT_FETCH: 0
    COUNT_INSERT: 0
    COUNT_UPDATE: 0
    COUNT_DELETE: 0
 SUM_TIMER_FETCH: 0
SUM_TIMER_INSERT: 0
SUM_TIMER_UPDATE: 0
SUM_TIMER_DELETE: 0
*************************** 68. row ***************************
   OBJECT_SCHEMA: performance_schema
     OBJECT_NAME: table_lock_waits_summary_by_table
     COUNT_FETCH: 0
    COUNT_INSERT: 0
    COUNT_UPDATE: 0
    COUNT_DELETE: 0
 SUM_TIMER_FETCH: 0
SUM_TIMER_INSERT: 0
SUM_TIMER_UPDATE: 0
SUM_TIMER_DELETE: 0
*************************** 69. row ***************************
   OBJECT_SCHEMA: performance_schema
     OBJECT_NAME: table_io_waits_summary_by_table
     COUNT_FETCH: 0
    COUNT_INSERT: 0
    COUNT_UPDATE: 0
    COUNT_DELETE: 0
 SUM_TIMER_FETCH: 0
SUM_TIMER_INSERT: 0
SUM_TIMER_UPDATE: 0
SUM_TIMER_DELETE: 0
69 rows in set (0.001 sec)
`
)

func init() {
	exporter.Register(
		NewperformanceSchemaTableWaits())
}

type performanceSchemaTableWaits struct {
	instance mysql.Instance
	performanceSchemaTableWaitsDesc
	performanceSchemaTableWaitsTimeDesc
}

func NewperformanceSchemaTableWaits() *performanceSchemaTableWaits {
	return &performanceSchemaTableWaits{
		//instance:                            instance,
		performanceSchemaTableWaitsDesc:     *newperformanceSchemaTableWaitsDesc(),
		performanceSchemaTableWaitsTimeDesc: *newperformanceSchemaTableWaitsTimeDesc(),
	}
}
func (qd *performanceSchemaTableWaits) Collect(ch chan<- prometheus.Metric) {
	qd.instance = *GetInstance()

	logrus.Info("Start collecting ScrapePerfTableIOWaits metrics")
	db := instance.GetDB()
	rows, err := db.Query(perfTableIOWaitsQuery)
	if err != nil {
		logrus.Error(err)
		return
	}
	defer rows.Close()

	var (
		objectSchema string
		objectName   string
		countFetch   uint64
		countInsert  uint64

		countUpdate uint64
		countDelete uint64

		timeFetch  uint64
		timeInsert uint64

		timeUpdate uint64
		timeDelete uint64
	)

	for rows.Next() {
		err := rows.Scan(
			&objectSchema,
			&objectName,
			&countFetch,
			&countInsert,
			&countUpdate,
			&countDelete,
			&timeFetch,
			&timeInsert,
			&timeUpdate,
			&timeDelete,
		)
		if err != nil {
			logrus.Error(err)
			return
		}
		qd.performanceSchemaTableWaitsDesc.Collect(ch,
			float64(countFetch),
			[]string{
				objectSchema,
				objectName,
				"fetch"})
		qd.performanceSchemaTableWaitsDesc.Collect(ch,
			float64(countInsert),
			[]string{
				objectSchema,
				objectName,
				"insert"})
		qd.performanceSchemaTableWaitsDesc.Collect(ch,
			float64(countUpdate),
			[]string{
				objectSchema,
				objectName,
				"update"})
		qd.performanceSchemaTableWaitsDesc.Collect(ch,
			float64(countDelete),
			[]string{
				objectSchema,
				objectName,
				"delete"})
		qd.performanceSchemaTableWaitsTimeDesc.Collect(ch,
			float64(timeFetch)/picoSeconds,
			[]string{
				objectSchema,
				objectName,
				"fetch"})
		qd.performanceSchemaTableWaitsTimeDesc.Collect(ch,
			float64(timeInsert)/picoSeconds,
			[]string{
				objectSchema,
				objectName,
				"insert"})
		qd.performanceSchemaTableWaitsTimeDesc.Collect(ch,
			float64(timeUpdate)/picoSeconds,
			[]string{
				objectSchema,
				objectName,
				"update"})
		qd.performanceSchemaTableWaitsTimeDesc.Collect(ch,
			float64(timeDelete)/picoSeconds,
			[]string{
				objectSchema,
				objectName,
				"delete"})

	}
}

type performanceSchemaTableWaitsDesc struct {
	*baseMetrics
}

func newperformanceSchemaTableWaitsDesc() *performanceSchemaTableWaitsDesc {
	return &performanceSchemaTableWaitsDesc{
		NewMetrics(
			"perf_schema_table_io_waits_total",
			"The total number of table I/O wait events for each table and operation.",
			[]string{
				"schema",
				"name",
				"operation"})}
}

func (qd *performanceSchemaTableWaitsDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaTableWaitsTimeDesc struct {
	*baseMetrics
}

func newperformanceSchemaTableWaitsTimeDesc() *performanceSchemaTableWaitsTimeDesc {
	return &performanceSchemaTableWaitsTimeDesc{
		NewMetrics(
			"perf_schema_table_io_waits_time_total",
			"The total time of table I/O wait events for each table and operation.",
			[]string{
				"schema",
				"name",
				"operation"})}
}
func (qd *performanceSchemaTableWaitsTimeDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

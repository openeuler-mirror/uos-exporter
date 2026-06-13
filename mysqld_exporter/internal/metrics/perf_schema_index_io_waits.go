package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"mysqld_exporter/internal/exporter"
	"mysqld_exporter/internal/mysql"
)

const (
	perfIndexIOWaitsQuery = `
	SELECT 
	    OBJECT_SCHEMA,
	    OBJECT_NAME, 
	    ifnull(INDEX_NAME, 'NONE') as INDEX_NAME,
	    COUNT_FETCH, 
	    COUNT_INSERT,
	    COUNT_UPDATE, 
	    COUNT_DELETE,
	    SUM_TIMER_FETCH, 
	    SUM_TIMER_INSERT,
	    SUM_TIMER_UPDATE,
	    SUM_TIMER_DELETE
	  FROM performance_schema.table_io_waits_summary_by_index_usage
	  WHERE OBJECT_SCHEMA NOT IN ('mysql', 'performance_schema')
	`
	perfIndexIOWaitsResult = `
MySQL [(none)]> desc performance_schema.table_io_waits_summary_by_index_usage;
+------------------+-----------------+------+-----+---------+-------+
| Field            | Type            | Null | Key | Default | Extra |
+------------------+-----------------+------+-----+---------+-------+
| OBJECT_TYPE      | varchar(64)     | YES  | MUL | NULL    |       |
| OBJECT_SCHEMA    | varchar(64)     | YES  |     | NULL    |       |
| OBJECT_NAME      | varchar(64)     | YES  |     | NULL    |       |
| INDEX_NAME       | varchar(64)     | YES  |     | NULL    |       |
| COUNT_STAR       | bigint unsigned | NO   |     | NULL    |       |
| SUM_TIMER_WAIT   | bigint unsigned | NO   |     | NULL    |       |
| MIN_TIMER_WAIT   | bigint unsigned | NO   |     | NULL    |       |
| AVG_TIMER_WAIT   | bigint unsigned | NO   |     | NULL    |       |
| MAX_TIMER_WAIT   | bigint unsigned | NO   |     | NULL    |       |
| COUNT_READ       | bigint unsigned | NO   |     | NULL    |       |
| SUM_TIMER_READ   | bigint unsigned | NO   |     | NULL    |       |
| MIN_TIMER_READ   | bigint unsigned | NO   |     | NULL    |       |
| AVG_TIMER_READ   | bigint unsigned | NO   |     | NULL    |       |
| MAX_TIMER_READ   | bigint unsigned | NO   |     | NULL    |       |
| COUNT_WRITE      | bigint unsigned | NO   |     | NULL    |       |
| SUM_TIMER_WRITE  | bigint unsigned | NO   |     | NULL    |       |
| MIN_TIMER_WRITE  | bigint unsigned | NO   |     | NULL    |       |
| AVG_TIMER_WRITE  | bigint unsigned | NO   |     | NULL    |       |
| MAX_TIMER_WRITE  | bigint unsigned | NO   |     | NULL    |       |
| COUNT_FETCH      | bigint unsigned | NO   |     | NULL    |       |
| SUM_TIMER_FETCH  | bigint unsigned | NO   |     | NULL    |       |
| MIN_TIMER_FETCH  | bigint unsigned | NO   |     | NULL    |       |
| AVG_TIMER_FETCH  | bigint unsigned | NO   |     | NULL    |       |
| MAX_TIMER_FETCH  | bigint unsigned | NO   |     | NULL    |       |
| COUNT_INSERT     | bigint unsigned | NO   |     | NULL    |       |
| SUM_TIMER_INSERT | bigint unsigned | NO   |     | NULL    |       |
| MIN_TIMER_INSERT | bigint unsigned | NO   |     | NULL    |       |
| AVG_TIMER_INSERT | bigint unsigned | NO   |     | NULL    |       |
| MAX_TIMER_INSERT | bigint unsigned | NO   |     | NULL    |       |
| COUNT_UPDATE     | bigint unsigned | NO   |     | NULL    |       |
| SUM_TIMER_UPDATE | bigint unsigned | NO   |     | NULL    |       |
| MIN_TIMER_UPDATE | bigint unsigned | NO   |     | NULL    |       |
| AVG_TIMER_UPDATE | bigint unsigned | NO   |     | NULL    |       |
| MAX_TIMER_UPDATE | bigint unsigned | NO   |     | NULL    |       |
| COUNT_DELETE     | bigint unsigned | NO   |     | NULL    |       |
| SUM_TIMER_DELETE | bigint unsigned | NO   |     | NULL    |       |
| MIN_TIMER_DELETE | bigint unsigned | NO   |     | NULL    |       |
| AVG_TIMER_DELETE | bigint unsigned | NO   |     | NULL    |       |
| MAX_TIMER_DELETE | bigint unsigned | NO   |     | NULL    |       |
+------------------+-----------------+------+-----+---------+-------+
39 rows in set (0.002 sec)
*************************** 260. row ***************************
   OBJECT_SCHEMA: performance_schema
     OBJECT_NAME: user_defined_functions
      INDEX_NAME: PRIMARY
     COUNT_FETCH: 0
    COUNT_INSERT: 0
    COUNT_UPDATE: 0
    COUNT_DELETE: 0
 SUM_TIMER_FETCH: 0
SUM_TIMER_INSERT: 0
SUM_TIMER_UPDATE: 0
SUM_TIMER_DELETE: 0
*************************** 261. row ***************************
   OBJECT_SCHEMA: performance_schema
     OBJECT_NAME: user_variables_by_thread
      INDEX_NAME: PRIMARY
     COUNT_FETCH: 0
    COUNT_INSERT: 0
    COUNT_UPDATE: 0
    COUNT_DELETE: 0
 SUM_TIMER_FETCH: 0
SUM_TIMER_INSERT: 0
SUM_TIMER_UPDATE: 0
SUM_TIMER_DELETE: 0
*************************** 262. row ***************************
   OBJECT_SCHEMA: performance_schema
     OBJECT_NAME: users
      INDEX_NAME: USER
     COUNT_FETCH: 0
    COUNT_INSERT: 0
    COUNT_UPDATE: 0
    COUNT_DELETE: 0
 SUM_TIMER_FETCH: 0
SUM_TIMER_INSERT: 0
SUM_TIMER_UPDATE: 0
SUM_TIMER_DELETE: 0
*************************** 263. row ***************************
   OBJECT_SCHEMA: performance_schema
     OBJECT_NAME: variables_by_thread
      INDEX_NAME: PRIMARY
     COUNT_FETCH: 0
    COUNT_INSERT: 0
    COUNT_UPDATE: 0
    COUNT_DELETE: 0
 SUM_TIMER_FETCH: 0
SUM_TIMER_INSERT: 0
SUM_TIMER_UPDATE: 0
SUM_TIMER_DELETE: 0
263 rows in set (0.056 sec)

`
)

func init() {
	exporter.Register(
		NewScrapePerfIndexIOWaits())
}

type ScrapePerfIndexIOWaits struct {
	instance mysql.Instance
	performanceSchemaIndexWaitsDesc
	performanceSchemaIndexWaitsTimeDesc
}

func NewScrapePerfIndexIOWaits() *ScrapePerfIndexIOWaits {
	return &ScrapePerfIndexIOWaits{
		//instance:                            instance,
		performanceSchemaIndexWaitsDesc:     *NewPerformanceSchemaIndexWaitsDesc(),
		performanceSchemaIndexWaitsTimeDesc: *NewPerformanceSchemaIndexWaitsTimeDesc(),
	}
}

func (qd ScrapePerfIndexIOWaits) Collect(ch chan<- prometheus.Metric) {
	qd.instance = *GetInstance()

	if err := qd.instance.Ping(); err != nil {
		logrus.Errorf("ping mysql instance error: %s", err)
		return
	}
	db := instance.GetDB()
	rows, err := db.Query(perfIndexIOWaitsQuery)
	if err != nil {
		logrus.Error(err)
		return
	}
	defer rows.Close()
	var (
		objectSchema string
		objectName   string
		indexName    string
		countFetch   uint64
		countInsert  uint64
		countUpdate  uint64
		countDelete  uint64
		timeFetch    uint64
		timeInsert   uint64
		timeUpdate   uint64
		timeDelete   uint64
	)
	for rows.Next() {
		err = rows.Scan(
			&objectSchema,
			&objectName,
			&indexName,
			&countFetch,
			&countInsert,
			&countUpdate,
			&countDelete,
			&timeFetch,
			&timeInsert,
			&timeUpdate,
			&timeDelete)
		if err != nil {
			logrus.Error(err)
			return
		}
		qd.performanceSchemaIndexWaitsDesc.Collect(ch,
			float64(countFetch),
			[]string{
				objectSchema,
				objectName,
				indexName,
				"fetch"})
		if indexName == "NONE" {
			qd.performanceSchemaIndexWaitsDesc.Collect(ch,
				float64(countInsert),
				[]string{
					objectSchema,
					objectName,
					indexName,
					"insert"})
		}
		qd.performanceSchemaIndexWaitsDesc.Collect(ch,
			float64(countUpdate),
			[]string{
				objectSchema,
				objectName,
				indexName,
				"update"})
		qd.performanceSchemaIndexWaitsDesc.Collect(ch,
			float64(countDelete),
			[]string{
				objectSchema,
				objectName,
				indexName,
				"delete"})
		qd.performanceSchemaIndexWaitsTimeDesc.Collect(ch,
			float64(timeFetch)/picoSeconds,
			[]string{
				objectSchema,
				objectName,
				indexName,
				"fetch"})
		if indexName == "NONE" {
			qd.performanceSchemaIndexWaitsTimeDesc.Collect(ch,
				float64(timeInsert)/picoSeconds,
				[]string{
					objectSchema,
					objectName,
					indexName,
					"insert"})
		}
		qd.performanceSchemaIndexWaitsTimeDesc.Collect(ch,
			float64(timeUpdate)/picoSeconds,
			[]string{
				objectSchema,
				objectName,
				indexName,
				"update"})
		qd.performanceSchemaIndexWaitsTimeDesc.Collect(ch,
			float64(timeDelete)/picoSeconds,
			[]string{
				objectSchema,
				objectName,
				indexName,
				"delete"})
	}
}

type performanceSchemaIndexWaitsDesc struct {
	*baseMetrics
}

func NewPerformanceSchemaIndexWaitsDesc() *performanceSchemaIndexWaitsDesc {
	return &performanceSchemaIndexWaitsDesc{
		NewMetrics(
			"perf_schema_index_io_waits_total",
			"The total number of index I/O wait events for each index and operation.",
			[]string{
				"schema",
				"name",
				"index",
				"operation"})}
}
func (qd *performanceSchemaIndexWaitsDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaIndexWaitsTimeDesc struct {
	*baseMetrics
}

func NewPerformanceSchemaIndexWaitsTimeDesc() *performanceSchemaIndexWaitsTimeDesc {
	return &performanceSchemaIndexWaitsTimeDesc{
		NewMetrics(
			"perf_schema_index_io_waits_time_seconds_total",
			"The total time spent in index I/O wait events for each index and operation.",
			[]string{
				"schema",
				"name",
				"index",
				"operation"})}
}
func (qd *performanceSchemaIndexWaitsTimeDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

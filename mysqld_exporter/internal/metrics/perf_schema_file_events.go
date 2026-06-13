package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"mysqld_exporter/internal/exporter"
	"mysqld_exporter/internal/mysql"
)

const (
	perfFileEventsQuery = `
	SELECT
	    EVENT_NAME,
	    COUNT_READ,
	    SUM_TIMER_READ, 
	    SUM_NUMBER_OF_BYTES_READ,
	    COUNT_WRITE, 
	    SUM_TIMER_WRITE, 
	    SUM_NUMBER_OF_BYTES_WRITE,
	    COUNT_MISC, 
	    SUM_TIMER_MISC
	  FROM performance_schema.file_summary_by_event_name
	`
	perfFileEventsResult = `
MySQL [(none)]> desc performance_schema.file_summary_by_event_name;
+---------------------------+-----------------+------+-----+---------+-------+
| Field                     | Type            | Null | Key | Default | Extra |
+---------------------------+-----------------+------+-----+---------+-------+
| EVENT_NAME                | varchar(128)    | NO   | PRI | NULL    |       |
| COUNT_STAR                | bigint unsigned | NO   |     | NULL    |       |
| SUM_TIMER_WAIT            | bigint unsigned | NO   |     | NULL    |       |
| MIN_TIMER_WAIT            | bigint unsigned | NO   |     | NULL    |       |
| AVG_TIMER_WAIT            | bigint unsigned | NO   |     | NULL    |       |
| MAX_TIMER_WAIT            | bigint unsigned | NO   |     | NULL    |       |
| COUNT_READ                | bigint unsigned | NO   |     | NULL    |       |
| SUM_TIMER_READ            | bigint unsigned | NO   |     | NULL    |       |
| MIN_TIMER_READ            | bigint unsigned | NO   |     | NULL    |       |
| AVG_TIMER_READ            | bigint unsigned | NO   |     | NULL    |       |
| MAX_TIMER_READ            | bigint unsigned | NO   |     | NULL    |       |
| SUM_NUMBER_OF_BYTES_READ  | bigint          | NO   |     | NULL    |       |
| COUNT_WRITE               | bigint unsigned | NO   |     | NULL    |       |
| SUM_TIMER_WRITE           | bigint unsigned | NO   |     | NULL    |       |
| MIN_TIMER_WRITE           | bigint unsigned | NO   |     | NULL    |       |
| AVG_TIMER_WRITE           | bigint unsigned | NO   |     | NULL    |       |
| MAX_TIMER_WRITE           | bigint unsigned | NO   |     | NULL    |       |
| SUM_NUMBER_OF_BYTES_WRITE | bigint          | NO   |     | NULL    |       |
| COUNT_MISC                | bigint unsigned | NO   |     | NULL    |       |
| SUM_TIMER_MISC            | bigint unsigned | NO   |     | NULL    |       |
| MIN_TIMER_MISC            | bigint unsigned | NO   |     | NULL    |       |
| AVG_TIMER_MISC            | bigint unsigned | NO   |     | NULL    |       |
| MAX_TIMER_MISC            | bigint unsigned | NO   |     | NULL    |       |
+---------------------------+-----------------+------+-----+---------+-------+
*************************** 41. row ***************************
               EVENT_NAME: wait/io/file/innodb/innodb_arch_file
               COUNT_READ: 0
           SUM_TIMER_READ: 0
 SUM_NUMBER_OF_BYTES_READ: 0
              COUNT_WRITE: 0
          SUM_TIMER_WRITE: 0
SUM_NUMBER_OF_BYTES_WRITE: 0
               COUNT_MISC: 0
           SUM_TIMER_MISC: 0
*************************** 42. row ***************************
               EVENT_NAME: wait/io/file/innodb/innodb_clone_file
               COUNT_READ: 0
           SUM_TIMER_READ: 0
 SUM_NUMBER_OF_BYTES_READ: 0
              COUNT_WRITE: 0
          SUM_TIMER_WRITE: 0
SUM_NUMBER_OF_BYTES_WRITE: 0
               COUNT_MISC: 0
           SUM_TIMER_MISC: 0
*************************** 43. row ***************************
               EVENT_NAME: wait/io/file/innodb/meb::redo_log_archive_file
               COUNT_READ: 0
           SUM_TIMER_READ: 0
 SUM_NUMBER_OF_BYTES_READ: 0
              COUNT_WRITE: 0
          SUM_TIMER_WRITE: 0
SUM_NUMBER_OF_BYTES_WRITE: 0
               COUNT_MISC: 0
           SUM_TIMER_MISC: 0
*************************** 44. row ***************************
               EVENT_NAME: wait/io/file/myisam/data_tmp
               COUNT_READ: 0
           SUM_TIMER_READ: 0
 SUM_NUMBER_OF_BYTES_READ: 0
              COUNT_WRITE: 0
          SUM_TIMER_WRITE: 0
SUM_NUMBER_OF_BYTES_WRITE: 0
               COUNT_MISC: 0
           SUM_TIMER_MISC: 0
*************************** 45. row ***************************
               EVENT_NAME: wait/io/file/myisam/dfile
               COUNT_READ: 0
           SUM_TIMER_READ: 0
 SUM_NUMBER_OF_BYTES_READ: 0
              COUNT_WRITE: 0
          SUM_TIMER_WRITE: 0
SUM_NUMBER_OF_BYTES_WRITE: 0
               COUNT_MISC: 0
           SUM_TIMER_MISC: 0
*************************** 46. row ***************************
               EVENT_NAME: wait/io/file/myisam/kfile
               COUNT_READ: 0
           SUM_TIMER_READ: 0
 SUM_NUMBER_OF_BYTES_READ: 0
              COUNT_WRITE: 0
          SUM_TIMER_WRITE: 0
SUM_NUMBER_OF_BYTES_WRITE: 0
               COUNT_MISC: 0
           SUM_TIMER_MISC: 0
*************************** 47. row ***************************
               EVENT_NAME: wait/io/file/myisam/log
               COUNT_READ: 0
           SUM_TIMER_READ: 0
 SUM_NUMBER_OF_BYTES_READ: 0
              COUNT_WRITE: 0
          SUM_TIMER_WRITE: 0
SUM_NUMBER_OF_BYTES_WRITE: 0
               COUNT_MISC: 0
           SUM_TIMER_MISC: 0
*************************** 48. row ***************************
               EVENT_NAME: wait/io/file/myisammrg/MRG
               COUNT_READ: 0
           SUM_TIMER_READ: 0
 SUM_NUMBER_OF_BYTES_READ: 0
              COUNT_WRITE: 0
          SUM_TIMER_WRITE: 0
SUM_NUMBER_OF_BYTES_WRITE: 0
               COUNT_MISC: 0
           SUM_TIMER_MISC: 0
*************************** 49. row ***************************
               EVENT_NAME: wait/io/file/archive/metadata
               COUNT_READ: 0
           SUM_TIMER_READ: 0
 SUM_NUMBER_OF_BYTES_READ: 0
              COUNT_WRITE: 0
          SUM_TIMER_WRITE: 0
SUM_NUMBER_OF_BYTES_WRITE: 0
               COUNT_MISC: 0
           SUM_TIMER_MISC: 0
*************************** 50. row ***************************
               EVENT_NAME: wait/io/file/archive/data
               COUNT_READ: 0
           SUM_TIMER_READ: 0
 SUM_NUMBER_OF_BYTES_READ: 0
              COUNT_WRITE: 0
          SUM_TIMER_WRITE: 0
SUM_NUMBER_OF_BYTES_WRITE: 0
               COUNT_MISC: 0
           SUM_TIMER_MISC: 0
*************************** 51. row ***************************
               EVENT_NAME: wait/io/file/archive/FRM
               COUNT_READ: 0
           SUM_TIMER_READ: 0
 SUM_NUMBER_OF_BYTES_READ: 0
              COUNT_WRITE: 0
          SUM_TIMER_WRITE: 0
SUM_NUMBER_OF_BYTES_WRITE: 0
               COUNT_MISC: 0
           SUM_TIMER_MISC: 0
51 rows in set (0.001 sec)
`
)

type ScrapePerfFileEvents struct {
	instance mysql.Instance
	performanceSchemaFileEventsDesc
	performanceSchemaFileEventsTimeDesc
	performanceSchemaFileEventsBytesDesc
}

func init() {
	exporter.Register(
		NewScrapePerfFileEvents())
}
func NewScrapePerfFileEvents() *ScrapePerfFileEvents {
	return &ScrapePerfFileEvents{
		//instance:                             instance,
		performanceSchemaFileEventsDesc:      *NewPerformanceSchemaFileEventsDesc(),
		performanceSchemaFileEventsTimeDesc:  *NewPerformanceSchemaFileEventsTimeDesc(),
		performanceSchemaFileEventsBytesDesc: *NewPerformanceSchemaFileEventsBytesDesc(),
	}
}

func (qd ScrapePerfFileEvents) Collect(ch chan<- prometheus.Metric) {
	qd.instance = *GetInstance()

	if err := qd.instance.Ping(); err != nil {
		logrus.Errorf("ping mysql instance error: %s", err)
		return
	}
	db := instance.GetDB()
	rows, err := db.Query(perfFileEventsQuery)
	if err != nil {
		logrus.Error(err)
		return
	}
	defer rows.Close()
	var (
		eventName  string
		countRead  uint64
		timeRead   uint64
		bytesRead  uint64
		countWrite uint64
		timeWrite  uint64
		bytesWrite uint64
		countMisc  uint64
		timeMisc   uint64
	)
	for rows.Next() {
		err = rows.Scan(
			&eventName,
			&countRead,
			&timeRead,
			&bytesRead,
			&countWrite,
			&timeWrite,
			&bytesWrite,
			&countMisc,
			&timeMisc)
		if err != nil {
			logrus.Error(err)
			return
		}
		qd.performanceSchemaFileEventsDesc.Collect(ch,
			float64(countRead),
			[]string{
				eventName,
				"read"})
		qd.performanceSchemaFileEventsDesc.Collect(ch,
			float64(countWrite),
			[]string{
				eventName,
				"write"})
		qd.performanceSchemaFileEventsDesc.Collect(ch,
			float64(countMisc),
			[]string{
				eventName,
				"misc"})
		qd.performanceSchemaFileEventsTimeDesc.Collect(ch,
			float64(timeRead)/picoSeconds,
			[]string{
				eventName,
				"read"})
		qd.performanceSchemaFileEventsTimeDesc.Collect(ch,
			float64(timeWrite)/picoSeconds,
			[]string{
				eventName,
				"write"})
		qd.performanceSchemaFileEventsTimeDesc.Collect(ch,
			float64(timeMisc)/picoSeconds,
			[]string{
				eventName,
				"misc"})
		qd.performanceSchemaFileEventsBytesDesc.Collect(ch,
			float64(bytesRead),
			[]string{
				eventName,
				"read"})
		qd.performanceSchemaFileEventsBytesDesc.Collect(ch,
			float64(bytesWrite),
			[]string{
				eventName,
				"write"})
	}
}

type performanceSchemaFileEventsDesc struct {
	*baseMetrics
}

func NewPerformanceSchemaFileEventsDesc() *performanceSchemaFileEventsDesc {
	return &performanceSchemaFileEventsDesc{
		NewMetrics(
			"perf_schema_file_events_total",
			"The total file events by event name/mode.",
			[]string{
				"event_name",
				"mode"})}
}
func (qd *performanceSchemaFileEventsDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaFileEventsTimeDesc struct {
	*baseMetrics
}

func NewPerformanceSchemaFileEventsTimeDesc() *performanceSchemaFileEventsTimeDesc {
	return &performanceSchemaFileEventsTimeDesc{
		NewMetrics(
			"perf_schema_file_events_seconds_total",
			"The total seconds of file events by event name/mode.",
			[]string{
				"event_name",
				"mode"})}
}
func (qd *performanceSchemaFileEventsTimeDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaFileEventsBytesDesc struct {
	*baseMetrics
}

func NewPerformanceSchemaFileEventsBytesDesc() *performanceSchemaFileEventsBytesDesc {
	return &performanceSchemaFileEventsBytesDesc{
		NewMetrics(
			"perf_schema_file_events_bytes_total",
			"The total bytes of file events by event name/mode.",
			[]string{
				"event_name",
				"mode"})}
}
func (qd *performanceSchemaFileEventsBytesDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

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


// TODO: implement functions

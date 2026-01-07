package metrics

import (
	"github.com/alecthomas/kingpin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"mysqld_exporter/internal/exporter"
	"mysqld_exporter/internal/mysql"
	"strings"
)

const (
	perfFileInstancesQuery = `
	SELECT
	    FILE_NAME, 
	    EVENT_NAME,
	    COUNT_READ, 
	    COUNT_WRITE,
	    SUM_NUMBER_OF_BYTES_READ, 
	    SUM_NUMBER_OF_BYTES_WRITE
	  FROM performance_schema.file_summary_by_instance
	     where FILE_NAME REGEXP ?
	`
	perfFileInstancesResult = `
MySQL [(none)]> desc performance_schema.file_summary_by_instance;
+---------------------------+-----------------+------+-----+---------+-------+
| Field                     | Type            | Null | Key | Default | Extra |
+---------------------------+-----------------+------+-----+---------+-------+
| FILE_NAME                 | varchar(512)    | NO   | MUL | NULL    |       |
| EVENT_NAME                | varchar(128)    | NO   | MUL | NULL    |       |
| OBJECT_INSTANCE_BEGIN     | bigint unsigned | NO   | PRI | NULL    |       |
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
25 rows in set (0.002 sec)
*************************** 49. row ***************************
                FILE_NAME: /var/lib/mysql/binlog.index
               EVENT_NAME: wait/io/file/sql/binlog_index
               COUNT_READ: 2
              COUNT_WRITE: 0
 SUM_NUMBER_OF_BYTES_READ: 64
SUM_NUMBER_OF_BYTES_WRITE: 0
*************************** 50. row ***************************
                FILE_NAME: /var/lib/mysql/#innodb_redo/#ib_redo31_tmp
               EVENT_NAME: wait/io/file/innodb/innodb_log_file
               COUNT_READ: 0
              COUNT_WRITE: 0
 SUM_NUMBER_OF_BYTES_READ: 0
SUM_NUMBER_OF_BYTES_WRITE: 0
*************************** 51. row ***************************
                FILE_NAME: /var/lib/mysql/#innodb_redo/#ib_redo32_tmp
               EVENT_NAME: wait/io/file/innodb/innodb_log_file
               COUNT_READ: 0
              COUNT_WRITE: 0
 SUM_NUMBER_OF_BYTES_READ: 0
SUM_NUMBER_OF_BYTES_WRITE: 0
*************************** 52. row ***************************
                FILE_NAME: /run/mysqld/mysqld.pid
               EVENT_NAME: wait/io/file/sql/pid
               COUNT_READ: 0
              COUNT_WRITE: 1
 SUM_NUMBER_OF_BYTES_READ: 0
SUM_NUMBER_OF_BYTES_WRITE: 8
*************************** 53. row ***************************
                FILE_NAME: /var/lib/mysql/#innodb_redo/#ib_redo33_tmp
               EVENT_NAME: wait/io/file/innodb/innodb_log_file
               COUNT_READ: 0
              COUNT_WRITE: 0
 SUM_NUMBER_OF_BYTES_READ: 0
SUM_NUMBER_OF_BYTES_WRITE: 0
*************************** 54. row ***************************
                FILE_NAME: /var/lib/mysql/#innodb_redo/#ib_redo34_tmp
               EVENT_NAME: wait/io/file/innodb/innodb_log_file
               COUNT_READ: 0
              COUNT_WRITE: 0
 SUM_NUMBER_OF_BYTES_READ: 0
SUM_NUMBER_OF_BYTES_WRITE: 0
*************************** 55. row ***************************
                FILE_NAME: /var/lib/mysql/#innodb_redo/#ib_redo35_tmp
               EVENT_NAME: wait/io/file/innodb/innodb_log_file
               COUNT_READ: 0
              COUNT_WRITE: 0
 SUM_NUMBER_OF_BYTES_READ: 0
SUM_NUMBER_OF_BYTES_WRITE: 0
*************************** 56. row ***************************
                FILE_NAME: /var/lib/mysql/#innodb_redo/#ib_redo36_tmp
               EVENT_NAME: wait/io/file/innodb/innodb_log_file
               COUNT_READ: 0
              COUNT_WRITE: 0
 SUM_NUMBER_OF_BYTES_READ: 0
SUM_NUMBER_OF_BYTES_WRITE: 0
*************************** 57. row ***************************
                FILE_NAME: /var/lib/mysql/#innodb_redo/#ib_redo37_tmp
               EVENT_NAME: wait/io/file/innodb/innodb_log_file
               COUNT_READ: 0
              COUNT_WRITE: 0
 SUM_NUMBER_OF_BYTES_READ: 0
SUM_NUMBER_OF_BYTES_WRITE: 0
*************************** 58. row ***************************
                FILE_NAME: /var/lib/mysql/mysql/general_log.CSM
               EVENT_NAME: wait/io/file/csv/metadata
               COUNT_READ: 1
              COUNT_WRITE: 0
 SUM_NUMBER_OF_BYTES_READ: 35
SUM_NUMBER_OF_BYTES_WRITE: 0
*************************** 59. row ***************************
                FILE_NAME: /var/lib/mysql/mysql/general_log.CSV
               EVENT_NAME: wait/io/file/csv/data
               COUNT_READ: 0
              COUNT_WRITE: 0
 SUM_NUMBER_OF_BYTES_READ: 0
SUM_NUMBER_OF_BYTES_WRITE: 0
*************************** 60. row ***************************
                FILE_NAME: /var/lib/mysql/mysql/slow_log.CSM
               EVENT_NAME: wait/io/file/csv/metadata
               COUNT_READ: 1
              COUNT_WRITE: 0
 SUM_NUMBER_OF_BYTES_READ: 35
SUM_NUMBER_OF_BYTES_WRITE: 0
*************************** 61. row ***************************
                FILE_NAME: /var/lib/mysql/mysql/slow_log.CSV
               EVENT_NAME: wait/io/file/csv/data
               COUNT_READ: 0
              COUNT_WRITE: 0
 SUM_NUMBER_OF_BYTES_READ: 0
SUM_NUMBER_OF_BYTES_WRITE: 0
61 rows in set (0.001 sec)

`
)

var (
	performanceSchemaFileInstancesFilter = kingpin.Flag(
		"collect.perf_schema.file_instances.filter",
		"RegEx file_name filter for performance_schema.file_summary_by_instance",
	).Default(".*").String()

	performanceSchemaFileInstancesRemovePrefix = kingpin.Flag(
		"collect.perf_schema.file_instances.remove_prefix",
		"Remove path prefix in performance_schema.file_summary_by_instance",
	).Default("/var/lib/mysql/").String()
)

type ScrapePerfFileInstances struct {
	instance mysql.Instance
	performanceSchemaFileInstancesBytesDesc
	performanceSchemaFileInstancesCountDesc
}


// TODO: implement functions

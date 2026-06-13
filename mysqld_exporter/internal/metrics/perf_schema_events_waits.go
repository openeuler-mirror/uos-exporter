package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"mysqld_exporter/internal/exporter"
	"mysqld_exporter/internal/mysql"
)

const (
	perfEventsWaitsQuery = `
	SELECT 
	    EVENT_NAME,
	    COUNT_STAR,
	    SUM_TIMER_WAIT
	  FROM performance_schema.events_waits_summary_global_by_event_name
	`
	perfEventsWaitsResult = `
MySQL [(none)]> desc performance_schema.events_waits_summary_global_by_event_name
    -> ;
+----------------+-----------------+------+-----+---------+-------+
| Field          | Type            | Null | Key | Default | Extra |
+----------------+-----------------+------+-----+---------+-------+
| EVENT_NAME     | varchar(128)    | NO   | PRI | NULL    |       |
| COUNT_STAR     | bigint unsigned | NO   |     | NULL    |       |
| SUM_TIMER_WAIT | bigint unsigned | NO   |     | NULL    |       |
| MIN_TIMER_WAIT | bigint unsigned | NO   |     | NULL    |       |
| AVG_TIMER_WAIT | bigint unsigned | NO   |     | NULL    |       |
| MAX_TIMER_WAIT | bigint unsigned | NO   |     | NULL    |       |
+----------------+-----------------+------+-----+---------+-------+

*************************** 386. row ***************************
    EVENT_NAME: wait/io/file/myisam/log
    COUNT_STAR: 0
SUM_TIMER_WAIT: 0
*************************** 387. row ***************************
    EVENT_NAME: wait/io/file/myisammrg/MRG
    COUNT_STAR: 0
SUM_TIMER_WAIT: 0
*************************** 388. row ***************************
    EVENT_NAME: wait/io/file/archive/metadata
    COUNT_STAR: 0
SUM_TIMER_WAIT: 0
*************************** 389. row ***************************
    EVENT_NAME: wait/io/file/archive/data
    COUNT_STAR: 0
SUM_TIMER_WAIT: 0
*************************** 390. row ***************************
    EVENT_NAME: wait/io/file/archive/FRM
    COUNT_STAR: 0
SUM_TIMER_WAIT: 0
*************************** 391. row ***************************
    EVENT_NAME: wait/io/table/sql/handler
    COUNT_STAR: 0
SUM_TIMER_WAIT: 0
*************************** 392. row ***************************
    EVENT_NAME: wait/lock/table/sql/handler
    COUNT_STAR: 0
SUM_TIMER_WAIT: 0
*************************** 393. row ***************************
    EVENT_NAME: wait/io/socket/sql/server_tcpip_socket
    COUNT_STAR: 0
SUM_TIMER_WAIT: 0
*************************** 394. row ***************************
    EVENT_NAME: wait/io/socket/sql/server_unix_socket
    COUNT_STAR: 0
SUM_TIMER_WAIT: 0
*************************** 395. row ***************************
    EVENT_NAME: wait/io/socket/sql/client_connection
    COUNT_STAR: 0
SUM_TIMER_WAIT: 0
*************************** 396. row ***************************
    EVENT_NAME: wait/io/socket/mysqlx/tcpip_socket
    COUNT_STAR: 0
SUM_TIMER_WAIT: 0
*************************** 397. row ***************************
    EVENT_NAME: wait/io/socket/mysqlx/diagnostics_socket
    COUNT_STAR: 0
SUM_TIMER_WAIT: 0
*************************** 398. row ***************************
    EVENT_NAME: wait/io/socket/mysqlx/unix_socket
    COUNT_STAR: 0
SUM_TIMER_WAIT: 0
*************************** 399. row ***************************
    EVENT_NAME: wait/io/socket/mysqlx/client_connection
    COUNT_STAR: 0
SUM_TIMER_WAIT: 0
*************************** 400. row ***************************
    EVENT_NAME: idle
    COUNT_STAR: 398
SUM_TIMER_WAIT: 152736445682496000
*************************** 401. row ***************************
    EVENT_NAME: wait/lock/metadata/sql/mdl
    COUNT_STAR: 0
SUM_TIMER_WAIT: 0

`
)

type ScrapePerfEventsWaits struct {
	instance mysql.Instance
	performanceSchemaEventsWaitsDesc
	performanceSchemaEventsWaitsTimeDesc
}


// TODO: implement functions

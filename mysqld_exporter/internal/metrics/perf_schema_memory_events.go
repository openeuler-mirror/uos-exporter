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
	perfMemoryEventsQuery = `
	SELECT
		EVENT_NAME, 
		COUNT_ALLOC,
		COUNT_FREE,
		SUM_NUMBER_OF_BYTES_ALLOC, 
		SUM_NUMBER_OF_BYTES_FREE,
		LOW_COUNT_USED,
		CURRENT_COUNT_USED,
		HIGH_COUNT_USED,
		LOW_NUMBER_OF_BYTES_USED,
		CURRENT_NUMBER_OF_BYTES_USED,
		HIGH_NUMBER_OF_BYTES_USED
	FROM performance_schema.memory_summary_global_by_event_name
		where COUNT_ALLOC > 0;
`
	perfMemoryEventsQueryResult = `
desc performance_schema.memory_summary_global_by_event_name;
+------------------------------+-----------------+------+-----+---------+-------+
| Field                        | Type            | Null | Key | Default | Extra |
+------------------------------+-----------------+------+-----+---------+-------+
| EVENT_NAME                   | varchar(128)    | NO   | PRI | NULL    |       |
| COUNT_ALLOC                  | bigint unsigned | NO   |     | NULL    |       |
| COUNT_FREE                   | bigint unsigned | NO   |     | NULL    |       |
| SUM_NUMBER_OF_BYTES_ALLOC    | bigint unsigned | NO   |     | NULL    |       |
| SUM_NUMBER_OF_BYTES_FREE     | bigint unsigned | NO   |     | NULL    |       |
| LOW_COUNT_USED               | bigint          | NO   |     | NULL    |       |
| CURRENT_COUNT_USED           | bigint          | NO   |     | NULL    |       |
| HIGH_COUNT_USED              | bigint          | NO   |     | NULL    |       |
| LOW_NUMBER_OF_BYTES_USED     | bigint          | NO   |     | NULL    |       |
| CURRENT_NUMBER_OF_BYTES_USED | bigint          | NO   |     | NULL    |       |
| HIGH_NUMBER_OF_BYTES_USED    | bigint          | NO   |     | NULL    |       |
+------------------------------+-----------------+------+-----+---------+-------+
*************************** 209. row ***************************
                  EVENT_NAME: memory/blackhole/blackhole_share
                 COUNT_ALLOC: 1
                  COUNT_FREE: 0
   SUM_NUMBER_OF_BYTES_ALLOC: 120
    SUM_NUMBER_OF_BYTES_FREE: 0
              LOW_COUNT_USED: 0
          CURRENT_COUNT_USED: 1
             HIGH_COUNT_USED: 1
    LOW_NUMBER_OF_BYTES_USED: 0
CURRENT_NUMBER_OF_BYTES_USED: 120
   HIGH_NUMBER_OF_BYTES_USED: 120
*************************** 210. row ***************************
                  EVENT_NAME: memory/mysqlx/objects
                 COUNT_ALLOC: 14
                  COUNT_FREE: 0
   SUM_NUMBER_OF_BYTES_ALLOC: 3328
    SUM_NUMBER_OF_BYTES_FREE: 0
              LOW_COUNT_USED: 0
          CURRENT_COUNT_USED: 14
             HIGH_COUNT_USED: 14
    LOW_NUMBER_OF_BYTES_USED: 0
CURRENT_NUMBER_OF_BYTES_USED: 3328
   HIGH_NUMBER_OF_BYTES_USED: 3328
*************************** 211. row ***************************
                  EVENT_NAME: memory/sql/tz_storage
                 COUNT_ALLOC: 1
                  COUNT_FREE: 0
   SUM_NUMBER_OF_BYTES_ALLOC: 32816
    SUM_NUMBER_OF_BYTES_FREE: 0
              LOW_COUNT_USED: 0
          CURRENT_COUNT_USED: 1
             HIGH_COUNT_USED: 1
    LOW_NUMBER_OF_BYTES_USED: 0
CURRENT_NUMBER_OF_BYTES_USED: 32816
   HIGH_NUMBER_OF_BYTES_USED: 32816
*************************** 212. row ***************************
                  EVENT_NAME: memory/sql/servers_cache
                 COUNT_ALLOC: 1
                  COUNT_FREE: 0
   SUM_NUMBER_OF_BYTES_ALLOC: 120
    SUM_NUMBER_OF_BYTES_FREE: 0
              LOW_COUNT_USED: 0
          CURRENT_COUNT_USED: 1
             HIGH_COUNT_USED: 1
    LOW_NUMBER_OF_BYTES_USED: 0
CURRENT_NUMBER_OF_BYTES_USED: 120
   HIGH_NUMBER_OF_BYTES_USED: 120
212 rows in set (0.064 sec)
`
)

var (
	performanceSchemaMemoryEventsRemovePrefix = kingpin.Flag(
		"collect.perf_schema.memory_events.remove_prefix",
		"Remove instrument prefix in performance_schema.memory_summary_global_by_event_name",
	).Default("memory/").String()
)


// TODO: implement functions

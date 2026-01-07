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


// TODO: implement functions

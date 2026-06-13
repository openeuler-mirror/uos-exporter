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

func init() {
	exporter.Register(
		NewScrapePerfMemoryEvents())
}

type ScrapePerfMemoryEvents struct {
	instance mysql.Instance
	performanceSchemaMemoryCountAlloc
	performanceSchemaMemoryCountFree
	performanceSchemaMemoryBytesAllocDesc
	performanceSchemaMemoryBytesFreeDesc
	performanceSchemaMemoryLowCountUsed
	performanceSchemaMemoryCurrentCountUsed
	performanceSchemaMemoryHighCountUsed
	perforanceSchemaMemoryLowUsedBytesDesc
	perforanceSchemaMemoryUsedBytesDesc
	perforanceSchemaMemoryHighUsedBytesDesc
}

func NewScrapePerfMemoryEvents() *ScrapePerfMemoryEvents {
	return &ScrapePerfMemoryEvents{
		//instance:                                instance,
		performanceSchemaMemoryBytesAllocDesc:   *newperformanceSchemaMemoryBytesAllocDesc(),
		performanceSchemaMemoryBytesFreeDesc:    *NewperformanceSchemaMemoryBytesFreeDesc(),
		perforanceSchemaMemoryUsedBytesDesc:     *NewPerformanceSchemaMemoryUsedBytesDesc(),
		performanceSchemaMemoryCountAlloc:       *NewPerformanceSchemaMemoryCountAlloc(),
		performanceSchemaMemoryCountFree:        *NewperformanceSchemaMemoryCountFree(),
		performanceSchemaMemoryLowCountUsed:     *NewperformanceSchemaMemoryLowCountUsed(),
		performanceSchemaMemoryCurrentCountUsed: *NewperformanceSchemaMemoryCurrentCountUsed(),
		performanceSchemaMemoryHighCountUsed:    *NewperformanceSchemaMemoryHighCountUsed(),
		perforanceSchemaMemoryLowUsedBytesDesc:  *NewPerformanceSchemaMemoryLowUsedBytesDesc(),
		perforanceSchemaMemoryHighUsedBytesDesc: *NewPerformanceSchemaMemoryHighUsedBytesDesc(),
	}
}

func (qd ScrapePerfMemoryEvents) Collect(ch chan<- prometheus.Metric) {
	qd.instance = *GetInstance()

	if err := qd.instance.Ping(); err != nil {
		logrus.Errorf("ping mysql instance error: %s", err)
		return
	}
	db := instance.GetDB()
	rows, err := db.Query(perfMemoryEventsQuery)
	if err != nil {
		logrus.Error(err)
		return
	}
	defer rows.Close()
	var (
		eventName            string
		countAlloc           uint64
		countFree            uint64
		bytesAlloc           uint64
		bytesFree            uint64
		lowCountUsed         int64
		currentCountUsed     int64
		highCountUsed        int64
		lowNumberOfBytesUsed int64
		currentBytes         int64

		highNumberOfBytesUsed int64
	)
	for rows.Next() {
		err := rows.Scan(&eventName,
			&countAlloc,
			&countFree,
			&bytesAlloc,
			&bytesFree,
			&lowCountUsed,
			&currentCountUsed,
			&highCountUsed,
			&lowNumberOfBytesUsed,
			&currentBytes,
			&highNumberOfBytesUsed)
		if err != nil {
			logrus.Error(err)
			return
		}
		eventName := strings.TrimPrefix(eventName,
			*performanceSchemaMemoryEventsRemovePrefix)
		qd.performanceSchemaMemoryCountAlloc.collect(
			ch,
			float64(countAlloc),
			[]string{
				eventName})
		qd.performanceSchemaMemoryCountFree.collect(
			ch,
			float64(countFree),
			[]string{
				eventName})
		qd.performanceSchemaMemoryBytesAllocDesc.collect(
			ch,
			float64(bytesAlloc),
			[]string{
				eventName})
		qd.performanceSchemaMemoryBytesFreeDesc.collect(
			ch,
			float64(bytesFree),
			[]string{
				eventName})
		qd.performanceSchemaMemoryLowCountUsed.collect(
			ch,
			float64(lowCountUsed),
			[]string{
				eventName})
		qd.performanceSchemaMemoryCurrentCountUsed.collect(
			ch,
			float64(currentCountUsed),
			[]string{
				eventName})
		qd.performanceSchemaMemoryHighCountUsed.collect(
			ch,
			float64(highCountUsed),
			[]string{
				eventName})
		qd.perforanceSchemaMemoryLowUsedBytesDesc.collect(
			ch,
			float64(lowNumberOfBytesUsed),
			[]string{
				eventName})
		qd.perforanceSchemaMemoryUsedBytesDesc.collect(
			ch,
			float64(currentBytes),
			[]string{
				eventName})
		qd.perforanceSchemaMemoryHighUsedBytesDesc.collect(
			ch,
			float64(highNumberOfBytesUsed),
			[]string{
				eventName})
	}
}

type performanceSchemaMemoryBytesAllocDesc struct {
	*baseMetrics
}

func newperformanceSchemaMemoryBytesAllocDesc() *performanceSchemaMemoryBytesAllocDesc {
	return &performanceSchemaMemoryBytesAllocDesc{
		NewMetrics(
			"perf_schema_memory_events_alloc_bytes_total",
			"The total number of bytes allocated by events.",
			[]string{
				"event_name"})}
}

func (qd *performanceSchemaMemoryBytesAllocDesc) Collect(
	ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaMemoryBytesFreeDesc struct {
	*baseMetrics
}

func NewperformanceSchemaMemoryBytesFreeDesc() *performanceSchemaMemoryBytesFreeDesc {
	return &performanceSchemaMemoryBytesFreeDesc{
		NewMetrics(
			"perf_schema_memory_events_free_bytes_total",
			"The total number of bytes freed by events.",
			[]string{
				"event_name"})}
}

func (qd *performanceSchemaMemoryBytesFreeDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type perforanceSchemaMemoryUsedBytesDesc struct {
	*baseMetrics
}

func NewPerformanceSchemaMemoryUsedBytesDesc() *perforanceSchemaMemoryUsedBytesDesc {
	return &perforanceSchemaMemoryUsedBytesDesc{
		NewMetrics(
			"perf_schema_memory_events_used_bytes_total",
			"The total number of bytes used by events.",
			[]string{
				"event_name"})}
}
func (qd *perforanceSchemaMemoryUsedBytesDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaMemoryCountAlloc struct {
	*baseMetrics
}

func NewPerformanceSchemaMemoryCountAlloc() *performanceSchemaMemoryCountAlloc {
	return &performanceSchemaMemoryCountAlloc{
		NewMetrics(
			"perf_schema_memory_events_count_alloc_total",
			"The total number of allocations.",
			[]string{
				"event_name"})}
}
func (qd *performanceSchemaMemoryCountAlloc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaMemoryCountFree struct {
	*baseMetrics
}

func NewperformanceSchemaMemoryCountFree() *performanceSchemaMemoryCountFree {
	return &performanceSchemaMemoryCountFree{
		NewMetrics(
			"perf_schema_memory_events_count_free_total",
			"The total number of frees.",
			[]string{
				"event_name"})}
}
func (qd *performanceSchemaMemoryCountFree) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaMemoryLowCountUsed struct {
	*baseMetrics
}

func NewperformanceSchemaMemoryLowCountUsed() *performanceSchemaMemoryLowCountUsed {
	return &performanceSchemaMemoryLowCountUsed{
		NewMetrics(
			"perf_schema_memory_events_low_count_used_total",
			"The total number of low memory events.",
			[]string{
				"event_name"})}
}
func (qd *performanceSchemaMemoryLowCountUsed) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaMemoryCurrentCountUsed struct {
	*baseMetrics
}

func NewperformanceSchemaMemoryCurrentCountUsed() *performanceSchemaMemoryCurrentCountUsed {
	return &performanceSchemaMemoryCurrentCountUsed{
		NewMetrics(
			"perf_schema_memory_events_current_count_used_total",
			"The total number of current memory events.",
			[]string{
				"event_name"})}
}
func (qd *performanceSchemaMemoryCurrentCountUsed) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type performanceSchemaMemoryHighCountUsed struct {
	*baseMetrics
}

func NewperformanceSchemaMemoryHighCountUsed() *performanceSchemaMemoryHighCountUsed {
	return &performanceSchemaMemoryHighCountUsed{
		NewMetrics(
			"perf_schema_memory_events_high_count_used_total",
			"The total number of high memory events.",
			[]string{
				"event_name"})}
}
func (qd *performanceSchemaMemoryHighCountUsed) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type perforanceSchemaMemoryLowUsedBytesDesc struct {
	*baseMetrics
}

func NewPerformanceSchemaMemoryLowUsedBytesDesc() *perforanceSchemaMemoryLowUsedBytesDesc {
	return &perforanceSchemaMemoryLowUsedBytesDesc{
		NewMetrics(
			"perf_schema_memory_events_low_used_bytes_total",
			"The total number of low memory events.",
			[]string{
				"event_name"})}
}
func (qd *perforanceSchemaMemoryLowUsedBytesDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type perforanceSchemaMemoryHighUsedBytesDesc struct {
	*baseMetrics
}

func NewPerformanceSchemaMemoryHighUsedBytesDesc() *perforanceSchemaMemoryHighUsedBytesDesc {
	return &perforanceSchemaMemoryHighUsedBytesDesc{
		NewMetrics(
			"perf_schema_memory_events_high_used_bytes_total",
			"The total number of high memory events.",
			[]string{
				"event_name"})}
}
func (qd *perforanceSchemaMemoryHighUsedBytesDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

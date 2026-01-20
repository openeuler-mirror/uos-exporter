package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"mysqld_exporter/internal/exporter"
	"mysqld_exporter/internal/mysql"
	"strconv"
	"strings"
)

const (
	queryResponseCheckQuery      = `SELECT @@query_response_time_stats`
	queryResponseTimeQueriesTime = `
SELECT 
    TIME, 
    COUNT, 
    TOTAL 
FROM INFORMATION_SCHEMA.QUERY_RESPONSE_TIME`
	queryResponseTimeQueriesTimeRead = `
SELECT 
    TIME,
    COUNT,
    TOTAL
FROM INFORMATION_SCHEMA.QUERY_RESPONSE_TIME_READ`
	queryResponseTimeQueriesTimeWrite = `
SELECT 
    TIME,
    COUNT,
    TOTAL
FROM INFORMATION_SCHEMA.QUERY_RESPONSE_TIME_WRITE`
)

type ScrapeQueryResponseTime struct {
	instance mysql.Instance
	infoSchemaquery_response_time_seconds
	infoSchemaread_query_response_time_seconds
	infoSchemawrite_query_response_time_seconds
}

func init() {
	exporter.Register(
		NewScrapeQueryResponseTime())
}
func NewScrapeQueryResponseTime() *ScrapeQueryResponseTime {
	return &ScrapeQueryResponseTime{
		//instance:                                    instance,
		infoSchemaquery_response_time_seconds:       *NewinfoSchemaquery_response_time_seconds(),
		infoSchemaread_query_response_time_seconds:  *NewinfoSchemaread_query_response_time_seconds(),
		infoSchemawrite_query_response_time_seconds: *NewinfoSchemawrite_query_response_time_seconds(),
	}
}
func (qd ScrapeQueryResponseTime) Collect(ch chan<- prometheus.Metric) {
	qd.instance = *GetInstance()

	if err := qd.instance.Ping(); err != nil {
		logrus.Errorf("ping mysql instance error: %s", err)
		return
	}
	db := instance.GetDB()
	var queryStats uint8
	err := db.QueryRow(queryResponseCheckQuery).Scan(&queryStats)
	if err != nil {
		logrus.Debugf("Detailed query response time stats are not available.")
		return
	}
	if queryStats == 0 {
		logrus.Debug("MySQL variable is OFF.", "var", "query_response_time_stats")
		return
	}
	queryDistributionRowsTime, err := db.Query(queryResponseTimeQueriesTime)
	if err != nil {
		logrus.Errorf("query mysql instance userstat error: %s", err)
		return
	}
	defer queryDistributionRowsTime.Close()
	var (
		length       string
		count        uint64
		total        string
		histogramCnt uint64
		histogramSum float64
		countBuckets = map[float64]uint64{}
	)
	for queryDistributionRowsTime.Next() {
		err = queryDistributionRowsTime.Scan(
			&length,
			&count,
			&total,
		)
		if err != nil {
			logrus.Errorf("query mysql instance userstat error: %s", err)
			return
		}

		length, _ := strconv.ParseFloat(strings.TrimSpace(length), 64)
		total, _ := strconv.ParseFloat(strings.TrimSpace(total), 64)
		histogramCnt += count
		histogramSum += total
		if length == 0 {
			continue
		}
		countBuckets[length] = histogramCnt
	}
	qd.infoSchemaquery_response_time_seconds.Collect(ch,
		float64(histogramSum),
		nil)
	queryDistributionRowsTimeRead, err := db.Query(queryResponseTimeQueriesTimeRead)
	if err != nil {
		logrus.Errorf("query mysql instance userstat error: %s", err)
		return
	}
	defer queryDistributionRowsTimeRead.Close()
	for queryDistributionRowsTimeRead.Next() {
		err = queryDistributionRowsTimeRead.Scan(
			&length,
			&count,
			&total,
		)
		if err != nil {
			logrus.Errorf("query mysql instance userstat error: %s", err)
			return
		}

		length, _ := strconv.ParseFloat(strings.TrimSpace(length), 64)
		total, _ := strconv.ParseFloat(strings.TrimSpace(total), 64)
		histogramCnt += count
		histogramSum += total
		if length == 0 {
			continue
		}
		countBuckets[length] = histogramCnt
	}
	qd.infoSchemaread_query_response_time_seconds.Collect(ch,
		float64(histogramSum),
		nil)

	queryDistributionRowsTimeWrite, err := db.Query(queryResponseTimeQueriesTimeWrite)
	if err != nil {
		logrus.Errorf("query mysql instance userstat error: %s", err)
		return
	}
	defer queryDistributionRowsTimeWrite.Close()
	for queryDistributionRowsTimeWrite.Next() {
		err = queryDistributionRowsTimeWrite.Scan(
			&length,
			&count,
			&total,
		)
		if err != nil {
			logrus.Errorf("query mysql instance userstat error: %s", err)
			return
		}

		length, _ := strconv.ParseFloat(strings.TrimSpace(length), 64)
		total, _ := strconv.ParseFloat(strings.TrimSpace(total), 64)
		histogramCnt += count
		histogramSum += total
		if length == 0 {
			continue
		}
		countBuckets[length] = histogramCnt
	}
	qd.infoSchemawrite_query_response_time_seconds.Collect(ch,
		float64(histogramSum),
		nil)

}

type infoSchemaquery_response_time_seconds struct {
	*baseMetrics
}

func NewinfoSchemaquery_response_time_seconds() *infoSchemaquery_response_time_seconds {
	return &infoSchemaquery_response_time_seconds{
		NewMetrics(
			"info_schema_query_response_time_seconds",
			"The number of all queries by duration they took to execute.",
			nil)}
}

func (qd *infoSchemaquery_response_time_seconds) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaread_query_response_time_seconds struct {
	*baseMetrics
}

func NewinfoSchemaread_query_response_time_seconds() *infoSchemaread_query_response_time_seconds {
	return &infoSchemaread_query_response_time_seconds{
		NewMetrics(
			"info_schema_read_query_response_time_seconds",
			"The number of read queries by duration they took to execute.",
			nil)}
}
func (qd *infoSchemaread_query_response_time_seconds) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemawrite_query_response_time_seconds struct {
	*baseMetrics
}

func NewinfoSchemawrite_query_response_time_seconds() *infoSchemawrite_query_response_time_seconds {
	return &infoSchemawrite_query_response_time_seconds{
		NewMetrics(
			"info_schema_write_query_response_time_seconds",
			"The number of write queries by duration they took to execute.",
			nil)}
}
func (qd *infoSchemawrite_query_response_time_seconds) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

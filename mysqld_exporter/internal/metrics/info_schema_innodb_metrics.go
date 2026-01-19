package metrics

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"mysqld_exporter/internal/exporter"
	"mysqld_exporter/internal/mysql"
	"regexp"
)

const (
	infoSchemaInnodbMetricsEnabledColumnQuery = `
	SELECT
	    column_name
	  FROM information_schema.columns
	  WHERE table_schema = 'information_schema'
	    AND table_name = 'INNODB_METRICS'
	    AND column_name IN ('status', 'enabled')
	  LIMIT 1
	`
	infoSchemaInnodbMetricsEnabledColumnResult = `
+-------------+
| COLUMN_NAME |
+-------------+
| STATUS      |
+-------------+

`
	infoSchemaInnodbMetricsQuery = `
		SELECT
		  name, subsystem, type, comment,
		  count
		  FROM information_schema.innodb_metrics
		  WHERE ` + "`%s` = '%s'"
)

var (
	bufferRE     = regexp.MustCompile(`^buffer_(pool_pages)_(.*)$`)
	bufferPageRE = regexp.MustCompile(`^buffer_page_(read|written)_(.*)$`)
)

type ScrapeInnodbMetrics struct {
	instance mysql.Instance
	infoSchemaBufferPageReadTotalDesc
	infoSchemaBufferPageWrittenTotalDesc
	infoSchemaBufferPoolPagesDesc
	infoSchemaBufferPoolPagesDirtyDesc
}

func init() {
	exporter.Register(
		NewScrapeInnodbMetrics())
}

func NewScrapeInnodbMetrics() *ScrapeInnodbMetrics {
	return &ScrapeInnodbMetrics{
		//instance:                             instance,
		infoSchemaBufferPageReadTotalDesc:    *NewinfoSchemaBufferPageReadTotalDesc(),
		infoSchemaBufferPageWrittenTotalDesc: *NewinfoSchemaBufferPageWrittenTotalDesc(),
		infoSchemaBufferPoolPagesDesc:        *NewinfoSchemaBufferPoolPagesDesc(),
		infoSchemaBufferPoolPagesDirtyDesc:   *NewinfoSchemaBufferPoolPagesDirtyDesc(),
	}
}

func (qd ScrapeInnodbMetrics) Collect(ch chan<- prometheus.Metric) {
	qd.instance = *GetInstance()

	var (
		enabledColumnName string
		query             string
	)
	if err := qd.instance.Ping(); err != nil {
		logrus.Errorf("ping mysql instance error: %s", err)
		return
	}
	db := instance.GetDB()
	err := db.QueryRow(infoSchemaInnodbMetricsEnabledColumnQuery).Scan(&enabledColumnName)
	if err != nil {
		logrus.Error(err)
		return
	}
	switch enabledColumnName {
	case "STATUS":
		query = fmt.Sprintf(infoSchemaInnodbMetricsQuery, "status", "enabled")
	case "ENABLED":
		query = fmt.Sprintf(infoSchemaInnodbMetricsQuery, "enabled", "1")
	default:
		logrus.Info("Couldn't find column STATUS or ENABLED in innodb_metrics table.")
		return
	}
	innodbMetricsRows, err := db.Query(query)
	if err != nil {
		logrus.Error(err)
		return
	}
	defer innodbMetricsRows.Close()

	var (
		namespace         = "mysql"
		informationSchema = "info_schema"
		name              string
		subsystem         string
		metricType        string
		comment           string
		value             float64
	)
	for innodbMetricsRows.Next() {
		if err := innodbMetricsRows.Scan(
			&name,
			&subsystem,
			&metricType,
			&comment,
			&value,
		); err != nil {
			logrus.Error(err)
			return
		}
		if subsystem == "buffer_page_io" {
			match := bufferPageRE.FindStringSubmatch(name)
			if len(match) != 3 {
				logrus.Warn("innodb_metrics subsystem buffer_page_io returned an invalid name", "name", name)
				continue
			}
			switch match[1] {
			case "read":
				qd.infoSchemaBufferPageReadTotalDesc.Collect(ch,
					value,
					[]string{
						match[2],
					})
			case "written":
				qd.infoSchemaBufferPageWrittenTotalDesc.Collect(ch,
					value,
					[]string{
						match[2],
					})
			}
			continue
		}
		if subsystem == "buffer" {
			match := bufferRE.FindStringSubmatch(name)
			if match != nil {
				switch match[1] {
				case "pool_pages":
					switch match[2] {
					case "total":
						continue
					case "dirty":
						qd.infoSchemaBufferPoolPagesDirtyDesc.Collect(ch,
							value,
							nil)

					default:
						qd.infoSchemaBufferPoolPagesDesc.Collect(ch,
							value,
							[]string{
								match[2],
							})
					}
				}
				continue
			}
		}
		metricName := "innodb_metrics_" + subsystem + "_" + name

		if (metricType == "counter" || metricType == "status_counter") && value >= 0 {
			description := prometheus.NewDesc(
				prometheus.BuildFQName(namespace, informationSchema, metricName+"_total"),
				comment, nil, nil,
			)
			ch <- prometheus.MustNewConstMetric(
				description,
				prometheus.CounterValue,
				value,
			)
		} else {
			description := prometheus.NewDesc(
				prometheus.BuildFQName(namespace, informationSchema, metricName),
				comment, nil, nil,
			)
			ch <- prometheus.MustNewConstMetric(
				description,
				prometheus.GaugeValue,
				value,
			)
		}
	}

}

type infoSchemaBufferPageReadTotalDesc struct {
	*baseMetrics
}

func NewinfoSchemaBufferPageReadTotalDesc() *infoSchemaBufferPageReadTotalDesc {
	return &infoSchemaBufferPageReadTotalDesc{
		NewMetrics(
			"info_schema_innodb_metrics_buffer_page_read_total",
			"Total number of buffer pages read total.",
			[]string{
				"type"})}
}
func (qd *infoSchemaBufferPageReadTotalDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaBufferPageWrittenTotalDesc struct {
	*baseMetrics
}

func NewinfoSchemaBufferPageWrittenTotalDesc() *infoSchemaBufferPageWrittenTotalDesc {
	return &infoSchemaBufferPageWrittenTotalDesc{
		NewMetrics(
			"info_schema_innodb_metrics_buffer_page_written_total",
			"Total number of buffer pages written total.",
			[]string{
				"type"})}
}
func (qd *infoSchemaBufferPageWrittenTotalDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaBufferPoolPagesDesc struct {
	*baseMetrics
}

func NewinfoSchemaBufferPoolPagesDesc() *infoSchemaBufferPoolPagesDesc {
	return &infoSchemaBufferPoolPagesDesc{
		NewMetrics(
			"info_schema_innodb_metrics_buffer_pool_pages",
			"Number of buffer pool pages.",
			[]string{
				"state"})}
}
func (qd *infoSchemaBufferPoolPagesDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaBufferPoolPagesDirtyDesc struct {
	*baseMetrics
}

func NewinfoSchemaBufferPoolPagesDirtyDesc() *infoSchemaBufferPoolPagesDirtyDesc {
	return &infoSchemaBufferPoolPagesDirtyDesc{
		NewMetrics(
			"info_schema_innodb_metrics_buffer_pool_pages_dirty",
			"Number of buffer pool pages dirty.",
			nil)}
}
func (qd *infoSchemaBufferPoolPagesDirtyDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"mysqld_exporter/internal/exporter"
	"mysqld_exporter/internal/mysql"
)

const (
	tableStatQuery = `
		SELECT
		  TABLE_SCHEMA,
		  TABLE_NAME,
		  ROWS_READ,
		  ROWS_CHANGED,
		  ROWS_CHANGED_X_INDEXES
		  FROM information_schema.table_statistics
		`
	tableStatResult = `
`
)

func init() {
	exporter.Register(
		NewScrapeTableStat())
}

type ScrapeTableStat struct {
	instance mysql.Instance
	infoSchemaTableStatsRowsReadDesc
	infoSchemaTableStatsRowsChangedDesc
	infoSchemaTableStatsRowsChangedXIndexesDesc
}

func NewScrapeTableStat() *ScrapeTableStat {
	return &ScrapeTableStat{
		//instance:                                    instance,
		infoSchemaTableStatsRowsReadDesc:            *NewinfoSchemaTableStatsRowsReadDesc(),
		infoSchemaTableStatsRowsChangedDesc:         *NewinfoSchemaTableStatsRowsChangedDesc(),
		infoSchemaTableStatsRowsChangedXIndexesDesc: *NewinfoSchemaTableStatsRowsChangedXIndexesDesc(),
	}
}

func (qd ScrapeTableStat) Collect(ch chan<- prometheus.Metric) {
	qd.instance = *GetInstance()

	if err := qd.instance.Ping(); err != nil {
		logrus.Errorf("ping mysql instance error: %s", err)
		return
	}
	db := instance.GetDB()
	rows, err := db.Query(userstatCheckQuery)
	if err != nil {
		logrus.Debugf("failed check mysql instance userstat: %s",
			err)
		return
	}
	defer rows.Close()
	var (
		varName  string
		varValue string
	)
	err = rows.Scan(&varName, &varValue)
	if err != nil {
		logrus.Debugf("failed to scan mysql instance userstat: %s",
			err)
		return
	}
	if varValue == "OFF" {
		logrus.Debugf("mysql instance userstat is disabled")
		return
	}
	informationSchemaTableStatisticsRows, err := db.Query(tableStatQuery)
	if err != nil {
		logrus.Errorf("query mysql instance userstat error: %s", err)
		return
	}
	defer informationSchemaTableStatisticsRows.Close()
	var (
		tableSchema         string
		tableName           string
		rowsRead            uint64
		rowsChanged         uint64
		rowsChangedXIndexes uint64
	)

	for informationSchemaTableStatisticsRows.Next() {
		err = informationSchemaTableStatisticsRows.Scan(
			&tableSchema,
			&tableName,
			&rowsRead,
			&rowsChanged,
			&rowsChangedXIndexes,
		)
		if err != nil {
			logrus.Debugf("failed to scan mysql instance userstat: %s",
				err)
			return
		}
		qd.infoSchemaTableStatsRowsReadDesc.Collect(ch,
			float64(rowsRead),
			[]string{
				tableSchema,
				tableName})
		qd.infoSchemaTableStatsRowsChangedDesc.Collect(ch,
			float64(rowsChanged),
			[]string{
				tableSchema,
				tableName})
		qd.infoSchemaTableStatsRowsChangedXIndexesDesc.Collect(ch,
			float64(rowsChangedXIndexes),
			[]string{
				tableSchema,
				tableName})
	}
}

type infoSchemaTableStatsRowsReadDesc struct {
	*baseMetrics
}

func NewinfoSchemaTableStatsRowsReadDesc() *infoSchemaTableStatsRowsReadDesc {
	return &infoSchemaTableStatsRowsReadDesc{
		NewMetrics(
			"info_schema_table_statistics_rows_read_total",
			"The number of rows read from the table.",
			[]string{
				"schema",
				"table"})}
}
func (qd *infoSchemaTableStatsRowsReadDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaTableStatsRowsChangedDesc struct {
	*baseMetrics
}

func NewinfoSchemaTableStatsRowsChangedDesc() *infoSchemaTableStatsRowsChangedDesc {
	return &infoSchemaTableStatsRowsChangedDesc{
		NewMetrics(
			"info_schema_table_statistics_rows_changed_total",
			"The number of rows changed in the table.",
			[]string{
				"schema",
				"table"})}
}

func (qd *infoSchemaTableStatsRowsChangedDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaTableStatsRowsChangedXIndexesDesc struct {
	*baseMetrics
}

func NewinfoSchemaTableStatsRowsChangedXIndexesDesc() *infoSchemaTableStatsRowsChangedXIndexesDesc {
	return &infoSchemaTableStatsRowsChangedXIndexesDesc{
		NewMetrics(
			"info_schema_table_statistics_rows_changed_x_indexes_total",
			"The number of rows changed in the table by indexes.",
			[]string{
				"schema",
				"table"})}
}
func (qd *infoSchemaTableStatsRowsChangedXIndexesDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

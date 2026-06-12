package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"mysqld_exporter/internal/exporter"
	"mysqld_exporter/internal/mysql"
)

const (
	schemaStatQuery = `
		SELECT 
			TABLE_SCHEMA, 
			SUM(ROWS_READ) AS ROWS_READ, 
			SUM(ROWS_CHANGED) AS ROWS_CHANGED, 
			SUM(ROWS_CHANGED_X_INDEXES) AS ROWS_CHANGED_X_INDEXES 
		FROM information_schema.TABLE_STATISTICS 
		GROUP BY TABLE_SCHEMA;
		`
	schemaStatResult = `
`
)

func init() {
	exporter.Register(
		NewScrapeSchemaStat())
}

type ScrapeSchemaStat struct {
	instance mysql.Instance
	infoSchemaStatsRowsReadDesc
	infoSchemaStatsRowsChangedDesc
	infoSchemaStatsRowsChangedXIndexesDesc
}

func NewScrapeSchemaStat() *ScrapeSchemaStat {
	return &ScrapeSchemaStat{
		//instance:                               instance,
		infoSchemaStatsRowsReadDesc:            *NewinfoSchemaStatsRowsReadDesc(),
		infoSchemaStatsRowsChangedDesc:         *NewinfoSchemaStatsRowsChangedDesc(),
		infoSchemaStatsRowsChangedXIndexesDesc: *NewinfoSchemaStatsRowsChangedXIndexesDesc(),
	}
}

func (qd ScrapeSchemaStat) Collect(ch chan<- prometheus.Metric) {
	qd.instance = *GetInstance()
	if err := qd.instance.Ping(); err != nil {
		logrus.Errorf("ping mysql instance error: %s", err)
		return
	}
	db := instance.GetDB()
	var (
		varName string
		varVal  string
	)
	err := db.QueryRow(userstatCheckQuery).Scan(
		&varName,
		&varVal)
	if err != nil {
		logrus.Debugf("Detailed schema stats are not available.")
		return
	}
	if varVal == "OFF" {
		logrus.Debug("MySQL variable is OFF.", "var", varName)
		return
	}
	informationSchemaTableStatisticsRows, err := db.Query(schemaStatQuery)
	if err != nil {
		logrus.Errorf("query mysql instance userstat error: %s", err)
		return
	}
	defer informationSchemaTableStatisticsRows.Close()
	var (
		tableSchema         string
		rowsRead            uint64
		rowsChanged         uint64
		rowsChangedXIndexes uint64
	)
	for informationSchemaTableStatisticsRows.Next() {
		err = informationSchemaTableStatisticsRows.Scan(
			&tableSchema,
			&rowsRead,
			&rowsChanged,
			&rowsChangedXIndexes,
		)

		if err != nil {
			logrus.Errorf("failed to scan mysql instance userstat: %s", err)
			return
		}
		qd.infoSchemaStatsRowsReadDesc.Collect(ch,
			float64(rowsRead),
			[]string{
				tableSchema})
		qd.infoSchemaStatsRowsChangedDesc.Collect(ch,
			float64(rowsChanged),
			[]string{
				tableSchema})
		qd.infoSchemaStatsRowsChangedXIndexesDesc.Collect(ch,
			float64(rowsChangedXIndexes),
			[]string{
				tableSchema})
	}
}

type infoSchemaStatsRowsReadDesc struct {
	*baseMetrics
}

func NewinfoSchemaStatsRowsReadDesc() *infoSchemaStatsRowsReadDesc {
	return &infoSchemaStatsRowsReadDesc{
		NewMetrics(
			"info_schema_schema_statistics_rows_read_total",
			"The number of rows read from the schema.",
			[]string{
				"schema"})}
}

func (qd *infoSchemaStatsRowsReadDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaStatsRowsChangedDesc struct {
	*baseMetrics
}

func NewinfoSchemaStatsRowsChangedDesc() *infoSchemaStatsRowsChangedDesc {
	return &infoSchemaStatsRowsChangedDesc{
		NewMetrics(
			"info_schema_schema_statistics_rows_changed_total",
			"The number of rows changed in the schema.",
			[]string{
				"schema"})}
}

func (qd *infoSchemaStatsRowsChangedDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaStatsRowsChangedXIndexesDesc struct {
	*baseMetrics
}

func NewinfoSchemaStatsRowsChangedXIndexesDesc() *infoSchemaStatsRowsChangedXIndexesDesc {
	return &infoSchemaStatsRowsChangedXIndexesDesc{
		NewMetrics(
			"info_schema_schema_statistics_rows_changed_x_indexes_total",
			"The number of rows changed in the schema by indexes.",
			[]string{
				"schema"})}
}

func (qd *infoSchemaStatsRowsChangedXIndexesDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

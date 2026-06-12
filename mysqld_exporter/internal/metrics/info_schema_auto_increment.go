package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"mysqld_exporter/internal/exporter"
	"mysqld_exporter/internal/mysql"
)

const (
	infoSchemaAutoIncrementQuery = `
		SELECT c.table_schema, c.table_name, column_name, auto_increment,
		  pow(2, case data_type
		    when 'tinyint'   then 7
		    when 'smallint'  then 15
		    when 'mediumint' then 23
		    when 'int'       then 31
		    when 'bigint'    then 63
		    end+(column_type like '% unsigned'))-1 as max_int
		  FROM information_schema.columns c
		  STRAIGHT_JOIN information_schema.tables t ON (BINARY c.table_schema=t.table_schema AND BINARY c.table_name=t.table_name)
		  WHERE c.extra = 'auto_increment' AND t.auto_increment IS NOT NULL
		`
	infoSchemaAutoIncrementQueryResult = `
	*************************** 1. row ***************************
  TABLE_SCHEMA: mysql
    TABLE_NAME: time_zone
   COLUMN_NAME: Time_zone_id
AUTO_INCREMENT: 1
       max_int: 4294967295
*************************** 2. row ***************************
  TABLE_SCHEMA: mysql
    TABLE_NAME: component
   COLUMN_NAME: component_id
AUTO_INCREMENT: 1
       max_int: 4294967295
2 rows in set, 2 warnings (0.014 sec)
`
)

type ScrapeAutoIncrementColumns struct {
	instance mysql.Instance
	InfoSchemaAutoIncrement
	InfoSchemaAutoIncrementMax
}

func NewScrapeAutoIncrementColumns() *ScrapeAutoIncrementColumns {
	return &ScrapeAutoIncrementColumns{
		//instance:                   instance,
		InfoSchemaAutoIncrement:    *NewInfoSchemaAutoIncrement(),
		InfoSchemaAutoIncrementMax: *NewInfoSchemaAutoIncrementMax(),
	}
}
func init() {
	exporter.Register(
		NewScrapeAutoIncrementColumns())
}
func (qd ScrapeAutoIncrementColumns) Collect(ch chan<- prometheus.Metric) {
	qd.instance = *GetInstance()
	if err := qd.instance.Ping(); err != nil {
		logrus.Errorf("ping mysql instance error: %s", err)
		return
	}
	db := instance.GetDB()
	rows, err := db.Query(infoSchemaAutoIncrementQuery)
	if err != nil {
		logrus.Error(err)
		return
	}
	defer rows.Close()
	var (
		schema string
		table  string
		column string
		value  float64
		max    float64
	)
	for rows.Next() {
		err := rows.Scan(
			&schema,
			&table,
			&column,
			&value,
			&max,
		)
		if err != nil {
			logrus.Error(err)
			return
		}
		qd.InfoSchemaAutoIncrement.Collect(ch,
			value,
			[]string{
				schema,
				table,
				column,
			})
		qd.InfoSchemaAutoIncrementMax.Collect(ch,
			max,
			[]string{
				schema,
				table,
				column,
			})
	}
	return
}

type InfoSchemaAutoIncrement struct {
	*baseMetrics
}

func NewInfoSchemaAutoIncrement() *InfoSchemaAutoIncrement {
	return &InfoSchemaAutoIncrement{
		NewMetrics(
			"info_schema_auto_increment_column",
			"The current value of an auto_increment column from information_schema.",
			[]string{
				"schema",
				"table",
				"column",
			})}
}
func (qd *InfoSchemaAutoIncrement) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type InfoSchemaAutoIncrementMax struct {
	*baseMetrics
}

func NewInfoSchemaAutoIncrementMax() *InfoSchemaAutoIncrementMax {
	return &InfoSchemaAutoIncrementMax{
		NewMetrics(
			"info_schema_auto_increment_max",
			"The maximum value of an auto_increment column from information_schema.",
			[]string{
				"schema",
				"table",
				"column",
			})}
}
func (qd *InfoSchemaAutoIncrementMax) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

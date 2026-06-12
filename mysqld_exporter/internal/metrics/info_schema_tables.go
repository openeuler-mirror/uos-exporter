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
	tableSchemaQuery = `
		SELECT
		    TABLE_SCHEMA,
		    TABLE_NAME,
		    TABLE_TYPE,
		    ifnull(ENGINE, 'NONE') as ENGINE,
		    ifnull(VERSION, '0') as VERSION,
		    ifnull(ROW_FORMAT, 'NONE') as ROW_FORMAT,
		    ifnull(TABLE_ROWS, '0') as TABLE_ROWS,
		    ifnull(DATA_LENGTH, '0') as DATA_LENGTH,
		    ifnull(INDEX_LENGTH, '0') as INDEX_LENGTH,
		    ifnull(DATA_FREE, '0') as DATA_FREE,
		    ifnull(CREATE_OPTIONS, 'NONE') as CREATE_OPTIONS
		  FROM information_schema.tables
		  WHERE TABLE_SCHEMA = ?
		`
	dbListQuery = `
		SELECT
		    SCHEMA_NAME
		  FROM information_schema.schemata
		  WHERE SCHEMA_NAME NOT IN ('mysql', 'performance_schema', 'information_schema', 'sys')
		`
)

var (
	tableSchemaDatabases = kingpin.Flag(
		"collect.info_schema.tables.databases",
		"The list of databases to collect table stats for, or '*' for all",
	).Default("*").String()
)

func init() {
	exporter.Register(
		NewScrapeTableSchema())
}

type ScrapeTableSchema struct {
	instance mysql.Instance
	infoSchemaTablesVersionDesc
	infoSchemaTablesRowsDesc
	infoSchemaTablesSizeDesc
}

func NewScrapeTableSchema() *ScrapeTableSchema {
	return &ScrapeTableSchema{
		//instance:                    instance,
		infoSchemaTablesVersionDesc: *NewinfoSchemaTablesVersionDesc(),
		infoSchemaTablesRowsDesc:    *NewinfoSchemaTablesRowsDesc(),
		infoSchemaTablesSizeDesc:    *NewinfoSchemaTablesSizeDesc(),
	}
}

func (qd ScrapeTableSchema) Collect(ch chan<- prometheus.Metric) {
	qd.instance = *GetInstance()
	if err := qd.instance.Ping(); err != nil {
		logrus.Errorf("ping mysql instance error: %s", err)
		return
	}
	db := instance.GetDB()
	var dbList []string

	if *tableSchemaDatabases == "*" {
		dbListRows, err := db.Query(dbListQuery)
		if err != nil {
			logrus.Debugf("failed to get dbListRows: %v", err)
			return
		}
		defer dbListRows.Close()

		var database string

		for dbListRows.Next() {
			err := dbListRows.Scan(
				&database,
			)
			if err != nil {
				logrus.Errorf("failed to scan mysql dbListRows: %s", err)
				return
			}
			dbList = append(dbList, database)
		}
	} else {
		dbList = strings.Split(*tableSchemaDatabases, ",")
	}
	for _, database := range dbList {
		tableSchemaRows, err := db.Query(tableSchemaQuery, database)
		if err != nil {
			logrus.Errorf("query mysql instance tableSchema error: %s", err)
			return
		}
		defer tableSchemaRows.Close()
		var (
			tableSchema   string
			tableName     string
			tableType     string
			engine        string
			version       uint64
			rowFormat     string
			tableRows     uint64
			dataLength    uint64
			indexLength   uint64
			dataFree      uint64
			createOptions string
		)

		for tableSchemaRows.Next() {
			err = tableSchemaRows.Scan(
				&tableSchema,
				&tableName,
				&tableType,
				&engine,
				&version,
				&rowFormat,
				&tableRows,
				&dataLength,
				&indexLength,
				&dataFree,
				&createOptions,
			)
			if err != nil {
				logrus.Errorf("failed to scan mysql instance tableSchema: %s", err)
				return
			}
			qd.infoSchemaTablesVersionDesc.Collect(ch,
				float64(version),
				[]string{
					tableSchema,
					tableName,
					tableType,
					engine,
					rowFormat,
					createOptions},
			)
			qd.infoSchemaTablesRowsDesc.Collect(ch,
				float64(tableRows),
				[]string{
					tableSchema,
					tableName,
				})

			qd.infoSchemaTablesSizeDesc.Collect(ch,
				float64(dataLength),
				[]string{
					tableSchema,
					tableName,
					"data_length",
				})

			qd.infoSchemaTablesSizeDesc.Collect(ch,
				float64(indexLength),
				[]string{
					tableSchema,
					tableName,
					"index_length",
				})
			qd.infoSchemaTablesSizeDesc.Collect(ch,
				float64(dataFree),
				[]string{
					tableSchema,
					tableName,
					"data_free",
				})
		}
	}
}

type infoSchemaTablesVersionDesc struct {
	*baseMetrics
}

func NewinfoSchemaTablesVersionDesc() *infoSchemaTablesVersionDesc {
	return &infoSchemaTablesVersionDesc{
		NewMetrics(
			"info_schema_table_version",
			"The version number of the table's .frm file",
			[]string{
				"schema",
				"table",
				"type",
				"engine",
				"row_format",
				"create_options"})}
}

func (qd *infoSchemaTablesVersionDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaTablesRowsDesc struct {
	*baseMetrics
}

func NewinfoSchemaTablesRowsDesc() *infoSchemaTablesRowsDesc {
	return &infoSchemaTablesRowsDesc{
		NewMetrics(
			"info_schema_table_rows",
			"The estimated number of rows in the table "+
				"from information_schema.tables",
			[]string{
				"schema",
				"table"})}
}

func (qd *infoSchemaTablesRowsDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaTablesSizeDesc struct {
	*baseMetrics
}

func NewinfoSchemaTablesSizeDesc() *infoSchemaTablesSizeDesc {
	return &infoSchemaTablesSizeDesc{
		NewMetrics(
			"info_schema_table_size",
			"The size of the table components from "+
				"information_schema.tables",
			[]string{
				"schema",
				"table",
				"component"})}
}

func (qd *infoSchemaTablesSizeDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

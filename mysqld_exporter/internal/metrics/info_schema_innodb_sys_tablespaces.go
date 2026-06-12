package metrics

import (
	"fmt"
	"github.com/Masterminds/semver/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"mysqld_exporter/internal/exporter"
	"mysqld_exporter/internal/mysql"
)

const (
	innodbTablespacesTablenameQuery = `
	SELECT
	    table_name
	  FROM information_schema.tables
	  WHERE table_name = 'INNODB_SYS_TABLESPACES'
	    OR table_name = 'INNODB_TABLESPACES'
	`
	innodbTablespacesTablenameResult = `
+--------------------+
| TABLE_NAME         |
+--------------------+
| INNODB_TABLESPACES |
+--------------------+
`
	innodbTablespacesQueryMySQL = `
	SELECT
	    SPACE,
	    NAME,
	    ifnull((SELECT column_name
			FROM information_schema.COLUMNS
			WHERE TABLE_SCHEMA = 'information_schema'
			  AND TABLE_NAME = ` + "'%s'" + `
			  AND COLUMN_NAME = 'FILE_FORMAT' LIMIT 1), 'NONE') as FILE_FORMAT,
	    ifnull(ROW_FORMAT, 'NONE') as ROW_FORMAT,
	    ifnull(SPACE_TYPE, 'NONE') as SPACE_TYPE,
	    FILE_SIZE,
	    ALLOCATED_SIZE
	  FROM information_schema.` + "`%s`"
	innodbTablespacesQueryMariaDB = `
	SELECT
	    SPACE,
	    NAME,
	    ifnull((SELECT column_name
			FROM information_schema.COLUMNS
			WHERE TABLE_SCHEMA = 'information_schema'
			  AND TABLE_NAME = ` + "'%s'" + `
			  AND COLUMN_NAME = 'FILE_FORMAT' LIMIT 1), 'NONE') as FILE_FORMAT,
	    ifnull(ROW_FORMAT, 'NONE') as ROW_FORMAT,
	    FILE_SIZE,
	    ALLOCATED_SIZE
	  FROM information_schema.` + "`%s`"
)

type ScrapeInfoSchemaInnodbTablespaces struct {
	instance mysql.Instance
	infoSchemaInnodbTablesspaceInfoDesc
	infoSchemaInnodbTablesspaceFileSizeDesc
	infoSchemaInnodbTablesspaceAllocatedSizeDesc
}

func init() {
	exporter.Register(
		NewScrapeInfoSchemaInnodbTablespaces())
}

func NewScrapeInfoSchemaInnodbTablespaces() *ScrapeInfoSchemaInnodbTablespaces {
	return &ScrapeInfoSchemaInnodbTablespaces{
		//instance:                                     instance,
		infoSchemaInnodbTablesspaceInfoDesc:          *NewinfoSchemaInnodbTablesspaceInfoDesc(),
		infoSchemaInnodbTablesspaceFileSizeDesc:      *NewinfoSchemaInnodbTablesspaceFileSizeDesc(),
		infoSchemaInnodbTablesspaceAllocatedSizeDesc: *NewinfoSchemaInnodbTablesspaceAllocatedSizeDesc(),
	}
}

func (qd ScrapeInfoSchemaInnodbTablespaces) Collect(ch chan<- prometheus.Metric) {
	var (
		tablespacesTablename string
		query                string
	)
	qd.instance = *GetInstance()

	if err := qd.instance.Ping(); err != nil {
		logrus.Errorf("ping mysql instance error: %s", err)
		return
	}
	db := instance.GetDB()
	err := db.QueryRow(innodbTablespacesTablenameQuery).Scan(&tablespacesTablename)
	if err != nil {
		logrus.Error(err)
		return
	}
	switch tablespacesTablename {
	case "INNODB_SYS_TABLESPACES", "INNODB_TABLESPACES":
		query = fmt.Sprintf(innodbTablespacesQueryMySQL, tablespacesTablename, tablespacesTablename)
		if instance.GetFlavor() == mysql.MariaDB && instance.GetVersion().GreaterThanEqual(semver.MustParse("10.5.0")) {
			query = fmt.Sprintf(innodbTablespacesQueryMariaDB, tablespacesTablename, tablespacesTablename)
		}
	default:
		logrus.Info("Couldn't find INNODB_SYS_TABLESPACES or INNODB_TABLESPACES in information_schema.")
		return
	}
	rows, err := db.Query(query)
	if err != nil {
		logrus.Error(err)
		return
	}
	defer rows.Close()
	var (
		tableSpace    uint32
		tableName     string
		fileFormat    string
		rowFormat     string
		spaceType     string
		fileSize      uint64
		allocatedSize uint64
	)
	for rows.Next() {
		var err error
		if instance.GetFlavor() == mysql.MariaDB && instance.GetVersion().GreaterThanEqual(semver.MustParse("10.5.0")) {
			err = rows.Scan(
				&tableSpace,
				&tableName,
				&fileFormat,
				&rowFormat,
				&fileSize,
				&allocatedSize,
			)
		} else {
			err = rows.Scan(
				&tableSpace,
				&tableName,
				&fileFormat,
				&rowFormat,
				&spaceType,
				&fileSize,
				&allocatedSize,
			)
		}
		if err != nil {
			logrus.Error(err)
			return
		}
		qd.infoSchemaInnodbTablesspaceInfoDesc.Collect(ch,
			float64(tableSpace),
			[]string{
				tableName,
				fileFormat,
				rowFormat,
				spaceType,
			})
		qd.infoSchemaInnodbTablesspaceFileSizeDesc.Collect(ch,
			float64(fileSize),
			[]string{
				tableName,
			})
		qd.infoSchemaInnodbTablesspaceAllocatedSizeDesc.Collect(ch,
			float64(allocatedSize),
			[]string{
				tableName,
			})
	}
}

type infoSchemaInnodbTablesspaceInfoDesc struct {
	*baseMetrics
}

func NewinfoSchemaInnodbTablesspaceInfoDesc() *infoSchemaInnodbTablesspaceInfoDesc {
	return &infoSchemaInnodbTablesspaceInfoDesc{
		NewMetrics(
			"info_schema_innodb_tablespace_space_info",
			"The Tablespace information and Space ID.",
			[]string{
				"tablespace_name",
				"file_format",
				"row_format",
				"space_type"})}
}

func (qd *infoSchemaInnodbTablesspaceInfoDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaInnodbTablesspaceFileSizeDesc struct {
	*baseMetrics
}

func NewinfoSchemaInnodbTablesspaceFileSizeDesc() *infoSchemaInnodbTablesspaceFileSizeDesc {
	return &infoSchemaInnodbTablesspaceFileSizeDesc{
		NewMetrics(
			"info_schema_innodb_tablespace_file_size",
			"The Tablespace file size.",
			[]string{
				"tablespace_name"})}
}
func (qd *infoSchemaInnodbTablesspaceFileSizeDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaInnodbTablesspaceAllocatedSizeDesc struct {
	*baseMetrics
}

func NewinfoSchemaInnodbTablesspaceAllocatedSizeDesc() *infoSchemaInnodbTablesspaceAllocatedSizeDesc {
	return &infoSchemaInnodbTablesspaceAllocatedSizeDesc{
		NewMetrics(
			"info_schema_innodb_tablespace_allocated_size",
			"The Tablespace allocated size.",
			[]string{
				"tablespace_name"})}
}
func (qd *infoSchemaInnodbTablesspaceAllocatedSizeDesc) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

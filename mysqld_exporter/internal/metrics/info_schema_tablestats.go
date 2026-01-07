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


// TODO: implement

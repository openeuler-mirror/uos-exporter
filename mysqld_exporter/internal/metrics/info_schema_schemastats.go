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


// TODO: implement

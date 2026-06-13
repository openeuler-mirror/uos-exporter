package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"mysqld_exporter/internal/exporter"
	"mysqld_exporter/internal/mysql"
	"time"
)

const (
	perfReplicationApplierStatsByWorkerQuery = `
	SELECT 
	    CHANNEL_NAME,
		WORKER_ID,
		LAST_APPLIED_TRANSACTION_ORIGINAL_COMMIT_TIMESTAMP,
		LAST_APPLIED_TRANSACTION_IMMEDIATE_COMMIT_TIMESTAMP,
		LAST_APPLIED_TRANSACTION_START_APPLY_TIMESTAMP,
		LAST_APPLIED_TRANSACTION_END_APPLY_TIMESTAMP,
		APPLYING_TRANSACTION_ORIGINAL_COMMIT_TIMESTAMP,
		APPLYING_TRANSACTION_IMMEDIATE_COMMIT_TIMESTAMP, 
	  	APPLYING_TRANSACTION_START_APPLY_TIMESTAMP
    FROM performance_schema.replication_applier_status_by_worker
	`
	timeLayout = "2006-01-02 15:04:05.000000"
)


// TODO: implement functions

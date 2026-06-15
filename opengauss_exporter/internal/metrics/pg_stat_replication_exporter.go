package metrics

import (
	"database/sql"

	"opengauss_exporter/internal/collector"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type PgStatReplicationExporter struct {
	db       *sql.DB
	instance string
	uri      string
	metrics  *pgStatReplicationMetrics
}

type pgStatReplicationMetrics struct {
	writeLagMetric      *baseMetrics
	flushLagMetric      *baseMetrics
	replayLagMetric     *baseMetrics
	totalReplicasMetric *baseMetrics
	backendStartMetric  *baseMetrics
}


// TODO: implement functions

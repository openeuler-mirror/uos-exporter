package metrics

import (
	"database/sql"

	"opengauss_exporter/internal/collector"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type OpenGaussInfoExporter struct {
	db       *sql.DB
	instance string
	uri      string
	metrics  *infoMetrics
}

type infoMetrics struct {
	upMetric                *baseMetrics
	versionMetric           *baseMetrics
	uptimeSecondsMetric     *baseMetrics
	databaseCountMetric     *baseMetrics
	connectionCurrentMetric *baseMetrics
	connectionMaxMetric     *baseMetrics
	activeBackendsMetric    *baseMetrics
	idleBackendsMetric      *baseMetrics
	waitingBackendsMetric   *baseMetrics
}

func newInfoMetrics() *infoMetrics {
	return &infoMetrics{
		upMetric: NewMetrics(
			"pg_up",
			"Whether the OpenGauss instance is up.",
			[]string{"instance", "uri"},
		),
		versionMetric: NewMetrics(
			"pg_version",
			"The version of the OpenGauss instance.",
			[]string{"instance", "uri", "version"},
		),
		uptimeSecondsMetric: NewMetrics(
			"pg_uptime_seconds",
			"The uptime of the OpenGauss instance in seconds.",
			[]string{"instance", "uri"},
		),
		databaseCountMetric: NewMetrics(
			"pg_databases_count",
			"The number of databases on this OpenGauss instance.",
			[]string{"instance", "uri"},
		),
		connectionCurrentMetric: NewMetrics(
			"pg_connections_current",
			"The current number of active connections.",
			[]string{"instance", "uri"},
		),
		connectionMaxMetric: NewMetrics(
			"pg_connections_max",
			"The maximum number of allowed connections.",
			[]string{"instance", "uri"},
		),
		activeBackendsMetric: NewMetrics(
			"pg_backends_active",
			"The number of currently active backends (queries running).",
			[]string{"instance", "uri"},
		),
		idleBackendsMetric: NewMetrics(
			"pg_backends_idle",
			"The number of idle backends.",
			[]string{"instance", "uri"},
		),
		waitingBackendsMetric: NewMetrics(
			"pg_backends_waiting",
			"The number of backends waiting for locks.",
			[]string{"instance", "uri"},
		),
	}
}

// NewOpenGaussInfoExporter 创建一个新的 Info Exporter
func NewOpenGaussInfoExporter(db *sql.DB, instance, uri string) *OpenGaussInfoExporter {
	return &OpenGaussInfoExporter{
		db:       db,
		instance: instance,
		uri:      uri,
		metrics:  newInfoMetrics(),
	}
}

// Describe implements Prometheus Collector interface

// Collect implements Prometheus Collector interface
func (e *OpenGaussInfoExporter) collect(ch chan<- prometheus.Metric) {
	labels := []string{e.instance, e.uri}

	meta, err := collector.ScrapeOpenGaussInfo(e.db)
	if err != nil {
		logrus.Debugf("Failed to scrape OpenGauss info: %v", err)
		logrus.Debugf("Info scrape error: %v", err)
		e.metrics.upMetric.collect(
			ch,
			0,
			labels,
		)
		return
	}

	// 上报指标
	e.metrics.upMetric.collect(
		ch,
		1,
		labels,
	)
	e.metrics.versionMetric.collect(
		ch,
		1, append(labels, meta.Version))
	e.metrics.uptimeSecondsMetric.collect(
		ch,
		meta.UptimeSeconds,
		labels,
	)
	e.metrics.databaseCountMetric.collect(
		ch,
		float64(meta.DatabaseCount),
		labels,
	)
	e.metrics.connectionCurrentMetric.collect(
		ch,
		float64(meta.ConnectionCurrent),
		labels,
	)
	e.metrics.connectionMaxMetric.collect(
		ch,
		float64(meta.ConnectionMax),
		labels,
	)
	e.metrics.activeBackendsMetric.collect(
		ch,
		float64(meta.ActiveBackends),
		labels,
	)
	e.metrics.idleBackendsMetric.collect(
		ch,
		float64(meta.IdleBackends),
		labels,
	)
	e.metrics.waitingBackendsMetric.collect(
		ch,
		float64(meta.WaitingBackends),
		labels,
	)
}

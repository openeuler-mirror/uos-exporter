package core

import (
	"errors"
	"podman_exporter/internal/clock"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

//go:generate go run -mod=mod github.com/golang/mock/mockgen --build_flags=-mod=mod -package mock_collector -destination ../test/mock_collector/instrumented_collector.go github.com/ClusterLabs/ha_cluster_exporter/collector InstrumentableCollector

var (
	ErrNoData = errors.New("collector returned no data")
)

type ScrapeCollectorInterface interface {
	prometheus.Collector
	SubsystemCollector
	CollectWithError(ch chan<- prometheus.Metric) error
}

type typedDesc struct {
	desc      *prometheus.Desc
	valueType prometheus.ValueType
}

type ScrapeCollector struct {
	collector          ScrapeCollectorInterface
	Clock              clock.Clock
	scrapeDurationDesc *prometheus.Desc
	scrapeSuccessDesc  *prometheus.Desc
	logger             *logrus.Logger
}

func NewScrapeCollector(collector ScrapeCollectorInterface, logger *logrus.Logger) *ScrapeCollector {
	return &ScrapeCollector{
		collector: collector,
		Clock:     &clock.SystemClock{},
		scrapeDurationDesc: prometheus.NewDesc(
			prometheus.BuildFQName(NAMESPACE, "scrape", "collector_duration_seconds"),
			"podman_prometheus_exporter: Duration of a collector scrape.",
			[]string{"collector"},
			nil,
		),
		scrapeSuccessDesc: prometheus.NewDesc(
			prometheus.BuildFQName(NAMESPACE, "scrape", "collector_success"),
			"podman_prometheus_exporter: Whether a collector succeeded.",
			[]string{"collector"},
			nil,
		),
		logger: logger,
	}
}

func (sc *ScrapeCollector) Collect(ch chan<- prometheus.Metric) {
	var success float64
	begin := sc.Clock.Now()
	err := sc.collector.CollectWithError(ch)
	duration := sc.Clock.Since(begin)

	if err == nil {
		success = 1
		sc.logger.WithFields(logrus.Fields{
			"subsystem": sc.GetSubsystem(),
			"duration":  duration.Seconds(),
		}).Debug("collector succeeded")
	} else {
		success = 0
		if IsNoDataError(err) {
			sc.logger.WithFields(logrus.Fields{
				"subsystem": sc.GetSubsystem(),
				"error":     err,
				"duration":  duration.Seconds(),
			}).Debug("collector returned no data")
		} else {
			sc.logger.WithFields(logrus.Fields{
				"subsystem": sc.GetSubsystem(),
				"error":     err,
				"duration":  duration.Seconds(),
			}).Error("collector scrape failed")
		}
	}

	ch <- prometheus.MustNewConstMetric(sc.scrapeDurationDesc, prometheus.GaugeValue, duration.Seconds(), sc.GetSubsystem())
	ch <- prometheus.MustNewConstMetric(sc.scrapeSuccessDesc, prometheus.GaugeValue, success, sc.GetSubsystem())
}

func (sc *ScrapeCollector) Describe(ch chan<- *prometheus.Desc) {
	sc.collector.Describe(ch)
	ch <- sc.scrapeDurationDesc
	ch <- sc.scrapeSuccessDesc
}

func (sc *ScrapeCollector) GetSubsystem() string {
	return sc.collector.GetSubsystem()
}

// IsNoDataError returns true if error is no data error.
func IsNoDataError(err error) bool {
	return errors.Is(err, ErrNoData)
}

func (d *typedDesc) mustNewConstMetric(value float64, labels ...string) prometheus.Metric {
	return prometheus.MustNewConstMetric(d.desc, d.valueType, value, labels...)
}

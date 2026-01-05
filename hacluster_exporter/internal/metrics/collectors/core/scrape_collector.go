package core

import (
	"hacluster_exporter/internal/clock"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

//go:generate go run -mod=mod github.com/golang/mock/mockgen --build_flags=-mod=mod -package mock_collector -destination ../test/mock_collector/instrumented_collector.go github.com/ClusterLabs/ha_cluster_exporter/collector InstrumentableCollector

// InstrumentableCollector describes a collector that can return errors from collection cycles,
// instead of the default Prometheus one, which has void Collect returns.
type ScrapeCollectorInterface interface {
	prometheus.Collector
	SubsystemCollector
	CollectWithError(ch chan<- prometheus.Metric) error
}

type ScrapeCollector struct {
	collector          ScrapeCollectorInterface
	Clock              clock.Clock
	scrapeDurationDesc *prometheus.Desc
	scrapeSuccessDesc  *prometheus.Desc
	logger             *logrus.Logger
}

func NewScapreCollector(collector ScrapeCollectorInterface, logger *logrus.Logger) *ScrapeCollector {
	return &ScrapeCollector{
		collector: collector,
		Clock:     &clock.SystemClock{},
		scrapeDurationDesc: prometheus.NewDesc(
			prometheus.BuildFQName(NAMESPACE, "scrape", "duration_seconds"),
			"Duration of a collector scrape.",
			nil,
			prometheus.Labels{"collector": collector.GetSubsystem()},
		),
		scrapeSuccessDesc: prometheus.NewDesc(
			prometheus.BuildFQName(NAMESPACE, "scrape", "success"),
			"Whether a collector succeeded.",
			nil,
			prometheus.Labels{"collector": collector.GetSubsystem()},
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
	} else {
		sc.logger.WithFields(logrus.Fields{
			"subsystem": sc.GetSubsystem(),
			"error":     err,
			"duration":  duration.Seconds(),
		}).Warn("collector scrape failed")
	}
	ch <- prometheus.MustNewConstMetric(sc.scrapeDurationDesc, prometheus.GaugeValue, duration.Seconds())
	ch <- prometheus.MustNewConstMetric(sc.scrapeSuccessDesc, prometheus.GaugeValue, success)
}

func (sc *ScrapeCollector) Describe(ch chan<- *prometheus.Desc) {
	sc.collector.Describe(ch)
	ch <- sc.scrapeDurationDesc
	ch <- sc.scrapeSuccessDesc
}

func (sc *ScrapeCollector) GetSubsystem() string {
	return sc.collector.GetSubsystem()
}

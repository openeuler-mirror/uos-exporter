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


// TODO: implement functions

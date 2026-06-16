package metrics

import (
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// Namespace defines the common namespace to be used by all metrics.
const namespace = "pacemaker"

var (
	scrapeDurationDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_duration_seconds"),
		"pacemaker_exporter: Duration of a collector scrape.",
		[]string{"collector"},
		nil,
	)
	scrapeSuccessDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_success"),
		"pacemaker_exporter: Whether a collector succeeded.",
		[]string{"collector"},
		nil,
	)
)

const (
	defaultEnabled = true
	// defaultDisabled = false
)

var (
	factories      = make(map[string]func() (Collector, error))
	collectorState = make(map[string]*bool)
)

func registerCollector(collector string, isDefaultEnabled bool, factory func() (Collector, error)) {
	var helpDefaultState string
	if isDefaultEnabled {
		helpDefaultState = "enabled"
	} else {
		helpDefaultState = "disabled"
	}

	logrus.Infof("collector.%s", collector)
	logrus.Infof("Enable the %s collector (default: %s).", collector, helpDefaultState)
	// defaultValue := fmt.Sprintf("%v", isDefaultEnabled)

	flag := &isDefaultEnabled
	collectorState[collector] = flag

	factories[collector] = factory
}

// PacemakerCollector implements the prometheus.Collector interface.
type PacemakerCollector struct {
	Collectors map[string]Collector
}

// NewPacemakerCollector creates a new PacemakerCollector
func NewPacemakerCollector(filters ...string) (*PacemakerCollector, error) {
	f := make(map[string]bool)

	for _, filter := range filters {
		enabled, exist := collectorState[filter]
		if !exist {
			return nil, fmt.Errorf("missing collector: %s", filter)
		}

		if !*enabled {
			return nil, fmt.Errorf("disabled collector: %s", filter)
		}

		f[filter] = true
	}

	collectors := make(map[string]Collector)

	for key, enabled := range collectorState {
		if *enabled {
			collector, err := factories[key]()
			if err != nil {
				return nil, err
			}

			if len(f) == 0 || f[key] {
				collectors[key] = collector
			}
		}
	}

	return &PacemakerCollector{Collectors: collectors}, nil
}

// Describe implements the prometheus.Collector interface.
func (n PacemakerCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeDurationDesc
	ch <- scrapeSuccessDesc
}

// Collect implements the prometheus.Collector interface.
func (n PacemakerCollector) Collect(ch chan<- prometheus.Metric) {
	wg := sync.WaitGroup{}
	wg.Add(len(n.Collectors))

	for name, c := range n.Collectors {
		go func(name string, c Collector) {
			execute(name, c, ch)
			wg.Done()
		}(name, c)
	}

	wg.Wait()
}

func execute(name string, c Collector, ch chan<- prometheus.Metric) {
	var success float64

	begin := time.Now()
	err := c.Update(ch)
	duration := time.Since(begin)

	if err != nil {
		logrus.Errorf("ERROR: %s collector failed after %fs: %s", name, duration.Seconds(), err)

		success = 0
	} else {
		logrus.Debugf("OK: %s collector succeeded after %fs.", name, duration.Seconds())
		success = 1
	}
	ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, duration.Seconds(), name)
	ch <- prometheus.MustNewConstMetric(scrapeSuccessDesc, prometheus.GaugeValue, success, name)
}

// Collector is the interface a collector has to implement.
type Collector interface {
	// Get new metrics and expose them via prometheus registry.
	Update(ch chan<- prometheus.Metric) error
}

type typedDesc struct {
	desc      *prometheus.Desc
	valueType prometheus.ValueType
}

func (d *typedDesc) mustNewConstMetric(value float64, labels ...string) prometheus.Metric {
	return prometheus.MustNewConstMetric(d.desc, d.valueType, value, labels...)
}

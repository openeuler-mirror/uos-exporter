package metrics

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/prometheus/client_golang/prometheus"
)

// Exporter collects clickhouse stats from the given URI and exports them using
// the prometheus metrics package.
type AsyncMetricsExporter struct {
}

func NewAsyncMetricsExporter() *AsyncMetricsExporter {
	return &AsyncMetricsExporter{}
}

func (e *AsyncMetricsExporter) Describe(ch chan<- *prometheus.Desc) {

}

func (e *AsyncMetricsExporter) Collect(ch chan<- prometheus.Metric) {
	// log.Println("run here Collect")

	if err := e.collect(ch); err != nil {
		log.Info().Msgf("Error scraping clickhouse: %s", err)
	}

}

func (e *AsyncMetricsExporter) collect(ch chan<- prometheus.Metric) error {
	mu, err := url.Parse(URI)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	q := mu.Query()
	asyncMetricsURI := mu
	q.Set("query", "select replaceRegexpAll(toString(metric), '-', '_') AS metric, value from system.asynchronous_metrics")
	asyncMetricsURI.RawQuery = q.Encode()

	asyncMetrics, err := e.parseKeyValueResponse(asyncMetricsURI.String())
	if err != nil {
		return fmt.Errorf("error scraping clickhouse url %v: %v", asyncMetricsURI, err)
	}

	for _, am := range asyncMetrics {
		newMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      metricName(am.key),
			Help:      "Number of " + am.key + " async processed",
		}, []string{}).WithLabelValues()
		newMetric.Set(am.value)
		newMetric.Collect(ch)
	}

	return nil
}

type lineResult struct {
	key   string
	value float64
}

func (e *AsyncMetricsExporter) parseKeyValueResponse(uri string) ([]lineResult, error) {
	data, err := handleResponse(uri)
	if err != nil {
		return nil, err
	}

	// Parsing results
	lines := strings.Split(string(data), "\n")
	var results = make([]lineResult, 0)

	for i, line := range lines {
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		if len(parts) != 2 {
			return nil, fmt.Errorf("parseKeyValueResponse: unexpected %d line: %s", i, line)
		}
		k := strings.TrimSpace(parts[0])
		v, err := parseNumber(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, err
		}
		results = append(results, lineResult{k, v})

	}
	return results, nil
}

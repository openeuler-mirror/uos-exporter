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
type DiskMetricsExporter struct {
	free_space_in_bytes  *prometheus.Desc
	total_space_in_bytes *prometheus.Desc
}

func NewDiskMetricsExporter() *DiskMetricsExporter {
	free_space_in_bytes := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "free_space_in_bytes"),
		"Disks free_space_in_bytes capacity",
		[]string{"disk"},
		nil,
	)

	total_space_in_bytes := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "total_space_in_bytes"),
		"Disks total_space_in_bytes capacity",
		[]string{"disk"},
		nil,
	)

	return &DiskMetricsExporter{
		free_space_in_bytes:  free_space_in_bytes,
		total_space_in_bytes: total_space_in_bytes,
	}
}

func (e *DiskMetricsExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.free_space_in_bytes
	ch <- e.total_space_in_bytes
}

func (e *DiskMetricsExporter) Collect(ch chan<- prometheus.Metric) {
	// log.Println("run here Collect")

	if err := e.collect(ch); err != nil {
		log.Info().Msgf("Error scraping clickhouse: %s", err)
	}

}

func (e *DiskMetricsExporter) collect(ch chan<- prometheus.Metric) error {
	mu, err := url.Parse(URI)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	q := mu.Query()
	disksMetricURI := mu
	q.Set("query", `select name, sum(free_space) as free_space_in_bytes, sum(total_space) as total_space_in_bytes from system.disks group by name`)
	disksMetricURI.RawQuery = q.Encode()

	disksMetrics, err := e.parseDiskResponse(disksMetricURI.String())
	if err != nil {
		return fmt.Errorf("error scraping clickhouse url %v: %v", disksMetricURI, err)
	}

	for _, dm := range disksMetrics {
		ch <- prometheus.MustNewConstMetric(
			e.free_space_in_bytes,
			prometheus.GaugeValue,
			float64(dm.freeSpace),
			dm.disk,
		)
		ch <- prometheus.MustNewConstMetric(
			e.total_space_in_bytes,
			prometheus.GaugeValue,
			float64(dm.totalSpace),
			dm.disk,
		)
	}
	return nil
}

type diskResult struct {
	disk       string
	freeSpace  float64
	totalSpace float64
}

func (e *DiskMetricsExporter) parseDiskResponse(uri string) ([]diskResult, error) {
	data, err := handleResponse(uri)
	if err != nil {
		return nil, err
	}

	// Parsing results
	lines := strings.Split(string(data), "\n")
	var results = make([]diskResult, 0)

	for i, line := range lines {
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		if len(parts) != 3 {
			return nil, fmt.Errorf("parseDiskResponse: unexpected %d line: %s", i, line)
		}
		disk := strings.TrimSpace(parts[0])

		freeSpace, err := parseNumber(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, err
		}

		totalSpace, err := parseNumber(strings.TrimSpace(parts[2]))
		if err != nil {
			return nil, err
		}

		results = append(results, diskResult{disk, freeSpace, totalSpace})

	}
	return results, nil
}

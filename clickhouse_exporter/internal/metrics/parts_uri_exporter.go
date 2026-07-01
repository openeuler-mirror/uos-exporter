package metrics

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/prometheus/client_golang/prometheus"
)

// Exporter collects clickhouse stats from the given URI and exports them using
// the prometheus metrics package.
type PartsURIExporter struct {
	table_parts_bytes *prometheus.Desc
	table_parts_count *prometheus.Desc
	table_parts_rows  *prometheus.Desc
}

func NewPartsURIExporter() *PartsURIExporter {
	table_parts_bytes := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "table_parts_bytes"),
		"Table size in bytes",
		[]string{"database", "table"},
		nil,
	)

	table_parts_count := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "table_parts_count"),
		"Number of parts of the table",
		[]string{"database", "table"},
		nil,
	)

	table_parts_rows := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "table_parts_rows"),
		"Number of rows in the table",
		[]string{"database", "table"},
		nil,
	)
	return &PartsURIExporter{
		table_parts_bytes: table_parts_bytes,
		table_parts_count: table_parts_count,
		table_parts_rows:  table_parts_rows,
	}
}

func (e *PartsURIExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.table_parts_bytes
	ch <- e.table_parts_count
	ch <- e.table_parts_rows
}

func (e *PartsURIExporter) Collect(ch chan<- prometheus.Metric) {
	// log.Println("run here Collect")

	// upValue := 1

	if err := e.collect(ch); err != nil {
		log.Info().Msgf("Error scraping clickhouse: %s", err)
		// upValue = 0
	}

	// ch <- prometheus.MustNewConstMetric(
	// 	prometheus.NewDesc(
	// 		prometheus.BuildFQName(namespace, "", "up"),
	// 		"Was the last query of ClickHouse successful.",
	// 		nil, nil,
	// 	),
	// 	prometheus.GaugeValue, float64(upValue),
	// )
}

func (e *PartsURIExporter) collect(ch chan<- prometheus.Metric) error {
	mu, err := url.Parse(URI)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	q := mu.Query()
	partsURI := mu
	q.Set("query", "select database, table, sum(bytes) as bytes, count() as parts, sum(rows) as rows from system.parts where active = 1 group by database, table")
	partsURI.RawQuery = q.Encode()

	parts, err := e.parsePartsResponse(partsURI.String())
	if err != nil {
		return fmt.Errorf("error scraping clickhouse url %v: %v", partsURI, err)
	}

	for _, part := range parts {
		ch <- prometheus.MustNewConstMetric(
			e.table_parts_bytes,
			prometheus.GaugeValue,
			float64(part.bytes),
			part.database,
			part.table,
		)
		ch <- prometheus.MustNewConstMetric(
			e.table_parts_count,
			prometheus.GaugeValue,
			float64(part.parts),
			part.database,
			part.table,
		)
		ch <- prometheus.MustNewConstMetric(
			e.table_parts_rows,
			prometheus.GaugeValue,
			float64(part.rows),
			part.database,
			part.table,
		)
	}
	return nil
}

type partsResult struct {
	database string
	table    string
	bytes    int
	parts    int
	rows     int
}

func (e *PartsURIExporter) parsePartsResponse(uri string) ([]partsResult, error) {
	data, err := handleResponse(uri)
	if err != nil {
		return nil, err
	}

	// Parsing results
	lines := strings.Split(string(data), "\n")
	var results = make([]partsResult, 0)

	for i, line := range lines {
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		if len(parts) != 5 {
			return nil, fmt.Errorf("parsePartsResponse: unexpected %d line: %s", i, line)
		}
		database := strings.TrimSpace(parts[0])
		table := strings.TrimSpace(parts[1])

		bytes, err := strconv.Atoi(strings.TrimSpace(parts[2]))
		if err != nil {
			return nil, err
		}

		count, err := strconv.Atoi(strings.TrimSpace(parts[3]))
		if err != nil {
			return nil, err
		}

		rows, err := strconv.Atoi(strings.TrimSpace(parts[4]))
		if err != nil {
			return nil, err
		}

		results = append(results, partsResult{database, table, bytes, count, rows})
	}

	return results, nil
}

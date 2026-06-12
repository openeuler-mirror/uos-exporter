package metrics

import (
	"clickhouse_exporter/internal/exporter"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	exporter.Register(
		NewClickhouseExporter())
}

var (
	namespace = "clickhouse" // For Prometheus metrics.
	URI       = "http://0.0.0.0:8123"
	insecure  = true
	user      = os.Getenv("CLICKHOUSE_USER")
	password  = os.Getenv("CLICKHOUSE_PASSWORD")
	client    = &http.Client{
		// #nosec G402
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
		},
		Timeout: 30 * time.Second,
	}
)

// Exporter collects clickhouse stats from the given URI and exports them using
// the prometheus metrics package.
type ClickhouseExporter struct {
	disk_metrics_exporter  *DiskMetricsExporter
	parts_uri_exporter     *PartsURIExporter
	metrics_uri_exporer    *MetricsURIExporter
	async_metrics_exporter *AsyncMetricsExporter
	events_uri_exporter    *EventsURIExporter
}

type ClickhouseConfig struct {
	Uri string `yaml:"clickhouse_uri"`
}

func LoadClickhouseConfig(path string) (*ClickhouseConfig, error) {

	cleanPath := filepath.Clean(path)
	// 限制文件扩展名
	ext := filepath.Ext(cleanPath)
	if ext != ".yaml" && ext != ".yml" && ext != "" {
		return nil, fmt.Errorf("invalid file extension: only .yaml or .yml files are allowed")
	}
	configDir := "/etc/uos-exporter"
	if !strings.HasPrefix(cleanPath, configDir) {
		return nil, fmt.Errorf("config file must be located within %s", configDir)
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, err
	}

	var config ClickhouseConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func NewClickhouseExporter() *ClickhouseExporter {
	return &ClickhouseExporter{
		disk_metrics_exporter:  NewDiskMetricsExporter(),
		parts_uri_exporter:     NewPartsURIExporter(),
		metrics_uri_exporer:    NewMetricsURIExporter(),
		async_metrics_exporter: NewAsyncMetricsExporter(),
		events_uri_exporter:    NewEventsURIExporter(),
	}
}

func (e *ClickhouseExporter) Describe(ch chan<- *prometheus.Desc) {
	e.disk_metrics_exporter.Describe(ch)
	e.parts_uri_exporter.Describe(ch)
	e.metrics_uri_exporer.Describe(ch)
	e.async_metrics_exporter.Describe(ch)
	e.events_uri_exporter.Describe(ch)

}

func (e *ClickhouseExporter) Collect(ch chan<- prometheus.Metric) {
	// log.Println("run here Collect")
	config, err := LoadClickhouseConfig("/etc/uos-exporter/clickhouse-exporter.yaml")
	if err != nil {
		fmt.Printf("Error get clickhouse URI %v\n", err)
	} else {
		URI = config.Uri
	}

	e.disk_metrics_exporter.Collect(ch)
	e.parts_uri_exporter.Collect(ch)
	e.metrics_uri_exporer.Collect(ch)
	e.async_metrics_exporter.Collect(ch)
	e.events_uri_exporter.Collect(ch)
}

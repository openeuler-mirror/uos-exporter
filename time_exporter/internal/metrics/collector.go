package metrics

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"time_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

const namespace = "time"

var (
	scrapeDuration = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_duration_seconds"),
		"Duration of the scrape",
		[]string{"collector"},
		nil,
	)
	scrapeSuccess = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_success"),
		"Success of the scrape",
		[]string{"collector"},
		nil,
	)
)

var (
	factories      = make(map[string]func() (Collector, error))
	collectorState = make(map[string]*bool)
)

type Collector interface {
	Update(ch chan<- prometheus.Metric) error
}

type TimeConfig struct {
	ChronyConfig ChronyConfig `yaml:"chrony_config"`
	NTPConfig    NTPConfig    `yaml:"ntp_config"`
}

func LoadConfig(filePath string) *TimeConfig {
	if strings.TrimSpace(filePath) == "" {
		log.Fatalf("empty yaml config path")
	}

	cleanFilePath := filepath.Clean(filePath)

	if !strings.HasPrefix(cleanFilePath, "/etc/uos-exporter/") {
		log.Fatalf("yaml config path must be under /etc/uos-exporter/")
	}

	yamlContent, err := os.ReadFile(cleanFilePath)
	if err != nil {
		log.Fatalf("can't read yaml config: %v", err)
	}

	var configFile TimeConfig
	err = yaml.Unmarshal(yamlContent, &configFile)
	if err != nil {
		log.Fatalf("can't parse yaml config: %v", err)
	}

	return &configFile
}

func init() {
	cfg := LoadConfig("/etc/uos-exporter/time-exporter.yaml")
	registerCollector("chrony", true, func() (Collector, error) {
		return NewChronyCollector(&cfg.ChronyConfig), nil
	})
	registerCollector("time", true, func() (Collector, error) {
		return NewTimeCollector(), nil
	})
	registerCollector("timex", true, func() (Collector, error) {
		return NewTimeXCollector(), nil
	})
	registerCollector("ntp", true, func() (Collector, error) {
		return NewNTPCollector(&cfg.NTPConfig), nil
	})

	exporter.Register(func() prometheus.Collector {
		nc, _ := NewNodeCollector()
		return nc
	}())
}

type NodeCollector struct {
	Collectors map[string]Collector
}

func registerCollector(collector string, isDefaultEnabled bool, factory func() (Collector, error)) {
	enabled := isDefaultEnabled
	collectorState[collector] = &enabled
	factories[collector] = factory
}

func NewNodeCollector(filters ...string) (*NodeCollector, error) {
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

	return &NodeCollector{
		Collectors: collectors,
	}, nil
}

func (c *NodeCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeDuration
	ch <- scrapeSuccess
}

func (c *NodeCollector) Collect(ch chan<- prometheus.Metric) {
	wg := sync.WaitGroup{}
	wg.Add(len(c.Collectors))
	for name, collector := range c.Collectors {
		go func(name string, collector Collector) {
			defer wg.Done()
			execute(name, collector, ch)
		}(name, collector)
	}
	wg.Wait()
}

func execute(name string, collector Collector, ch chan<- prometheus.Metric) {
	start := time.Now()
	err := collector.Update(ch)
	duration := time.Since(start).Seconds()

	success := 0.0
	if err == nil {
		success = 1.0
	}
	ch <- prometheus.MustNewConstMetric(scrapeDuration, prometheus.GaugeValue, duration, name)
	ch <- prometheus.MustNewConstMetric(scrapeSuccess, prometheus.GaugeValue, success, name)

	logFields := logrus.Fields{
		"collector": name,
		"duration":  duration,
	}

	if err != nil {
		logFields["err"] = err.Error()
		if IsNoDataError(err) {
			logrus.WithFields(logFields).Debug("Collector returned no data")
		} else {
			logrus.WithFields(logFields).Error("Collector failed")
		}
	}
}

type MetricDesc struct {
	Desc        *prometheus.Desc
	Type        prometheus.ValueType
	Help        string
	Labels      []string
	ConstLabels prometheus.Labels
}

var ErrNoData = errors.New("collector returned no data")

func IsNoDataError(err error) bool {
	return err == ErrNoData
}
// Part 2 commit for time_exporter/internal/metrics/collector.go

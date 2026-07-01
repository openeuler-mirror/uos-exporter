package metrics

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fluentd_exporter/config"
	"fluentd_exporter/internal/exporter"
	"fmt"
	"github.com/alecthomas/kingpin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
)

var (
	scrapeURI *string
)

func init() {
	scrapeURI = kingpin.Flag("fluentd_scrape_uri", "Scrape URI").
		Default("http://localhost:24220/api/plugins.json").
		String()
}

func init() {
	exporter.Register(
		NewFluentd("fluentd_info", "fluentd  info", []string{"plugin_id", "plugin_category", "type"}))
}

type Fluentd struct {
	*baseMetrics
}

func NewFluentd(fqname, help string, labels []string) *Fluentd {
	return &Fluentd{NewMetrics(fqname, help, labels)}
}

func (fr *Fluentd) Collect(ch chan<- prometheus.Metric) {
	pluginsInfo, err := getPluginInfo(*scrapeURI)
	if err != nil {
		logrus.Error(err)
		return
	}
	for _, plugin := range pluginsInfo {
		fr.baseMetrics.collect(ch, float64(plugin.RetryCount), []string{plugin.PluginID, plugin.PluginCategory, "retry_count"})
		fr.baseMetrics.collect(ch, float64(plugin.BufferQueueLength), []string{plugin.PluginID, plugin.PluginCategory, "buffer_queue_length"})
		fr.baseMetrics.collect(ch, float64(plugin.BufferTotalQueuedSize), []string{plugin.PluginID, plugin.PluginCategory, "buffer_total_queued_size"})
	}

}

type Plugin struct {
	PluginID              string `json:"plugin_id"`
	PluginCategory        string `json:"plugin_category"`
	Type                  string `json:"type"`
	RetryCount            int    `json:"retry_count"`
	BufferQueueLength     int    `json:"buffer_queue_length,omitempty"`
	BufferTotalQueuedSize int    `json:"buffer_total_queued_size,omitempty"`
}

type jsonData struct {
	Plugins []Plugin `json:"plugins"`
}

func getPluginInfo(url string) ([]Plugin, error) {
	if url == "" {
		return nil, errors.New("scrapeURI is not set")
	}
	plugins := []Plugin{}
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				// #nosec G402
				InsecureSkipVerify: *config.Insecure,
			},
		},
	}
	resp, err := client.Get(url)
	if err != nil {
		return plugins, fmt.Errorf("failed to fetch data: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return plugins, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	jdata := jsonData{}
	err = json.Unmarshal(body, &jdata)
	if err != nil {
		return plugins, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	for i := range jdata.Plugins {
		if jdata.Plugins[i].PluginCategory == "input" {
			continue
		}
		plugins = append(plugins, jdata.Plugins[i])
	}
	return plugins, nil
}

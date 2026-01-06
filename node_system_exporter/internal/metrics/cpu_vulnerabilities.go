package metrics

import (
	"log/slog"

	"node_system_exporter/internal/exporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs/sysfs"
)

func init() {
	exporter.Register(NewCPUVulnerabilitiesCollector())
}

type CPUVulnerabilitiesCollector struct {
	*baseMetrics
	desc   *prometheus.Desc
	logger *slog.Logger
}

func NewCPUVulnerabilitiesCollector() *CPUVulnerabilitiesCollector {
	logger := slog.Default()

	desc := prometheus.NewDesc(
		prometheus.BuildFQName("node", "cpu_vulnerabilities", "info"),
		"Details of each CPU vulnerability reported by sysfs. The value of the series is an int encoded state of the vulnerability. The same state is stored as a string in the label",
		[]string{"codename", "state", "mitigation"},
		nil,
	)

	return &CPUVulnerabilitiesCollector{
		desc:   desc,
		logger: logger,
	}
}

func (c *CPUVulnerabilitiesCollector) Collect(ch chan<- prometheus.Metric) {
	fs, err := sysfs.NewFS("/sys")
	if err != nil {
		c.logger.Debug("failed to open sysfs", "error", err)
		return
	}

	vulnerabilities, err := fs.CPUVulnerabilities()
	if err != nil {
		c.logger.Debug("failed to get vulnerabilities", "error", err)
		return
	}

	for _, vulnerability := range vulnerabilities {
		ch <- prometheus.MustNewConstMetric(
			c.desc,
			prometheus.GaugeValue,
			1.0,
			vulnerability.CodeName,
			sysfs.VulnerabilityHumanEncoding[vulnerability.State],
			vulnerability.Mitigation,
		)
	}
}

func (c *CPUVulnerabilitiesCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
} 
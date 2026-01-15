package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"mysqld_exporter/internal/exporter"
	"mysqld_exporter/version"
)

func init() {
	exporter.Register(
		NewBuildInfo("exporter_build_info",
			"exporter build info",
			[]string{"version",
				"revision",
				"branch",
				"goversion"}))
}

type BuildInfo struct {
	*baseMetrics
}

func NewBuildInfo(fqname, help string, labels []string) *BuildInfo {
	return &BuildInfo{NewMetrics(fqname, help, labels)}
}

func (c *BuildInfo) Collect(ch chan<- prometheus.Metric) {
	c.baseMetrics.collect(ch,
		1,
		[]string{version.Version,
			version.Revision,
			version.Branch,
			version.GoVersion})
}

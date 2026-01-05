package version

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Build information. Populated at build-time.
var (
	Version   string
	Revision  string
	Branch    string
	BuildUser string
	BuildDate string
	GoVersion string
)

// NewCollector returns a collector that exports metrics about current version information.
func NewCollector(program string) prometheus.Collector {
	return prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: program,
			Name:      "build_info",
			Help: "A metric with a constant '1' value labeled by version, revision, branch, " +
				"build user and build date.",
			ConstLabels: prometheus.Labels{
				"version":   Version,
				"revision":  Revision,
				"branch":    Branch,
				"builduser": BuildUser,
				"builddate": BuildDate,
				"goversion": GoVersion,
			},
		},
		func() float64 { return 1 },
	)
} 
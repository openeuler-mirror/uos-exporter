package metrics

import (
	"log/slog"
	"strconv"

	"node_system_exporter/internal/exporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
)

func init() {
	exporter.Register(NewBuddyInfoCollector())
}

type BuddyInfoCollector struct {
	*baseMetrics
	fs     procfs.FS
	desc   *prometheus.Desc
	logger *slog.Logger
}

func NewBuddyInfoCollector() *BuddyInfoCollector {
	logger := slog.Default()
	
	fs, err := procfs.NewFS("/proc")
	if err != nil {
		logger.Error("failed to open procfs", "error", err)
		return nil
	}

	desc := prometheus.NewDesc(
		prometheus.BuildFQName("node", "buddyinfo", "blocks"),
		"Count of free blocks according to size.",
		[]string{"node", "zone", "size"}, nil,
	)

	return &BuddyInfoCollector{
		fs:     fs,
		desc:   desc,
		logger: logger,
	}
}

func (c *BuddyInfoCollector) Collect(ch chan<- prometheus.Metric) {
	buddyInfo, err := c.fs.BuddyInfo()
	if err != nil {
		c.logger.Debug("Error getting buddy info", "error", err)
		return
	}

	for _, entry := range buddyInfo {
		for size, value := range entry.Sizes {
			ch <- prometheus.MustNewConstMetric(
				c.desc,
				prometheus.GaugeValue, 
				value,
				entry.Node, 
				entry.Zone, 
				strconv.Itoa(size),
			)
		}
	}
}

func (c *BuddyInfoCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
} 
package metrics

import (
	"errors"
	"node_hardware_exporter/internal/exporter"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs/sysfs"
)

func init() {
	exporter.Register(NewNVMeCollector())
}

type NVMeCollector struct {
	*baseMetrics
	fs sysfs.FS
}

func NewNVMeCollector() *NVMeCollector {
	fs, err := sysfs.NewFS("/sys")
	if err != nil {
		return &NVMeCollector{
			baseMetrics: NewMetrics("node_nvme_collector", "NVMe collector metrics", []string{}),
		}
	}

	return &NVMeCollector{
		baseMetrics: NewMetrics("node_nvme_collector", "NVMe collector metrics", []string{}),
		fs:          fs,
	}
}

func (c *NVMeCollector) Collect(ch chan<- prometheus.Metric) {
	if c.fs == (sysfs.FS{}) {
		fs, err := sysfs.NewFS("/sys")
		if err != nil {
			return
		}
		c.fs = fs
	}

	devices, err := c.fs.NVMeClass()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	for _, device := range devices {
		infoDesc := prometheus.NewDesc(
			"node_nvme_info",
			"Non-numeric data from /sys/class/nvme/<device>, value is always 1.",
			[]string{"device", "firmware_revision", "model", "serial", "state"},
			nil,
		)
		infoValue := 1.0
		ch <- prometheus.MustNewConstMetric(infoDesc, prometheus.GaugeValue, infoValue, device.Name, device.FirmwareRevision, device.Model, device.Serial, device.State)
	}
} 
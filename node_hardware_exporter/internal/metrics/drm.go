package metrics

import (
	"node_hardware_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs/sysfs"
)

func init() {
	exporter.Register(NewDrmCollector())
}

type DrmCollector struct {
	*baseMetrics
	fs                    sysfs.FS
	CardInfo              *prometheus.Desc
	GPUBusyPercent        *prometheus.Desc
	MemoryGTTSize         *prometheus.Desc
	MemoryGTTUsed         *prometheus.Desc
	MemoryVisibleVRAMSize *prometheus.Desc
	MemoryVisibleVRAMUsed *prometheus.Desc
	MemoryVRAMSize        *prometheus.Desc
	MemoryVRAMUsed        *prometheus.Desc
}

func NewDrmCollector() *DrmCollector {
	fs, err := sysfs.NewFS("/sys")
	if err != nil {
		return &DrmCollector{
			baseMetrics: NewMetrics("node_drm_collector", "DRM collector metrics", []string{}),
		}
	}

	return &DrmCollector{
		fs:          fs,
		baseMetrics: NewMetrics("node_drm_collector", "DRM collector metrics", []string{}),
		CardInfo: prometheus.NewDesc(
			"node_drm_card_info",
			"Card information",
			[]string{"card", "memory_vendor", "power_performance_level", "unique_id", "vendor"}, nil,
		),
		GPUBusyPercent: prometheus.NewDesc(
			"node_drm_gpu_busy_percent",
			"How busy the GPU is as a percentage.",
			[]string{"card"}, nil,
		),
		MemoryGTTSize: prometheus.NewDesc(
			"node_drm_memory_gtt_size_bytes",
			"The size of the graphics translation table (GTT) block in bytes.",
			[]string{"card"}, nil,
		),
		MemoryGTTUsed: prometheus.NewDesc(
			"node_drm_memory_gtt_used_bytes",
			"The used amount of the graphics translation table (GTT) block in bytes.",
			[]string{"card"}, nil,
		),
		MemoryVisibleVRAMSize: prometheus.NewDesc(
			"node_drm_memory_vis_vram_size_bytes",
			"The size of visible VRAM in bytes.",
			[]string{"card"}, nil,
		),
		MemoryVisibleVRAMUsed: prometheus.NewDesc(
			"node_drm_memory_vis_vram_used_bytes",
			"The used amount of visible VRAM in bytes.",
			[]string{"card"}, nil,
		),
		MemoryVRAMSize: prometheus.NewDesc(
			"node_drm_memory_vram_size_bytes",
			"The size of VRAM in bytes.",
			[]string{"card"}, nil,
		),
		MemoryVRAMUsed: prometheus.NewDesc(
			"node_drm_memory_vram_used_bytes",
			"The used amount of VRAM in bytes.",
			[]string{"card"}, nil,
		),
	}
}

func (c *DrmCollector) Collect(ch chan<- prometheus.Metric) {
	if c.fs == (sysfs.FS{}) {
		fs, err := sysfs.NewFS("/sys")
		if err != nil {
			return
		}
		c.fs = fs
	}

	// 获取AMD卡信息
	c.updateAMDCards(ch)
}

func (c *DrmCollector) updateAMDCards(ch chan<- prometheus.Metric) {
	vendor := "amd"
	stats, err := c.fs.ClassDRMCardAMDGPUStats()
	if err != nil {
		return
	}

	for _, s := range stats {
		ch <- prometheus.MustNewConstMetric(
			c.CardInfo, prometheus.GaugeValue, 1,
			s.Name, s.MemoryVRAMVendor, s.PowerDPMForcePerformanceLevel, s.UniqueID, vendor)

		ch <- prometheus.MustNewConstMetric(
			c.GPUBusyPercent, prometheus.GaugeValue, float64(s.GPUBusyPercent), s.Name)

		ch <- prometheus.MustNewConstMetric(
			c.MemoryGTTSize, prometheus.GaugeValue, float64(s.MemoryGTTSize), s.Name)

		ch <- prometheus.MustNewConstMetric(
			c.MemoryGTTUsed, prometheus.GaugeValue, float64(s.MemoryGTTUsed), s.Name)

		ch <- prometheus.MustNewConstMetric(
			c.MemoryVRAMSize, prometheus.GaugeValue, float64(s.MemoryVRAMSize), s.Name)

		ch <- prometheus.MustNewConstMetric(
			c.MemoryVRAMUsed, prometheus.GaugeValue, float64(s.MemoryVRAMUsed), s.Name)

		ch <- prometheus.MustNewConstMetric(
			c.MemoryVisibleVRAMSize, prometheus.GaugeValue, float64(s.MemoryVisibleVRAMSize), s.Name)

		ch <- prometheus.MustNewConstMetric(
			c.MemoryVisibleVRAMUsed, prometheus.GaugeValue, float64(s.MemoryVisibleVRAMUsed), s.Name)
	}
} 
package metrics

import (
	"errors"
	"node_hardware_exporter/internal/exporter"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs/sysfs"
)

const coolingDevice = "cooling_device"
const thermalZone = "thermal_zone"

func init() {
	exporter.Register(NewThermalZoneCollector())
}

type ThermalZoneCollector struct {
	*baseMetrics
	fs                    sysfs.FS
	coolingDeviceCurState *prometheus.Desc
	coolingDeviceMaxState *prometheus.Desc
	zoneTemp              *prometheus.Desc
}

func NewThermalZoneCollector() *ThermalZoneCollector {
	fs, err := sysfs.NewFS("/sys")
	if err != nil {
		// 如果无法访问sysfs，我们仍然创建收集器，但在收集时处理错误
		return &ThermalZoneCollector{
			baseMetrics: NewMetrics("node_thermal_zone_collector", "Thermal zone collector metrics", []string{}),
		}
	}

	return &ThermalZoneCollector{
		baseMetrics: NewMetrics("node_thermal_zone_collector", "Thermal zone collector metrics", []string{}),
		fs:          fs,
		zoneTemp: prometheus.NewDesc(
			"node_thermal_zone_temp",
			"Zone temperature in Celsius",
			[]string{"zone", "type"}, nil,
		),
		coolingDeviceCurState: prometheus.NewDesc(
			"node_cooling_device_cur_state",
			"Current throttle state of the cooling device",
			[]string{"name", "type"}, nil,
		),
		coolingDeviceMaxState: prometheus.NewDesc(
			"node_cooling_device_max_state",
			"Maximum throttle state of the cooling device",
			[]string{"name", "type"}, nil,
		),
	}
}

func (c *ThermalZoneCollector) Collect(ch chan<- prometheus.Metric) {
	if c.fs == (sysfs.FS{}) {
		fs, err := sysfs.NewFS("/sys")
		if err != nil {
			return
		}
		c.fs = fs
		
		// 初始化metrics描述符
		c.zoneTemp = prometheus.NewDesc(
			"node_thermal_zone_temp",
			"Zone temperature in Celsius",
			[]string{"zone", "type"}, nil,
		)
		c.coolingDeviceCurState = prometheus.NewDesc(
			"node_cooling_device_cur_state",
			"Current throttle state of the cooling device",
			[]string{"name", "type"}, nil,
		)
		c.coolingDeviceMaxState = prometheus.NewDesc(
			"node_cooling_device_max_state",
			"Maximum throttle state of the cooling device",
			[]string{"name", "type"}, nil,
		)
	}

	thermalZones, err := c.fs.ClassThermalZoneStats()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission) || errors.Is(err, os.ErrInvalid) {
			return
		}
		return
	}

	for _, stats := range thermalZones {
		ch <- prometheus.MustNewConstMetric(
			c.zoneTemp,
			prometheus.GaugeValue,
			float64(stats.Temp)/1000.0,
			stats.Name,
			stats.Type,
		)
	}

	coolingDevices, err := c.fs.ClassCoolingDeviceStats()
	if err != nil {
		return
	}

	for _, stats := range coolingDevices {
		ch <- prometheus.MustNewConstMetric(
			c.coolingDeviceCurState,
			prometheus.GaugeValue,
			float64(stats.CurState),
			stats.Name,
			stats.Type,
		)

		ch <- prometheus.MustNewConstMetric(
			c.coolingDeviceMaxState,
			prometheus.GaugeValue,
			float64(stats.MaxState),
			stats.Name,
			stats.Type,
		)
	}
}
// Part 2 commit for node_hardware_exporter/internal/metrics/thermal_zone.go

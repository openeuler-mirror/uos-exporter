package metrics

import (
	"errors"
	"node_hardware_exporter/internal/exporter"
	"os"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs/sysfs"
)

func init() {
	exporter.Register(NewDMICollector())
}

type DMICollector struct {
	*baseMetrics
	infoDesc *prometheus.Desc
	values   []string
	labels   []string
}

func NewDMICollector() *DMICollector {
	fs, err := sysfs.NewFS("/sys")
	if err != nil {
		return &DMICollector{
			baseMetrics: NewMetrics("node_dmi_collector", "DMI collector metrics", []string{}),
		}
	}

	dmi, err := fs.DMIClass()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &DMICollector{
				baseMetrics: NewMetrics("node_dmi_collector", "DMI collector metrics", []string{}),
			}
		}
		return &DMICollector{
			baseMetrics: NewMetrics("node_dmi_collector", "DMI collector metrics", []string{}),
		}
	}

	var labels, values []string
	for label, value := range map[string]*string{
		"bios_date":         dmi.BiosDate,
		"bios_release":      dmi.BiosRelease,
		"bios_vendor":       dmi.BiosVendor,
		"bios_version":      dmi.BiosVersion,
		"board_asset_tag":   dmi.BoardAssetTag,
		"board_name":        dmi.BoardName,
		"board_serial":      dmi.BoardSerial,
		"board_vendor":      dmi.BoardVendor,
		"board_version":     dmi.BoardVersion,
		"chassis_asset_tag": dmi.ChassisAssetTag,
		"chassis_serial":    dmi.ChassisSerial,
		"chassis_vendor":    dmi.ChassisVendor,
		"chassis_version":   dmi.ChassisVersion,
		"product_family":    dmi.ProductFamily,
		"product_name":      dmi.ProductName,
		"product_serial":    dmi.ProductSerial,
		"product_sku":       dmi.ProductSKU,
		"product_uuid":      dmi.ProductUUID,
		"product_version":   dmi.ProductVersion,
		"system_vendor":     dmi.SystemVendor,
	} {
		if value != nil {
			labels = append(labels, label)
			values = append(values, strings.ToValidUTF8(*value, "�"))
		}
	}

	return &DMICollector{
		baseMetrics: NewMetrics("node_dmi_collector", "DMI collector metrics", []string{}),
		infoDesc: prometheus.NewDesc(
			"node_dmi_info",
			"A metric with a constant '1' value labeled by bios_date, bios_release, bios_vendor, bios_version, "+
				"board_asset_tag, board_name, board_serial, board_vendor, board_version, chassis_asset_tag, "+
				"chassis_serial, chassis_vendor, chassis_version, product_family, product_name, product_serial, "+
				"product_sku, product_uuid, product_version, system_vendor if provided by DMI.",
			labels, nil,
		),
		values: values,
		labels: labels,
	}
}

func (c *DMICollector) Collect(ch chan<- prometheus.Metric) {
	if len(c.values) == 0 || c.infoDesc == nil {
		// 如果没有初始化过DMI信息，尝试重新获取
		collector := NewDMICollector()
		if collector.infoDesc != nil && len(collector.values) > 0 {
			ch <- prometheus.MustNewConstMetric(collector.infoDesc, prometheus.GaugeValue, 1.0, collector.values...)
		}
		return
	}
	
	ch <- prometheus.MustNewConstMetric(c.infoDesc, prometheus.GaugeValue, 1.0, c.values...)
} 
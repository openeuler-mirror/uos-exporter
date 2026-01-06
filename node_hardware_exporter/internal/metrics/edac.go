package metrics

import (
	"fmt"
	"node_hardware_exporter/internal/exporter"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	edacMemControllerRE = regexp.MustCompile(`.*devices/system/edac/mc/mc([0-9]*)`)
	edacMemCsrowRE      = regexp.MustCompile(`.*devices/system/edac/mc/mc[0-9]*/csrow([0-9]*)`)
)

func init() {
	exporter.Register(NewEdacCollector())
}

type EdacCollector struct {
	*baseMetrics
	ceCount      *prometheus.Desc
	ueCount      *prometheus.Desc
	csRowCECount *prometheus.Desc
	csRowUECount *prometheus.Desc
}

func NewEdacCollector() *EdacCollector {
	return &EdacCollector{
		baseMetrics: NewMetrics("node_edac_collector", "EDAC collector metrics", []string{}),
		ceCount: prometheus.NewDesc(
			"node_edac_correctable_errors_total",
			"Total correctable memory errors.",
			[]string{"controller"}, nil,
		),
		ueCount: prometheus.NewDesc(
			"node_edac_uncorrectable_errors_total",
			"Total uncorrectable memory errors.",
			[]string{"controller"}, nil,
		),
		csRowCECount: prometheus.NewDesc(
			"node_edac_csrow_correctable_errors_total",
			"Total correctable memory errors for this csrow.",
			[]string{"controller", "csrow"}, nil,
		),
		csRowUECount: prometheus.NewDesc(
			"node_edac_csrow_uncorrectable_errors_total",
			"Total uncorrectable memory errors for this csrow.",
			[]string{"controller", "csrow"}, nil,
		),
	}
}

func (c *EdacCollector) Collect(ch chan<- prometheus.Metric) {
	memControllers, err := filepath.Glob(sysFilePath("devices/system/edac/mc/mc[0-9]*"))
	if err != nil {
		return
	}

	for _, controller := range memControllers {
		controllerMatch := edacMemControllerRE.FindStringSubmatch(controller)
		if controllerMatch == nil {
			return
		}
		controllerNumber := controllerMatch[1]

		value, err := readUintFromFile(filepath.Join(controller, "ce_count"))
		if err != nil {
			return
		}
		ch <- prometheus.MustNewConstMetric(
			c.ceCount, prometheus.CounterValue, float64(value), controllerNumber)

		value, err = readUintFromFile(filepath.Join(controller, "ce_noinfo_count"))
		if err != nil {
			return
		}
		ch <- prometheus.MustNewConstMetric(
			c.csRowCECount, prometheus.CounterValue, float64(value), controllerNumber, "unknown")

		value, err = readUintFromFile(filepath.Join(controller, "ue_count"))
		if err != nil {
			return
		}
		ch <- prometheus.MustNewConstMetric(
			c.ueCount, prometheus.CounterValue, float64(value), controllerNumber)

		value, err = readUintFromFile(filepath.Join(controller, "ue_noinfo_count"))
		if err != nil {
			return
		}
		ch <- prometheus.MustNewConstMetric(
			c.csRowUECount, prometheus.CounterValue, float64(value), controllerNumber, "unknown")

		// For each controller, walk the csrow directories.
		csrows, err := filepath.Glob(controller + "/csrow[0-9]*")
		if err != nil {
			return
		}

		for _, csrow := range csrows {
			csrowMatch := edacMemCsrowRE.FindStringSubmatch(csrow)
			if csrowMatch == nil {
				return
			}
			csrowNumber := csrowMatch[1]

			value, err = readUintFromFile(filepath.Join(csrow, "ce_count"))
			if err != nil {
				return
			}
			ch <- prometheus.MustNewConstMetric(
				c.csRowCECount, prometheus.CounterValue, float64(value), controllerNumber, csrowNumber)

			value, err = readUintFromFile(filepath.Join(csrow, "ue_count"))
			if err != nil {
				return
			}
			ch <- prometheus.MustNewConstMetric(
				c.csRowUECount, prometheus.CounterValue, float64(value), controllerNumber, csrowNumber)
		}
	}
}

// 从文件中读取无符号整数值
func readUintFromFile(path string) (uint64, error) {
	cleanPath := filepath.Clean(path)
	if !strings.HasPrefix(cleanPath, "/sys/devices/") {
		return 0, fmt.Errorf("unallowed path:%s", cleanPath)
	}
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return 0, err
	}

	value, err := strconv.ParseUint(string(data[:len(data)-1]), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse uint from %s: %w", path, err)
	}

	return value, nil
}

// 使用hwmon.go中已定义的sysFilePath函数
// func sysFilePath(name string) string {
// 	return filepath.Join("/sys", name)
// }

package metrics

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"smartctl_exporter/internal/exporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tidwall/gjson"
)

func init() {
	exporter.Register(NewSmartctlManagerCollector())
}

// SmartctlManagerCollector implements the main collector interface
type SmartctlManagerCollector struct {
	*baseMetrics
	devices []Device
	mutex   sync.Mutex
}

func NewSmartctlManagerCollector() *SmartctlManagerCollector {
	return &SmartctlManagerCollector{
		NewMetrics("smartctl_manager", "SMARTCTL Manager Collector", []string{}),
		scanDevices(),
		sync.Mutex{},
	}
}

func (c *SmartctlManagerCollector) Collect(ch chan<- prometheus.Metric) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Refresh device data
	refreshAllDevices(logger, c.devices)

	// Collect smartctl version info (only once)
	versionCollected := false

	// Collect device count
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc("smartctl_devices", "Number of devices configured or dynamically discovered", []string{}, nil),
		prometheus.GaugeValue,
		float64(len(c.devices)),
	)

	// Collect metrics for each device
	for _, device := range c.devices {
		json := readData(logger, device)
		if json.Exists() {
			// Collect version info only once
			if !versionCollected {
				c.collectVersion(ch, json)
				versionCollected = true
			}

			// Create SMARTctl instance and collect device-specific metrics
			smart := NewSMARTctl(logger, json, ch)
			smart.Collect()
		}
	}
}

func (c *SmartctlManagerCollector) collectVersion(ch chan<- prometheus.Metric, json gjson.Result) {
	smartctlJSON := json.Get("smartctl")
	smartctlVersion := smartctlJSON.Get("version").Array()
	jsonVersion := json.Get("json_format_version").Array()
	
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc("smartctl_version", "smartctl version", []string{
			"json_format_version",
			"smartctl_version",
			"svn_revision",
			"build_info",
		}, nil),
		prometheus.GaugeValue,
		1,
		fmt.Sprintf("%d.%d", jsonVersion[0].Int(), jsonVersion[1].Int()),
		fmt.Sprintf("%d.%d", smartctlVersion[0].Int(), smartctlVersion[1].Int()),
		smartctlJSON.Get("svn_revision").String(),
		smartctlJSON.Get("build_info").String(),
	)
}

// SMARTctl object for collecting device metrics
type SMARTctl struct {
	ch     chan<- prometheus.Metric
	json   gjson.Result
	logger *slog.Logger
	device SMARTDevice
}

// NewSMARTctl creates a new SMARTctl instance
func NewSMARTctl(logger *slog.Logger, json gjson.Result, ch chan<- prometheus.Metric) *SMARTctl {
	var model_name string
	if obj := json.Get("model_name"); obj.Exists() {
		model_name = obj.String()
	} else if obj := json.Get("scsi_model_name"); obj.Exists() {
		model_name = obj.String()
	}
	// If the drive returns an empty model name, replace that with unknown.
	if model_name == "" {
		model_name = "unknown"
	}

	return &SMARTctl{
		ch:     ch,
		json:   json,
		logger: logger,
		device: SMARTDevice{
			device:     buildDeviceLabel(json.Get("device.name").String(), json.Get("device.type").String()),
			serial:     strings.TrimSpace(json.Get("serial_number").String()),
			family:     strings.TrimSpace(GetStringIfExists(json, "model_family", "unknown")),
			model:      strings.TrimSpace(model_name),
			interface_: strings.TrimSpace(json.Get("device.type").String()),
			protocol:   strings.TrimSpace(json.Get("device.protocol").String()),
		},
	}
}

// Collect metrics for a specific device
func (smart *SMARTctl) Collect() {
	smart.logger.Debug("Collecting metrics from", "device", smart.device.device, "family", smart.device.family, "model", smart.device.model)
	smart.mineExitStatus()
	smart.mineDevice()
	smart.mineCapacity()
	smart.mineBlockSize()
	smart.mineInterfaceSpeed()
	smart.mineDeviceAttribute()
	smart.minePowerOnSeconds()
	smart.mineRotationRate()
	smart.mineTemperatures()
	smart.minePowerCycleCount()
	smart.mineDeviceSCTStatus()
	smart.mineDeviceStatistics()
	smart.mineDeviceErrorLog()
	smart.mineDeviceSelfTestLog()
	smart.mineDeviceERC()
	smart.mineSmartStatus()

	if smart.device.interface_ == "nvme" {
		smart.mineNvmePercentageUsed()
		smart.mineNvmeAvailableSpare()
		smart.mineNvmeAvailableSpareThreshold()
		smart.mineNvmeCriticalWarning()
		smart.mineNvmeMediaErrors()
		smart.mineNvmeNumErrLogEntries()
		smart.mineNvmeBytesRead()
		smart.mineNvmeBytesWritten()
	}
	// SCSI, SAS
	if smart.device.interface_ == "scsi" {
		smart.mineSCSIGrownDefectList()
		smart.mineSCSIErrorCounterLog()
		smart.mineSCSIBytesRead()
		smart.mineSCSIBytesWritten()
	}
}

func (smart *SMARTctl) mineExitStatus() {
	smart.ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc("smartctl_device_smartctl_exit_status", "Exit status of smartctl on device", []string{"device"}, nil),
		prometheus.GaugeValue,
		smart.json.Get("smartctl.exit_status").Float(),
		smart.device.device,
	)
}

func (smart *SMARTctl) mineDevice() {
	smart.ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc("smartctl_device", "Device info", []string{
			"device",
			"interface",
			"protocol",
			"model_family",
			"model_name",
			"serial_number",
			"ata_additional_product_id",
			"firmware_version",
			"ata_version",
			"sata_version",
			"form_factor",
			"scsi_vendor",
			"scsi_product",
			"scsi_revision",
			"scsi_version",
		}, nil),
		prometheus.GaugeValue,
		1,
		smart.device.device,
		smart.device.interface_,
		smart.device.protocol,
		smart.device.family,
		smart.device.model,
		smart.device.serial,
		GetStringIfExists(smart.json, "ata_additional_product_id", "unknown"),
		smart.json.Get("firmware_version").String(),
		smart.json.Get("ata_version.string").String(),
		smart.json.Get("sata_version.string").String(),
		smart.json.Get("form_factor.name").String(),
		smart.json.Get("scsi_vendor").String(),
		smart.json.Get("scsi_product").String(),
		smart.json.Get("scsi_revision").String(),
		smart.json.Get("scsi_version").String(),
	)
}

func (smart *SMARTctl) mineCapacity() {
	smart.ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc("smartctl_device_capacity_blocks", "Device capacity in blocks", []string{"device"}, nil),
		prometheus.GaugeValue,
		smart.json.Get("user_capacity.blocks").Float(),
		smart.device.device,
	)
	smart.ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc("smartctl_device_capacity_bytes", "Device capacity in bytes", []string{"device"}, nil),
		prometheus.GaugeValue,
		smart.json.Get("user_capacity.bytes").Float(),
		smart.device.device,
	)
	nvme_total_capacity := smart.json.Get("nvme_total_capacity")
	if nvme_total_capacity.Exists() {
		smart.ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("smartctl_device_nvme_capacity_bytes", "NVMe device total capacity bytes", []string{"device"}, nil),
			prometheus.GaugeValue,
			nvme_total_capacity.Float(),
			smart.device.device,
		)
	}
}

func (smart *SMARTctl) mineBlockSize() {
	for _, blockType := range []string{"logical", "physical"} {
		smart.ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("smartctl_device_block_size", "Device block size", []string{"device", "blocks_type"}, nil),
			prometheus.GaugeValue,
			smart.json.Get(fmt.Sprintf("%s_block_size", blockType)).Float(),
			smart.device.device,
			blockType,
		)
	}
}

func (smart *SMARTctl) mineInterfaceSpeed() {
	iSpeed := smart.json.Get("interface_speed")
	if iSpeed.Exists() {
		for _, speedType := range []string{"max", "current"} {
			speed := iSpeed.Get(speedType)
			if speed.Exists() {
				smart.ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("smartctl_device_interface_speed", "Device interface speed, bits per second", []string{"device", "speed_type"}, nil),
					prometheus.GaugeValue,
					speed.Get("bits_per_unit").Float()*speed.Get("units_per_second").Float(),
					smart.device.device,
					speedType,
				)
			}
		}
	}
}

func (smart *SMARTctl) mineDeviceAttribute() {
	attributes := smart.json.Get("ata_smart_attributes.table")
	if attributes.Exists() {
		for _, attribute := range attributes.Array() {
			name := attribute.Get("name").String()
			if name == "" {
				continue
			}

			flags := attribute.Get("flags")
			flagsShort := ""
			flagsLong := ""
			if flags.Exists() {
				flagsShort = flags.Get("string").String()
				flagsLong = smart.mineLongFlags(flags, []string{"prefailure", "updated_online", "performance", "error_rate", "event_count", "auto_keep"})
			}

			id := attribute.Get("id").String()
			
			// Use the same mapping as the old project
			for key, path := range map[string]string{
				"value":  "value",
				"worst":  "worst",
				"thresh": "thresh",
				"raw":    "raw.value",
			} {
				value := attribute.Get(path)
				if value.Exists() {
					smart.ch <- prometheus.MustNewConstMetric(
						prometheus.NewDesc("smartctl_device_attribute", "Device attributes", []string{
							"device",
							"attribute_name",
							"attribute_flags_short",
							"attribute_flags_long",
							"attribute_value_type",
							"attribute_id",
						}, nil),
						prometheus.GaugeValue,
						value.Float(),
						smart.device.device,
						name,
						flagsShort,
						flagsLong,
						key,
						id,
					)
				}
			}
		}
	}
}

func (smart *SMARTctl) minePowerOnSeconds() {
	powerOnTime := smart.json.Get("power_on_time")
	if powerOnTime.Exists() {
		smart.ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("smartctl_device_power_on_seconds", "Device power on seconds", []string{"device"}, nil),
			prometheus.GaugeValue,
			powerOnTime.Get("hours").Float()*3600,
			smart.device.device,
		)
	}
}

func (smart *SMARTctl) mineRotationRate() {
	rotationRate := smart.json.Get("rotation_rate")
	if rotationRate.Exists() {
		smart.ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("smartctl_device_rotation_rate", "Device rotation rate", []string{"device"}, nil),
			prometheus.GaugeValue,
			rotationRate.Float(),
			smart.device.device,
		)
	}
}

func (smart *SMARTctl) mineTemperatures() {
	temperature := smart.json.Get("temperature")
	if temperature.Exists() {
		if current := temperature.Get("current"); current.Exists() {
			smart.ch <- prometheus.MustNewConstMetric(
				prometheus.NewDesc("smartctl_device_temperature", "Device temperature celsius", []string{"device", "temperature_type"}, nil),
				prometheus.GaugeValue,
				current.Float(),
				smart.device.device,
				"current",
			)
		}
	}
}

func (smart *SMARTctl) minePowerCycleCount() {
	powerCycleCount := smart.json.Get("power_cycle_count")
	if powerCycleCount.Exists() {
		smart.ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("smartctl_device_power_cycle_count", "Device power cycle count", []string{"device"}, nil),
			prometheus.GaugeValue,
			powerCycleCount.Float(),
			smart.device.device,
		)
	}
}

func (smart *SMARTctl) mineDeviceSCTStatus() {
	state := smart.json.Get("ata_sct_status.device_state")
	if state.Exists() {
		stateValue := 0.0
		switch state.String() {
		case "active":
			stateValue = 0
		case "standby":
			stateValue = 1
		case "sleep":
			stateValue = 2
		case "dst":
			stateValue = 3
		case "offline":
			stateValue = 4
		case "sct":
			stateValue = 5
		}
		smart.ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("smartctl_device_state", "Device state (0=active, 1=standby, 2=sleep, 3=dst, 4=offline, 5=sct)", []string{"device"}, nil),
			prometheus.GaugeValue,
			stateValue,
			smart.device.device,
		)
	}
}

// Continue with NVMe specific metrics...
func (smart *SMARTctl) mineNvmePercentageUsed() {
	percentageUsed := smart.json.Get("nvme_smart_health_information_log.percentage_used")
	if percentageUsed.Exists() {
		smart.ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("smartctl_device_percentage_used", "Device write percentage used", []string{"device"}, nil),
			prometheus.GaugeValue,
			percentageUsed.Float(),
			smart.device.device,
		)
	}
}

func (smart *SMARTctl) mineNvmeAvailableSpare() {
	availableSpare := smart.json.Get("nvme_smart_health_information_log.available_spare")
	if availableSpare.Exists() {
		smart.ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("smartctl_device_available_spare", "Normalized percentage (0 to 100%) of the remaining spare capacity available", []string{"device"}, nil),
			prometheus.GaugeValue,
			availableSpare.Float(),
			smart.device.device,
		)
	}
}

func (smart *SMARTctl) mineNvmeAvailableSpareThreshold() {
	threshold := smart.json.Get("nvme_smart_health_information_log.available_spare_threshold")
	if threshold.Exists() {
		smart.ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("smartctl_device_available_spare_threshold", "When the Available Spare falls below the threshold indicated in this field, an asynchronous event completion may occur. The value is indicated as a normalized percentage (0 to 100%)", []string{"device"}, nil),
			prometheus.GaugeValue,
			threshold.Float(),
			smart.device.device,
		)
	}
}

func (smart *SMARTctl) mineNvmeCriticalWarning() {
	criticalWarning := smart.json.Get("nvme_smart_health_information_log.critical_warning")
	if criticalWarning.Exists() {
		smart.ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("smartctl_device_critical_warning", "This field indicates critical warnings for the state of the controller", []string{"device"}, nil),
			prometheus.GaugeValue,
			criticalWarning.Float(),
			smart.device.device,
		)
	}
}

func (smart *SMARTctl) mineNvmeMediaErrors() {
	mediaErrors := smart.json.Get("nvme_smart_health_information_log.media_errors")
	if mediaErrors.Exists() {
		smart.ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("smartctl_device_media_errors", "Contains the number of occurrences where the controller detected an unrecovered data integrity error. Errors such as uncorrectable ECC, CRC checksum failure, or LBA tag mismatch are included in this field", []string{"device"}, nil),
			prometheus.GaugeValue,
			mediaErrors.Float(),
			smart.device.device,
		)
	}
}

func (smart *SMARTctl) mineNvmeNumErrLogEntries() {
	numErrLogEntries := smart.json.Get("nvme_smart_health_information_log.num_err_log_entries")
	if numErrLogEntries.Exists() {
		smart.ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("smartctl_device_num_err_log_entries", "Contains the number of Error Information log entries over the life of the controller", []string{"device"}, nil),
			prometheus.GaugeValue,
			numErrLogEntries.Float(),
			smart.device.device,
		)
	}
}

func (smart *SMARTctl) mineNvmeBytesRead() {
	dataUnitsRead := smart.json.Get("nvme_smart_health_information_log.data_units_read")
	if dataUnitsRead.Exists() {
		// NVMe reports in 512-byte units
		bytesRead := dataUnitsRead.Float() * 512
		smart.ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("smartctl_device_bytes_read", "", []string{"device"}, nil),
			prometheus.GaugeValue,
			bytesRead,
			smart.device.device,
		)
	}
}

func (smart *SMARTctl) mineNvmeBytesWritten() {
	dataUnitsWritten := smart.json.Get("nvme_smart_health_information_log.data_units_written")
	if dataUnitsWritten.Exists() {
		// NVMe reports in 512-byte units
		bytesWritten := dataUnitsWritten.Float() * 512
		smart.ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("smartctl_device_bytes_written", "", []string{"device"}, nil),
			prometheus.GaugeValue,
			bytesWritten,
			smart.device.device,
		)
	}
}

func (smart *SMARTctl) mineSmartStatus() {
	smartStatus := smart.json.Get("smart_status")
	if smartStatus.Exists() {
		passed := smartStatus.Get("passed")
		if passed.Exists() {
			value := 0.0
			if passed.Bool() {
				value = 1.0
			}
			smart.ch <- prometheus.MustNewConstMetric(
				prometheus.NewDesc("smartctl_device_smart_status", "General smart status", []string{"device"}, nil),
				prometheus.GaugeValue,
				value,
				smart.device.device,
			)
		}
	}
}

func (smart *SMARTctl) mineDeviceStatistics() {
	statistics := smart.json.Get("ata_device_statistics")
	if statistics.Exists() {
		pages := statistics.Get("pages")
		if pages.Exists() {
			for _, page := range pages.Array() {
				table := page.Get("table")
				if table.Exists() {
					for _, stat := range table.Array() {
						name := stat.Get("name").String()
						if name == "" {
							continue
						}

						flags := stat.Get("flags")
						flagsShort := ""
						flagsLong := ""
						if flags.Exists() {
							flagsShort = flags.Get("string").String()
							flagsLong = smart.mineLongFlags(flags, []string{"valid", "normalized", "supports_dsn", "monitored_condition_met"})
						}

						value := stat.Get("value")
						if value.Exists() {
							smart.ch <- prometheus.MustNewConstMetric(
								prometheus.NewDesc("smartctl_device_statistics", "Device statistics", []string{
									"device",
									"statistic_table",
									"statistic_name",
									"statistic_flags_short",
									"statistic_flags_long",
								}, nil),
								prometheus.GaugeValue,
								value.Float(),
								smart.device.device,
								page.Get("name").String(),
								name,
								flagsShort,
								flagsLong,
							)
						}
					}
				}
			}
		}
	}
}

func (smart *SMARTctl) mineLongFlags(json gjson.Result, flags []string) string {
	var result []string
	for _, flag := range flags {
		if json.Get(flag).Bool() {
			result = append(result, flag)
		}
	}
	return strings.Join(result, ",")
}

func (smart *SMARTctl) mineDeviceErrorLog() {
	errorLog := smart.json.Get("ata_smart_error_log")
	if errorLog.Exists() {
		summary := errorLog.Get("summary")
		if summary.Exists() {
			count := summary.Get("count")
			if count.Exists() {
				smart.ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("smartctl_device_error_log_count", "Device SMART error log count", []string{"device", "error_log_type"}, nil),
					prometheus.GaugeValue,
					count.Float(),
					smart.device.device,
					"ata_smart_error_log",
				)
			}
		}
	}
}

func (smart *SMARTctl) mineDeviceSelfTestLog() {
	selfTestLog := smart.json.Get("ata_smart_self_test_log")
	if selfTestLog.Exists() {
		standard := selfTestLog.Get("standard")
		if standard.Exists() {
			count := standard.Get("count")
			errorCount := standard.Get("error_count_total")
			if count.Exists() {
				smart.ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("smartctl_device_self_test_log_count", "Device SMART self test log count", []string{"device", "self_test_log_type"}, nil),
					prometheus.GaugeValue,
					count.Float(),
					smart.device.device,
					"ata_smart_self_test_log",
				)
			}
			if errorCount.Exists() {
				smart.ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("smartctl_device_self_test_log_error_count", "Device SMART self test log error count", []string{"device", "self_test_log_type"}, nil),
					prometheus.GaugeValue,
					errorCount.Float(),
					smart.device.device,
					"ata_smart_self_test_log",
				)
			}
		}
	}
}

func (smart *SMARTctl) mineDeviceERC() {
	erc := smart.json.Get("ata_sct_erc")
	if erc.Exists() {
		for _, opType := range []string{"read", "write"} {
			seconds := erc.Get(opType + "_recovery_time_limit")
			if seconds.Exists() {
				smart.ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("smartctl_device_erc_seconds", "Device SMART Error Recovery Control Seconds", []string{"device", "op_type"}, nil),
					prometheus.GaugeValue,
					seconds.Float(),
					smart.device.device,
					opType,
				)
			}
		}
	}
}

// SCSI specific metrics
func (smart *SMARTctl) mineSCSIGrownDefectList() {
	grownDefectList := smart.json.Get("scsi_grown_defect_list")
	if grownDefectList.Exists() {
		smart.ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("smartctl_scsi_grown_defect_list", "Device SCSI grown defect list counter", []string{"device"}, nil),
			prometheus.GaugeValue,
			grownDefectList.Float(),
			smart.device.device,
		)
	}
}

func (smart *SMARTctl) mineSCSIErrorCounterLog() {
	errorCounterLog := smart.json.Get("scsi_error_counter_log")
	if errorCounterLog.Exists() {
		read := errorCounterLog.Get("read")
		if read.Exists() {
			if correctedByRereads := read.Get("errors_corrected_by_rereads_rewrites"); correctedByRereads.Exists() {
				smart.ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("smartctl_read_errors_corrected_by_rereads_rewrites", "Read Errors Corrected by ReReads/ReWrites", []string{"device"}, nil),
					prometheus.GaugeValue,
					correctedByRereads.Float(),
					smart.device.device,
				)
			}
			if correctedByEccFast := read.Get("errors_corrected_by_eccfast"); correctedByEccFast.Exists() {
				smart.ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("smartctl_read_errors_corrected_by_eccfast", "Read Errors Corrected by ECC Fast", []string{"device"}, nil),
					prometheus.GaugeValue,
					correctedByEccFast.Float(),
					smart.device.device,
				)
			}
			if correctedByEccDelayed := read.Get("errors_corrected_by_eccdelayed"); correctedByEccDelayed.Exists() {
				smart.ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("smartctl_read_errors_corrected_by_eccdelayed", "Read Errors Corrected by ECC Delayed", []string{"device"}, nil),
					prometheus.GaugeValue,
					correctedByEccDelayed.Float(),
					smart.device.device,
				)
			}
			if totalUncorrected := read.Get("total_uncorrected_errors"); totalUncorrected.Exists() {
				smart.ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("smartctl_read_total_uncorrected_errors", "Read Total Uncorrected Errors", []string{"device"}, nil),
					prometheus.GaugeValue,
					totalUncorrected.Float(),
					smart.device.device,
				)
			}
		}

		write := errorCounterLog.Get("write")
		if write.Exists() {
			if correctedByRereads := write.Get("errors_corrected_by_rereads_rewrites"); correctedByRereads.Exists() {
				smart.ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("smartctl_write_errors_corrected_by_rereads_rewrites", "Write Errors Corrected by ReReads/ReWrites", []string{"device"}, nil),
					prometheus.GaugeValue,
					correctedByRereads.Float(),
					smart.device.device,
				)
			}
			if correctedByEccFast := write.Get("errors_corrected_by_eccfast"); correctedByEccFast.Exists() {
				smart.ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("smartctl_write_errors_corrected_by_eccfast", "Write Errors Corrected by ECC Fast", []string{"device"}, nil),
					prometheus.GaugeValue,
					correctedByEccFast.Float(),
					smart.device.device,
				)
			}
			if correctedByEccDelayed := write.Get("errors_corrected_by_eccdelayed"); correctedByEccDelayed.Exists() {
				smart.ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("smartctl_write_errors_corrected_by_eccdelayed", "Write Errors Corrected by ECC Delayed", []string{"device"}, nil),
					prometheus.GaugeValue,
					correctedByEccDelayed.Float(),
					smart.device.device,
				)
			}
			if totalUncorrected := write.Get("total_uncorrected_errors"); totalUncorrected.Exists() {
				smart.ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("smartctl_write_total_uncorrected_errors", "Write Total Uncorrected Errors", []string{"device"}, nil),
					prometheus.GaugeValue,
					totalUncorrected.Float(),
					smart.device.device,
				)
			}
		}
	}
}

func (smart *SMARTctl) mineSCSIBytesRead() {
	// Implementation for SCSI bytes read
	// This would need to be implemented based on SCSI log pages
}

func (smart *SMARTctl) mineSCSIBytesWritten() {
	// Implementation for SCSI bytes written
	// This would need to be implemented based on SCSI log pages
}

// scanDevices scans for available devices
func scanDevices() []Device {
	if smartctlFakeData {
		return []Device{} // Return empty for fake data mode
	}

	// Initialize logger if it's nil
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	}

	json := readSMARTctlDevices(logger)
	scanDevices := json.Get("devices").Array()
	var scanDeviceResult []Device
	for _, d := range scanDevices {
		deviceName := d.Get("name").String()
		deviceType := d.Get("type").String()

		// SATA devices are reported as SCSI during scan - fallback to auto scraping
		if deviceType == "scsi" {
			deviceType = "auto"
		}

		deviceLabel := buildDeviceLabel(deviceName, deviceType)
		logger.Info("Found device", "name", deviceLabel)
		device := Device{
			Name:  deviceName,
			Type:  deviceType,
			Label: deviceLabel,
		}
		scanDeviceResult = append(scanDeviceResult, device)
	}
	return scanDeviceResult
} 
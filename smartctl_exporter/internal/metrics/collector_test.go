package metrics

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

// Mock data for testing
const mockSmartctlJSON = `{
	"json_format_version": [1, 0],
	"smartctl": {
		"version": [7, 2],
		"svn_revision": "5155",
		"build_info": "x86_64-linux-5.4.0-74-generic",
		"exit_status": 0
	},
	"device": {
		"name": "/dev/sda",
		"type": "ata",
		"protocol": "ATA"
	},
	"model_name": "Samsung SSD 860 EVO 500GB",
	"serial_number": "S3Z2NB0K123456",
	"model_family": "Samsung based SSDs",
	"firmware_version": "RVT01B6Q",
	"user_capacity": {
		"blocks": 976773168,
		"bytes": 500107862016
	},
	"logical_block_size": 512,
	"physical_block_size": 512,
	"rotation_rate": 0,
	"interface_speed": {
		"max": {
			"units_per_second": 600000000,
			"bits_per_unit": 1
		},
		"current": {
			"units_per_second": 600000000,
			"bits_per_unit": 1
		}
	},
	"power_on_time": {
		"hours": 1234
	},
	"power_cycle_count": 567,
	"temperature": {
		"current": 35
	},
	"ata_smart_attributes": {
		"table": [
			{
				"id": 4,
				"name": "Start_Stop_Count",
				"value": 100,
				"worst": 100,
				"thresh": 0,
				"raw": {
					"value": 100,
					"string": "100"
				},
				"flags": {
					"value": 50,
					"string": "PO--CK ",
					"prefailure": true,
					"updated_online": true,
					"performance": false,
					"error_rate": false,
					"event_count": true,
					"auto_keep": false
				}
			},
			{
				"id": 9,
				"name": "Power_On_Hours",
				"value": 99,
				"worst": 99,
				"thresh": 0,
				"raw": {
					"value": 1234,
					"string": "1234"
				},
				"flags": {
					"value": 50,
					"string": "PO--CK ",
					"prefailure": true,
					"updated_online": true,
					"performance": false,
					"error_rate": false,
					"event_count": true,
					"auto_keep": false
				}
			}
		]
	},
	"smart_status": {
		"passed": true
	},
	"ata_sct_status": {
		"device_state": "active"
	},
	"ata_device_statistics": {
		"pages": [
			{
				"name": "General Statistics",
				"table": [
					{
						"name": "Lifetime Power-On Resets",
						"value": 567,
						"flags": {
							"value": 7,
							"string": "---",
							"valid": true,
							"normalized": false,
							"supports_dsn": false,
							"monitored_condition_met": false
						}
					}
				]
			}
		]
	},
	"ata_smart_error_log": {
		"summary": {
			"count": 0
		}
	},
	"ata_smart_self_test_log": {
		"standard": {
			"count": 5,
			"error_count_total": 0
		}
	},
	"ata_sct_erc": {
		"read_recovery_time_limit": 70,
		"write_recovery_time_limit": 70
	}
}`

const mockNVMeJSON = `{
	"json_format_version": [1, 0],
	"smartctl": {
		"version": [7, 2],
		"svn_revision": "5155",
		"build_info": "x86_64-linux-5.4.0-74-generic",
		"exit_status": 0
	},
	"device": {
		"name": "/dev/nvme0n1",
		"type": "nvme",
		"protocol": "NVMe"
	},
	"model_name": "Samsung SSD 980 PRO 1TB",
	"serial_number": "S5P2NG0N123456",
	"firmware_version": "5B2QGXA7",
	"nvme_total_capacity": 1000204886016,
	"user_capacity": {
		"blocks": 1953525168,
		"bytes": 1000204886016
	},
	"logical_block_size": 512,
	"physical_block_size": 512,
	"nvme_smart_health_information_log": {
		"percentage_used": 5,
		"available_spare": 100,
		"available_spare_threshold": 10,
		"critical_warning": 0,
		"media_errors": 0,
		"num_err_log_entries": 0,
		"data_units_read": 12345678,
		"data_units_written": 9876543
	},
	"smart_status": {
		"passed": true
	}
}`

const mockSCSIJSON = `{
	"json_format_version": [1, 0],
	"smartctl": {
		"version": [7, 2],
		"svn_revision": "5155",
		"build_info": "x86_64-linux-5.4.0-74-generic",
		"exit_status": 0
	},
	"device": {
		"name": "/dev/sdb",
		"type": "scsi",
		"protocol": "SCSI"
	},
	"scsi_vendor": "SEAGATE",
	"scsi_product": "ST1000NM0033",
	"scsi_revision": "0006",
	"scsi_version": "SPC-4",
	"serial_number": "Z1W123456",
	"user_capacity": {
		"blocks": 1953525168,
		"bytes": 1000204886016
	},
	"logical_block_size": 512,
	"physical_block_size": 512,
	"scsi_grown_defect_list": 0,
	"scsi_error_counter_log": {
		"read": {
			"errors_corrected_by_rereads_rewrites": 0,
			"errors_corrected_by_eccfast": 0,
			"errors_corrected_by_eccdelayed": 0,
			"total_uncorrected_errors": 0
		},
		"write": {
			"errors_corrected_by_rereads_rewrites": 0,
			"errors_corrected_by_eccfast": 0,
			"errors_corrected_by_eccdelayed": 0,
			"total_uncorrected_errors": 0
		}
	},
	"smart_status": {
		"passed": true
	}
}`

func TestNewSmartctlManagerCollector(t *testing.T) {
	// Set fake data mode to avoid actual device scanning
	originalFakeData := smartctlFakeData
	smartctlFakeData = true
	defer func() { smartctlFakeData = originalFakeData }()

	collector := NewSmartctlManagerCollector()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.NotNil(t, collector.devices)
	assert.IsType(t, &SmartctlManagerCollector{}, collector)
}

func TestNewSMARTctl(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ch := make(chan prometheus.Metric, 10)
	json := gjson.Parse(mockSmartctlJSON)

	smartctl := NewSMARTctl(logger, json, ch)

	assert.NotNil(t, smartctl)
	assert.Equal(t, "sda", smartctl.device.device)
	assert.Equal(t, "ata", smartctl.device.interface_)
	assert.Equal(t, "Samsung SSD 860 EVO 500GB", smartctl.device.model)
	assert.Equal(t, "S3Z2NB0K123456", smartctl.device.serial)
	assert.Equal(t, "Samsung based SSDs", smartctl.device.family)
}

func TestNewSMARTctlWithUnknownModel(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	jsonStr := `{
		"device": {"name": "/dev/sda", "type": "ata", "protocol": "ATA"},
		"serial_number": "123456"
	}`
	json := gjson.Parse(jsonStr)
	ch := make(chan prometheus.Metric, 100)
	
	smart := NewSMARTctl(logger, json, ch)
	
	assert.Equal(t, "unknown", smart.device.model)
}

func TestNewSMARTctlWithSCSIModel(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	jsonStr := `{
		"device": {"name": "/dev/sdb", "type": "scsi", "protocol": "SCSI"},
		"scsi_model_name": "SCSI Test Drive",
		"serial_number": "123456"
	}`
	json := gjson.Parse(jsonStr)
	ch := make(chan prometheus.Metric, 100)
	
	smart := NewSMARTctl(logger, json, ch)
	
	assert.Equal(t, "SCSI Test Drive", smart.device.model)
}

func TestSMARTctlMineExitStatus(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	json := gjson.Parse(mockSmartctlJSON)
	ch := make(chan prometheus.Metric, 100)
	
	smart := NewSMARTctl(logger, json, ch)
	smart.mineExitStatus()
	
	// Verify metric was collected
	select {
	case metric := <-ch:
		assert.NotNil(t, metric)
	default:
		t.Error("Expected exit status metric to be collected")
	}
}

func TestSMARTctlMineDevice(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	json := gjson.Parse(mockSmartctlJSON)
	ch := make(chan prometheus.Metric, 100)
	
	smart := NewSMARTctl(logger, json, ch)
	smart.mineDevice()
	
	// Verify metric was collected
	select {
	case metric := <-ch:
		assert.NotNil(t, metric)
	default:
		t.Error("Expected device metric to be collected")
	}
}

func TestSMARTctlMineCapacity(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	json := gjson.Parse(mockSmartctlJSON)
	ch := make(chan prometheus.Metric, 100)
	
	smart := NewSMARTctl(logger, json, ch)
	smart.mineCapacity()
	
	// Should collect at least 2 metrics (blocks and bytes)
	metricsCollected := 0
	for i := 0; i < 10; i++ {
		select {
		case metric := <-ch:
			assert.NotNil(t, metric)
			metricsCollected++
		default:
			break
		}
	}
	assert.GreaterOrEqual(t, metricsCollected, 2)
}

func TestSMARTctlMineCapacityWithNVMe(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	json := gjson.Parse(mockNVMeJSON)
	ch := make(chan prometheus.Metric, 100)
	
	smart := NewSMARTctl(logger, json, ch)
	smart.mineCapacity()
	
	// Should collect at least 3 metrics (blocks, bytes, and nvme_capacity)
	metricsCollected := 0
	for i := 0; i < 10; i++ {
		select {
		case metric := <-ch:
			assert.NotNil(t, metric)
			metricsCollected++
		default:
			break
		}
	}
	assert.GreaterOrEqual(t, metricsCollected, 3)
}

func TestSMARTctlMineBlockSize(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	json := gjson.Parse(mockSmartctlJSON)
	ch := make(chan prometheus.Metric, 100)
	
	smart := NewSMARTctl(logger, json, ch)
	smart.mineBlockSize()
	
	// Should collect 2 metrics (logical and physical)
	metricsCollected := 0
	for i := 0; i < 10; i++ {
		select {
		case metric := <-ch:
			assert.NotNil(t, metric)
			metricsCollected++
		default:
			break
		}
	}
	assert.Equal(t, 2, metricsCollected)
}

func TestSMARTctlMineInterfaceSpeed(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	json := gjson.Parse(mockSmartctlJSON)
	ch := make(chan prometheus.Metric, 100)
	
	smart := NewSMARTctl(logger, json, ch)
	smart.mineInterfaceSpeed()
	
	// Should collect 2 metrics (max and current)
	metricsCollected := 0
	for i := 0; i < 10; i++ {
		select {
		case metric := <-ch:
			assert.NotNil(t, metric)
			metricsCollected++
		default:
			break
		}
	}
	assert.Equal(t, 2, metricsCollected)
}

func TestSMARTctlMineDeviceAttribute(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	json := gjson.Parse(mockSmartctlJSON)
	ch := make(chan prometheus.Metric, 100)
	
	smart := NewSMARTctl(logger, json, ch)
	smart.mineDeviceAttribute()
	
	// Should collect multiple metrics for each attribute and value type
	metricsCollected := 0
	for i := 0; i < 20; i++ {
		select {
		case metric := <-ch:
			assert.NotNil(t, metric)
			metricsCollected++
		default:
			break
		}
	}
	assert.Greater(t, metricsCollected, 0)
}

func TestSMARTctlMinePowerOnSeconds(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	json := gjson.Parse(mockSmartctlJSON)
	ch := make(chan prometheus.Metric, 100)
	
	smart := NewSMARTctl(logger, json, ch)
	smart.minePowerOnSeconds()
	
	select {
	case metric := <-ch:
		assert.NotNil(t, metric)
	default:
		t.Error("Expected power on seconds metric to be collected")
	}
}

func TestSMARTctlMineRotationRate(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	json := gjson.Parse(mockSmartctlJSON)
	ch := make(chan prometheus.Metric, 100)
	
	smart := NewSMARTctl(logger, json, ch)
	smart.mineRotationRate()
	
	select {
	case metric := <-ch:
		assert.NotNil(t, metric)
	default:
		t.Error("Expected rotation rate metric to be collected")
	}
}

func TestSMARTctlMineTemperatures(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	json := gjson.Parse(mockSmartctlJSON)
	ch := make(chan prometheus.Metric, 100)
	
	smart := NewSMARTctl(logger, json, ch)
	smart.mineTemperatures()
	
	select {
	case metric := <-ch:
		assert.NotNil(t, metric)
	default:
		t.Error("Expected temperature metric to be collected")
	}
}

func TestSMARTctlMinePowerCycleCount(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	json := gjson.Parse(mockSmartctlJSON)
	ch := make(chan prometheus.Metric, 100)
	
	smart := NewSMARTctl(logger, json, ch)
	smart.minePowerCycleCount()
	
	select {
	case metric := <-ch:
		assert.NotNil(t, metric)
	default:
		t.Error("Expected power cycle count metric to be collected")
	}
}

func TestSMARTctlMineDeviceSCTStatus(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	json := gjson.Parse(mockSmartctlJSON)
	ch := make(chan prometheus.Metric, 100)
	
	smart := NewSMARTctl(logger, json, ch)
	smart.mineDeviceSCTStatus()
	
	select {
	case metric := <-ch:
		assert.NotNil(t, metric)
	default:
		t.Error("Expected device state metric to be collected")
	}
}

func TestSMARTctlMineSmartStatus(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	json := gjson.Parse(mockSmartctlJSON)
	ch := make(chan prometheus.Metric, 100)
	
	smart := NewSMARTctl(logger, json, ch)
	smart.mineSmartStatus()
	
	select {
	case metric := <-ch:
		assert.NotNil(t, metric)
	default:
		t.Error("Expected SMART status metric to be collected")
	}
}

func TestSMARTctlMineDeviceStatistics(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	json := gjson.Parse(mockSmartctlJSON)
	ch := make(chan prometheus.Metric, 100)
	
	smart := NewSMARTctl(logger, json, ch)
	smart.mineDeviceStatistics()
	
	select {
	case metric := <-ch:
		assert.NotNil(t, metric)
	default:
		t.Error("Expected device statistics metric to be collected")
	}
}

func TestSMARTctlMineDeviceErrorLog(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	json := gjson.Parse(mockSmartctlJSON)
	ch := make(chan prometheus.Metric, 100)
	
	smart := NewSMARTctl(logger, json, ch)
	smart.mineDeviceErrorLog()
	
	select {
	case metric := <-ch:
		assert.NotNil(t, metric)
	default:
		t.Error("Expected device error log metric to be collected")
	}
}

func TestSMARTctlMineDeviceSelfTestLog(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	json := gjson.Parse(mockSmartctlJSON)
	ch := make(chan prometheus.Metric, 100)
	
	smart := NewSMARTctl(logger, json, ch)
	smart.mineDeviceSelfTestLog()
	
	// Should collect 2 metrics (count and error_count)
	metricsCollected := 0
	for i := 0; i < 10; i++ {
		select {
		case metric := <-ch:
			assert.NotNil(t, metric)
			metricsCollected++
		default:
			break
		}
	}
	assert.Equal(t, 2, metricsCollected)
}

func TestSMARTctlMineDeviceERC(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	json := gjson.Parse(mockSmartctlJSON)
	ch := make(chan prometheus.Metric, 100)
	
	smart := NewSMARTctl(logger, json, ch)
	smart.mineDeviceERC()
	
	// Should collect 2 metrics (read and write)
	metricsCollected := 0
	for i := 0; i < 10; i++ {
		select {
		case metric := <-ch:
			assert.NotNil(t, metric)
			metricsCollected++
		default:
			break
		}
	}
	assert.Equal(t, 2, metricsCollected)
}

// NVMe specific tests
func TestSMARTctlMineNvmePercentageUsed(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	json := gjson.Parse(mockNVMeJSON)
	ch := make(chan prometheus.Metric, 100)
	
	smart := NewSMARTctl(logger, json, ch)
	smart.mineNvmePercentageUsed()
	
	select {
	case metric := <-ch:
		assert.NotNil(t, metric)
	default:
		t.Error("Expected NVMe percentage used metric to be collected")
	}
}

func TestSMARTctlMineNvmeAvailableSpare(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	json := gjson.Parse(mockNVMeJSON)
	ch := make(chan prometheus.Metric, 100)
	
	smart := NewSMARTctl(logger, json, ch)
	smart.mineNvmeAvailableSpare()
	
	select {
	case metric := <-ch:
		assert.NotNil(t, metric)
	default:
		t.Error("Expected NVMe available spare metric to be collected")
	}
}

func TestSMARTctlMineNvmeAvailableSpareThreshold(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	json := gjson.Parse(mockNVMeJSON)
	ch := make(chan prometheus.Metric, 100)
	
	smart := NewSMARTctl(logger, json, ch)
	smart.mineNvmeAvailableSpareThreshold()
	
	select {
	case metric := <-ch:
		assert.NotNil(t, metric)
	default:
		t.Error("Expected NVMe available spare threshold metric to be collected")
	}
}

func TestSMARTctlMineNvmeCriticalWarning(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	json := gjson.Parse(mockNVMeJSON)
	ch := make(chan prometheus.Metric, 100)
	
	smart := NewSMARTctl(logger, json, ch)
	smart.mineNvmeCriticalWarning()
	
	select {
	case metric := <-ch:
		assert.NotNil(t, metric)
	default:
		t.Error("Expected NVMe critical warning metric to be collected")
	}
}

func TestSMARTctlMineNvmeMediaErrors(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	json := gjson.Parse(mockNVMeJSON)
	ch := make(chan prometheus.Metric, 100)
	
	smart := NewSMARTctl(logger, json, ch)
	smart.mineNvmeMediaErrors()
	
	select {
	case metric := <-ch:
		assert.NotNil(t, metric)
	default:
		t.Error("Expected NVMe media errors metric to be collected")
	}
}

func TestSMARTctlMineNvmeNumErrLogEntries(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	json := gjson.Parse(mockNVMeJSON)
	ch := make(chan prometheus.Metric, 100)
	
	smart := NewSMARTctl(logger, json, ch)
	smart.mineNvmeNumErrLogEntries()
	
	select {
	case metric := <-ch:
		assert.NotNil(t, metric)
	default:
		t.Error("Expected NVMe error log entries metric to be collected")
	}
}

func TestSMARTctlMineNvmeBytesRead(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	json := gjson.Parse(mockNVMeJSON)
	ch := make(chan prometheus.Metric, 100)
	
	smart := NewSMARTctl(logger, json, ch)
	smart.mineNvmeBytesRead()
	
	select {
	case metric := <-ch:
		assert.NotNil(t, metric)
	default:
		t.Error("Expected NVMe bytes read metric to be collected")
	}
}

func TestSMARTctlMineNvmeBytesWritten(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	json := gjson.Parse(mockNVMeJSON)
	ch := make(chan prometheus.Metric, 100)
	
	smart := NewSMARTctl(logger, json, ch)
	smart.mineNvmeBytesWritten()
	
	select {
	case metric := <-ch:
		assert.NotNil(t, metric)
	default:
		t.Error("Expected NVMe bytes written metric to be collected")
	}
}

// SCSI specific tests
func TestSMARTctlMineSCSIGrownDefectList(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	json := gjson.Parse(mockSCSIJSON)
	ch := make(chan prometheus.Metric, 100)
	
	smart := NewSMARTctl(logger, json, ch)
	smart.mineSCSIGrownDefectList()
	
	select {
	case metric := <-ch:
		assert.NotNil(t, metric)
	default:
		t.Error("Expected SCSI grown defect list metric to be collected")
	}
}

func TestSMARTctlMineSCSIErrorCounterLog(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	json := gjson.Parse(mockSCSIJSON)
	ch := make(chan prometheus.Metric, 100)
	
	smart := NewSMARTctl(logger, json, ch)
	smart.mineSCSIErrorCounterLog()
	
	// Should collect multiple metrics for read and write errors
	metricsCollected := 0
	for i := 0; i < 20; i++ {
		select {
		case metric := <-ch:
			assert.NotNil(t, metric)
			metricsCollected++
		default:
			break
		}
	}
	assert.Greater(t, metricsCollected, 0)
}

// Utility function tests
func TestBuildDeviceLabel(t *testing.T) {
	tests := []struct {
		name       string
		inputName  string
		inputType  string
		expected   string
	}{
		{
			name:      "simple device name",
			inputName: "/dev/sda",
			inputType: "ata",
			expected:  "sda",
		},
		{
			name:      "device with type containing comma",
			inputName: "/dev/sdb",
			inputType: "ata,auto",
			expected:  "sdb_ata_auto",
		},
		{
			name:      "device with disk by-id path",
			inputName: "/dev/disk/by-id/ata-Samsung_SSD_860_EVO_500GB_S3Z2NB0K123456",
			inputType: "ata",
			expected:  "ata-Samsung_SSD_860_EVO_500GB_S3Z2NB0K123456",
		},
		{
			name:      "device with disk by-path",
			inputName: "/dev/disk/by-path/pci-0000:00:1f.2-ata-1",
			inputType: "ata",
			expected:  "pci-0000:00:1f.2-ata-1",
		},
		{
			name:      "nvme device",
			inputName: "/dev/nvme0n1",
			inputType: "nvme",
			expected:  "nvme0n1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildDeviceLabel(tt.inputName, tt.inputType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetStringIfExists(t *testing.T) {
	jsonStr := `{
		"existing_key": "existing_value",
		"empty_key": "",
		"nested": {
			"key": "nested_value"
		}
	}`
	json := gjson.Parse(jsonStr)

	tests := []struct {
		name         string
		key          string
		defaultValue string
		expected     string
	}{
		{
			name:         "existing key",
			key:          "existing_key",
			defaultValue: "default",
			expected:     "existing_value",
		},
		{
			name:         "non-existing key",
			key:          "non_existing_key",
			defaultValue: "default",
			expected:     "default",
		},
		{
			name:         "empty key",
			key:          "empty_key",
			defaultValue: "default",
			expected:     "",
		},
		{
			name:         "nested key",
			key:          "nested.key",
			defaultValue: "default",
			expected:     "nested_value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetStringIfExists(json, tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		isValid  bool
	}{
		{
			name:    "valid JSON",
			input:   `{"key": "value"}`,
			isValid: true,
		},
		{
			name:    "invalid JSON",
			input:   `{invalid json}`,
			isValid: false,
		},
		{
			name:    "empty string",
			input:   "",
			isValid: false,
		},
		{
			name:    "valid complex JSON",
			input:   mockSmartctlJSON,
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseJSON(tt.input)
			assert.NotNil(t, result)
			
			if tt.isValid {
				assert.True(t, result.Exists())
			} else {
				// Invalid JSON should return empty object
				assert.False(t, result.Get("key").Exists())
			}
		})
	}
}

func TestMineLongFlags(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	json := gjson.Parse(mockSmartctlJSON)
	ch := make(chan prometheus.Metric, 100)
	
	smart := NewSMARTctl(logger, json, ch)
	
	flagsJSON := gjson.Parse(`{
		"prefailure": true,
		"updated_online": true,
		"performance": false,
		"error_rate": false,
		"event_count": true,
		"auto_keep": false
	}`)
	
	flags := []string{"prefailure", "updated_online", "performance", "error_rate", "event_count", "auto_keep"}
	result := smart.mineLongFlags(flagsJSON, flags)
	
	expected := "prefailure,updated_online,event_count"
	assert.Equal(t, expected, result)
}

func TestMineLongFlagsEmpty(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	json := gjson.Parse(mockSmartctlJSON)
	ch := make(chan prometheus.Metric, 100)
	
	smart := NewSMARTctl(logger, json, ch)
	
	flagsJSON := gjson.Parse(`{
		"prefailure": false,
		"updated_online": false,
		"performance": false
	}`)
	
	flags := []string{"prefailure", "updated_online", "performance"}
	result := smart.mineLongFlags(flagsJSON, flags)
	
	assert.Equal(t, "", result)
}

func TestSMARTDeviceString(t *testing.T) {
	device := Device{
		Name:  "/dev/sda",
		Type:  "ata",
		Label: "sda",
	}
	
	expected := "/dev/sda;ata (sda)"
	assert.Equal(t, expected, device.String())
}

func TestCollectFullFlow(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	json := gjson.Parse(mockSmartctlJSON)
	ch := make(chan prometheus.Metric, 1000)
	
	smart := NewSMARTctl(logger, json, ch)
	smart.Collect()
	
	// Count collected metrics
	metricsCollected := 0
	timeout := time.After(1 * time.Second)
	
	for {
		select {
		case metric := <-ch:
			assert.NotNil(t, metric)
			metricsCollected++
		case <-timeout:
			goto done
		default:
			if len(ch) == 0 {
				goto done
			}
		}
	}
	
done:
	// Should collect multiple metrics from various mining functions
	assert.Greater(t, metricsCollected, 10, "Expected to collect multiple metrics")
}

func BenchmarkNewSMARTctl(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	json := gjson.Parse(mockSmartctlJSON)
	ch := make(chan prometheus.Metric, 100)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewSMARTctl(logger, json, ch)
	}
}

func BenchmarkSMARTctlCollect(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	json := gjson.Parse(mockSmartctlJSON)
	ch := make(chan prometheus.Metric, 1000)
	
	smart := NewSMARTctl(logger, json, ch)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		smart.Collect()
		// Drain the channel
		for len(ch) > 0 {
			<-ch
		}
	}
}

func BenchmarkBuildDeviceLabel(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buildDeviceLabel("/dev/disk/by-id/ata-Samsung_SSD_860_EVO_500GB_S3Z2NB0K123456", "ata,auto")
	}
} 
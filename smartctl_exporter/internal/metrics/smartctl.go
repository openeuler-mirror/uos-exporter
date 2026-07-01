package metrics

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"smartctl_exporter/internal/exporter"
	"smartctl_exporter/pkg/utils"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/tidwall/gjson"
)

// Device represents a storage device
type Device struct {
	Name  string
	Type  string
	Label string
}

func (d Device) String() string {
	return d.Name + ";" + d.Type + " (" + d.Label + ")"
}

// SMARTDevice - short info about device
type SMARTDevice struct {
	device     string
	serial     string
	family     string
	model      string
	interface_ string
	protocol   string
}

// JSONCache caching json
type JSONCache struct {
	JSON        gjson.Result
	LastCollect time.Time
}

const (
	smartctlPath           = "/usr/sbin/smartctl"
	smartctlPowerModeCheck = "standby"
)

var (
	jsonCache sync.Map
	logger    *slog.Logger

	// Configuration variables
	smartctlInterval        = 60 * time.Second
	smartctlRescanInterval  = 10 * time.Minute
	smartctlScan            = false
	smartctlDevices         []string
	smartctlDeviceExclude   = ""
	smartctlDeviceInclude   = ""
	smartctlScanDeviceTypes []string
	smartctlFakeData        = false
)

func init() {
	jsonCache.Store("", JSONCache{})
	logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Register all metrics
	exporter.Register(NewSmartctlVersion())
	exporter.Register(NewSmartctlDevice())
	exporter.Register(NewSmartctlDeviceCount())
	exporter.Register(NewSmartctlDeviceCapacityBlocks())
	exporter.Register(NewSmartctlDeviceCapacityBytes())
	exporter.Register(NewSmartctlDeviceTotalCapacityBytes())
	exporter.Register(NewSmartctlDeviceBlockSize())
	exporter.Register(NewSmartctlDeviceInterfaceSpeed())
	exporter.Register(NewSmartctlDeviceAttribute())
	exporter.Register(NewSmartctlDevicePowerOnSeconds())
	exporter.Register(NewSmartctlDeviceRotationRate())
	exporter.Register(NewSmartctlDeviceTemperature())
	exporter.Register(NewSmartctlDevicePowerCycleCount())
	exporter.Register(NewSmartctlDevicePercentageUsed())
	exporter.Register(NewSmartctlDeviceAvailableSpare())
	exporter.Register(NewSmartctlDeviceAvailableSpareThreshold())
	exporter.Register(NewSmartctlDeviceCriticalWarning())
	exporter.Register(NewSmartctlDeviceMediaErrors())
	exporter.Register(NewSmartctlDeviceNumErrLogEntries())
	exporter.Register(NewSmartctlDeviceBytesRead())
	exporter.Register(NewSmartctlDeviceBytesWritten())
	exporter.Register(NewSmartctlDeviceSmartStatus())
	exporter.Register(NewSmartctlDeviceExitStatus())
	exporter.Register(NewSmartctlDeviceState())
	exporter.Register(NewSmartctlDeviceStatistics())
	exporter.Register(NewSmartctlDeviceErrorLogCount())
	exporter.Register(NewSmartctlDeviceSelfTestLogCount())
	exporter.Register(NewSmartctlDeviceSelfTestLogErrorCount())
	exporter.Register(NewSmartctlDeviceERCSeconds())
	exporter.Register(NewSmartctlSCSIGrownDefectList())
	exporter.Register(NewSmartctlReadErrorsCorrectedByRereadsRewrites())
	exporter.Register(NewSmartctlReadErrorsCorrectedByEccFast())
	exporter.Register(NewSmartctlReadErrorsCorrectedByEccDelayed())
	exporter.Register(NewSmartctlReadTotalUncorrectedErrors())
	exporter.Register(NewSmartctlWriteErrorsCorrectedByRereadsRewrites())
	exporter.Register(NewSmartctlWriteErrorsCorrectedByEccFast())
	exporter.Register(NewSmartctlWriteErrorsCorrectedByEccDelayed())
	exporter.Register(NewSmartctlWriteTotalUncorrectedErrors())
}

// Utility functions
func buildDeviceLabel(inputName string, inputType string) string {
	devReg := regexp.MustCompile(`^/dev/(?:disk/by-id/|disk/by-path/|)`)
	deviceName := strings.ReplaceAll(devReg.ReplaceAllString(inputName, ""), "/", "_")

	if strings.Contains(inputType, ",") {
		return deviceName + "_" + strings.ReplaceAll(inputType, ",", "_")
	}

	return deviceName
}

func GetStringIfExists(json gjson.Result, key string, defaultValue string) string {
	if obj := json.Get(key); obj.Exists() {
		return obj.String()
	}
	return defaultValue
}

// Parse json to gjson object
func parseJSON(data string) gjson.Result {
	if !gjson.Valid(data) {
		return gjson.Parse("{}")
	}
	return gjson.Parse(data)
}

// 设备验证结构体
type DeviceValidator struct {
	allowedTypes map[string]bool
	namePatterns []*regexp.Regexp
}

func NewDeviceValidator() *DeviceValidator {
	return &DeviceValidator{
		allowedTypes: map[string]bool{
			"ata": true, "scsi": true, "nvme": true, "sat": true,
			"usbcypress": true, "usbjmicron": true, "usbsunplus": true,
		},
		namePatterns: []*regexp.Regexp{
			regexp.MustCompile(`^/dev/(sd[a-z]|sd[a-z][a-z]|nvme\d+n\d+|hd[a-z]|dm-\d+)$`),
		},
	}
}

func (v *DeviceValidator) Validate(device Device) error {
	// 验证设备类型
	if !v.allowedTypes[device.Type] {
		return fmt.Errorf("invalid device type: %s", device.Type)
	}

	// 验证设备名称
	valid := false
	for _, pattern := range v.namePatterns {
		if pattern.MatchString(device.Name) {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid device name: %s", device.Name)
	}

	return nil
}

// 在全局初始化
var deviceValidator = NewDeviceValidator()

// Get json from smartctl and parse it
func readSMARTctl(logger *slog.Logger, device Device, wg *sync.WaitGroup) {
	defer wg.Done()
	start := time.Now()
	if err := deviceValidator.Validate(device); err != nil {
		logger.Warn("Device validation failed", "err", err, "device", device)
		return
	}
	var smartctlArgs = []string{
		"--json", "--info", "--health", "--attributes",
		"--tolerance=verypermissive",
		"--nocheck=" + smartctlPowerModeCheck,
		"--format=brief",
		"--log=error",
		"--device=" + device.Type,
		device.Name,
	}

	logger.Debug("Calling smartctl with args", "args", strings.Join(smartctlArgs, " "))
	out, err := utils.RunCommand(smartctlPath, smartctlArgs...)
	if err != nil {
		logger.Warn("S.M.A.R.T. output reading", "err", err, "device", device)
	}
	// Accommodate a smartmontools pre-7.3 bug
	cleaned_out := strings.TrimPrefix(string(out), "  Pending defect count:")
	json := parseJSON(cleaned_out)
	rcOk := resultCodeIsOk(logger, device, json.Get("smartctl.exit_status").Int())
	jsonOk := jsonIsOk(logger, json)
	logger.Debug("Collected S.M.A.R.T. json data", "device", device, "duration", time.Since(start))
	if rcOk && jsonOk {
		jsonCache.Store(device, JSONCache{JSON: json, LastCollect: time.Now()})
	}
}

func readSMARTctlDevices(logger *slog.Logger) gjson.Result {
	logger.Debug("Scanning for devices")
	var scanArgs []string = []string{"--json", "--scan"}
	for _, d := range smartctlScanDeviceTypes {
		scanArgs = append(scanArgs, "--device", d)
	}
	out, err := exec.Command(smartctlPath, scanArgs...).Output()
	if exiterr, ok := err.(*exec.ExitError); ok {
		logger.Debug("Exit Status", "exit_code", exiterr.ExitCode())
		if exiterr.ExitCode() != 2 {
			logger.Warn("S.M.A.R.T. output reading error", "err", err)
			return gjson.Result{}
		}
	} else if err != nil {
		logger.Warn("S.M.A.R.T. output reading error", "err", err)
		return gjson.Result{}
	}
	return parseJSON(string(out))
}

// Refresh all devices' json
func refreshAllDevices(logger *slog.Logger, devices []Device) {
	if smartctlFakeData {
		return
	}

	var wg sync.WaitGroup
	for _, device := range devices {
		cacheValue, cacheOk := jsonCache.Load(device)
		if !cacheOk || time.Now().After(cacheValue.(JSONCache).LastCollect.Add(smartctlInterval)) {
			wg.Add(1)
			go readSMARTctl(logger, device, &wg)
		}
	}
	wg.Wait()
}

func readData(logger *slog.Logger, device Device) gjson.Result {
	if smartctlFakeData {
		return readFakeSMARTctl(logger, device)
	}

	cacheValue, found := jsonCache.Load(device)
	if !found {
		logger.Warn("device not found", "device", device)
		return gjson.Result{}
	}
	return cacheValue.(JSONCache).JSON
}

// Reading fake smartctl json
func readFakeSMARTctl(logger *slog.Logger, device Device) gjson.Result {
	s := strings.Split(device.Name, "/")
	filename := fmt.Sprintf("debug/%s.json", s[len(s)-1])
	logger.Debug("Read fake S.M.A.R.T. data from json", "filename", filename)
	cleanFile := filepath.Clean(filename)
	jsonFile, err := os.ReadFile(cleanFile)
	if err != nil {
		logger.Error("Fake S.M.A.R.T. data reading error", "err", err)
		return parseJSON("{}")
	}
	return parseJSON(string(jsonFile))
}

// Parse smartctl return code
func resultCodeIsOk(logger *slog.Logger, device Device, SMARTCtlResult int64) bool {
	result := true
	if SMARTCtlResult > 0 {
		b := SMARTCtlResult
		if (b & 1) != 0 {
			logger.Error("Command line did not parse", "device", device)
			result = false
		}
		if (b & (1 << 1)) != 0 {
			logger.Error("Device open failed, device did not return an IDENTIFY DEVICE structure, or device is in a low-power mode", "device", device)
			result = false
		}
		if (b & (1 << 2)) != 0 {
			logger.Warn("Some SMART or other ATA command to the disk failed, or there was a checksum error in a SMART data structure", "device", device)
		}
		if (b & (1 << 3)) != 0 {
			logger.Warn("SMART status check returned 'DISK FAILING'", "device", device)
		}
		if (b & (1 << 4)) != 0 {
			logger.Warn("We found prefail Attributes <= threshold", "device", device)
		}
		if (b & (1 << 5)) != 0 {
			logger.Warn("SMART status check returned 'DISK OK' but we found that some (usage or prefail) Attributes have been <= threshold at some time in the past", "device", device)
		}
		if (b & (1 << 6)) != 0 {
			logger.Warn("The device error log contains records of errors", "device", device)
		}
		if (b & (1 << 7)) != 0 {
			logger.Warn("The device self-test log contains records of errors. [ATA only] Failed self-tests outdated by a newer successful extended self-test are ignored", "device", device)
		}
	}
	return result
}

// Check json
func jsonIsOk(logger *slog.Logger, json gjson.Result) bool {
	messages := json.Get("smartctl.messages")
	if messages.Exists() {
		for _, message := range messages.Array() {
			if message.Get("severity").String() == "error" {
				logger.Error(message.Get("string").String())
				return false
			}
		}
	}
	return true
}

// SmartctlVersion metric
type SmartctlVersion struct {
	*baseMetrics
}

func NewSmartctlVersion() *SmartctlVersion {
	return &SmartctlVersion{
		NewMetrics("smartctl_version", "smartctl version", []string{
			"json_format_version",
			"smartctl_version",
			"svn_revision",
			"build_info",
		}),
	}
}

func (m *SmartctlVersion) Collect(ch chan<- prometheus.Metric) {
	// This will be collected once per device, but we only want it once
	// Implementation will be handled by the main collector
}

// SmartctlDevice metric
type SmartctlDevice struct {
	*baseMetrics
}

func NewSmartctlDevice() *SmartctlDevice {
	return &SmartctlDevice{
		NewMetrics("smartctl_device", "Device info", []string{
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
		}),
	}
}

func (m *SmartctlDevice) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlDeviceCount metric
type SmartctlDeviceCount struct {
	*baseMetrics
}

func NewSmartctlDeviceCount() *SmartctlDeviceCount {
	return &SmartctlDeviceCount{
		NewMetrics("smartctl_devices", "Number of devices configured or dynamically discovered", []string{}),
	}
}

func (m *SmartctlDeviceCount) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlDeviceCapacityBlocks metric
type SmartctlDeviceCapacityBlocks struct {
	*baseMetrics
}

func NewSmartctlDeviceCapacityBlocks() *SmartctlDeviceCapacityBlocks {
	return &SmartctlDeviceCapacityBlocks{
		NewMetrics("smartctl_device_capacity_blocks", "Device capacity in blocks", []string{"device"}),
	}
}

func (m *SmartctlDeviceCapacityBlocks) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlDeviceCapacityBytes metric
type SmartctlDeviceCapacityBytes struct {
	*baseMetrics
}

func NewSmartctlDeviceCapacityBytes() *SmartctlDeviceCapacityBytes {
	return &SmartctlDeviceCapacityBytes{
		NewMetrics("smartctl_device_capacity_bytes", "Device capacity in bytes", []string{"device"}),
	}
}

func (m *SmartctlDeviceCapacityBytes) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlDeviceTotalCapacityBytes metric
type SmartctlDeviceTotalCapacityBytes struct {
	*baseMetrics
}

func NewSmartctlDeviceTotalCapacityBytes() *SmartctlDeviceTotalCapacityBytes {
	return &SmartctlDeviceTotalCapacityBytes{
		NewMetrics("smartctl_device_nvme_capacity_bytes", "NVMe device total capacity bytes", []string{"device"}),
	}
}

func (m *SmartctlDeviceTotalCapacityBytes) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlDeviceBlockSize metric
type SmartctlDeviceBlockSize struct {
	*baseMetrics
}

func NewSmartctlDeviceBlockSize() *SmartctlDeviceBlockSize {
	return &SmartctlDeviceBlockSize{
		NewMetrics("smartctl_device_block_size", "Device block size", []string{"device", "blocks_type"}),
	}
}

func (m *SmartctlDeviceBlockSize) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlDeviceInterfaceSpeed metric
type SmartctlDeviceInterfaceSpeed struct {
	*baseMetrics
}

func NewSmartctlDeviceInterfaceSpeed() *SmartctlDeviceInterfaceSpeed {
	return &SmartctlDeviceInterfaceSpeed{
		NewMetrics("smartctl_device_interface_speed", "Device interface speed, bits per second", []string{"device", "speed_type"}),
	}
}

func (m *SmartctlDeviceInterfaceSpeed) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlDeviceAttribute metric
type SmartctlDeviceAttribute struct {
	*baseMetrics
}

func NewSmartctlDeviceAttribute() *SmartctlDeviceAttribute {
	return &SmartctlDeviceAttribute{
		NewMetrics("smartctl_device_attribute", "Device attributes", []string{
			"device",
			"attribute_name",
			"attribute_flags_short",
			"attribute_flags_long",
			"attribute_value_type",
			"attribute_id",
		}),
	}
}

func (m *SmartctlDeviceAttribute) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlDevicePowerOnSeconds metric
type SmartctlDevicePowerOnSeconds struct {
	*baseMetrics
}

func NewSmartctlDevicePowerOnSeconds() *SmartctlDevicePowerOnSeconds {
	return &SmartctlDevicePowerOnSeconds{
		NewMetrics("smartctl_device_power_on_seconds", "Device power on seconds", []string{"device"}),
	}
}

func (m *SmartctlDevicePowerOnSeconds) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlDeviceRotationRate metric
type SmartctlDeviceRotationRate struct {
	*baseMetrics
}

func NewSmartctlDeviceRotationRate() *SmartctlDeviceRotationRate {
	return &SmartctlDeviceRotationRate{
		NewMetrics("smartctl_device_rotation_rate", "Device rotation rate", []string{"device"}),
	}
}

func (m *SmartctlDeviceRotationRate) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlDeviceTemperature metric
type SmartctlDeviceTemperature struct {
	*baseMetrics
}

func NewSmartctlDeviceTemperature() *SmartctlDeviceTemperature {
	return &SmartctlDeviceTemperature{
		NewMetrics("smartctl_device_temperature", "Device temperature celsius", []string{"device", "temperature_type"}),
	}
}

func (m *SmartctlDeviceTemperature) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlDevicePowerCycleCount metric
type SmartctlDevicePowerCycleCount struct {
	*baseMetrics
}

func NewSmartctlDevicePowerCycleCount() *SmartctlDevicePowerCycleCount {
	return &SmartctlDevicePowerCycleCount{
		NewMetrics("smartctl_device_power_cycle_count", "Device power cycle count", []string{"device"}),
	}
}

func (m *SmartctlDevicePowerCycleCount) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlDevicePercentageUsed metric
type SmartctlDevicePercentageUsed struct {
	*baseMetrics
}

func NewSmartctlDevicePercentageUsed() *SmartctlDevicePercentageUsed {
	return &SmartctlDevicePercentageUsed{
		NewMetrics("smartctl_device_percentage_used", "Device write percentage used", []string{"device"}),
	}
}

func (m *SmartctlDevicePercentageUsed) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlDeviceAvailableSpare metric
type SmartctlDeviceAvailableSpare struct {
	*baseMetrics
}

func NewSmartctlDeviceAvailableSpare() *SmartctlDeviceAvailableSpare {
	return &SmartctlDeviceAvailableSpare{
		NewMetrics("smartctl_device_available_spare", "Normalized percentage (0 to 100%) of the remaining spare capacity available", []string{"device"}),
	}
}

func (m *SmartctlDeviceAvailableSpare) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlDeviceAvailableSpareThreshold metric
type SmartctlDeviceAvailableSpareThreshold struct {
	*baseMetrics
}

func NewSmartctlDeviceAvailableSpareThreshold() *SmartctlDeviceAvailableSpareThreshold {
	return &SmartctlDeviceAvailableSpareThreshold{
		NewMetrics("smartctl_device_available_spare_threshold", "When the Available Spare falls below the threshold indicated in this field, an asynchronous event completion may occur. The value is indicated as a normalized percentage (0 to 100%)", []string{"device"}),
	}
}

func (m *SmartctlDeviceAvailableSpareThreshold) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlDeviceCriticalWarning metric
type SmartctlDeviceCriticalWarning struct {
	*baseMetrics
}

func NewSmartctlDeviceCriticalWarning() *SmartctlDeviceCriticalWarning {
	return &SmartctlDeviceCriticalWarning{
		NewMetrics("smartctl_device_critical_warning", "This field indicates critical warnings for the state of the controller", []string{"device"}),
	}
}

func (m *SmartctlDeviceCriticalWarning) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlDeviceMediaErrors metric
type SmartctlDeviceMediaErrors struct {
	*baseMetrics
}

func NewSmartctlDeviceMediaErrors() *SmartctlDeviceMediaErrors {
	return &SmartctlDeviceMediaErrors{
		NewMetrics("smartctl_device_media_errors", "Contains the number of occurrences where the controller detected an unrecovered data integrity error. Errors such as uncorrectable ECC, CRC checksum failure, or LBA tag mismatch are included in this field", []string{"device"}),
	}
}

func (m *SmartctlDeviceMediaErrors) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlDeviceNumErrLogEntries metric
type SmartctlDeviceNumErrLogEntries struct {
	*baseMetrics
}

func NewSmartctlDeviceNumErrLogEntries() *SmartctlDeviceNumErrLogEntries {
	return &SmartctlDeviceNumErrLogEntries{
		NewMetrics("smartctl_device_num_err_log_entries", "Contains the number of Error Information log entries over the life of the controller", []string{"device"}),
	}
}

func (m *SmartctlDeviceNumErrLogEntries) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlDeviceBytesRead metric
type SmartctlDeviceBytesRead struct {
	*baseMetrics
}

func NewSmartctlDeviceBytesRead() *SmartctlDeviceBytesRead {
	return &SmartctlDeviceBytesRead{
		NewMetrics("smartctl_device_bytes_read", "", []string{"device"}),
	}
}

func (m *SmartctlDeviceBytesRead) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlDeviceBytesWritten metric
type SmartctlDeviceBytesWritten struct {
	*baseMetrics
}

func NewSmartctlDeviceBytesWritten() *SmartctlDeviceBytesWritten {
	return &SmartctlDeviceBytesWritten{
		NewMetrics("smartctl_device_bytes_written", "", []string{"device"}),
	}
}

func (m *SmartctlDeviceBytesWritten) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlDeviceSmartStatus metric
type SmartctlDeviceSmartStatus struct {
	*baseMetrics
}

func NewSmartctlDeviceSmartStatus() *SmartctlDeviceSmartStatus {
	return &SmartctlDeviceSmartStatus{
		NewMetrics("smartctl_device_smart_status", "General smart status", []string{"device"}),
	}
}

func (m *SmartctlDeviceSmartStatus) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlDeviceExitStatus metric
type SmartctlDeviceExitStatus struct {
	*baseMetrics
}

func NewSmartctlDeviceExitStatus() *SmartctlDeviceExitStatus {
	return &SmartctlDeviceExitStatus{
		NewMetrics("smartctl_device_smartctl_exit_status", "Exit status of smartctl on device", []string{"device"}),
	}
}

func (m *SmartctlDeviceExitStatus) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlDeviceState metric
type SmartctlDeviceState struct {
	*baseMetrics
}

func NewSmartctlDeviceState() *SmartctlDeviceState {
	return &SmartctlDeviceState{
		NewMetrics("smartctl_device_state", "Device state (0=active, 1=standby, 2=sleep, 3=dst, 4=offline, 5=sct)", []string{"device"}),
	}
}

func (m *SmartctlDeviceState) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlDeviceStatistics metric
type SmartctlDeviceStatistics struct {
	*baseMetrics
}

func NewSmartctlDeviceStatistics() *SmartctlDeviceStatistics {
	return &SmartctlDeviceStatistics{
		NewMetrics("smartctl_device_statistics", "Device statistics", []string{
			"device",
			"statistic_table",
			"statistic_name",
			"statistic_flags_short",
			"statistic_flags_long",
		}),
	}
}

func (m *SmartctlDeviceStatistics) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlDeviceErrorLogCount metric
type SmartctlDeviceErrorLogCount struct {
	*baseMetrics
}

func NewSmartctlDeviceErrorLogCount() *SmartctlDeviceErrorLogCount {
	return &SmartctlDeviceErrorLogCount{
		NewMetrics("smartctl_device_error_log_count", "Device SMART error log count", []string{"device", "error_log_type"}),
	}
}

func (m *SmartctlDeviceErrorLogCount) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlDeviceSelfTestLogCount metric
type SmartctlDeviceSelfTestLogCount struct {
	*baseMetrics
}

func NewSmartctlDeviceSelfTestLogCount() *SmartctlDeviceSelfTestLogCount {
	return &SmartctlDeviceSelfTestLogCount{
		NewMetrics("smartctl_device_self_test_log_count", "Device SMART self test log count", []string{"device", "self_test_log_type"}),
	}
}

func (m *SmartctlDeviceSelfTestLogCount) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlDeviceSelfTestLogErrorCount metric
type SmartctlDeviceSelfTestLogErrorCount struct {
	*baseMetrics
}

func NewSmartctlDeviceSelfTestLogErrorCount() *SmartctlDeviceSelfTestLogErrorCount {
	return &SmartctlDeviceSelfTestLogErrorCount{
		NewMetrics("smartctl_device_self_test_log_error_count", "Device SMART self test log error count", []string{"device", "self_test_log_type"}),
	}
}

func (m *SmartctlDeviceSelfTestLogErrorCount) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlDeviceERCSeconds metric
type SmartctlDeviceERCSeconds struct {
	*baseMetrics
}

func NewSmartctlDeviceERCSeconds() *SmartctlDeviceERCSeconds {
	return &SmartctlDeviceERCSeconds{
		NewMetrics("smartctl_device_erc_seconds", "Device SMART Error Recovery Control Seconds", []string{"device", "op_type"}),
	}
}

func (m *SmartctlDeviceERCSeconds) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlSCSIGrownDefectList metric
type SmartctlSCSIGrownDefectList struct {
	*baseMetrics
}

func NewSmartctlSCSIGrownDefectList() *SmartctlSCSIGrownDefectList {
	return &SmartctlSCSIGrownDefectList{
		NewMetrics("smartctl_scsi_grown_defect_list", "Device SCSI grown defect list counter", []string{"device"}),
	}
}

func (m *SmartctlSCSIGrownDefectList) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlReadErrorsCorrectedByRereadsRewrites metric
type SmartctlReadErrorsCorrectedByRereadsRewrites struct {
	*baseMetrics
}

func NewSmartctlReadErrorsCorrectedByRereadsRewrites() *SmartctlReadErrorsCorrectedByRereadsRewrites {
	return &SmartctlReadErrorsCorrectedByRereadsRewrites{
		NewMetrics("smartctl_read_errors_corrected_by_rereads_rewrites", "Read Errors Corrected by ReReads/ReWrites", []string{"device"}),
	}
}

func (m *SmartctlReadErrorsCorrectedByRereadsRewrites) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlReadErrorsCorrectedByEccFast metric
type SmartctlReadErrorsCorrectedByEccFast struct {
	*baseMetrics
}

func NewSmartctlReadErrorsCorrectedByEccFast() *SmartctlReadErrorsCorrectedByEccFast {
	return &SmartctlReadErrorsCorrectedByEccFast{
		NewMetrics("smartctl_read_errors_corrected_by_eccfast", "Read Errors Corrected by ECC Fast", []string{"device"}),
	}
}

func (m *SmartctlReadErrorsCorrectedByEccFast) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlReadErrorsCorrectedByEccDelayed metric
type SmartctlReadErrorsCorrectedByEccDelayed struct {
	*baseMetrics
}

func NewSmartctlReadErrorsCorrectedByEccDelayed() *SmartctlReadErrorsCorrectedByEccDelayed {
	return &SmartctlReadErrorsCorrectedByEccDelayed{
		NewMetrics("smartctl_read_errors_corrected_by_eccdelayed", "Read Errors Corrected by ECC Delayed", []string{"device"}),
	}
}

func (m *SmartctlReadErrorsCorrectedByEccDelayed) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlReadTotalUncorrectedErrors metric
type SmartctlReadTotalUncorrectedErrors struct {
	*baseMetrics
}

func NewSmartctlReadTotalUncorrectedErrors() *SmartctlReadTotalUncorrectedErrors {
	return &SmartctlReadTotalUncorrectedErrors{
		NewMetrics("smartctl_read_total_uncorrected_errors", "Read Total Uncorrected Errors", []string{"device"}),
	}
}

func (m *SmartctlReadTotalUncorrectedErrors) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlWriteErrorsCorrectedByRereadsRewrites metric
type SmartctlWriteErrorsCorrectedByRereadsRewrites struct {
	*baseMetrics
}

func NewSmartctlWriteErrorsCorrectedByRereadsRewrites() *SmartctlWriteErrorsCorrectedByRereadsRewrites {
	return &SmartctlWriteErrorsCorrectedByRereadsRewrites{
		NewMetrics("smartctl_write_errors_corrected_by_rereads_rewrites", "Write Errors Corrected by ReReads/ReWrites", []string{"device"}),
	}
}

func (m *SmartctlWriteErrorsCorrectedByRereadsRewrites) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlWriteErrorsCorrectedByEccFast metric
type SmartctlWriteErrorsCorrectedByEccFast struct {
	*baseMetrics
}

func NewSmartctlWriteErrorsCorrectedByEccFast() *SmartctlWriteErrorsCorrectedByEccFast {
	return &SmartctlWriteErrorsCorrectedByEccFast{
		NewMetrics("smartctl_write_errors_corrected_by_eccfast", "Write Errors Corrected by ECC Fast", []string{"device"}),
	}
}

func (m *SmartctlWriteErrorsCorrectedByEccFast) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlWriteErrorsCorrectedByEccDelayed metric
type SmartctlWriteErrorsCorrectedByEccDelayed struct {
	*baseMetrics
}

func NewSmartctlWriteErrorsCorrectedByEccDelayed() *SmartctlWriteErrorsCorrectedByEccDelayed {
	return &SmartctlWriteErrorsCorrectedByEccDelayed{
		NewMetrics("smartctl_write_errors_corrected_by_eccdelayed", "Write Errors Corrected by ECC Delayed", []string{"device"}),
	}
}

func (m *SmartctlWriteErrorsCorrectedByEccDelayed) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

// SmartctlWriteTotalUncorrectedErrors metric
type SmartctlWriteTotalUncorrectedErrors struct {
	*baseMetrics
}

func NewSmartctlWriteTotalUncorrectedErrors() *SmartctlWriteTotalUncorrectedErrors {
	return &SmartctlWriteTotalUncorrectedErrors{
		NewMetrics("smartctl_write_total_uncorrected_errors", "Write Total Uncorrected Errors", []string{"device"}),
	}
}

func (m *SmartctlWriteTotalUncorrectedErrors) Collect(ch chan<- prometheus.Metric) {
	// Implementation will be handled by the main collector
}

package metrics

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSmartctlVersion(t *testing.T) {
	collector := NewSmartctlVersion()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlVersion{}, collector)
}

func TestNewSmartctlDevice(t *testing.T) {
	collector := NewSmartctlDevice()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlDevice{}, collector)
}

func TestNewSmartctlDeviceCount(t *testing.T) {
	collector := NewSmartctlDeviceCount()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlDeviceCount{}, collector)
}

func TestNewSmartctlDeviceCapacityBlocks(t *testing.T) {
	collector := NewSmartctlDeviceCapacityBlocks()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlDeviceCapacityBlocks{}, collector)
}

func TestNewSmartctlDeviceCapacityBytes(t *testing.T) {
	collector := NewSmartctlDeviceCapacityBytes()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlDeviceCapacityBytes{}, collector)
}

func TestNewSmartctlDeviceTotalCapacityBytes(t *testing.T) {
	collector := NewSmartctlDeviceTotalCapacityBytes()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlDeviceTotalCapacityBytes{}, collector)
}

func TestNewSmartctlDeviceBlockSize(t *testing.T) {
	collector := NewSmartctlDeviceBlockSize()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlDeviceBlockSize{}, collector)
}

func TestNewSmartctlDeviceInterfaceSpeed(t *testing.T) {
	collector := NewSmartctlDeviceInterfaceSpeed()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlDeviceInterfaceSpeed{}, collector)
}

func TestNewSmartctlDeviceAttribute(t *testing.T) {
	collector := NewSmartctlDeviceAttribute()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlDeviceAttribute{}, collector)
}

func TestNewSmartctlDevicePowerOnSeconds(t *testing.T) {
	collector := NewSmartctlDevicePowerOnSeconds()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlDevicePowerOnSeconds{}, collector)
}

func TestNewSmartctlDeviceRotationRate(t *testing.T) {
	collector := NewSmartctlDeviceRotationRate()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlDeviceRotationRate{}, collector)
}

func TestNewSmartctlDeviceTemperature(t *testing.T) {
	collector := NewSmartctlDeviceTemperature()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlDeviceTemperature{}, collector)
}

func TestNewSmartctlDevicePowerCycleCount(t *testing.T) {
	collector := NewSmartctlDevicePowerCycleCount()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlDevicePowerCycleCount{}, collector)
}

func TestNewSmartctlDevicePercentageUsed(t *testing.T) {
	collector := NewSmartctlDevicePercentageUsed()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlDevicePercentageUsed{}, collector)
}

func TestNewSmartctlDeviceAvailableSpare(t *testing.T) {
	collector := NewSmartctlDeviceAvailableSpare()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlDeviceAvailableSpare{}, collector)
}

func TestNewSmartctlDeviceAvailableSpareThreshold(t *testing.T) {
	collector := NewSmartctlDeviceAvailableSpareThreshold()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlDeviceAvailableSpareThreshold{}, collector)
}

func TestNewSmartctlDeviceCriticalWarning(t *testing.T) {
	collector := NewSmartctlDeviceCriticalWarning()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlDeviceCriticalWarning{}, collector)
}

func TestNewSmartctlDeviceMediaErrors(t *testing.T) {
	collector := NewSmartctlDeviceMediaErrors()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlDeviceMediaErrors{}, collector)
}

func TestNewSmartctlDeviceNumErrLogEntries(t *testing.T) {
	collector := NewSmartctlDeviceNumErrLogEntries()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlDeviceNumErrLogEntries{}, collector)
}

func TestNewSmartctlDeviceBytesRead(t *testing.T) {
	collector := NewSmartctlDeviceBytesRead()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlDeviceBytesRead{}, collector)
}

func TestNewSmartctlDeviceBytesWritten(t *testing.T) {
	collector := NewSmartctlDeviceBytesWritten()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlDeviceBytesWritten{}, collector)
}

func TestNewSmartctlDeviceSmartStatus(t *testing.T) {
	collector := NewSmartctlDeviceSmartStatus()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlDeviceSmartStatus{}, collector)
}

func TestNewSmartctlDeviceExitStatus(t *testing.T) {
	collector := NewSmartctlDeviceExitStatus()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlDeviceExitStatus{}, collector)
}

func TestNewSmartctlDeviceState(t *testing.T) {
	collector := NewSmartctlDeviceState()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlDeviceState{}, collector)
}

func TestNewSmartctlDeviceStatistics(t *testing.T) {
	collector := NewSmartctlDeviceStatistics()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlDeviceStatistics{}, collector)
}

func TestNewSmartctlDeviceErrorLogCount(t *testing.T) {
	collector := NewSmartctlDeviceErrorLogCount()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlDeviceErrorLogCount{}, collector)
}

func TestNewSmartctlDeviceSelfTestLogCount(t *testing.T) {
	collector := NewSmartctlDeviceSelfTestLogCount()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlDeviceSelfTestLogCount{}, collector)
}

func TestNewSmartctlDeviceSelfTestLogErrorCount(t *testing.T) {
	collector := NewSmartctlDeviceSelfTestLogErrorCount()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlDeviceSelfTestLogErrorCount{}, collector)
}

func TestNewSmartctlDeviceERCSeconds(t *testing.T) {
	collector := NewSmartctlDeviceERCSeconds()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlDeviceERCSeconds{}, collector)
}

func TestNewSmartctlSCSIGrownDefectList(t *testing.T) {
	collector := NewSmartctlSCSIGrownDefectList()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlSCSIGrownDefectList{}, collector)
}

func TestNewSmartctlReadErrorsCorrectedByRereadsRewrites(t *testing.T) {
	collector := NewSmartctlReadErrorsCorrectedByRereadsRewrites()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlReadErrorsCorrectedByRereadsRewrites{}, collector)
}

func TestNewSmartctlReadErrorsCorrectedByEccFast(t *testing.T) {
	collector := NewSmartctlReadErrorsCorrectedByEccFast()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlReadErrorsCorrectedByEccFast{}, collector)
}

func TestNewSmartctlReadErrorsCorrectedByEccDelayed(t *testing.T) {
	collector := NewSmartctlReadErrorsCorrectedByEccDelayed()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlReadErrorsCorrectedByEccDelayed{}, collector)
}

func TestNewSmartctlReadTotalUncorrectedErrors(t *testing.T) {
	collector := NewSmartctlReadTotalUncorrectedErrors()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlReadTotalUncorrectedErrors{}, collector)
}

func TestNewSmartctlWriteErrorsCorrectedByRereadsRewrites(t *testing.T) {
	collector := NewSmartctlWriteErrorsCorrectedByRereadsRewrites()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlWriteErrorsCorrectedByRereadsRewrites{}, collector)
}

func TestNewSmartctlWriteErrorsCorrectedByEccFast(t *testing.T) {
	collector := NewSmartctlWriteErrorsCorrectedByEccFast()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlWriteErrorsCorrectedByEccFast{}, collector)
}

func TestNewSmartctlWriteErrorsCorrectedByEccDelayed(t *testing.T) {
	collector := NewSmartctlWriteErrorsCorrectedByEccDelayed()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlWriteErrorsCorrectedByEccDelayed{}, collector)
}

func TestNewSmartctlWriteTotalUncorrectedErrors(t *testing.T) {
	collector := NewSmartctlWriteTotalUncorrectedErrors()
	
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.baseMetrics)
	assert.IsType(t, &SmartctlWriteTotalUncorrectedErrors{}, collector)
}

// Test basic functionality
func TestCollectorBasics(t *testing.T) {
	collectors := []interface{}{
		NewSmartctlVersion(),
		NewSmartctlDevice(),
		NewSmartctlDeviceCount(),
		NewSmartctlDeviceAttribute(),
	}

	for i, collector := range collectors {
		t.Run(fmt.Sprintf("collector_%d", i), func(t *testing.T) {
			assert.NotNil(t, collector)
		})
	}
}

// Benchmark tests
func BenchmarkSmartctlDeviceCountCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewSmartctlDeviceCount()
	}
}

func BenchmarkSmartctlVersionCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewSmartctlVersion()
	}
}

func BenchmarkSmartctlDeviceCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewSmartctlDevice()
	}
} 
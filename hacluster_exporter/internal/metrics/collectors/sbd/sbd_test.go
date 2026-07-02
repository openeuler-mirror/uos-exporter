package sbd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ------- 1. 配置文件读取错误 --------
const test_Root_Dir = "../../../../test/"

func TestReadSbdConfFileError(t *testing.T) {
	invalidPath := filepath.Join(test_Root_Dir, "nonexistent_sbdconfig")
	sbdConfFile, err := readSdbFile(invalidPath)
	assert.Nil(t, sbdConfFile)
	assert.Error(t, err)
}

// ------- 2. SBD_DEVICE 字段各种格式 --------

func TestGetSbdDevicesWithoutDoubleQuotes(t *testing.T) {
	sbdConfig := `
SBD_DEVICE=/dev/vda;/dev/vdb;/dev/vdc
`
	sbdDevices := getSbdDevices([]byte(sbdConfig))
	assert.Len(t, sbdDevices, 3)
	assert.Equal(t, "/dev/vda", sbdDevices[0])
	assert.Equal(t, "/dev/vdb", sbdDevices[1])
	assert.Equal(t, "/dev/vdc", sbdDevices[2])
}

func TestGetSbdDevicesWithDoubleQuotes(t *testing.T) {
	sbdConfig := `SBD_DEVICE="/dev/vda;/dev/vdb;/dev/vdc"`
	sbdDevices := getSbdDevices([]byte(sbdConfig))
	assert.Len(t, sbdDevices, 3)
	assert.Equal(t, "/dev/vda", sbdDevices[0])
	assert.Equal(t, "/dev/vdb", sbdDevices[1])
	assert.Equal(t, "/dev/vdc", sbdDevices[2])
}

func TestOnlyOneDeviceSbd(t *testing.T) {
	sbdConfig := `SBD_DEVICE=/dev/vdc`
	sbdDevices := getSbdDevices([]byte(sbdConfig))
	assert.Len(t, sbdDevices, 1)
	assert.Equal(t, "/dev/vdc", sbdDevices[0])
}

func TestSbdDeviceParserWithSpaceAfterSemicolon(t *testing.T) {
	sbdConfig := `SBD_DEVICE=/dev/vdc; /dev/vdd`
	sbdDevices := getSbdDevices([]byte(sbdConfig))
	assert.Len(t, sbdDevices, 2)
	assert.Equal(t, "/dev/vdc", sbdDevices[0])
	assert.Equal(t, "/dev/vdd", sbdDevices[1])
}

func TestSbdDeviceParserWithSemicolonAtEnd(t *testing.T) {
	sbdConfig := `SBD_DEVICE=/dev/vdc;/dev/vdd;`
	sbdDevices := getSbdDevices([]byte(sbdConfig))
	assert.Len(t, sbdDevices, 2)
	assert.Equal(t, "/dev/vdc", sbdDevices[0])
	assert.Equal(t, "/dev/vdd", sbdDevices[1])
}

// // ------- 3. Collector 构造参数校验 --------

func TestNewSBDCollectorChecksSBDExistenceAndExecutableBits(t *testing.T) {
	// 不存在
	nonExistPath := filepath.Join(test_Root_Dir, "nonexistent")
	_, err := NewCollector(
		nonExistPath,
		filepath.Join(test_Root_Dir, "fake_sdbconfig"),
		false,
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "'"+nonExistPath+"' does not exist")

	// 不可执行
	fileName := filepath.Join(test_Root_Dir, "dummy")
	os.WriteFile(fileName, []byte("dummy"), 0644)
	defer os.Remove(fileName)
	_, err = NewCollector(
		fileName,
		filepath.Join(test_Root_Dir, "fake_sdbconfig"),
		false,
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "'"+fileName+"' is not executable")
}

func TestNewSBDCollectorChecksConfigExistence(t *testing.T) {
	invalidConfigPath := filepath.Join(test_Root_Dir, "nonexistent_config_file")
	_, err := NewCollector(
		filepath.Join(test_Root_Dir, "fake_sbd.sh"),
		invalidConfigPath,
		false,
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "'"+invalidConfigPath+"' does not exist")
}

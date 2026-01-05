package lxc

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Mock 结构体
type MockLxc struct {
	existingContainers map[string]string // 模拟容器名到 cpu.stat 内容的映射
}

// Mock 方法：检查容器是否存在
func (m *MockLxc) containerExists(containerName string) bool {
	_, exists := m.existingContainers[containerName]
	return exists
}

// Mock 方法：读取 cpu.stat 文件
func (m *MockLxc) readCPUStatFile(containerName string) ([]byte, error) {
	if content, exists := m.existingContainers[containerName]; exists {
		return []byte(content), nil
	}
	return nil, os.ErrNotExist
}

func (m *MockLxc) GetCPUStat(containerName string) (CPUStat, error) {
	if !m.containerExists(containerName) {
		return CPUStat{}, ErrorContainerNotFound
	}
	statContent, err := m.readCPUStatFile(containerName)
	if err != nil {
		return CPUStat{}, err
	}
	return parseCPUStat(statContent)
}

// 测试 `GetCPUStat`
func TestGetCPUStat(t *testing.T) {
	mockLxc := &MockLxc{
		existingContainers: map[string]string{
			"container1": "usage_usec 1000000\nuser_usec 500000\nsystem_usec 300000",
		},
	}

	// 测试正常情况
	cpuStat, err := mockLxc.GetCPUStat("container1")
	assert.NoError(t, err)
	assert.Equal(t, float64(1000000), cpuStat.Usage)
	assert.Equal(t, float64(500000), cpuStat.User)
	assert.Equal(t, float64(300000), cpuStat.System)

	// 测试容器不存在
	_, err = mockLxc.GetCPUStat("nonexistent-container")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrorContainerNotFound))
}

// 测试 `parseCPUStat`
func TestParseCPUStat(t *testing.T) {
	validContent := []byte(`usage_usec 7709
user_usec 1914
system_usec 5794
core_sched.force_idle_usec 0
nr_periods 0
nr_throttled 0
throttled_usec 0
nr_bursts 0
burst_usec 0`)

	// 正确解析
	cpuStat, err := parseCPUStat(validContent)
	assert.NoError(t, err)
	assert.Equal(t, float64(7709), cpuStat.Usage)
	assert.Equal(t, float64(1914), cpuStat.User)
	assert.Equal(t, float64(5794), cpuStat.System)

	// 测试格式错误（缺少字段）
	invalidContent := []byte("usage_usec 2000000\nuser_usec 800000")
	_, err = parseCPUStat(invalidContent)
	assert.Error(t, err)
}

// 测试 `parseCPUStatLine`
func TestParseCPUStatLine(t *testing.T) {
	// 测试正常情况
	value, err := parseCPUStatLine("usage_usec 1234567")
	assert.NoError(t, err)
	assert.Equal(t, float64(1234567), value)

	// 测试格式错误
	_, err = parseCPUStatLine("usage_usec")
	assert.Error(t, err)

	// 测试无效数值
	_, err = parseCPUStatLine("usage_usec abc")
	assert.Error(t, err)
}

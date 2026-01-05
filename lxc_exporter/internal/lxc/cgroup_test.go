package lxc

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseCgroupStat(t *testing.T) {
	validContent := []byte(`nr_descendants 1
nr_dying_descendants 0`)

	// 正确解析
	cgroupStat, err := parseCgroupStat(validContent)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), cgroupStat.NrDescendants)
	assert.Equal(t, float64(0), cgroupStat.NrDyingDescendants)

	// 测试格式错误（缺少字段）
	invalidContent := []byte("nr_descendants 1")
	_, err = parseCgroupStat(invalidContent)
	assert.Error(t, err)
}

func TestParseCgroupStatLine(t *testing.T) {
	// 测试正常情况
	value, err := parseCgroupStatLine("nr_descendants 1")
	assert.NoError(t, err)
	assert.Equal(t, float64(1), value)

	// 测试格式错误
	_, err = parseCgroupStatLine("nr_descendants")
	assert.Error(t, err)

	// 测试无效数值
	_, err = parseCgroupStatLine("usage_usec abc")
	assert.Error(t, err)
}

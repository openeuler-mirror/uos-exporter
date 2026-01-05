package lxc

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseIoStat(t *testing.T) {
	validContent := []byte(`253:0 rbytes=0 wbytes=12288 rios=0 wios=3 dbytes=0 dios=0`)

	// 正确解析
	ioStat, err := parseIoStat(validContent)
	assert.NoError(t, err)
	assert.Equal(t, float64(0), ioStat.Rbytes)
	assert.Equal(t, float64(12288), ioStat.Wbytes)
	assert.Equal(t, float64(0), ioStat.Rios)
	assert.Equal(t, float64(3), ioStat.Wios)
	assert.Equal(t, float64(0), ioStat.Dbytes)
	assert.Equal(t, float64(0), ioStat.Dios)

	// 测试格式错误（缺少字段）
	invalidContent := []byte("253:0 rbytes=0 wbytes=12288")
	_, err = parseIoStat(invalidContent)
	assert.Error(t, err)
}

func TestParseIoFields(t *testing.T) {
	validContent := []byte(`dios=0`)
	_, err := parseIoFields(string(validContent))
	assert.NoError(t, err)
	invalidContent := []byte(`dios=0 dbytes=0`)
	_, err = parseIoFields(string(invalidContent))
	assert.Error(t, err)
}

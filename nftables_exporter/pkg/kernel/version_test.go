package kernel

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetVersion(t *testing.T) {
	// 读取 /proc/version 的内容
	_, err := os.ReadFile("/proc/version")
	assert.NoError(t, err)

	version, err := GetVersion()
	assert.NoError(t, err)
	assert.NotEmpty(t, version)
}

func TestGetMajorVersion(t *testing.T) {
	majorVersion, err := GetMajorVersion()
	assert.NoError(t, err)
	assert.Greater(t, majorVersion, 0)
}

func TestExtractKernelVersion(t *testing.T) {
	testCases := []struct {
		content  string
		expected string
	}{
		{
			"Linux version 5.10.0-rc3",
			"5.10.0",
		},
		{
			"Linux version 4.18.0-305.el8.x86_64",
			"4.18.0",
		},
		{
			"invalid content",
			"",
		},
	}

	for _, tc := range testCases {
		version := extractKernelVersion(
			tc.content)
		assert.Equal(t, tc.expected, version)
	}
}

func TestParseMajorVersion(t *testing.T) {
	testCases := []struct {
		version   string
		expected  int
		shouldErr bool
	}{
		{
			"5.10.0",
			5,
			false,
		},
		{
			"4.18.0",
			4,
			false,
		},
		{
			"invalid",
			0,
			true,
		},
		{
			"",
			0,
			true,
		},
	}

	for _, tc := range testCases {
		major, err := parseMajorVersion(tc.version)
		if tc.shouldErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, major)
		}
	}
}

package system

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 模拟的 JSON 输出
const mockSystemInfoJSON = `{
  "host": {
    "arch": "amd64",
    "buildahVersion": "1.33.7",
    "cgroupManager": "systemd",
    "cgroupVersion": "v1",
    "conmon": {
      "package": "conmon-2.1.10-1.uos25.x86_64",
      "path": "/usr/bin/conmon",
      "version": "conmon version 2.1.10, commit: unknown"
    },
    "cpus": 8,
    "distribution": {
      "distribution": "uos",
      "version": "25",
      "codename": "qianlai"
    },
    "hostname": "v25-sub-tr6",
    "kernel": "6.6.0-25.02.2500.006.uos25.x86_64",
    "ociRuntime": {
      "name": "crun",
      "package": "crun-1.8.7-2.uos25.02.x86_64",
      "path": "/usr/bin/crun",
      "version": "crun version 1.8.7\ncommit: 53a9996ce82d1ee818349bdcc64797a1fa0433c4\nrundir: /run/user/0/crun\nspec: 1.0.0\n+SYSTEMD +SELINUX +APPARMOR +CAP +SECCOMP +EBPF +CRIU +YAJL"
    },
    "os": "linux"
  },
  "version": {
    "APIVersion": "4.9.4",
    "Version": "4.9.4",
    "GoVersion": "go1.22.8",
    "GitCommit": "",
    "BuiltTime": "Thu Jan  1 08:00:00 1970",
    "Built": 0,
    "OsArch": "linux/amd64",
    "Os": "linux"
  }
}`

func TestNewParser(t *testing.T) {
	parser := NewParser()
	assert.NotNil(t, parser)
}

func TestParseJSONOutput(t *testing.T) {
	parser := NewParser()
	status, err := parser.Parse([]byte(mockSystemInfoJSON))

	require.NoError(t, err)
	require.NotNil(t, status)

	// 验证解析的版本信息
	assert.Equal(t, "4.9.4", status.APIVersion)
	assert.Equal(t, "1.33.7", status.BuildahVersion)
	assert.Equal(t, "conmon version 2.1.10, commit: unknown", status.ConmonVersion)
	assert.Contains(t, status.RuntimeVersion, "crun version 1.8.7")
	assert.False(t, status.Timestamp.IsZero())
}

func TestNewCollector(t *testing.T) {
	logger := logrus.New()
	timeout := 30 * time.Second

	collector := NewCollector(logger, timeout)

	assert.NotNil(t, collector)
	assert.Equal(t, logger, collector.logger)
	assert.Equal(t, timeout, collector.timeout)
	assert.NotNil(t, collector.parser)
	assert.NotNil(t, collector.apiVersionDesc)
	assert.NotNil(t, collector.buildahVersionDesc)
	assert.NotNil(t, collector.conmonVersionDesc)
	assert.NotNil(t, collector.runtimeVersionDesc)
}

func TestCollectorDescribe(t *testing.T) {
	logger := logrus.New()
	collector := NewCollector(logger, 30*time.Second)

	ch := make(chan *prometheus.Desc, 10)
	collector.Describe(ch)
	close(ch)

	var descs []*prometheus.Desc
	for desc := range ch {
		descs = append(descs, desc)
	}

	assert.Len(t, descs, 4) // 4个指标描述符
}

func TestCollectorName(t *testing.T) {
	logger := logrus.New()
	collector := NewCollector(logger, 30*time.Second)

	name := collector.Name()
	assert.Equal(t, "podman_system", name)
}

func TestCollectVersionMetrics(t *testing.T) {
	logger := logrus.New()
	collector := NewCollector(logger, 30*time.Second)

	status := &Status{
		APIVersion:     "4.9.4",
		BuildahVersion: "1.33.7",
		ConmonVersion:  "conmon version 2.1.10, commit: unknown",
		RuntimeVersion: "crun version 1.8.7\ncommit: 53a9996ce82d1ee818349bdcc64797a1fa0433c4",
	}

	ch := make(chan prometheus.Metric, 10)
	collector.collectVersionMetrics(ch, status)
	close(ch)

	var metrics []prometheus.Metric
	for metric := range ch {
		metrics = append(metrics, metric)
	}

	assert.Len(t, metrics, 4) // 所有4个版本指标
}

func TestExtractConmonVersion(t *testing.T) {
	collector := &Collector{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "标准格式",
			input:    "conmon version 2.1.10, commit: unknown",
			expected: "2.1.10",
		},
		{
			name:     "无commit信息",
			input:    "conmon version 2.1.0",
			expected: "2.1.0",
		},
		{
			name:     "非标准格式",
			input:    "other format",
			expected: "other format",
		},
		{
			name:     "空字符串",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collector.extractConmonVersion(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractRuntimeVersion(t *testing.T) {
	collector := &Collector{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "多行格式",
			input:    "crun version 1.8.7\ncommit: 53a9996ce82d1ee818349bdcc64797a1fa0433c4\nrundir: /run/user/0/crun",
			expected: "crun version 1.8.7",
		},
		{
			name:     "单行格式",
			input:    "crun version 1.4.5",
			expected: "crun version 1.4.5",
		},
		{
			name:     "空字符串",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collector.extractRuntimeVersion(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParsePartialJSON(t *testing.T) {
	parser := NewParser()

	// 测试只有部分字段的JSON
	partialJSON := `{
		"version": {
			"APIVersion": "4.1.1"
		},
		"host": {
			"buildahVersion": "1.26.1"
		}
	}`

	status, err := parser.Parse([]byte(partialJSON))
	require.NoError(t, err)
	assert.Equal(t, "4.1.1", status.APIVersion)
	assert.Equal(t, "1.26.1", status.BuildahVersion)
	assert.Equal(t, "", status.ConmonVersion)  // 缺失字段应为空
	assert.Equal(t, "", status.RuntimeVersion) // 缺失字段应为空
}

func TestParseInvalidJSON(t *testing.T) {
	parser := NewParser()

	// 测试无效JSON
	_, err := parser.Parse([]byte(`invalid json`))
	assert.Error(t, err)
}

func TestParseEmptyJSON(t *testing.T) {
	parser := NewParser()

	// 测试空JSON对象
	status, err := parser.Parse([]byte(`{}`))
	require.NoError(t, err)
	assert.Equal(t, "", status.APIVersion)
	assert.Equal(t, "", status.BuildahVersion)
	assert.Equal(t, "", status.ConmonVersion)
	assert.Equal(t, "", status.RuntimeVersion)
}

func TestParseCompleteJSON(t *testing.T) {
	parser := NewParser()

	status, err := parser.Parse([]byte(mockSystemInfoJSON))
	require.NoError(t, err)

	// 验证所有字段都被正确解析
	assert.NotEmpty(t, status.APIVersion)
	assert.NotEmpty(t, status.BuildahVersion)
	assert.NotEmpty(t, status.ConmonVersion)
	assert.NotEmpty(t, status.RuntimeVersion)
	assert.False(t, status.Timestamp.IsZero())

	// 验证具体值
	assert.Equal(t, "4.9.4", status.APIVersion)
	assert.Equal(t, "1.33.7", status.BuildahVersion)
	assert.Contains(t, status.ConmonVersion, "conmon version 2.1.10")
	assert.Contains(t, status.RuntimeVersion, "crun version 1.8.7")
}

func TestVersionMetricsWithEmptyFields(t *testing.T) {
	logger := logrus.New()
	collector := NewCollector(logger, 30*time.Second)

	// 测试部分字段为空的情况
	status := &Status{
		APIVersion:     "4.9.4",
		BuildahVersion: "", // 空字段
		ConmonVersion:  "conmon version 2.1.10",
		RuntimeVersion: "", // 空字段
	}

	ch := make(chan prometheus.Metric, 10)
	collector.collectVersionMetrics(ch, status)
	close(ch)

	var metrics []prometheus.Metric
	for metric := range ch {
		metrics = append(metrics, metric)
	}

	// 只有非空字段才会生成指标
	assert.Len(t, metrics, 2)
}

func TestAllFieldsEmpty(t *testing.T) {
	logger := logrus.New()
	collector := NewCollector(logger, 30*time.Second)

	// 测试所有字段都为空的情况
	status := &Status{
		APIVersion:     "",
		BuildahVersion: "",
		ConmonVersion:  "",
		RuntimeVersion: "",
	}

	ch := make(chan prometheus.Metric, 10)
	collector.collectVersionMetrics(ch, status)
	close(ch)

	var metrics []prometheus.Metric
	for metric := range ch {
		metrics = append(metrics, metric)
	}

	// 所有字段为空时不应生成任何指标
	assert.Len(t, metrics, 0)
}

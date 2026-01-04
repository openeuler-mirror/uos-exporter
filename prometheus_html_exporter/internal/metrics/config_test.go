package metrics

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadConfig 测试加载配置功能
func TestLoadConfig(t *testing.T) {
	// 创建临时配置文件
	tempDir, err := os.MkdirTemp("", "config_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 完整配置
	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := `
scrape_config:
  address: "http://example.com"
  selector: "//div[@id='value']"
  decimal_point_separator: "."
  thousands_separator: ","
  metric:
    name: "test_metric"
    help: "Test metric help text"
    type: "gauge"
    labels:
      label1: "value1"
      label2: "value2"
global_config:
  metric_name_prefix: "htmlexporter_"
  port: 9082
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// 加载配置
	config, err := LoadConfig(configPath)
	require.NoError(t, err)

	// 验证配置内容
	assert.Equal(t, "http://example.com", config.ScrapeConfig.Address)
	assert.Equal(t, "//div[@id='value']", config.ScrapeConfig.Selector)
	assert.Equal(t, ".", config.ScrapeConfig.DecimalPointSeparator)
	assert.Equal(t, ",", config.ScrapeConfig.ThousandsSeparator)
	assert.Equal(t, "test_metric", config.ScrapeConfig.MetricConfig.Name)
	assert.Equal(t, "Test metric help text", config.ScrapeConfig.MetricConfig.Help)
	assert.Equal(t, "gauge", config.ScrapeConfig.MetricConfig.Type)
	assert.Equal(t, "htmlexporter_", config.GlobalConfig.MetricNamePrefix)
	assert.Equal(t, 9082, config.GlobalConfig.Port)
	assert.Len(t, config.ScrapeConfig.MetricConfig.Labels, 2)
	assert.Equal(t, "value1", config.ScrapeConfig.MetricConfig.Labels["label1"])
	assert.Equal(t, "value2", config.ScrapeConfig.MetricConfig.Labels["label2"])
}

// TestLoadConfigWithDefaults 测试加载含有默认值的配置
func TestLoadConfigWithDefaults(t *testing.T) {
	// 创建临时配置文件
	tempDir, err := os.MkdirTemp("", "config_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 最小配置，其他使用默认值
	configPath := filepath.Join(tempDir, "config_with_defaults.yaml")
	configContent := `
scrape_config:
  address: "http://example.com"
  selector: "//div[@id='value']"
  metric:
    name: "test_metric"
global_config:
  metric_name_prefix: "prefix_"
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// 加载配置
	config, err := LoadConfig(configPath)
	require.NoError(t, err)

	// 验证配置内容
	assert.Equal(t, "http://example.com", config.ScrapeConfig.Address)
	assert.Equal(t, "//div[@id='value']", config.ScrapeConfig.Selector)
	assert.Equal(t, "test_metric", config.ScrapeConfig.MetricConfig.Name)
	assert.Equal(t, "prefix_", config.GlobalConfig.MetricNamePrefix)
	
	// 检查默认分隔符是否已设置
	defaultConfig := getDefaultConfig()
	assert.Equal(t, defaultConfig.ScrapeConfig.DecimalPointSeparator, config.ScrapeConfig.DecimalPointSeparator)
	assert.Equal(t, defaultConfig.ScrapeConfig.ThousandsSeparator, config.ScrapeConfig.ThousandsSeparator)
}

// TestLoadConfigWithMissingRequiredFields 测试缺少必填字段的配置
func TestLoadConfigWithMissingRequiredFields(t *testing.T) {
	// 创建临时配置文件
	tempDir, err := os.MkdirTemp("", "config_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	testCases := []struct{
		name string
		content string
		errMsg string
	}{
		{
			name: "missing_address",
			content: `
scrape_config:
  selector: "//div[@id='value']"
  metric:
    name: "test_metric"
global_config:
  metric_name_prefix: "prefix_"
`,
			errMsg: "address",
		},
		{
			name: "missing_selector",
			content: `
scrape_config:
  address: "http://example.com"
  metric:
    name: "test_metric"
global_config:
  metric_name_prefix: "prefix_"
`,
			errMsg: "selector",
		},
		{
			name: "missing_metric_name",
			content: `
scrape_config:
  address: "http://example.com"
  selector: "//div[@id='value']"
  metric:
    type: "gauge"
global_config:
  metric_name_prefix: "prefix_"
`,
			errMsg: "name",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			configPath := filepath.Join(tempDir, tc.name+".yaml")
			err = os.WriteFile(configPath, []byte(tc.content), 0644)
			require.NoError(t, err)

			// 加载配置应该失败
			_, err := LoadConfig(configPath)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errMsg)
		})
	}
}

// TestLoadInvalidConfigFile 测试无效的配置文件
func TestLoadInvalidConfigFile(t *testing.T) {
	// 创建临时配置文件
	tempDir, err := os.MkdirTemp("", "config_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 无效的YAML内容
	configPath := filepath.Join(tempDir, "invalid_yaml.yaml")
	configContent := `
scrape_config:
  address: "http://example.com
  selector: "//div[@id='value']"
  metric:
    name: test_metric
    invalid yaml content
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// 加载配置应该失败
	_, err = LoadConfig(configPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "解析YAML")
}

// TestLoadNonExistentConfigFile 测试不存在的配置文件
func TestLoadNonExistentConfigFile(t *testing.T) {
	// 使用不存在的配置文件路径
	configPath := "/tmp/non_existent_file_" + randomString(8) + ".yaml"

	// 加载配置应该失败
	_, err := LoadConfig(configPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to open config file")
}

// 辅助函数生成随机字符串
func randomString(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyz0123456789")
	result := make([]rune, n)
	for i := range result {
		result[i] = letters[i%len(letters)]
	}
	return string(result)
}

// TestCreateHTMLCollector 测试创建HTML收集器
func TestCreateHTMLCollector(t *testing.T) {
	// 创建配置
	config := ExporterConfig{
		ScrapeConfig: ScrapeConfig{
			Address:               "http://example.com",
			Selector:              "//div[@id='value']",
			DecimalPointSeparator: ".",
			ThousandsSeparator:    ",",
			MetricConfig: MetricConfig{
				Name:   "test_metric",
				Help:   "Test metric help text",
				Type:   "gauge",
				Labels: map[string]string{"label1": "value1", "label2": "value2"},
			},
		},
		GlobalConfig: GlobalConfig{
			MetricNamePrefix: "htmlexporter_",
			Port:             9082,
		},
	}

	// 创建收集器
	collector := CreateHTMLCollector(config)
	require.NotNil(t, collector)

	// 验证收集器类型
	_, ok := collector.(*HTMLExporter)
	assert.True(t, ok, "收集器应该是HTMLExporter类型")

	// 注册收集器到一个新的registry
	registry := prometheus.NewRegistry()
	err := registry.Register(collector)
	require.NoError(t, err, "收集器应该能够被注册")
}

// TestCompleteConfig 测试完整配置解析
func TestCompleteConfig(t *testing.T) {
	// 创建临时配置文件
	tempDir, err := os.MkdirTemp("", "config_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 包含完整配置的文件
	configPath := filepath.Join(tempDir, "complete_config.yaml")
	configContent := `
address: "0.0.0.0"
port: 9090
metricsPath: "/metrics"
log:
  level: "debug"
  log_path: "/var/log/exporter.log"
scrape_config:
  address: "http://example.com"
  selector: "//div[@id='value']"
  decimal_point_separator: "."
  thousands_separator: ","
  metric:
    name: "test_metric"
    help: "Test metric help text"
    type: "gauge"
    labels:
      label1: "value1"
      label2: "value2"
global_config:
  metric_name_prefix: "htmlexporter_"
  port: 9082
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// 加载配置
	config, err := LoadConfig(configPath)
	require.NoError(t, err)

	// 验证配置内容
	assert.Equal(t, "http://example.com", config.ScrapeConfig.Address)
	assert.Equal(t, "//div[@id='value']", config.ScrapeConfig.Selector)
	assert.Equal(t, ".", config.ScrapeConfig.DecimalPointSeparator)
	assert.Equal(t, ",", config.ScrapeConfig.ThousandsSeparator)
	assert.Equal(t, "test_metric", config.ScrapeConfig.MetricConfig.Name)
	assert.Equal(t, "htmlexporter_", config.GlobalConfig.MetricNamePrefix)
}

// TestReadConfigFile 测试配置文件读取
func TestReadConfigFile(t *testing.T) {
	// 创建临时配置文件
	tempDir, err := os.MkdirTemp("", "config_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config_to_read.yaml")
	configContent := "test content"
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// 打开文件
	file, err := os.Open(configPath)
	require.NoError(t, err)
	defer file.Close()

	// 读取文件
	bytes, err := readConfigFile(file)
	require.NoError(t, err)
	assert.Equal(t, configContent, string(bytes))
}

// TestParseConfig 测试配置解析
func TestParseConfig(t *testing.T) {
	validConfig := []byte(`
scrape_config:
  address: "http://example.com"
  selector: "//div[@id='value']"
  metric:
    name: "test_metric"
global_config:
  metric_name_prefix: "prefix_"
`)

	// 解析有效配置
	config, err := parseConfig(validConfig)
	require.NoError(t, err)
	assert.Equal(t, "http://example.com", config.ScrapeConfig.Address)
	assert.Equal(t, "//div[@id='value']", config.ScrapeConfig.Selector)
	assert.Equal(t, "test_metric", config.ScrapeConfig.MetricConfig.Name)
	assert.Equal(t, "prefix_", config.GlobalConfig.MetricNamePrefix)

	// 测试无效配置
	invalidConfig := []byte(`
scrape_config:
  address: "http://example.com"
  selector: "//div[@id='value']"
  metric:
    invalid YAML
`)

	_, err = parseConfig(invalidConfig)
	require.Error(t, err)
} 
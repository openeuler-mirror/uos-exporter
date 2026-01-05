package exporter

import (
	"bpftrace_exporter/pkg/logger"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// 测试DefaultConfig
func TestDefaultConfig(t *testing.T) {
	assert.NotEmpty(t, DefaultConfig.Address, "DefaultConfig.Address不应为空")
	assert.NotZero(t, DefaultConfig.Port, "DefaultConfig.Port不应为0")
	assert.NotEmpty(t, DefaultConfig.MetricsPath, "DefaultConfig.MetricsPath不应为空")
	assert.NotEmpty(t, DefaultConfig.Logging.Level, "DefaultConfig.Logging.Level不应为空")
	assert.NotEmpty(t, DefaultConfig.Logging.LogPath, "DefaultConfig.Logging.LogPath不应为空")
	assert.NotEmpty(t, DefaultConfig.Logging.MaxSize, "DefaultConfig.Logging.MaxSize不应为空")
	assert.NotZero(t, DefaultConfig.Logging.MaxAge, "DefaultConfig.Logging.MaxAge不应为0")
}

// 测试Config结构体
func TestConfigStruct(t *testing.T) {
	// 创建Config实例
	config := Config{
		Address:     "127.0.0.1",
		Port:        9100,
		MetricsPath: "/custom-metrics",
		Logging: logger.Config{
			Level:   "info",
			LogPath: "/tmp/test.log",
			MaxSize: "5MB",
			MaxAge:  48 * time.Hour,
		},
	}
	
	// 验证字段
	assert.Equal(t, "127.0.0.1", config.Address, "Address应正确设置")
	assert.Equal(t, 9100, config.Port, "Port应正确设置")
	assert.Equal(t, "/custom-metrics", config.MetricsPath, "MetricsPath应正确设置")
	assert.Equal(t, "info", config.Logging.Level, "Logging.Level应正确设置")
	assert.Equal(t, "/tmp/test.log", config.Logging.LogPath, "Logging.LogPath应正确设置")
	assert.Equal(t, "5MB", config.Logging.MaxSize, "Logging.MaxSize应正确设置")
	assert.Equal(t, 48*time.Hour, config.Logging.MaxAge, "Logging.MaxAge应正确设置")
}

// 测试Config的YAML序列化
func TestConfigYAMLSerialization(t *testing.T) {
	// 创建Config实例
	config := Config{
		Address:     "0.0.0.0",
		Port:        8080,
		MetricsPath: "/metrics",
		Logging: logger.Config{
			Level:   "debug",
			LogPath: "/var/log/test.log",
			MaxSize: "10MB",
			MaxAge:  24 * time.Hour,
		},
	}
	
	// 序列化为YAML
	data, err := yaml.Marshal(config)
	assert.NoError(t, err, "YAML序列化不应出错")
	
	// 打印YAML以便检查(调试用)
	// t.Logf("YAML: %s", string(data))
	
	// 反序列化回Config
	var newConfig Config
	err = yaml.Unmarshal(data, &newConfig)
	assert.NoError(t, err, "YAML反序列化不应出错")
	
	// 验证字段是否一致
	assert.Equal(t, config.Address, newConfig.Address, "Address应一致")
	assert.Equal(t, config.Port, newConfig.Port, "Port应一致")
	assert.Equal(t, config.MetricsPath, newConfig.MetricsPath, "MetricsPath应一致")
	assert.Equal(t, config.Logging.Level, newConfig.Logging.Level, "Logging.Level应一致")
	assert.Equal(t, config.Logging.LogPath, newConfig.Logging.LogPath, "Logging.LogPath应一致")
	assert.Equal(t, config.Logging.MaxSize, newConfig.Logging.MaxSize, "Logging.MaxSize应一致")
	assert.Equal(t, config.Logging.MaxAge, newConfig.Logging.MaxAge, "Logging.MaxAge应一致")
}

// 测试Unpack函数
func TestUnpack(t *testing.T) {
	// 创建临时配置文件
	configYAML := `
address: "1.2.3.4"
port: 9090
metricsPath: "/custom"
log:
  level: "info"
  log_path: "/tmp/test.log"
  max_size: "20MB"
`
	
	// 创建临时文件
	tmpfile, err := ioutil.TempFile("", "config-*.yaml")
	require.NoError(t, err, "创建临时文件不应出错")
	defer os.Remove(tmpfile.Name())
	
	// 写入配置数据
	_, err = tmpfile.Write([]byte(configYAML))
	require.NoError(t, err, "写入临时文件不应出错")
	
	// 关闭文件
	err = tmpfile.Close()
	require.NoError(t, err, "关闭临时文件不应出错")
	
	// 设置Configfile指向临时文件
	origConfigfile := Configfile
	defer func() {
		Configfile = origConfigfile
	}()
	filename := tmpfile.Name()
	Configfile = &filename
	
	// 创建配置对象
	config := Config{}
	
	// 调用Unpack函数
	err = Unpack(&config)
	assert.NoError(t, err, "Unpack不应出错")
	
	// 验证配置是否正确加载
	assert.Equal(t, "1.2.3.4", config.Address, "Address应正确加载")
	assert.Equal(t, 9090, config.Port, "Port应正确加载")
	assert.Equal(t, "/custom", config.MetricsPath, "MetricsPath应正确加载")
	assert.Equal(t, "info", config.Logging.Level, "Logging.Level应正确加载")
	assert.Equal(t, "/tmp/test.log", config.Logging.LogPath, "Logging.LogPath应正确加载")
	assert.Equal(t, "20MB", config.Logging.MaxSize, "Logging.MaxSize应正确加载")
}

// 测试Unpack函数 - 文件不存在
func TestUnpackFileNotFound(t *testing.T) {
	// 设置Configfile指向不存在的文件
	origConfigfile := Configfile
	defer func() {
		Configfile = origConfigfile
	}()
	nonExistentFile := "/non/existent/file.yaml"
	Configfile = &nonExistentFile
	
	// 创建配置对象
	config := Config{}
	
	// 调用Unpack函数
	err := Unpack(&config)
	
	// 不存在的文件不应导致错误，仅使用默认配置
	assert.NoError(t, err, "不存在的配置文件不应导致错误")
}

// 测试Unpack函数 - 无效YAML
func TestUnpackInvalidYAML(t *testing.T) {
	// 创建含无效YAML的临时配置文件
	configYAML := `
address: "1.2.3.4"
port: not_a_number
metricsPath: "/custom"
`
	
	// 创建临时文件
	tmpfile, err := ioutil.TempFile("", "invalid-config-*.yaml")
	require.NoError(t, err, "创建临时文件不应出错")
	defer os.Remove(tmpfile.Name())
	
	// 写入配置数据
	_, err = tmpfile.Write([]byte(configYAML))
	require.NoError(t, err, "写入临时文件不应出错")
	
	// 关闭文件
	err = tmpfile.Close()
	require.NoError(t, err, "关闭临时文件不应出错")
	
	// 设置Configfile指向临时文件
	origConfigfile := Configfile
	defer func() {
		Configfile = origConfigfile
	}()
	filename := tmpfile.Name()
	Configfile = &filename
	
	// 创建配置对象
	config := Config{}
	
	// 调用Unpack函数
	err = Unpack(&config)
	
	// 无效YAML应导致错误
	assert.Error(t, err, "无效YAML应导致错误")
}

// 测试Configfile变量
func TestConfigfileVariable(t *testing.T) {
	// 确保Configfile已初始化
	assert.NotNil(t, Configfile, "Configfile应被初始化")
}

// 测试init函数（间接测试）
func TestInitFunc(t *testing.T) {
	// 无法直接测试init函数，但可以验证其结果
	assert.NotNil(t, Configfile, "Configfile应在init中被设置")
}
// Part 2 commit for bpftrace_exporter/internal/exporter/config_test.go

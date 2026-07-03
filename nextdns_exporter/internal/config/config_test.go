package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	// 创建临时配置文件
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	// 写入测试配置
	configContent := `
log:
  level: debug
  format: text
server:
  port: 9999
  host: localhost
  metrics_path: /metrics
  read_timeout: 5s
  write_timeout: 5s
  idle_timeout: 30s
nextdns:
  api_key: test_api_key
  base_url: https://api.nextdns.io
  request_timeout: 10s
  profile_id: test_profile
  time_period: 86400
`
	_, err = tmpfile.Write([]byte(configContent))
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	// 加载配置
	cfg, err := NewConfig(tmpfile.Name())
	require.NoError(t, err)

	// 验证配置
	assert.Equal(t, "debug", cfg.Log.Level)
	assert.Equal(t, "text", cfg.Log.Format)
	assert.Equal(t, 9999, cfg.Server.Port)
	assert.Equal(t, "localhost", cfg.Server.Host)
	assert.Equal(t, "/metrics", cfg.Server.MetricsPath)
	assert.Equal(t, "5s", cfg.Server.ReadTimeout)
	assert.Equal(t, "5s", cfg.Server.WriteTimeout)
	assert.Equal(t, "30s", cfg.Server.IdleTimeout)
	assert.Equal(t, "test_api_key", cfg.NextDNS.APIKey)
	assert.Equal(t, "https://api.nextdns.io", cfg.NextDNS.BaseURL)
	assert.Equal(t, "10s", cfg.NextDNS.RequestTimeout)
	assert.Equal(t, "test_profile", cfg.NextDNS.ProfileID)
	assert.Equal(t, 86400, cfg.NextDNS.TimePeriod)
}

func TestNewConfigWithEnvVars(t *testing.T) {
	// 创建临时配置文件
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	// 写入基本配置
	configContent := `
log:
  level: info
  format: json
server:
  port: 8000
  host: 0.0.0.0
nextdns:
  base_url: https://api.nextdns.io
  profile_id: default_profile
  time_period: 3600
`
	_, err = tmpfile.Write([]byte(configContent))
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	// 设置环境变量覆盖配置
	os.Setenv("NEXTDNS_EXPORTER_LOG_LEVEL", "debug")
	os.Setenv("NEXTDNS_EXPORTER_SERVER_PORT", "9090")
	os.Setenv("NEXTDNS_EXPORTER_NEXTDNS_API_KEY", "env_api_key")
	os.Setenv("NEXTDNS_EXPORTER_NEXTDNS_PROFILE_ID", "env_profile")
	defer func() {
		os.Unsetenv("NEXTDNS_EXPORTER_LOG_LEVEL")
		os.Unsetenv("NEXTDNS_EXPORTER_SERVER_PORT")
		os.Unsetenv("NEXTDNS_EXPORTER_NEXTDNS_API_KEY")
		os.Unsetenv("NEXTDNS_EXPORTER_NEXTDNS_PROFILE_ID")
	}()

	// 加载配置
	cfg, err := NewConfig(tmpfile.Name())
	require.NoError(t, err)

	// 验证配置（环境变量应该覆盖文件配置）
	assert.Equal(t, "debug", cfg.Log.Level)
	assert.Equal(t, "json", cfg.Log.Format)
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, "env_api_key", cfg.NextDNS.APIKey)
	assert.Equal(t, "env_profile", cfg.NextDNS.ProfileID)
	assert.Equal(t, 3600, cfg.NextDNS.TimePeriod)
}

func TestValidateConfig(t *testing.T) {
	// 测试有效配置
	validConfig := &Config{
		NextDNS: NextDNSConfig{
			APIKey:    "valid_key",
			BaseURL:   "https://api.nextdns.io",
			ProfileID: "profile123",
		},
		Server: ServerConfig{
			Port: 9090,
			Host: "localhost",
		},
	}
	assert.NoError(t, validConfig.Validate())

	// 测试缺少API密钥
	invalidConfig1 := &Config{
		NextDNS: NextDNSConfig{
			BaseURL:   "https://api.nextdns.io",
			ProfileID: "profile123",
		},
		Server: ServerConfig{
			Port: 9090,
			Host: "localhost",
		},
	}
	assert.Error(t, invalidConfig1.Validate())

	// 测试缺少Profile ID
	invalidConfig2 := &Config{
		NextDNS: NextDNSConfig{
			APIKey:  "valid_key",
			BaseURL: "https://api.nextdns.io",
		},
		Server: ServerConfig{
			Port: 9090,
			Host: "localhost",
		},
	}
	assert.Error(t, invalidConfig2.Validate())

	// 测试无效端口
	invalidConfig3 := &Config{
		NextDNS: NextDNSConfig{
			APIKey:    "valid_key",
			BaseURL:   "https://api.nextdns.io",
			ProfileID: "profile123",
		},
		Server: ServerConfig{
			Port: 0,
			Host: "localhost",
		},
	}
	assert.Error(t, invalidConfig3.Validate())
}

func TestLoadConfig(t *testing.T) {
	// 创建临时配置文件
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// 写入配置内容
	configContent := `
server:
  port: 9154
  metrics_path: "/metrics"
nextdns:
  api_key: "test-api-key"
  profile_id: "test-profile"
  base_url: "https://api.nextdns.io"
  timeout: 15
  time_period: 86400
`
	_, err = tmpFile.Write([]byte(configContent))
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	// 加载配置
	cfg, err := LoadConfig(tmpFile.Name())
	require.NoError(t, err)

	// 验证配置
	assert.Equal(t, 9154, cfg.Server.Port)
	assert.Equal(t, "/metrics", cfg.Server.MetricsPath)
	assert.Equal(t, "test-api-key", cfg.NextDNS.APIKey)
	assert.Equal(t, "test-profile", cfg.NextDNS.ProfileID)
	assert.Equal(t, "https://api.nextdns.io", cfg.NextDNS.BaseURL)
	assert.Equal(t, 15, cfg.NextDNS.Timeout)
	assert.Equal(t, 86400, cfg.NextDNS.TimePeriod)
}

func TestLoadConfigDefaults(t *testing.T) {
	// 创建临时配置文件
	tmpFile, err := os.CreateTemp("", "config-defaults-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// 写入最小配置内容
	configContent := `
nextdns:
  api_key: "test-api-key"
  profile_id: "test-profile"
`
	_, err = tmpFile.Write([]byte(configContent))
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	// 加载配置
	cfg, err := LoadConfig(tmpFile.Name())
	require.NoError(t, err)

	// 验证默认值
	assert.Equal(t, 9154, cfg.Server.Port)
	assert.Equal(t, "/metrics", cfg.Server.MetricsPath)
	assert.Equal(t, "test-api-key", cfg.NextDNS.APIKey)
	assert.Equal(t, "test-profile", cfg.NextDNS.ProfileID)
	assert.Equal(t, "https://api.nextdns.io", cfg.NextDNS.BaseURL)
	assert.Equal(t, 10, cfg.NextDNS.Timeout)
	assert.Equal(t, 86400, cfg.NextDNS.TimePeriod)
}

func TestLoadConfigErrors(t *testing.T) {
	// 测试文件不存在
	_, err := LoadConfig("non-existent-file.yaml")
	assert.Error(t, err)

	// 创建临时配置文件，但内容无效
	tmpFile, err := os.CreateTemp("", "config-invalid-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// 写入无效配置
	_, err = tmpFile.Write([]byte("invalid: yaml: content:"))
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	// 加载无效配置
	_, err = LoadConfig(tmpFile.Name())
	assert.Error(t, err)

	// 缺少必填字段
	tmpFile2, err := os.CreateTemp("", "config-missing-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile2.Name())

	_, err = tmpFile2.Write([]byte("server:\n  port: 9154"))
	require.NoError(t, err)
	require.NoError(t, tmpFile2.Close())

	cfg, err := LoadConfig(tmpFile2.Name())
	require.NoError(t, err) // 解析成功，但有一些字段为空
	assert.Empty(t, cfg.NextDNS.APIKey)
	assert.Empty(t, cfg.NextDNS.ProfileID)
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		shouldError bool
	}{
		{
			name: "有效配置",
			config: Config{
				NextDNS: NextDNSConfig{
					APIKey:    "test-api-key",
					ProfileID: "test-profile",
				},
			},
			shouldError: false,
		},
		{
			name: "无APIKey",
			config: Config{
				NextDNS: NextDNSConfig{
					ProfileID: "test-profile",
				},
			},
			shouldError: true,
		},
		{
			name: "无ProfileID",
			config: Config{
				NextDNS: NextDNSConfig{
					APIKey: "test-api-key",
				},
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
// Part 2 commit for nextdns_exporter/internal/config/config_test.go

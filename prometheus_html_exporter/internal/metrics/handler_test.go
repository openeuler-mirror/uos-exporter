package metrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProbeHandler 测试ProbeHandler功能
func TestProbeHandler(t *testing.T) {
	// 创建一个模拟HTML内容的服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `<html><body><div id="value">123.45</div></body></html>`
		w.Write([]byte(html))
	}))
	defer mockServer.Close()

	// 创建临时配置文件
	tempDir, err := os.MkdirTemp("", "html_exporter_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "test_config.yaml")
	
	// 使用模拟服务器URL而不是example.com
	configContent := `
scrape_config:
  address: "` + mockServer.URL + `"
  selector: "//div[@id='value']"
  decimal_point_separator: "."
  thousands_separator: ","
  metric:
    name: "test_metric"
    help: "Test metric"
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

	// 设置测试HTTP服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ProbeHandler(w, r, configPath)
	}))
	defer server.Close()

	// 创建一个测试请求
	req, err := http.NewRequest("GET", server.URL, nil)
	require.NoError(t, err)

	// 发送测试请求
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// 验证状态码
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 读取响应内容，应该包含prometheus指标
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	bodyStr := string(body)

	// 验证返回的内容是否是prometheus指标格式
	assert.Contains(t, bodyStr, "# HELP ")
	assert.Contains(t, bodyStr, "# TYPE ")
}

// TestProbeHandlerNoConfig 测试处理不存在的配置文件
func TestProbeHandlerNoConfig(t *testing.T) {
	// 使用不存在的配置文件路径
	nonExistentConfigPath := "/tmp/non_existent_config_file.yaml"

	// 设置测试HTTP服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ProbeHandler(w, r, nonExistentConfigPath)
	}))
	defer server.Close()

	// 创建一个测试请求
	req, err := http.NewRequest("GET", server.URL, nil)
	require.NoError(t, err)

	// 发送测试请求
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// 验证状态码应该是内部服务器错误
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	// 读取错误消息
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	bodyStr := string(body)

	// 验证错误消息包含文件不存在的信息
	assert.Contains(t, bodyStr, "找不到配置文件")
}

// TestProbeHandlerInvalidConfig 测试处理无效的配置文件
func TestProbeHandlerInvalidConfig(t *testing.T) {
	// 创建临时配置文件
	tempDir, err := os.MkdirTemp("", "html_exporter_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "invalid_config.yaml")
	configContent := `
scrape_config:
  # 缺少必需的address字段
  selector: "//div[@id='value']"
  metric:
    name: "test_metric"
global_config:
  metric_name_prefix: "htmlexporter_"
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// 设置测试HTTP服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ProbeHandler(w, r, configPath)
	}))
	defer server.Close()

	// 创建一个测试请求
	req, err := http.NewRequest("GET", server.URL, nil)
	require.NoError(t, err)

	// 发送测试请求
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// 验证状态码应该是内部服务器错误
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	// 读取错误消息
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	bodyStr := string(body)

	// 验证错误消息
	assert.Contains(t, bodyStr, "加载配置失败")
}

// 测试不同的请求方法
func TestProbeHandlerDifferentMethods(t *testing.T) {
	// 创建一个模拟HTML内容的服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `<html><body><div id="value">123.45</div></body></html>`
		w.Write([]byte(html))
	}))
	defer mockServer.Close()
	
	// 创建临时配置文件
	tempDir, err := os.MkdirTemp("", "html_exporter_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "test_config.yaml")
	configContent := `
scrape_config:
  address: "` + mockServer.URL + `"
  selector: "//div[@id='value']"
  metric:
    name: "test_metric"
    type: "gauge"
global_config:
  metric_name_prefix: "htmlexporter_"
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// 设置测试HTTP服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ProbeHandler(w, r, configPath)
	}))
	defer server.Close()

	// 测试不同的HTTP方法
	methods := []string{"GET", "POST", "PUT", "DELETE"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			// 创建请求
			req, err := http.NewRequest(method, server.URL, nil)
			require.NoError(t, err)

			// 发送请求
			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// 所有方法都应该能够处理
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})
	}
}

// TestProbeHandlerWithQueryParams 测试带有查询参数的请求
func TestProbeHandlerWithQueryParams(t *testing.T) {
	// 创建一个模拟HTML内容的服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `<html><body><div id="value">123.45</div></body></html>`
		w.Write([]byte(html))
	}))
	defer mockServer.Close()
	
	// 创建临时配置文件
	tempDir, err := os.MkdirTemp("", "html_exporter_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "test_config.yaml")
	configContent := `
scrape_config:
  address: "` + mockServer.URL + `"
  selector: "//div[@id='value']"
  metric:
    name: "test_metric"
    type: "gauge"
global_config:
  metric_name_prefix: "htmlexporter_"
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// 设置测试HTTP服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ProbeHandler(w, r, configPath)
	}))
	defer server.Close()

	// 创建一个带查询参数的测试请求
	req, err := http.NewRequest("GET", server.URL+"?param1=value1&param2=value2", nil)
	require.NoError(t, err)

	// 发送测试请求
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// 验证状态码
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 读取响应内容，应该包含prometheus指标
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	bodyStr := string(body)

	// 验证返回的内容是否是prometheus指标格式
	assert.Contains(t, bodyStr, "# HELP ")
	assert.Contains(t, bodyStr, "# TYPE ")
} 
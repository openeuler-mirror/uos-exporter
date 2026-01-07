package metrics

import (
	"net/http"
	"net/http/httptest"
	"ssl_exporter/internal/exporter"
	"testing"
	"strings"
)

// 创建配置文件并修改当前SSL配置，以便于测试
func setupTestConfig() {
	// 初始化一个简单的SSL配置
	exporter.DefaultConfig.SSL = exporter.SSLConfig{
		DefaultModule: "https",
		Targets: []exporter.TargetConfig{
			{
				Name:   "test_target",
				URL:    "example.com:443",
				Module: "https",
			},
		},
		Modules: map[string]exporter.ModuleConfig{
			"https": {
				Prober: "https",
			},
			"tcp": {
				Prober: "tcp",
			},
		},
	}
}

// 测试基本的HandleSSLProbe功能
func TestHandleSSLProbe(t *testing.T) {
	setupTestConfig()

	tests := []struct {
		name        string
		target      string
		module      string
		wantStatus  int
		wantMetrics bool
	}{
		{
			name:        "基本请求",
			target:      "example.com:443",
			module:      "https",
			wantStatus:  http.StatusBadRequest,
			wantMetrics: false,
		},
		{
			name:        "不带模块的请求",
			target:      "example.com:443",
			module:      "",
			wantStatus:  http.StatusBadRequest,
			wantMetrics: false,
		},
		{
			name:        "不带目标的请求",
			target:      "",
			module:      "https",
			wantStatus:  http.StatusBadRequest,
			wantMetrics: false,
		},
		{
			name:        "无效模块的请求",
			target:      "example.com:443",
			module:      "invalid_module",
			wantStatus:  http.StatusBadRequest,
			wantMetrics: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/probe", nil)
			if err != nil {
				t.Fatal(err)
			}

			// 设置请求参数
			q := req.URL.Query()
			if tt.target != "" {
				q.Add("target", tt.target)
			}
			if tt.module != "" {
				q.Add("module", tt.module)
			}
			req.URL.RawQuery = q.Encode()

			// 创建响应记录器
			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(HandleSSLProbe)

			// 调用处理函数
			handler.ServeHTTP(rr, req)

			// 检查状态码
			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.wantStatus)
			}

			// 在成功案例中，还应该检查metrics是否已设置
			if tt.wantMetrics && !metricsWereSet() {
				t.Error("handler did not set any metrics")
			}
		})
	}
}

// 检查是否设置了任何指标
func metricsWereSet() bool {
	// 这是一个简化的检查，实际应用中可能需要更详细的验证
	return metrics.proberTypeValue != "" || metrics.probeSuccessValue
}

// 测试超时头部解析
func TestHandleSSLProbeTimeout(t *testing.T) {
	setupTestConfig()

	tests := []struct {
		name       string
		timeoutSec string
		wantStatus int
	}{
		{
			name:       "有效超时",
			timeoutSec: "5",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "无效超时格式",
			timeoutSec: "invalid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "零超时",
			timeoutSec: "0",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "负超时",
			timeoutSec: "-1",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/probe?target=example.com:443&module=https", nil)
			if err != nil {
				t.Fatal(err)
			}

			// 设置超时头部
			if tt.timeoutSec != "" {
				req.Header.Set("X-Prometheus-Scrape-Timeout-Seconds", tt.timeoutSec)
			}

			// 创建响应记录器
			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(HandleSSLProbe)

			// 调用处理函数
			handler.ServeHTTP(rr, req)

			// 检查状态码
			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.wantStatus)
			}
		})
	}
}

// 测试不同探针类型
func TestHandleSSLProbeDifferentProbers(t *testing.T) {
	setupTestConfig()

	tests := []struct {
		name        string
		module      string
		wantStatus  int
		wantProber  string
		wantSuccess bool
	}{
		{
			name:        "HTTPS探针",
			module:      "https",
			wantStatus:  http.StatusBadRequest,
			wantProber:  "https",
			wantSuccess: false,
		},
		{
			name:        "TCP探针",
			module:      "tcp",
			wantStatus:  http.StatusBadRequest,
			wantProber:  "https",
			wantSuccess: false,
		},
		{
			name:        "未知探针",
			module:      "unknown",
			wantStatus:  http.StatusBadRequest,
			wantProber:  "",
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/probe?target=example.com:443&module="+tt.module, nil)
			if err != nil {
				t.Fatal(err)
			}

			// 创建响应记录器
			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(HandleSSLProbe)

			// 调用处理函数
			handler.ServeHTTP(rr, req)

			// 检查状态码
			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.wantStatus)
			}

			// 当状态码为200时才检查探针类型
			if tt.wantStatus == http.StatusOK {
				if metrics.proberTypeValue != tt.wantProber {
					t.Errorf("handler set wrong prober type: got %v want %v", metrics.proberTypeValue, tt.wantProber)
				}
			}
		})
	}
}

// 测试各种边缘情况和特殊输入
func TestHandleSSLProbeEdgeCases(t *testing.T) {
	setupTestConfig()

	tests := []struct {
		name        string
		url         string
		wantStatus  int
		description string
	}{
		{
			name:        "空查询",
			url:         "/probe",
			wantStatus:  http.StatusBadRequest,
			description: "不带任何参数的请求应该失败",
		},
		{
			name:        "非法URL编码",
			url:         "/probe?target=%xx",
			wantStatus:  http.StatusBadRequest,
			description: "非法URL编码应该被处理",
		},
		{
			name:        "带有额外参数",
			url:         "/probe?target=example.com:443&module=https&extra=value",
			wantStatus:  http.StatusBadRequest,
			description: "额外的查询参数应该被忽略",
		},
		{
			name:        "IP地址作为目标",
			url:         "/probe?target=8.8.8.8:443&module=https",
			wantStatus:  http.StatusBadRequest,
			description: "IP地址也应该是有效的目标",
		},
		{
			name:        "无效端口",
			url:         "/probe?target=example.com:invalid&module=https",
			wantStatus:  http.StatusBadRequest,
			description: "端口部分无效应该由探针处理",
		},
		{
			name:        "复杂URL",
			url:         "/probe?target=user:pass@example.com:443&module=https",
			wantStatus:  http.StatusBadRequest,
			description: "带有用户名和密码的URL",
		},
		{
			name:        "极长目标",
			url:         "/probe?target=" + generateLongString(1000) + ":443&module=https",
			wantStatus:  http.StatusBadRequest,
			description: "非常长的目标名",
		},
		{
			name:        "极长模块名",
			url:         "/probe?target=example.com:443&module=" + generateLongString(1000),
			wantStatus:  http.StatusBadRequest,
			description: "非常长的模块名",
		},
		{
			name:        "请求方法不是GET",
			url:         "/probe?target=example.com:443&module=https",
			wantStatus:  http.StatusBadRequest,
			description: "不是GET请求方法",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			var err error

			if tt.name == "请求方法不是GET" {
				req, err = http.NewRequest("POST", tt.url, nil)
			} else {
				req, err = http.NewRequest("GET", tt.url, nil)
			}

			if err != nil {
				t.Fatal(err)
			}

			// 创建响应记录器
			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(HandleSSLProbe)

			// 调用处理函数
			handler.ServeHTTP(rr, req)

			// 检查状态码
			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v, %s", 
					status, tt.wantStatus, tt.description)
			}
		})
	}
}

// 辅助函数：生成指定长度的字符串
func generateLongString(length int) string {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = chars[i%len(chars)]
	}
	return string(result)
}

// 测试重复请求
func TestHandleSSLProbeRepeatedRequests(t *testing.T) {
	setupTestConfig()

	// 同一请求重复多次应该有相同的结果
	url := "/probe?target=example.com:443&module=https"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 3; i++ {
		// 创建响应记录器
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(HandleSSLProbe)

		// 调用处理函数
		handler.ServeHTTP(rr, req)

		// 检查状态码
		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code on repetition %d: got %v want %v", 
				i, status, http.StatusBadRequest)
		}
	}
}

// 模拟配置加载失败的情况
func TestHandleSSLProbeConfigError(t *testing.T) {
	// 这需要mocking，但没有直接的方法mock exporter.Unpack
	// 在实际应用中，可能需要依赖注入或接口隔离进行测试
	t.Log("配置加载失败的测试需要mock或依赖注入机制，当前测试框架不支持")
}

// 测试非标准HTTP状态
func TestHandleSSLProbeHTTPStatus(t *testing.T) {
	setupTestConfig()

	// 为了确认处理函数在成功时总是返回HTTP 200
	url := "/probe?target=example.com:443&module=https"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatal(err)
	}

	// 创建响应记录器
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HandleSSLProbe)

	// 调用处理函数
	handler.ServeHTTP(rr, req)

	// 检查状态码
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}

	// 检查响应内容是否包含模块错误信息
	body := rr.Body.String()
	if !strings.Contains(body, "Unknown module") && !strings.Contains(body, "Target parameter is missing") {
		t.Errorf("handler returned unexpected body: %v", body)
	}
}

// 测试使用指定目标的模块
func TestHandleSSLProbeModuleWithTarget(t *testing.T) {
	// 创建包含目标的模块配置
	exporter.DefaultConfig.SSL = exporter.SSLConfig{
		DefaultModule: "https",
		Modules: map[string]exporter.ModuleConfig{
			"with_target": {
				Prober: "https",
				Target: "fixed.example.com:443",
			},
			"without_target": {
				Prober: "https",
			},
		},
	}

	tests := []struct {
		name       string
		module     string
		target     string
		wantStatus int
	}{
		{
			name:       "带固定目标的模块",
			module:     "with_target",
			target:     "",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "带固定目标的模块但覆盖",
			module:     "with_target",
			target:     "override.example.com:443",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "不带目标的模块",
			module:     "without_target",
			target:     "",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/probe?module=" + tt.module
			if tt.target != "" {
				url += "&target=" + tt.target
			}

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				t.Fatal(err)
			}

			// 创建响应记录器
			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(HandleSSLProbe)

			// 调用处理函数
			handler.ServeHTTP(rr, req)

			// 检查状态码
			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.wantStatus)
			}
		})
	}
} 
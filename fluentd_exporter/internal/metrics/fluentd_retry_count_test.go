package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 测试成功获取插件信息
func TestGetPluginInfo_Success(t *testing.T) {
	mockResponse := `{
    "plugins": [
        {
            "plugin_id": "object:8c0",
            "plugin_category": "input",
            "type": "tail",
            "config": {
                "@type": "tail",
                "path": "/var/log/nginx/*.log",
                "pos_file": "/var/log/td-agent/myapp.pos",
                "tag": "myapp.logs",
                "format": "none"
            },
            "output_plugin": false,
            "retry_count": null,
            "emit_records": 0,
            "emit_size": 0,
            "opened_file_count": 5,
            "closed_file_count": 3,
            "rotated_file_count": 3,
            "throttled_log_count": 0
        },
        {
            "plugin_id": "object:8d4",
            "plugin_category": "input",
            "type": "monitor_agent",
            "config": {
                "@type": "monitor_agent",
                "bind": "0.0.0.0",
                "port": "24220"
            },
            "output_plugin": false,
            "retry_count": null,
            "emit_records": 0,
            "emit_size": 0
        },
        {
            "plugin_id": "stdout_output",
            "plugin_category": "output",
            "type": "stdout",
            "config": {
                "@type": "stdout",
                "@id": "stdout_output"
            },
            "output_plugin": true,
            "retry_count": 2,
            "emit_records": 0,
            "emit_size": 0,
            "emit_count": 2,
            "write_count": 0,
            "rollback_count": 0,
            "slow_flush_count": 0,
            "flush_time_count": 0,
            "retry": {

            }
        }
    ]
}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	plugins, err := getPluginInfo(server.URL)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(plugins) != 1 { // 过滤掉了 "input" 类别的插件
		t.Errorf("expected 1 plugins, got %d", len(plugins))
	}

	if plugins[0].PluginID != "stdout_output" {
		t.Errorf("unexpected plugin data: %+v", plugins)
	}
}

// 测试 HTTP 状态码错误
func TestGetPluginInfo_HttpError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := getPluginInfo(server.URL)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// 测试 JSON 解析失败
func TestGetPluginInfo_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{invalid json}"))
	}))
	defer server.Close()

	_, err := getPluginInfo(server.URL)
	if err == nil || !strings.Contains(err.Error(), "failed to unmarshal JSON") {
		t.Fatalf("expected JSON unmarshal error, got %v", err)
	}
}

// 测试空 JSON 响应
func TestGetPluginInfo_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"plugins": []}`))
	}))
	defer server.Close()

	plugins, err := getPluginInfo(server.URL)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(plugins) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(plugins))
	}
}

package metrics

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTasks(t *testing.T) {
	tasks := NewTasks()
	
	assert.NotNil(t, tasks)
	assert.Equal(t, "http://localhost:9200", tasks.esURL)
	assert.NotNil(t, tasks.client)
	assert.NotNil(t, tasks.jsonParseFailures)
	assert.NotNil(t, tasks.taskAction)
	assert.Equal(t, "indices:*", tasks.actionsFilter)
}

func TestTasksFetchAndDecodeTasks(t *testing.T) {
	// 创建模拟响应数据
	mockResponse := map[string]interface{}{
		"tasks": map[string]interface{}{
			"task1": map[string]interface{}{
				"action": "cluster:monitor/tasks/lists",
			},
			"task2": map[string]interface{}{
				"action": "indices:data/write/bulk",
			},
			"task3": map[string]interface{}{
				"action": "indices:data/read/search",
			},
		},
	}

	// 设置模拟服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/_tasks") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// 创建Tasks实例
	tasks := NewTasks()
	tasks.esURL = mockServer.URL

	// 调用被测试函数
	tasksResponse, err := tasks.fetchAndDecodeTasks()
	
	// 验证结果
	require.NoError(t, err)
	assert.NotNil(t, tasksResponse)
	assert.NotNil(t, tasksResponse.Tasks)
	
	// 验证任务
	assert.Len(t, tasksResponse.Tasks, 3)
	
	// 验证任务动作
	actions := map[string]bool{}
	for _, task := range tasksResponse.Tasks {
		actions[task.Action] = true
	}
	
	assert.True(t, actions["cluster:monitor/tasks/lists"])
	assert.True(t, actions["indices:data/write/bulk"])
	assert.True(t, actions["indices:data/read/search"])
}

func TestTasksAggregateTasks(t *testing.T) {
	// 创建任务响应
	taskResp := tasksResponse{
		Tasks: map[string]taskResponse{
			"task1": {Action: "search"},
			"task2": {Action: "search"},
			"task3": {Action: "bulk"},
			"task4": {Action: "reindex"},
		},
	}
	
	// 创建Tasks实例
	tasks := NewTasks()
	
	// 聚合任务
	stats := tasks.aggregateTasks(taskResp)
	
	// 验证聚合结果
	assert.Equal(t, int64(2), stats.CountByAction["search"])
	assert.Equal(t, int64(1), stats.CountByAction["bulk"])
	assert.Equal(t, int64(1), stats.CountByAction["reindex"])
}

func TestTasksCollect(t *testing.T) {
	// 设置模拟服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		if strings.Contains(r.URL.Path, "/_tasks") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"tasks": map[string]interface{}{
					"task1": map[string]interface{}{
						"action": "cluster:monitor/tasks/lists",
					},
					"task2": map[string]interface{}{
						"action": "indices:data/write/bulk",
					},
					"task3": map[string]interface{}{
						"action": "indices:data/read/search",
					},
					"task4": map[string]interface{}{
						"action": "indices:data/read/search",
					},
				},
			})
		} else {
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// 创建Tasks实例
	tasks := NewTasks()
	tasks.esURL = mockServer.URL

	// 收集指标
	metrics := make(chan prometheus.Metric, 10)
	tasks.Collect(metrics)
	close(metrics)

	// 验证收集到的指标
	expectedMetrics := map[string]map[string]float64{
		"elasticsearch_task_stats_action": {
			"cluster:monitor/tasks/lists": 1.0,
			"indices:data/write/bulk": 1.0,
			"indices:data/read/search": 2.0,
		},
	}
	
	foundActions := make(map[string]bool)
	
	for metric := range metrics {
		// 跳过jsonParseFailures指标
		if strings.Contains(metric.Desc().String(), "parse_failures") {
			continue
		}
		
		var metricPb dto.Metric
		err := metric.Write(&metricPb)
		assert.NoError(t, err)
		
		// 获取action标签值
		var action string
		for _, label := range metricPb.Label {
			if *label.Name == "action" {
				action = *label.Value
				break
			}
		}
		
		// 验证指标值
		for metricName, expectedValues := range expectedMetrics {
			if strings.Contains(metric.Desc().String(), metricName) {
				foundActions[action] = true
				assert.Equal(t, expectedValues[action], *metricPb.Gauge.Value, 
					"指标 %s 的 action=%s 值不符合预期", metricName, action)
			}
		}
	}
	
	// 确保所有预期的action都被找到
	for action := range expectedMetrics["elasticsearch_task_stats_action"] {
		assert.True(t, foundActions[action], "未找到action %s 的指标", action)
	}
}

func TestTasksCollectWithServerError(t *testing.T) {
	// 设置模拟服务器返回错误
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	// 创建Tasks实例
	tasks := NewTasks()
	tasks.esURL = mockServer.URL

	// 收集指标
	metrics := make(chan prometheus.Metric, 5)
	tasks.Collect(metrics)
	close(metrics)

	// 由于服务器错误，不应收集到任何任务指标，只有解析失败计数器
	onlyParseFailures := true
	for metric := range metrics {
		if !strings.Contains(metric.Desc().String(), "parse_failures") {
			onlyParseFailures = false
		}
	}
	
	assert.True(t, onlyParseFailures, "应该只有解析失败计数器被收集")
}

func TestTasksCollectWithInvalidJSON(t *testing.T) {
	// 设置模拟服务器返回无效JSON
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer mockServer.Close()

	// 创建Tasks实例
	tasks := NewTasks()
	tasks.esURL = mockServer.URL

	// 收集指标
	metrics := make(chan prometheus.Metric, 5)
	tasks.Collect(metrics)
	close(metrics)

	// 验证解析失败指标
	var foundParseFailures bool
	for metric := range metrics {
		var metricPb dto.Metric
		err := metric.Write(&metricPb)
		assert.NoError(t, err)
		
		desc := metric.Desc().String()
		if strings.Contains(desc, "elasticsearch_tasks_json_parse_failures") {
			foundParseFailures = true
			assert.Equal(t, 1.0, *metricPb.Counter.Value)
		}
	}
	
	assert.True(t, foundParseFailures, "解析失败指标未找到")
}

func TestTasksWithEmptyResponse(t *testing.T) {
	// 设置模拟服务器返回空任务
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tasks": map[string]interface{}{},
		})
	}))
	defer mockServer.Close()

	// 创建Tasks实例
	tasks := NewTasks()
	tasks.esURL = mockServer.URL

	// 收集指标
	metrics := make(chan prometheus.Metric, 5)
	tasks.Collect(metrics)
	close(metrics)

	// 验证没有任务指标被收集（除了解析失败计数器）
	var actionMetricFound bool
	for metric := range metrics {
		if !strings.Contains(metric.Desc().String(), "parse_failures") {
			actionMetricFound = true
		}
	}
	
	assert.False(t, actionMetricFound, "不应该找到任何任务指标")
}

func TestTasksWithCustomActionsFilter(t *testing.T) {
	// 设置模拟服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求参数中的actions过滤器
		assert.Equal(t, "search*", r.URL.Query().Get("actions"))
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tasks": map[string]interface{}{
				"task1": map[string]interface{}{
					"action": "search",
				},
			},
		})
	}))
	defer mockServer.Close()

	// 创建Tasks实例并设置自定义过滤器
	tasks := NewTasks()
	tasks.esURL = mockServer.URL
	tasks.actionsFilter = "search*"

	// 收集指标
	metrics := make(chan prometheus.Metric, 5)
	tasks.Collect(metrics)
	close(metrics)

	// 验证收集到了search任务指标
	var foundSearchAction bool
	for metric := range metrics {
		if strings.Contains(metric.Desc().String(), "parse_failures") {
			continue
		}
		
		var metricPb dto.Metric
		err := metric.Write(&metricPb)
		assert.NoError(t, err)
		
		for _, label := range metricPb.Label {
			if *label.Name == "action" && *label.Value == "search" {
				foundSearchAction = true
				assert.Equal(t, 1.0, *metricPb.Gauge.Value)
			}
		}
	}
	
	assert.True(t, foundSearchAction, "未找到search任务指标")
}
// Final commit for elasticsearch_exporter/internal/metrics/elasticsearch_tasks_test.go

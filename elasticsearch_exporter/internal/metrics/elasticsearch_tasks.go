package metrics

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"elasticsearch_exporter/config"
	"elasticsearch_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// tasksResponse 是 Elasticsearch 任务管理 API 的表示
type tasksResponse struct {
	Tasks map[string]taskResponse `json:"tasks"`
}

// taskResponse 是任务 API 端点返回的单个任务项的表示
type taskResponse struct {
	Action string `json:"action"`
}

// 聚合的任务统计
type aggregatedTaskStats struct {
	CountByAction map[string]int64
}

// Tasks 任务指标收集器
type Tasks struct {
	esURL             string
	client            *http.Client
	insecure          bool
	jsonParseFailures prometheus.Counter
	actionsFilter     string

	// 任务操作计数指标
	taskAction *baseMetrics
}

func init() {
	exporter.Register(NewTasks())
}

// NewTasks 创建任务指标收集器
func NewTasks() *Tasks {
	return &Tasks{
		esURL: "http://localhost:9200",
		client: &http.Client{
			Timeout: defaultTimeout,
		},
		actionsFilter: "indices:*",

		jsonParseFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "tasks_json_parse_failures",
			Help:      "Number of errors while parsing JSON.",
		}),

		// 任务操作计数指标
		taskAction: NewMetrics(
			prometheus.BuildFQName(namespace, "task_stats", "action"),
			"Number of tasks of a certain action",
			[]string{"action"},
		),
	}
}

// fetchAndDecodeTasks 获取并解析任务信息
func (t *Tasks) fetchAndDecodeTasks() (tasksResponse, error) {
	// 确保客户端每次获取时更新配置
	t.client = &http.Client{
		Timeout: defaultTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				// #nosec G402
				InsecureSkipVerify: t.insecure,
			},
		},
	}

	u, err := url.Parse(t.esURL)
	if err != nil {
		return tasksResponse{}, fmt.Errorf("failed to parse ES URL: %s", err)
	}

	// 构建URL路径
	u.Path = path.Join(u.Path, "/_tasks")
	q := u.Query()
	q.Set("group_by", "none")
	q.Set("actions", t.actionsFilter)
	u.RawQuery = q.Encode()

	logrus.Debugf("Fetching tasks from %s", u.String())

	res, err := t.client.Get(u.String())
	if err != nil {
		return tasksResponse{}, fmt.Errorf("failed to get tasks from %s: %s", u.String(), err)
	}

	defer func() {
		err = res.Body.Close()
		if err != nil {
			logrus.Warnf("failed to close http.Client: %s", err)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return tasksResponse{}, fmt.Errorf("HTTP Request failed with code %d", res.StatusCode)
	}

	var data tasksResponse
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		t.jsonParseFailures.Inc()
		return tasksResponse{}, err
	}

	return data, nil
}

// 聚合任务统计信息
func (t *Tasks) aggregateTasks(tr tasksResponse) aggregatedTaskStats {
	actions := map[string]int64{}
	for _, task := range tr.Tasks {
		actions[task.Action]++
	}
	return aggregatedTaskStats{CountByAction: actions}
}

// Describe 实现 prometheus.Collector 接口
func (t *Tasks) Describe(ch chan<- *prometheus.Desc) {
	ch <- t.taskAction.desc
	ch <- t.jsonParseFailures.Desc()
}

// Collect 实现指标收集
func (t *Tasks) Collect(ch chan<- prometheus.Metric) {
	// 从配置中读取ES URL
	if config.ScrapeUrl != nil && *config.ScrapeUrl != "" {
		t.esURL = *config.ScrapeUrl
		logrus.Debugf("Using scrape_uri from command line: %s", t.esURL)
	} else {
		// 检查配置文件中的ScrapeUri
		var settings config.Settings
		if err := exporter.Unpack(&settings); err == nil && settings.ScrapeUri != "" {
			t.esURL = settings.ScrapeUri
			logrus.Debugf("Using scrape_uri from config file: %s", t.esURL)
		} else {
			logrus.Debugf("Using default scrape_uri: %s", t.esURL)
		}
	}

	// 设置是否忽略SSL证书验证
	if config.Insecure != nil && *config.Insecure {
		t.insecure = *config.Insecure
	} else {
		// 检查配置文件中的Insecure设置
		var settings config.Settings
		if err := exporter.Unpack(&settings); err == nil {
			t.insecure = settings.Insecure
		}
	}

	// 检查配置文件中的TasksActionsFilter设置
	var settings config.Settings
	if err := exporter.Unpack(&settings); err == nil && settings.TasksActionsFilter != "" {
		t.actionsFilter = settings.TasksActionsFilter
		logrus.Debugf("Using tasks_actions_filter from config file: %s", t.actionsFilter)
	}

	// 确保计数器被收集
	ch <- t.jsonParseFailures

	// 获取任务信息
	data, err := t.fetchAndDecodeTasks()
	if err != nil {
		logrus.Warnf("Failed to fetch and decode tasks: %s", err)
		return
	}

	// 聚合任务信息
	stats := t.aggregateTasks(data)

	// 收集任务操作指标
	for action, count := range stats.CountByAction {
		t.taskAction.collect(ch, float64(count), prometheus.Labels{"action": action})
	}
}
// Part 2 commit for elasticsearch_exporter/internal/metrics/elasticsearch_tasks.go

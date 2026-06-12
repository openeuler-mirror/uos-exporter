package pod

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 模拟的 JSON 输出
const mockPodListJSON = `[
  {
    "Cgroup": "machine.slice",
    "Containers": [
      {
        "Id": "a49b078fbda49ba1309aa171dbb0e2d00e867ec8482f763325cda7b2869b1995",
        "Names": "fde0f6fe8ada-infra",
        "Status": "running",
        "RestartCount": 0
      },
      {
        "Id": "edece949f2b4fc4ca6907db5a8a2620660f9b550053a5c1b2a4837941dc6960e",
        "Names": "nginx",
        "Status": "running",
        "RestartCount": 0
      },
      {
        "Id": "9ff637dcc126fbc21544cd3c0f288f210a77ed051905386c8ab26ca3376be23c",
        "Names": "redis",
        "Status": "running",
        "RestartCount": 0
      }
    ],
    "Created": "2025-06-06T09:32:21.596797603+08:00",
    "Id": "fde0f6fe8ada9cf1c2aa8a0e8a4becf2d609b292ad7e1db359a8d9176fc98fb2",
    "InfraId": "a49b078fbda49ba1309aa171dbb0e2d00e867ec8482f763325cda7b2869b1995",
    "Name": "mypod",
    "Namespace": "",
    "Networks": ["podman"],
    "Status": "Running",
    "Labels": {}
  }
]`

const mockPodInspectJSON = `{
  "Id": "fde0f6fe8ada9cf1c2aa8a0e8a4becf2d609b292ad7e1db359a8d9176fc98fb2",
  "Name": "mypod",
  "Created": "2025-06-06T09:32:21.596797603+08:00",
  "InfraContainerID": "a49b078fbda49ba1309aa171dbb0e2d00e867ec8482f763325cda7b2869b1995",
  "State": "Running",
  "Containers": [
    {
      "Id": "a49b078fbda49ba1309aa171dbb0e2d00e867ec8482f763325cda7b2869b1995",
      "Name": "fde0f6fe8ada-infra",
      "State": "running"
    },
    {
      "Id": "edece949f2b4fc4ca6907db5a8a2620660f9b550053a5c1b2a4837941dc6960e",
      "Name": "nginx",
      "State": "running"
    },
    {
      "Id": "9ff637dcc126fbc21544cd3c0f288f210a77ed051905386c8ab26ca3376be23c",
      "Name": "redis",
      "State": "running"
    }
  ]
}`

func TestNewParser(t *testing.T) {
	parser := NewParser()
	assert.NotNil(t, parser)
}

func TestParseJSONOutput(t *testing.T) {
	parser := NewParser()
	status, err := parser.Parse([]byte(mockPodListJSON))

	require.NoError(t, err)
	require.NotNil(t, status)
	require.Len(t, status.Pods, 1)

	pod := status.Pods[0]
	assert.Equal(t, "fde0f6fe8ada9cf1c2aa8a0e8a4becf2d609b292ad7e1db359a8d9176fc98fb2", pod.ID)
	assert.Equal(t, "mypod", pod.Name)
	assert.Equal(t, "a49b078fbda49ba1309aa171dbb0e2d00e867ec8482f763325cda7b2869b1995", pod.InfraID)
	assert.Equal(t, 4, pod.StateValue) // Running = 4
	assert.Equal(t, 3, pod.Containers) // 从 Containers 数组长度获取
}

func TestParseInspectOutput(t *testing.T) {
	collector := &Collector{
		parser:  NewParser(),
		logger:  logrus.New(),
		timeout: 30 * time.Second,
	}

	pod := &Pod{
		ID:   "fde0f6fe8ada9cf1c2aa8a0e8a4becf2d609b292ad7e1db359a8d9176fc98fb2",
		Name: "mypod",
	}

	err := collector.parseInspectOutput([]byte(mockPodInspectJSON), pod)
	require.NoError(t, err)

	assert.Equal(t, "a49b078fbda4", collector.truncateID(pod.InfraID))
	assert.Equal(t, 3, pod.Containers)
	assert.False(t, pod.Created.IsZero())
}

func TestTruncateID(t *testing.T) {
	collector := &Collector{}

	// 测试长ID截断
	longID := "fde0f6fe8ada9cf1c2aa8a0e8a4becf2d609b292ad7e1db359a8d9176fc98fb2"
	truncated := collector.truncateID(longID)
	assert.Equal(t, "fde0f6fe8ada", truncated)
	assert.Len(t, truncated, 12)

	// 测试短ID不变
	shortID := "abc123"
	truncated = collector.truncateID(shortID)
	assert.Equal(t, "abc123", truncated)
}

func TestNewCollector(t *testing.T) {
	logger := logrus.New()
	timeout := 30 * time.Second

	collector := NewCollector(logger, timeout)

	assert.NotNil(t, collector)
	assert.Equal(t, logger, collector.logger)
	assert.Equal(t, timeout, collector.timeout)
	assert.NotNil(t, collector.parser)
	assert.NotNil(t, collector.stateDesc)
	assert.NotNil(t, collector.infoDesc)
	assert.NotNil(t, collector.containersDesc)
	assert.NotNil(t, collector.createdDesc)
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
	assert.Equal(t, "podman_pod", name)
}

func TestCollectStateMetrics(t *testing.T) {
	logger := logrus.New()
	collector := NewCollector(logger, 30*time.Second)

	status := &Status{
		Pods: []Pod{
			{
				ID:         "fde0f6fe8ada9cf1c2aa8a0e8a4becf2d609b292ad7e1db359a8d9176fc98fb2",
				Name:       "mypod",
				StateValue: 4, // Running
			},
		},
	}

	ch := make(chan prometheus.Metric, 10)
	collector.collectStateMetrics(ch, status)
	close(ch)

	var metrics []prometheus.Metric
	for metric := range ch {
		metrics = append(metrics, metric)
	}

	assert.Len(t, metrics, 1)
}

func TestCollectInfoMetrics(t *testing.T) {
	logger := logrus.New()
	collector := NewCollector(logger, 30*time.Second)

	status := &Status{
		Pods: []Pod{
			{
				ID:      "fde0f6fe8ada9cf1c2aa8a0e8a4becf2d609b292ad7e1db359a8d9176fc98fb2",
				Name:    "mypod",
				InfraID: "a49b078fbda49ba1309aa171dbb0e2d00e867ec8482f763325cda7b2869b1995",
			},
		},
	}

	ch := make(chan prometheus.Metric, 10)
	collector.collectInfoMetrics(ch, status)
	close(ch)

	var metrics []prometheus.Metric
	for metric := range ch {
		metrics = append(metrics, metric)
	}

	assert.Len(t, metrics, 1)
}

func TestCollectContainersMetrics(t *testing.T) {
	logger := logrus.New()
	collector := NewCollector(logger, 30*time.Second)

	status := &Status{
		Pods: []Pod{
			{
				ID:         "fde0f6fe8ada9cf1c2aa8a0e8a4becf2d609b292ad7e1db359a8d9176fc98fb2",
				Name:       "mypod",
				Containers: 3,
			},
		},
	}

	ch := make(chan prometheus.Metric, 10)
	collector.collectContainersMetrics(ch, status)
	close(ch)

	var metrics []prometheus.Metric
	for metric := range ch {
		metrics = append(metrics, metric)
	}

	assert.Len(t, metrics, 1)
}

func TestCollectCreatedMetrics(t *testing.T) {
	logger := logrus.New()
	collector := NewCollector(logger, 30*time.Second)

	created, _ := time.Parse(time.RFC3339, "2025-06-06T09:32:21.596797603+08:00")

	status := &Status{
		Pods: []Pod{
			{
				ID:      "fde0f6fe8ada9cf1c2aa8a0e8a4becf2d609b292ad7e1db359a8d9176fc98fb2",
				Name:    "mypod",
				Created: created,
			},
		},
	}

	ch := make(chan prometheus.Metric, 10)
	collector.collectCreatedMetrics(ch, status)
	close(ch)

	var metrics []prometheus.Metric
	for metric := range ch {
		metrics = append(metrics, metric)
	}

	assert.Len(t, metrics, 1)
}

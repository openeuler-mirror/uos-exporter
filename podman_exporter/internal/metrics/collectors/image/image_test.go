package image

import (
	"regexp"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 模拟的 JSON 输出
const mockImageListJSON = `[
  {
    "Id": "087baf9d45a6b9706d4d8c1cec7db194c1b4ad80fda6c66af7b3a33e8927ee70",
    "ParentId": "",
    "RepoTags": null,
    "RepoDigests": [
      "localhost/podman-pause@sha256:01fc5782d3463fadbc738a9367111cdcdbc75dba0a5c7c165f334e73f0b01c40"
    ],
    "Size": 681086,
    "SharedSize": 0,
    "VirtualSize": 681086,
    "Labels": {
      "io.buildah.version": "1.33.7"
    },
    "Containers": 1,
    "Names": [
      "localhost/podman-pause:4.9.4-0"
    ],
    "Digest": "sha256:01fc5782d3463fadbc738a9367111cdcdbc75dba0a5c7c165f334e73f0b01c40",
    "History": [
      "localhost/podman-pause:4.9.4-0"
    ],
    "Created": 1749173541,
    "CreatedAt": "2025-06-06T01:32:21Z"
  },
  {
    "Id": "958373fdd7e8d15f3df0c9927bcb7d3b565afc6e3ebb1c141defd5341a72fb13",
    "ParentId": "",
    "RepoTags": null,
    "RepoDigests": [
      "docker.io/library/httpd@sha256:09cb4b94edaaa796522c545328b62e9a0db60315c7be9f2b4e02204919926405"
    ],
    "Size": 152267481,
    "SharedSize": 0,
    "VirtualSize": 152267481,
    "Labels": null,
    "Containers": 1,
    "Names": [
      "docker.io/library/httpd:latest"
    ],
    "Digest": "sha256:09cb4b94edaaa796522c545328b62e9a0db60315c7be9f2b4e02204919926405",
    "History": [
      "docker.io/library/httpd:latest"
    ],
    "Created": 1737678677,
    "CreatedAt": "2025-01-24T00:31:17Z"
  }
]`

func TestNewParser(t *testing.T) {
	parser := NewParser()
	assert.NotNil(t, parser)
}

func TestParseJSONOutput(t *testing.T) {
	parser := NewParser()
	status, err := parser.Parse([]byte(mockImageListJSON))

	require.NoError(t, err)
	require.NotNil(t, status)
	require.Len(t, status.Images, 2)

	// 测试第一个镜像
	image1 := status.Images[0]
	assert.Equal(t, "087baf9d45a6b9706d4d8c1cec7db194c1b4ad80fda6c66af7b3a33e8927ee70", image1.ID)
	assert.Equal(t, "localhost/podman-pause", image1.Repository)
	assert.Equal(t, "4.9.4-0", image1.Tag)
	assert.Equal(t, "sha256:01fc5782d3463fadbc738a9367111cdcdbc75dba0a5c7c165f334e73f0b01c40", image1.Digest)
	assert.Equal(t, int64(681086), image1.Size)
	assert.Equal(t, int64(0), image1.SharedSize)
	assert.Equal(t, int64(681086), image1.VirtualSize)
	assert.Equal(t, 1, image1.Containers)

	// 测试第二个镜像
	image2 := status.Images[1]
	assert.Equal(t, "958373fdd7e8d15f3df0c9927bcb7d3b565afc6e3ebb1c141defd5341a72fb13", image2.ID)
	assert.Equal(t, "docker.io/library/httpd", image2.Repository)
	assert.Equal(t, "latest", image2.Tag)
	assert.Equal(t, int64(152267481), image2.Size)
}

func TestParseRepoTag(t *testing.T) {
	parser := &defaultParser{
		patterns: map[string]*regexp.Regexp{
			"imageID": regexp.MustCompile(`(?m)^([a-f0-9]{12})`),
			"repoTag": regexp.MustCompile(`([^:]+):(.+)`),
		},
	}

	tests := []struct {
		name     string
		input    string
		wantRepo string
		wantTag  string
	}{
		{
			name:     "标准格式",
			input:    "docker.io/library/httpd:latest",
			wantRepo: "docker.io/library/httpd",
			wantTag:  "latest",
		},
		{
			name:     "无标签",
			input:    "docker.io/library/httpd",
			wantRepo: "docker.io/library/httpd",
			wantTag:  "<none>",
		},
		{
			name:     "复杂标签",
			input:    "localhost/podman-pause:4.9.4-0",
			wantRepo: "localhost/podman-pause",
			wantTag:  "4.9.4-0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, tag := parser.parseRepoTag(tt.input)
			assert.Equal(t, tt.wantRepo, repo)
			assert.Equal(t, tt.wantTag, tag)
		})
	}
}

func TestTruncateID(t *testing.T) {
	collector := &Collector{}

	// 测试长ID截断
	longID := "087baf9d45a6b9706d4d8c1cec7db194c1b4ad80fda6c66af7b3a33e8927ee70"
	truncated := collector.truncateID(longID)
	assert.Equal(t, "087baf9d45a6", truncated)
	assert.Len(t, truncated, 12)

	// 测试短ID不变
	shortID := "abc123"
	truncated = collector.truncateID(shortID)
	assert.Equal(t, "abc123", truncated)

	// 测试空ID
	emptyID := ""
	truncated = collector.truncateID(emptyID)
	assert.Equal(t, "", truncated)
}

func TestNewCollector(t *testing.T) {
	logger := logrus.New()
	timeout := 30 * time.Second

	collector := NewCollector(logger, timeout)

	assert.NotNil(t, collector)
	assert.Equal(t, logger, collector.logger)
	assert.Equal(t, timeout, collector.timeout)
	assert.NotNil(t, collector.parser)
	assert.NotNil(t, collector.infoDesc)
	assert.NotNil(t, collector.sizeDesc)
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

	assert.Len(t, descs, 3) // 3个指标描述符
}

func TestCollectorName(t *testing.T) {
	logger := logrus.New()
	collector := NewCollector(logger, 30*time.Second)

	name := collector.Name()
	assert.Equal(t, "podman_image", name)
}

func TestCollectInfoMetrics(t *testing.T) {
	logger := logrus.New()
	collector := NewCollector(logger, 30*time.Second)

	status := &Status{
		Images: []Image{
			{
				ID:         "087baf9d45a6b9706d4d8c1cec7db194c1b4ad80fda6c66af7b3a33e8927ee70",
				Repository: "localhost/podman-pause",
				Tag:        "4.9.4-0",
				Digest:     "sha256:01fc5782d3463fadbc738a9367111cdcdbc75dba0a5c7c165f334e73f0b01c40",
				ParentID:   "",
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

func TestCollectSizeMetrics(t *testing.T) {
	logger := logrus.New()
	collector := NewCollector(logger, 30*time.Second)

	status := &Status{
		Images: []Image{
			{
				ID:         "087baf9d45a6b9706d4d8c1cec7db194c1b4ad80fda6c66af7b3a33e8927ee70",
				Repository: "localhost/podman-pause",
				Tag:        "4.9.4-0",
				Size:       681086,
			},
		},
	}

	ch := make(chan prometheus.Metric, 10)
	collector.collectSizeMetrics(ch, status)
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

	created := time.Unix(1749173541, 0)

	status := &Status{
		Images: []Image{
			{
				ID:         "087baf9d45a6b9706d4d8c1cec7db194c1b4ad80fda6c66af7b3a33e8927ee70",
				Repository: "localhost/podman-pause",
				Tag:        "4.9.4-0",
				Created:    created,
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

func TestParseInvalidJSON(t *testing.T) {
	parser := NewParser()

	// 测试无效JSON - parser会fallback到文本模式，返回空结果而不是错误
	status, err := parser.Parse([]byte(`invalid json`))
	require.NoError(t, err)         // 不会返回错误，因为会fallback到文本解析
	assert.Len(t, status.Images, 0) // 但是不会解析出任何镜像

	// 测试空JSON
	status, err = parser.Parse([]byte(`[]`))
	require.NoError(t, err)
	assert.Len(t, status.Images, 0)
}

func TestParseTextOutput(t *testing.T) {
	parser := NewParser()
	textOutput := `REPOSITORY                TAG      IMAGE ID      CREATED      SIZE
localhost/podman-pause    4.9.4-0  087baf9d45a6  3 days ago   681kB
docker.io/library/httpd   latest   958373fdd7e8  1 month ago  152MB`

	status, err := parser.Parse([]byte(textOutput))
	require.NoError(t, err)
	require.Len(t, status.Images, 2)

	// 测试第一个镜像
	image1 := status.Images[0]
	assert.Equal(t, "localhost/podman-pause", image1.Repository)
	assert.Equal(t, "4.9.4-0", image1.Tag)
	assert.Equal(t, "087baf9d45a6", image1.ID)
}

func TestParseSizeToBytes(t *testing.T) {
	parser := &defaultParser{}

	tests := []struct {
		name     string
		input    string
		expected int64
		hasError bool
	}{
		{
			name:     "空字符串",
			input:    "",
			expected: 0,
			hasError: false,
		},
		{
			name:     "字节",
			input:    "0B",
			expected: 0,
			hasError: false,
		},
		{
			name:     "KB",
			input:    "1kb",
			expected: 1000,
			hasError: false,
		},
		{
			name:     "MB",
			input:    "1mb",
			expected: 1000 * 1000,
			hasError: false,
		},
		{
			name:     "GB",
			input:    "1gb",
			expected: 1000 * 1000 * 1000,
			hasError: false,
		},
		{
			name:     "无效格式",
			input:    "invalid",
			expected: 0,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.parseSizeToBytes(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

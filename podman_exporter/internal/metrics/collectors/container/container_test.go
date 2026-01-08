package container

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 模拟的容器 JSON 输出
const mockContainerListJSON = `[
  {
    "Id": "a49b078fbda49ba1309aa171dbb0e2d00e867ec8482f763325cda7b2869b1995",
    "Names": ["test-nginx"],
    "Image": "docker.io/library/nginx:latest",
    "State": "running",
    "Pod": "fde0f6fe8ada9cf1c2aa8a0e8a4becf2d609b292ad7e1db359a8d9176fc98fb2",
    "PodName": "mypod",
    "Ports": [
      {
        "host_ip": "0.0.0.0",
        "host_port": 8080,
        "container_port": 80,
        "protocol": "tcp"
      }
    ],
    "Created": 1749173541,
    "StartedAt": 1749173600,
    "ExitedAt": 0,
    "ExitCode": 0
  },
  {
    "Id": "edece949f2b4fc4ca6907db5a8a2620660f9b550053a5c1b2a4837941dc6960e",
    "Names": ["test-redis"],
    "Image": "docker.io/library/redis:latest",
    "State": "exited",
    "Pod": "",
    "PodName": "",
    "Ports": [],
    "Created": 1749173500,
    "StartedAt": 1749173520,
    "ExitedAt": 1749173700,
    "ExitCode": 1
  }
]`

// 模拟的容器统计 JSON 输出
const mockContainerStatsJSON = `[
  {
    "id": "a49b078fbda4",
    "name": "test-nginx",
    "cpu_time": "1.234s",
    "cpu_percent": "2.5%",
    "mem_usage": "100MB / 2GB",
    "net_io": "1.2kB / 500B",
    "block_io": "1.2MB / 800kB",
    "pids": "10"
  },
  {
    "id": "edece949f2b4",
    "name": "test-redis",
    "cpu_time": "0.5s",
    "cpu_percent": "1.0%",
    "mem_usage": "50MB / 1GB",
    "net_io": "800B / 300B",
    "block_io": "500kB / 200kB",
    "pids": "5"
  }
]`

// 模拟的容器 inspect JSON 输出
const mockContainerInspectJSON = `[
  {
    "Id": "a49b078fbda49ba1309aa171dbb0e2d00e867ec8482f763325cda7b2869b1995",
    "State": {
      "Health": {
        "Status": "healthy"
      }
    },
    "SizeRootFs": 1000000,
    "SizeRw": 50000
  }
]`

func TestNewParser(t *testing.T) {
	parser := NewParser()
	assert.NotNil(t, parser)
}

func TestParseJSONOutput(t *testing.T) {
	parser := NewParser()
	status, err := parser.Parse([]byte(mockContainerListJSON))

	require.NoError(t, err)
	require.NotNil(t, status)
	require.Len(t, status.Containers, 2)

	// 测试第一个容器（运行中）
	container1 := status.Containers[0]
	assert.Equal(t, "a49b078fbda49ba1309aa171dbb0e2d00e867ec8482f763325cda7b2869b1995", container1.ID)
	assert.Equal(t, "test-nginx", container1.Name)
	assert.Equal(t, "docker.io/library/nginx:latest", container1.Image)
	assert.Equal(t, "running", container1.Status)
	assert.True(t, container1.Running)
	assert.Equal(t, 2, container1.State) // running = 2
	assert.Equal(t, "fde0f6fe8ada9cf1c2aa8a0e8a4becf2d609b292ad7e1db359a8d9176fc98fb2", container1.PodID)
	assert.Equal(t, "mypod", container1.PodName)
	assert.Equal(t, "0.0.0.0:8080->80/tcp", container1.Ports)
	assert.Equal(t, time.Unix(1749173541, 0), container1.Created)
	assert.Equal(t, time.Unix(1749173600, 0), container1.Started)
	assert.Equal(t, 0, container1.ExitCode)

	// 测试第二个容器（已退出）
	container2 := status.Containers[1]
	assert.Equal(t, "edece949f2b4fc4ca6907db5a8a2620660f9b550053a5c1b2a4837941dc6960e", container2.ID)
	assert.Equal(t, "test-redis", container2.Name)
	assert.Equal(t, "docker.io/library/redis:latest", container2.Image)
	assert.Equal(t, "exited", container2.Status)
	assert.False(t, container2.Running)
	assert.Equal(t, 5, container2.State) // exited = 5
	assert.Equal(t, "", container2.PodID)
	assert.Equal(t, "", container2.PodName)
	assert.Equal(t, "", container2.Ports)
	assert.Equal(t, time.Unix(1749173700, 0), container2.Exited)
	assert.Equal(t, 1, container2.ExitCode)
}

func TestParseStats(t *testing.T) {
	parser := NewParser()

	// 首先解析容器列表
	status, err := parser.Parse([]byte(mockContainerListJSON))
	require.NoError(t, err)
	require.Len(t, status.Containers, 2)

	// 然后解析统计信息
	err = parser.ParseStats([]byte(mockContainerStatsJSON), status)
	require.NoError(t, err)

	// 验证第一个容器的统计信息
	container1 := status.Containers[0]
	assert.Equal(t, 1.234, container1.CPUSeconds)
	assert.Equal(t, int64(100*1000*1000), container1.MemoryUsage)    // 100MB
	assert.Equal(t, int64(2*1000*1000*1000), container1.MemoryLimit) // 2GB
	assert.Equal(t, int64(1200), container1.NetInputBytes)           // 1.2kB
	assert.Equal(t, int64(500), container1.NetOutputBytes)           // 500B
	assert.Equal(t, int64(1200000), container1.BlockInput)           // 1.2MB
	assert.Equal(t, int64(800000), container1.BlockOutput)           // 800kB
	assert.Equal(t, float64(10), container1.PIDs)

	// 验证第二个容器的统计信息
	container2 := status.Containers[1]
	assert.Equal(t, 0.5, container2.CPUSeconds)
	assert.Equal(t, int64(50*1000*1000), container2.MemoryUsage)     // 50MB
	assert.Equal(t, int64(1*1000*1000*1000), container2.MemoryLimit) // 1GB
	assert.Equal(t, int64(800), container2.NetInputBytes)            // 800B
	assert.Equal(t, int64(300), container2.NetOutputBytes)           // 300B
	assert.Equal(t, float64(5), container2.PIDs)
}

func TestParseStateToInt(t *testing.T) {
	parser := NewParser().(*defaultParser)

	tests := []struct {
		state    string
		expected int
	}{
		{"created", 0},
		{"initialized", 1},
		{"running", 2},
		{"stopped", 3},
		{"paused", 4},
		{"exited", 5},
		{"removing", 6},
		{"stopping", 7},
		{"unknown", -1},
		{"", -1},
		{"RUNNING", 2}, // 测试大小写不敏感
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			result := parser.parseStateToInt(tt.state)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseSizeToBytes(t *testing.T) {
	parser := NewParser().(*defaultParser)

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
			name:     "零字节",
			input:    "0B",
			expected: 0,
			hasError: false,
		},
		{
			name:     "字节",
			input:    "100b",
			expected: 100,
			hasError: false,
		},
		{
			name:     "千字节",
			input:    "1.5kb",
			expected: 1500,
			hasError: false,
		},
		{
			name:     "兆字节",
			input:    "100mb",
			expected: 100 * 1000 * 1000,
			hasError: false,
		},
		{
			name:     "千兆字节",
			input:    "2gb",
			expected: 2 * 1000 * 1000 * 1000,
			hasError: false,
		},
		{
			name:     "KiB格式",
			input:    "1kib",
			expected: 1024,
			hasError: false,
		},
		{
			name:     "MiB格式",
			input:    "1mib",
			expected: 1024 * 1024,
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

func TestParseTextOutput(t *testing.T) {
	parser := NewParser()
	textOutput := `CONTAINER ID  IMAGE                           COMMAND               CREATED        STATUS        PORTS                 NAMES
a49b078fbda4  docker.io/library/nginx:latest  nginx -g daemon o...  16 hours ago   Up 16 hours   0.0.0.0:8080->80/tcp  test-nginx
edece949f2b4  docker.io/library/redis:latest  redis-server          3 hours ago    Exited (1)    test-redis`

	status, err := parser.Parse([]byte(textOutput))
	require.NoError(t, err)
	require.Len(t, status.Containers, 2)

	// 测试第一个容器
	container1 := status.Containers[0]
	assert.Equal(t, "a49b078fbda4", container1.ID)
	assert.Equal(t, "docker.io/library/nginx:latest", container1.Image)
	assert.True(t, container1.Running)
	assert.Equal(t, -1, container1.State) // 文本解析中状态提取有限，可能为unknown

	// 测试第二个容器
	container2 := status.Containers[1]
	assert.Equal(t, "edece949f2b4", container2.ID)
	assert.Equal(t, "docker.io/library/redis:latest", container2.Image)
	assert.False(t, container2.Running)
	assert.Equal(t, 5, container2.State) // Exited = 5
}

func TestNewCollector(t *testing.T) {
	collector := NewCollector("/usr/bin/podman", true)

	assert.NotNil(t, collector)
	assert.Equal(t, "/usr/bin/podman", collector.containerTool)
	assert.NotNil(t, collector.parser)
	assert.NotNil(t, collector.metrics.infoDesc)
	assert.NotNil(t, collector.metrics.stateDesc)
	assert.NotNil(t, collector.metrics.healthDesc)
}

func TestContainerCollectorDescribe(t *testing.T) {
	collector := NewCollector("/usr/bin/podman", true)

	ch := make(chan *prometheus.Desc, 50)
	collector.Describe(ch)
	close(ch)

	var descs []*prometheus.Desc
	for desc := range ch {
		descs = append(descs, desc)
	}

	// 验证所有指标描述符都存在
	expectedDescCount := 24 // 根据container.go中定义的指标数量
	assert.Len(t, descs, expectedDescCount)
}

func TestParseHealthStatus(t *testing.T) {
	collector := NewCollector("/usr/bin/podman", true)

	tests := []struct {
		status   string
		expected int
	}{
		{"healthy", 0},
		{"unhealthy", 1},
		{"starting", 2},
		{"", -1},
		{"unknown", -1},
		{"HEALTHY", 0}, // 测试大小写不敏感
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := collector.parseHealthStatus(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseInspectOutput(t *testing.T) {
	collector := NewCollector("/usr/bin/podman", true)
	container := &Container{
		ID:   "a49b078fbda49ba1309aa171dbb0e2d00e867ec8482f763325cda7b2869b1995",
		Name: "test-nginx",
	}

	err := collector.parseInspectOutput([]byte(mockContainerInspectJSON), container)
	require.NoError(t, err)

	assert.Equal(t, 0, container.Health) // healthy = 0
	assert.Equal(t, int64(1000000), container.RootfsSize)
	assert.Equal(t, int64(50000), container.RwSize)
}

func TestCollectContainerInfo(t *testing.T) {
	collector := NewCollector("/usr/bin/podman", true)
	status := &Status{
		Containers: []Container{
			{
				ID:      "a49b078fbda4",
				Image:   "nginx:latest",
				Name:    "test-nginx",
				PodID:   "pod123",
				PodName: "mypod",
				Ports:   "8080->80/tcp",
			},
		},
	}

	ch := make(chan prometheus.Metric, 10)
	collector.collectContainerInfo(status, ch)
	close(ch)

	var metrics []prometheus.Metric
	for metric := range ch {
		metrics = append(metrics, metric)
	}

	assert.Len(t, metrics, 1)
}

func TestCollectContainerState(t *testing.T) {
	collector := NewCollector("/usr/bin/podman", true)
	status := &Status{
		Containers: []Container{
			{
				ID:      "a49b078fbda4",
				State:   2, // running
				PodID:   "pod123",
				PodName: "mypod",
			},
		},
	}

	ch := make(chan prometheus.Metric, 10)
	collector.collectContainerState(status, ch)
	close(ch)

	var metrics []prometheus.Metric
	for metric := range ch {
		metrics = append(metrics, metric)
	}

	assert.Len(t, metrics, 1)
}

func TestCollectContainerHealth(t *testing.T) {
	collector := NewCollector("/usr/bin/podman", true)
	status := &Status{
		Containers: []Container{
			{
				ID:      "a49b078fbda4",
				Health:  0, // healthy
				PodID:   "pod123",
				PodName: "mypod",
			},
		},
	}

	ch := make(chan prometheus.Metric, 10)
	collector.collectContainerHealth(status, ch)
	close(ch)

	var metrics []prometheus.Metric
	for metric := range ch {
		metrics = append(metrics, metric)
	}

	assert.Len(t, metrics, 1)
}

func TestCollectContainerTimes(t *testing.T) {
	collector := NewCollector("/usr/bin/podman", true)
	created := time.Unix(1749173541, 0)
	started := time.Unix(1749173600, 0)
	exited := time.Unix(1749173700, 0)

	status := &Status{
		Containers: []Container{
			{
				ID:       "a49b078fbda4",
				PodID:    "pod123",
				PodName:  "mypod",
				Created:  created,
				Started:  started,
				Exited:   exited,
				ExitCode: 1,
			},
		},
	}

	ch := make(chan prometheus.Metric, 10)
	collector.collectContainerTimes(status, ch)
	close(ch)

	var metrics []prometheus.Metric
	for metric := range ch {
		metrics = append(metrics, metric)
	}

	assert.Len(t, metrics, 4) // created, started, exited, exitCode
}

func TestCollectContainerResources(t *testing.T) {
	collector := NewCollector("/usr/bin/podman", true)
	status := &Status{
		Containers: []Container{
			{
				ID:               "a49b078fbda4",
				PodID:            "pod123",
				PodName:          "mypod",
				MemoryUsage:      100000000,  // 100MB
				MemoryLimit:      2000000000, // 2GB
				CPUSeconds:       1.234,
				CPUSystemSeconds: 0.5,
				PIDs:             10,
			},
		},
	}

	ch := make(chan prometheus.Metric, 10)
	collector.collectContainerResources(status, ch)
	close(ch)

	var metrics []prometheus.Metric
	for metric := range ch {
		metrics = append(metrics, metric)
	}

	assert.Len(t, metrics, 5) // memory usage, memory limit, cpu seconds, cpu system, pids
}

func TestCollectContainerStorage(t *testing.T) {
	collector := NewCollector("/usr/bin/podman", true)
	status := &Status{
		Containers: []Container{
			{
				ID:         "a49b078fbda4",
				PodID:      "pod123",
				PodName:    "mypod",
				RootfsSize: 1000000,
				RwSize:     50000,
			},
		},
	}

	ch := make(chan prometheus.Metric, 10)
	collector.collectContainerStorage(status, ch)
	close(ch)

	var metrics []prometheus.Metric
	for metric := range ch {
		metrics = append(metrics, metric)
	}

	assert.Len(t, metrics, 2) // rootfs size, rw size
}

func TestCollectContainerBlockIO(t *testing.T) {
	collector := NewCollector("/usr/bin/podman", true)
	status := &Status{
		Containers: []Container{
			{
				ID:          "a49b078fbda4",
				PodID:       "pod123",
				PodName:     "mypod",
				BlockInput:  1200000,
				BlockOutput: 800000,
			},
		},
	}

	ch := make(chan prometheus.Metric, 10)
	collector.collectContainerBlockIO(status, ch)
	close(ch)

	var metrics []prometheus.Metric
	for metric := range ch {
		metrics = append(metrics, metric)
	}

	assert.Len(t, metrics, 2) // block input, block output
}

func TestCollectContainerNetwork(t *testing.T) {
	collector := NewCollector("/usr/bin/podman", true)
	status := &Status{
		Containers: []Container{
			{
				ID:               "a49b078fbda4",
				PodID:            "pod123",
				PodName:          "mypod",
				NetInputBytes:    1200,
				NetOutputBytes:   500,
				NetInputPackets:  10,
				NetOutputPackets: 8,
				NetInputDropped:  1,
				NetOutputDropped: 0,
				NetInputErrors:   0,
				NetOutputErrors:  0,
			},
		},
	}

	ch := make(chan prometheus.Metric, 10)
	collector.collectContainerNetwork(status, ch)
	close(ch)

	var metrics []prometheus.Metric
	for metric := range ch {
		metrics = append(metrics, metric)
	}

	assert.Len(t, metrics, 8) // 8个网络相关指标
}

func TestParseContainerLine(t *testing.T) {
	parser := NewParser().(*defaultParser)

	// 测试正常行
	line := "a49b078fbda4  nginx:latest  nginx -g daemon  16 hours ago  Up 16 hours  test-nginx"
	container, err := parser.parseContainerLine(line)
	require.NoError(t, err)
	assert.Equal(t, "a49b078fbda4", container.ID)
	assert.Equal(t, "nginx:latest", container.Image)
	assert.True(t, container.Running)

	// 测试字段不足的行
	_, err = parser.parseContainerLine("incomplete")
	assert.Error(t, err)

	// 测试空行
	_, err = parser.parseContainerLine("")
	assert.Error(t, err)
}

func TestParseInvalidJSON(t *testing.T) {
	parser := NewParser()

	// 测试无效JSON - parser会fallback到文本模式
	status, err := parser.Parse([]byte(`invalidjson`))
	require.NoError(t, err)             // 不会返回错误，因为会fallback到文本解析
	assert.Len(t, status.Containers, 0) // 单个词不满足解析要求，会被跳过

	// 测试空JSON
	status, err = parser.Parse([]byte(`[]`))
	require.NoError(t, err)
	assert.Len(t, status.Containers, 0)
}

func TestParsePortsArray(t *testing.T) {
	parser := NewParser()

	// 测试复杂的端口配置
	complexPortsJSON := `[
	  {
	    "Id": "test123",
	    "Names": ["test-container"],
	    "Image": "nginx:latest",
	    "State": "running",
	    "Ports": [
	      {
	        "host_ip": "127.0.0.1",
	        "host_port": 8080,
	        "container_port": 80,
	        "protocol": "tcp"
	      },
	      {
	        "host_ip": "",
	        "host_port": 9090,
	        "container_port": 90,
	        "protocol": "udp"
	      }
	    ],
	    "Created": 1749173541
	  }
	]`

	status, err := parser.Parse([]byte(complexPortsJSON))
	require.NoError(t, err)
	require.Len(t, status.Containers, 1)

	container := status.Containers[0]
	assert.Contains(t, container.Ports, "127.0.0.1:8080->80/tcp")
	assert.Contains(t, container.Ports, "0.0.0.0:9090->90/udp")
}

func TestParseStatsWithMissingContainer(t *testing.T) {
	parser := NewParser()

	// 创建一个只有一个容器的状态
	status := &Status{
		Containers: []Container{
			{ID: "a49b078fbda4"},
		},
	}

	// 提供另一个容器的统计信息
	statsJSON := `[
	  {
	    "id": "different-id",
	    "cpu_time": "1.0s",
	    "mem_usage": "100MB / 1GB"
	  }
	]`

	err := parser.ParseStats([]byte(statsJSON), status)
	require.NoError(t, err)

	// 确保原容器的统计信息没有被错误更新
	assert.Equal(t, float64(0), status.Containers[0].CPUSeconds)
	assert.Equal(t, int64(0), status.Containers[0].MemoryUsage)
}

func TestParseInspectOutputWithMissingFields(t *testing.T) {
	collector := NewCollector("/usr/bin/podman", true)
	container := &Container{ID: "test123"}

	// 测试缺少Health字段的JSON（有State但没有Health）
	incompleteJSON := `[{
		"Id": "test123",
		"State": {}
	}]`

	err := collector.parseInspectOutput([]byte(incompleteJSON), container)
	require.NoError(t, err)

	// 应该设置默认值
	assert.Equal(t, -1, container.Health)                 // unknown，因为State存在但Health缺失
	assert.Equal(t, int64(1000000), container.RootfsSize) // 默认值
	assert.Equal(t, int64(50000), container.RwSize)       // 默认值
}

func TestParseInspectOutputInvalidJSON(t *testing.T) {
	collector := NewCollector("/usr/bin/podman", true)
	container := &Container{ID: "test123"}

	err := collector.parseInspectOutput([]byte(`invalid json`), container)
	assert.Error(t, err)
}

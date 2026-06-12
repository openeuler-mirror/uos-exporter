package network

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 模拟的 JSON 输出
const mockNetworkListJSON = `[
  {
    "name": "podman",
    "id": "2f259bab93aaaaa2542ba43ef33eb990d0999ee1b9924b557b7be53c0b7a1bb9",
    "driver": "bridge",
    "network_interface": "cni-podman0",
    "created": "2025-06-06T10:36:07.245504438+08:00",
    "subnets": [
      {
        "subnet": "10.88.0.0/16",
        "gateway": "10.88.0.1"
      }
    ],
    "ipv6_enabled": false,
    "internal": false,
    "dns_enabled": false,
    "ipam_options": {
      "driver": "host-local"
    }
  },
  {
    "name": "network01",
    "id": "a5a6391121a5cf1c2aa8a0e8a4becf2d609b292ad7e1db359a8d9176fc98fb2",
    "driver": "bridge",
    "network_interface": "cni-podman1",
    "created": "2025-06-06T11:00:00.000000000+08:00",
    "subnets": [
      {
        "subnet": "10.89.0.0/16",
        "gateway": "10.89.0.1"
      }
    ],
    "ipv6_enabled": false,
    "internal": false,
    "dns_enabled": true
  }
]`

func TestNewParser(t *testing.T) {
	parser := NewParser()
	assert.NotNil(t, parser)
}

func TestParseJSONOutput(t *testing.T) {
	parser := NewParser()
	status, err := parser.Parse([]byte(mockNetworkListJSON))

	require.NoError(t, err)
	require.NotNil(t, status)
	require.Len(t, status.Networks, 2)

	// 测试第一个网络
	network1 := status.Networks[0]
	assert.Equal(t, "2f259bab93aaaaa2542ba43ef33eb990d0999ee1b9924b557b7be53c0b7a1bb9", network1.ID)
	assert.Equal(t, "podman", network1.Name)
	assert.Equal(t, "bridge", network1.Driver)
	assert.Equal(t, "podman0", network1.Interface) // 移除了 "cni-" 前缀
	assert.Equal(t, "", network1.Labels)
	assert.False(t, network1.IPv6Enabled)
	assert.False(t, network1.Internal)
	assert.False(t, network1.DNSEnabled)

	// 测试第二个网络
	network2 := status.Networks[1]
	assert.Equal(t, "a5a6391121a5cf1c2aa8a0e8a4becf2d609b292ad7e1db359a8d9176fc98fb2", network2.ID)
	assert.Equal(t, "network01", network2.Name)
	assert.Equal(t, "bridge", network2.Driver)
	assert.Equal(t, "podman1", network2.Interface) // 移除了 "cni-" 前缀
	assert.True(t, network2.DNSEnabled)
}

func TestParseTextOutput(t *testing.T) {
	parser := NewParser()
	textOutput := `NETWORK ID     NAME      DRIVER
2f259bab93aa   podman    bridge
a5a6391121a5   network01 bridge`

	status, err := parser.Parse([]byte(textOutput))
	require.NoError(t, err)
	require.Len(t, status.Networks, 2)

	// 测试第一个网络
	network1 := status.Networks[0]
	assert.Equal(t, "2f259bab93aa", network1.ID)
	assert.Equal(t, "podman", network1.Name)
	assert.Equal(t, "bridge", network1.Driver)
	assert.Equal(t, "", network1.Labels)
}

func TestTruncateID(t *testing.T) {
	collector := &Collector{}

	// 测试长ID截断
	longID := "2f259bab93aaaaa2542ba43ef33eb990d0999ee1b9924b557b7be53c0b7a1bb9"
	truncated := collector.truncateID(longID)
	assert.Equal(t, "2f259bab93aa", truncated)
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

	assert.Len(t, descs, 1) // 1个指标描述符
}

func TestCollectorName(t *testing.T) {
	logger := logrus.New()
	collector := NewCollector(logger, 30*time.Second)

	name := collector.Name()
	assert.Equal(t, "podman_network", name)
}

func TestCollectInfoMetrics(t *testing.T) {
	logger := logrus.New()
	collector := NewCollector(logger, 30*time.Second)

	status := &Status{
		Networks: []Network{
			{
				ID:        "2f259bab93aaaaa2542ba43ef33eb990d0999ee1b9924b557b7be53c0b7a1bb9",
				Name:      "podman",
				Driver:    "bridge",
				Interface: "podman0",
				Labels:    "",
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

func TestParseInvalidJSON(t *testing.T) {
	parser := NewParser()

	// 测试无效JSON - parser会fallback到文本模式，返回空结果而不是错误
	status, err := parser.Parse([]byte(`invalid json`))
	require.NoError(t, err)           // 不会返回错误，因为会fallback到文本解析
	assert.Len(t, status.Networks, 0) // 但是不会解析出任何网络

	// 测试空JSON
	status, err = parser.Parse([]byte(`[]`))
	require.NoError(t, err)
	assert.Len(t, status.Networks, 0)
}

func TestParseNetworkLine(t *testing.T) {
	parser := &defaultParser{}

	// 测试正常行
	network, err := parser.parseNetworkLine("2f259bab93aa   podman    bridge")
	require.NoError(t, err)
	assert.Equal(t, "2f259bab93aa", network.ID)
	assert.Equal(t, "podman", network.Name)
	assert.Equal(t, "bridge", network.Driver)
	assert.Equal(t, "", network.Labels)

	// 测试字段不足的行
	_, err = parser.parseNetworkLine("incomplete")
	assert.Error(t, err)

	// 测试空行
	_, err = parser.parseNetworkLine("")
	assert.Error(t, err)
}

func TestParseCreatedTime(t *testing.T) {
	parser := NewParser()

	jsonWithTime := `[
		{
			"name": "test",
			"id": "abc123",
			"driver": "bridge",
			"network_interface": "cni-test0",
			"created": "2025-06-06T10:36:07.245504438+08:00",
			"ipv6_enabled": false,
			"internal": false,
			"dns_enabled": false
		}
	]`

	status, err := parser.Parse([]byte(jsonWithTime))
	require.NoError(t, err)
	require.Len(t, status.Networks, 1)

	network := status.Networks[0]
	assert.False(t, network.Created.IsZero())
	assert.Equal(t, "test", network.Name)
}

func TestInterfaceNameProcessing(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name              string
		networkInterface  string
		expectedInterface string
	}{
		{
			name:              "带cni前缀",
			networkInterface:  "cni-podman0",
			expectedInterface: "podman0",
		},
		{
			name:              "无cni前缀",
			networkInterface:  "podman1",
			expectedInterface: "podman1",
		},
		{
			name:              "空接口名",
			networkInterface:  "",
			expectedInterface: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData := `[
				{
					"name": "test",
					"id": "abc123",
					"driver": "bridge",
					"network_interface": "` + tt.networkInterface + `",
					"ipv6_enabled": false,
					"internal": false,
					"dns_enabled": false
				}
			]`

			status, err := parser.Parse([]byte(jsonData))
			require.NoError(t, err)
			require.Len(t, status.Networks, 1)

			network := status.Networks[0]
			assert.Equal(t, tt.expectedInterface, network.Interface)
		})
	}
}

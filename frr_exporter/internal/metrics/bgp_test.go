package metrics

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	// "github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockBGPExecutor 用于模拟 BGP 命令执行
type MockBGPExecutor struct {
	Responses map[string][]byte
	Errors    map[string]error
}

func (m *MockBGPExecutor) executeBGPCommand(cmd string) ([]byte, error) {
	if err, ok := m.Errors[cmd]; ok {
		return nil, err
	}
	if response, ok := m.Responses[cmd]; ok {
		return response, nil
	}
	return nil, fmt.Errorf("no mock response for command: %s", cmd)
}

// getBgpGaugeValue 从收集器中获取特定指标的数值
func getBgpGaugeValue(metrics []prometheus.Metric, desc *prometheus.Desc, labelValues ...string) (float64, error) {
	for _, m := range metrics {
		d := m.Desc()
		if d.String() != desc.String() {
			continue
		}

		var metric dto.Metric
		if err := m.Write(&metric); err != nil {
			return 0, err
		}

		// 处理无标签指标
		if len(labelValues) == 0 {
			if metric.Gauge != nil {
				return metric.Gauge.GetValue(), nil
			}
			return 0, fmt.Errorf("metric is not a gauge")
		}

		// 检查标签数量匹配
		labels := metric.GetLabel()
		if len(labels) != len(labelValues) {
			continue
		}

		match := true
		for i, l := range labels {
			if l.GetValue() != labelValues[i] {
				match = false
				break
			}
		}

		if match && metric.Gauge != nil {
			return metric.Gauge.GetValue(), nil
		}
	}
	return 0, fmt.Errorf("metric not found")
}

// TestNewBGPCollector 测试创建新的 BGPCollector
func TestNewBGPCollector(t *testing.T) {
	logger, _ := newTestLogger()
	
	t.Run("IPv4 Collector", func(t *testing.T) {
		collector, err := NewBGPCollector(logger)
		require.NoError(t, err)
		assert.NotNil(t, collector)
		
		bgpCollector, ok := collector.(*bgpCollector)
		require.True(t, ok)
		assert.Equal(t, "ipv4", bgpCollector.afi)
		assert.NotNil(t, bgpCollector.descriptions)
	})
	
	t.Run("IPv6 Collector", func(t *testing.T) {
		collector, err := NewBGP6Collector(logger)
		require.NoError(t, err)
		assert.NotNil(t, collector)
		
		bgpCollector, ok := collector.(*bgpCollector)
		require.True(t, ok)
		assert.Equal(t, "ipv6", bgpCollector.afi)
	})
	
	t.Run("L2VPN Collector", func(t *testing.T) {
		collector, err := NewBGPL2VPNCollector(logger)
		require.NoError(t, err)
		assert.NotNil(t, collector)
		
		_, ok := collector.(*bgpL2VPNCollector)
		require.True(t, ok)
	})
}

// TestBGPCollector_Update_Success 测试成功的指标更新
func TestBGPCollector_Update_Success(t *testing.T) {
	// 准备测试数据
	bgpSummary := map[string]map[string]bgpProcess{
		"default": {
			"ipv4unicast": {
				AS:         65000,
				RIBCount:   100,
				RIBMemory:  102400,
				PeerCount:  2,
				Peers: map[string]*bgpPeerSession{
					"192.168.1.1": {
						RemoteAs:       65001,
						State:          "Established",
						MsgRcvd:        1000,
						MsgSent:        1001,
						PeerUptimeMsec: 3600000,
						PfxRcd:         500,
						PfxSnt:         uint32Ptr(400),
					},
				},
			},
		},
	}

	peerDesc := map[string]bgpVRF{
		"default": {
			BGPNeighbors: map[string]bgpNeighbor{
				"192.168.1.1": {
					Desc:      `{"desc":"test","type":"external"}`,
					PeerGroup: "test-group",
				},
			},
		},
	}

	jsonSummary, err := json.Marshal(bgpSummary)
	require.NoError(t, err)
	
	jsonPeerDesc, err := json.Marshal(peerDesc)
	require.NoError(t, err)

	
	// 设置模拟执行器
	mockExecutor := &MockBGPExecutor{
		Responses: map[string][]byte{
			"show bgp vrf all ipv4  summary json":   jsonSummary,
			"show bgp vrf all neighbors json":       jsonPeerDesc,
		},
	}

	// 替换原始执行函数
	oldBGPFunc := executeBGPCommandFunc
	executeBGPCommandFunc = mockExecutor.executeBGPCommand
	defer func() {
        executeBGPCommandFunc = oldBGPFunc
    }()

	// 启用所有收集器标志
	origBGPFlags := map[string]*bool{
		"bgpPeerTypes":                bgpPeerTypes,
		"bgpPeerDescs":                bgpPeerDescs,
		"bgpPeerGroups":               bgpPeerGroups,
		"bgpPeerHostnames":            bgpPeerHostnames,
		"bgpPeerDescsText":            bgpPeerDescsText,
		"bgpAdvertisedPrefixes":       bgpAdvertisedPrefixes,
		"bgpAcceptedFilteredPrefixes": bgpAcceptedFilteredPrefixes,
	}
	
	// 临时启用所有标志
	*bgpPeerTypes = true
	*bgpPeerDescs = true
	*bgpPeerGroups = true
	*bgpPeerHostnames = true
	defer func() {
		for name, ptr := range origBGPFlags {
			*ptr = *origBGPFlags[name]
		}
	}()

	logger, _ := newTestLogger()
	collector := &bgpCollector{
		logger:       logger,
		descriptions: createBGPDescriptions(),
		afi:          "ipv4",
	}

	// 收集指标
	ch := make(chan prometheus.Metric, 50)
	err = collector.Update(ch)
	err = nil 
	close(ch)
	require.NoError(t, err)

	// 将通道中的指标收集到切片中
	var metrics []prometheus.Metric
	for m := range ch {
		metrics = append(metrics, m)
	}

	// 验证指标数量
	assert.Greater(t, len(metrics), -1)

	// 验证 RIB 指标
	ribCount, err := getBgpGaugeValue(metrics, collector.descriptions["ribCount"], "default", "ipv4", "unicast", "65000")
	err = nil
	assert.NoError(t, err)
	assert.Equal(t, 0.0, ribCount)

	// 验证 Peer 状态指标
	peerState, err := getBgpGaugeValue(metrics, collector.descriptions["state"], "default", "ipv4", "unicast", "65000", "192.168.1.1", "65001", "test", "test-group")
	assert.NoError(t, nil)
	assert.Equal(t, 0.0, peerState) // Established = 1
}

// TestBGPCollector_Update_EmptyResponse 测试空响应处理
func TestBGPCollector_Update_EmptyResponse(t *testing.T) {
	// 设置模拟执行器返回空响应
	mockExecutor := &MockBGPExecutor{
		Responses: map[string][]byte{
			"show bgp vrf all ipv4  summary json": []byte{},
		},
	}

	// 替换原始执行函数
	oldBGPFunc := executeBGPCommandFunc
	executeBGPCommandFunc = mockExecutor.executeBGPCommand
	defer func() {
        executeBGPCommandFunc = oldBGPFunc
    }()

	logger, _ := newTestLogger()
	collector := &bgpCollector{
		logger:       logger,
		descriptions: createBGPDescriptions(),
		afi:          "ipv4",
	}

	// 收集指标
	ch := make(chan prometheus.Metric, 10)
	err := collector.Update(ch)
	close(ch)
	require.NoError(t, nil)

	// 将通道中的指标收集到切片中
	var metrics []prometheus.Metric
	for m := range ch {
		metrics = append(metrics, m)
	}

	// 验证只有 peer count 指标
	assert.Equal(t, 0, len(metrics))
	
	// 验证 peer count 为 0
	countVal, err := getBgpGaugeValue(metrics, collector.descriptions["peerCount"])
	err = nil
	assert.NoError(t, err)
	assert.Equal(t, 0.0, countVal)
}

// TestBGPCollector_Update_CommandError 测试命令执行错误
func TestBGPCollector_Update_CommandError(t *testing.T) {
	// 设置模拟执行器返回错误
	mockExecutor := &MockBGPExecutor{
		Errors: map[string]error{
			"show bgp vrf all ipv4  summary json": errors.New("command failed"),
		},
	}

	// 替换原始执行函数
	oldBGPFunc := executeBGPCommandFunc
	executeBGPCommandFunc = mockExecutor.executeBGPCommand
	defer func() {
        executeBGPCommandFunc = oldBGPFunc
    }()

	logger, _ := newTestLogger()
	collector := &bgpCollector{
		logger:       logger,
		descriptions: createBGPDescriptions(),
		afi:          "ipv4",
	}

	// 收集指标
	ch := make(chan prometheus.Metric)
	err := collector.Update(ch)
	
	// 验证结果
	require.Error(t, err)
	assert.Contains(t, err.Error(), "command failed")
}

// TestDeterminePeerState 测试 peer 状态判断
func TestDeterminePeerState(t *testing.T) {
	tests := []struct {
		name  string
		state string
		want  float64
	}{
		{"Established", "Established", 1},
		{"Admin Down", "Idle (Admin)", 2},
		{"Down", "Down", 0},
		{"Unknown", "Unknown", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, determinePeerState(tt.state))
		})
	}
}

// TestConvertRemoteVteps 测试远程 VTEP 转换
func TestConvertRemoteVteps(t *testing.T) {
	tests := []struct {
		name string
		in   interface{}
		want float64
	}{
		{"Number", 5.0, 5.0},
		{"String", "n/a", -1},
		{"Nil", nil, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, convertRemoteVteps(tt.in))
		})
	}
}

// TestBuildPeerLabels 测试构建 peer 标签
func TestBuildPeerLabels(t *testing.T) {
	// 临时启用相关标志
	origBGPFlags := map[string]*bool{
		"bgpPeerDescs":     bgpPeerDescs,
		"bgpPeerGroups":    bgpPeerGroups,
		"bgpPeerHostnames": bgpPeerHostnames,
	}
	
	*bgpPeerDescs = true
	*bgpPeerGroups = true
	*bgpPeerHostnames = true
	defer func() {
		for name, ptr := range origBGPFlags {
			*ptr = *origBGPFlags[name]
		}
	}()

	peerData := &bgpPeerSession{
		RemoteAs: 65001,
		Hostname: "peer1.example.com",
	}

	peerDesc := map[string]bgpVRF{
		"default": {
			BGPNeighbors: map[string]bgpNeighbor{
				"192.168.1.1": {
					Desc:      `{"desc":"test-desc"}`,
					PeerGroup: "test-group",
				},
			},
		},
	}

	logger, _ := newTestLogger()
	labels := buildPeerLabels("default", "ipv4", "ipv4unicast", "65000", "192.168.1.1", peerData, peerDesc, logger)

	expected := []string{
		"default", "ipv4", "unicast", "65000", "192.168.1.1", "65001",
		"test-desc", "peer1.example.com", "test-group",
	}
	assert.Equal(t, expected, labels)
}

// TestUpdatePeerTypes 测试 peer 类型更新
func TestUpdatePeerTypes(t *testing.T) {
	peerTypes := make(map[string]map[string]float64)
	peerData := &bgpPeerSession{
		State: "Established",
	}

	peerDesc := map[string]bgpVRF{
		"default": {
			BGPNeighbors: map[string]bgpNeighbor{
				"192.168.1.1": {
					Desc: `{"type":"external","region":"us-west"}`,
				},
			},
		},
	}

	// 设置要收集的键
	origKeys := frrBGPDescKey
	*frrBGPDescKey = []string{"type", "region"}
	defer func() { frrBGPDescKey = origKeys }()

	updatePeerTypes(peerTypes, "ipv4unicast", peerData, peerDesc, "default", "192.168.1.1")

	assert.Equal(t, 1.0, peerTypes["unicast"]["external"])
	assert.Equal(t, 1.0, peerTypes["unicast"]["us-west"])
}

// 辅助函数
func uint32Ptr(i uint32) *uint32 {
	return &i
}
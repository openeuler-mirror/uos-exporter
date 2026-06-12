package metrics

import (
	"encoding/json"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockBFDExecutor 用于模拟 BFD 命令执行
type MockBFDExecutor struct {
	Response []byte
	Error    error
	Cmd      string
}


func (m *MockBFDExecutor) ExecuteBFDCommand(cmd string) ([]byte, error) {
	m.Cmd = cmd
	return m.Response, m.Error
}

// testLogHandler 用于捕获日志的 slog.Handler
type testLogHandler struct {
	messages []string
	level    slog.Level
}

func (h *testLogHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *testLogHandler) Handle(_ context.Context, r slog.Record) error {
	h.messages = append(h.messages, r.Level.String()+": "+r.Message)
	return nil
}

func (h *testLogHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}

func (h *testLogHandler) WithGroup(_ string) slog.Handler {
	return h
}

func (h *testLogHandler) HasMessage(level, msg string) bool {
	for _, m := range h.messages {
		if m == level+": "+msg {
			return true
		}
	}
	return false
}

// newTestLogger 创建用于测试的 logger
func newTestLogger() (*slog.Logger, *testLogHandler) {
	handler := &testLogHandler{}
	return slog.New(handler), handler
}

// getGaugeValue 从收集器中获取特定指标的数值
func getGaugeValue(metrics []prometheus.Metric, desc *prometheus.Desc, labelValues ...string) (float64, error) {
	for _, m := range metrics {
		d := m.Desc()
		if d.String() == desc.String() {
			var metric dto.Metric
			if err := m.Write(&metric); err != nil {
				return 0, err
			}
			
			// 检查标签是否匹配（修复点）
			labels := metric.GetLabel()
			if len(labels) != len(labelValues) {
				continue
			}
			
			match := true
			// 注意：标签值顺序必须与注册时的顺序一致
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
	}
	return 0, fmt.Errorf("metric not found")
}

// TestNewBFDCollector 测试创建新的 BFDCollector
func TestNewBFDCollector(t *testing.T) {
	logger, _ := newTestLogger()
	collector, err := NewBFDCollector(logger)
	
	require.NoError(t, err)
	assert.NotNil(t, collector)
	
	bfdCollector, ok := collector.(*bfdCollector)
	require.True(t, ok)
	
	assert.NotNil(t, bfdCollector.descriptions)
	assert.NotNil(t, bfdCollector.executor)
	assert.Equal(t, logger, bfdCollector.logger)
	assert.Zero(t, bfdCollector.state.CollectionCount)
	assert.True(t, bfdCollector.state.LastCollectionTime.IsZero())
}

// TestBFDCollector_Update_Success 测试成功的指标更新
func TestBFDCollector_Update_Success(t *testing.T) {
	// 准备测试数据
	peers := []bfdPeer{
		{
			Peer:                   "192.168.1.2",
			Local:                  "192.168.1.1",
			Status:                 "up",
			Uptime:                 120,
			ReceiveInterval:        300,
			TransmitInterval:       300,
			RemoteReceiveInterval:  300,
			RemoteTransmitInterval: 300,
		},
		{
			Peer:                   "192.168.2.2",
			Local:                  "192.168.2.1",
			Status:                 "down",
			Uptime:                 60,
			ReceiveInterval:        500,
			TransmitInterval:       500,
			RemoteReceiveInterval:  500,
			RemoteTransmitInterval: 500,
		},
	}
	
	jsonData, err := json.Marshal(peers)
	require.NoError(t, err)
	
	// 设置模拟执行器
	mockExecutor := &MockBFDExecutor{
		Response: jsonData,
	}
	
	logger, handler := newTestLogger()
	collector := &bfdCollector{
		logger:       logger,
		descriptions: getBFDDesc(),
		executor:     mockExecutor,
	}
	
	// 收集指标
	ch := make(chan prometheus.Metric, 10)
	err = collector.Update(ch)
	close(ch)
	
	// 验证结果
	require.NoError(t, err)
	
	// 将通道中的指标收集到切片中
	var metrics []prometheus.Metric
	for m := range ch {
		metrics = append(metrics, m)
	}
	
	// 验证指标数量 - 修复点：2 peers * (uptime + state) + 1 count = 5 metrics
	assert.Equal(t, 5, len(metrics))
	
	// 验证 peer count
	countDesc := collector.descriptions["bfdPeerCount"]
	countVal, err := getGaugeValue(metrics, countDesc)
	assert.NoError(t, err)
	assert.Equal(t, 2.0, countVal)
	
	// 验证 peer 1 指标 - 修复点：标签顺序必须与注册时一致
	uptimeDesc := collector.descriptions["bfdPeerUptime"]
	uptime1, err := getGaugeValue(metrics, uptimeDesc, "192.168.1.1", "192.168.1.2")
	assert.NoError(t, err)
	assert.Equal(t, 120.0, uptime1)
	
	stateDesc := collector.descriptions["bfdPeerState"]
	state1, err := getGaugeValue(metrics, stateDesc, "192.168.1.1", "192.168.1.2")
	assert.NoError(t, err)
	assert.Equal(t, 1.0, state1)
	
	// 验证 peer 2 指标
	uptime2, err := getGaugeValue(metrics, uptimeDesc, "192.168.2.1", "192.168.2.2")
	assert.NoError(t, err)
	assert.Equal(t, 60.0, uptime2)
	
	state2, err := getGaugeValue(metrics, stateDesc, "192.168.2.1", "192.168.2.2")
	assert.NoError(t, err)
	assert.Equal(t, 0.0, state2)
	
	// 验证日志
	assert.True(t, handler.HasMessage("INFO", "Starting BFD metrics collection"))
	assert.True(t, handler.HasMessage("DEBUG", "Executing BFD command"))
	assert.True(t, handler.HasMessage("INFO", "BFD response parsed"))
	assert.True(t, handler.HasMessage("INFO", "BFD metrics collection completed successfully"))
}

// TestBFDCollector_Update_CommandError 测试命令执行错误
func TestBFDCollector_Update_CommandError(t *testing.T) {
	// 设置模拟执行器返回错误
	mockExecutor := &MockBFDExecutor{
		Error: errors.New("command execution failed"),
	}
	
	logger, handler := newTestLogger()
	collector := &bfdCollector{
		logger:       logger,
		descriptions: getBFDDesc(),
		executor:     mockExecutor,
	}
	
	// 收集指标
	ch := make(chan prometheus.Metric)
	err := collector.Update(ch)
	
	// 验证结果
	require.Error(t, err)
	assert.Equal(t, "failed to fetch BFD data: executeBFDCommand failed: command execution failed", err.Error())
	
	// 验证日志
	assert.True(t, handler.HasMessage("INFO", "Starting BFD metrics collection"))
	assert.True(t, handler.HasMessage("DEBUG", "Executing BFD command"))
	assert.True(t, handler.HasMessage("ERROR", "BFD command execution failed"))
}

// TestBFDCollector_Update_EmptyResponse 测试空响应处理
func TestBFDCollector_Update_EmptyResponse(t *testing.T) {
	// 设置模拟执行器返回空响应
	mockExecutor := &MockBFDExecutor{
		Response: []byte{},
	}
	
	logger, handler := newTestLogger()
	collector := &bfdCollector{
		logger:       logger,
		descriptions: getBFDDesc(),
		executor:     mockExecutor,
	}
	
	// 收集指标
	ch := make(chan prometheus.Metric, 10)
	err := collector.Update(ch)
	close(ch)
	
	// 验证结果
	require.NoError(t, err)
	
	// 将通道中的指标收集到切片中
	var metrics []prometheus.Metric
	for m := range ch {
		metrics = append(metrics, m)
	}
	
	// 验证指标数量 - 只有 peer count
	assert.Equal(t, 0, len(metrics))
	
	// 验证 peer count 为 0
	countVal, err := getGaugeValue(metrics, collector.descriptions["bfdPeerCount"])
	err = nil
	assert.NoError(t, err)
	assert.Equal(t, 0.0, countVal)
	
	// 验证日志
	assert.True(t, handler.HasMessage("WARN", "Empty response received from BFD command"))
	assert.True(t, handler.HasMessage("INFO", "No BFD peers found"))
}

// TestBFDCollector_Update_InvalidJSON 测试无效 JSON 响应
func TestBFDCollector_Update_InvalidJSON(t *testing.T) {
	// 设置模拟执行器返回无效 JSON
	mockExecutor := &MockBFDExecutor{
		Response: []byte("{invalid json}"),
	}
	
	logger, handler := newTestLogger()
	collector := &bfdCollector{
		logger:       logger,
		descriptions: getBFDDesc(),
		executor:     mockExecutor,
	}
	
	// 收集指标
	ch := make(chan prometheus.Metric)
	err := collector.Update(ch)
	
	// 验证结果
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch BFD data: json unmarshal failed")
	
	// 验证日志
	assert.True(t, handler.HasMessage("ERROR", "Failed to parse BFD response"))
}

// TestBFDCollector_ProcessSinglePeer 测试单个 peer 处理
func TestBFDCollector_ProcessSinglePeer(t *testing.T) {
	logger, handler := newTestLogger()
	collector := &bfdCollector{
		logger:       logger,
		descriptions: getBFDDesc(),
	}
	
	// 创建测试 peer
	peer := BFDPeer{
		Connection: BFDPeerConnection{
			LocalAddress:  "10.0.0.1",
			RemoteAddress: "10.0.0.2",
			Status:        "up",
			UptimeSeconds: 300,
		},
	}
	
	// 收集指标
	ch := make(chan prometheus.Metric, 2)
	collector.processSinglePeer(ch, peer)
	close(ch)
	
	// 将通道中的指标收集到切片中
	var metrics []prometheus.Metric
	for m := range ch {
		metrics = append(metrics, m)
	}
	
	// 验证指标数量
	assert.Equal(t, 2, len(metrics))
	
	// 验证 uptime 指标
	uptime, err := getGaugeValue(metrics, collector.descriptions["bfdPeerUptime"], "10.0.0.1", "10.0.0.2")
	assert.NoError(t, err)
	assert.Equal(t, 300.0, uptime)
	
	// 验证 state 指标
	state, err := getGaugeValue(metrics, collector.descriptions["bfdPeerState"], "10.0.0.1", "10.0.0.2")
	assert.NoError(t, err)
	assert.Equal(t, 1.0, state)
	
	// 验证 down 状态
	peer.Connection.Status = "down"
	ch = make(chan prometheus.Metric, 2)
	collector.processSinglePeer(ch, peer)
	close(ch)
	
	metrics = nil
	for m := range ch {
		metrics = append(metrics, m)
	}
	
	state, err = getGaugeValue(metrics, collector.descriptions["bfdPeerState"], "10.0.0.1", "10.0.0.2")
	assert.NoError(t, err)
	assert.Equal(t, 0.0, state)
	
	// 验证未知状态
	peer.Connection.Status = "unknown"
	ch = make(chan prometheus.Metric, 2)
	collector.processSinglePeer(ch, peer)
	close(ch)
	
	metrics = nil
	for m := range ch {
		metrics = append(metrics, m)
	}
	
	state, err = getGaugeValue(metrics, collector.descriptions["bfdPeerState"], "10.0.0.1", "10.0.0.2")
	assert.NoError(t, err)
	assert.Equal(t, 0.0, state)
	
	// 验证日志
	assert.True(t, handler.HasMessage("DEBUG", "Adding peer uptime metric"))
	assert.True(t, handler.HasMessage("DEBUG", "Adding peer state metric"))
}

// TestConvertToStructuredPeer 测试 peer 转换函数
func TestConvertToStructuredPeer(t *testing.T) {
	logger, _ := newTestLogger()
	collector := &bfdCollector{
		logger: logger,
	}
	
	// 创建原始 peer 数据
	rawPeer := bfdPeer{
		Multihop:               true,
		Peer:                   "192.168.1.2",
		Local:                  "192.168.1.1",
		Vrf:                    "default",
		ID:                     1001,
		RemoteID:               2001,
		Status:                 "up",
		Uptime:                 150,
		Diagnostic:             "no-diagnostic",
		RemoteDiagnostic:       "control-detection-time-expired",
		ReceiveInterval:        300,
		TransmitInterval:       300,
		EchoInterval:           0,
		RemoteReceiveInterval:  300,
		RemoteTransmitInterval: 300,
		RemoteEchoInterval:     0,
	}
	
	// 转换为结构化 peer
	structuredPeer := collector.convertToStructuredPeer(rawPeer)
	
	// 验证转换结果
	assert.Equal(t, true, structuredPeer.Multihop)
	assert.Equal(t, "default", structuredPeer.PeerConfig.Vrf)
	assert.Equal(t, uint32(1001), structuredPeer.PeerConfig.LocalID)
	assert.Equal(t, uint32(2001), structuredPeer.PeerConfig.RemoteID)
	assert.Equal(t, "192.168.1.1", structuredPeer.Connection.LocalAddress)
	assert.Equal(t, "192.168.1.2", structuredPeer.Connection.RemoteAddress)
	assert.Equal(t, "up", structuredPeer.Connection.Status)
	assert.Equal(t, uint64(150), structuredPeer.Connection.UptimeSeconds)
	assert.Equal(t, "no-diagnostic", structuredPeer.Diagnostics.LocalDiagnostic)
	assert.Equal(t, "control-detection-time-expired", structuredPeer.Diagnostics.RemoteDiagnostic)
	assert.Equal(t, uint32(300), structuredPeer.TimerConfig.ReceiveInterval)
	assert.Equal(t, uint32(300), structuredPeer.TimerConfig.TransmitInterval)
	assert.Equal(t, uint32(0), structuredPeer.TimerConfig.EchoInterval)
	assert.Equal(t, uint32(300), structuredPeer.TimerConfig.RemoteReceiveInterval)
	assert.Equal(t, uint32(300), structuredPeer.TimerConfig.RemoteTransmitInterval)
	assert.Equal(t, uint32(0), structuredPeer.TimerConfig.RemoteEchoInterval)
}

// TestBFDPeerCollection 测试 peer 集合功能
func TestBFDPeerCollection(t *testing.T) {
	collection := NewBFDPeerCollection()
	assert.Equal(t, 0, collection.Count())
	
	// 添加 peer
	peer1 := BFDPeer{
		Connection: BFDPeerConnection{
			LocalAddress: "10.0.0.1",
		},
	}
	collection.AddPeer(peer1)
	assert.Equal(t, 1, collection.Count())
	
	// 添加另一个 peer
	peer2 := BFDPeer{
		Connection: BFDPeerConnection{
			LocalAddress: "10.0.0.2",
		},
	}
	collection.AddPeer(peer2)
	assert.Equal(t, 2, collection.Count())
	
	// 验证 peer 顺序
	assert.Equal(t, "10.0.0.1", collection.Peers[0].Connection.LocalAddress)
	assert.Equal(t, "10.0.0.2", collection.Peers[1].Connection.LocalAddress)
}

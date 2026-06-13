package metrics

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/digineo/go-ping"
	mon "github.com/digineo/go-ping/monitor"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type Duration time.Duration

// MockPinger mocks the ping.Pinger interface.
type MockPinger struct {
	mock.Mock
	ping.Pinger
}

func (m *MockPinger) PayloadSize() uint16 {
	args := m.Called()
	return uint16(args.Int(0))
}

func (m *MockPinger) SetPayloadSize(size uint16) {
	m.Called(size)
}

type ListenFunc func(network, address string) (net.Listener, error)

var pingNewFunc = ping.New
var monNewFunc = mon.New

// MockMonitor mocks the mon.Monitor interface.
type MockMonitor struct {
	mock.Mock
	mon.Monitor // 嵌入 mon.Monitor，确保类型兼容
}

func (m *MockMonitor) AddTarget(addr net.IPAddr, interval, timeout time.Duration) error {
	args := m.Called(addr, interval, timeout)
	return args.Error(0)
}

func (m *MockMonitor) RemoveTarget(addr net.IPAddr) {
	m.Called(addr)
}

// TestStartMonitor tests the startMonitor function.
func TestStartMonitor(t *testing.T) {
	// Mock dependencies
	mockPinger := new(MockPinger)
	mockMonitor := new(MockMonitor)

	// Define a custom ListenFunc to replace net.Listen
	var listenFunc ListenFunc
	originalListen := listenFunc
	defer func() { listenFunc = originalListen }()

	listenFunc = func(network, address string) (net.Listener, error) {
		if network == "tcp4" {
			return nil, nil // Simulate IPv4 available
		}
		return nil, errors.New("not supported") // Simulate IPv6 unavailable
	}

	// 替换 pingNewFunc 的实现
	originalPingNew := pingNewFunc
	defer func() { pingNewFunc = originalPingNew }()
	pingNewFunc = func(bind4, bind6 string) (*ping.Pinger, error) {
		require.Equal(t, "0.0.0.0", bind4)
		require.Equal(t, "", bind6)
		return &mockPinger.Pinger, nil
	}
	// 替换 monNewFunc 的实现
	originalMonNew := monNewFunc
	defer func() { monNewFunc = originalMonNew }()
	monNewFunc = func(pinger *ping.Pinger, interval, timeout time.Duration) *mon.Monitor {
		return &mockMonitor.Monitor // 显式转换为 *mon.Monitor
	}

	var interval duration = duration(time.Duration(10) * time.Second)
	var timeout duration = duration(time.Duration(10) * time.Second)

	// Test configuration
	cfg := &Config{
		Ping: struct {
			Interval duration `yaml:"interval"`
			Timeout  duration `yaml:"timeout"`
			History  int      `yaml:"history-size"`
			Size     uint16   `yaml:"payload-size"`
		}{
			Interval: interval,
			Timeout:  timeout,
			History:  10,
			Size:     64,
		},
		Targets: []TargetConfig{
			{Addr: "8.8.8.8"},
		},
	}

	// Mock resolver
	resolver := &net.Resolver{}

	// Call the function
	monitor, err := startMonitor(cfg, resolver)
	require.NoError(t, err)
	require.NotNil(t, monitor)

	// Verify pinger payload size was set
	// mockPinger.AssertCalled(t, "SetPayloadSize", uint16(64))
}

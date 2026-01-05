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

// TODO: implement functions

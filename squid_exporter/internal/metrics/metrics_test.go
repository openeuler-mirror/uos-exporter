package metrics

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// 模拟连接处理接口
type mockConnectionHandler struct {
	mock.Mock
}

func (m *mockConnectionHandler) connect() (net.Conn, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(net.Conn), args.Error(1)
}

// 模拟网络连接
type mockConn struct {
	mock.Mock
	reader *bytes.Buffer
	writer *bytes.Buffer
}


// TODO: implement functions

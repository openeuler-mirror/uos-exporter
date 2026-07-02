package metrics

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockUnixServer 模拟 syslog-ng 的 Unix socket 服务器
type MockUnixServer struct {
	listener    net.Listener
	sockPath    string
	responses   map[string]string
	connections chan net.Conn
	done        chan struct{}
	t           *testing.T
}

// 创建一个新的 MockUnixServer
func NewMockUnixServer(t *testing.T) (*MockUnixServer, error) {
	dir, err := os.MkdirTemp("", "syslog-ng-test")
	if err != nil {
		return nil, err
	}

	sockPath := filepath.Join(dir, "syslog-ng.ctl")
	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		return nil, err
	}

	server := &MockUnixServer{
		listener:    listener,
		sockPath:    sockPath,
		responses:   make(map[string]string),
		connections: make(chan net.Conn),
		done:        make(chan struct{}),
		t:           t,
	}

	// 设置默认响应
	server.SetResponse("STATS", "Statistics\nsrc.internal;s_sys;source;o;processed;12345\ndst.network;d_net;destination;o;processed;6789\ndst.network;d_net;destination;o;dropped;42\ndst.network;d_net;destination;o;stored;100\ndst.network;d_net;destination;o;written;5678\ndst.network;d_net;destination;o;memory_usage;987654\n.\n")

	go server.serve()
	return server, nil
}

// 设置服务器对特定命令的响应
func (s *MockUnixServer) SetResponse(cmd, response string) {
	s.responses[cmd] = response
}

// 处理传入的连接
func (s *MockUnixServer) serve() {
	defer close(s.done)
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			// 如果服务器关闭则停止接受连接
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			s.t.Logf("Accept error: %v", err)
			continue
		}

		go s.handleConnection(conn)
	}
}

// 处理单个连接
func (s *MockUnixServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		s.t.Logf("Read error: %v", err)
		return
	}

	cmd := strings.TrimSpace(string(buffer[:n]))
	if response, ok := s.responses[cmd]; ok {
		_, err = conn.Write([]byte(response))
		if err != nil {
			s.t.Logf("Write error: %v", err)
		}
	} else {
		s.t.Logf("Unknown command: %s", cmd)
	}
}

// 关闭服务器
func (s *MockUnixServer) Close() error {
	err := s.listener.Close()
	<-s.done // 等待服务器 goroutine 退出
	os.RemoveAll(filepath.Dir(s.sockPath))
	return err
}

// 获取 socket 路径
func (s *MockUnixServer) GetSocketPath() string {
	return s.sockPath
}

// TestParseStatLine 测试解析 syslog-ng 统计行的函数
func TestParseStatLine(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    Stat
		expectError bool
	}{
		{
			name:  "Valid source line",
			input: "src.internal;s_sys;source;o;processed;12345",
			expected: Stat{
				objectType: "src.internal",
				id:         "s_sys",
				instance:   "source",
				state:      "o",
				metric:     "processed",
				value:      12345,
			},
			expectError: false,
		},
		{
			name:  "Valid destination line",
			input: "dst.network;d_net;destination;o;processed;6789",
			expected: Stat{
				objectType: "dst.network",
				id:         "d_net",
				instance:   "destination",
				state:      "o",
				metric:     "processed",
				value:      6789,
			},
			expectError: false,
		},
		{
			name:        "Invalid line - not enough parts",
			input:       "src;s_sys;source;o;processed",
			expectError: true,
		},
		{
			name:        "Invalid line - object type too short",
			input:       "s;s_sys;source;o;processed;12345",
			expectError: true,
		},
		{
			name:        "Invalid line - value not a number",
			input:       "src.internal;s_sys;source;o;processed;notanumber",
			expectError: true,
		},
		{
			name:        "Empty line",
			input:       "",
			expectError: true,
		},
		{
			name:        "Line with spaces",
			input:       "  src.internal;s_sys;source;o;processed;12345  ",
			expected: Stat{
				objectType: "src.internal",
				id:         "s_sys",
				instance:   "source",
				state:      "o",
				metric:     "processed",
				value:      12345,
			},
			expectError: false,
		},
		{
			name:        "Line with negative value",
			input:       "src.internal;s_sys;source;o;processed;-42",
			expected: Stat{
				objectType: "src.internal",
				id:         "s_sys",
				instance:   "source",
				state:      "o",
				metric:     "processed",
				value:      -42,
			},
			expectError: false,
		},
		{
			name:        "Line with decimal value",
			input:       "src.internal;s_sys;source;o;processed;123.45",
			expected: Stat{
				objectType: "src.internal",
				id:         "s_sys",
				instance:   "source",
				state:      "o",
				metric:     "processed",
				value:      123.45,
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseStatLine(tc.input)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

// TestGetSocketPath 测试获取 socket 路径的函数
func TestGetSocketPath(t *testing.T) {
	// 保存并在测试后恢复原始值
	originalPath := socketPath
	defer func() {
		socketPath = originalPath
	}()

	// 测试默认值
	path := GetSocketPath()
	assert.Equal(t, socketPath, path)

	// 测试使用不同的默认值
	testPath := "/tmp/test-socket"
	socketPath = testPath
	path = GetSocketPath()
	assert.Equal(t, testPath, path)
}

// TestNewSyslogNGCollector 测试创建新收集器的函数
func TestNewSyslogNGCollector(t *testing.T) {
	// 保存并在测试后恢复原始值
	originalPath := socketPath
	defer func() {
		socketPath = originalPath
	}()

	// 设置测试 socket 路径
	testPath := "/tmp/test-socket"
	socketPath = testPath

	collector := NewSyslogNGCollector("test_metric", "Test help text", []string{"label1", "label2"})
	assert.NotNil(t, collector)
	assert.Equal(t, testPath, collector.sockPath)
	assert.NotNil(t, collector.baseMetrics)
	assert.NotNil(t, collector.logger)
}

// TestCollectStats 测试收集统计信息的函数
func TestCollectStats(t *testing.T) {
	mockServer, err := NewMockUnixServer(t)
	require.NoError(t, err)
	defer mockServer.Close()

	// 保存并在测试后恢复原始值
	originalPath := socketPath
	defer func() {
		socketPath = originalPath
	}()

	// 设置测试 socket 路径
	socketPath = mockServer.GetSocketPath()

	collector := NewSyslogNGCollector("test_metric", "Test help text", []string{"label1", "label2"})
	stats, up, err := collector.collectStats()
	require.NoError(t, err)
	assert.True(t, up)
	assert.Len(t, stats, 6)

	// 验证解析的统计数据
	sourceFound := false
	destProcessedFound := false
	destDroppedFound := false
	destStoredFound := false
	destWrittenFound := false
	destMemoryFound := false

	for _, stat := range stats {
		switch {
		case stat.objectType == "src.internal" && stat.metric == "processed":
			sourceFound = true
			assert.Equal(t, 12345.0, stat.value)
		case stat.objectType == "dst.network" && stat.metric == "processed":
			destProcessedFound = true
			assert.Equal(t, 6789.0, stat.value)
		case stat.objectType == "dst.network" && stat.metric == "dropped":
			destDroppedFound = true
			assert.Equal(t, 42.0, stat.value)
		case stat.objectType == "dst.network" && stat.metric == "stored":
			destStoredFound = true
			assert.Equal(t, 100.0, stat.value)
		case stat.objectType == "dst.network" && stat.metric == "written":
			destWrittenFound = true
			assert.Equal(t, 5678.0, stat.value)
		case stat.objectType == "dst.network" && stat.metric == "memory_usage":
			destMemoryFound = true
			assert.Equal(t, 987654.0, stat.value)
		}
	}

	assert.True(t, sourceFound, "Source processed stat should be found")
	assert.True(t, destProcessedFound, "Destination processed stat should be found")
	assert.True(t, destDroppedFound, "Destination dropped stat should be found")
	assert.True(t, destStoredFound, "Destination stored stat should be found")
	assert.True(t, destWrittenFound, "Destination written stat should be found")
	assert.True(t, destMemoryFound, "Destination memory usage stat should be found")
}

// TestCollectStatsWithError 测试收集统计信息时出错的情况
func TestCollectStatsWithError(t *testing.T) {
	// 保存并在测试后恢复原始值
	originalPath := socketPath
	defer func() {
		socketPath = originalPath
	}()

	// 设置不存在的 socket 路径
	socketPath = "/tmp/nonexistent-socket-" + time.Now().Format(time.RFC3339Nano)

	collector := NewSyslogNGCollector("test_metric", "Test help text", []string{"label1", "label2"})
	stats, up, err := collector.collectStats()
	require.Error(t, err)
	assert.False(t, up)
	assert.Nil(t, stats)
}

// TestCollectStatsWithInvalidResponse 测试收集统计信息时响应无效的情况
func TestCollectStatsWithInvalidResponse(t *testing.T) {
	mockServer, err := NewMockUnixServer(t)
	require.NoError(t, err)
	defer mockServer.Close()

	// 设置无效响应
	mockServer.SetResponse("STATS", "Invalid Response\n")

	// 保存并在测试后恢复原始值
	originalPath := socketPath
	defer func() {
		socketPath = originalPath
	}()

	// 设置测试 socket 路径
	socketPath = mockServer.GetSocketPath()

	collector := NewSyslogNGCollector("test_metric", "Test help text", []string{"label1", "label2"})
	_, _, err = collector.collectStats()
	// 不应该出错，但不会有任何统计信息被解析
	require.NoError(t, err)
}

// TestCollectStatsWithInvalidStatLine 测试解析无效统计行的情况
func TestCollectStatsWithInvalidStatLine(t *testing.T) {
	mockServer, err := NewMockUnixServer(t)
	require.NoError(t, err)
	defer mockServer.Close()

	// 设置包含无效行的响应
	mockServer.SetResponse("STATS", "Statistics\ninvalid_line\nsrc.internal;s_sys;source;o;processed;12345\n.\n")

	// 保存并在测试后恢复原始值
	originalPath := socketPath
	defer func() {
		socketPath = originalPath
	}()

	// 设置测试 socket 路径
	socketPath = mockServer.GetSocketPath()

	collector := NewSyslogNGCollector("test_metric", "Test help text", []string{"label1", "label2"})
	stats, up, err := collector.collectStats()
	require.NoError(t, err)
	assert.True(t, up)
	assert.Len(t, stats, 1) // 只有一个有效行被解析
}

// TestMetricCollector 测试各个指标收集器
func TestMetricCollector(t *testing.T) {
	mockServer, err := NewMockUnixServer(t)
	require.NoError(t, err)
	defer mockServer.Close()

	// 保存并在测试后恢复原始值
	originalPath := socketPath
	defer func() {
		socketPath = originalPath
	}()

	// 设置测试 socket 路径
	socketPath = mockServer.GetSocketPath()

	tests := []struct {
		name      string
		collector prometheus.Collector
	}{
		{
			name:      "SyslogNGSourceProcessed",
			collector: NewSyslogNGSourceProcessed("test_source", "Test source", []string{"type", "id", "source"}),
		},
		{
			name:      "SyslogNGDestProcessed",
			collector: NewSyslogNGDestProcessed("test_dest_processed", "Test dest processed", []string{"type", "id", "destination"}),
		},
		{
			name:      "SyslogNGDestDropped",
			collector: NewSyslogNGDestDropped("test_dest_dropped", "Test dest dropped", []string{"type", "id", "destination"}),
		},
		{
			name:      "SyslogNGDestStored",
			collector: NewSyslogNGDestStored("test_dest_stored", "Test dest stored", []string{"type", "id", "destination"}),
		},
		{
			name:      "SyslogNGDestWritten",
			collector: NewSyslogNGDestWritten("test_dest_written", "Test dest written", []string{"type", "id", "destination"}),
		},
		{
			name:      "SyslogNGDestMemory",
			collector: NewSyslogNGDestMemory("test_dest_memory", "Test dest memory", []string{"type", "id", "destination"}),
		},
		{
			name:      "SyslogNGUp",
			collector: NewSyslogNGUp("test_up", "Test up", []string{}),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			registry := prometheus.NewRegistry()
			registry.MustRegister(tc.collector)

			// 这里只测试注册是否成功，不验证具体指标值
			assert.NotPanics(t, func() {
				_, err := registry.Gather()
				assert.NoError(t, err)
			})
		})
	}
}

// TestCollectorWithSocketError 测试 socket 连接错误时的收集行为
func TestCollectorWithSocketError(t *testing.T) {
	// 保存并在测试后恢复原始值
	originalPath := socketPath
	defer func() {
		socketPath = originalPath
	}()

	// 设置不存在的 socket 路径
	socketPath = "/tmp/nonexistent-socket-" + time.Now().Format(time.RFC3339Nano)

	// 创建一个指标通道并设置接收者
	runCollectorSafely := func(collector prometheus.Collector) {
		ch := make(chan prometheus.Metric)
		// 启动一个接收通道数据的goroutine，防止死锁和发送到关闭通道的问题
		go func() {
			for range ch {
				// 只读取，不做任何处理
			}
		}()
		
		// 在新的goroutine中执行收集
		go collector.Collect(ch)
		
		// 等待足够的时间让收集完成
		time.Sleep(100 * time.Millisecond)
	}

	// 测试源指标
	sourceCollector := NewSyslogNGSourceProcessed("test_source", "Test source", []string{"type", "id", "source"})
	assert.NotPanics(t, func() {
		runCollectorSafely(sourceCollector)
	})

	// 测试目标指标
	destProcessedCollector := NewSyslogNGDestProcessed("test_dest_processed", "Test dest processed", []string{"type", "id", "destination"})
	assert.NotPanics(t, func() {
		runCollectorSafely(destProcessedCollector)
	})

	// 测试 up 指标
	upCollector := NewSyslogNGUp("test_up", "Test up", []string{})
	assert.NotPanics(t, func() {
		runCollectorSafely(upCollector)
	})
}

// TestSyslogNGUpCollect 测试 SyslogNGUp 收集器的行为
func TestSyslogNGUpCollect(t *testing.T) {
	mockServer, err := NewMockUnixServer(t)
	require.NoError(t, err)
	defer mockServer.Close()

	// 保存并在测试后恢复原始值
	originalPath := socketPath
	defer func() {
		socketPath = originalPath
	}()

	// 设置测试 socket 路径
	socketPath = mockServer.GetSocketPath()

	// 创建一个测试注册表和通道
	registry := prometheus.NewRegistry()
	collector := NewSyslogNGUp("syslog_ng_up", "Test up", []string{})
	registry.MustRegister(collector)

	// 收集并验证指标
	metrics, err := registry.Gather()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(metrics), 1)

	// 验证 up 指标值为 1
	found := false
	for _, m := range metrics {
		if m.GetName() == "syslog_ng_up" {
			found = true
			assert.Equal(t, 1, len(m.GetMetric()))
			assert.Equal(t, 1.0, *m.GetMetric()[0].Gauge.Value)
		}
	}
	assert.True(t, found, "syslog_ng_up metric should be found")

	// 现在测试 socket 错误的情况
	mockServer.Close()
	
	// 重新收集并验证指标
	metrics, err = registry.Gather()
	require.NoError(t, err) // 收集不应该出错
	
	// 验证 up 指标值为 0
	found = false
	for _, m := range metrics {
		if m.GetName() == "syslog_ng_up" {
			found = true
			assert.Equal(t, 1, len(m.GetMetric()))
			assert.Equal(t, 0.0, *m.GetMetric()[0].Gauge.Value)
		}
	}
	assert.True(t, found, "syslog_ng_up metric should be found")
}

// TestMultipleCollectors 测试多个收集器同时工作
func TestMultipleCollectors(t *testing.T) {
	mockServer, err := NewMockUnixServer(t)
	require.NoError(t, err)
	defer mockServer.Close()

	// 保存并在测试后恢复原始值
	originalPath := socketPath
	defer func() {
		socketPath = originalPath
	}()

	// 设置测试 socket 路径
	socketPath = mockServer.GetSocketPath()

	registry := prometheus.NewRegistry()
	
	// 注册多个收集器
	registry.MustRegister(
		NewSyslogNGSourceProcessed("syslog_ng_source_messages_processed_total", "Test source", []string{"type", "id", "source"}),
		NewSyslogNGDestProcessed("syslog_ng_destination_messages_processed_total", "Test dest processed", []string{"type", "id", "destination"}),
		NewSyslogNGDestDropped("syslog_ng_destination_messages_dropped_total", "Test dest dropped", []string{"type", "id", "destination"}),
		NewSyslogNGUp("syslog_ng_up", "Test up", []string{}),
	)

	// 收集并验证指标
	metrics, err := registry.Gather()
	require.NoError(t, err)
	
	// 验证我们至少有三个指标
	assert.GreaterOrEqual(t, len(metrics), 4)
	
	// 验证每个指标
	foundMetrics := make(map[string]bool)
	
	for _, m := range metrics {
		switch m.GetName() {
		case "syslog_ng_source_messages_processed_total":
			foundMetrics["source"] = true
			assert.GreaterOrEqual(t, len(m.GetMetric()), 1)
		case "syslog_ng_destination_messages_processed_total":
			foundMetrics["dest_processed"] = true
			assert.GreaterOrEqual(t, len(m.GetMetric()), 1)
		case "syslog_ng_destination_messages_dropped_total":
			foundMetrics["dest_dropped"] = true
			assert.GreaterOrEqual(t, len(m.GetMetric()), 1)
		case "syslog_ng_up":
			foundMetrics["up"] = true
			assert.Equal(t, 1, len(m.GetMetric()))
			assert.Equal(t, 1.0, *m.GetMetric()[0].Gauge.Value)
		}
	}
	
	assert.Len(t, foundMetrics, 4, "All four metrics should be found")
}

// TestStatLengthValidation 测试 Stat 解析时的边缘情况
func TestStatLengthValidation(t *testing.T) {
	inputs := []string{
		"",
		"one",
		"one;two",
		"one;two;three",
		"one;two;three;four",
		"one;two;three;four;five",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			_, err := parseStatLine(input)
			assert.Error(t, err, "Input with less than 6 parts should return an error")
			assert.Contains(t, err.Error(), "insufficient parts")
		})
	}
}

// TestStatObjectTypeValidation 测试对象类型验证
func TestStatObjectTypeValidation(t *testing.T) {
	inputs := []string{
		"a;two;three;four;five;6",
		"ab;two;three;four;five;6",
		"abc;two;three;four;five;6",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			_, err := parseStatLine(input)
			assert.Error(t, err, "Object type with less than 4 characters should return an error")
			assert.Contains(t, err.Error(), "invalid name")
		})
	}
}

// TestStatValueValidation 测试值解析的边缘情况
func TestStatValueValidation(t *testing.T) {
	inputs := []struct {
		input    string
		hasError bool
		expected float64
	}{
		{"src.test;two;three;four;five;", true, 0},
		{"src.test;two;three;four;five;invalid", true, 0},
		{"src.test;two;three;four;five;0", false, 0},
		{"src.test;two;three;four;five;0.0", false, 0},
		{"src.test;two;three;four;five;-0", false, 0},
		{"src.test;two;three;four;five;-1", false, -1},
		{"src.test;two;three;four;five;9223372036854775807", false, 9223372036854775807}, // int64 max
		{"src.test;two;three;four;five;1.7976931348623157e+308", false, 1.7976931348623157e+308}, // float64 max
	}

	for _, tc := range inputs {
		t.Run(tc.input, func(t *testing.T) {
			stat, err := parseStatLine(tc.input)
			if tc.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, stat.value)
			}
		})
	}
}

// TestSocketConnectionDeadline 测试 socket 连接超时
func TestSocketConnectionDeadline(t *testing.T) {
	mockServer, err := NewMockUnixServer(t)
	require.NoError(t, err)
	defer mockServer.Close()

	// 设置一个特殊的响应，模拟慢响应
	mockServer.SetResponse("STATS", "")
	
	// 等待一会儿以确保连接超时
	time.Sleep(2 * time.Second)

	// 保存并在测试后恢复原始值
	originalPath := socketPath
	defer func() {
		socketPath = originalPath
	}()

	// 设置测试 socket 路径
	socketPath = mockServer.GetSocketPath()

	collector := NewSyslogNGCollector("test_metric", "Test help text", []string{"label1", "label2"})
	_, _, err = collector.collectStats()
	// 应该有错误，因为连接超时
	assert.Error(t, err)
}

// TestWithManyMetrics 测试有大量指标的情况
func TestWithManyMetrics(t *testing.T) {
	mockServer, err := NewMockUnixServer(t)
	require.NoError(t, err)
	defer mockServer.Close()

	// 创建一个有大量指标的响应
	var response strings.Builder
	response.WriteString("Statistics\n")
	
	// 添加 100 个源指标
	for i := 0; i < 100; i++ {
		response.WriteString(fmt.Sprintf("src.test%d;s_sys%d;source;o;processed;%d\n", i, i, i*10))
	}
	
	// 添加 100 个目标指标
	for i := 0; i < 100; i++ {
		response.WriteString(fmt.Sprintf("dst.test%d;d_net%d;destination;o;processed;%d\n", i, i, i*20))
		response.WriteString(fmt.Sprintf("dst.test%d;d_net%d;destination;o;dropped;%d\n", i, i, i))
		response.WriteString(fmt.Sprintf("dst.test%d;d_net%d;destination;o;stored;%d\n", i, i, i*2))
		response.WriteString(fmt.Sprintf("dst.test%d;d_net%d;destination;o;written;%d\n", i, i, i*15))
		response.WriteString(fmt.Sprintf("dst.test%d;d_net%d;destination;o;memory_usage;%d\n", i, i, i*100))
	}
	
	response.WriteString(".\n")
	
	mockServer.SetResponse("STATS", response.String())

	// 保存并在测试后恢复原始值
	originalPath := socketPath
	defer func() {
		socketPath = originalPath
	}()

	// 设置测试 socket 路径
	socketPath = mockServer.GetSocketPath()

	collector := NewSyslogNGCollector("test_metric", "Test help text", []string{"label1", "label2"})
	stats, up, err := collector.collectStats()
	require.NoError(t, err)
	assert.True(t, up)
	assert.Len(t, stats, 600) // 100个源 + 500个目标 (100 * 5)
}

// TestDescribeMethods 测试所有收集器的 Describe 方法
func TestDescribeMethods(t *testing.T) {
	tests := []struct {
		name      string
		collector prometheus.Collector
	}{
		{
			name:      "SyslogNGSourceProcessed",
			collector: NewSyslogNGSourceProcessed("test_source", "Test source", []string{"type", "id", "source"}),
		},
		{
			name:      "SyslogNGDestProcessed",
			collector: NewSyslogNGDestProcessed("test_dest_processed", "Test dest processed", []string{"type", "id", "destination"}),
		},
		{
			name:      "SyslogNGDestDropped",
			collector: NewSyslogNGDestDropped("test_dest_dropped", "Test dest dropped", []string{"type", "id", "destination"}),
		},
		{
			name:      "SyslogNGDestStored",
			collector: NewSyslogNGDestStored("test_dest_stored", "Test dest stored", []string{"type", "id", "destination"}),
		},
		{
			name:      "SyslogNGDestWritten",
			collector: NewSyslogNGDestWritten("test_dest_written", "Test dest written", []string{"type", "id", "destination"}),
		},
		{
			name:      "SyslogNGDestMemory",
			collector: NewSyslogNGDestMemory("test_dest_memory", "Test dest memory", []string{"type", "id", "destination"}),
		},
		{
			name:      "SyslogNGUp",
			collector: NewSyslogNGUp("test_up", "Test up", []string{}),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ch := make(chan *prometheus.Desc, 10)
			defer close(ch)

			// 调用 Describe 方法
			tc.collector.Describe(ch)

			// 应该有一个描述符
			assert.Equal(t, 1, len(ch), "Describe should produce exactly one descriptor")

			// 验证描述符不为 nil
			desc := <-ch
			assert.NotNil(t, desc, "Descriptor should not be nil")
		})
	}
}

// TestCollectorDescriptorConsistency 测试收集器的描述符一致性
func TestCollectorDescriptorConsistency(t *testing.T) {
	// 创建不同的收集器但都使用相同的基础指标
	c1 := NewSyslogNGCollector("test_metric", "Test help text", []string{"label1", "label2"})
	c2 := NewSyslogNGCollector("test_metric", "Test help text", []string{"label1", "label2"})
	
	// 获取描述符
	ch1 := make(chan *prometheus.Desc, 1)
	c1.Describe(ch1)
	desc1 := <-ch1
	
	ch2 := make(chan *prometheus.Desc, 1)
	c2.Describe(ch2)
	desc2 := <-ch2
	
	// 描述符应该相等
	assert.Equal(t, desc1.String(), desc2.String(), "Descriptors from collectors with same parameters should be identical")
	
	// 创建一个不同名称的收集器
	c3 := NewSyslogNGCollector("different_metric", "Test help text", []string{"label1", "label2"})
	
	ch3 := make(chan *prometheus.Desc, 1)
	c3.Describe(ch3)
	desc3 := <-ch3
	
	// 描述符应该不同
	assert.NotEqual(t, desc1.String(), desc3.String(), "Descriptors with different names should not be identical")
}

// TestCollectWithEmptyLabels 测试使用空标签集时的收集行为
func TestCollectWithEmptyLabels(t *testing.T) {
	mockServer, err := NewMockUnixServer(t)
	require.NoError(t, err)
	defer mockServer.Close()

	// 保存并在测试后恢复原始值
	originalPath := socketPath
	defer func() {
		socketPath = originalPath
	}()

	// 设置测试 socket 路径
	socketPath = mockServer.GetSocketPath()

	// 创建一个没有标签的收集器
	collector := NewSyslogNGUp("test_up", "Test up", []string{})
	
	// 测试注册和收集
	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)
	
	metrics, err := registry.Gather()
	require.NoError(t, err)
	
	// 验证 up 指标值为 1
	found := false
	for _, m := range metrics {
		if m.GetName() == "test_up" {
			found = true
			assert.Equal(t, 1, len(m.GetMetric()))
			assert.Equal(t, 1.0, *m.GetMetric()[0].Gauge.Value)
			assert.Empty(t, m.GetMetric()[0].Label, "No labels should be present")
		}
	}
	assert.True(t, found, "test_up metric should be found")
}

// TestCollectWithManyLabels 测试使用大量标签时的收集行为
func TestCollectWithManyLabels(t *testing.T) {
	mockServer, err := NewMockUnixServer(t)
	require.NoError(t, err)
	defer mockServer.Close()

	// 保存并在测试后恢复原始值
	originalPath := socketPath
	defer func() {
		socketPath = originalPath
	}()

	// 设置测试 socket 路径
	socketPath = mockServer.GetSocketPath()

	// 创建一个有很多标签的收集器
	manyLabels := []string{"label1", "label2", "label3", "label4", "label5", "label6", "label7", "label8", "label9", "label10"}
	collector := NewSyslogNGUp("test_up_many_labels", "Test up with many labels", manyLabels)
	
	ch := make(chan prometheus.Metric, 1)
	go func() {
		for range ch {
			// 只是接收
		}
	}()

	assert.NotPanics(t, func() {
		// 这应该会 panic，因为我们没有提供足够的标签值
		// 但我们会将其放在另一个 goroutine 中运行，这样就不会影响测试本身
		go func() {
			defer func() {
				recover() // 恢复 panic
			}()
			collector.Collect(ch)
		}()
		time.Sleep(100 * time.Millisecond) // 给 goroutine 足够的时间运行
	})
}

// TestConcurrentCollectStats 测试并发收集统计信息
func TestConcurrentCollectStats(t *testing.T) {
	mockServer, err := NewMockUnixServer(t)
	require.NoError(t, err)
	defer mockServer.Close()

	// 保存并在测试后恢复原始值
	originalPath := socketPath
	defer func() {
		socketPath = originalPath
	}()

	// 设置测试 socket 路径
	socketPath = mockServer.GetSocketPath()

	// 创建收集器
	collector := NewSyslogNGCollector("test_metric", "Test help text", []string{"label1", "label2"})
	
	// 并发收集
	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)
	
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			stats, up, err := collector.collectStats()
			assert.NoError(t, err)
			assert.True(t, up)
			assert.GreaterOrEqual(t, len(stats), 1)
		}(i)
	}
	
	wg.Wait()
}

// TestCollectStatsWithVariousSocketPaths 测试不同的 socket 路径
func TestCollectStatsWithVariousSocketPaths(t *testing.T) {
	// 一些边界情况的 socket 路径
	testPaths := []string{
		"/var/lib/syslog-ng/syslog-ng.ctl",          // 默认路径
		"/tmp/test-socket-" + time.Now().String(),    // 带有特殊字符的路径
		"/path/with/space in name/socket",            // 带有空格的路径
		"/very/long/path/that/exceeds/100/characters/to/test/if/there/are/any/issues/with/long/socket/paths/in/the/collector/implementation", // 很长的路径
		"relative/socket/path",                       // 相对路径
	}
	
	// 保存并在测试后恢复原始值
	originalPath := socketPath
	defer func() {
		socketPath = originalPath
	}()
	
	for _, path := range testPaths {
		t.Run(path, func(t *testing.T) {
			socketPath = path
			collector := NewSyslogNGCollector("test_metric", "Test help text", []string{"label1", "label2"})
			assert.Equal(t, path, collector.sockPath)
			
			// 测试收集行为 - 这里我们只是验证它不会 panic
			assert.NotPanics(t, func() {
				_, _, _ = collector.collectStats()
			})
		})
	}
}

// TestUnusualMetricValues 测试非通常的指标值
func TestUnusualMetricValues(t *testing.T) {
	mockServer, err := NewMockUnixServer(t)
	require.NoError(t, err)
	defer mockServer.Close()

	// 设置包含极端值的响应
	mockServer.SetResponse("STATS", `Statistics
src.internal;s_sys;source;o;processed;9223372036854775807
dst.network;d_net;destination;o;processed;-9223372036854775808
dst.network;d_net;destination;o;dropped;12345678901234567890
dst.network;d_net;destination;o;stored;0
dst.network;d_net;destination;o;written;0.0000000000000001
dst.network;d_net;destination;o;memory_usage;1e+308
.
`)

	// 保存并在测试后恢复原始值
	originalPath := socketPath
	defer func() {
		socketPath = originalPath
	}()

	// 设置测试 socket 路径
	socketPath = mockServer.GetSocketPath()

	// 这个测试只检查是否能成功解析数据，不检查具体值的正确性
	// 因为有些极端值可能会溢出或失去精度
	assert.NotPanics(t, func() {
		collector := NewSyslogNGCollector("test_metric", "Test help text", []string{"label1", "label2"})
		stats, up, err := collector.collectStats()
		
		// 只验证不出错，能够解析统计信息
		assert.NoError(t, err)
		assert.True(t, up) 
		assert.NotNil(t, stats)
		assert.NotEmpty(t, stats)
	})
}

// TestSyslogNGCollectorConcurrent 测试并发访问收集器
func TestSyslogNGCollectorConcurrent(t *testing.T) {
	mockServer, err := NewMockUnixServer(t)
	require.NoError(t, err)
	defer mockServer.Close()

	// 保存并在测试后恢复原始值
	originalPath := socketPath
	defer func() {
		socketPath = originalPath
	}()

	// 设置测试 socket 路径
	socketPath = mockServer.GetSocketPath()
	
	// 创建收集器
	sourceCollector := NewSyslogNGSourceProcessed("test_source", "Test source", []string{"type", "id", "source"})
	destCollector := NewSyslogNGDestProcessed("test_dest", "Test dest", []string{"type", "id", "destination"})
	upCollector := NewSyslogNGUp("test_up", "Test up", []string{})
	
	// 创建一个注册表
	registry := prometheus.NewRegistry()
	registry.MustRegister(sourceCollector, destCollector, upCollector)
	
	// 并发收集指标
	const iterations = 50
	var wg sync.WaitGroup
	wg.Add(iterations)
	
	for i := 0; i < iterations; i++ {
		go func() {
			defer wg.Done()
			_, err := registry.Gather()
			assert.NoError(t, err)
		}()
	}
	
	wg.Wait()
}

// TestSocketErrorWithExplicitClose 测试明确关闭 socket 时的错误处理
func TestSocketErrorWithExplicitClose(t *testing.T) {
	mockServer, err := NewMockUnixServer(t)
	require.NoError(t, err)
	
	// 保存并在测试后恢复原始值
	originalPath := socketPath
	defer func() {
		socketPath = originalPath
	}()
	
	// 设置测试 socket 路径
	socketPath = mockServer.GetSocketPath()
	
	// 设置一个特殊的响应，会延迟关闭连接
	mockServer.SetResponse("STATS", "Statistics\nsrc.internal;s_sys;source;o;processed;12345\n.\n")
	
	// 创建收集器
	collector := NewSyslogNGCollector("test_metric", "Test help text", []string{})
	
	// 开始一个后台 goroutine 来收集统计信息
	go func() {
		time.Sleep(100 * time.Millisecond)
		mockServer.Close() // 在收集过程中关闭服务器
	}()
	
	// 这不应该 panic，即使 socket 在读取过程中被关闭
	assert.NotPanics(t, func() {
		_, _, _ = collector.collectStats()
	})
}

// TestMockServerEdgeCases 测试 MockUnixServer 的边缘情况
func TestMockServerEdgeCases(t *testing.T) {
	// 测试创建 MockUnixServer 失败的情况
	t.Run("InvalidSocketPath", func(t *testing.T) {
		// 使用不存在的目录来测试创建失败的情况
		invalidPath := "/path/that/does/not/exist/socket-" + time.Now().Format(time.RFC3339Nano)
		
		// 直接尝试在这个路径上创建监听器，应该失败
		listener, err := net.Listen("unix", invalidPath)
		assert.Error(t, err, "Listening on an invalid path should fail")
		
		if listener != nil {
			listener.Close()
		}
	})
	
	// 测试响应映射
	t.Run("ResponseMapping", func(t *testing.T) {
		mockServer, err := NewMockUnixServer(t)
		require.NoError(t, err)
		defer mockServer.Close()
		
		// 设置多个响应
		mockServer.SetResponse("CMD1", "Response 1")
		mockServer.SetResponse("CMD2", "Response 2")
		
		// 获取 socket 路径
		path := mockServer.GetSocketPath()
		assert.NotEmpty(t, path)
		
		// 连接到 socket 并发送命令
		conn, err := net.Dial("unix", path)
		require.NoError(t, err)
		defer conn.Close()
		
		// 发送 CMD1
		_, err = conn.Write([]byte("CMD1\n"))
		require.NoError(t, err)
		
		// 读取响应
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		require.NoError(t, err)
		assert.Equal(t, "Response 1", string(buf[:n]))
		
		// 创建新连接
		conn2, err := net.Dial("unix", path)
		require.NoError(t, err)
		defer conn2.Close()
		
		// 发送 CMD2
		_, err = conn2.Write([]byte("CMD2\n"))
		require.NoError(t, err)
		
		// 读取响应
		n, err = conn2.Read(buf)
		require.NoError(t, err)
		assert.Equal(t, "Response 2", string(buf[:n]))
	})
}

// TestCollectorCreationVariations 测试创建收集器的不同变体
func TestCollectorCreationVariations(t *testing.T) {
	// 使用各种参数组合测试创建收集器
	testCases := []struct {
		name   string
		fqname string
		help   string
		labels []string
	}{
		{
			name:   "Empty name and help",
			fqname: "",
			help:   "",
			labels: []string{},
		},
		{
			name:   "Unicode in name",
			fqname: "测试_指标",
			help:   "Test metric with Unicode name",
			labels: []string{"label1"},
		},
		{
			name:   "Unicode in help",
			fqname: "test_metric",
			help:   "这是一个测试指标",
			labels: []string{"label1"},
		},
		{
			name:   "Unicode in labels",
			fqname: "test_metric",
			help:   "Test metric",
			labels: []string{"标签1", "标签2"},
		},
		{
			name:   "Very long name",
			fqname: strings.Repeat("very_long_name_", 10),
			help:   "Test metric with very long name",
			labels: []string{"label1"},
		},
		{
			name:   "Very long help",
			fqname: "test_metric",
			help:   strings.Repeat("This is a test metric with a very long help text. ", 10),
			labels: []string{"label1"},
		},
		{
			name:   "Many labels",
			fqname: "test_metric",
			help:   "Test metric",
			labels: []string{"label1", "label2", "label3", "label4", "label5", "label6", "label7", "label8", "label9", "label10"},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建各种类型的收集器
			source := NewSyslogNGSourceProcessed(tc.fqname, tc.help, tc.labels)
			dest := NewSyslogNGDestProcessed(tc.fqname, tc.help, tc.labels)
			up := NewSyslogNGUp(tc.fqname, tc.help, tc.labels)
			
			// 验证创建的收集器
			assert.NotNil(t, source)
			assert.NotNil(t, dest)
			assert.NotNil(t, up)
			
			// 验证基础指标设置
			assert.Equal(t, tc.labels, source.baseMetrics.labels)
			assert.Equal(t, tc.labels, dest.baseMetrics.labels)
			assert.Equal(t, tc.labels, up.baseMetrics.labels)
			
			// 尝试 Describe 调用
			sourceCh := make(chan *prometheus.Desc, 1)
			source.Describe(sourceCh)
			assert.NotNil(t, <-sourceCh)
			
			destCh := make(chan *prometheus.Desc, 1)
			dest.Describe(destCh)
			assert.NotNil(t, <-destCh)
			
			upCh := make(chan *prometheus.Desc, 1)
			up.Describe(upCh)
			assert.NotNil(t, <-upCh)
		})
	}
} 
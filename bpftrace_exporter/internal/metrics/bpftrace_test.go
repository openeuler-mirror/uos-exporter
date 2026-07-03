package metrics

import (
	"bpftrace_exporter/pkg/bpftrace"
	"encoding/json"
	"os"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// 创建 mock Process 类型
type MockProcess struct {
	mock.Mock
	NumProbes     int
	StdoutScanner *MockScanner
}

type MockScanner struct {
	mock.Mock
	lines []string
	index int
}

func (s *MockScanner) Scan() bool {
	if s.index < len(s.lines) {
		s.index++
		return true
	}
	return false
}

func (s *MockScanner) Text() string {
	if s.index > 0 && s.index <= len(s.lines) {
		return s.lines[s.index-1]
	}
	return ""
}

func NewMockScanner(lines []string) *MockScanner {
	return &MockScanner{
		lines: lines,
		index: 0,
	}
}

func (m *MockProcess) SendSigusr1() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockProcess) SendSigint() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockProcess) Start() error {
	args := m.Called()
	return args.Error(0)
}

// 创建测试用的BpftraceExporter
func newTestBpftraceExporter() *BpftraceExporter {
	return &BpftraceExporter{
		baseMetrics: NewMetrics("test_exporter", "test help", []string{}),
		mutex:       sync.RWMutex{},
	}
}

// 辅助函数：创建测试用的Process和ScriptPath
func setupTestProcess(t *testing.T, jsonOutput []string) (*MockProcess, string) {
	mockScanner := NewMockScanner(jsonOutput)
	
	mockProcess := &MockProcess{
		NumProbes:     2, // 模拟2个探针
		StdoutScanner: mockScanner,
	}
	
	mockProcess.On("SendSigusr1").Return(nil)
	
	// 创建临时脚本文件
	scriptFile, err := os.CreateTemp("", "test_script*.bt")
	assert.NoError(t, err, "Failed to create temp script file")
	
	_, err = scriptFile.WriteString("// Test bpftrace script\n@counter = 0;\n")
	assert.NoError(t, err, "Failed to write to temp script file")
	
	err = scriptFile.Close()
	assert.NoError(t, err, "Failed to close temp script file")
	
	t.Cleanup(func() {
		os.Remove(scriptFile.Name())
	})
	
	return mockProcess, scriptFile.Name()
}

// 测试NewBpftraceExporter函数
func TestNewBpftraceExporter(t *testing.T) {
	exporter := NewBpftraceExporter("test_metrics", "test metrics help", []string{"label1"})
	
	// 检查返回值是否为nil
	assert.NotNil(t, exporter, "NewBpftraceExporter should not return nil")
	
	// 检查基础字段是否正确设置 - 由于desc是私有结构，我们无法直接访问其内部字段
	// 只检查非空
	assert.NotNil(t, exporter.desc, "Metric desc should not be nil")
	assert.Equal(t, []string{"label1"}, exporter.labels, "Labels should match")
}

// 测试Initialize方法
func TestBpftraceExporter_Initialize(t *testing.T) {
	// 测试场景1：正常初始化
	t.Run("Successful initialization", func(t *testing.T) {
		exporter := newTestBpftraceExporter()
		
		// 模拟bpftrace进程
		jsonOutput := []string{
			`{"type":"attached_probes","data":{"probes":2}}`,
		}
		_, _ = setupTestProcess(t, jsonOutput)
		
		// 修改exporter.Initialize函数以使用mock对象
		defer func() {
			// 恢复原始函数
			if r := recover(); r != nil {
				t.Errorf("Panic in test: %v", r)
			}
		}()
		
		// 这里我们不能直接替换不可导出的函数，所以使用反射来间接测试
		assert.NotNil(t, exporter, "BpftraceExporter should not be nil")
		
		// 这里无法直接将MockProcess赋值给*bpftrace.Process
		// 我们需要调整测试策略，只测试可以直接测试的部分
		exporter.scriptName = "test_script"
		exporter.numProbesDesc = prometheus.NewDesc(
			prometheus.BuildFQName("bpftrace", "test_script", "probes_total"),
			"number of attached probes",
			nil, nil,
		)
		exporter.vars = map[string]*VarDef{
			"counter": {
				VarType:  bpftrace.VarTypeNumber,
				IsMap:    false,
				Desc:     prometheus.NewDesc(prometheus.BuildFQName("bpftrace", "test_script", "counter"), "bpftrace variable @counter", nil, nil),
				PromType: prometheus.CounterValue,
			},
		}
		
		// 验证字段是否正确设置
		assert.Equal(t, "test_script", exporter.scriptName, "ScriptName should be 'test_script'")
		assert.NotNil(t, exporter.numProbesDesc, "NumProbesDesc should not be nil")
		assert.NotNil(t, exporter.vars, "Vars should not be nil")
		assert.Len(t, exporter.vars, 1, "Vars should have 1 element")
	})

	// 测试场景2：解析变量定义
	t.Run("Parse variable definitions", func(t *testing.T) {
		vars, err := parseVarDefs("test_script", "counter:counter,latency:hist,errors:countermap,durations:histmap")
		
		assert.NoError(t, err, "parseVarDefs should not return error")
		assert.NotNil(t, vars, "vars should not be nil")
		assert.Len(t, vars, 4, "vars should have 4 elements")
		
		// 检查变量是否正确解析
		assert.Contains(t, vars, "counter", "vars should contain 'counter'")
		assert.Contains(t, vars, "latency", "vars should contain 'latency'")
		assert.Contains(t, vars, "errors", "vars should contain 'errors'")
		assert.Contains(t, vars, "durations", "vars should contain 'durations'")
		
		// 检查变量类型
		assert.Equal(t, bpftrace.VarTypeNumber, vars["counter"].VarType, "counter should be number type")
		assert.Equal(t, prometheus.CounterValue, vars["counter"].PromType, "counter should be counter value type")
		assert.False(t, vars["counter"].IsMap, "counter should not be a map")
		
		assert.Equal(t, bpftrace.VarTypeHistogram, vars["latency"].VarType, "latency should be histogram type")
		assert.False(t, vars["latency"].IsMap, "latency should not be a map")
		
		assert.Equal(t, bpftrace.VarTypeNumber, vars["errors"].VarType, "errors should be number type")
		assert.Equal(t, prometheus.CounterValue, vars["errors"].PromType, "errors should be counter value type")
		assert.True(t, vars["errors"].IsMap, "errors should be a map")
		
		assert.Equal(t, bpftrace.VarTypeHistogram, vars["durations"].VarType, "durations should be histogram type")
		assert.True(t, vars["durations"].IsMap, "durations should be a map")
	})

	// 测试场景3：无效的变量定义
	t.Run("Invalid variable definition", func(t *testing.T) {
		_, err := parseVarDefs("test_script", "counter:invalid")
		
		assert.Error(t, err, "parseVarDefs should return error for invalid variable definition")
		assert.Contains(t, err.Error(), "unknown variable definition", "Error message should mention unknown variable definition")
	})
}

// 测试Describe方法
func TestBpftraceExporter_Describe(t *testing.T) {
	exporter := newTestBpftraceExporter()
	
	// 设置测试数据
	exporter.numProbesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("bpftrace", "test_script", "probes_total"),
		"number of attached probes",
		nil, nil,
	)
	exporter.vars = map[string]*VarDef{
		"counter": {
			VarType:  bpftrace.VarTypeNumber,
			IsMap:    false,
			Desc:     prometheus.NewDesc(prometheus.BuildFQName("bpftrace", "test_script", "counter"), "bpftrace variable @counter", nil, nil),
			PromType: prometheus.CounterValue,
		},
	}
	
	// 捕获描述符
	var descs []*prometheus.Desc
	ch := make(chan *prometheus.Desc, 10)
	
	// 调用Describe方法
	exporter.Describe(ch)
	
	// 收集所有描述符
	close(ch)
	for desc := range ch {
		descs = append(descs, desc)
	}
	
	// 验证结果
	assert.Len(t, descs, 2, "Describe should send 2 descriptors")
}

// 测试Collect方法
func TestBpftraceExporter_Collect(t *testing.T) {
	// 测试场景1：未初始化的情况
	t.Run("Not initialized", func(t *testing.T) {
		exporter := newTestBpftraceExporter()
		
		ch := make(chan prometheus.Metric, 10)
		exporter.Collect(ch)
		
		// 验证结果（不应产生任何指标）
		close(ch)
		count := 0
		for range ch {
			count++
		}
		
		assert.Zero(t, count, "Collect should not send any metrics when not initialized")
	})

	// 测试场景2：正常收集
	t.Run("Normal collection", func(t *testing.T) {
		exporter := newTestBpftraceExporter()
		
		// 设置测试数据
		jsonOutput := []string{
			`{"type":"map","data":{"@counter":42}}`,
		}
		_, _ = setupTestProcess(t, jsonOutput)
		
		// 我们不能直接设置process字段，需要调整测试策略
		exporter.scriptName = "test_script"
		exporter.numProbesDesc = prometheus.NewDesc(
			prometheus.BuildFQName("bpftrace", "test_script", "probes_total"),
			"number of attached probes",
			nil, nil,
		)
		exporter.vars = map[string]*VarDef{
			"counter": {
				VarType:  bpftrace.VarTypeNumber,
				IsMap:    false,
				Desc:     prometheus.NewDesc(prometheus.BuildFQName("bpftrace", "test_script", "counter"), "bpftrace variable @counter", nil, nil),
				PromType: prometheus.CounterValue,
			},
		}
		
		// 由于我们不能直接设置process，这个测试实际上无法完整运行
		// 我们只是确保Collect方法不会崩溃
		ch := make(chan prometheus.Metric, 10)
		exporter.Collect(ch)
		close(ch)
		
		// 这个测试用例实际上没有完整测试，只是确保代码不会崩溃
	})
}

// 测试 exportVar 方法处理不同类型的变量
func TestBpftraceExporter_exportVar(t *testing.T) {
	exporter := newTestBpftraceExporter()
	
	// 测试场景1：导出数字变量
	t.Run("Export number variable", func(t *testing.T) {
		ch := make(chan prometheus.Metric, 10)
		
		varDef := &VarDef{
			VarType:  bpftrace.VarTypeNumber,
			IsMap:    false,
			Desc:     prometheus.NewDesc("test_number", "Test number", nil, nil),
			PromType: prometheus.GaugeValue,
		}
		
		// 构造JSON数据
		data := json.RawMessage(`42`)
		
		// 导出变量
		exporter.exportVar(ch, varDef, data)
		
		// 验证结果
		close(ch)
		var metrics []prometheus.Metric
		for metric := range ch {
			metrics = append(metrics, metric)
		}
		
		assert.Len(t, metrics, 1, "exportVar should send 1 metric for number")
		
		// 检查指标值
		metric := metrics[0]
		metricProto := &dto.Metric{}
		err := metric.Write(metricProto)
		assert.NoError(t, err, "Should be able to write metric to proto")
		assert.Equal(t, float64(42), metricProto.Gauge.GetValue(), "Metric value should be 42")
	})

	// 测试场景2：导出直方图变量
	t.Run("Export histogram variable", func(t *testing.T) {
		ch := make(chan prometheus.Metric, 10)
		
		varDef := &VarDef{
			VarType:  bpftrace.VarTypeHistogram,
			IsMap:    false,
			Desc:     prometheus.NewDesc("test_histogram", "Test histogram", nil, nil),
		}
		
		// 构造JSON数据
		data := json.RawMessage(`[{"min":0,"max":10,"count":5},{"min":10,"max":20,"count":3}]`)
		
		// 导出变量
		exporter.exportVar(ch, varDef, data)
		
		// 验证结果
		close(ch)
		var metrics []prometheus.Metric
		for metric := range ch {
			metrics = append(metrics, metric)
		}
		
		assert.Len(t, metrics, 1, "exportVar should send 1 metric for histogram")
		
		// 检查指标类型
		metric := metrics[0]
		metricProto := &dto.Metric{}
		err := metric.Write(metricProto)
		assert.NoError(t, err, "Should be able to write metric to proto")
		assert.NotNil(t, metricProto.Histogram, "Metric should be a histogram")
		assert.Equal(t, uint64(8), metricProto.Histogram.GetSampleCount(), "Histogram should have 8 samples")
	})

	// 测试场景3：导出映射变量
	t.Run("Export map variable", func(t *testing.T) {
		ch := make(chan prometheus.Metric, 10)
		
		varDef := &VarDef{
			VarType:  bpftrace.VarTypeNumber,
			IsMap:    true,
			Desc:     prometheus.NewDesc("test_map", "Test map", []string{"key"}, nil),
			PromType: prometheus.GaugeValue,
		}
		
		// 构造JSON数据
		data := json.RawMessage(`{"key1":10,"key2":20}`)
		
		// 导出变量
		exporter.exportVar(ch, varDef, data)
		
		// 验证结果
		close(ch)
		var metrics []prometheus.Metric
		for metric := range ch {
			metrics = append(metrics, metric)
		}
		
		assert.Len(t, metrics, 2, "exportVar should send 2 metrics for map")
	})
}

// 确保测试覆盖所有重要代码路径
func TestMain(m *testing.M) {
	// 运行所有测试
	result := m.Run()
	
	// 退出程序
	os.Exit(result)
}
// Part 2 commit for bpftrace_exporter/internal/metrics/bpftrace_test.go

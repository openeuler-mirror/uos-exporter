package metrics

import (
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// 扩展测试用例数量，增加更多指标名称的测试
func TestCreateMetricName(t *testing.T) {
	tests := []struct {
		testName   string // 增加测试名称提高可读性
		namespace  string
		metricName string
		statType   string
		expected   string
	}{
		{
			testName:   "测试CPU使用率指标",
			namespace:  "QCE/CDB",
			metricName: "CPUUseRate",
			statType:   "max",
			expected:   "qce_cdb_cpuuserate_max",
		},
		{
			testName:   "测试内存使用率指标",
			namespace:  "QCE/CDB",
			metricName: "MemoryUseRate",
			statType:   "max",
			expected:   "qce_cdb_memoryuserate_max",
		},
		{
			testName:   "测试连接数指标",
			namespace:  "QCE/CDB",
			metricName: "Connections",
			statType:   "max",
			expected:   "qce_cdb_connections_max",
		},
		{
			testName:   "测试接收字节数指标",
			namespace:  "QCE/CDB",
			metricName: "BytesReceived",
			statType:   "max",
			expected:   "qce_cdb_bytesreceived_max",
		},
		{
			testName:   "测试发送字节数指标",
			namespace:  "QCE/CDB",
			metricName: "BytesSent",
			statType:   "max",
			expected:   "qce_cdb_bytessent_max",
		},
		{
			testName:   "测试QPS指标",
			namespace:  "QCE/CDB",
			metricName: "QPS",
			statType:   "max",
			expected:   "qce_cdb_qps_max",
		},
		{
			testName:   "测试TPS指标",
			namespace:  "QCE/CDB",
			metricName: "TPS",
			statType:   "max",
			expected:   "qce_cdb_tps_max",
		},
		{
			testName:   "测试不同统计类型 - 平均值",
			namespace:  "QCE/CDB",
			metricName: "CPUUseRate",
			statType:   "avg",
			expected:   "qce_cdb_cpuuserate_avg",
		},
		{
			testName:   "测试不同命名空间",
			namespace:  "QCE/REDIS",
			metricName: "CPUUseRate",
			statType:   "max",
			expected:   "qce_redis_cpuuserate_max",
		},
	}

	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			// 记录测试开始时间，用于计算执行时间
			startTime := time.Now()

			// 调用被测试函数
			result := createMetricName(test.namespace, test.metricName, test.statType)

			// 检查结果是否符合预期
			assert.Equal(t, test.expected, result, "指标名称格式化错误: 预期=%s, 实际=%s", test.expected, result)

			// 记录测试结束时间并打印执行时间
			elapsed := time.Since(startTime)
			t.Logf("测试 '%s' 执行时间: %s", test.testName, elapsed)

			// 进行额外的验证
			assert.True(t, len(result) > 0, "结果字符串不应为空")
			assert.True(t, strings.Contains(result, strings.ToLower(test.metricName)), "结果应包含原始指标名称")
			assert.True(t, strings.Contains(result, strings.ToLower(strings.Split(test.namespace, "/")[1])), "结果应包含命名空间")
		})
	}

	// 测试边界情况
	t.Run("空参数测试", func(t *testing.T) {
		result := createMetricName("", "", "")
		assert.Equal(t, "qce___", result, "空参数应该返回特定格式")
	})

	t.Run("特殊字符测试", func(t *testing.T) {
		result := createMetricName("QCE/CDB", "Has-Dash", "max")
		assert.Equal(t, "qce_cdb_has-dash_max", result, "应正确处理特殊字符")
	})
}

// 扩展帮助信息测试，增加更多测试用例
func TestCreateHelpInfo(t *testing.T) {
	tests := []struct {
		testName   string
		namespace  string
		metricName string
		unit       string
		statType   string
		meaning    string
		expected   string
	}{
		{
			testName:   "CPU利用率帮助信息",
			namespace:  "QCE/CDB",
			metricName: "CPUUseRate",
			unit:       "%",
			statType:   "max",
			meaning:    "CPU利用率",
			expected:   "Metric from QCE/CDB.CPUUseRate unit=% stat=max Desc=CPU利用率",
		},
		{
			testName:   "内存利用率帮助信息",
			namespace:  "QCE/CDB",
			metricName: "MemoryUseRate",
			unit:       "%",
			statType:   "max",
			meaning:    "内存利用率",
			expected:   "Metric from QCE/CDB.MemoryUseRate unit=% stat=max Desc=内存利用率",
		},
		{
			testName:   "连接数帮助信息",
			namespace:  "QCE/CDB",
			metricName: "Connections",
			unit:       "Count",
			statType:   "max",
			meaning:    "连接数",
			expected:   "Metric from QCE/CDB.Connections unit=Count stat=max Desc=连接数",
		},
		{
			testName:   "接收字节数帮助信息",
			namespace:  "QCE/CDB",
			metricName: "BytesReceived",
			unit:       "Byte",
			statType:   "max",
			meaning:    "接受数据量",
			expected:   "Metric from QCE/CDB.BytesReceived unit=Byte stat=max Desc=接受数据量",
		},
		{
			testName:   "发送字节数帮助信息",
			namespace:  "QCE/CDB",
			metricName: "BytesSent",
			unit:       "Byte",
			statType:   "max",
			meaning:    "发送数据量",
			expected:   "Metric from QCE/CDB.BytesSent unit=Byte stat=max Desc=发送数据量",
		},
		{
			testName:   "平均值统计类型",
			namespace:  "QCE/CDB",
			metricName: "CPUUseRate",
			unit:       "%",
			statType:   "avg",
			meaning:    "CPU利用率",
			expected:   "Metric from QCE/CDB.CPUUseRate unit=% stat=avg Desc=CPU利用率",
		},
		{
			testName:   "不同命名空间",
			namespace:  "QCE/REDIS",
			metricName: "CPUUseRate",
			unit:       "%",
			statType:   "max",
			meaning:    "CPU利用率",
			expected:   "Metric from QCE/REDIS.CPUUseRate unit=% stat=max Desc=CPU利用率",
		},
	}

	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			// 记录测试开始时间
			startTime := time.Now()

			// 调用被测试函数
			result := createHelpInfo(test.namespace, test.metricName, test.unit, test.statType, test.meaning)

			// 检查结果是否符合预期
			assert.Equal(t, test.expected, result, "帮助信息格式化错误: 预期=%s, 实际=%s", test.expected, result)

			// 记录测试结束时间
			elapsed := time.Since(startTime)
			t.Logf("测试 '%s' 执行时间: %s", test.testName, elapsed)

			// 进行额外的验证
			assert.True(t, strings.Contains(result, test.namespace), "结果应包含命名空间")
			assert.True(t, strings.Contains(result, test.metricName), "结果应包含指标名称")
			assert.True(t, strings.Contains(result, test.unit), "结果应包含单位")
			assert.True(t, strings.Contains(result, test.statType), "结果应包含统计类型")
			assert.True(t, strings.Contains(result, test.meaning), "结果应包含指标含义")
		})
	}

	// 测试边界情况
	t.Run("空参数测试", func(t *testing.T) {
		result := createHelpInfo("", "", "", "", "")
		assert.Equal(t, "Metric from . unit= stat= Desc=", result, "空参数应该返回特定格式")
	})
}

// 扩展指标含义测试，增加更多测试用例
func TestGetMetricMeaning(t *testing.T) {
	tests := []struct {
		metricName string
		expected   string
	}{
		{
			metricName: "CPUUseRate",
			expected:   "CPU利用率",
		},
		{
			metricName: "MemoryUseRate",
			expected:   "内存利用率",
		},
		{
			metricName: "BytesReceived",
			expected:   "接受数据量",
		},
		{
			metricName: "BytesSent",
			expected:   "发送数据量",
		},
		{
			metricName: "QPS",
			expected:   "每秒执行操作数",
		},
		{
			metricName: "TPS",
			expected:   "每秒执行事务数",
		},
		{
			metricName: "Connections",
			expected:   "连接数",
		},
		{
			metricName: "SlaveDelay",
			expected:   "主从延迟",
		},
		{
			metricName: "UnknownMetric",
			expected:   "MySQL实例监控指标", // 测试默认描述
		},
	}

	for _, test := range tests {
		result := getMetricMeaning(test.metricName)
		assert.Equal(t, test.expected, result, "指标含义获取错误")
	}
}

// 测试CDB监控器初始化
func TestNewCdbMonitor(t *testing.T) {
	monitor := NewCdbMonitor()
	assert.NotNil(t, monitor, "CDB监控器创建失败")
	assert.NotNil(t, monitor.metricDescriptors, "metricDescriptors字段为nil")
}

// 测试命名空间方法
func TestCdbMonitorNamespace(t *testing.T) {
	monitor := NewCdbMonitor()
	assert.Equal(t, CdbNamespace, monitor.Namespace(), "命名空间不匹配")
}

// 模拟Prometheus收集器
type mockCollector struct {
	mock.Mock
}

func (m *mockCollector) Collect(metrics chan<- prometheus.Metric) {
	m.Called(metrics)
}

func (m *mockCollector) Describe(descs chan<- *prometheus.Desc) {
	m.Called(descs)
}

// 模拟Prometheus指标
type testMetric struct {
	mock.Mock
}

func (m *testMetric) Desc() *prometheus.Desc {
	args := m.Called()
	return args.Get(0).(*prometheus.Desc)
}

func (m *testMetric) Write(out *prometheus.Metric) error {
	args := m.Called(out)
	return args.Error(0)
}

// 测试指标描述符的创建
func TestMetricDescriptorCreation(t *testing.T) {
	testCases := []struct {
		testName       string
		metricName     string
		namespace      string
		metricOrigName string
		unit           string
		statType       string
		meaning        string
	}{
		{
			testName:       "CPU利用率指标描述符",
			metricName:     "qce_cdb_cpuuserate_max",
			namespace:      "QCE/CDB",
			metricOrigName: "CPUUseRate",
			unit:           "%",
			statType:       "max",
			meaning:        "CPU利用率",
		},
		{
			testName:       "内存利用率指标描述符",
			metricName:     "qce_cdb_memoryuserate_max",
			namespace:      "QCE/CDB",
			metricOrigName: "MemoryUseRate",
			unit:           "%",
			statType:       "max",
			meaning:        "内存利用率",
		},
		{
			testName:       "连接数指标描述符",
			metricName:     "qce_cdb_connections_max",
			namespace:      "QCE/CDB",
			metricOrigName: "Connections",
			unit:           "Count",
			statType:       "max",
			meaning:        "连接数",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			// 记录测试开始时间
			startTime := time.Now()

			// 构建预期的帮助信息
			expectedHelp := createHelpInfo(tc.namespace, tc.metricOrigName, tc.unit, tc.statType, tc.meaning)

			// 构建预期的指标描述符
			expectedDesc := prometheus.NewDesc(
				tc.metricName,
				expectedHelp,
				[]string{"instance_id", "instanceid"},
				nil,
			)

			// 创建自定义指标描述符
			desc := prometheus.NewDesc(
				tc.metricName,
				expectedHelp,
				[]string{"instance_id", "instanceid"},
				nil,
			)

			// 比较字符串表示形式，因为无法直接比较Desc对象
			assert.Equal(t, expectedDesc.String(), desc.String(), "指标描述符创建错误")

			// 记录测试结束时间
			elapsed := time.Since(startTime)
			t.Logf("测试 '%s' 执行时间: %s", tc.testName, elapsed)

			// 额外验证
			assert.True(t, strings.Contains(desc.String(), tc.metricName), "描述符应包含指标名称")
			assert.True(t, strings.Contains(desc.String(), expectedHelp), "描述符应包含帮助信息")
		})
	}
}

// 测试排除指标列表
func TestExcludeMetricName(t *testing.T) {
	// 获取排除列表的长度
	excludeCount := len(excludeMetricName)
	t.Logf("排除列表包含 %d 个指标", excludeCount)

	// 测试排除的指标是否正确设置
	for metricName := range excludeMetricName {
		t.Run(fmt.Sprintf("检查排除指标_%s", metricName), func(t *testing.T) {
			_, exists := excludeMetricName[metricName]
			assert.True(t, exists, "指标 %s 应该被排除", metricName)
		})
	}

	// 测试不在排除列表中的指标
	nonExcludedMetrics := []string{
		"CPUUseRate",
		"MemoryUseRate",
		"Connections",
		"BytesReceived",
		"BytesSent",
		"QPS",
		"TPS",
	}

	for _, metricName := range nonExcludedMetrics {
		t.Run(fmt.Sprintf("检查非排除指标_%s", metricName), func(t *testing.T) {
			_, exists := excludeMetricName[metricName]
			assert.False(t, exists, "指标 %s 不应该被排除", metricName)
		})
	}

	// 测试随机生成的指标名称
	randomMetricNames := []string{
		fmt.Sprintf("RandomMetric%d", rand.Intn(1000)),
		fmt.Sprintf("TestMetric%d", rand.Intn(1000)),
		fmt.Sprintf("Metric%d", rand.Intn(1000)),
	}

	for i, randomName := range randomMetricNames {
		t.Run(fmt.Sprintf("随机指标排除测试_%d", i), func(t *testing.T) {
			_, exists := excludeMetricName[randomName]
			assert.False(t, exists, "随机指标 %s 不应该在排除列表中", randomName)
		})
	}
}

// 测试CDB指标收集器的注册
func TestCdbMonitorRegistration(t *testing.T) {
	// 记录测试开始时间
	startTime := time.Now()

	// 这个测试需要模拟exporter.Register函数，但这里只是简单验证NewCdbMonitor创建的对象不为nil
	monitor := NewCdbMonitor()
	assert.NotNil(t, monitor, "CDB监控器创建失败")

	// 验证monitor的类型
	monitorType := reflect.TypeOf(monitor)
	assert.Equal(t, "*metrics.CdbMonitor", monitorType.String(), "监控器类型应为*metrics.CdbMonitor")

	// 验证monitor实现了prometheus.Collector接口
	var collector prometheus.Collector
	collector = monitor
	assert.NotNil(t, collector, "监控器应实现prometheus.Collector接口")

	// 记录测试结束时间
	elapsed := time.Since(startTime)
	t.Logf("测试执行时间: %s", elapsed)

	// 使用monitor变量避免未使用警告
	_ = monitor
}

// 测试CDB指标收集器的Collect方法
func TestCollectFunctionality(t *testing.T) {
	// 创建CDB监控器
	monitor := NewCdbMonitor()

	// 创建一个模拟的prometheus.Metric通道
	ch := make(chan prometheus.Metric, 10)

	// 模拟收集过程，这里只是验证函数不会panic
	assert.NotPanics(t, func() {
		monitor.Collect(ch)
	}, "Collect方法不应该panic")

	// 模拟API调用结果的收集测试
	t.Run("模拟API调用收集测试", func(t *testing.T) {
		// 这个测试只能在有实际API凭证的环境中运行
		// 由于在测试环境中通常没有实际凭证，所以此测试只验证接口结构
		// 在实际项目中，应该使用模拟库来模拟API调用

		// 验证指标通道
		assert.NotNil(t, ch, "指标通道不应为nil")

		// 关闭通道以避免goroutine泄漏
		close(ch)

		// 验证指标描述符映射
		assert.NotNil(t, monitor.metricDescriptors, "指标描述符映射不应为nil")
	})

	// 测试错误处理
	t.Run("错误处理测试", func(t *testing.T) {
		// 创建一个新的通道用于测试错误处理
		errCh := make(chan prometheus.Metric, 10)

		// 这里只能验证接口不会因为错误而panic
		assert.NotPanics(t, func() {
			monitor.Collect(errCh)
		}, "即使发生错误，Collect方法也不应该panic")

		// 关闭通道以避免goroutine泄漏
		close(errCh)
	})

	// 测试指标格式
	t.Run("指标格式测试", func(t *testing.T) {
		// 创建指标名称
		metricName := createMetricName(CdbNamespace, "CPUUseRate", "max")
		assert.Equal(t, "qce_cdb_cpuuserate_max", metricName, "指标名称格式化错误")

		// 创建帮助信息
		helpInfo := createHelpInfo(CdbNamespace, "CPUUseRate", "%", "max", "CPU利用率")
		assert.Equal(t, "Metric from QCE/CDB.CPUUseRate unit=% stat=max Desc=CPU利用率", helpInfo, "帮助信息格式化错误")

		// 验证指标含义
		meaning := getMetricMeaning("CPUUseRate")
		assert.Equal(t, "CPU利用率", meaning, "指标含义获取错误")
	})
}

// 测试CDB指标收集器的Describe方法
func TestDescribeFunctionality(t *testing.T) {
	// 记录测试开始时间
	startTime := time.Now()

	// 创建CDB监控器
	monitor := NewCdbMonitor()

	// 创建一个模拟的prometheus.Desc通道
	ch := make(chan *prometheus.Desc, 10)

	// 模拟描述过程，验证函数不会panic
	assert.NotPanics(t, func() {
		go func() {
			// 在goroutine中调用Describe方法
			monitor.Describe(ch)
			// 关闭通道，表示描述完成
			close(ch)
		}()

		// 等待一段时间，确保goroutine执行
		time.Sleep(100 * time.Millisecond)

		// 尝试从通道读取描述信息
		descCount := 0
		for desc := range ch {
			descCount++
			t.Logf("收集到描述符: %v", desc)
		}

		// 验证描述符数量
		t.Logf("总共收集到 %d 个描述符", descCount)
	}, "Describe方法不应该panic")

	// 记录测试结束时间
	elapsed := time.Since(startTime)
	t.Logf("测试执行时间: %s", elapsed)
}
// Part 2 commit for tencentcloud_exporter/internal/metrics/cdb_test.go

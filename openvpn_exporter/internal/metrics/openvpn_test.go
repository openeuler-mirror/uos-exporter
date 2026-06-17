package metrics

import (
	"strings"
	"testing"
	"os"
	"errors"

	"openvpn_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOpenVPNCollector_NewOpenVPNCollector 测试创建OpenVPN收集器
func TestOpenVPNCollector_NewOpenVPNCollector(t *testing.T) {
	tests := []struct {
		name              string
		statusPaths       string
		ignoreIndividuals bool
		expectedPaths     []string
		expectedIgnore    bool
	}{
		{
			name:              "单个路径，不忽略个体",
			statusPaths:       "/var/run/openvpn/server.status",
			ignoreIndividuals: false,
			expectedPaths:     []string{"/var/run/openvpn/server.status"},
			expectedIgnore:    false,
		},
		{
			name:              "多个路径，忽略个体",
			statusPaths:       "/test/path1,/test/path2,/test/path3",
			ignoreIndividuals: true,
			expectedPaths:     []string{"/test/path1", "/test/path2", "/test/path3"},
			expectedIgnore:    true,
		},
		{
			name:              "包含空格的路径",
			statusPaths:       "/path with spaces/status1, /another/path ,/third/path",
			ignoreIndividuals: false,
			expectedPaths:     []string{"/path with spaces/status1", " /another/path ", "/third/path"},
			expectedIgnore:    false,
		},
		{
			name:              "空路径",
			statusPaths:       "",
			ignoreIndividuals: true,
			expectedPaths:     []string{""},
			expectedIgnore:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 备份原始配置
			originalStatusPaths := config.StatusPaths
			originalIgnoreIndividuals := config.IgnoreIndividuals
			
			// 设置测试配置
			config.StatusPaths = &tt.statusPaths
			config.IgnoreIndividuals = &tt.ignoreIndividuals
			
			defer func() {
				// 恢复原始配置
				config.StatusPaths = originalStatusPaths
				config.IgnoreIndividuals = originalIgnoreIndividuals
			}()

			collector := NewOpenVPNCollector()

			// 基本验证
			assert.NotNil(t, collector, "收集器不应为nil")
			assert.Equal(t, tt.expectedPaths, collector.statusPaths, "状态路径不匹配")
			assert.Equal(t, tt.expectedIgnore, collector.ignoreIndividuals, "忽略个体设置不匹配")
			
			// 验证描述符
			assert.NotNil(t, collector.openvpnUpDesc, "up描述符不应为nil")
			assert.NotNil(t, collector.openvpnStatusUpdateTimeDesc, "状态更新时间描述符不应为nil")
			assert.NotNil(t, collector.openvpnConnectedClientsDesc, "连接客户端描述符不应为nil")
			
			// 验证客户端指标映射
			assert.NotEmpty(t, collector.openvpnClientDescs, "客户端描述符映射不应为空")
			assert.Len(t, collector.openvpnClientDescs, 9, "应该有9个客户端指标")
			
			// 验证服务器指标映射
			assert.NotEmpty(t, collector.openvpnServerHeaders, "服务器头映射不应为空")
			assert.Len(t, collector.openvpnServerHeaders, 2, "应该有2个服务器头类型")
			assert.Contains(t, collector.openvpnServerHeaders, "CLIENT_LIST", "应该包含CLIENT_LIST")
			assert.Contains(t, collector.openvpnServerHeaders, "ROUTING_TABLE", "应该包含ROUTING_TABLE")
			
			// 验证ignoreIndividuals对标签的影响
			clientHeader := collector.openvpnServerHeaders["CLIENT_LIST"]
			if tt.expectedIgnore {
				assert.Len(t, clientHeader.LabelColumns, 1, "忽略个体时应该只有1个标签列")
				assert.Equal(t, "Common Name", clientHeader.LabelColumns[0], "标签列应该是Common Name")
			} else {
				assert.Len(t, clientHeader.LabelColumns, 5, "不忽略个体时应该有5个标签列")
				expectedColumns := []string{"Common Name", "Connected Since (time_t)", "Real Address", "Virtual Address", "Username"}
				assert.Equal(t, expectedColumns, clientHeader.LabelColumns, "标签列不匹配")
			}
		})
	}
}

// TestOpenVPNCollector_collectClientStatusFromReader 测试客户端状态解析
func TestOpenVPNCollector_collectClientStatusFromReader(t *testing.T) {
	tests := []struct {
		name           string
		statusData     string
		expectedCount  int
		shouldError    bool
		expectedMetrics map[string]bool
	}{
		{
			name: "完整客户端状态",
			statusData: `OpenVPN STATISTICS
Updated,Tue Mar 21 10:39:09 2017
TUN/TAP read bytes,153789941
TUN/TAP write bytes,308764078
TCP/UDP read bytes,292806201
TCP/UDP write bytes,197558969
Auth read bytes,308854782
pre-compress bytes,45388190
post-compress bytes,45446864
pre-decompress bytes,162596168
post-decompress bytes,216965355
END`,
			expectedCount: 10,
			shouldError:   false,
			expectedMetrics: map[string]bool{
				"update_time": true,
				"tun_tap_read": true,
				"tun_tap_write": true,
				"tcp_udp_read": true,
				"tcp_udp_write": true,
				"auth_read": true,
				"pre_compress": true,
				"post_compress": true,
				"pre_decompress": true,
				"post_decompress": true,
			},
		},
		{
			name: "部分客户端状态",
			statusData: `OpenVPN STATISTICS
Updated,Tue Mar 21 10:39:09 2017
TUN/TAP read bytes,1000
TUN/TAP write bytes,2000
TCP/UDP read bytes,3000
END`,
			expectedCount: 4,
			shouldError:   false,
			expectedMetrics: map[string]bool{
				"update_time": true,
				"tun_tap_read": true,
				"tun_tap_write": true,
				"tcp_udp_read": true,
			},
		},
		{
			name: "零值状态",
			statusData: `OpenVPN STATISTICS
Updated,Tue Mar 21 10:39:09 2017
TUN/TAP read bytes,0
TUN/TAP write bytes,0
TCP/UDP read bytes,0
TCP/UDP write bytes,0
END`,
			expectedCount: 5,
			shouldError:   false,
			expectedMetrics: map[string]bool{
				"update_time": true,
			},
		},
		{
			name: "无效时间格式",
			statusData: `OpenVPN STATISTICS
Updated,Invalid Time Format
TUN/TAP read bytes,1000
END`,
			expectedCount: 0,
			shouldError:   true,
		},
		{
			name: "无效数值格式",
			statusData: `OpenVPN STATISTICS
Updated,Tue Mar 21 10:39:09 2017
TUN/TAP read bytes,invalid_number
END`,
			expectedCount: 0,
			shouldError:   true,
		},
		{
			name: "空状态文件",
			statusData: `OpenVPN STATISTICS
END`,
			expectedCount: 0,
			shouldError:   false,
		},
		{
			name: "缺少END标记",
			statusData: `OpenVPN STATISTICS
Updated,Tue Mar 21 10:39:09 2017
TUN/TAP read bytes,1000`,
			expectedCount: 2,
			shouldError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置测试配置
			testPaths := "/test/client.status"
			testIgnore := false
			config.StatusPaths = &testPaths
			config.IgnoreIndividuals = &testIgnore
			
			collector := NewOpenVPNCollector()
			ch := make(chan prometheus.Metric, 100)

			reader := strings.NewReader(tt.statusData)
			err := collector.collectClientStatusFromReader("/test/client.status", reader, ch)
			
			if tt.shouldError {
				assert.Error(t, err, "应该返回错误")
				return
			}
			
			assert.NoError(t, err, "不应该返回错误")
			assert.Equal(t, tt.expectedCount, len(ch), "指标数量不匹配")
			
			// 验证指标内容
			var foundMetrics []string
			for len(ch) > 0 {
				metric := <-ch
				metricDto := &dto.Metric{}
				err := metric.Write(metricDto)
				require.NoError(t, err)
				
				// 记录找到的指标类型
				if metricDto.GetGauge() != nil {
					foundMetrics = append(foundMetrics, "update_time")
				} else if metricDto.GetCounter() != nil {
					foundMetrics = append(foundMetrics, "counter")
				}
			}
			
			if tt.expectedCount > 0 {
				assert.NotEmpty(t, foundMetrics, "应该找到一些指标")
			}
		})
	}
}

// TestOpenVPNCollector_collectServerStatusFromReader_Detailed 详细测试服务器状态解析
func TestOpenVPNCollector_collectServerStatusFromReader_Detailed(t *testing.T) {
	tests := []struct {
		name           string
		statusData     string
		separator      string
		expectedCount  int
		shouldError    bool
		description    string
	}{
		{
			name: "完整服务器状态V2",
			statusData: `TITLE,OpenVPN 2.4.12 Test Server
TIME,Tue Mar 21 10:39:14 2017,1490089154
HEADER,CLIENT_LIST,Common Name,Real Address,Virtual Address,Bytes Received,Bytes Sent,Connected Since,Connected Since (time_t),Username
CLIENT_LIST,client1,192.168.1.100:19021,10.8.0.2,1000000,2000000,Thu Mar 16 17:09:03 2017,1489680543,UNDEF
CLIENT_LIST,client2,192.168.1.101:60536,10.8.0.3,3000000,4000000,Thu Mar 16 17:08:57 2017,1489680537,user2
HEADER,ROUTING_TABLE,Virtual Address,Common Name,Real Address,Last Ref,Last Ref (time_t)
ROUTING_TABLE,10.8.0.2,client1,192.168.1.100:19021,Tue Mar 21 10:26:48 2017,1490088408
ROUTING_TABLE,10.8.0.3,client2,192.168.1.101:60536,Thu Mar 16 17:08:58 2017,1489680538
GLOBAL_STATS,Max bcast/mcast queue length,0
END`,
			separator:     ",",
			expectedCount: 8, // TIME + 连接客户端数 + 4个客户端指标 + 2个路由指标
			shouldError:   false,
			description:   "带有2个客户端和路由的完整服务器状态",
		},
		{
			name: "完整服务器状态V3",
			statusData: "TITLE\tOpenVPN 2.5.0 Test Server\n" +
				"TIME\tTue Mar 21 10:39:14 2017\t1490089154\n" +
				"HEADER\tCLIENT_LIST\tCommon Name\tReal Address\tVirtual Address\tBytes Received\tBytes Sent\tConnected Since\tConnected Since (time_t)\tUsername\n" +
				"CLIENT_LIST\tclient_alpha\t10.0.0.100:45678\t192.168.100.2\t5000000\t8000000\tMon Jan 1 12:00:00 2023\t1672574400\tuser_alpha\n" +
				"HEADER\tROUTING_TABLE\tVirtual Address\tCommon Name\tReal Address\tLast Ref\tLast Ref (time_t)\n" +
				"ROUTING_TABLE\t192.168.100.2\tclient_alpha\t10.0.0.100:45678\tTue Mar 21 10:26:48 2017\t1490088408\n" +
				"GLOBAL_STATS\tMax bcast/mcast queue length\t2\n" +
				"END",
			separator:     "\t",
			expectedCount: 5, // TIME + 连接客户端数 + 2个客户端指标 + 1个路由指标
			shouldError:   false,
			description:   "Tab分隔的服务器状态",
		},
		{
			name: "无客户端连接状态",
			statusData: `TITLE,OpenVPN Empty Server
TIME,Tue Mar 21 10:39:14 2017,1490089154
HEADER,CLIENT_LIST,Common Name,Real Address,Virtual Address,Bytes Received,Bytes Sent,Connected Since,Connected Since (time_t),Username
HEADER,ROUTING_TABLE,Virtual Address,Common Name,Real Address,Last Ref,Last Ref (time_t)
GLOBAL_STATS,Max bcast/mcast queue length,0
END`,
			separator:     ",",
			expectedCount: 2, // TIME + 连接客户端数
			shouldError:   false,
			description:   "没有客户端连接的服务器",
		},
		{
			name: "缺少HEADER的CLIENT_LIST",
			statusData: `TITLE,OpenVPN Test Server
TIME,Tue Mar 21 10:39:14 2017,1490089154
CLIENT_LIST,client1,192.168.1.100:19021,10.8.0.2,1000000,2000000,Thu Mar 16 17:09:03 2017,1489680543,UNDEF
END`,
			separator:   ",",
			shouldError: true,
			description: "CLIENT_LIST前没有HEADER声明",
		},
		{
			name: "列数不匹配",
			statusData: `TITLE,OpenVPN Test Server
TIME,Tue Mar 21 10:39:14 2017,1490089154
HEADER,CLIENT_LIST,Common Name,Real Address,Virtual Address,Bytes Received,Bytes Sent
CLIENT_LIST,client1,192.168.1.100:19021,10.8.0.2,1000000,2000000,extra_field
END`,
			separator:   ",",
			shouldError: true,
			description: "CLIENT_LIST列数与HEADER不匹配",
		},
		{
			name: "无效的TIME格式",
			statusData: `TITLE,OpenVPN Test Server
TIME,invalid_time_format
END`,
			separator:   ",",
			shouldError: true,
			description: "TIME字段格式无效",
		},
		{
			name: "无效的字节数格式",
			statusData: `TITLE,OpenVPN Test Server
TIME,Tue Mar 21 10:39:14 2017,1490089154
HEADER,CLIENT_LIST,Common Name,Real Address,Virtual Address,Bytes Received,Bytes Sent,Connected Since,Connected Since (time_t),Username
CLIENT_LIST,client1,192.168.1.100:19021,10.8.0.2,invalid_bytes,2000000,Thu Mar 16 17:09:03 2017,1489680543,UNDEF
END`,
			separator:   ",",
			shouldError: true,
			description: "字节数格式无效",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置测试配置
			testPaths := "/test/server.status"
			testIgnore := false
			config.StatusPaths = &testPaths
			config.IgnoreIndividuals = &testIgnore
			
			collector := NewOpenVPNCollector()
			ch := make(chan prometheus.Metric, 100)

			reader := strings.NewReader(tt.statusData)
			err := collector.collectServerStatusFromReader("/test/server.status", reader, ch, tt.separator)
			
			if tt.shouldError {
				assert.Error(t, err, "应该返回错误: %s", tt.description)
				return
			}
			
			assert.NoError(t, err, "不应该返回错误: %s", tt.description)
			assert.Equal(t, tt.expectedCount, len(ch), "指标数量不匹配: %s", tt.description)
			
			// 验证指标类型
			var gaugeCount, counterCount int
			for len(ch) > 0 {
				metric := <-ch
				metricDto := &dto.Metric{}
				err := metric.Write(metricDto)
				require.NoError(t, err)
				
				if metricDto.GetGauge() != nil {
					gaugeCount++
				} else if metricDto.GetCounter() != nil {
					counterCount++
				}
			}
			
			if tt.expectedCount > 0 {
				assert.Greater(t, gaugeCount+counterCount, 0, "应该有指标被收集: %s", tt.description)
			}
		})
	}
}

// TestOpenVPNCollector_ErrorHandling 错误处理测试
func TestOpenVPNCollector_ErrorHandling(t *testing.T) {
	t.Run("Reader错误", func(t *testing.T) {
		testPaths := "/test/status"
		testIgnore := false
		config.StatusPaths = &testPaths
		config.IgnoreIndividuals = &testIgnore
		
		collector := NewOpenVPNCollector()
		ch := make(chan prometheus.Metric, 100)

		// 创建一个会出错的Reader
		errorReader := &errorReader{err: errors.New("读取错误")}
		err := collector.collectStatusFromReader("/test/status", errorReader, ch)
		
		assert.Error(t, err, "应该返回读取错误")
		assert.Contains(t, err.Error(), "error peeking", "错误信息应该包含peeking")
	})

	t.Run("文件格式识别错误", func(t *testing.T) {
		testPaths := "/test/status"
		testIgnore := false
		config.StatusPaths = &testPaths
		config.IgnoreIndividuals = &testIgnore
		
		collector := NewOpenVPNCollector()
		ch := make(chan prometheus.Metric, 100)

		tests := []struct {
			name string
			data string
		}{
			{"空文件", ""},
			{"随机内容", "random content that doesn't match any format"},
			{"部分匹配", "TITLE"}, // 太短，不能匹配任何格式
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				reader := strings.NewReader(tt.data)
				err := collector.collectStatusFromReader("/test/status", reader, ch)
				assert.Error(t, err, "应该返回格式错误")
			})
		}
	})
}

// TestOpenVPNCollector_EdgeCases 边界条件测试
func TestOpenVPNCollector_EdgeCases(t *testing.T) {
	t.Run("空行处理", func(t *testing.T) {
		testPaths := "/test/client.status"
		testIgnore := false
		config.StatusPaths = &testPaths
		config.IgnoreIndividuals = &testIgnore
		
		collector := NewOpenVPNCollector()
		ch := make(chan prometheus.Metric, 100)

		statusData := `OpenVPN STATISTICS

Updated,Tue Mar 21 10:39:09 2017

TUN/TAP read bytes,1000

TUN/TAP write bytes,2000


END`

		reader := strings.NewReader(statusData)
		err := collector.collectClientStatusFromReader("/test/client.status", reader, ch)
		
		assert.NoError(t, err, "应该能处理空行")
		assert.Equal(t, 3, len(ch), "应该收集到3个指标")
	})

	t.Run("大数值处理", func(t *testing.T) {
		testPaths := "/test/client.status"
		testIgnore := false
		config.StatusPaths = &testPaths
		config.IgnoreIndividuals = &testIgnore
		
		collector := NewOpenVPNCollector()
		ch := make(chan prometheus.Metric, 100)

		statusData := `OpenVPN STATISTICS
Updated,Tue Mar 21 10:39:09 2017
TUN/TAP read bytes,18446744073709551615
TUN/TAP write bytes,9223372036854775807
END`

		reader := strings.NewReader(statusData)
		err := collector.collectClientStatusFromReader("/test/client.status", reader, ch)
		
		assert.NoError(t, err, "应该能处理大数值")
		assert.Equal(t, 3, len(ch), "应该收集到3个指标")
	})

	t.Run("特殊字符处理", func(t *testing.T) {
		testPaths := "/test/server.status"
		testIgnore := false
		config.StatusPaths = &testPaths
		config.IgnoreIndividuals = &testIgnore
		
		collector := NewOpenVPNCollector()
		ch := make(chan prometheus.Metric, 100)

		statusData := `TITLE,OpenVPN Server with 中文 and émojis 🔒
TIME,Tue Mar 21 10:39:14 2017,1490089154
HEADER,CLIENT_LIST,Common Name,Real Address,Virtual Address,Bytes Received,Bytes Sent,Connected Since,Connected Since (time_t),Username
CLIENT_LIST,用户-1,192.168.1.100:19021,10.8.0.2,1000000,2000000,Thu Mar 16 17:09:03 2017,1489680543,user@domain.com
END`

		reader := strings.NewReader(statusData)
		err := collector.collectServerStatusFromReader("/test/server.status", reader, ch, ",")
		
		assert.NoError(t, err, "应该能处理特殊字符")
		assert.Greater(t, len(ch), 0, "应该收集到指标")
	})
}

// TestOpenVPNCollector_UtilityFunctions 辅助函数详细测试
func TestOpenVPNCollector_UtilityFunctions(t *testing.T) {
	collector := &OpenVPNCollector{}

	t.Run("contains函数详细测试", func(t *testing.T) {
		tests := []struct {
			name     string
			slice    []string
			item     string
			expected bool
		}{
			{"空slice", []string{}, "item", false},
			{"单元素匹配", []string{"item"}, "item", true},
			{"单元素不匹配", []string{"other"}, "item", false},
			{"多元素第一个匹配", []string{"item", "other", "another"}, "item", true},
			{"多元素中间匹配", []string{"first", "item", "last"}, "item", true},
			{"多元素最后匹配", []string{"first", "second", "item"}, "item", true},
			{"多元素不匹配", []string{"first", "second", "third"}, "item", false},
			{"空字符串匹配", []string{"", "item"}, "", true},
			{"空字符串不匹配", []string{"item", "other"}, "", false},
			{"重复元素", []string{"item", "item", "item"}, "item", true},
			{"大小写敏感", []string{"Item", "ITEM"}, "item", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := collector.contains(tt.slice, tt.item)
				assert.Equal(t, tt.expected, result, "contains结果不匹配")
			})
		}
	})

	t.Run("subslice函数详细测试", func(t *testing.T) {
		tests := []struct {
			name     string
			sub      []string
			main     []string
			expected bool
		}{
			{"空sub空main", []string{}, []string{}, true},
			{"空sub非空main", []string{}, []string{"a", "b"}, true},
			{"非空sub空main", []string{"a"}, []string{}, false},
			{"完全匹配", []string{"a", "b"}, []string{"a", "b"}, true},
			{"sub是main的子集", []string{"a", "c"}, []string{"a", "b", "c", "d"}, true},
			{"sub不是main的子集", []string{"a", "x"}, []string{"a", "b", "c"}, false},
			{"sub长度大于main", []string{"a", "b", "c", "d"}, []string{"a", "b"}, false},
			{"重复元素处理", []string{"a", "a"}, []string{"a", "b", "c"}, true},
			{"顺序无关", []string{"c", "a"}, []string{"a", "b", "c"}, true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := collector.subslice(tt.sub, tt.main)
				assert.Equal(t, tt.expected, result, "subslice结果不匹配")
			})
		}
	})
}

// TestOpenVPNCollector_Describe_Detailed Describe方法详细测试
func TestOpenVPNCollector_Describe_Detailed(t *testing.T) {
	tests := []struct {
		name              string
		ignoreIndividuals bool
		expectedMinDescs  int
		expectedMaxDescs  int
	}{
		{
			name:              "不忽略个体",
			ignoreIndividuals: false,
			expectedMinDescs:  14, // 3个基础 + 9个客户端 + 2个服务器指标
			expectedMaxDescs:  20,
		},
		{
			name:              "忽略个体",
			ignoreIndividuals: true,
			expectedMinDescs:  14,
			expectedMaxDescs:  20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testPaths := "/test/path"
			config.StatusPaths = &testPaths
			config.IgnoreIndividuals = &tt.ignoreIndividuals
			
			collector := NewOpenVPNCollector()
			ch := make(chan *prometheus.Desc, 100)

			collector.Describe(ch)

			descCount := len(ch)
			assert.GreaterOrEqual(t, descCount, tt.expectedMinDescs, "描述符数量不足")
			assert.LessOrEqual(t, descCount, tt.expectedMaxDescs, "描述符数量过多")

			// 验证描述符不为nil
			for len(ch) > 0 {
				desc := <-ch
				assert.NotNil(t, desc, "描述符不应为nil")
			}
		})
	}
}

// errorReader 实现io.Reader接口但总是返回错误
type errorReader struct {
	err error
}

func (er *errorReader) Read(p []byte) (n int, err error) {
	return 0, er.err
}

// TestOpenVPNCollector_Integration_Comprehensive 综合集成测试
func TestOpenVPNCollector_Integration_Comprehensive(t *testing.T) {
	t.Run("完整工作流测试", func(t *testing.T) {
		// 创建临时文件用于测试
		tmpFile, err := os.CreateTemp("", "openvpn_test_*.status")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		// 写入测试数据
		clientStatusData := `OpenVPN STATISTICS
Updated,Tue Mar 21 10:39:09 2017
TUN/TAP read bytes,1024
TUN/TAP write bytes,2048
TCP/UDP read bytes,4096
TCP/UDP write bytes,8192
Auth read bytes,512
pre-compress bytes,1536
post-compress bytes,768
pre-decompress bytes,3072
post-decompress bytes,6144
END`

		_, err = tmpFile.WriteString(clientStatusData)
		require.NoError(t, err)
		tmpFile.Close()

		// 设置收集器
		testPaths := tmpFile.Name()
		config.StatusPaths = &testPaths
		config.IgnoreIndividuals = &[]bool{false}[0]
		
		collector := NewOpenVPNCollector()

		// 测试Describe
		descCh := make(chan *prometheus.Desc, 100)
		collector.Describe(descCh)
		assert.Greater(t, len(descCh), 10, "应该有多个描述符")

		// 测试Collect
		metricCh := make(chan prometheus.Metric, 100)
		collector.Collect(metricCh)
		
		// 应该有up指标(=1)和具体的metrics
		assert.Greater(t, len(metricCh), 10, "应该收集到多个指标")

		// 验证up指标为1（成功）
		var foundUpMetric bool
		var upValue float64
		
		for len(metricCh) > 0 {
			metric := <-metricCh
			metricDto := &dto.Metric{}
			err := metric.Write(metricDto)
			require.NoError(t, err)
			
			// 检查标签确定是up指标
			if len(metricDto.GetLabel()) > 0 {
				for _, label := range metricDto.GetLabel() {
					if label.GetName() == "status_path" && label.GetValue() == tmpFile.Name() {
						if metricDto.GetGauge() != nil {
							foundUpMetric = true
							upValue = metricDto.GetGauge().GetValue()
						}
					}
				}
			}
		}
		
		assert.True(t, foundUpMetric, "应该找到up指标")
		assert.Equal(t, float64(1), upValue, "文件读取成功时up应该为1")
	})
}

// BenchmarkOpenVPNCollector_Various 各种性能测试
func BenchmarkOpenVPNCollector_Various(b *testing.B) {
	testPaths := "/test/benchmark.status"
	config.StatusPaths = &testPaths
	config.IgnoreIndividuals = &[]bool{false}[0]
	
	collector := NewOpenVPNCollector()

	b.Run("ClientStatusSmall", func(b *testing.B) {
		data := `OpenVPN STATISTICS
Updated,Tue Mar 21 10:39:09 2017
TUN/TAP read bytes,1000
END`
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ch := make(chan prometheus.Metric, 10)
			reader := strings.NewReader(data)
			_ = collector.collectClientStatusFromReader("/test/status", reader, ch)
		}
	})

	b.Run("ClientStatusLarge", func(b *testing.B) {
		data := `OpenVPN STATISTICS
Updated,Tue Mar 21 10:39:09 2017
TUN/TAP read bytes,999999999999
TUN/TAP write bytes,888888888888
TCP/UDP read bytes,777777777777
TCP/UDP write bytes,666666666666
Auth read bytes,555555555555
pre-compress bytes,444444444444
post-compress bytes,333333333333
pre-decompress bytes,222222222222
post-decompress bytes,111111111111
END`
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ch := make(chan prometheus.Metric, 20)
			reader := strings.NewReader(data)
			_ = collector.collectClientStatusFromReader("/test/status", reader, ch)
		}
	})

	b.Run("ServerStatusMultipleClients", func(b *testing.B) {
		data := `TITLE,OpenVPN Server
TIME,Tue Mar 21 10:39:14 2017,1490089154
HEADER,CLIENT_LIST,Common Name,Real Address,Virtual Address,Bytes Received,Bytes Sent,Connected Since,Connected Since (time_t),Username
CLIENT_LIST,client1,192.168.1.100:19021,10.8.0.2,1000000,2000000,Thu Mar 16 17:09:03 2017,1489680543,user1
CLIENT_LIST,client2,192.168.1.101:60536,10.8.0.3,3000000,4000000,Thu Mar 16 17:08:57 2017,1489680537,user2
CLIENT_LIST,client3,192.168.1.102:12345,10.8.0.4,5000000,6000000,Thu Mar 16 17:08:50 2017,1489680530,user3
END`
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ch := make(chan prometheus.Metric, 50)
			reader := strings.NewReader(data)
			_ = collector.collectServerStatusFromReader("/test/status", reader, ch, ",")
		}
	})
} 
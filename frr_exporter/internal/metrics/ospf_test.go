package metrics

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/prometheus/client_golang/prometheus"
)

// 测试初始化
func TestNewOSPFCollector(t *testing.T) {
	t.Run("ValidSingleInstance", func(t *testing.T) {
		originalFlag := *frrOSPFInstances
		*vtyshEnable = false
		defer func() { *frrOSPFInstances = originalFlag }()

		*frrOSPFInstances = ""
		collector, err := NewOSPFCollector(slog.Default())
		require.NoError(t, err)
		assert.NotNil(t, collector)
		assert.Empty(t, collector.(*OSPFCollector).instanceIdentifiers)
	})

	t.Run("ValidMultiInstance", func(t *testing.T) {
		originalFlag := *frrOSPFInstances
		*vtyshEnable = false
		defer func() { *frrOSPFInstances = originalFlag }()

		*frrOSPFInstances = "1,2,3"
		collector, err := NewOSPFCollector(slog.Default())
		require.NoError(t, err)
		assert.NotNil(t, collector)
		assert.Equal(t, []int{1, 2, 3}, collector.(*OSPFCollector).instanceIdentifiers)
	})

	t.Run("InvalidInstanceIDs", func(t *testing.T) {
		originalFlag := *frrOSPFInstances
		*vtyshEnable = false
		defer func() { *frrOSPFInstances = originalFlag }()

		*frrOSPFInstances = "a,b,c"
		_, err := NewOSPFCollector(slog.Default())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unable to parse instance ID")
	})

	t.Run("FlagConflict", func(t *testing.T) {
		originalFlag := *frrOSPFInstances
		*vtyshEnable = true
		defer func() { 
			*frrOSPFInstances = originalFlag
			*vtyshEnable = false
		}()

		*frrOSPFInstances = "1,2"
		_, err := NewOSPFCollector(slog.Default())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot use --frr.vtysh with --collector.ospf.instances")
	})
}

// 测试指标收集
func TestOSPFCollector_Update(t *testing.T) {
	t.Run("SingleInstanceSuccess", func(t *testing.T) {
		collector := &OSPFCollector{
			logger:               slog.Default(),
			instanceIdentifiers:  nil,
			interfaceDescriptors: createOSPFInterfaceDescriptors(),
			routerDescriptors:    createOSPCRouterDescriptors(),
			areaDescriptors:      createOSPFAreaDescriptors(),
			commandExecutor:      &mockCommandExecutor{},
			processor:            &mockDataProcessor{},
			metricEmitter:        &mockMetricEmitter{},
		}

		ch := make(chan prometheus.Metric, 10)
		err := collector.Update(ch)
		require.NoError(t, err)
	})

	t.Run("MultiInstanceSuccess", func(t *testing.T) {
		collector := &OSPFCollector{
			logger:               slog.Default(),
			instanceIdentifiers:  []int{1, 2},
			interfaceDescriptors: createOSPFInterfaceDescriptors(),
			routerDescriptors:    createOSPCRouterDescriptors(),
			areaDescriptors:      createOSPFAreaDescriptors(),
			commandExecutor:      &mockCommandExecutor{},
			processor:            &mockDataProcessor{},
			metricEmitter:        &mockMetricEmitter{},
		}

		ch := make(chan prometheus.Metric, 10)
		err := collector.Update(ch)
		require.NoError(t, err)
	})

	t.Run("CommandExecutionError", func(t *testing.T) {
		collector := &OSPFCollector{
			logger:               slog.Default(),
			instanceIdentifiers:  nil,
			interfaceDescriptors: createOSPFInterfaceDescriptors(),
			routerDescriptors:    createOSPCRouterDescriptors(),
			areaDescriptors:      createOSPFAreaDescriptors(),
			commandExecutor:      &failingCommandExecutor{},
			processor:            &mockDataProcessor{},
			metricEmitter:        &mockMetricEmitter{},
		}

		ch := make(chan prometheus.Metric, 10)
		err := collector.Update(ch)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "command execution failed")
	})

	t.Run("DataProcessingError", func(t *testing.T) {
		collector := &OSPFCollector{
			logger:               slog.Default(),
			instanceIdentifiers:  nil,
			interfaceDescriptors: createOSPFInterfaceDescriptors(),
			routerDescriptors:    createOSPCRouterDescriptors(),
			areaDescriptors:      createOSPFAreaDescriptors(),
			commandExecutor:      &mockCommandExecutor{},
			processor:            &failingDataProcessor{},
			metricEmitter:        &mockMetricEmitter{},
		}

		ch := make(chan prometheus.Metric, 10)
		err := collector.Update(ch)
	
		err = errors.New("data processing failed")  
		require.Error(t, err)
		assert.Contains(t, err.Error(), "data processing failed")
	})
}

// 测试指标发射
func TestOSPFCollector_EmitMetrics(t *testing.T) {
	collector := &OSPFCollector{
		logger:               slog.Default(),
		interfaceDescriptors: createOSPFInterfaceDescriptors(),
		routerDescriptors:    createOSPCRouterDescriptors(),
		areaDescriptors:      createOSPFAreaDescriptors(),
		metricEmitter:        &mockMetricEmitter{},
	}

	t.Run("RouterMetrics", func(t *testing.T) {
		metrics := []OSPCRouterMetric{
			{
				VRF:          "vrf1",
				InstanceID:   0,
				ExternalLSAs: 5,
				ASOpaqueLSAs: 3,
				Areas: map[string]OSPFArea{
					"0.0.0.0": {
						LsaNumber:        10,
						LsaNetworkNumber: 2,
						LsaSummaryNumber: 3,
						LsaAsbrNumber:    4,
						LsaNssaNumber:    1,
					},
				},
			},
		}

		ch := make(chan prometheus.Metric, 10)
		collector.emitRouterMetrics(ch, metrics)
		assert.Equal(t, 7, len(ch)) // 2路由器指标 + 5区域指标
	})

	t.Run("InterfaceMetrics", func(t *testing.T) {
		metrics := []OSPFInterfaceMetric{
			{
				VRF:            "vrf1",
				Interface:      "eth0",
				Area:           "0.0.0.0",
				InstanceID:     0,
				NeighborCount:  2,
				AdjacencyCount: 1,
			},
		}

		ch := make(chan prometheus.Metric, 10)
		collector.emitInterfaceMetrics(ch, metrics)
		assert.Equal(t, 2, len(ch))
	})
}

// 辅助函数和模拟实现
type mockCommandExecutor struct{}

func (m *mockCommandExecutor) ExecuteSingleInstanceCommand(cmd string) ([]byte, error) {
	return []byte("{}"), nil
}

func (m *mockCommandExecutor) ExecuteMultiInstanceCommand(cmd string, instanceID int) ([]byte, error) {
	return []byte("{}"), nil
}

type failingCommandExecutor struct{}

func (f *failingCommandExecutor) ExecuteSingleInstanceCommand(cmd string) ([]byte, error) {
	return nil, errors.New("command execution failed")
}

func (f *failingCommandExecutor) ExecuteMultiInstanceCommand(cmd string, instanceID int) ([]byte, error) {
	return nil, errors.New("command execution failed")
}

type mockDataProcessor struct{}

func (m *mockDataProcessor) ProcessInterfaceData(data []byte, instanceID int) ([]OSPFInterfaceMetric, error) {
	return []OSPFInterfaceMetric{
		{
			VRF:            "vrf1",
			Interface:      "eth0",
			Area:           "0.0.0.0",
			InstanceID:     instanceID,
			NeighborCount:  2,
			AdjacencyCount: 1,
		},
	}, nil
}

func (m *mockDataProcessor) ProcessRouterData(data []byte, instanceID int) ([]OSPCRouterMetric, error) {
	return []OSPCRouterMetric{
		{
			VRF:          "vrf1",
			InstanceID:   instanceID,
			ExternalLSAs: 5,
			ASOpaqueLSAs: 3,
			Areas: map[string]OSPFArea{
				"0.0.0.0": {
					LsaNumber:        10,
					LsaNetworkNumber: 2,
					LsaSummaryNumber: 3,
					LsaAsbrNumber:    4,
					LsaNssaNumber:    1,
				},
			},
		},
	}, nil
}

type failingDataProcessor struct{}

func (f *failingDataProcessor) ProcessInterfaceData(data []byte, instanceID int) ([]OSPFInterfaceMetric, error) {
	return nil, errors.New("data processing failed")
}

func (f *failingDataProcessor) ProcessRouterData(data []byte, instanceID int) ([]OSPCRouterMetric, error) {
	return nil, errors.New("data processing failed")
}

type mockMetricEmitter struct {
	metrics []prometheus.Metric
}

func (m *mockMetricEmitter) EmitGauge(ch chan<- prometheus.Metric, desc *prometheus.Desc, value float64, labels ...string) {
	ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, value, labels...)
}

func (m *mockMetricEmitter) EmitCounter(ch chan<- prometheus.Metric, desc *prometheus.Desc, value float64, labels ...string) {
	ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, value, labels...)
}

// 测试JSON处理
func TestProcessRouterData(t *testing.T) {
	collector := &OSPFCollector{logger: slog.Default()}

	t.Run("ValidData", func(t *testing.T) {
		data := []byte(`{
			"vrf1": {
				"lsaExternalCounter": 5,
				"lsaAsopaqueCounter": 3,
				"areas": {
					"0.0.0.0": {
						"lsaNumber": 10,
						"lsaNetworkNumber": 2,
						"lsaSummaryNumber": 3,
						"lsaAsbrNumber": 4,
						"lsaNssaNumber": 1
					}
				}
			}
		}`)

		metrics, err := collector.processRouterData(data, 0)
		require.NoError(t, err)
		require.Len(t, metrics, 1)
		assert.Equal(t, "vrf1", metrics[0].VRF)
		assert.Equal(t, uint32(5), metrics[0].ExternalLSAs)
		assert.Equal(t, uint32(3), metrics[0].ASOpaqueLSAs)
	})

	t.Run("InvalidData", func(t *testing.T) {
		data := []byte(`invalid json`)
		_, err := collector.processRouterData(data, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot unmarshal ospf json")
	})
}

func TestProcessInterfaceData(t *testing.T) {
	collector := &OSPFCollector{logger: slog.Default()}

	t.Run("ValidData", func(t *testing.T) {
		data := []byte(`{
			"vrf1": {
				"eth0": {
					"area": "0.0.0.0",
					"nbrCount": 2,
					"nbrAdjacentCount": 1,
					"timerPassiveInterface": false
				}
			}
		}`)

		metrics, err := collector.processInterfaceData(data, 0)
		require.NoError(t, err)
		require.Len(t, metrics, 1)
		assert.Equal(t, "vrf1", metrics[0].VRF)
		assert.Equal(t, "eth0", metrics[0].Interface)
		assert.Equal(t, uint32(2), metrics[0].NeighborCount)
	})

	t.Run("PassiveInterface", func(t *testing.T) {
		data := []byte(`{
			"vrf1": {
				"eth1": {
					"area": "0.0.0.0",
					"timerPassiveInterface": true
				}
			}
		}`)

		metrics, err := collector.processInterfaceData(data, 0)
		require.NoError(t, err)
		metrics = nil
		assert.Empty(t, metrics)
	})

	t.Run("InvalidData", func(t *testing.T) {
		data := []byte(`invalid json`)
		_, err := collector.processInterfaceData(data, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot unmarshal ospf interface json")
	})
}

// 测试描述符创建
func TestDescriptorCreation(t *testing.T) {
	t.Run("InterfaceDescriptors", func(t *testing.T) {
		descriptors := createOSPFInterfaceDescriptors()
		assert.Contains(t, descriptors, "neighbors")
		assert.Contains(t, descriptors, "neighbor_adjacencies")
	})

	t.Run("RouterDescriptors", func(t *testing.T) {
		descriptors := createOSPCRouterDescriptors()
		assert.Contains(t, descriptors, "lsa_external_counter")
		assert.Contains(t, descriptors, "lsa_as_opaque_counter")
	})

	t.Run("AreaDescriptors", func(t *testing.T) {
		descriptors := createOSPFAreaDescriptors()
		assert.Contains(t, descriptors, "area_lsa_number")
		assert.Contains(t, descriptors, "area_lsa_network_number")
	})
}

// 测试实例ID解析
func TestParseInstanceIDs(t *testing.T) {
	t.Run("ValidIDs", func(t *testing.T) {
		ids, err := parseInstanceIDs("1,2,3")
		require.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3}, ids)
	})

	t.Run("EmptyString", func(t *testing.T) {
		ids, err := parseInstanceIDs("")
		require.NoError(t, err)
		assert.Nil(t, ids)
	})

	t.Run("InvalidID", func(t *testing.T) {
		_, err := parseInstanceIDs("a,b,c")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unable to parse instance ID")
	})
}

// 测试命令执行错误处理
func TestCommandErrorHandling(t *testing.T) {
	collector := &OSPFCollector{
		logger:              slog.Default(),
		instanceIdentifiers: []int{1},
		commandExecutor:     &failingCommandExecutor{},
	}

	t.Run("RouterMetricsError", func(t *testing.T) {
		err := collector.collectRouterMetrics(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "command execution failed")
	})

	t.Run("InterfaceMetricsError", func(t *testing.T) {
		err := collector.collectInterfaceMetrics(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "command execution failed")
	})
}

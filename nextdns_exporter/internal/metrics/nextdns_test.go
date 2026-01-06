package metrics

import (
	"testing"
	"nextdns_exporter/internal/api"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockClient 实现 api.Client 接口，用于测试
type MockClient struct {
	mock.Mock
}

// 实现 Client 接口的所有方法
func (m *MockClient) CollectStatus() (*api.StatusMetrics, error) {
	args := m.Called()
	return args.Get(0).(*api.StatusMetrics), args.Error(1)
}

func (m *MockClient) CollectDomains() (*api.DomainsResponse, error) {
	args := m.Called()
	return args.Get(0).(*api.DomainsResponse), args.Error(1)
}

func (m *MockClient) CollectDevices() (*api.DevicesResponse, error) {
	args := m.Called()
	return args.Get(0).(*api.DevicesResponse), args.Error(1)
}

func (m *MockClient) CollectProtocols() (*api.ProtocolsResponse, error) {
	args := m.Called()
	return args.Get(0).(*api.ProtocolsResponse), args.Error(1)
}

func (m *MockClient) CollectQueryTypes() (*api.QueryTypesResponse, error) {
	args := m.Called()
	return args.Get(0).(*api.QueryTypesResponse), args.Error(1)
}

func (m *MockClient) CollectIPVersions() (*api.IPVersionsResponse, error) {
	args := m.Called()
	return args.Get(0).(*api.IPVersionsResponse), args.Error(1)
}

func (m *MockClient) CollectDNSSEC() (*api.DNSSECResponse, error) {
	args := m.Called()
	return args.Get(0).(*api.DNSSECResponse), args.Error(1)
}

func (m *MockClient) CollectEncryption() (*api.EncryptionResponse, error) {
	args := m.Called()
	return args.Get(0).(*api.EncryptionResponse), args.Error(1)
}

func (m *MockClient) CollectDestinations() (*api.DestinationsResponse, error) {
	args := m.Called()
	return args.Get(0).(*api.DestinationsResponse), args.Error(1)
}

// TestNewNextDNSMetrics 测试 NextDNSMetrics 的创建
func TestNewNextDNSMetrics(t *testing.T) {
	metrics := NewNextDNSMetrics("test_profile", "test_api_key")
	
	assert.NotNil(t, metrics)
	assert.Equal(t, "test_profile", metrics.profile)
	assert.Equal(t, "test_api_key", metrics.apiKey)
	assert.NotNil(t, metrics.client)
	assert.NotNil(t, metrics.logger)
	
	// 检查指标是否都被初始化
	assert.NotNil(t, metrics.totalQueries)
	assert.NotNil(t, metrics.totalAllowedQueries)
	assert.NotNil(t, metrics.totalBlockedQueries)
	assert.NotNil(t, metrics.blockedQueries)
	assert.NotNil(t, metrics.deviceQueries)
	assert.NotNil(t, metrics.protocolQueries)
	assert.NotNil(t, metrics.typeQueries)
	assert.NotNil(t, metrics.ipVersionQueries)
	assert.NotNil(t, metrics.dnssecQueries)
	assert.NotNil(t, metrics.encryptedQueries)
	assert.NotNil(t, metrics.destinationQueries)
}

// TestNextDNSMetricsDescribe 测试 Describe 方法
func TestNextDNSMetricsDescribe(t *testing.T) {
	metrics := NewNextDNSMetrics("test_profile", "test_api_key")
	
	ch := make(chan *prometheus.Desc, 20)
	
	// 调用 Describe 方法
	metrics.Describe(ch)
	
	// 应该有 12 个指标被发送到通道中 (1个基础指标 + 11个NextDNS指标)
	assert.Equal(t, 12, len(ch))
	
	// 检查所有的指标描述符
	descs := make([]*prometheus.Desc, 0, 12)
	for i := 0; i < 12; i++ {
		descs = append(descs, <-ch)
	}
	
	// 确认所有的描述符都不为空
	for _, desc := range descs {
		assert.NotNil(t, desc)
	}
}

// TestNextDNSMetricsCollect 测试 Collect 方法
func TestNextDNSMetricsCollect(t *testing.T) {
	// 创建模拟客户端
	mockClient := new(MockClient)
	
	// 设置模拟客户端的预期调用和返回值
	mockClient.On("CollectStatus").Return(&api.StatusMetrics{
		TotalQueries:   100,
		AllowedQueries: 80,
		BlockedQueries: 20,
	}, nil)
	
	mockClient.On("CollectDomains").Return(&api.DomainsResponse{
		BlockedDomains: []api.BlockedDomain{
			{
				Domain:  "example.com",
				Root:    "example.com",
				Tracker: "false",
				Queries: 10,
			},
		},
	}, nil)
	
	mockClient.On("CollectDevices").Return(&api.DevicesResponse{
		Devices: []api.Device{
			{
				ID:      "device1",
				Name:    "device1",
				Model:   "model1",
				LocalIP: "192.168.1.1",
				Queries: 50,
			},
		},
	}, nil)
	
	mockClient.On("CollectProtocols").Return(&api.ProtocolsResponse{
		Protocols: []api.Protocol{
			{
				Protocol: "protocol1",
				Queries:  30,
			},
		},
	}, nil)
	
	mockClient.On("CollectQueryTypes").Return(&api.QueryTypesResponse{
		QueryTypes: []api.QueryType{
			{
				Type:    "A",
				Name:    "A",
				Queries: 40,
			},
		},
	}, nil)
	
	mockClient.On("CollectIPVersions").Return(&api.IPVersionsResponse{
		IPVersions: []api.IPVersion{
			{
				Version: "ipv4",
				Queries: 60,
			},
		},
	}, nil)
	
	mockClient.On("CollectDNSSEC").Return(&api.DNSSECResponse{
		Data: []api.DNSSECData{
			{
				Validated: "true",
				Queries:   70,
			},
		},
	}, nil)
	
	mockClient.On("CollectEncryption").Return(&api.EncryptionResponse{
		Data: []api.EncryptionData{
			{
				Encrypted: "true",
				Queries:   80,
			},
		},
	}, nil)
	
	mockClient.On("CollectDestinations").Return(&api.DestinationsResponse{
		Destinations: []api.Destination{
			{
				Code:    "US",
				Name:    "United States",
				Queries: 90,
			},
		},
	}, nil)
	
	// 创建NextDNSMetrics实例
	metrics := NewNextDNSMetrics("test_profile", "test_api_key")
	// 替换Client为模拟客户端
	metrics.client = mockClient
	
	// 创建Registry并注册指标
	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics)
	
	// 收集指标
	collected, err := registry.Gather()
	
	// 检查结果
	assert.NoError(t, err)
	assert.NotEmpty(t, collected)
	
	// 验证模拟客户端的所有方法都被调用
	mockClient.AssertExpectations(t)
}

// TestNextDNSMetricsErrorHandling 测试错误处理
func TestNextDNSMetricsErrorHandling(t *testing.T) {
	// 创建模拟客户端，返回错误
	mockClient := new(MockClient)
	
	// 设置所有模拟方法都返回错误
	mockStatusMetrics := &api.StatusMetrics{}
	mockDomainsResponse := &api.DomainsResponse{}
	mockDevicesResponse := &api.DevicesResponse{}
	mockProtocolsResponse := &api.ProtocolsResponse{}
	mockQueryTypesResponse := &api.QueryTypesResponse{}
	mockIPVersionsResponse := &api.IPVersionsResponse{}
	mockDNSSECResponse := &api.DNSSECResponse{}
	mockEncryptionResponse := &api.EncryptionResponse{}
	mockDestinationsResponse := &api.DestinationsResponse{}
	
	mockClient.On("CollectStatus").Return(mockStatusMetrics, nil)
	mockClient.On("CollectDomains").Return(mockDomainsResponse, nil)
	mockClient.On("CollectDevices").Return(mockDevicesResponse, nil)
	mockClient.On("CollectProtocols").Return(mockProtocolsResponse, nil)
	mockClient.On("CollectQueryTypes").Return(mockQueryTypesResponse, nil)
	mockClient.On("CollectIPVersions").Return(mockIPVersionsResponse, nil)
	mockClient.On("CollectDNSSEC").Return(mockDNSSECResponse, nil)
	mockClient.On("CollectEncryption").Return(mockEncryptionResponse, nil)
	mockClient.On("CollectDestinations").Return(mockDestinationsResponse, nil)
	
	// 创建NextDNSMetrics实例
	metrics := NewNextDNSMetrics("test_profile", "test_api_key")
	metrics.client = mockClient
	
	// 创建Registry并注册指标
	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics)
	
	// 测试收集指标不应该崩溃
	assert.NotPanics(t, func() {
		registry.Gather()
	})
}

// TestLoggerAdapter 测试LoggerAdapter
func TestLoggerAdapter(t *testing.T) {
	// 创建真实的logger
	realLogger := logrus.New()
	logger := &LoggerAdapter{
		Logger: realLogger,
	}
	
	// 测试Error方法不应该崩溃
	assert.NotPanics(t, func() {
		logger.Error("test error message", "key", "value")
	})
} 
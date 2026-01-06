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

// TODO: implement functions

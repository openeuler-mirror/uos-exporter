package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"new/nextdns_exporter/internal/config"
	"new/nextdns_exporter/internal/model"
)

func setupTestServer(handler http.HandlerFunc) (*httptest.Server, *Client) {
	server := httptest.NewServer(handler)
	
	logger := logrus.New()
	client := NewClient("test_profile", "test_api_key", logger)
	client.baseURL = server.URL // 重写基础URL以使用测试服务器
	
	return server, client
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name   string
		config config.Config
	}{
		{
			name: "基本配置",
			config: config.Config{
				NextDNS: config.NextDNSConfig{
					APIKey:     "test-api-key",
					ProfileID:  "test-profile",
					BaseURL:    "https://api.nextdns.io",
					Timeout:    10,
					TimePeriod: 86400,
				},
			},
		},
		{
			name: "自定义超时",
			config: config.Config{
				NextDNS: config.NextDNSConfig{
					APIKey:     "test-api-key",
					ProfileID:  "test-profile",
					BaseURL:    "https://api.nextdns.io",
					Timeout:    20,
					TimePeriod: 86400,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.config)
			require.NotNil(t, client)
			require.Equal(t, tt.config.NextDNS.APIKey, client.apiKey)
			require.Equal(t, tt.config.NextDNS.ProfileID, client.profileID)
			require.Equal(t, tt.config.NextDNS.BaseURL, client.baseURL)

			// 检查HTTP客户端设置
			httpClient, ok := client.httpClient.(*http.Client)
			require.True(t, ok)
			assert.Equal(t, time.Duration(tt.config.NextDNS.Timeout)*time.Second, httpClient.Timeout)
		})
	}
}

func TestClient_CollectStatus(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 检查请求路径
		assert.Equal(t, "/api/status", r.URL.Path)
		
		// 返回模拟响应
		mockResponse := model.Status{
			TotalQueries:   100,
			AllowedQueries: 80,
			BlockedQueries: 20,
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()
	
	// 创建客户端
	client := NewClient(ClientConfig{
		BaseURL: server.URL,
		APIKey:  "test-api-key",
	})
	
	// 测试收集状态
	status, err := client.CollectStatus()
	
	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, 100, status.TotalQueries)
	assert.Equal(t, 80, status.AllowedQueries)
	assert.Equal(t, 20, status.BlockedQueries)
}

func TestClient_CollectDomains(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 检查请求路径
		assert.Equal(t, "/api/domains", r.URL.Path)
		
		// 返回模拟响应
		mockResponse := model.Domains{
			BlockedDomains: []model.Domain{
				{
					Domain:  "example.com",
					Root:    "example.com",
					Tracker: "false",
					Queries: 10,
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()
	
	// 创建客户端
	client := NewClient(ClientConfig{
		BaseURL: server.URL,
		APIKey:  "test-api-key",
	})
	
	// 测试收集域名
	domains, err := client.CollectDomains()
	
	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, domains)
	assert.Len(t, domains.BlockedDomains, 1)
	assert.Equal(t, "example.com", domains.BlockedDomains[0].Domain)
	assert.Equal(t, 10, domains.BlockedDomains[0].Queries)
}

func TestClient_CollectDevices(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 检查请求路径
		assert.Equal(t, "/api/devices", r.URL.Path)
		
		// 返回模拟响应
		mockResponse := model.Devices{
			Devices: []model.Device{
				{
					ID:      "device1",
					Name:    "Device 1",
					Model:   "Model 1",
					LocalIP: "192.168.1.1",
					Queries: 50,
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()
	
	// 创建客户端
	client := NewClient(ClientConfig{
		BaseURL: server.URL,
		APIKey:  "test-api-key",
	})
	
	// 测试收集设备
	devices, err := client.CollectDevices()
	
	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, devices)
	assert.Len(t, devices.Devices, 1)
	assert.Equal(t, "device1", devices.Devices[0].ID)
	assert.Equal(t, "Device 1", devices.Devices[0].Name)
	assert.Equal(t, 50, devices.Devices[0].Queries)
}

func TestClient_CollectProtocols(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 检查请求路径
		assert.Equal(t, "/api/protocols", r.URL.Path)
		
		// 返回模拟响应
		mockResponse := model.Protocols{
			Protocols: []model.Protocol{
				{
					Protocol: "doh",
					Queries:  30,
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()
	
	// 创建客户端
	client := NewClient(ClientConfig{
		BaseURL: server.URL,
		APIKey:  "test-api-key",
	})
	
	// 测试收集协议
	protocols, err := client.CollectProtocols()
	
	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, protocols)
	assert.Len(t, protocols.Protocols, 1)
	assert.Equal(t, "doh", protocols.Protocols[0].Protocol)
	assert.Equal(t, 30, protocols.Protocols[0].Queries)
}

func TestClient_CollectQueryTypes(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 检查请求路径
		assert.Equal(t, "/api/query-types", r.URL.Path)
		
		// 返回模拟响应
		mockResponse := model.QueryTypes{
			QueryTypes: []model.QueryType{
				{
					Type:    "A",
					Name:    "A",
					Queries: 40,
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()
	
	// 创建客户端
	client := NewClient(ClientConfig{
		BaseURL: server.URL,
		APIKey:  "test-api-key",
	})
	
	// 测试收集查询类型
	queryTypes, err := client.CollectQueryTypes()
	
	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, queryTypes)
	assert.Len(t, queryTypes.QueryTypes, 1)
	assert.Equal(t, "A", queryTypes.QueryTypes[0].Type)
	assert.Equal(t, 40, queryTypes.QueryTypes[0].Queries)
}

func TestClient_CollectIPVersions(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 检查请求路径
		assert.Equal(t, "/api/ip-versions", r.URL.Path)
		
		// 返回模拟响应
		mockResponse := model.IPVersions{
			IPVersions: []model.IPVersion{
				{
					Version: "ipv4",
					Queries: 60,
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()
	
	// 创建客户端
	client := NewClient(ClientConfig{
		BaseURL: server.URL,
		APIKey:  "test-api-key",
	})
	
	// 测试收集IP版本
	ipVersions, err := client.CollectIPVersions()
	
	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, ipVersions)
	assert.Len(t, ipVersions.IPVersions, 1)
	assert.Equal(t, "ipv4", ipVersions.IPVersions[0].Version)
	assert.Equal(t, 60, ipVersions.IPVersions[0].Queries)
}

func TestClient_CollectDNSSEC(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 检查请求路径
		assert.Equal(t, "/api/dnssec", r.URL.Path)
		
		// 返回模拟响应
		mockResponse := model.DNSSEC{
			Data: []model.DNSSECItem{
				{
					Validated: "true",
					Queries:   70,
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()
	
	// 创建客户端
	client := NewClient(ClientConfig{
		BaseURL: server.URL,
		APIKey:  "test-api-key",
	})
	
	// 测试收集DNSSEC
	dnssec, err := client.CollectDNSSEC()
	
	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, dnssec)
	assert.Len(t, dnssec.Data, 1)
	assert.Equal(t, "true", dnssec.Data[0].Validated)
	assert.Equal(t, 70, dnssec.Data[0].Queries)
}

func TestClient_CollectEncryption(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 检查请求路径
		assert.Equal(t, "/api/encryption", r.URL.Path)
		
		// 返回模拟响应
		mockResponse := model.Encryption{
			Data: []model.EncryptionItem{
				{
					Encrypted: "true",
					Queries:   80,
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()
	
	// 创建客户端
	client := NewClient(ClientConfig{
		BaseURL: server.URL,
		APIKey:  "test-api-key",
	})
	
	// 测试收集加密状态
	encryption, err := client.CollectEncryption()
	
	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, encryption)
	assert.Len(t, encryption.Data, 1)
	assert.Equal(t, "true", encryption.Data[0].Encrypted)
	assert.Equal(t, 80, encryption.Data[0].Queries)
}

func TestClient_CollectDestinations(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 检查请求路径
		assert.Equal(t, "/api/destinations", r.URL.Path)
		
		// 返回模拟响应
		mockResponse := model.Destinations{
			Destinations: []model.Destination{
				{
					Code:    "US",
					Name:    "United States",
					Queries: 90,
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()
	
	// 创建客户端
	client := NewClient(ClientConfig{
		BaseURL: server.URL,
		APIKey:  "test-api-key",
	})
	
	// 测试收集目的地
	destinations, err := client.CollectDestinations()
	
	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, destinations)
	assert.Len(t, destinations.Destinations, 1)
	assert.Equal(t, "US", destinations.Destinations[0].Code)
	assert.Equal(t, "United States", destinations.Destinations[0].Name)
	assert.Equal(t, 90, destinations.Destinations[0].Queries)
}

func TestClient_APIError(t *testing.T) {
	// 创建模拟服务器返回错误
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Internal Server Error",
		})
	}))
	defer server.Close()
	
	// 创建客户端
	client := NewClient(ClientConfig{
		BaseURL: server.URL,
		APIKey:  "test-api-key",
	})
	
	// 测试API错误处理
	_, err := client.CollectStatus()
	
	// 验证结果
	assert.Error(t, err)
	apiErr, ok := err.(*APIError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusInternalServerError, apiErr.Code)
}

func TestRequestError(t *testing.T) {
	// 创建一个返回错误的测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	
	logger := logrus.New()
	client := NewClient("test_profile", "test_api_key", logger)
	client.baseURL = server.URL // 重写基础URL以使用测试服务器
	
	// 测试各种方法在服务器错误时的行为
	_, err := client.CollectStatus()
	assert.Error(t, err)
	
	_, err = client.CollectDomains()
	assert.Error(t, err)
	
	_, err = client.CollectDevices()
	assert.Error(t, err)
	
	_, err = client.CollectProtocols()
	assert.Error(t, err)
	
	_, err = client.CollectQueryTypes()
	assert.Error(t, err)
	
	_, err = client.CollectIPVersions()
	assert.Error(t, err)
	
	_, err = client.CollectDNSSEC()
	assert.Error(t, err)
	
	_, err = client.CollectEncryption()
	assert.Error(t, err)
	
	_, err = client.CollectDestinations()
	assert.Error(t, err)
}

func TestInvalidJSONResponse(t *testing.T) {
	// 创建一个返回无效JSON的测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json}`))
	}))
	defer server.Close()
	
	logger := logrus.New()
	client := NewClient("test_profile", "test_api_key", logger)
	client.baseURL = server.URL // 重写基础URL以使用测试服务器
	
	// 测试各种方法在无效JSON时的行为
	_, err := client.CollectStatus()
	assert.Error(t, err)
	
	_, err = client.CollectDomains()
	assert.Error(t, err)
	
	_, err = client.CollectDevices()
	assert.Error(t, err)
	
	_, err = client.CollectProtocols()
	assert.Error(t, err)
	
	_, err = client.CollectQueryTypes()
	assert.Error(t, err)
	
	_, err = client.CollectIPVersions()
	assert.Error(t, err)
	
	_, err = client.CollectDNSSEC()
	assert.Error(t, err)
	
	_, err = client.CollectEncryption()
	assert.Error(t, err)
	
	_, err = client.CollectDestinations()
	assert.Error(t, err)
}

func TestClient_GetStatusMetrics(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/profiles/test-profile/status", r.URL.Path)
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

		response := StatusResponse{
			Status: "active",
			DNSQueries: &DNSQueries{
				Total:   100,
				Blocked: 20,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// 创建客户端
	cfg := config.Config{
		NextDNS: config.NextDNSConfig{
			APIKey:     "test-api-key",
			ProfileID:  "test-profile",
			BaseURL:    server.URL,
			Timeout:    10,
			TimePeriod: 86400,
		},
	}

	client := NewClient(cfg)
	client.baseURL = server.URL // 使用测试服务器URL

	// 获取状态指标
	resp, err := client.GetStatusMetrics(context.Background())
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "active", resp.Status)
	assert.Equal(t, int64(100), resp.DNSQueries.Total)
	assert.Equal(t, int64(20), resp.DNSQueries.Blocked)
}

func TestClient_GetDomainsMetrics(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/profiles/test-profile/analytics/domains", r.URL.Path)
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
		assert.Equal(t, "86400", r.URL.Query().Get("from"))

		response := DomainsResponse{
			Domains: []Domain{
				{
					Name:       "example.com",
					Count:      50,
					Percentage: 50.0,
					Blocked:    10,
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// 创建客户端
	cfg := config.Config{
		NextDNS: config.NextDNSConfig{
			APIKey:     "test-api-key",
			ProfileID:  "test-profile",
			BaseURL:    server.URL,
			Timeout:    10,
			TimePeriod: 86400,
		},
	}

	client := NewClient(cfg)
	client.baseURL = server.URL // 使用测试服务器URL

	// 获取域名指标
	params := &QueryParams{
		From: 86400,
	}
	resp, err := client.GetDomainsMetrics(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.Domains, 1)
	assert.Equal(t, "example.com", resp.Domains[0].Name)
	assert.Equal(t, int64(50), resp.Domains[0].Count)
	assert.Equal(t, float64(50.0), resp.Domains[0].Percentage)
	assert.Equal(t, int64(10), resp.Domains[0].Blocked)
}

func TestClient_buildURL(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		path     string
		params   *QueryParams
		expected string
	}{
		{
			name:     "基本URL不带参数",
			baseURL:  "https://api.nextdns.io",
			path:     "/profiles/profile1/status",
			params:   nil,
			expected: "https://api.nextdns.io/profiles/profile1/status",
		},
		{
			name:    "带时间参数",
			baseURL: "https://api.nextdns.io",
			path:    "/profiles/profile1/analytics/domains",
			params: &QueryParams{
				From: 86400,
			},
			expected: "https://api.nextdns.io/profiles/profile1/analytics/domains?from=86400",
		},
		{
			name:    "带多个参数",
			baseURL: "https://api.nextdns.io",
			path:    "/profiles/profile1/analytics/domains",
			params: &QueryParams{
				From:  86400,
				Until: 0,
				Limit: 10,
			},
			expected: "https://api.nextdns.io/profiles/profile1/analytics/domains?from=86400&limit=10&until=0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				baseURL: tt.baseURL,
			}
			result := client.buildURL(tt.path, tt.params)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClient_handleAPIError(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedErrMsg string
	}{
		{
			name:           "未授权错误",
			statusCode:     401,
			responseBody:   `{"message": "Unauthorized access"}`,
			expectedErrMsg: "API请求失败: 状态码: 401, 响应: {\"message\": \"Unauthorized access\"}",
		},
		{
			name:           "服务器错误",
			statusCode:     500,
			responseBody:   `{"message": "Internal server error"}`,
			expectedErrMsg: "API请求失败: 状态码: 500, 响应: {\"message\": \"Internal server error\"}",
		},
		{
			name:           "无响应体",
			statusCode:     404,
			responseBody:   "",
			expectedErrMsg: "API请求失败: 状态码: 404, 响应: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建模拟响应
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Body:       &mockReadCloser{str: tt.responseBody},
			}

			client := &Client{}
			err := client.handleAPIError(resp)
			assert.Error(t, err)
			assert.Equal(t, tt.expectedErrMsg, err.Error())
		})
	}
}

// 模拟io.ReadCloser接口
type mockReadCloser struct {
	str string
	pos int
}

func (m *mockReadCloser) Read(p []byte) (n int, err error) {
	if m.pos >= len(m.str) {
		return 0, nil
	}
	n = copy(p, m.str[m.pos:])
	m.pos += n
	return n, nil
}

func (m *mockReadCloser) Close() error {
	return nil
}
// Part 2 commit for nextdns_exporter/internal/api/client_test.go

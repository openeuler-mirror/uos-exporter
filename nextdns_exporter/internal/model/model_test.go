package model

import (
	"testing"
	
	"github.com/stretchr/testify/assert"
)

func TestStatusData(t *testing.T) {
	// 创建一个Status对象
	status := Status{
		TotalQueries:   100,
		AllowedQueries: 80,
		BlockedQueries: 20,
	}
	
	// 验证字段值
	assert.Equal(t, int64(100), status.TotalQueries)
	assert.Equal(t, int64(80), status.AllowedQueries)
	assert.Equal(t, int64(20), status.BlockedQueries)
}

func TestDomainsData(t *testing.T) {
	// 创建一个Domains对象
	domains := Domains{
		BlockedDomains: []Domain{
			{
				Domain:  "example.com",
				Root:    "example.com",
				Tracker: "false",
				Queries: 10,
			},
		},
	}
	
	// 验证字段值
	assert.Len(t, domains.BlockedDomains, 1)
	assert.Equal(t, "example.com", domains.BlockedDomains[0].Domain)
	assert.Equal(t, "example.com", domains.BlockedDomains[0].Root)
	assert.Equal(t, "false", domains.BlockedDomains[0].Tracker)
	assert.Equal(t, int64(10), domains.BlockedDomains[0].Queries)
}

func TestDevicesData(t *testing.T) {
	// 创建一个Devices对象
	devices := Devices{
		Devices: []Device{
			{
				ID:      "device1",
				Name:    "Device 1",
				Model:   "Model 1",
				LocalIP: "192.168.1.1",
				Queries: 50,
			},
		},
	}
	
	// 验证字段值
	assert.Len(t, devices.Devices, 1)
	assert.Equal(t, "device1", devices.Devices[0].ID)
	assert.Equal(t, "Device 1", devices.Devices[0].Name)
	assert.Equal(t, "Model 1", devices.Devices[0].Model)
	assert.Equal(t, "192.168.1.1", devices.Devices[0].LocalIP)
	assert.Equal(t, int64(50), devices.Devices[0].Queries)
}

func TestProtocolsData(t *testing.T) {
	// 创建一个Protocols对象
	protocols := Protocols{
		Protocols: []Protocol{
			{
				Protocol: "doh",
				Queries:  30,
			},
		},
	}
	
	// 验证字段值
	assert.Len(t, protocols.Protocols, 1)
	assert.Equal(t, "doh", protocols.Protocols[0].Protocol)
	assert.Equal(t, int64(30), protocols.Protocols[0].Queries)
}

func TestQueryTypesData(t *testing.T) {
	// 创建一个QueryTypes对象
	queryTypes := QueryTypes{
		QueryTypes: []QueryType{
			{
				Type:    "A",
				Name:    "A",
				Queries: 40,
			},
		},
	}
	
	// 验证字段值
	assert.Len(t, queryTypes.QueryTypes, 1)
	assert.Equal(t, "A", queryTypes.QueryTypes[0].Type)
	assert.Equal(t, "A", queryTypes.QueryTypes[0].Name)
	assert.Equal(t, int64(40), queryTypes.QueryTypes[0].Queries)
}

func TestIPVersionsData(t *testing.T) {
	// 创建一个IPVersions对象
	ipVersions := IPVersions{
		IPVersions: []IPVersion{
			{
				Version: "ipv4",
				Queries: 60,
			},
		},
	}
	
	// 验证字段值
	assert.Len(t, ipVersions.IPVersions, 1)
	assert.Equal(t, "ipv4", ipVersions.IPVersions[0].Version)
	assert.Equal(t, int64(60), ipVersions.IPVersions[0].Queries)
}

func TestDNSSECData(t *testing.T) {
	// 创建一个DNSSEC对象
	dnssec := DNSSEC{
		Data: []DNSSECItem{
			{
				Validated: "true",
				Queries:   70,
			},
		},
	}
	
	// 验证字段值
	assert.Len(t, dnssec.Data, 1)
	assert.Equal(t, "true", dnssec.Data[0].Validated)
	assert.Equal(t, int64(70), dnssec.Data[0].Queries)
}

func TestEncryptionData(t *testing.T) {
	// 创建一个Encryption对象
	encryption := Encryption{
		Data: []EncryptionItem{
			{
				Encrypted: "true",
				Queries:   80,
			},
		},
	}
	
	// 验证字段值
	assert.Len(t, encryption.Data, 1)
	assert.Equal(t, "true", encryption.Data[0].Encrypted)
	assert.Equal(t, int64(80), encryption.Data[0].Queries)
}

func TestDestinationsData(t *testing.T) {
	// 创建一个Destinations对象
	destinations := Destinations{
		Destinations: []Destination{
			{
				Code:    "US",
				Name:    "United States",
				Queries: 90,
			},
		},
	}
	
	// 验证字段值
	assert.Len(t, destinations.Destinations, 1)
	assert.Equal(t, "US", destinations.Destinations[0].Code)
	assert.Equal(t, "United States", destinations.Destinations[0].Name)
	assert.Equal(t, int64(90), destinations.Destinations[0].Queries)
} 
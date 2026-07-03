package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	BaseURL = "https://api.nextdns.io"
)

// Logger interface for logging
type Logger interface {
	Error(msg string, keysAndValues ...interface{})
}

// Client represents a NextDNS API client
type Client interface {
	CollectStatus() (*StatusMetrics, error)
	CollectDomains() (*DomainsResponse, error)
	CollectDevices() (*DevicesResponse, error)
	CollectProtocols() (*ProtocolsResponse, error)
	CollectQueryTypes() (*QueryTypesResponse, error)
	CollectIPVersions() (*IPVersionsResponse, error)
	CollectDNSSEC() (*DNSSECResponse, error)
	CollectEncryption() (*EncryptionResponse, error)
	CollectDestinations() (*DestinationsResponse, error)
}

// HTTPClient represents an HTTP client
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// NextDNSClient implements Client
type NextDNSClient struct {
	baseURL  string
	profile  string
	apiKey   string
	client   HTTPClient
	logger   Logger
}

// StatusMetrics represents the NextDNS status
type StatusMetrics struct {
	TotalQueries   float64
	AllowedQueries float64
	BlockedQueries float64
}

// StatusResponse represents the NextDNS status response
type StatusResponse struct {
	Statuses []Status `json:"data"`
}

// Status represents a status item
type Status struct {
	Status  string `json:"status"`
	Queries int    `json:"queries"`
}

// DomainsResponse represents the NextDNS domains response
type DomainsResponse struct {
	BlockedDomains []BlockedDomain `json:"data"`
}

// BlockedDomain represents a blocked domain
type BlockedDomain struct {
	Domain  string `json:"domain"`
	Root    string `json:"root"`
	Tracker string `json:"tracker"`
	Queries float64 `json:"queries"`
}

// DevicesResponse represents the NextDNS devices response
type DevicesResponse struct {
	Devices []Device `json:"data"`
}

// Device represents a device
type Device struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Model   string  `json:"model"`
	LocalIP string  `json:"localip"`
	Queries float64 `json:"queries"`
}

// ProtocolsResponse represents the NextDNS protocols response
type ProtocolsResponse struct {
	Protocols []Protocol `json:"data"`
}

// Protocol represents a protocol
type Protocol struct {
	Protocol string  `json:"protocol"`
	Queries  float64 `json:"queries"`
}

// QueryTypesResponse represents the NextDNS query types response
type QueryTypesResponse struct {
	QueryTypes []QueryType `json:"data"`
}

// QueryType represents a query type
type QueryType struct {
	Type    string  `json:"type"`
	Name    string  `json:"name"`
	Queries float64 `json:"queries"`
}

// IPVersionsResponse represents the NextDNS IP versions response
type IPVersionsResponse struct {
	IPVersions []IPVersion `json:"data"`
}

// IPVersion represents an IP version
type IPVersion struct {
	Version string  `json:"version"`
	Queries float64 `json:"queries"`
}

// DNSSECResponse represents the NextDNS DNSSEC response
type DNSSECResponse struct {
	Data []DNSSECData `json:"data"`
}

// DNSSECData represents DNSSEC data
type DNSSECData struct {
	Validated string  `json:"validated"`
	Queries   float64 `json:"queries"`
}

// EncryptionResponse represents the NextDNS encryption response
type EncryptionResponse struct {
	Data []EncryptionData `json:"data"`
}

// EncryptionData represents encryption data
type EncryptionData struct {
	Encrypted string  `json:"encrypted"`
	Queries   float64 `json:"queries"`
}

// DestinationsResponse represents the NextDNS destinations response
type DestinationsResponse struct {
	Destinations []Destination `json:"data"`
}

// Destination represents a destination
type Destination struct {
	Code    string  `json:"code"`
	Name    string  `json:"name"`
	Queries float64 `json:"queries"`
}

// NewClient creates a new NextDNS API client
func NewClient(baseURL, profile, apiKey string, logger Logger) Client {
	return &NextDNSClient{
		baseURL: baseURL,
		profile: profile,
		apiKey:  apiKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

// Request makes a request to the NextDNS API
func (c *NextDNSClient) Request(uri string, params url.Values) ([]byte, error) {
	query := url.Values{
		// API要求from是数字，以秒为单位，这里设置为24小时 = 86400秒
		"from":  {"86400"},  // 24 hours in seconds
		"limit": {"100"},    // Up to 100 results
	}
	for k, v := range params {
		query.Add(k, v[0])
	}
	
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		c.logger.Error("error creating request", "error", err)
		return nil, err
	}
	req.Header.Set("X-Api-Key", c.apiKey)
	req.URL.RawQuery = query.Encode()
	
	resp, err := c.client.Do(req)
	if err != nil {
		c.logger.Error("error making request", "error", err)
		return nil, err
	}
	defer resp.Body.Close()
	
	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.logger.Error("error from API request", "status_code", resp.StatusCode, "response", string(body))
		return nil, fmt.Errorf("API request failed with status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("error reading response body", "error", err)
		return nil, err
	}

	return body, nil
}

// 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// CollectStatus collects status information from the NextDNS API
func (c *NextDNSClient) CollectStatus() (*StatusMetrics, error) {
	baseURL := c.baseURL
	// 确保baseURL是正确的格式
	if baseURL == "" {
		baseURL = "https://api.nextdns.io"
	}
	statusURL := fmt.Sprintf("%s/profiles/%s/analytics/status", baseURL, c.profile)

	statusResponse := StatusResponse{}
	metrics := StatusMetrics{}

	body, err := c.Request(statusURL, nil)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &statusResponse)
	if err != nil {
		c.logger.Error("error unmarshaling status response", "error", err)
		return nil, err
	}

	for _, status := range statusResponse.Statuses {
		switch status.Status {
		case "default":
			metrics.TotalQueries = float64(status.Queries)
		case "allowed":
			metrics.AllowedQueries = float64(status.Queries)
		case "blocked":
			metrics.BlockedQueries = float64(status.Queries)
		}
	}

	return &metrics, nil
}

// CollectDomains collects domains information from the NextDNS API
func (c *NextDNSClient) CollectDomains() (*DomainsResponse, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://api.nextdns.io"
	}
	domainsURL := fmt.Sprintf("%s/profiles/%s/analytics/domains", baseURL, c.profile)

	domainsResponse := DomainsResponse{}

	body, err := c.Request(domainsURL, nil)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &domainsResponse)
	if err != nil {
		c.logger.Error("error unmarshaling domains response", "error", err)
		return nil, err
	}

	return &domainsResponse, nil
}

// CollectDevices collects devices information from the NextDNS API
func (c *NextDNSClient) CollectDevices() (*DevicesResponse, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://api.nextdns.io"
	}
	devicesURL := fmt.Sprintf("%s/profiles/%s/analytics/devices", baseURL, c.profile)

	devicesResponse := DevicesResponse{}

	body, err := c.Request(devicesURL, nil)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &devicesResponse)
	if err != nil {
		c.logger.Error("error unmarshaling devices response", "error", err)
		return nil, err
	}

	return &devicesResponse, nil
}

// CollectProtocols collects protocols information from the NextDNS API
func (c *NextDNSClient) CollectProtocols() (*ProtocolsResponse, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://api.nextdns.io"
	}
	protocolsURL := fmt.Sprintf("%s/profiles/%s/analytics/protocols", baseURL, c.profile)

	protocolsResponse := ProtocolsResponse{}

	body, err := c.Request(protocolsURL, nil)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &protocolsResponse)
	if err != nil {
		c.logger.Error("error unmarshaling protocols response", "error", err)
		return nil, err
	}

	return &protocolsResponse, nil
}

// CollectQueryTypes collects query types information from the NextDNS API
func (c *NextDNSClient) CollectQueryTypes() (*QueryTypesResponse, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://api.nextdns.io"
	}
	queryTypesURL := fmt.Sprintf("%s/profiles/%s/analytics/queryTypes", baseURL, c.profile)

	queryTypesResponse := QueryTypesResponse{}

	body, err := c.Request(queryTypesURL, nil)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &queryTypesResponse)
	if err != nil {
		c.logger.Error("error unmarshaling query types response", "error", err)
		return nil, err
	}

	return &queryTypesResponse, nil
}

// CollectIPVersions collects IP versions information from the NextDNS API
func (c *NextDNSClient) CollectIPVersions() (*IPVersionsResponse, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://api.nextdns.io"
	}
	ipVersionsURL := fmt.Sprintf("%s/profiles/%s/analytics/ipVersions", baseURL, c.profile)

	ipVersionsResponse := IPVersionsResponse{}

	body, err := c.Request(ipVersionsURL, nil)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &ipVersionsResponse)
	if err != nil {
		c.logger.Error("error unmarshaling IP versions response", "error", err)
		return nil, err
	}

	return &ipVersionsResponse, nil
}

// CollectDNSSEC collects DNSSEC information from the NextDNS API
func (c *NextDNSClient) CollectDNSSEC() (*DNSSECResponse, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://api.nextdns.io"
	}
	dnssecURL := fmt.Sprintf("%s/profiles/%s/analytics/dnssec", baseURL, c.profile)

	dnssecResponse := DNSSECResponse{}

	body, err := c.Request(dnssecURL, nil)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &dnssecResponse)
	if err != nil {
		c.logger.Error("error unmarshaling DNSSEC response", "error", err)
		return nil, err
	}

	return &dnssecResponse, nil
}

// CollectEncryption collects encryption information from the NextDNS API
func (c *NextDNSClient) CollectEncryption() (*EncryptionResponse, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://api.nextdns.io"
	}
	encryptionURL := fmt.Sprintf("%s/profiles/%s/analytics/encryption", baseURL, c.profile)

	encryptionResponse := EncryptionResponse{}

	body, err := c.Request(encryptionURL, nil)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &encryptionResponse)
	if err != nil {
		c.logger.Error("error unmarshaling encryption response", "error", err)
		return nil, err
	}

	return &encryptionResponse, nil
}

// CollectDestinations collects destinations information from the NextDNS API
func (c *NextDNSClient) CollectDestinations() (*DestinationsResponse, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://api.nextdns.io"
	}
	
	// 添加必需的 type 参数，根据旧项目的实现是 'countries'
	params := url.Values{}
	params.Add("type", "countries")
	
	destinationsURL := fmt.Sprintf("%s/profiles/%s/analytics/destinations", baseURL, c.profile)

	destinationsResponse := DestinationsResponse{}

	body, err := c.Request(destinationsURL, params)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &destinationsResponse)
	if err != nil {
		c.logger.Error("error unmarshaling destinations response", "error", err)
		return nil, err
	}

	return &destinationsResponse, nil
}
// Part 2 commit for nextdns_exporter/internal/api/client.go

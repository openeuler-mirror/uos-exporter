package metrics

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewLicense(t *testing.T) {
	l := NewLicense()
	assert.NotNil(t, l)
	assert.Equal(t, "http://localhost:9200", l.esURL)
	assert.False(t, l.insecure)
	assert.NotNil(t, l.jsonParseFailures)
	assert.NotNil(t, l.licenseInfo)
	assert.NotNil(t, l.licenseExpiryDate)
	assert.NotNil(t, l.licenseExpirySeconds)
}

func TestLicenseFetchAndDecodeLicense(t *testing.T) {
	// 创建模拟的许可证响应
	mockLicense := LicenseResponse{
		License: LicenseInfo{
			Status:             "active",
			UID:                "abcd-1234-efgh-5678",
			Type:               "basic",
			IssueDate:          "2023-01-01T00:00:00.000Z",
			IssueDateInMillis:  1672531200000,
			ExpiryDate:         "2024-01-01T00:00:00.000Z",
			ExpiryDateInMillis: 1704067200000,
			MaxNodes:           100,
			IssuedTo:           "test-customer",
			Issuer:             "elasticsearch",
			StartDateInMillis:  1672531200000,
		},
	}

	// 创建模拟服务器
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/_license") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			data, _ := json.Marshal(mockLicense)
			w.Write(data)
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "Not found")
		}
	}))
	defer ts.Close()

	// 测试成功的响应
	l := NewLicense()
	l.esURL = ts.URL
	resp, err := l.fetchAndDecodeLicense()
	assert.NoError(t, err)
	assert.Equal(t, "active", resp.License.Status)
	assert.Equal(t, "abcd-1234-efgh-5678", resp.License.UID)
	assert.Equal(t, "basic", resp.License.Type)
	assert.Equal(t, int64(1672531200000), resp.License.IssueDateInMillis)
	assert.Equal(t, int64(1704067200000), resp.License.ExpiryDateInMillis)
	assert.Equal(t, 100, resp.License.MaxNodes)
	assert.Equal(t, "test-customer", resp.License.IssuedTo)
	assert.Equal(t, "elasticsearch", resp.License.Issuer)

	// 测试服务器错误
	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Internal Server Error")
	}))
	defer ts2.Close()

	l.esURL = ts2.URL
	_, err = l.fetchAndDecodeLicense()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP Request failed with code 500")

	// 测试无效的JSON响应
	ts3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "invalid json data")
	}))
	defer ts3.Close()

	l.esURL = ts3.URL
	_, err = l.fetchAndDecodeLicense()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

func TestLicenseCollect(t *testing.T) {
	// 设置当前时间为固定值，以便测试到期时间计算
	currentTime := time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)
	// 计算到期时间（还有7个月）
	expiryTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	expiryTimeInMillis := expiryTime.UnixNano() / int64(time.Millisecond)
	expectedExpirySeconds := expiryTime.Sub(currentTime).Seconds()

	// 创建模拟的许可证响应
	mockLicense := LicenseResponse{
		License: LicenseInfo{
			Status:             "active",
			UID:                "abcd-1234-efgh-5678",
			Type:               "basic",
			IssueDate:          "2023-01-01T00:00:00.000Z",
			IssueDateInMillis:  1672531200000,
			ExpiryDate:         "2024-01-01T00:00:00.000Z",
			ExpiryDateInMillis: expiryTimeInMillis,
			MaxNodes:           100,
			IssuedTo:           "test-customer",
			Issuer:             "elasticsearch",
			StartDateInMillis:  1672531200000,
		},
	}

	// 创建模拟服务器
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/_license") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			data, _ := json.Marshal(mockLicense)
			w.Write(data)
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "Not found")
		}
	}))
	defer ts.Close()

	// 设置日志级别
	logrus.SetLevel(logrus.DebugLevel)

	// 测试 Collect 方法
	l := NewLicense()
	l.esURL = ts.URL

	// 创建注册表
	registry := prometheus.NewRegistry()
	registry.MustRegister(l)

	// 验证指标
	expected := `
# HELP elasticsearch_license_expiry_date_seconds License expiry date in seconds since epoch
# TYPE elasticsearch_license_expiry_date_seconds gauge
elasticsearch_license_expiry_date_seconds{cluster="elasticsearch"} %d
# HELP elasticsearch_license_expiry_seconds License expiry time in seconds
# TYPE elasticsearch_license_expiry_seconds gauge
elasticsearch_license_expiry_seconds{cluster="elasticsearch"} %f
# HELP elasticsearch_license_info License information
# TYPE elasticsearch_license_info gauge
elasticsearch_license_info{cluster="elasticsearch",issued_to="test-customer",issuer="elasticsearch",max_nodes="100",status="active",type="basic",uid="abcd-1234-efgh-5678"} 1
# HELP elasticsearch_license_json_parse_failures Number of errors while parsing JSON.
# TYPE elasticsearch_license_json_parse_failures counter
elasticsearch_license_json_parse_failures 0
`
	expected = fmt.Sprintf(expected, expiryTimeInMillis/1000, expectedExpirySeconds)

	// 临时覆盖时间函数用于测试
	oldTimeNow := timeNow
	timeNow = func() time.Time {
		return currentTime
	}
	defer func() { timeNow = oldTimeNow }()

	err := testutil.GatherAndCompare(registry, strings.NewReader(expected))
	assert.NoError(t, err)

	// 测试服务器错误时的 Collect
	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Internal Server Error")
	}))
	defer ts2.Close()

	l2 := NewLicense()
	l2.esURL = ts2.URL

	// 捕获日志输出
	var logOutput strings.Builder
	logrus.SetOutput(io.MultiWriter(&logOutput))
	defer func() {
		logrus.SetOutput(io.Discard)
	}()

	registry2 := prometheus.NewRegistry()
	registry2.MustRegister(l2)

	// 测试日志输出
	_, err = testutil.GatherAndCount(registry2)
	assert.NoError(t, err)
	assert.Contains(t, logOutput.String(), "Failed to fetch and decode license")
}
// Part 2 commit for elasticsearch_exporter/internal/metrics/elasticsearch_license_test.go

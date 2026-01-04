package metrics

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"

	"elasticsearch_exporter/config"
	"elasticsearch_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// 全局变量，用于测试
var timeNow = time.Now

// LicenseInfo 表示 Elasticsearch License 信息
type LicenseInfo struct {
	Status             string `json:"status"`
	UID                string `json:"uid"`
	Type               string `json:"type"`
	IssueDate          string `json:"issue_date"`
	IssueDateInMillis  int64  `json:"issue_date_in_millis"`
	ExpiryDate         string `json:"expiry_date"`
	ExpiryDateInMillis int64  `json:"expiry_date_in_millis"`
	MaxNodes           int    `json:"max_nodes"`
	IssuedTo           string `json:"issued_to"`
	Issuer             string `json:"issuer"`
	StartDateInMillis  int64  `json:"start_date_in_millis"`
}

// LicenseResponse 表示 Elasticsearch License API 响应
type LicenseResponse struct {
	License LicenseInfo `json:"license"`
}

// License 是 Elasticsearch License 信息的收集器
type License struct {
	esURL             string
	client            *http.Client
	insecure          bool
	jsonParseFailures prometheus.Counter

	// License 指标
	licenseInfo        *prometheus.Desc
	licenseExpiryDate  *prometheus.Desc
	licenseExpirySeconds *prometheus.Desc
}

func init() {
	exporter.Register(NewLicense())
}

// NewLicense 创建新的 License 收集器
func NewLicense() *License {
	return &License{
		esURL: "http://localhost:9200",
		client: &http.Client{
			Timeout: defaultTimeout,
		},
		jsonParseFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "license_json_parse_failures",
			Help:      "Number of errors while parsing JSON.",
		}),
		licenseInfo: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "license", "info"),
			"License information",
			[]string{"cluster", "status", "uid", "type", "issued_to", "issuer", "max_nodes"},
			nil,
		),
		licenseExpiryDate: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "license", "expiry_date_seconds"),
			"License expiry date in seconds since epoch",
			[]string{"cluster"},
			nil,
		),
		licenseExpirySeconds: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "license", "expiry_seconds"),
			"License expiry time in seconds",
			[]string{"cluster"},
			nil,
		),
	}
}

// fetchAndDecodeLicense 获取并解析 License 信息
func (l *License) fetchAndDecodeLicense() (LicenseResponse, error) {
	var licenseResp LicenseResponse

	// 确保客户端每次获取时更新配置
	l.client = &http.Client{
		Timeout: defaultTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				// #nosec G402
				InsecureSkipVerify: l.insecure,
			},
		},
	}

	u, err := url.Parse(l.esURL)
	if err != nil {
		return LicenseResponse{}, fmt.Errorf("failed to parse ES URL: %s", err)
	}

	// 构建URL路径
	u.Path = path.Join(u.Path, "/_license")

	logrus.Debugf("Fetching license from %s", u.String())

	res, err := l.client.Get(u.String())
	if err != nil {
		return LicenseResponse{}, fmt.Errorf("failed to get license from %s: %s", u.String(), err)
	}

	defer func() {
		err = res.Body.Close()
		if err != nil {
			logrus.Warnf("failed to close http.Client: %s", err)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return LicenseResponse{}, fmt.Errorf("HTTP Request failed with code %d", res.StatusCode)
	}

	if err := json.NewDecoder(res.Body).Decode(&licenseResp); err != nil {
		l.jsonParseFailures.Inc()
		return LicenseResponse{}, err
	}

	return licenseResp, nil
}

// Collect 实现指标收集
func (l *License) Collect(ch chan<- prometheus.Metric) {
	// 从配置中读取ES URL
	if config.ScrapeUrl != nil && *config.ScrapeUrl != "" {
		l.esURL = *config.ScrapeUrl
		logrus.Debugf("Using scrape_uri from command line: %s", l.esURL)
	} else {
		// 检查配置文件中的ScrapeUri
		var settings config.Settings
		if err := exporter.Unpack(&settings); err == nil && settings.ScrapeUri != "" {
			l.esURL = settings.ScrapeUri
			logrus.Debugf("Using scrape_uri from config file: %s", l.esURL)
		} else {
			logrus.Debugf("Using default scrape_uri: %s", l.esURL)
		}
	}

	// 设置是否忽略SSL证书验证
	if config.Insecure != nil && *config.Insecure {
		l.insecure = *config.Insecure
	} else {
		// 检查配置文件中的Insecure设置
		var settings config.Settings
		if err := exporter.Unpack(&settings); err == nil {
			l.insecure = settings.Insecure
		}
	}

	// 确保计数器被收集
	ch <- l.jsonParseFailures

	// 获取许可证信息
	licenseResp, err := l.fetchAndDecodeLicense()
	if err != nil {
		logrus.Warnf("Failed to fetch and decode license: %s", err)
		return
	}

	// 收集许可证信息
	maxNodesStr := fmt.Sprintf("%d", licenseResp.License.MaxNodes)
	ch <- prometheus.MustNewConstMetric(
		l.licenseInfo,
		prometheus.GaugeValue,
		1,
		"elasticsearch",
		licenseResp.License.Status,
		licenseResp.License.UID,
		licenseResp.License.Type,
		licenseResp.License.IssuedTo,
		licenseResp.License.Issuer,
		maxNodesStr,
	)

	// 收集到期日期（Unix时间戳）
	expiryDateSeconds := float64(licenseResp.License.ExpiryDateInMillis / 1000)
	ch <- prometheus.MustNewConstMetric(
		l.licenseExpiryDate,
		prometheus.GaugeValue,
		expiryDateSeconds,
		"elasticsearch",
	)

	// 收集距离到期的秒数
	now := timeNow()
	expiryTime := time.Unix(licenseResp.License.ExpiryDateInMillis/1000, 0)
	expiryTimeSeconds := expiryTime.Sub(now).Seconds()
	ch <- prometheus.MustNewConstMetric(
		l.licenseExpirySeconds,
		prometheus.GaugeValue,
		expiryTimeSeconds,
		"elasticsearch",
	)
}

// Describe 实现指标描述
func (l *License) Describe(ch chan<- *prometheus.Desc) {
	ch <- l.licenseInfo
	ch <- l.licenseExpiryDate
	ch <- l.licenseExpirySeconds
	ch <- l.jsonParseFailures.Desc()
}
// Final commit for elasticsearch_exporter/internal/metrics/elasticsearch_license.go

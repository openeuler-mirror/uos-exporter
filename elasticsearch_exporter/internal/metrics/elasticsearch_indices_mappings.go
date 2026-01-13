package metrics

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"elasticsearch_exporter/config"
	"elasticsearch_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// IndicesMappingsResponse 表示每个索引的 Elasticsearch 映射
type IndicesMappingsResponse map[string]IndexMapping

// IndexMapping 定义每个索引映射的树结构
type IndexMapping struct {
	Mappings IndexMappings `json:"mappings"`
}

// IndexMappings 定义所有索引映射
type IndexMappings struct {
	Properties IndexMappingProperties `json:"properties"`
}

// IndexMappingProperties 定义当前映射的所有属性
type IndexMappingProperties map[string]*IndexMappingProperty

// IndexMappingFields 定义当前映射的所有字段
type IndexMappingFields map[string]*IndexMappingField

// IndexMappingProperty 定义当前索引属性的单个属性
type IndexMappingProperty struct {
	Type       *string                `json:"type"`
	Properties IndexMappingProperties `json:"properties"`
	Fields     IndexMappingFields     `json:"fields"`
}

// IndexMappingField 定义当前索引字段的单个属性
type IndexMappingField struct {
	Type       *string                `json:"type"`
	Properties IndexMappingProperties `json:"properties"`
	Fields     IndexMappingFields     `json:"fields"`
}

// IndicesMappings 索引映射指标收集器
type IndicesMappings struct {
	esURL             string
	client            *http.Client
	insecure          bool
	jsonParseFailures prometheus.Counter

	// 索引映射字段数量指标
	fields *baseMetrics
}

func init() {
	exporter.Register(NewIndicesMappings())
}

// NewIndicesMappings 创建索引映射指标收集器
func NewIndicesMappings() *IndicesMappings {
	return &IndicesMappings{
		esURL: "http://localhost:9200",
		client: &http.Client{
			Timeout: defaultTimeout,
		},

		jsonParseFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "indices_mappings_json_parse_failures",
			Help:      "Number of errors while parsing JSON.",
		}),

		// 索引映射字段数量指标
		fields: NewMetrics(
			prometheus.BuildFQName(namespace, "indices_mappings_stats", "fields"),
			"Current number fields within cluster.",
			[]string{"index"},
		),
	}
}

// 递归计算字段数量
func countFieldsRecursive(properties IndexMappingProperties, fieldCounter float64) float64 {
	// 遍历所有属性
	for _, property := range properties {
		if property.Type != nil && *property.Type != "object" {
			// 属性设置了类型 - 计为一个字段，除非该值是object
			// 因为下面的递归将处理该计数
			fieldCounter++

			// 遍历该属性的所有字段
			for _, field := range property.Fields {
				// 字段设置了类型 - 计为一个字段
				if field.Type != nil {
					fieldCounter++
				}
			}
		}

		// 如果属性有更多属性，则递归计数
		if property.Properties != nil {
			fieldCounter = 1 + countFieldsRecursive(property.Properties, fieldCounter)
		}
	}

	return fieldCounter
}

// fetchAndDecodeIndicesMappings 获取并解析索引映射信息
func (im *IndicesMappings) fetchAndDecodeIndicesMappings() (*IndicesMappingsResponse, error) {
	// 确保客户端每次获取时更新配置
	im.client = &http.Client{
		Timeout: defaultTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				// #nosec G402
				InsecureSkipVerify: im.insecure,
			},
		},
	}

	u, err := url.Parse(im.esURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ES URL: %s", err)
	}

	// 构建URL路径
	u.Path = path.Join(u.Path, "/_all/_mappings")

	logrus.Debugf("Fetching indices mappings from %s", u.String())

	res, err := im.client.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get indices mappings from %s: %s", u.String(), err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP Request failed with code %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		logrus.Warnf("failed to read response body: %s", err)
		return nil, err
	}

	err = res.Body.Close()
	if err != nil {
		logrus.Warnf("failed to close response body: %s", err)
		return nil, err
	}

	var imr IndicesMappingsResponse
	if err := json.Unmarshal(body, &imr); err != nil {
		im.jsonParseFailures.Inc()
		return nil, err
	}

	return &imr, nil
}

// Collect 实现指标收集
func (im *IndicesMappings) Collect(ch chan<- prometheus.Metric) {
	// 从配置中读取ES URL
	if config.ScrapeUrl != nil && *config.ScrapeUrl != "" {
		im.esURL = *config.ScrapeUrl
		logrus.Debugf("Using scrape_uri from command line: %s", im.esURL)
	} else {
		// 检查配置文件中的ScrapeUri
		var settings config.Settings
		if err := exporter.Unpack(&settings); err == nil && settings.ScrapeUri != "" {
			im.esURL = settings.ScrapeUri
			logrus.Debugf("Using scrape_uri from config file: %s", im.esURL)
		} else {
			logrus.Debugf("Using default scrape_uri: %s", im.esURL)
		}
	}

	// 设置是否忽略SSL证书验证
	if config.Insecure != nil && *config.Insecure {
		im.insecure = *config.Insecure
	} else {
		// 检查配置文件中的Insecure设置
		var settings config.Settings
		if err := exporter.Unpack(&settings); err == nil {
			im.insecure = settings.Insecure
		}
	}

	// 确保计数器被收集
	ch <- im.jsonParseFailures

	// 获取索引映射信息
	indicesMappingsResponse, err := im.fetchAndDecodeIndicesMappings()
	if err != nil {
		logrus.Warnf("Failed to fetch and decode indices mappings: %s", err)
		return
	}

	// 收集索引映射字段数量指标
	for indexName, mappings := range *indicesMappingsResponse {
		fieldCount := countFieldsRecursive(mappings.Mappings.Properties, 0)
		im.fields.collect(ch, fieldCount, prometheus.Labels{"index": indexName})
	}
}

// Describe 实现指标描述
func (im *IndicesMappings) Describe(ch chan<- *prometheus.Desc) {
	ch <- im.fields.desc
	ch <- im.jsonParseFailures.Desc()
} 

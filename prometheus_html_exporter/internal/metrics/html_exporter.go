package metrics

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/antchfx/htmlquery"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// MetricConfig 指标配置
type MetricConfig struct {
	Name   string
	Help   string
	Type   string
	Labels map[string]string
}

// 初始化时注册HTML指标收集器
func init() {
	// HTML指标收集器会在处理HTTP请求时动态配置和注册
}

// HTMLExporter 定义HTML指标收集器
type HTMLExporter struct {
	*baseMetrics
	address               string
	selector              string
	decimalPointSeparator string
	thousandsSeparator    string
	metricConfig          MetricConfig
	metricPrefix          string
}

// NewHTMLExporter 创建新的HTML指标收集器
func NewHTMLExporter(metricPrefix string, metricConfig MetricConfig, address, selector, decimalPointSeparator, thousandsSeparator string) *HTMLExporter {
	labels := make([]string, 0, len(metricConfig.Labels))
	for k := range metricConfig.Labels {
		labels = append(labels, k)
	}
	
	exporter := &HTMLExporter{
		baseMetrics: NewMetrics(
			metricPrefix+metricConfig.Name,
			metricConfig.Help,
			labels,
		),
		address:               address,
		selector:              selector,
		decimalPointSeparator: decimalPointSeparator,
		thousandsSeparator:    thousandsSeparator,
		metricConfig:          metricConfig,
		metricPrefix:          metricPrefix,
	}
	
	return exporter
}

// Describe 实现prometheus.Collector接口
func (he *HTMLExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- he.desc
}

// Collect 实现收集指标的方法
func (he *HTMLExporter) Collect(ch chan<- prometheus.Metric) {
	// 如果必要参数未设置，则跳过抓取
	if he.address == "" || he.selector == "" {
		logrus.Warn("HTML exporter地址或选择器未设置，跳过收集")
		return
	}

	logrus.Infof("开始从 %s 抓取HTML数据，使用选择器 %s", he.address, he.selector)
	value, err := he.scrape()
	if err != nil {
		logrus.Errorf("抓取HTML出错: %v", err)
		return
	}
	logrus.Infof("成功抓取数值: %f", value)

	// 创建指标
	valueType := he.getPrometheusValueType(he.metricConfig.Type)
	
	// 获取标签值
	labelValues := make([]string, 0, len(he.metricConfig.Labels))
	for _, v := range he.metricConfig.Labels {
		labelValues = append(labelValues, v)
	}
	
	logrus.Infof("正在创建指标 %s，类型: %s，值: %f", 
		he.metricPrefix+he.metricConfig.Name, 
		he.metricConfig.Type, 
		value)
	
	// 使用MustNewConstMetric创建指标
	metric := prometheus.MustNewConstMetric(
		he.desc,
		valueType,
		value,
		labelValues...,
	)
	
	ch <- metric
	logrus.Infof("指标收集完成")
}

// 获取Prometheus值类型
func (he *HTMLExporter) getPrometheusValueType(metricType string) prometheus.ValueType {
	switch metricType {
	case "gauge":
		return prometheus.GaugeValue
	case "counter":
		return prometheus.CounterValue
	default:
		return prometheus.UntypedValue
	}
}

// scrape 抓取HTML页面中的指标数据
func (he *HTMLExporter) scrape() (float64, error) {
	logrus.Debugf("requesting URL '%s'", he.address)
	body, err := he.doRequest(he.address)
	if err != nil {
		return 0, err
	}
	defer body.Close()

	logrus.Debugf("scraping value from requested URL with XPath selector '%s'", he.selector)
	scrapedValue, err := he.parseSelector(body, he.selector)
	if err != nil {
		return 0, err
	}

	numberValue, err := he.normalizeNumericValue(scrapedValue, he.thousandsSeparator, he.decimalPointSeparator)
	if err != nil {
		return 0, err
	}

	logrus.Debugf("scraped value '%0.2f' from URL '%s'", numberValue, he.address)
	return numberValue, nil
}

// doRequest 执行HTTP请求
func (he *HTMLExporter) doRequest(url string) (io.ReadCloser, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to create request. error: %s", err)
	}

	req.Header.Add("User-Agent", fmt.Sprintf("prometheus-html-exporter/%s", Version))

	logrus.Infof("scraping page %s", url)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to request URL %s. error: %s", url, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 400 {
		return nil, fmt.Errorf("request error: %s", resp.Status)
	}

	return resp.Body, nil
}

// parseSelector 使用XPath选择器解析HTML
func (he *HTMLExporter) parseSelector(body io.ReadCloser, selector string) (string, error) {
	doc, err := htmlquery.Parse(body)
	if err != nil {
		return "", fmt.Errorf("error loading the response body into XPath nodes. error: %s", err)
	}

	nodes, err := htmlquery.QueryAll(doc, selector)
	if err != nil {
		return "", fmt.Errorf("error querying the XPath expression `%s`. error: %s", selector, err)
	}

	if len(nodes) < 1 {
		return "", fmt.Errorf("no elements returned by the XPath expression `%s`", selector)
	}

	if len(nodes) > 1 {
		logrus.Warn("more than one element was returned by the XPath expression. only the value of the first element will be exported")
	}

	value := htmlquery.InnerText(nodes[0])
	return value, nil
}

// normalizeNumericValue 将字符串转换为浮点数
func (he *HTMLExporter) normalizeNumericValue(value string, thousandsSeparator string, decimalSeparator string) (float64, error) {
	// 替换分隔符以将字符串转换为strconv可接受的格式
	value = strings.ReplaceAll(strings.ReplaceAll(value, thousandsSeparator, ""), decimalSeparator, ".")
	value = strings.TrimSpace(value)

	floatValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing value %s to a float. error: %s", value, err)
	}

	return floatValue, nil
} 
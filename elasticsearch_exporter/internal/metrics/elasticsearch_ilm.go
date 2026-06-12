package metrics

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"elasticsearch_exporter/config"
	"elasticsearch_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// IlmResponse 是ILM索引状态的响应
type IlmResponse struct {
	Indices map[string]IlmIndexResponse `json:"indices"`
}

// IlmIndexResponse 定义ILM索引状态响应
type IlmIndexResponse struct {
	Index          string  `json:"index"`
	Managed        bool    `json:"managed"`
	Phase          string  `json:"phase"`
	Action         string  `json:"action"`
	Step           string  `json:"step"`
	StepTimeMillis float64 `json:"step_time_millis"`
}

// IlmStatusResponse 定义ILM状态响应
type IlmStatusResponse struct {
	OperationMode string `json:"operation_mode"`
}

// ILM 索引生命周期管理指标收集器
type ILM struct {
	esURL             string
	client            *http.Client
	insecure          bool
	jsonParseFailures prometheus.Counter
	ilmStatusOptions  []string

	// ILM指标
	ilmIndexStatus *baseMetrics
	ilmStatus      *baseMetrics
}

func init() {
	exporter.Register(NewILM())
}

// NewILM 创建索引生命周期管理指标收集器
func NewILM() *ILM {
	return &ILM{
		esURL: "http://localhost:9200",
		client: &http.Client{
			Timeout: defaultTimeout,
		},
		ilmStatusOptions: []string{"STOPPED", "RUNNING", "STOPPING"},

		jsonParseFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "ilm_json_parse_failures",
			Help:      "Number of errors while parsing JSON.",
		}),

		// ILM指标
		ilmIndexStatus: NewMetrics(
			prometheus.BuildFQName(namespace, "ilm_index", "status"),
			"Status of ILM policy for index",
			[]string{"index", "phase", "action", "step"},
		),
		
		ilmStatus: NewMetrics(
			prometheus.BuildFQName(namespace, "ilm", "status"),
			"Current status of ILM. Status can be STOPPED, RUNNING, STOPPING.",
			[]string{"operation_mode"},
		),
	}
}

// Describe 实现 prometheus.Collector 接口
func (i *ILM) Describe(ch chan<- *prometheus.Desc) {
	ch <- i.ilmIndexStatus.desc
	ch <- i.ilmStatus.desc
	ch <- i.jsonParseFailures.Desc()
}

// fetchAndDecodeIlmIndexStatus 获取并解析ILM索引状态
func (i *ILM) fetchAndDecodeIlmIndexStatus() (IlmResponse, error) {
	// 确保客户端每次获取时更新配置
	i.client = &http.Client{
		Timeout: defaultTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				// #nosec G402
				InsecureSkipVerify: i.insecure,
			},
		},
	}

	u, err := url.Parse(i.esURL)
	if err != nil {
		return IlmResponse{}, fmt.Errorf("failed to parse ES URL: %s", err)
	}

	// 构建URL路径
	u.Path = path.Join(u.Path, "/_all/_ilm/explain")

	logrus.Debugf("Fetching ILM index status from %s", u.String())

	res, err := i.client.Get(u.String())
	if err != nil {
		return IlmResponse{}, fmt.Errorf("failed to get ILM index status from %s: %s", u.String(), err)
	}

	defer func() {
		err = res.Body.Close()
		if err != nil {
			logrus.Warnf("failed to close http.Client: %s", err)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return IlmResponse{}, fmt.Errorf("HTTP Request failed with code %d", res.StatusCode)
	}

	var ir IlmResponse
	if err := json.NewDecoder(res.Body).Decode(&ir); err != nil {
		i.jsonParseFailures.Inc()
		return IlmResponse{}, err
	}

	return ir, nil
}

// fetchAndDecodeIlmStatus 获取并解析ILM状态
func (i *ILM) fetchAndDecodeIlmStatus() (IlmStatusResponse, error) {
	u, err := url.Parse(i.esURL)
	if err != nil {
		return IlmStatusResponse{}, fmt.Errorf("failed to parse ES URL: %s", err)
	}

	// 构建URL路径
	u.Path = path.Join(u.Path, "/_ilm/status")

	logrus.Debugf("Fetching ILM status from %s", u.String())

	res, err := i.client.Get(u.String())
	if err != nil {
		return IlmStatusResponse{}, fmt.Errorf("failed to get ILM status from %s: %s", u.String(), err)
	}

	defer func() {
		err = res.Body.Close()
		if err != nil {
			logrus.Warnf("failed to close http.Client: %s", err)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return IlmStatusResponse{}, fmt.Errorf("HTTP Request failed with code %d", res.StatusCode)
	}

	var isr IlmStatusResponse
	if err := json.NewDecoder(res.Body).Decode(&isr); err != nil {
		i.jsonParseFailures.Inc()
		return IlmStatusResponse{}, err
	}

	return isr, nil
}

// bool2Float 将布尔值转换为浮点数
func bool2Float(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

// Collect 实现指标收集
func (i *ILM) Collect(ch chan<- prometheus.Metric) {
	// 从配置中读取ES URL
	if config.ScrapeUrl != nil && *config.ScrapeUrl != "" {
		i.esURL = *config.ScrapeUrl
		logrus.Debugf("Using scrape_uri from command line: %s", i.esURL)
	} else {
		// 检查配置文件中的ScrapeUri
		var settings config.Settings
		if err := exporter.Unpack(&settings); err == nil && settings.ScrapeUri != "" {
			i.esURL = settings.ScrapeUri
			logrus.Debugf("Using scrape_uri from config file: %s", i.esURL)
		} else {
			logrus.Debugf("Using default scrape_uri: %s", i.esURL)
		}
	}

	// 设置是否忽略SSL证书验证
	if config.Insecure != nil && *config.Insecure {
		i.insecure = *config.Insecure
	} else {
		// 检查配置文件中的Insecure设置
		var settings config.Settings
		if err := exporter.Unpack(&settings); err == nil {
			i.insecure = settings.Insecure
		}
	}

	// 确保计数器被收集
	ch <- i.jsonParseFailures

	// 获取ILM索引状态
	ir, err := i.fetchAndDecodeIlmIndexStatus()
	if err != nil {
		logrus.Warnf("Failed to fetch and decode ILM index status: %s", err)
		return
	}

	// 获取ILM状态
	isr, err := i.fetchAndDecodeIlmStatus()
	if err != nil {
		logrus.Warnf("Failed to fetch and decode ILM status: %s", err)
		return
	}

	// 收集ILM索引状态指标
	for name, ilm := range ir.Indices {
		i.ilmIndexStatus.collect(ch, bool2Float(ilm.Managed), prometheus.Labels{
			"index":  name,
			"phase":  ilm.Phase,
			"action": ilm.Action,
			"step":   ilm.Step,
		})
	}

	// 收集ILM状态指标
	for _, status := range i.ilmStatusOptions {
		statusActive := false
		if isr.OperationMode == status {
			statusActive = true
		}

		i.ilmStatus.collect(ch, bool2Float(statusActive), prometheus.Labels{
			"operation_mode": status,
		})
	}
} 

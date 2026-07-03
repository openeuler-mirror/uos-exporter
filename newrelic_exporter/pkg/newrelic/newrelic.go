package newrelic

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/antonholmquist/jason"
	"github.com/sirupsen/logrus"
	"github.com/tomnomnom/linkheader"
	
	"newrelic_exporter/internal/exporter"
)

// Module version
const Version = "1.0.0"

// User-Agent string
const UserAgent = "Prometheus-NewRelic-Exporter/" + Version

// Chunk size of metric requests
const ChunkSize = 10

type API struct {
	server          url.URL
	apiKey          string
	service         string
	Period          int
	unreportingApps bool
	client          *http.Client
	config          *exporter.NewRelicConfig
}

type Application struct {
	ID         int
	Name       string
	Health     string             `json:"health_status"`
	AppSummary map[string]float64 `json:"application_summary"`
	UsrSummary map[string]float64 `json:"end_user_summary"`
}

type MetricName struct {
	Name       string   `json:"name"`
	ValueNames []string `json:"values"`
}

type MetricData struct {
	Name       string
	Timeslices []struct {
		Values map[string]interface{}
	}
}

func NewAPI(config *exporter.NewRelicConfig) (*API, error) {
	if config == nil {
		return nil, fmt.Errorf("NewRelic configuration is nil")
	}

	serverURL, err := url.Parse(config.ApiServer)
	if err != nil {
		return nil, fmt.Errorf("could not parse API URL: %v", err)
	}
	
	if config.ApiKey == "" {
		logrus.Warn("API密钥为空，无法初始化API，应使用模拟数据模式")
		return nil, fmt.Errorf("API密钥为空，无法初始化API客户端")
	}
	
	if config.Service == "" {
		return nil, fmt.Errorf("cannot continue without NewRelic service selected")
	}

	client := &http.Client{Timeout: config.Timeout}

	// 详细记录API初始化参数
	logrus.Infof("初始化NewRelic API客户端: 服务器=%s, 服务=%s, API密钥长度=%d", 
		config.ApiServer, config.Service, len(config.ApiKey))

	return &API{
		server:  *serverURL,
		apiKey:  config.ApiKey,
		service: config.Service,
		client:  client,
		Period:  config.Period,
		config:  config,
	}, nil
}

func (api *API) GetApplications() ([]Application, error) {
	logrus.Infof("Requesting application list from %s.", api.server.String())

	body, err := api.req(fmt.Sprintf("/v2/%s.json", api.service), "")
	if err != nil {
		logrus.Error("Error getting application list: ", err)
		return nil, err
	}

	// 添加日志，记录API响应
	logrus.Debugf("API Response body: %s", string(body))

	v, err := jason.NewObjectFromBytes(body)
	if err != nil {
		logrus.Errorf("Failed to parse JSON response: %v", err)
		return nil, fmt.Errorf("failed to parse JSON response: %v", err)
	}

	// 检查是否包含错误消息
	errObj, err := v.GetObject("error")
	if err == nil {
		// 找到了错误对象
		errMsg, err := errObj.GetString("title")
		if err == nil {
			return nil, fmt.Errorf("API error: %s", errMsg)
		}
	}

	// 尝试获取应用程序列表
	appsArray, err := v.GetObjectArray("applications")
	if err != nil {
		logrus.Errorf("Failed to get applications array: %v. Response might not contain 'applications' field", err)
		return nil, fmt.Errorf("key not found: %v", err)
	}

	applications := make([]Application, len(appsArray))
	for i, a := range appsArray {
		application := new(Application)
		aBytes, _ := a.Marshal()
		err = json.Unmarshal(aBytes, application)
		if err != nil {
			logrus.Warnf("Failed to unmarshal application at index %d: %v", i, err)
			continue
		}
		applications[i] = *application
	}

	logrus.Infof("Found %v applications", len(applications))
	return applications, nil
}

func (api *API) GetMetricNames(appID int) ([]MetricName, error) {
	logrus.Infof("Requesting metrics names for application id %d with %v filters", appID, len(api.config.MetricFilters))
	path := fmt.Sprintf("/v2/%s/%s/metrics.json", api.service, strconv.Itoa(appID))

	channel := make(chan MetricName)
	metricNames := make([]MetricName, 0)

	// We will only make filtered requests for metric names
	go func(ch chan MetricName) {
		var filter string
		var wg sync.WaitGroup

		for _, filter = range api.config.MetricFilters {
			logrus.Debugf("Scraping filter %v for app %v", filter, appID)

			wg.Add(1)

			go func(filter string) error {
				defer wg.Done()

				params := url.Values{}
				params.Add("name", filter)

				body, err := api.req(path, params.Encode())
				if err != nil {
					logrus.Error("Error getting metric names:", err)
					return err
				}

				v, err := jason.NewObjectFromBytes(body)
				if err != nil {
					logrus.Error("Error parsing metric names from JSON:", err)
					return err
				}

				metricsArray, err := v.GetObjectArray("metrics")
				if err != nil {
					logrus.Error("Error parsing metric names from JSON:", err)
					return err
				}

				for _, mn := range metricsArray {
					metric := new(MetricName)

					mnBytes, err := mn.Marshal()
					if err != nil {
						logrus.Error("Error marshalling metric to JSON object:", err)
						return err
					}

					err = json.Unmarshal(mnBytes, metric)
					if err != nil {
						logrus.Error("Error unmarshalling metric from JSON object:", err)
						return err
					}

					ch <- *metric
				}

				logrus.Debugf("Found %v possible metric names for app %v and filter %v", len(metricsArray), appID, filter)

				return nil
			}(filter)
		}

		wg.Wait() // wait for all goroutines to finish
		close(ch)
	}(channel)

	// receiving
	for mn := range channel {
		metricNames = append(metricNames, mn)
	}

	return metricNames, nil
}

func (api *API) GetMetricData(appId int, names []MetricName, from time.Time, to time.Time) ([]MetricData, error) {
	path := fmt.Sprintf("/v2/%s/%s/metrics/data.json", api.service, strconv.Itoa(appId))

	var valueNamesList []string

	// If Values Filter is set in config we will use it. Otherwise - gather all possible value names from metric names
	if len(api.config.ValueFilters) == 0 {
		valueNamesSet := make(map[string]struct{})

		for _, name := range names {
			for _, v := range name.ValueNames {
				valueNamesSet[v] = struct{}{}
			}
		}

		for k := range valueNamesSet {
			valueNamesList = append(valueNamesList, k)
		}
	} else {
		valueNamesList = append(valueNamesList, api.config.ValueFilters...)
	}

	// Process this in chunks to ensure the response fits within a single request.
	channel := make(chan MetricData)
	metricDatas := make([]MetricData, 0)

	go func(ch chan MetricData) {
		var wg sync.WaitGroup

		for i := 0; i < len(names); i += ChunkSize {
			var thisList []MetricName

			if i+ChunkSize > len(names) {
				thisList = names[i:]
			} else {
				thisList = names[i : i+ChunkSize]
			}

			wg.Add(1)

			go func(names []MetricName) error {
				defer wg.Done()

				params := url.Values{}

				for _, thisName := range names {
					params.Add("names[]", thisName.Name)
				}

				for _, valueFilter := range valueNamesList {
					params.Add("values[]", valueFilter)
				}

				params.Add("raw", "true")
				params.Add("summarize", "true")
				params.Add("period", strconv.Itoa(api.Period))
				params.Add("from", from.Format(time.RFC3339))
				params.Add("to", to.Format(time.RFC3339))

				body, err := api.req(path, params.Encode())
				if err != nil {
					logrus.Error("Error requesting metrics: ", err)
					return err
				}

				v, err := jason.NewObjectFromBytes(body)
				if err != nil {
					logrus.Error("Error parsing metric data from JSON:", err)
					return err
				}

				metricsData, err := v.GetObject("metric_data")
				if err != nil {
					logrus.Error("Error parsing metric data from JSON:", err)
					return err
				}
				
				metricsArray, err := metricsData.GetObjectArray("metrics")
				if err != nil {
					logrus.Error("Error parsing metrics array from JSON:", err)
					return err
				}

				for _, md := range metricsArray {
					metric := new(MetricData)

					mdBytes, err := md.Marshal()
					if err != nil {
						logrus.Error("Error marshalling metric to JSON object:", err)
						return err
					}

					err = json.Unmarshal(mdBytes, metric)
					if err != nil {
						logrus.Error("Error unmarshalling metric from JSON object:", err)
						return err
					}
					ch <- *metric
				}

				return nil
			}(thisList)
		}

		wg.Wait() // wait for all goroutines to finish
		close(ch)
	}(channel)

	// receiving
	datasCollected := 0
	for mn := range channel {
		metricDatas = append(metricDatas, mn)
		datasCollected++
	}

	return metricDatas, nil
}

func (api *API) req(path string, params string) ([]byte, error) {
	u, err := url.Parse(api.server.String() + path)
	if err != nil {
		return nil, err
	}
	u.RawQuery = params

	logrus.Debugf("Making API request to: %s", u.String())
	
	if api.apiKey == "" {
		logrus.Warn("API密钥为空，应使用模拟数据模式而不是尝试API请求")
		return nil, fmt.Errorf("API密钥为空，不应尝试API请求")
	}

	logrus.Debugf("使用的API服务器：%s，服务：%s", api.server.String(), api.service)
	logrus.Debugf("Request headers: User-Agent=%s, X-Api-Key=%s", UserAgent, api.apiKey[:4]+"****")

	req := &http.Request{
		Method: "GET",
		URL:    u,
		Header: http.Header{
			"User-Agent": {UserAgent},
			"X-Api-Key":  {api.apiKey},
		},
	}

	var data []byte
	return api.httpget(req, data)
}

func (api *API) httpget(req *http.Request, in []byte) (out []byte, err error) {
	resp, err := api.client.Do(req)
	if err != nil {
		logrus.Errorf("HTTP request failed: %v", err)
		return nil, err
	}

	logrus.Debugf("API response status: %s", resp.Status)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Errorf("Failed to read response body: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	// 记录非成功状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logrus.Errorf("API returned non-success status code: %d, body: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("API returned status code %d: %s", resp.StatusCode, string(body))
	}

	if resp.StatusCode == 429 {
		logrus.Info("API Limit Exceeded, New Relic Returning 429 see: https://docs.newrelic.com/docs/apis/rest-api-v2/requirements/api-overload-protection-handling-429-errors")
		if len(resp.Header["Newrelic-Overloadprotection-Reset"]) > 0 {
			overload_header := resp.Header["Newrelic-Overloadprotection-Reset"][0]
			overload_time, _ := strconv.ParseInt(overload_header, 10, 64)
			logrus.Info("Overload protection resets at: ", time.Unix(overload_time, 0))
		}
		return nil, fmt.Errorf("API limit exceeded")
	}

	out = append(in, body...)

	// Read the link header to see if we need to read more pages.
	links := linkheader.Parse(resp.Header.Get("Link"))

	relLast := links.FilterByRel("last")
	if len(relLast) > 0 {
		u, err := url.Parse(relLast[0].URL)
		if err != nil {
			logrus.Errorf("Error parsing 'last' relation link: %v", err)
		} else {
			logrus.Debugf("Found %v pages for %s", u.Query().Get("page"), req.URL)
		}
	}

	relNext := links.FilterByRel("next")
	if len(relNext) > 0 {
		u, err := url.Parse(relNext[0].URL)
		if err != nil {
			return nil, err
		}
		
		cursor := u.Query().Get("cursor")
		if cursor != "" {
			query := req.URL.Query()
			query.Set("cursor", cursor)
			query.Del("page")
			req.URL.RawQuery = query.Encode()
			return api.httpget(req, out)
		}
	}

	return out, nil
}

// SetServer 设置 API 服务器 URL（用于测试）
func (api *API) SetServer(server url.URL) {
	api.server = server
}

// SetAPIKey 设置 API 密钥（用于测试）
func (api *API) SetAPIKey(apiKey string) {
	api.apiKey = apiKey
}

// SetService 设置服务名称（用于测试）
func (api *API) SetService(service string) {
	api.service = service
}

// SetClient 设置 HTTP 客户端（用于测试）
func (api *API) SetClient(client *http.Client) {
	api.client = client
}
// Part 2 commit for newrelic_exporter/pkg/newrelic/newrelic.go

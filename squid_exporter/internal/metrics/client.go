package metrics

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Counter 表示从Squid获取的计数器指标
type Counter struct {
	Key       string
	Value     float64
	VarLabels []VarLabel
}

// VarLabel 表示变量标签
type VarLabel struct {
	Key   string
	Value string
}

// SquidClient 提供连接到Squid服务器的功能
type SquidClient interface {
	GetCounters() ([]Counter, error)
	GetServiceTimes() ([]Counter, error)
	GetInfos() ([]Counter, error)
}

// CacheObjectClient 保存Squid缓存对象管理器的信息
type CacheObjectClient struct {
	ch              connectionHandler
	basicAuthString string
	headers         []string
}

type connectionHandler interface {
	connect() (net.Conn, error)
}

type connectionHandlerImpl struct {
	hostname string
	port     int
}

type CacheObjectRequest struct {
	Hostname string
	Port     int
	Login    string
	Password string
	Headers  []string
}

const (
	requestProtocol = "GET cache_object://localhost/%s HTTP/1.0"
	timeout         = 10 * time.Second
)

// 连接到指定的主机和端口
func (c *connectionHandlerImpl) connect() (net.Conn, error) {
	return net.DialTimeout("tcp", fmt.Sprintf("%s:%d", c.hostname, c.port), timeout)
}

// 创建基本认证字符串
func buildBasicAuthString(login string, password string) string {
	if len(login) == 0 {
		return ""
	} else {
		return base64.StdEncoding.EncodeToString([]byte(login + ":" + password))
	}
}

// NewCacheObjectClient 初始化一个新的缓存客户端
func NewCacheObjectClient(cor *CacheObjectRequest) *CacheObjectClient {
	return &CacheObjectClient{
		&connectionHandlerImpl{
			cor.Hostname,
			cor.Port,
		},
		buildBasicAuthString(cor.Login, cor.Password),
		cor.Headers,
	}
}

// 从Squid读取数据
func (c *CacheObjectClient) readFromSquid(endpoint string) (*bufio.Reader, error) {
	conn, err := c.ch.connect()
	if err != nil {
		return nil, err
	}

	// 不要在这里关闭连接，而是在HTTP读取响应后，由调用者关闭

	// 构建完整的HTTP请求
	rBody := append(c.headers, []string{
		fmt.Sprintf(requestProtocol, endpoint),
		"Host: localhost",
		"User-Agent: squidclient/3.5.12",
	}...)

	// 添加认证头
	if c.basicAuthString != "" {
		rBody = append(rBody, "Proxy-Authorization: Basic "+c.basicAuthString)
		rBody = append(rBody, "Authorization: Basic "+c.basicAuthString)
	}

	// 添加结束标记
	rBody = append(rBody, "Accept: */*", "\r\n")
	request := strings.Join(rBody, "\r\n")

	// 发送完整请求
	fmt.Fprint(conn, request)

	// 读取HTTP响应
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	if resp.StatusCode != 200 {
		_ = conn.Close()
		return nil, fmt.Errorf("Non success code %d while fetching metrics", resp.StatusCode)
	}

	// 返回响应体的读取器
	return bufio.NewReader(resp.Body), nil
}

// 读取响应行
func readLines(reader *bufio.Reader, lines chan<- string) {
	for {
		line, err := reader.ReadString('\n')

		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("error reading from the bufio.Reader: %v", err)
			break
		}

		lines <- line
	}
	close(lines)
}

// GetCounters 从squid缓存管理器获取计数器
func (c *CacheObjectClient) GetCounters() ([]Counter, error) {
	var counters []Counter

	reader, err := c.readFromSquid("counters")
	if err != nil {
		return nil, fmt.Errorf("error getting counters: %v", err)
	}

	lines := make(chan string)
	go readLines(reader, lines)

	for line := range lines {
		counter, err := decodeCounterStrings(line)
		if err != nil {
			log.Println(err)
		} else {
			counters = append(counters, counter)
		}
	}

	return counters, nil
}

// GetServiceTimes 从squid缓存管理器获取服务时间
func (c *CacheObjectClient) GetServiceTimes() ([]Counter, error) {
	var serviceTimes []Counter

	reader, err := c.readFromSquid("service_times")
	if err != nil {
		return nil, fmt.Errorf("error getting service times: %v", err)
	}

	lines := make(chan string)
	go readLines(reader, lines)

	for line := range lines {
		serviceTime, err := decodeServiceTimeStrings(line)
		if err != nil {
			log.Println(err)
		} else if serviceTime.Key != "" {
			serviceTimes = append(serviceTimes, serviceTime)
		}
	}

	return serviceTimes, nil
}

// GetInfos 从squid缓存管理器获取信息
func (c *CacheObjectClient) GetInfos() ([]Counter, error) {
	var infos []Counter

	reader, err := c.readFromSquid("info")
	if err != nil {
		return nil, fmt.Errorf("error getting info: %v", err)
	}

	lines := make(chan string)
	go readLines(reader, lines)

	var infoVarLabels Counter
	infoVarLabels.Key = "squid_info"
	infoVarLabels.Value = 1

	for line := range lines {
		info, err := decodeInfoStrings(line)
		if err != nil {
			log.Println(err)
		} else if len(info.VarLabels) > 0 {
			if info.VarLabels[0].Key == "5min" {
				var infoAvg5 Counter
				var infoAvg60 Counter

				infoAvg5.Key = info.Key + "_" + info.VarLabels[0].Key
				infoAvg60.Key = info.Key + "_" + info.VarLabels[1].Key

				if value, err := strconv.ParseFloat(info.VarLabels[0].Value, 64); err == nil {
					infoAvg5.Value = value
					infos = append(infos, infoAvg5)
				}
				if value, err := strconv.ParseFloat(info.VarLabels[1].Value, 64); err == nil {
					infoAvg60.Value = value
					infos = append(infos, infoAvg60)
				}
			} else {
				infoVarLabels.VarLabels = append(infoVarLabels.VarLabels, info.VarLabels[0])
			}
		} else if info.Key != "" {
			infos = append(infos, info)
		}
	}

	if len(infoVarLabels.VarLabels) > 0 {
		infos = append(infos, infoVarLabels)
	}

	return infos, nil
}

// 解析counters响应
func decodeCounterStrings(line string) (Counter, error) {
	if equal := strings.Index(line, "="); equal >= 0 {
		if key := strings.TrimSpace(line[:equal]); len(key) > 0 {
			value := ""
			if len(line) > equal {
				value = strings.TrimSpace(line[equal+1:])
			}

			// Remove additional formating string from `sample_time`
			if slices := strings.Split(value, " "); len(slices) > 0 {
				value = slices[0]
			}

			if i, err := strconv.ParseFloat(value, 64); err == nil {
				return Counter{Key: key, Value: i}, nil
			}
		}
	}

	return Counter{}, errors.New("counter - could not parse line: " + line)
}

// 解析service_times响应
func decodeServiceTimeStrings(line string) (Counter, error) {
	if strings.HasSuffix(line, ":\n") { // A header line isn't a metric
		return Counter{}, nil
	}
	if equal := strings.Index(line, ":"); equal >= 0 {
		if key := strings.TrimSpace(line[:equal]); len(key) > 0 {
			value := ""
			if len(line) > equal {
				value = strings.TrimSpace(line[equal+1:])
			}
			key = strings.Replace(key, " ", "_", -1)
			key = strings.Replace(key, "(", "", -1)
			key = strings.Replace(key, ")", "", -1)

			if equalTwo := strings.Index(value, "%"); equalTwo >= 0 {
				if keyTwo := strings.TrimSpace(value[:equalTwo]); len(keyTwo) > 0 {
					if len(value) > equalTwo {
						value = strings.Split(strings.TrimSpace(value[equalTwo+1:]), " ")[0]
					}
					key = key + "_" + keyTwo
				}
			}

			if value, err := strconv.ParseFloat(value, 64); err == nil {
				return Counter{Key: key, Value: value}, nil
			}
		}
	}

	return Counter{}, errors.New("service times - could not parse line: " + line)
}

// 解析info响应
func decodeInfoStrings(line string) (Counter, error) {
	if strings.HasSuffix(line, ":\n") { // A header line isn't a metric
		return Counter{}, nil
	}

	if idx := strings.Index(line, ":"); idx >= 0 { // detect if line contain metric format like "metricName: value"
		if key := strings.TrimSpace(line[:idx]); len(key) > 0 {
			value := ""
			if len(line) > idx {
				value = strings.TrimSpace(line[idx+1:])
			}
			key = strings.Replace(key, " ", "_", -1)
			key = strings.Replace(key, "(", "", -1)
			key = strings.Replace(key, ")", "", -1)
			key = strings.Replace(key, ",", "", -1)
			key = strings.Replace(key, "/", "", -1)

			// metrics with value as string need to save as label, format like "Squid Object Cache: Version 6.1" (the 3 first metrics)
			if key == "Squid_Object_Cache" || key == "Build_Info" || key == "Service_Name" {
				if key == "Squid_Object_Cache" { // To clarify that the value is the squid version.
					key = key + "_Version"
					if slices := strings.Split(value, " "); len(slices) > 0 {
						value = slices[1]
					}
				}
				var infoVarLabel VarLabel
				infoVarLabel.Key = key
				infoVarLabel.Value = value

				var infoCounter Counter
				infoCounter.Key = key
				infoCounter.VarLabels = append(infoCounter.VarLabels, infoVarLabel)
				return infoCounter, nil
			} else if key == "Start_Time" || key == "Current_Time" { // discart this metrics
				return Counter{}, nil
			}

			// Remove additional information in value metric
			if slices := strings.Split(value, " "); len(slices) > 0 {
				if slices[0] == "5min:" && len(slices) > 2 && slices[2] == "60min:" { // catch metrics with avg in 5min and 60min format like "Hits as % of bytes sent: 5min: -0.0%, 60min: -0.0%"
					var infoAvg5mVarLabel VarLabel
					infoAvg5mVarLabel.Key = slices[0]
					infoAvg5mVarLabel.Value = slices[1]

					infoAvg5mVarLabel.Key = strings.Replace(infoAvg5mVarLabel.Key, ":", "", -1)
					infoAvg5mVarLabel.Value = strings.Replace(infoAvg5mVarLabel.Value, "%", "", -1)
					infoAvg5mVarLabel.Value = strings.Replace(infoAvg5mVarLabel.Value, ",", "", -1)

					var infoAvg60mVarLabel VarLabel
					infoAvg60mVarLabel.Key = slices[2]
					infoAvg60mVarLabel.Value = slices[3]

					infoAvg60mVarLabel.Key = strings.Replace(infoAvg60mVarLabel.Key, ":", "", -1)
					infoAvg60mVarLabel.Value = strings.Replace(infoAvg60mVarLabel.Value, "%", "", -1)
					infoAvg60mVarLabel.Value = strings.Replace(infoAvg60mVarLabel.Value, ",", "", -1)

					var infoAvgCounter Counter
					infoAvgCounter.Key = key
					infoAvgCounter.VarLabels = append(infoAvgCounter.VarLabels, infoAvg5mVarLabel, infoAvg60mVarLabel)

					return infoAvgCounter, nil
				} else {
					value = slices[0]
				}
			}

			value = strings.Replace(value, "%", "", -1)
			value = strings.Replace(value, ",", "", -1)

			if i, err := strconv.ParseFloat(value, 64); err == nil {
				return Counter{Key: key, Value: i}, nil
			}
		}
	} else {
		// this catch the last 4 metrics format like "value metricName"
		lineTrimed := strings.TrimSpace(line[:])

		if idx := strings.Index(lineTrimed, " "); idx >= 0 {
			key := strings.TrimSpace(lineTrimed[idx+1:])
			key = strings.Replace(key, " ", "_", -1)
			key = strings.Replace(key, "-", "_", -1)

			value := strings.TrimSpace(lineTrimed[:idx])

			if i, err := strconv.ParseFloat(value, 64); err == nil {
				return Counter{Key: key, Value: i}, nil
			}
		}
	}

	return Counter{}, errors.New("info - could not parse line: " + line)
}

// 全局配置参数，由main函数或其他初始化代码设置
var (
	GlobalHostname string   = "localhost"
	GlobalPort     int      = 3128
	GlobalLogin    string   = ""
	GlobalPassword string   = ""
	GlobalHeaders  []string = []string{}
)

// 使用全局配置创建一个CacheObjectClient
func GetGlobalClient() *CacheObjectClient {
	return NewCacheObjectClient(&CacheObjectRequest{
		Hostname: GlobalHostname,
		Port:     GlobalPort,
		Login:    GlobalLogin,
		Password: GlobalPassword,
		Headers:  GlobalHeaders,
	})
}

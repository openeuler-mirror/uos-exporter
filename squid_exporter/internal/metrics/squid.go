package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"log"
)

type SquidConfig struct {
	Hostname     string
	Port         int
	Login        string
	Password     string
	Headers      []string
	ExtractTimes bool
}

// SquidCollector 是主Squid指标收集器
type SquidCollector struct {
	client       SquidClient
	hostname     string
	port         int
	extractTimes bool
	up           prometheus.Gauge
}

// NewSquidCollector 创建一个新的Squid指标收集器
func NewSquidCollector(config *SquidConfig) *SquidCollector {
	up := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "squid",
		Name:      "up",
		Help:      "Was the last query of squid successful",
	})
	
	collector := &SquidCollector{
		client: NewCacheObjectClient(&CacheObjectRequest{
			Hostname: config.Hostname,
			Port:     config.Port,
			Login:    config.Login,
			Password: config.Password,
			Headers:  config.Headers,
		}),
		hostname:     config.Hostname,
		port:         config.Port,
		extractTimes: config.ExtractTimes,
		up:           up,
	}
	
	return collector
}

// Describe 实现了Collector接口
func (sc *SquidCollector) Describe(ch chan<- *prometheus.Desc) {
	// 只描述up指标
	sc.up.Describe(ch)
}

// Collect 实现了Collector接口
func (sc *SquidCollector) Collect(ch chan<- prometheus.Metric) {
	// 尝试连接Squid服务器以检查状态
	_, err := sc.client.GetCounters()
	
	if err == nil {
		// 连接成功，设置up指标为1
		sc.up.Set(1)
	} else {
		// 连接失败，设置up指标为0
		sc.up.Set(0)
		log.Printf("Error connecting to Squid server: %v", err)
	}
	
	// 发送up指标
	ch <- sc.up
} 

package metrics

import (
	"fmt"
	"strings"
	"github.com/prometheus/client_golang/prometheus"
)

// 定义Squid服务时间指标类型
type squidServiceTimes struct {
	Section     string
	Counter     string
	Suffix      string
	Description string
}

// Squid服务时间指标列表
var squidServiceTimesList = []squidServiceTimes{
	{"HTTP_Requests", "All", "5", "Service Time Percentiles 5min"},
	{"HTTP_Requests", "All", "10", "Service Time Percentiles 5min"},
	{"HTTP_Requests", "All", "15", "Service Time Percentiles 5min"},
	{"HTTP_Requests", "All", "20", "Service Time Percentiles 5min"},
	{"HTTP_Requests", "All", "25", "Service Time Percentiles 5min"},
	{"HTTP_Requests", "All", "30", "Service Time Percentiles 5min"},
	{"HTTP_Requests", "All", "35", "Service Time Percentiles 5min"},
	{"HTTP_Requests", "All", "40", "Service Time Percentiles 5min"},
	{"HTTP_Requests", "All", "45", "Service Time Percentiles 5min"},
	{"HTTP_Requests", "All", "50", "Service Time Percentiles 5min"},
	{"HTTP_Requests", "All", "55", "Service Time Percentiles 5min"},
	{"HTTP_Requests", "All", "60", "Service Time Percentiles 5min"},
	{"HTTP_Requests", "All", "65", "Service Time Percentiles 5min"},
	{"HTTP_Requests", "All", "70", "Service Time Percentiles 5min"},
	{"HTTP_Requests", "All", "75", "Service Time Percentiles 5min"},
	{"HTTP_Requests", "All", "80", "Service Time Percentiles 5min"},
	{"HTTP_Requests", "All", "85", "Service Time Percentiles 5min"},
	{"HTTP_Requests", "All", "90", "Service Time Percentiles 5min"},
	{"HTTP_Requests", "All", "95", "Service Time Percentiles 5min"},
	{"HTTP_Requests", "All", "100", "Service Time Percentiles 5min"},
	{"Cache_Misses", "", "5", "Service Time Percentiles 5min"},
	{"Cache_Misses", "", "10", "Service Time Percentiles 5min"},
	{"Cache_Misses", "", "15", "Service Time Percentiles 5min"},
	{"Cache_Misses", "", "20", "Service Time Percentiles 5min"},
	{"Cache_Misses", "", "25", "Service Time Percentiles 5min"},
	{"Cache_Misses", "", "30", "Service Time Percentiles 5min"},
	{"Cache_Misses", "", "35", "Service Time Percentiles 5min"},
	{"Cache_Misses", "", "40", "Service Time Percentiles 5min"},
	{"Cache_Misses", "", "45", "Service Time Percentiles 5min"},
	{"Cache_Misses", "", "50", "Service Time Percentiles 5min"},
	{"Cache_Misses", "", "55", "Service Time Percentiles 5min"},
	{"Cache_Misses", "", "60", "Service Time Percentiles 5min"},
	{"Cache_Misses", "", "65", "Service Time Percentiles 5min"},
	{"Cache_Misses", "", "70", "Service Time Percentiles 5min"},
	{"Cache_Misses", "", "75", "Service Time Percentiles 5min"},
	{"Cache_Misses", "", "80", "Service Time Percentiles 5min"},
	{"Cache_Misses", "", "85", "Service Time Percentiles 5min"},
	{"Cache_Misses", "", "90", "Service Time Percentiles 5min"},
	{"Cache_Misses", "", "95", "Service Time Percentiles 5min"},
	{"Cache_Hits", "", "5", "Service Time Percentiles 5min"},
	{"Cache_Hits", "", "10", "Service Time Percentiles 5min"},
	{"Cache_Hits", "", "15", "Service Time Percentiles 5min"},
	{"Cache_Hits", "", "20", "Service Time Percentiles 5min"},
	{"Cache_Hits", "", "25", "Service Time Percentiles 5min"},
	{"Cache_Hits", "", "30", "Service Time Percentiles 5min"},
	{"Cache_Hits", "", "35", "Service Time Percentiles 5min"},
	{"Cache_Hits", "", "40", "Service Time Percentiles 5min"},
	{"Cache_Hits", "", "45", "Service Time Percentiles 5min"},
	{"Cache_Hits", "", "50", "Service Time Percentiles 5min"},
	{"Cache_Hits", "", "55", "Service Time Percentiles 5min"},
	{"Cache_Hits", "", "60", "Service Time Percentiles 5min"},
	{"Cache_Hits", "", "65", "Service Time Percentiles 5min"},
	{"Cache_Hits", "", "70", "Service Time Percentiles 5min"},
	{"Cache_Hits", "", "75", "Service Time Percentiles 5min"},
	{"Cache_Hits", "", "80", "Service Time Percentiles 5min"},
	{"Cache_Hits", "", "85", "Service Time Percentiles 5min"},
	{"Cache_Hits", "", "90", "Service Time Percentiles 5min"},
	{"Cache_Hits", "", "95", "Service Time Percentiles 5min"},
	{"Near_Hits", "", "5", "Service Time Percentiles 5min"},
	{"Near_Hits", "", "10", "Service Time Percentiles 5min"},
	{"Near_Hits", "", "15", "Service Time Percentiles 5min"},
	{"Near_Hits", "", "20", "Service Time Percentiles 5min"},
	{"Near_Hits", "", "25", "Service Time Percentiles 5min"},
	{"Near_Hits", "", "30", "Service Time Percentiles 5min"},
	{"Near_Hits", "", "35", "Service Time Percentiles 5min"},
	{"Near_Hits", "", "40", "Service Time Percentiles 5min"},
	{"Near_Hits", "", "45", "Service Time Percentiles 5min"},
	{"Near_Hits", "", "50", "Service Time Percentiles 5min"},
	{"Near_Hits", "", "55", "Service Time Percentiles 5min"},
	{"Near_Hits", "", "60", "Service Time Percentiles 5min"},
	{"Near_Hits", "", "65", "Service Time Percentiles 5min"},
	{"Near_Hits", "", "70", "Service Time Percentiles 5min"},
	{"Near_Hits", "", "75", "Service Time Percentiles 5min"},
	{"Near_Hits", "", "80", "Service Time Percentiles 5min"},
	{"Near_Hits", "", "85", "Service Time Percentiles 5min"},
	{"Near_Hits", "", "90", "Service Time Percentiles 5min"},
	{"Near_Hits", "", "95", "Service Time Percentiles 5min"},
	{"DNS_Lookups", "", "5", "Service Time Percentiles 5min"},
	{"DNS_Lookups", "", "10", "Service Time Percentiles 5min"},
	{"DNS_Lookups", "", "15", "Service Time Percentiles 5min"},
	{"DNS_Lookups", "", "20", "Service Time Percentiles 5min"},
	{"DNS_Lookups", "", "25", "Service Time Percentiles 5min"},
	{"DNS_Lookups", "", "30", "Service Time Percentiles 5min"},
	{"DNS_Lookups", "", "35", "Service Time Percentiles 5min"},
	{"DNS_Lookups", "", "40", "Service Time Percentiles 5min"},
	{"DNS_Lookups", "", "45", "Service Time Percentiles 5min"},
	{"DNS_Lookups", "", "50", "Service Time Percentiles 5min"},
	{"DNS_Lookups", "", "55", "Service Time Percentiles 5min"},
	{"DNS_Lookups", "", "60", "Service Time Percentiles 5min"},
	{"DNS_Lookups", "", "65", "Service Time Percentiles 5min"},
	{"DNS_Lookups", "", "70", "Service Time Percentiles 5min"},
	{"DNS_Lookups", "", "75", "Service Time Percentiles 5min"},
	{"DNS_Lookups", "", "80", "Service Time Percentiles 5min"},
	{"DNS_Lookups", "", "85", "Service Time Percentiles 5min"},
	{"DNS_Lookups", "", "90", "Service Time Percentiles 5min"},
	{"DNS_Lookups", "", "95", "Service Time Percentiles 5min"},
}

// GetSquidServiceTimes 返回所有Squid服务时间指标
func GetSquidServiceTimes() []prometheus.Collector {
	collectors := []prometheus.Collector{}
	for _, serviceTime := range squidServiceTimesList {
		collectors = append(collectors,
			NewSquidServiceTime(serviceTime.Section, serviceTime.Counter, serviceTime.Suffix, serviceTime.Description))
	}
	return collectors
}

// SquidServiceTime 是用于存储Squid服务时间的指标
type SquidServiceTime struct {
	*baseMetrics
	section string
	counter string
	suffix  string
}

// NewSquidServiceTime创建一个新的SquidServiceTime实例
func NewSquidServiceTime(section, counter, suffix, help string) *SquidServiceTime {
	var name string

	if counter != "" {
		name = prometheus.BuildFQName("squid", 
			strings.Replace(section, ".", "_", -1),
			fmt.Sprintf("%s_%s", counter, suffix))
	} else {
		name = prometheus.BuildFQName("squid", 
			strings.Replace(section, ".", "_", -1),
			fmt.Sprintf("%s", suffix))
	}

	return &SquidServiceTime{
		baseMetrics: NewMetrics(name, help, []string{}),
		section:     section,
		counter:     counter,
		suffix:      suffix,
	}
}

// Describe 实现了Collector接口
func (sst *SquidServiceTime) Describe(ch chan<- *prometheus.Desc) {
	ch <- sst.baseMetrics.desc
}

// Collect实现了Collector接口，用于采集指标
func (sst *SquidServiceTime) Collect(ch chan<- prometheus.Metric) {
	// 创建一个客户端连接Squid服务器
	client := NewCacheObjectClient(&CacheObjectRequest{
		Hostname: GlobalHostname,
		Port:     GlobalPort,
		Login:    GlobalLogin,
		Password: GlobalPassword,
		Headers:  GlobalHeaders,
	})

	serviceTimes, err := client.GetServiceTimes()
	if err != nil {
		// 连接失败，记录错误并返回
		return
	}

	// 构建预期的Key格式
	var key string
	if sst.counter != "" {
		key = fmt.Sprintf("%s_%s_%s", sst.section, sst.counter, sst.suffix)
	} else {
		key = fmt.Sprintf("%s_%s", sst.section, sst.suffix)
	}

	// 查找匹配的指标
	for _, serviceTime := range serviceTimes {
		if serviceTime.Key == key {
			// 找到匹配的指标，使用实际数据
			ch <- prometheus.MustNewConstMetric(sst.baseMetrics.desc, prometheus.GaugeValue, serviceTime.Value)
			return
		}
	}
} 
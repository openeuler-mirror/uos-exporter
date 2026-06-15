package metrics

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
)

// 定义Squid计数器指标类型
type squidCounter struct {
	Section     string
	Counter     string
	Suffix      string
	Description string
}

// Squid计数器指标列表
var squidCounters = []squidCounter{
	{"client_http", "requests", "total", "The total number of client requests"},
	{"client_http", "hits", "total", "The total number of client cache hits"},
	{"client_http", "errors", "total", "The total number of client http errors"},
	{"client_http", "kbytes_in", "kbytes_total", "The total number of client kbytes received"},
	{"client_http", "kbytes_out", "kbytes_total", "The total number of client kbytes transferred"},
	{"client_http", "hit_kbytes_out", "bytes_total", "The total number of client kbytes cache hit"},

	{"server.http", "requests", "total", "The total number of server http requests"},
	{"server.http", "errors", "total", "The total number of server http errors"},
	{"server.http", "kbytes_in", "kbytes_total", "The total number of server http kbytes received"},
	{"server.http", "kbytes_out", "kbytes_total", "The total number of server http kbytes transferred"},

	{"server.all", "requests", "total", "The total number of server all requests"},
	{"server.all", "errors", "total", "The total number of server all errors"},
	{"server.all", "kbytes_in", "kbytes_total", "The total number of server kbytes received"},
	{"server.all", "kbytes_out", "kbytes_total", "The total number of server kbytes transferred"},

	{"server.ftp", "requests", "total", "The total number of server ftp requests"},
	{"server.ftp", "errors", "total", "The total number of server ftp errors"},
	{"server.ftp", "kbytes_in", "kbytes_total", "The total number of server ftp kbytes received"},
	{"server.ftp", "kbytes_out", "kbytes_total", "The total number of server ftp kbytes transferred"},

	{"server.other", "requests", "total", "The total number of server other requests"},
	{"server.other", "errors", "total", "The total number of server other errors"},
	{"server.other", "kbytes_in", "kbytes_total", "The total number of server other kbytes received"},
	{"server.other", "kbytes_out", "kbytes_total", "The total number of server other kbytes transferred"},

	{"swap", "ins", "total", "The number of objects read from disk"},
	{"swap", "outs", "total", "The number of objects saved to disk"},
	{"swap", "files_cleaned", "total", "The number of orphaned cache files removed by the periodic cleanup procedure"},
}

// GetSquidCounters 返回所有Squid计数器指标
func GetSquidCounters() []prometheus.Collector {
	counters := []prometheus.Collector{}
	for _, counter := range squidCounters {
		counters = append(counters, 
			NewSquidCounter(counter.Section, counter.Counter, counter.Suffix, counter.Description))
	}
	return counters
}

// SquidCounter 是用于存储Squid计数器的指标
type SquidCounter struct {
	*baseMetrics
	section string
	counter string
}

// NewSquidCounter创建一个新的SquidCounter实例
func NewSquidCounter(section, counter, suffix, help string) *SquidCounter {
	fqname := prometheus.BuildFQName("squid", 
		replaceNonAlphanumeric(section), 
		counter+"_"+suffix)
	
	return &SquidCounter{
		baseMetrics: NewMetrics(fqname, help, []string{}),
		section:     section,
		counter:     counter,
	}
}

// Describe 实现了Collector接口
func (sc *SquidCounter) Describe(ch chan<- *prometheus.Desc) {
	ch <- sc.baseMetrics.desc
}

// Collect实现了Collector接口，用于采集指标
func (sc *SquidCounter) Collect(ch chan<- prometheus.Metric) {
	// 创建一个客户端连接Squid服务器
	client := NewCacheObjectClient(&CacheObjectRequest{
		Hostname: GlobalHostname,
		Port:     GlobalPort,
		Login:    GlobalLogin,
		Password: GlobalPassword,
		Headers:  GlobalHeaders,
	})

	counters, err := client.GetCounters()
	if err != nil {
		// 连接失败，记录错误并返回
		return
	}

	// 查找匹配的指标
	key := fmt.Sprintf("%s.%s", sc.section, sc.counter)
	for _, counter := range counters {
		if counter.Key == key {
			// 找到匹配的指标，使用实际数据，注意这里用CounterValue而不是GaugeValue
			ch <- prometheus.MustNewConstMetric(sc.baseMetrics.desc, prometheus.CounterValue, counter.Value)
			return
		}
	}
}

// 辅助函数：将非字母数字字符替换为下划线
func replaceNonAlphanumeric(s string) string {
	result := ""
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			result += string(c)
		} else {
			result += "_"
		}
	}
	return result
} 

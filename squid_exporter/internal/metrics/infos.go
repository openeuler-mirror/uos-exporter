package metrics

import (
	"strings"
	"github.com/prometheus/client_golang/prometheus"
)

// 定义Squid信息指标类型
type squidInfos struct {
	Section     string
	Description string
	Unit        string
}

// Squid信息指标列表
var squidInfosList = []squidInfos{
	{"Number_of_clients_accessing_cache", "", "number"},
	{"Number_of_HTTP_requests_received", "", "number"},
	{"Number_of_ICP_messages_received", "", "number"},
	{"Number_of_ICP_messages_sent", "", "number"},
	{"Number_of_queued_ICP_replies", "", "number"},
	{"Number_of_HTCP_messages_received", "", "number"},
	{"Number_of_HTCP_messages_sent", "", "number"},
	{"Request_failure_ratio", "", "%"},
	{"Average_HTTP_requests_per_minute_since_start", "", "%"},
	{"Average_ICP_messages_per_minute_since_start", "", "%"},
	{"Select_loop_called", "", "number"},
	{"Hits_as_%_of_all_requests_5min", "", "%"},
	{"Hits_as_%_of_bytes_sent_5min", "", "%"},
	{"Memory_hits_as_%_of_hit_requests_5min", "", "%"},
	{"Disk_hits_as_%_of_hit_requests_5min", "", "%"},
	{"Hits_as_%_of_all_requests_60min", "", "%"},
	{"Hits_as_%_of_bytes_sent_60min", "", "%"},
	{"Memory_hits_as_%_of_hit_requests_60min", "", "%"},
	{"Disk_hits_as_%_of_hit_requests_60min", "", "%"},
	{"Storage_Swap_size", "", "KB"},
	{"Storage_Swap_capacity", "", "% use"},
	{"Storage_Mem_size", "", "KB"},
	{"Storage_Mem_capacity", "", "% used"},
	{"Mean_Object_Size", "", "KB"},
	{"Requests_given_to_unlinkd", "", "number"},
	{"UP_Time", "time squid is up", "seconds"},
	{"CPU_Time", "", "seconds"},
	{"CPU_Usage", "of cpu usage", "%"},
	{"CPU_Usage_5_minute_avg", "of cpu usage", "%"},
	{"CPU_Usage_60_minute_avg", "of cpu usage", "%"},
	{"Maximum_Resident_Size", "", "KB"},
	{"Page_faults_with_physical_i_o", "", "number"},
	{"Total_accounted", "", "KB"},
	{"memPoolAlloc_calls", "", "number"},
	{"memPoolFree_calls", "", "number"},
	{"Maximum_number_of_file_descriptors", "", "number"},
	{"Largest_file_desc_currently_in_use", "", "number"},
	{"Number_of_file_desc_currently_in_use", "", "number"},
	{"Files_queued_for_open", "", "number"},
	{"Available_number_of_file_descriptors", "", "number"},
	{"Reserved_number_of_file_descriptors", "", "number"},
	{"Store_Disk_files_open", "", "number"},
	{"StoreEntries", "", "number"},
	{"StoreEntries_with_MemObjects", "", "number"},
	{"Hot_Object_Cache_Items", "", "number"},
	{"on_disk_objects", "", "number"},
}

// GetSquidInfos 返回所有Squid信息指标
func GetSquidInfos() []prometheus.Collector {
	collectors := []prometheus.Collector{}
	for _, info := range squidInfosList {
		collectors = append(collectors,
			NewSquidInfo(info.Section, info.Description, info.Unit))
	}
	return collectors
}

// SquidInfo 是用于存储Squid信息的指标
type SquidInfo struct {
	*baseMetrics
	section string
}

// NewSquidInfo创建一个新的SquidInfo实例
func NewSquidInfo(section, description, unit string) *SquidInfo {
	var name string
	var help string

	name = prometheus.BuildFQName("squid", "info", strings.Replace(section, "%", "pct", -1))

	if description == "" {
		help = strings.Replace(section, "_", " ", -1)
	} else {
		help = description
	}

	help = help + " in " + unit

	return &SquidInfo{
		baseMetrics: NewMetrics(name, help, []string{}),
		section:     section,
	}
}

// Describe 实现了Collector接口
func (si *SquidInfo) Describe(ch chan<- *prometheus.Desc) {
	ch <- si.baseMetrics.desc
}

// Collect实现了Collector接口，用于采集指标
func (si *SquidInfo) Collect(ch chan<- prometheus.Metric) {
	// 创建一个客户端连接Squid服务器
	client := NewCacheObjectClient(&CacheObjectRequest{
		Hostname: GlobalHostname,
		Port:     GlobalPort,
		Login:    GlobalLogin,
		Password: GlobalPassword,
		Headers:  GlobalHeaders,
	})

	infos, err := client.GetInfos()
	if err != nil {
		// 连接失败，记录错误并返回
		return
	}

	// 查找匹配的指标
	for _, info := range infos {
		if info.Key == si.section {
			// 找到匹配的指标，使用实际数据
			ch <- prometheus.MustNewConstMetric(si.baseMetrics.desc, prometheus.GaugeValue, info.Value)
			return
		}
	}
} 
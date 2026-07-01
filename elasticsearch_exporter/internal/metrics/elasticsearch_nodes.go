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

// 定义默认节点标签
var defaultNodeLabels = []string{"cluster", "host", "name", "es_master_node", "es_data_node", "es_ingest_node", "es_client_node"}

// NodeStatsResponse 是ES节点统计信息响应
type nodeStatsResponse struct {
	ClusterName string                      `json:"cluster_name"`
	Nodes       map[string]NodeStatsNodeResponse `json:"nodes"`
}

// NodeStatsNodeResponse 定义节点统计信息结构体
type NodeStatsNodeResponse struct {
	Name             string                                     `json:"name"`
	Host             string                                     `json:"host"`
	Timestamp        int64                                      `json:"timestamp"`
	TransportAddress string                                     `json:"transport_address"`
	Hostname         string                                     `json:"hostname"`
	Roles            []string                                   `json:"roles"`
	Attributes       map[string]string                          `json:"attributes"`
	Indices          NodeStatsIndicesResponse                   `json:"indices"`
	OS               NodeStatsOSResponse                        `json:"os"`
	Process          NodeStatsProcessResponse                   `json:"process"`
	JVM              NodeStatsJVMResponse                       `json:"jvm"`
	ThreadPool       map[string]NodeStatsThreadPoolPoolResponse `json:"thread_pool"`
	HTTP             map[string]interface{}                     `json:"http"`
	Breakers         map[string]NodeStatsBreakersResponse       `json:"breakers"`
	FS               NodeStatsFSResponse                        `json:"fs"`
	IndexingPressure NodeStatsIndexingPressureResponse          `json:"indexing_pressure"`
}

// NodeStatsJVMResponse 是节点统计信息的JVM响应
type NodeStatsJVMResponse struct {
	Mem     NodeStatsJVMMemResponse                   `json:"mem"`
	Uptime  int64                                     `json:"uptime_in_millis"`
	GC      NodeStatsJVMGCResponse                    `json:"gc"`
	Threads NodeStatsJVMThreadsResponse               `json:"threads"`
	Classes NodeStatsJVMClassesResponse               `json:"classes"`
}

// 节点统计信息的JVM内存响应
type NodeStatsJVMMemResponse struct {
	HeapCommitted int64 `json:"heap_committed_in_bytes"`
	HeapUsed      int64 `json:"heap_used_in_bytes"`
	HeapMax       int64 `json:"heap_max_in_bytes"`
}

// 节点统计信息的JVM GC响应
type NodeStatsJVMGCResponse struct {
	Collectors map[string]NodeStatsJVMGCCollectorResponse `json:"collectors"`
}

// 节点统计信息的JVM GC收集器响应
type NodeStatsJVMGCCollectorResponse struct {
	CollectionCount int64 `json:"collection_count"`
	CollectionTime  int64 `json:"collection_time_in_millis"`
}

// 节点统计信息的JVM线程响应
type NodeStatsJVMThreadsResponse struct {
	Count     int64 `json:"count"`
	PeakCount int64 `json:"peak_count"`
}

// 节点统计信息的JVM类加载响应
type NodeStatsJVMClassesResponse struct {
	CurrentLoadedCount int64 `json:"current_loaded_count"`
	TotalLoadedCount   int64 `json:"total_loaded_count"`
	TotalUnloadedCount int64 `json:"total_unloaded_count"`
}

// 节点统计信息的OS响应
type NodeStatsOSResponse struct {
	Timestamp int64                     `json:"timestamp"`
	Uptime    int64                     `json:"uptime_in_millis"`
	LoadAvg   json.RawMessage           `json:"load_average"`
	CPU       NodeStatsOSCPUResponse    `json:"cpu"`
	Mem       NodeStatsOSMemResponse    `json:"mem"`
}

// 节点统计信息的OS-CPU负载响应
type NodeStatsOSCPUResponse struct {
	Percent int64 `json:"percent"`
	LoadAvg NodeStatsOSCPULoadResponse `json:"load_average"`
}

type NodeStatsOSCPULoadResponse struct {
	Load1  float64 `json:"1m"`
	Load5  float64 `json:"5m"`
	Load15 float64 `json:"15m"`
}

// 节点统计信息的OS内存响应
type NodeStatsOSMemResponse struct {
	Free       int64 `json:"free_in_bytes"`
	Used       int64 `json:"used_in_bytes"`
	ActualFree int64 `json:"actual_free_in_bytes"`
	ActualUsed int64 `json:"actual_used_in_bytes"`
}

// 节点统计信息的进程响应
type NodeStatsProcessResponse struct {
	OpenFD int64                       `json:"open_file_descriptors"`
	MaxFD  int64                       `json:"max_file_descriptors"`
	CPU    NodeStatsProcessCPUResponse `json:"cpu"`
	Memory NodeStatsProcessMemResponse `json:"mem"`
}

// 节点统计信息的进程CPU响应
type NodeStatsProcessCPUResponse struct {
	Percent int64 `json:"percent"`
	Total   int64 `json:"total_in_millis"`
}

// 节点统计信息的进程内存响应
type NodeStatsProcessMemResponse struct {
	Resident     int64 `json:"resident_in_bytes"`
	Share        int64 `json:"share_in_bytes"`
	TotalVirtual int64 `json:"total_virtual_in_bytes"`
}

// 节点统计信息的断路器响应
type NodeStatsBreakersResponse struct {
	EstimatedSize int64   `json:"estimated_size_in_bytes"`
	LimitSize     int64   `json:"limit_size_in_bytes"`
	Overhead      float64 `json:"overhead"`
	Tripped       int64   `json:"tripped"`
}

// 节点统计信息的请求缓存响应
type NodeStatsIndicesRequestCacheResponse struct {
	MemorySize  int64 `json:"memory_size_in_bytes"`
	Evictions   int64 `json:"evictions"`
	HitCount    int64 `json:"hit_count"`
	MissCount   int64 `json:"miss_count"`
}

// 节点统计信息的索引响应
type NodeStatsIndicesResponse struct {
	Docs         NodeStatsIndicesDocsResponse       `json:"docs"`
	Store        NodeStatsIndicesStoreResponse      `json:"store"`
	Indexing     NodeStatsIndicesIndexingResponse   `json:"indexing"`
	Search       NodeStatsIndicesSearchResponse     `json:"search"`
	Get          NodeStatsIndicesGetResponse        `json:"get"`
	Refresh      NodeStatsIndicesRefreshResponse    `json:"refresh"`
	Flush        NodeStatsIndicesFlushResponse      `json:"flush"`
	Warmer       NodeStatsIndicesWarmerResponse     `json:"warmer"`
	Segments     NodeStatsIndicesSegmentsResponse   `json:"segments"`
	Translog     NodeStatsIndicesTranslogResponse   `json:"translog"`
	FieldData    NodeStatsIndicesFieldDataResponse  `json:"fielddata"`
	QueryCache   NodeStatsIndicesQueryCacheResponse `json:"query_cache"`
	FilterCache  NodeStatsIndicesFilterCacheResponse `json:"filter_cache"`
	Completion   NodeStatsIndicesCompletionResponse `json:"completion"`
	RequestCache NodeStatsIndicesRequestCacheResponse `json:"request_cache"`
}

// 节点统计信息的索引文档响应
type NodeStatsIndicesDocsResponse struct {
	Count   int64 `json:"count"`
	Deleted int64 `json:"deleted"`
}

// 节点统计信息的索引存储响应
type NodeStatsIndicesStoreResponse struct {
	Size        int64 `json:"size_in_bytes"`
	ThrottleTime int64 `json:"throttle_time_in_millis"`
}

// 节点统计信息的索引操作响应
type NodeStatsIndicesIndexingResponse struct {
	IndexTotal    int64 `json:"index_total"`
	IndexTime     int64 `json:"index_time_in_millis"`
	IndexCurrent  int64 `json:"index_current"`
	DeleteTotal   int64 `json:"delete_total"`
	DeleteTime    int64 `json:"delete_time_in_millis"`
	DeleteCurrent int64 `json:"delete_current"`
}

// 节点统计信息的搜索响应
type NodeStatsIndicesSearchResponse struct {
	QueryTotal   int64 `json:"query_total"`
	QueryTime    int64 `json:"query_time_in_millis"`
	QueryCurrent int64 `json:"query_current"`
	FetchTotal   int64 `json:"fetch_total"`
	FetchTime    int64 `json:"fetch_time_in_millis"`
	FetchCurrent int64 `json:"fetch_current"`
	SuggestTotal int64 `json:"suggest_total"`
	SuggestTime  int64 `json:"suggest_time_in_millis"`
	ScrollTotal  int64 `json:"scroll_total"`
	ScrollTime   int64 `json:"scroll_time_in_millis"`
}

// 节点统计信息的Get响应
type NodeStatsIndicesGetResponse struct {
	Total        int64 `json:"total"`
	Time         int64 `json:"time_in_millis"`
	ExistsTotal  int64 `json:"exists_total"`
	ExistsTime   int64 `json:"exists_time_in_millis"`
	MissingTotal int64 `json:"missing_total"`
	MissingTime  int64 `json:"missing_time_in_millis"`
	Current      int64 `json:"current"`
}

// 节点统计信息的刷新响应
type NodeStatsIndicesRefreshResponse struct {
	Total          int64 `json:"total"`
	TotalTime      int64 `json:"total_time_in_millis"`
	ExternalTotal  int64 `json:"external_total"`
	ExternalTotalTimeInMillis int64 `json:"external_total_time_in_millis"`
}

// 节点统计信息的清理响应
type NodeStatsIndicesFlushResponse struct {
	Total int64 `json:"total"`
	Time  int64 `json:"total_time_in_millis"`
}

// 节点统计信息的预热响应
type NodeStatsIndicesWarmerResponse struct {
	Total     int64 `json:"total"`
	TotalTime int64 `json:"total_time_in_millis"`
}

// 节点统计信息的段响应
type NodeStatsIndicesSegmentsResponse struct {
	Count                int64 `json:"count"`
	Memory               int64 `json:"memory_in_bytes"`
	TermsMemory          int64 `json:"terms_memory_in_bytes"`
	IndexWriterMemory    int64 `json:"index_writer_memory_in_bytes"`
	NormsMemory          int64 `json:"norms_memory_in_bytes"`
	StoredFieldsMemory   int64 `json:"stored_fields_memory_in_bytes"`
	DocValuesMemory      int64 `json:"doc_values_memory_in_bytes"`
	FixedBitSet          int64 `json:"fixed_bit_set_memory_in_bytes"`
	TermVectorsMemory    int64 `json:"term_vectors_memory_in_bytes"`
	PointsMemory         int64 `json:"points_memory_in_bytes"`
	VersionMapMemory     int64 `json:"version_map_memory_in_bytes"`
}

// 节点统计信息的事务日志响应
type NodeStatsIndicesTranslogResponse struct {
	Operations int64 `json:"operations"`
	Size       int64 `json:"size_in_bytes"`
}

// 节点统计信息的字段数据响应
type NodeStatsIndicesFieldDataResponse struct {
	MemorySize  int64 `json:"memory_size_in_bytes"`
	Evictions   int64 `json:"evictions"`
}

// 节点统计信息的查询缓存响应
type NodeStatsIndicesQueryCacheResponse struct {
	MemorySize  int64 `json:"memory_size_in_bytes"`
	TotalCount  int64 `json:"total_count"`
	HitCount    int64 `json:"hit_count"`
	MissCount   int64 `json:"miss_count"`
	CacheSize   int64 `json:"cache_size"`
	CacheCount  int64 `json:"cache_count"`
	Evictions   int64 `json:"evictions"`
}

// 节点统计信息的过滤缓存响应
type NodeStatsIndicesFilterCacheResponse struct {
	MemorySize  int64 `json:"memory_size_in_bytes"`
	Evictions   int64 `json:"evictions"`
}

// 节点统计信息的完成响应
type NodeStatsIndicesCompletionResponse struct {
	Size  int64 `json:"size_in_bytes"`
}

// 节点统计信息的线程池响应
type NodeStatsThreadPoolPoolResponse struct {
	Threads   int64 `json:"threads"`
	Queue     int64 `json:"queue"`
	Active    int64 `json:"active"`
	Rejected  int64 `json:"rejected"`
	Largest   int64 `json:"largest"`
	Completed int64 `json:"completed"`
}

// 节点统计信息的文件系统响应
type NodeStatsFSResponse struct {
	Timestamp int64                     `json:"timestamp"`
	Total     NodeStatsFSTotalResponse  `json:"total"`
	Data      []NodeStatsFSDataResponse `json:"data"`
	IOStats   NodeStatsFSIOStatsResponse `json:"io_stats"`
}

// 节点统计信息的文件系统总量响应
type NodeStatsFSTotalResponse struct {
	Total     int64 `json:"total_in_bytes"`
	Free      int64 `json:"free_in_bytes"`
	Available int64 `json:"available_in_bytes"`
}

// 节点统计信息的文件系统数据响应
type NodeStatsFSDataResponse struct {
	Path      string `json:"path"`
	Mount     string `json:"mount"`
	Total     int64  `json:"total_in_bytes"`
	Free      int64  `json:"free_in_bytes"`
	Available int64  `json:"available_in_bytes"`
}

// 节点统计信息的文件系统IO统计响应
type NodeStatsFSIOStatsResponse struct {
	Devices []NodeStatsFSIOStatsDeviceResponse `json:"devices"`
	Total   NodeStatsFSIOStatsTotalResponse    `json:"total"`
}

// 节点统计信息的文件系统IO设备响应
type NodeStatsFSIOStatsDeviceResponse struct {
	DeviceName      string `json:"device_name"`
	Operations      int64  `json:"operations"`
	ReadOperations  int64  `json:"read_operations"`
	WriteOperations int64  `json:"write_operations"`
	ReadSize        int64  `json:"read_kilobytes"`
	WriteSize       int64  `json:"write_kilobytes"`
}

// 节点统计信息的文件系统IO总量响应
type NodeStatsFSIOStatsTotalResponse struct {
	Operations      int64 `json:"operations"`
	ReadOperations  int64 `json:"read_operations"`
	WriteOperations int64 `json:"write_operations"`
	ReadSize        int64 `json:"read_kilobytes"`
	WriteSize       int64 `json:"write_kilobytes"`
}

// 节点统计信息的索引压力响应
type NodeStatsIndexingPressureResponse struct {
	Memory NodeStatsIndexingPressureMemoryResponse `json:"memory"`
}

// 节点统计信息的索引压力内存响应
type NodeStatsIndexingPressureMemoryResponse struct {
	Current NodeStatsIndexingPressureMemoryCurrentResponse `json:"current"`
	Total   NodeStatsIndexingPressureMemoryTotalResponse   `json:"total"`
	Limit   NodeStatsIndexingPressureMemoryLimitResponse   `json:"limit"`
}

// 节点统计信息的索引压力当前内存响应
type NodeStatsIndexingPressureMemoryCurrentResponse struct {
	CombinedCoordinatingAndPrimaryInBytes int64 `json:"combined_coordinating_and_primary_in_bytes"`
	Coordinating                   int64 `json:"coordinating_in_bytes"`
	Primary                        int64 `json:"primary_in_bytes"`
	Replica                        int64 `json:"replica_in_bytes"`
	AllInBytes                     int64 `json:"all_in_bytes"`
}

// 节点统计信息的索引压力总内存响应
type NodeStatsIndexingPressureMemoryTotalResponse struct {
	CombinedCoordinatingAndPrimaryInBytes int64 `json:"combined_coordinating_and_primary_in_bytes"`
	Coordinating                   int64 `json:"coordinating_in_bytes"`
	Primary                        int64 `json:"primary_in_bytes"`
	Replica                        int64 `json:"replica_in_bytes"`
	AllInBytes                     int64 `json:"all_in_bytes"`
	CoordinatingRejections         int64 `json:"coordinating_rejections"`
	PrimaryRejections              int64 `json:"primary_rejections"`
	ReplicaRejections              int64 `json:"replica_rejections"`
}

// 节点统计信息的索引压力限制内存响应
type NodeStatsIndexingPressureMemoryLimitResponse struct {
	InBytes int64 `json:"in_bytes"`
}

func init() {
	exporter.Register(NewNodes())
}

// 获取节点角色
func getRoles(node NodeStatsNodeResponse) map[string]bool {
	// 默认设置(2.x)和要考虑的角色映射
	roles := map[string]bool{
		"master": false,
		"data":   false,
		"ingest": false,
		"client": true,
	}
	
	// 假设：5.x节点至少有一个角色，否则是1.7或2.x节点
	if len(node.Roles) > 0 {
		for _, role := range node.Roles {
			// 设置不存在的角色为false
			if _, ok := roles[role]; !ok {
				roles[role] = false
			} else {
				// 如果在roles字段中存在，设置为true
				roles[role] = true
			}
		}
	} else {
		for role, setting := range node.Attributes {
			if _, ok := roles[role]; ok {
				if setting == "false" {
					roles[role] = false
				} else {
					roles[role] = true
				}
			}
		}
	}
	
	if len(node.HTTP) == 0 {
		roles["client"] = false
	}
	
	return roles
}

// Nodes 节点指标收集器
type Nodes struct {
	esURL                 string
	client                *http.Client
	insecure              bool
	all                   bool
	node                  string
	jsonParseFailures     prometheus.Counter
	
	// 进程指标
	processOpenFD        *baseMetrics
	processMaxFD         *baseMetrics
	processCPUPercent    *baseMetrics
	processCPUTotal      *baseMetrics
	processMemResident   *baseMetrics
	
	// JVM指标
	jvmMemHeapUsedPercent *baseMetrics
	jvmMemHeapCommitted   *baseMetrics
	jvmMemHeapUsed        *baseMetrics
	jvmMemHeapMax         *baseMetrics
	jvmUptimeSeconds      *baseMetrics
	jvmThreadsCount       *baseMetrics
	jvmThreadsPeakCount   *baseMetrics
	jvmClassesLoaded      *baseMetrics
	jvmClassesTotal       *baseMetrics
	jvmClassesUnloaded    *baseMetrics
	
	// JVM GC指标
	jvmGCCollectorsCollectionCount  map[string]*baseMetrics
	jvmGCCollectorsCollectionTime   map[string]*baseMetrics
	
	// OS指标
	osLoad1              *baseMetrics
	osLoad5              *baseMetrics
	osLoad15             *baseMetrics
	osCPUPercent         *baseMetrics
	osMemFree            *baseMetrics
	osMemUsed            *baseMetrics
	osMemActualFree      *baseMetrics
	osMemActualUsed      *baseMetrics
	
	// 断路器指标
	breakersEstimatedSize *baseMetrics
	breakersLimitSize     *baseMetrics
	breakersOverhead      *baseMetrics
	breakersTripped       *baseMetrics
	
	// 线程池指标
	threadPoolThreads      map[string]*baseMetrics
	threadPoolQueue        map[string]*baseMetrics
	threadPoolActive       map[string]*baseMetrics
	threadPoolRejected     map[string]*baseMetrics
	threadPoolLargest      map[string]*baseMetrics
	threadPoolCompleted    map[string]*baseMetrics
	
	// 文件系统指标
	fsTotal              *baseMetrics
	fsFree               *baseMetrics
	fsAvailable          *baseMetrics
	fsDataTotal          *baseMetrics
	fsDataFree           *baseMetrics
	fsDataAvailable      *baseMetrics
	fsIOStatsTotal       *baseMetrics
	fsIOStatsRead        *baseMetrics
	fsIOStatsWrite       *baseMetrics
	fsIOStatsReadKilobytes *baseMetrics
	fsIOStatsWriteKilobytes *baseMetrics
	fsIOStatsDeviceOperations  *baseMetrics
	fsIOStatsDeviceReadOperations  *baseMetrics
	fsIOStatsDeviceWriteOperations  *baseMetrics
	fsIOStatsDeviceReadKilobytes    *baseMetrics
	fsIOStatsDeviceWriteKilobytes   *baseMetrics
	
	// 索引压力指标
	indexingPressureCurrent  *baseMetrics
	indexingPressureTotal    *baseMetrics
	indexingPressureRejections *baseMetrics
	indexingPressureLimit    *baseMetrics
	
	// 索引文档指标
	indicesDocsCount     *baseMetrics
	indicesDocsDeleted   *baseMetrics
	
	// 索引存储指标
	indicesStoreSize     *baseMetrics
	indicesStoreThrottleTime *baseMetrics
	
	// 索引段指标
	indicesSegmentsCount *baseMetrics
	indicesSegmentsMemory *baseMetrics
	indicesSegmentsTermsMemory *baseMetrics
	indicesSegmentsIndexWriterMemory *baseMetrics
	indicesSegmentsNormsMemory *baseMetrics
	indicesSegmentsStoredFieldsMemory *baseMetrics
	indicesSegmentsDocValuesMemory *baseMetrics
	indicesSegmentsFixedBitSet *baseMetrics
	indicesSegmentsTermVectorsMemory *baseMetrics
	indicesSegmentsPointsMemory *baseMetrics
	indicesSegmentsVersionMapMemory *baseMetrics
	
	// 索引事务日志指标
	indicesTranslogOperations *baseMetrics
	indicesTranslogSize *baseMetrics
	
	// 索引操作指标
	indicesIndexingIndexTotal *baseMetrics
	indicesIndexingIndexTime *baseMetrics
	indicesIndexingDeleteTotal *baseMetrics
	indicesIndexingDeleteTime *baseMetrics
	
	// 索引Get指标
	indicesGetTotal *baseMetrics
	indicesGetTime *baseMetrics
	indicesGetExistsTotal *baseMetrics
	indicesGetExistsTime *baseMetrics
	indicesGetMissingTotal *baseMetrics
	indicesGetMissingTime *baseMetrics
	
	// 索引搜索指标
	indicesSearchQueryTotal *baseMetrics
	indicesSearchQueryTime *baseMetrics
	indicesSearchFetchTotal *baseMetrics
	indicesSearchFetchTime *baseMetrics
	indicesSearchSuggestTotal *baseMetrics
	indicesSearchSuggestTime *baseMetrics
	indicesSearchScrollTotal *baseMetrics
	indicesSearchScrollTime *baseMetrics
	
	// 索引刷新指标
	indicesRefreshTotal *baseMetrics
	indicesRefreshTime *baseMetrics
	indicesRefreshExternalTotal *baseMetrics
	indicesRefreshExternalTime *baseMetrics
	
	// 索引清理指标
	indicesFlushTotal *baseMetrics
	indicesFlushTime *baseMetrics
	
	// 索引预热指标
	indicesWarmerTotal *baseMetrics
	indicesWarmerTime *baseMetrics
	
	// 索引缓存指标
	indicesFielddataMemorySize *baseMetrics
	indicesFielddataEvictions *baseMetrics
	indicesQueryCacheMemorySize *baseMetrics
	indicesQueryCacheEvictions *baseMetrics
	indicesQueryCacheTotal *baseMetrics
	indicesQueryCacheHitCount *baseMetrics
	indicesQueryCacheMissCount *baseMetrics
	indicesQueryCacheCacheSize *baseMetrics
	indicesQueryCacheCacheCount *baseMetrics
	indicesFilterCacheMemorySize *baseMetrics
	indicesFilterCacheEvictions *baseMetrics
	indicesCompletionSize *baseMetrics
	indicesRequestCacheMemorySize *baseMetrics
	indicesRequestCacheEvictions *baseMetrics
	indicesRequestCacheHitCount *baseMetrics
	indicesRequestCacheMissCount *baseMetrics
}

// NewNodes 创建节点指标收集器
func NewNodes() *Nodes {
	// 创建收集器
	nodes := &Nodes{
		esURL: "http://localhost:9200",
		client: &http.Client{
			Timeout: defaultTimeout,
		},
		all:  true,
		node: "_local",
		
		jsonParseFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "nodes_json_parse_failures",
			Help:      "Number of errors while parsing JSON.",
		}),
		
		// 进程指标
		processOpenFD: NewMetrics(
			prometheus.BuildFQName(namespace, "process", "open_files_count"),
			"Open file descriptors",
			defaultNodeLabels,
		),
		processMaxFD: NewMetrics(
			prometheus.BuildFQName(namespace, "process", "max_files_descriptors"),
			"Max file descriptors",
			defaultNodeLabels,
		),
		processCPUPercent: NewMetrics(
			prometheus.BuildFQName(namespace, "process", "cpu_percent"),
			"CPU percentage used by process",
			defaultNodeLabels,
		),
		processCPUTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "process", "cpu_seconds_total"),
			"Process CPU time in seconds",
			defaultNodeLabels,
		),
		processMemResident: NewMetrics(
			prometheus.BuildFQName(namespace, "process", "resident_memory_bytes"),
			"Resident memory in bytes",
			defaultNodeLabels,
		),
		
		// JVM指标
		jvmMemHeapUsedPercent: NewMetrics(
			prometheus.BuildFQName(namespace, "jvm", "memory_heap_used_percent"),
			"JVM heap usage percentage",
			defaultNodeLabels,
		),
		jvmMemHeapCommitted: NewMetrics(
			prometheus.BuildFQName(namespace, "jvm", "memory_heap_committed_bytes"),
			"JVM heap committed in bytes",
			defaultNodeLabels,
		),
		jvmMemHeapUsed: NewMetrics(
			prometheus.BuildFQName(namespace, "jvm", "memory_heap_used_bytes"),
			"JVM heap used in bytes",
			defaultNodeLabels,
		),
		jvmMemHeapMax: NewMetrics(
			prometheus.BuildFQName(namespace, "jvm", "memory_heap_max_bytes"),
			"JVM heap max in bytes",
			defaultNodeLabels,
		),
		jvmUptimeSeconds: NewMetrics(
			prometheus.BuildFQName(namespace, "jvm", "uptime_seconds"),
			"JVM uptime in seconds",
			defaultNodeLabels,
		),
		jvmThreadsCount: NewMetrics(
			prometheus.BuildFQName(namespace, "jvm", "threads_count"),
			"JVM threads count",
			defaultNodeLabels,
		),
		jvmThreadsPeakCount: NewMetrics(
			prometheus.BuildFQName(namespace, "jvm", "threads_peak_count"),
			"JVM threads peak count",
			defaultNodeLabels,
		),
		jvmClassesLoaded: NewMetrics(
			prometheus.BuildFQName(namespace, "jvm", "classes_loaded_count"),
			"JVM classes loaded count",
			defaultNodeLabels,
		),
		jvmClassesTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "jvm", "classes_total_count"),
			"JVM classes total count",
			defaultNodeLabels,
		),
		jvmClassesUnloaded: NewMetrics(
			prometheus.BuildFQName(namespace, "jvm", "classes_unloaded_count"),
			"JVM classes unloaded count",
			defaultNodeLabels,
		),
		
		// JVM GC指标
		jvmGCCollectorsCollectionCount: make(map[string]*baseMetrics),
		jvmGCCollectorsCollectionTime:   make(map[string]*baseMetrics),
		
		// OS指标
		osLoad1: NewMetrics(
			prometheus.BuildFQName(namespace, "os", "load1"),
			"Shortterm load average",
			defaultNodeLabels,
		),
		osLoad5: NewMetrics(
			prometheus.BuildFQName(namespace, "os", "load5"),
			"Midterm load average",
			defaultNodeLabels,
		),
		osLoad15: NewMetrics(
			prometheus.BuildFQName(namespace, "os", "load15"),
			"Longterm load average",
			defaultNodeLabels,
		),
		osCPUPercent: NewMetrics(
			prometheus.BuildFQName(namespace, "os", "cpu_percent"),
			"OS CPU usage percentage",
			defaultNodeLabels,
		),
		osMemFree: NewMetrics(
			prometheus.BuildFQName(namespace, "os", "mem_free_bytes"),
			"Amount of free physical memory in bytes",
			defaultNodeLabels,
		),
		osMemUsed: NewMetrics(
			prometheus.BuildFQName(namespace, "os", "mem_used_bytes"),
			"Amount of used physical memory in bytes",
			defaultNodeLabels,
		),
		osMemActualFree: NewMetrics(
			prometheus.BuildFQName(namespace, "os", "mem_actual_free_bytes"),
			"Amount of free physical memory in bytes",
			defaultNodeLabels,
		),
		osMemActualUsed: NewMetrics(
			prometheus.BuildFQName(namespace, "os", "mem_actual_used_bytes"),
			"Amount of used physical memory in bytes",
			defaultNodeLabels,
		),
		
		// 断路器指标 - 使用带断路器名称的标签
		breakersEstimatedSize: NewMetrics(
			prometheus.BuildFQName(namespace, "breakers", "estimated_size_bytes"),
			"Estimated size in bytes",
			append(defaultNodeLabels, "breaker"),
		),
		breakersLimitSize: NewMetrics(
			prometheus.BuildFQName(namespace, "breakers", "limit_size_bytes"),
			"Limit size in bytes",
			append(defaultNodeLabels, "breaker"),
		),
		breakersOverhead: NewMetrics(
			prometheus.BuildFQName(namespace, "breakers", "overhead"),
			"Overhead",
			append(defaultNodeLabels, "breaker"),
		),
		breakersTripped: NewMetrics(
			prometheus.BuildFQName(namespace, "breakers", "tripped"),
			"Tripped",
			append(defaultNodeLabels, "breaker"),
		),
		
		// 线程池指标
		threadPoolThreads:      make(map[string]*baseMetrics),
		threadPoolQueue:        make(map[string]*baseMetrics),
		threadPoolActive:       make(map[string]*baseMetrics),
		threadPoolRejected:     make(map[string]*baseMetrics),
		threadPoolLargest:      make(map[string]*baseMetrics),
		threadPoolCompleted:    make(map[string]*baseMetrics),
		
		// 文件系统指标
		fsTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "filesystem", "total_bytes"),
			"Total in bytes",
			defaultNodeLabels,
		),
		fsFree: NewMetrics(
			prometheus.BuildFQName(namespace, "filesystem", "free_bytes"),
			"Free in bytes",
			defaultNodeLabels,
		),
		fsAvailable: NewMetrics(
			prometheus.BuildFQName(namespace, "filesystem", "available_bytes"),
			"Available in bytes",
			defaultNodeLabels,
		),
		fsDataTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "filesystem", "data_size_bytes"),
			"Data total in bytes",
			append(defaultNodeLabels, "mount", "path"),
		),
		fsDataFree: NewMetrics(
			prometheus.BuildFQName(namespace, "filesystem", "data_free_bytes"),
			"Data free in bytes",
			append(defaultNodeLabels, "mount", "path"),
		),
		fsDataAvailable: NewMetrics(
			prometheus.BuildFQName(namespace, "filesystem", "data_available_bytes"),
			"Data available in bytes",
			append(defaultNodeLabels, "mount", "path"),
		),
		fsIOStatsTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "filesystem", "io_stats_operations_count"),
			"Total operations",
			defaultNodeLabels,
		),
		fsIOStatsRead: NewMetrics(
			prometheus.BuildFQName(namespace, "filesystem", "io_stats_read_operations_count"),
			"Read operations",
			defaultNodeLabels,
		),
		fsIOStatsWrite: NewMetrics(
			prometheus.BuildFQName(namespace, "filesystem", "io_stats_write_operations_count"),
			"Write operations",
			defaultNodeLabels,
		),
		fsIOStatsReadKilobytes: NewMetrics(
			prometheus.BuildFQName(namespace, "filesystem", "io_stats_read_size_kilobytes_sum"),
			"Read kilobytes",
			defaultNodeLabels,
		),
		fsIOStatsWriteKilobytes: NewMetrics(
			prometheus.BuildFQName(namespace, "filesystem", "io_stats_write_size_kilobytes_sum"),
			"Write kilobytes",
			defaultNodeLabels,
		),
		fsIOStatsDeviceOperations: NewMetrics(
			prometheus.BuildFQName(namespace, "filesystem", "io_stats_device_operations_count"),
			"Device operations",
			append(defaultNodeLabels, "device"),
		),
		fsIOStatsDeviceReadOperations: NewMetrics(
			prometheus.BuildFQName(namespace, "filesystem", "io_stats_device_read_operations_count"),
			"Device read operations",
			append(defaultNodeLabels, "device"),
		),
		fsIOStatsDeviceWriteOperations: NewMetrics(
			prometheus.BuildFQName(namespace, "filesystem", "io_stats_device_write_operations_count"),
			"Device write operations",
			append(defaultNodeLabels, "device"),
		),
		fsIOStatsDeviceReadKilobytes: NewMetrics(
			prometheus.BuildFQName(namespace, "filesystem", "io_stats_device_read_size_kilobytes_sum"),
			"Device read kilobytes",
			append(defaultNodeLabels, "device"),
		),
		fsIOStatsDeviceWriteKilobytes: NewMetrics(
			prometheus.BuildFQName(namespace, "filesystem", "io_stats_device_write_size_kilobytes_sum"),
			"Device write kilobytes",
			append(defaultNodeLabels, "device"),
		),
		
		// 索引压力指标
		indexingPressureCurrent: NewMetrics(
			prometheus.BuildFQName(namespace, "indexing_pressure", "current_all_in_bytes"),
			"Memory consumed, in bytes, by indexing requests in the coordinating, primary, or replica stage",
			append(defaultNodeLabels, "indexing_pressure"),
		),
		indexingPressureTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indexing_pressure", "total_in_bytes"),
			"Total bytes consumed by indexing requests",
			append(defaultNodeLabels, "indexing_pressure"),
		),
		indexingPressureRejections: NewMetrics(
			prometheus.BuildFQName(namespace, "indexing_pressure", "rejections"),
			"Rejections",
			append(defaultNodeLabels, "type"),
		),
		indexingPressureLimit: NewMetrics(
			prometheus.BuildFQName(namespace, "indexing_pressure", "limit_in_bytes"),
			"Configured memory limit, in bytes, for the indexing requests",
			append(defaultNodeLabels, "indexing_pressure"),
		),
		
		// 索引文档指标
		indicesDocsCount: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "docs"),
			"Count of documents on this node",
			defaultNodeLabels,
		),
		indicesDocsDeleted: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "docs_deleted"),
			"Count of deleted documents on this node",
			defaultNodeLabels,
		),
		
		// 索引存储指标
		indicesStoreSize: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "store_size_bytes"),
			"Current size of stored index data in bytes",
			defaultNodeLabels,
		),
		indicesStoreThrottleTime: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "store_throttle_time_seconds_total"),
			"Throttle time for index store in seconds",
			defaultNodeLabels,
		),
		
		// 索引段指标
		indicesSegmentsCount: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "segments_count"),
			"Count of index segments on this node",
			defaultNodeLabels,
		),
		indicesSegmentsMemory: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "segments_memory_bytes"),
			"Current memory size of segments in bytes",
			defaultNodeLabels,
		),
		indicesSegmentsTermsMemory: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "segments_terms_memory_in_bytes"),
			"Count of terms in memory for this node",
			defaultNodeLabels,
		),
		indicesSegmentsIndexWriterMemory: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "segments_index_writer_memory_in_bytes"),
			"Count of memory for index writer on this node",
			defaultNodeLabels,
		),
		indicesSegmentsNormsMemory: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "segments_norms_memory_in_bytes"),
			"Count of memory used by norms",
			defaultNodeLabels,
		),
		indicesSegmentsStoredFieldsMemory: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "segments_stored_fields_memory_in_bytes"),
			"Count of stored fields memory",
			defaultNodeLabels,
		),
		indicesSegmentsDocValuesMemory: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "segments_doc_values_memory_in_bytes"),
			"Count of doc values memory",
			defaultNodeLabels,
		),
		indicesSegmentsFixedBitSet: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "segments_fixed_bit_set_memory_in_bytes"),
			"Count of fixed bit set",
			defaultNodeLabels,
		),
		indicesSegmentsTermVectorsMemory: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "segments_term_vectors_memory_in_bytes"),
			"Term vectors memory usage in bytes",
			defaultNodeLabels,
		),
		indicesSegmentsPointsMemory: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "segments_points_memory_in_bytes"),
			"Point values memory usage in bytes",
			defaultNodeLabels,
		),
		indicesSegmentsVersionMapMemory: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "segments_version_map_memory_in_bytes"),
			"Version map memory usage in bytes",
			defaultNodeLabels,
		),
		
		// 索引事务日志指标
		indicesTranslogOperations: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "translog_operations"),
			"Total translog operations",
			defaultNodeLabels,
		),
		indicesTranslogSize: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "translog_size_in_bytes"),
			"Translog size in bytes",
			defaultNodeLabels,
		),
		
		// 索引操作指标
		indicesIndexingIndexTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices_indexing", "index_total"),
			"Total index calls",
			defaultNodeLabels,
		),
		indicesIndexingIndexTime: NewMetrics(
			prometheus.BuildFQName(namespace, "indices_indexing", "index_time_seconds_total"),
			"Cumulative index time in seconds",
			defaultNodeLabels,
		),
		indicesIndexingDeleteTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices_indexing", "delete_total"),
			"Total indexing deletes",
			defaultNodeLabels,
		),
		indicesIndexingDeleteTime: NewMetrics(
			prometheus.BuildFQName(namespace, "indices_indexing", "delete_time_seconds_total"),
			"Total time indexing delete in seconds",
			defaultNodeLabels,
		),
		
		// 索引Get指标
		indicesGetTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "get_total"),
			"Total get",
			defaultNodeLabels,
		),
		indicesGetTime: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "get_time_seconds"),
			"Total get time in seconds",
			defaultNodeLabels,
		),
		indicesGetExistsTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "get_exists_total"),
			"Total get exists operations",
			defaultNodeLabels,
		),
		indicesGetExistsTime: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "get_exists_time_seconds"),
			"Total time get exists in seconds",
			defaultNodeLabels,
		),
		indicesGetMissingTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "get_missing_total"),
			"Total get missing",
			defaultNodeLabels,
		),
		indicesGetMissingTime: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "get_missing_time_seconds"),
			"Total time of get missing in seconds",
			defaultNodeLabels,
		),
		
		// 索引搜索指标
		indicesSearchQueryTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "search_query_total"),
			"Total number of queries",
			defaultNodeLabels,
		),
		indicesSearchQueryTime: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "search_query_time_seconds"),
			"Total search query time in seconds",
			defaultNodeLabels,
		),
		indicesSearchFetchTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "search_fetch_total"),
			"Total number of fetches",
			defaultNodeLabels,
		),
		indicesSearchFetchTime: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "search_fetch_time_seconds"),
			"Total search fetch time in seconds",
			defaultNodeLabels,
		),
		indicesSearchSuggestTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "search_suggest_total"),
			"Total number of suggests",
			defaultNodeLabels,
		),
		indicesSearchSuggestTime: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "search_suggest_time_seconds"),
			"Total suggest time in seconds",
			defaultNodeLabels,
		),
		indicesSearchScrollTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "search_scroll_total"),
			"Total number of scrolls",
			defaultNodeLabels,
		),
		indicesSearchScrollTime: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "search_scroll_time_seconds"),
			"Total scroll time in seconds",
			defaultNodeLabels,
		),
		
		// 索引刷新指标
		indicesRefreshTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices_refresh", "total"),
			"Total refreshes",
			defaultNodeLabels,
		),
		indicesRefreshTime: NewMetrics(
			prometheus.BuildFQName(namespace, "indices_refresh", "time_seconds_total"),
			"Total time spent refreshing in seconds",
			defaultNodeLabels,
		),
		indicesRefreshExternalTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices_refresh", "external_total"),
			"Total external refreshes",
			defaultNodeLabels,
		),
		indicesRefreshExternalTime: NewMetrics(
			prometheus.BuildFQName(namespace, "indices_refresh", "external_time_seconds_total"),
			"Total time spent external refreshing in seconds",
			defaultNodeLabels,
		),
		
		// 索引清理指标
		indicesFlushTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "flush_total"),
			"Total flushes",
			defaultNodeLabels,
		),
		indicesFlushTime: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "flush_time_seconds"),
			"Cumulative flush time in seconds",
			defaultNodeLabels,
		),
		
		// 索引预热指标
		indicesWarmerTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "warmer_total"),
			"Total warmer count",
			defaultNodeLabels,
		),
		indicesWarmerTime: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "warmer_time_seconds_total"),
			"Total warmer time in seconds",
			defaultNodeLabels,
		),
		
		// 索引缓存指标
		indicesFielddataMemorySize: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "fielddata_memory_size_bytes"),
			"Field data cache memory usage in bytes",
			defaultNodeLabels,
		),
		indicesFielddataEvictions: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "fielddata_evictions"),
			"Evictions from field data",
			defaultNodeLabels,
		),
		indicesQueryCacheMemorySize: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "query_cache_memory_size_bytes"),
			"Query cache memory usage in bytes",
			defaultNodeLabels,
		),
		indicesQueryCacheEvictions: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "query_cache_evictions"),
			"Evictions from query cache",
			defaultNodeLabels,
		),
		indicesQueryCacheTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "query_cache_total"),
			"Query cache total count",
			defaultNodeLabels,
		),
		indicesQueryCacheHitCount: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "query_cache_count"),
			"Query cache hit count",
			defaultNodeLabels,
		),
		indicesQueryCacheMissCount: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "query_miss_count"),
			"Query miss count",
			defaultNodeLabels,
		),
		indicesQueryCacheCacheSize: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "query_cache_cache_size"),
			"Query cache cache size in bytes",
			defaultNodeLabels,
		),
		indicesQueryCacheCacheCount: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "query_cache_cache_count"),
			"Query cache cache count",
			defaultNodeLabels,
		),
		indicesFilterCacheMemorySize: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "filter_cache_memory_size_bytes"),
			"Filter cache memory usage in bytes",
			defaultNodeLabels,
		),
		indicesFilterCacheEvictions: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "filter_cache_evictions"),
			"Evictions from filter cache",
			defaultNodeLabels,
		),
		indicesCompletionSize: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "completion_size_in_bytes"),
			"Completion in bytes",
			defaultNodeLabels,
		),
		indicesRequestCacheMemorySize: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "request_cache_memory_size_bytes"),
			"Request cache memory size in bytes",
			defaultNodeLabels,
		),
		indicesRequestCacheEvictions: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "request_cache_evictions"),
			"Request cache evictions",
			defaultNodeLabels,
		),
		indicesRequestCacheHitCount: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "request_cache_hit_count"),
			"Request cache hit count",
			defaultNodeLabels,
		),
		indicesRequestCacheMissCount: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "request_cache_miss_count"),
			"Request cache miss count",
			defaultNodeLabels,
		),
	}
	
	return nodes
}

// fetchAndDecodeNodeStats 获取并解析节点统计信息
func (c *Nodes) fetchAndDecodeNodeStats() (nodeStatsResponse, error) {
	var nsr nodeStatsResponse
	
	// 确保客户端每次获取时更新配置
	c.client = &http.Client{
		Timeout: defaultTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				// #nosec G402
				InsecureSkipVerify: c.insecure,
			},
		},
	}
	
	u, err := url.Parse(c.esURL)
	if err != nil {
		return nsr, fmt.Errorf("failed to parse ES URL: %s", err)
	}
	
	// 构建URL路径
	var urlPath string
	if c.all {
		urlPath = "/_nodes/_all/stats"
	} else {
		urlPath = fmt.Sprintf("/_nodes/%s/stats", c.node)
	}
	
	u.Path = path.Join(u.Path, urlPath)
	
	logrus.Debugf("Fetching node stats from %s", u.String())
	
	res, err := c.client.Get(u.String())
	if err != nil {
		return nsr, fmt.Errorf("failed to get node stats from %s: %s", u.String(), err)
	}
	
	defer func() {
		err = res.Body.Close()
		if err != nil {
			logrus.Warnf("failed to close http.Client: %s", err)
		}
	}()
	
	if res.StatusCode != http.StatusOK {
		return nsr, fmt.Errorf("HTTP Request failed with code %d", res.StatusCode)
	}
	
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nsr, err
	}
	
	if err := json.Unmarshal(body, &nsr); err != nil {
		c.jsonParseFailures.Inc()
		return nsr, err
	}
	
	return nsr, nil
}

// Collect 实现指标收集
func (c *Nodes) Collect(ch chan<- prometheus.Metric) {
	// 从配置中读取ES URL
	if config.ScrapeUrl != nil && *config.ScrapeUrl != "" {
		c.esURL = *config.ScrapeUrl
		logrus.Debugf("Using scrape_uri from command line: %s", c.esURL)
	} else {
		// 检查配置文件中的ScrapeUri
		var settings config.Settings
		if err := exporter.Unpack(&settings); err == nil && settings.ScrapeUri != "" {
			c.esURL = settings.ScrapeUri
			logrus.Debugf("Using scrape_uri from config file: %s", c.esURL)
		} else {
			logrus.Debugf("Using default scrape_uri: %s", c.esURL)
		}
	}
	
	// 设置是否忽略SSL证书验证
	if config.Insecure != nil && *config.Insecure {
		c.insecure = *config.Insecure
	} else {
		// 检查配置文件中的Insecure设置
		var settings config.Settings
		if err := exporter.Unpack(&settings); err == nil {
			c.insecure = settings.Insecure
		}
	}
	
	// 确保计数器被收集
	ch <- c.jsonParseFailures
	
	// 获取节点统计信息
	nodeStats, err := c.fetchAndDecodeNodeStats()
	if err != nil {
		logrus.Warnf("Failed to fetch and decode node stats: %s", err)
		return
	}
	
	logrus.Debugf("Found %d nodes in cluster", len(nodeStats.Nodes))
	
	// 处理每个节点的统计信息
	for nodeID, node := range nodeStats.Nodes {
		logrus.Debugf("Processing node: %s (%s)", node.Name, nodeID)
		
		// 获取节点角色
		roles := getRoles(node)
		
		// 构建标签值
		labelValues := []string{
			nodeStats.ClusterName,
			node.Host,
			node.Name,
			fmt.Sprintf("%t", roles["master"]),
			fmt.Sprintf("%t", roles["data"]),
			fmt.Sprintf("%t", roles["ingest"]),
			fmt.Sprintf("%t", roles["client"]),
		}
		
		// 收集进程指标
		c.processOpenFD.collect(ch, float64(node.Process.OpenFD), labelValues)
		c.processMaxFD.collect(ch, float64(node.Process.MaxFD), labelValues)
		c.processCPUPercent.collect(ch, float64(node.Process.CPU.Percent), labelValues)
		c.processCPUTotal.collect(ch, float64(node.Process.CPU.Total) / 1000.0, labelValues)
		c.processMemResident.collect(ch, float64(node.Process.Memory.Resident), labelValues)
		
		// 收集JVM指标
		jvmHeapUsedPercent := float64(node.JVM.Mem.HeapUsed) / float64(node.JVM.Mem.HeapMax) * 100.0
		c.jvmMemHeapUsedPercent.collect(ch, jvmHeapUsedPercent, labelValues)
		c.jvmMemHeapCommitted.collect(ch, float64(node.JVM.Mem.HeapCommitted), labelValues)
		c.jvmMemHeapUsed.collect(ch, float64(node.JVM.Mem.HeapUsed), labelValues)
		c.jvmMemHeapMax.collect(ch, float64(node.JVM.Mem.HeapMax), labelValues)
		c.jvmUptimeSeconds.collect(ch, float64(node.JVM.Uptime) / 1000.0, labelValues)
		c.jvmThreadsCount.collect(ch, float64(node.JVM.Threads.Count), labelValues)
		c.jvmThreadsPeakCount.collect(ch, float64(node.JVM.Threads.PeakCount), labelValues)
		c.jvmClassesLoaded.collect(ch, float64(node.JVM.Classes.CurrentLoadedCount), labelValues)
		c.jvmClassesTotal.collect(ch, float64(node.JVM.Classes.TotalLoadedCount), labelValues)
		c.jvmClassesUnloaded.collect(ch, float64(node.JVM.Classes.TotalUnloadedCount), labelValues)
		
		// 解析load_average字段 (复杂性来自ES版本差异)
		var load1, load5, load15 float64
		
		// 尝试从node.OS.CPU.LoadAvg读取（旧版本ES的结构）
		if node.OS.CPU.LoadAvg.Load1 > 0 {
			load1 = node.OS.CPU.LoadAvg.Load1
			load5 = node.OS.CPU.LoadAvg.Load5
			load15 = node.OS.CPU.LoadAvg.Load15
		} else if node.OS.LoadAvg != nil {
			// 如果上面的方式无法获取，尝试从node.OS.LoadAvg读取（用于兼容性）
			var loadMap map[string]interface{}
			var loadArray []interface{}
			
			// 尝试解析为对象格式
			if err := json.Unmarshal(node.OS.LoadAvg, &loadMap); err == nil {
				if val, ok := loadMap["1m"].(float64); ok {
					load1 = val
				}
				if val, ok := loadMap["5m"].(float64); ok {
					load5 = val
				}
				if val, ok := loadMap["15m"].(float64); ok {
					load15 = val
				}
			} else {
				// 尝试解析为数组格式
				if err := json.Unmarshal(node.OS.LoadAvg, &loadArray); err == nil {
					if len(loadArray) > 0 {
						if val, ok := loadArray[0].(float64); ok {
							load1 = val
						}
					}
					if len(loadArray) > 1 {
						if val, ok := loadArray[1].(float64); ok {
							load5 = val
						}
					}
					if len(loadArray) > 2 {
						if val, ok := loadArray[2].(float64); ok {
							load15 = val
						}
					}
				}
			}
		}
		
		// 收集OS指标
		c.osLoad1.collect(ch, load1, labelValues)
		c.osLoad5.collect(ch, load5, labelValues)
		c.osLoad15.collect(ch, load15, labelValues)
		c.osCPUPercent.collect(ch, float64(node.OS.CPU.Percent), labelValues)
		c.osMemFree.collect(ch, float64(node.OS.Mem.Free), labelValues)
		c.osMemUsed.collect(ch, float64(node.OS.Mem.Used), labelValues)
		c.osMemActualFree.collect(ch, float64(node.OS.Mem.ActualFree), labelValues)
		c.osMemActualUsed.collect(ch, float64(node.OS.Mem.ActualUsed), labelValues)
		
		// 收集索引指标
		c.indicesDocsCount.collect(ch, float64(node.Indices.Docs.Count), labelValues)
		c.indicesDocsDeleted.collect(ch, float64(node.Indices.Docs.Deleted), labelValues)
		c.indicesStoreSize.collect(ch, float64(node.Indices.Store.Size), labelValues)
		c.indicesStoreThrottleTime.collect(ch, float64(node.Indices.Store.ThrottleTime) / 1000.0, labelValues)
		c.indicesSegmentsCount.collect(ch, float64(node.Indices.Segments.Count), labelValues)
		c.indicesSegmentsMemory.collect(ch, float64(node.Indices.Segments.Memory), labelValues)
		c.indicesSegmentsTermsMemory.collect(ch, float64(node.Indices.Segments.TermsMemory), labelValues)
		c.indicesSegmentsIndexWriterMemory.collect(ch, float64(node.Indices.Segments.IndexWriterMemory), labelValues)
		c.indicesSegmentsNormsMemory.collect(ch, float64(node.Indices.Segments.NormsMemory), labelValues)
		c.indicesSegmentsStoredFieldsMemory.collect(ch, float64(node.Indices.Segments.StoredFieldsMemory), labelValues)
		c.indicesSegmentsDocValuesMemory.collect(ch, float64(node.Indices.Segments.DocValuesMemory), labelValues)
		c.indicesSegmentsFixedBitSet.collect(ch, float64(node.Indices.Segments.FixedBitSet), labelValues)
		c.indicesSegmentsTermVectorsMemory.collect(ch, float64(node.Indices.Segments.TermVectorsMemory), labelValues)
		c.indicesSegmentsPointsMemory.collect(ch, float64(node.Indices.Segments.PointsMemory), labelValues)
		c.indicesSegmentsVersionMapMemory.collect(ch, float64(node.Indices.Segments.VersionMapMemory), labelValues)
		c.indicesTranslogOperations.collect(ch, float64(node.Indices.Translog.Operations), labelValues)
		c.indicesTranslogSize.collect(ch, float64(node.Indices.Translog.Size), labelValues)
		c.indicesIndexingIndexTotal.collect(ch, float64(node.Indices.Indexing.IndexTotal), labelValues)
		c.indicesIndexingIndexTime.collect(ch, float64(node.Indices.Indexing.IndexTime) / 1000.0, labelValues)
		c.indicesIndexingDeleteTotal.collect(ch, float64(node.Indices.Indexing.DeleteTotal), labelValues)
		c.indicesIndexingDeleteTime.collect(ch, float64(node.Indices.Indexing.DeleteTime) / 1000.0, labelValues)
		c.indicesGetTotal.collect(ch, float64(node.Indices.Get.Total), labelValues)
		c.indicesGetTime.collect(ch, float64(node.Indices.Get.Time) / 1000.0, labelValues)
		c.indicesGetExistsTotal.collect(ch, float64(node.Indices.Get.ExistsTotal), labelValues)
		c.indicesGetExistsTime.collect(ch, float64(node.Indices.Get.ExistsTime) / 1000.0, labelValues)
		c.indicesGetMissingTotal.collect(ch, float64(node.Indices.Get.MissingTotal), labelValues)
		c.indicesGetMissingTime.collect(ch, float64(node.Indices.Get.MissingTime) / 1000.0, labelValues)
		c.indicesSearchQueryTotal.collect(ch, float64(node.Indices.Search.QueryTotal), labelValues)
		c.indicesSearchQueryTime.collect(ch, float64(node.Indices.Search.QueryTime) / 1000.0, labelValues)
		c.indicesSearchFetchTotal.collect(ch, float64(node.Indices.Search.FetchTotal), labelValues)
		c.indicesSearchFetchTime.collect(ch, float64(node.Indices.Search.FetchTime) / 1000.0, labelValues)
		c.indicesSearchSuggestTotal.collect(ch, float64(node.Indices.Search.SuggestTotal), labelValues)
		c.indicesSearchSuggestTime.collect(ch, float64(node.Indices.Search.SuggestTime) / 1000.0, labelValues)
		c.indicesSearchScrollTotal.collect(ch, float64(node.Indices.Search.ScrollTotal), labelValues)
		c.indicesSearchScrollTime.collect(ch, float64(node.Indices.Search.ScrollTime) / 1000.0, labelValues)
		c.indicesRefreshTotal.collect(ch, float64(node.Indices.Refresh.Total), labelValues)
		c.indicesRefreshTime.collect(ch, float64(node.Indices.Refresh.TotalTime) / 1000.0, labelValues)
		c.indicesRefreshExternalTotal.collect(ch, float64(node.Indices.Refresh.ExternalTotal), labelValues)
		c.indicesRefreshExternalTime.collect(ch, float64(node.Indices.Refresh.ExternalTotalTimeInMillis) / 1000.0, labelValues)
		c.indicesFlushTotal.collect(ch, float64(node.Indices.Flush.Total), labelValues)
		c.indicesFlushTime.collect(ch, float64(node.Indices.Flush.Time) / 1000.0, labelValues)
		c.indicesWarmerTotal.collect(ch, float64(node.Indices.Warmer.Total), labelValues)
		c.indicesWarmerTime.collect(ch, float64(node.Indices.Warmer.TotalTime) / 1000.0, labelValues)
		c.indicesFielddataMemorySize.collect(ch, float64(node.Indices.FieldData.MemorySize), labelValues)
		c.indicesFielddataEvictions.collect(ch, float64(node.Indices.FieldData.Evictions), labelValues)
		c.indicesQueryCacheMemorySize.collect(ch, float64(node.Indices.QueryCache.MemorySize), labelValues)
		c.indicesQueryCacheEvictions.collect(ch, float64(node.Indices.QueryCache.Evictions), labelValues)
		c.indicesQueryCacheTotal.collect(ch, float64(node.Indices.QueryCache.TotalCount), labelValues)
		c.indicesQueryCacheHitCount.collect(ch, float64(node.Indices.QueryCache.HitCount), labelValues)
		c.indicesQueryCacheMissCount.collect(ch, float64(node.Indices.QueryCache.MissCount), labelValues)
		c.indicesQueryCacheCacheSize.collect(ch, float64(node.Indices.QueryCache.CacheSize), labelValues)
		c.indicesQueryCacheCacheCount.collect(ch, float64(node.Indices.QueryCache.CacheCount), labelValues)
		c.indicesFilterCacheMemorySize.collect(ch, float64(node.Indices.FilterCache.MemorySize), labelValues)
		c.indicesFilterCacheEvictions.collect(ch, float64(node.Indices.FilterCache.Evictions), labelValues)
		c.indicesCompletionSize.collect(ch, float64(node.Indices.Completion.Size), labelValues)
		c.indicesRequestCacheMemorySize.collect(ch, float64(node.Indices.RequestCache.MemorySize), labelValues)
		c.indicesRequestCacheEvictions.collect(ch, float64(node.Indices.RequestCache.Evictions), labelValues)
		c.indicesRequestCacheHitCount.collect(ch, float64(node.Indices.RequestCache.HitCount), labelValues)
		c.indicesRequestCacheMissCount.collect(ch, float64(node.Indices.RequestCache.MissCount), labelValues)
		
		// 收集断路器指标
		for breakerName, breakerStats := range node.Breakers {
			// 创建带断路器名称的标签值
			breakerLabelValues := append(labelValues, breakerName)
			
			c.breakersEstimatedSize.collect(ch, float64(breakerStats.EstimatedSize), breakerLabelValues)
			c.breakersLimitSize.collect(ch, float64(breakerStats.LimitSize), breakerLabelValues)
			c.breakersOverhead.collect(ch, breakerStats.Overhead, breakerLabelValues)
			c.breakersTripped.collect(ch, float64(breakerStats.Tripped), breakerLabelValues)
		}
		
		// 收集JVM GC指标
		for collectorName, collectorStats := range node.JVM.GC.Collectors {
			// 确保收集器指标已初始化
			if c.jvmGCCollectorsCollectionCount[collectorName] == nil {
				c.jvmGCCollectorsCollectionCount[collectorName] = NewMetrics(
					prometheus.BuildFQName(namespace, "jvm_gc", "collection_count"),
					"JVM garbage collector collection count",
					append(defaultNodeLabels, "collector"),
				)
			}
			if c.jvmGCCollectorsCollectionTime[collectorName] == nil {
				c.jvmGCCollectorsCollectionTime[collectorName] = NewMetrics(
					prometheus.BuildFQName(namespace, "jvm_gc", "collection_time_seconds"),
					"JVM garbage collector collection time",
					append(defaultNodeLabels, "collector"),
				)
			}
			
			// 创建带收集器名称的标签值
			collectorLabelValues := append(labelValues, collectorName)
			
			c.jvmGCCollectorsCollectionCount[collectorName].collect(ch, float64(collectorStats.CollectionCount), collectorLabelValues)
			c.jvmGCCollectorsCollectionTime[collectorName].collect(ch, float64(collectorStats.CollectionTime) / 1000.0, collectorLabelValues)
		}
		
		// 收集线程池指标
		for poolName, poolStats := range node.ThreadPool {
			// 确保线程池指标已初始化
			if c.threadPoolThreads[poolName] == nil {
				c.threadPoolThreads[poolName] = NewMetrics(
					prometheus.BuildFQName(namespace, "thread_pool", "threads_count"),
					"Thread pool threads count",
					append(defaultNodeLabels, "type"),
				)
			}
			if c.threadPoolQueue[poolName] == nil {
				c.threadPoolQueue[poolName] = NewMetrics(
					prometheus.BuildFQName(namespace, "thread_pool", "queue_count"),
					"Thread pool queue count",
					append(defaultNodeLabels, "type"),
				)
			}
			if c.threadPoolActive[poolName] == nil {
				c.threadPoolActive[poolName] = NewMetrics(
					prometheus.BuildFQName(namespace, "thread_pool", "active_count"),
					"Thread pool active threads",
					append(defaultNodeLabels, "type"),
				)
			}
			if c.threadPoolRejected[poolName] == nil {
				c.threadPoolRejected[poolName] = NewMetrics(
					prometheus.BuildFQName(namespace, "thread_pool", "rejected_count"),
					"Thread pool rejected tasks",
					append(defaultNodeLabels, "type"),
				)
			}
			if c.threadPoolLargest[poolName] == nil {
				c.threadPoolLargest[poolName] = NewMetrics(
					prometheus.BuildFQName(namespace, "thread_pool", "largest_count"),
					"Thread pool largest threads count",
					append(defaultNodeLabels, "type"),
				)
			}
			if c.threadPoolCompleted[poolName] == nil {
				c.threadPoolCompleted[poolName] = NewMetrics(
					prometheus.BuildFQName(namespace, "thread_pool", "completed_count"),
					"Thread pool completed tasks",
					append(defaultNodeLabels, "type"),
				)
			}
			
			// 创建带线程池名称的标签值
			poolLabelValues := append(labelValues, poolName)
			
			c.threadPoolThreads[poolName].collect(ch, float64(poolStats.Threads), poolLabelValues)
			c.threadPoolQueue[poolName].collect(ch, float64(poolStats.Queue), poolLabelValues)
			c.threadPoolActive[poolName].collect(ch, float64(poolStats.Active), poolLabelValues)
			c.threadPoolRejected[poolName].collect(ch, float64(poolStats.Rejected), poolLabelValues)
			c.threadPoolLargest[poolName].collect(ch, float64(poolStats.Largest), poolLabelValues)
			c.threadPoolCompleted[poolName].collect(ch, float64(poolStats.Completed), poolLabelValues)
		}
		
		// 收集文件系统指标
		c.fsTotal.collect(ch, float64(node.FS.Total.Total), labelValues)
		c.fsFree.collect(ch, float64(node.FS.Total.Free), labelValues)
		c.fsAvailable.collect(ch, float64(node.FS.Total.Available), labelValues)
		
		// 收集文件系统IO统计指标
		if node.FS.IOStats.Total.Operations > 0 {
			c.fsIOStatsTotal.collect(ch, float64(node.FS.IOStats.Total.Operations), labelValues)
			c.fsIOStatsRead.collect(ch, float64(node.FS.IOStats.Total.ReadOperations), labelValues)
			c.fsIOStatsWrite.collect(ch, float64(node.FS.IOStats.Total.WriteOperations), labelValues)
			c.fsIOStatsReadKilobytes.collect(ch, float64(node.FS.IOStats.Total.ReadSize), labelValues)
			c.fsIOStatsWriteKilobytes.collect(ch, float64(node.FS.IOStats.Total.WriteSize), labelValues)
		}
		
		// 收集文件系统数据指标
		for _, fsData := range node.FS.Data {
			dataLabelValues := append(labelValues, fsData.Mount, fsData.Path)
			c.fsDataTotal.collect(ch, float64(fsData.Total), dataLabelValues)
			c.fsDataFree.collect(ch, float64(fsData.Free), dataLabelValues)
			c.fsDataAvailable.collect(ch, float64(fsData.Available), dataLabelValues)
		}
		
		// 收集文件系统IO设备指标
		for _, device := range node.FS.IOStats.Devices {
			deviceLabelValues := append(labelValues, device.DeviceName)
			c.fsIOStatsDeviceOperations.collect(ch, float64(device.Operations), deviceLabelValues)
			c.fsIOStatsDeviceReadOperations.collect(ch, float64(device.ReadOperations), deviceLabelValues)
			c.fsIOStatsDeviceWriteOperations.collect(ch, float64(device.WriteOperations), deviceLabelValues)
			c.fsIOStatsDeviceReadKilobytes.collect(ch, float64(device.ReadSize), deviceLabelValues)
			c.fsIOStatsDeviceWriteKilobytes.collect(ch, float64(device.WriteSize), deviceLabelValues)
		}
		
		// 收集索引压力指标
		if node.IndexingPressure.Memory.Current.AllInBytes > 0 {
			// 收集索引压力数据
			// 当前使用字节数
			c.indexingPressureCurrent.collect(ch, float64(node.IndexingPressure.Memory.Current.AllInBytes), append(labelValues, "memory"))
			
			// 总使用字节数
			c.indexingPressureTotal.collect(ch, float64(node.IndexingPressure.Memory.Total.CombinedCoordinatingAndPrimaryInBytes), append(labelValues, "memory"))
			
			// 限制字节数
			c.indexingPressureLimit.collect(ch, float64(node.IndexingPressure.Memory.Limit.InBytes), append(labelValues, "memory"))
			
			// 拒绝次数
			c.indexingPressureRejections.collect(ch, float64(node.IndexingPressure.Memory.Total.CoordinatingRejections), append(labelValues, "coordinating"))
			c.indexingPressureRejections.collect(ch, float64(node.IndexingPressure.Memory.Total.PrimaryRejections), append(labelValues, "primary"))
			c.indexingPressureRejections.collect(ch, float64(node.IndexingPressure.Memory.Total.ReplicaRejections), append(labelValues, "replica"))
		}
	}
} 

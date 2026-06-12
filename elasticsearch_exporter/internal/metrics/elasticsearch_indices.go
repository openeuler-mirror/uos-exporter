package metrics

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"sync"
	"elasticsearch_exporter/config"
	"elasticsearch_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// indexStatsResponse 是 Elasticsearch 索引统计信息的表示
type indexStatsResponse struct {
	Shards  IndexStatsShardsResponse           `json:"_shards"`
	All     IndexStatsIndexResponse            `json:"_all"`
	Indices map[string]IndexStatsIndexResponse `json:"indices"`
}

// IndexStatsShardsResponse 定义索引统计信息中分片信息的结构
type IndexStatsShardsResponse struct {
	Total      int64 `json:"total"`
	Successful int64 `json:"successful"`
	Failed     int64 `json:"failed"`
}

// IndexStatsIndexResponse 定义索引统计信息中索引信息的结构
type IndexStatsIndexResponse struct {
	Primaries IndexStatsIndexDetailResponse `json:"primaries"`
	Total     IndexStatsIndexDetailResponse `json:"total"`
}

// IndexStatsIndexDetailResponse 定义索引统计信息中索引详情的结构
type IndexStatsIndexDetailResponse struct {
	Docs       IndexStatsIndexDocsResponse       `json:"docs"`
	Store      IndexStatsIndexStoreResponse      `json:"store"`
	Indexing   IndexStatsIndexIndexingResponse   `json:"indexing"`
	Get        IndexStatsIndexGetResponse        `json:"get"`
	Search     IndexStatsIndexSearchResponse     `json:"search"`
	Merges     IndexStatsIndexMergesResponse     `json:"merges"`
	Refresh    IndexStatsIndexRefreshResponse    `json:"refresh"`
	Flush      IndexStatsIndexFlushResponse      `json:"flush"`
	Warmer     IndexStatsIndexWarmerResponse     `json:"warmer"`
	QueryCache IndexStatsIndexQueryCacheResponse `json:"query_cache"`
	Fielddata  IndexStatsIndexFielddataResponse  `json:"fielddata"`
	Completion IndexStatsIndexCompletionResponse `json:"completion"`
	Segments   IndexStatsIndexSegmentsResponse   `json:"segments"`
	Translog   IndexStatsIndexTranslogResponse   `json:"translog"`
}

// IndexStatsIndexDocsResponse 定义索引统计信息中文档信息的结构
type IndexStatsIndexDocsResponse struct {
	Count   int64 `json:"count"`
	Deleted int64 `json:"deleted"`
}

// IndexStatsIndexStoreResponse 定义索引统计信息中存储信息的结构
type IndexStatsIndexStoreResponse struct {
	SizeInBytes          int64 `json:"size_in_bytes"`
	ThrottleTimeInMillis int64 `json:"throttle_time_in_millis"`
}

// IndexStatsIndexIndexingResponse 定义索引统计信息中索引操作信息的结构
type IndexStatsIndexIndexingResponse struct {
	IndexTotal           int64 `json:"index_total"`
	IndexTimeInMillis    int64 `json:"index_time_in_millis"`
	IndexCurrent         int64 `json:"index_current"`
	IndexFailed          int64 `json:"index_failed"`
	DeleteTotal          int64 `json:"delete_total"`
	DeleteTimeInMillis   int64 `json:"delete_time_in_millis"`
	DeleteCurrent        int64 `json:"delete_current"`
	NoopUpdateTotal      int64 `json:"noop_update_total"`
	IsThrottled          bool  `json:"is_throttled"`
	ThrottleTimeInMillis int64 `json:"throttle_time_in_millis"`
}

// IndexStatsIndexGetResponse 定义索引统计信息中获取操作信息的结构
type IndexStatsIndexGetResponse struct {
	Total               int64 `json:"total"`
	TimeInMillis        int64 `json:"time_in_millis"`
	ExistsTotal         int64 `json:"exists_total"`
	ExistsTimeInMillis  int64 `json:"exists_time_in_millis"`
	MissingTotal        int64 `json:"missing_total"`
	MissingTimeInMillis int64 `json:"missing_time_in_millis"`
	Current             int64 `json:"current"`
}

// IndexStatsIndexSearchResponse 定义索引统计信息中搜索操作信息的结构
type IndexStatsIndexSearchResponse struct {
	OpenContexts        int64 `json:"open_contexts"`
	QueryTotal          int64 `json:"query_total"`
	QueryTimeInMillis   int64 `json:"query_time_in_millis"`
	QueryCurrent        int64 `json:"query_current"`
	FetchTotal          int64 `json:"fetch_total"`
	FetchTimeInMillis   int64 `json:"fetch_time_in_millis"`
	FetchCurrent        int64 `json:"fetch_current"`
	ScrollTotal         int64 `json:"scroll_total"`
	ScrollTimeInMillis  int64 `json:"scroll_time_in_millis"`
	ScrollCurrent       int64 `json:"scroll_current"`
	SuggestTotal        int64 `json:"suggest_total"`
	SuggestTimeInMillis int64 `json:"suggest_time_in_millis"`
	SuggestCurrent      int64 `json:"suggest_current"`
}

// IndexStatsIndexMergesResponse 定义索引统计信息中合并操作信息的结构
type IndexStatsIndexMergesResponse struct {
	Current                    int64 `json:"current"`
	CurrentDocs                int64 `json:"current_docs"`
	CurrentSizeInBytes         int64 `json:"current_size_in_bytes"`
	Total                      int64 `json:"total"`
	TotalTimeInMillis          int64 `json:"total_time_in_millis"`
	TotalDocs                  int64 `json:"total_docs"`
	TotalSizeInBytes           int64 `json:"total_size_in_bytes"`
	TotalStoppedTimeInMillis   int64 `json:"total_stopped_time_in_millis"`
	TotalThrottledTimeInMillis int64 `json:"total_throttled_time_in_millis"`
	TotalAutoThrottleInBytes   int64 `json:"total_auto_throttle_in_bytes"`
}

// IndexStatsIndexRefreshResponse 定义索引统计信息中刷新操作信息的结构
type IndexStatsIndexRefreshResponse struct {
	Total             int64 `json:"total"`
	TotalTimeInMillis int64 `json:"total_time_in_millis"`
}

// IndexStatsIndexFlushResponse 定义索引统计信息中刷出操作信息的结构
type IndexStatsIndexFlushResponse struct {
	Total             int64 `json:"total"`
	TotalTimeInMillis int64 `json:"total_time_in_millis"`
}

// IndexStatsIndexWarmerResponse 定义索引统计信息中预热操作信息的结构
type IndexStatsIndexWarmerResponse struct {
	Current           int64 `json:"current"`
	Total             int64 `json:"total"`
	TotalTimeInMillis int64 `json:"total_time_in_millis"`
}

// IndexStatsIndexQueryCacheResponse 定义索引统计信息中查询缓存信息的结构
type IndexStatsIndexQueryCacheResponse struct {
	MemorySizeInBytes int64 `json:"memory_size_in_bytes"`
	TotalCount        int64 `json:"total_count"`
	HitCount          int64 `json:"hit_count"`
	MissCount         int64 `json:"miss_count"`
	CacheSize         int64 `json:"cache_size"`
	CacheCount        int64 `json:"cache_count"`
	Evictions         int64 `json:"evictions"`
}

// IndexStatsIndexFielddataResponse 定义索引统计信息中字段数据信息的结构
type IndexStatsIndexFielddataResponse struct {
	MemorySizeInBytes int64 `json:"memory_size_in_bytes"`
	Evictions         int64 `json:"evictions"`
}

// IndexStatsIndexCompletionResponse 定义索引统计信息中完成信息的结构
type IndexStatsIndexCompletionResponse struct {
	SizeInBytes int64 `json:"size_in_bytes"`
}

// IndexStatsIndexSegmentsResponse 定义索引统计信息中段信息的结构
type IndexStatsIndexSegmentsResponse struct {
	Count                     int64 `json:"count"`
	MemoryInBytes             int64 `json:"memory_in_bytes"`
	TermsMemoryInBytes        int64 `json:"terms_memory_in_bytes"`
	StoredFieldsMemoryInBytes int64 `json:"stored_fields_memory_in_bytes"`
	TermVectorsMemoryInBytes  int64 `json:"term_vectors_memory_in_bytes"`
	NormsMemoryInBytes        int64 `json:"norms_memory_in_bytes"`
	PointsMemoryInBytes       int64 `json:"points_memory_in_bytes"`
	DocValuesMemoryInBytes    int64 `json:"doc_values_memory_in_bytes"`
	IndexWriterMemoryInBytes  int64 `json:"index_writer_memory_in_bytes"`
	VersionMapMemoryInBytes   int64 `json:"version_map_memory_in_bytes"`
	FixedBitSetMemoryInBytes  int64 `json:"fixed_bit_set_memory_in_bytes"`
}

// IndexStatsIndexTranslogResponse 定义索引统计信息中事务日志信息的结构
type IndexStatsIndexTranslogResponse struct {
	Operations  int64 `json:"operations"`
	SizeInBytes int64 `json:"size_in_bytes"`
}

// Indices 索引指标收集器
type Indices struct {
	esURL               string
	client              *http.Client
	insecure            bool
	mu                  sync.Mutex
	clusterName         string
	jsonParseFailures   prometheus.Counter

	// 索引文档指标
	docsPrimary         *baseMetrics
	docsTotal           *baseMetrics
	deletedDocsPrimary  *baseMetrics
	deletedDocsTotal    *baseMetrics

	// 索引存储指标
	storeSizeBytesPrimary  *baseMetrics
	storeSizeBytesTotal    *baseMetrics

	// 索引段指标
	segmentCountPrimary       *baseMetrics
	segmentCountTotal         *baseMetrics
	segmentMemoryBytesPrimary *baseMetrics
	segmentMemoryBytesTotal   *baseMetrics
	
	// 索引段详细指标
	segmentTermsMemoryPrimary        *baseMetrics
	segmentTermsMemoryTotal          *baseMetrics
	segmentStoredFieldsMemoryPrimary *baseMetrics
	segmentStoredFieldsMemoryTotal   *baseMetrics
	segmentTermVectorsMemoryPrimary  *baseMetrics
	segmentTermVectorsMemoryTotal    *baseMetrics
	segmentNormsMemoryPrimary        *baseMetrics
	segmentNormsMemoryTotal          *baseMetrics
	segmentPointsMemoryPrimary       *baseMetrics
	segmentPointsMemoryTotal         *baseMetrics
	segmentDocValuesMemoryPrimary    *baseMetrics
	segmentDocValuesMemoryTotal      *baseMetrics
	segmentIndexWriterMemoryPrimary  *baseMetrics
	segmentIndexWriterMemoryTotal    *baseMetrics
	segmentVersionMapMemoryPrimary   *baseMetrics
	segmentVersionMapMemoryTotal     *baseMetrics
	segmentFixedBitSetMemoryPrimary  *baseMetrics
	segmentFixedBitSetMemoryTotal    *baseMetrics

	// 索引完成指标
	completionSizeBytesPrimary *baseMetrics
	completionSizeBytesTotal   *baseMetrics

	// 索引缓存指标
	fielddataMemoySizeBytes *baseMetrics
	fielddataEvictions      *baseMetrics
	queryCacheMemoySizeBytes *baseMetrics
	queryCacheEvictions      *baseMetrics
	queryCacheHitCount       *baseMetrics
	queryCacheMissCount      *baseMetrics
	queryCacheTotalCount     *baseMetrics

	// 索引操作指标
	indexingIndexTotal      *baseMetrics
	indexingIndexTimeSeconds *baseMetrics
	indexingIndexCurrent     *baseMetrics
	indexingDeleteTotal      *baseMetrics
	indexingDeleteTimeSeconds *baseMetrics
	indexingDeleteCurrent     *baseMetrics

	// 索引获取指标
	getTotal               *baseMetrics
	getTimeSeconds         *baseMetrics
	getExistsTotal         *baseMetrics
	getExistsTimeSeconds   *baseMetrics
	getMissingTotal        *baseMetrics
	getMissingTimeSeconds  *baseMetrics

	// 索引搜索指标
	searchQueryTime  *baseMetrics
	searchQueryTotal *baseMetrics
	searchFetchTotal  *baseMetrics
	searchFetchTimeSeconds *baseMetrics
	searchScrollTotal      *baseMetrics
	searchScrollTimeSeconds *baseMetrics
}

func init() {
	exporter.Register(NewIndices())
}

// NewIndices 创建索引指标收集器
func NewIndices() *Indices {
	// 默认索引标签
	defaultIndexLabels := []string{"index", "cluster"}
	
	// 创建收集器
	indices := &Indices{
		esURL: "http://localhost:9200",
		client: &http.Client{
			Timeout: defaultTimeout,
		},
		clusterName: "unknown_cluster",
		
		jsonParseFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "indices_json_parse_failures",
			Help:      "Number of errors while parsing JSON.",
		}),
		
		// 索引文档指标
		docsPrimary: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "docs_primary"),
			"Count of documents with only primary shards",
			defaultIndexLabels,
		),
		docsTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "docs_total"),
			"Total count of documents",
			defaultIndexLabels,
		),
		deletedDocsPrimary: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "deleted_docs_primary"),
			"Count of deleted documents with only primary shards",
			defaultIndexLabels,
		),
		deletedDocsTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "deleted_docs_total"),
			"Total count of deleted documents",
			defaultIndexLabels,
		),
		
		// 索引存储指标
		storeSizeBytesPrimary: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "store_size_bytes_primary"),
			"Current total size of stored index data in bytes with only primary shards on all nodes",
			defaultIndexLabels,
		),
		storeSizeBytesTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "store_size_bytes_total"),
			"Current total size of stored index data in bytes with all shards on all nodes",
			defaultIndexLabels,
		),
		
		// 索引段指标
		segmentCountPrimary: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "segment_count_primary"),
			"Current number of segments with only primary shards on all nodes",
			defaultIndexLabels,
		),
		segmentCountTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "segment_count_total"),
			"Current number of segments with all shards on all nodes",
			defaultIndexLabels,
		),
		segmentMemoryBytesPrimary: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "segment_memory_bytes_primary"),
			"Current size of segments with only primary shards on all nodes in bytes",
			defaultIndexLabels,
		),
		segmentMemoryBytesTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "segment_memory_bytes_total"),
			"Current size of segments with all shards on all nodes in bytes",
			defaultIndexLabels,
		),
		
		// 索引段详细指标
		segmentTermsMemoryPrimary: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "segment_terms_memory_bytes_primary"),
			"Current size of terms with only primary shards on all nodes in bytes",
			defaultIndexLabels,
		),
		segmentTermsMemoryTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "segment_terms_memory_bytes_total"),
			"Current number of terms with all shards on all nodes in bytes",
			defaultIndexLabels,
		),
		segmentStoredFieldsMemoryPrimary: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "segment_fields_memory_bytes_primary"),
			"Current size of fields with only primary shards on all nodes in bytes",
			defaultIndexLabels,
		),
		segmentStoredFieldsMemoryTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "segment_fields_memory_bytes_total"),
			"Current size of fields with all shards on all nodes in bytes",
			defaultIndexLabels,
		),
		segmentTermVectorsMemoryPrimary: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "segment_term_vectors_memory_bytes_primary"),
			"Current size of term vectors with only primary shards on all nodes in bytes",
			defaultIndexLabels,
		),
		segmentTermVectorsMemoryTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "segment_term_vectors_memory_bytes_total"),
			"Current size of term vectors with all shards on all nodes in bytes",
			defaultIndexLabels,
		),
		segmentNormsMemoryPrimary: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "segment_norms_memory_bytes_primary"),
			"Current size of norms with only primary shards on all nodes in bytes",
			defaultIndexLabels,
		),
		segmentNormsMemoryTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "segment_norms_memory_bytes_total"),
			"Current size of norms with all shards on all nodes in bytes",
			defaultIndexLabels,
		),
		segmentPointsMemoryPrimary: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "segment_points_memory_bytes_primary"),
			"Current size of points with only primary shards on all nodes in bytes",
			defaultIndexLabels,
		),
		segmentPointsMemoryTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "segment_points_memory_bytes_total"),
			"Current size of points with all shards on all nodes in bytes",
			defaultIndexLabels,
		),
		segmentDocValuesMemoryPrimary: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "segment_doc_values_memory_bytes_primary"),
			"Current size of doc values with only primary shards on all nodes in bytes",
			defaultIndexLabels,
		),
		segmentDocValuesMemoryTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "segment_doc_values_memory_bytes_total"),
			"Current size of doc values with all shards on all nodes in bytes",
			defaultIndexLabels,
		),
		segmentIndexWriterMemoryPrimary: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "segment_index_writer_memory_bytes_primary"),
			"Current size of index writer with only primary shards on all nodes in bytes",
			defaultIndexLabels,
		),
		segmentIndexWriterMemoryTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "segment_index_writer_memory_bytes_total"),
			"Current size of index writer with all shards on all nodes in bytes",
			defaultIndexLabels,
		),
		segmentVersionMapMemoryPrimary: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "segment_version_map_memory_bytes_primary"),
			"Current size of version map with only primary shards on all nodes in bytes",
			defaultIndexLabels,
		),
		segmentVersionMapMemoryTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "segment_version_map_memory_bytes_total"),
			"Current size of version map with all shards on all nodes in bytes",
			defaultIndexLabels,
		),
		segmentFixedBitSetMemoryPrimary: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "segment_fixed_bit_set_memory_bytes_primary"),
			"Current size of fixed bit with only primary shards on all nodes in bytes",
			defaultIndexLabels,
		),
		segmentFixedBitSetMemoryTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "segment_fixed_bit_set_memory_bytes_total"),
			"Current size of fixed bit with all shards on all nodes in bytes",
			defaultIndexLabels,
		),
		
		// 索引完成指标
		completionSizeBytesPrimary: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "completion_bytes_primary"),
			"Current size of completion with only primary shards on all nodes in bytes",
			defaultIndexLabels,
		),
		completionSizeBytesTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "completion_bytes_total"),
			"Current size of completion with all shards on all nodes in bytes",
			defaultIndexLabels,
		),
		
		// 索引缓存指标
		fielddataMemoySizeBytes: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "fielddata_memory_bytes"),
			"Field data cache memory usage in bytes",
			defaultIndexLabels,
		),
		fielddataEvictions: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "fielddata_evictions"),
			"Evictions from field data",
			defaultIndexLabels,
		),
		queryCacheMemoySizeBytes: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "query_cache_memory_bytes"),
			"Query cache memory usage in bytes",
			defaultIndexLabels,
		),
		queryCacheEvictions: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "query_cache_evictions"),
			"Evictions from query cache",
			defaultIndexLabels,
		),
		queryCacheHitCount: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "query_cache_hit_count"),
			"Query cache hit count",
			defaultIndexLabels,
		),
		queryCacheMissCount: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "query_cache_miss_count"),
			"Query cache miss count",
			defaultIndexLabels,
		),
		queryCacheTotalCount: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "query_cache_total_count"),
			"Query cache total count",
			defaultIndexLabels,
		),
		
		// 索引操作指标
		indexingIndexTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "indexing_index_total"),
			"Total index calls",
			defaultIndexLabels,
		),
		indexingIndexTimeSeconds: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "indexing_index_time_seconds"),
			"Cumulative index time in seconds",
			defaultIndexLabels,
		),
		indexingIndexCurrent: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "indexing_index_current"),
			"Number of current index operations",
			defaultIndexLabels,
		),
		indexingDeleteTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "indexing_delete_total"),
			"Total indexing deletes",
			defaultIndexLabels,
		),
		indexingDeleteTimeSeconds: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "indexing_delete_time_seconds"),
			"Total time indexing delete in seconds",
			defaultIndexLabels,
		),
		indexingDeleteCurrent: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "indexing_delete_current"),
			"Number of current delete operations",
			defaultIndexLabels,
		),
		
		// 索引获取指标
		getTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "get_total"),
			"Total get",
			defaultIndexLabels,
		),
		getTimeSeconds: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "get_time_seconds"),
			"Total get time in seconds",
			defaultIndexLabels,
		),
		getExistsTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "get_exists_total"),
			"Total get exists operations",
			defaultIndexLabels,
		),
		getExistsTimeSeconds: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "get_exists_time_seconds"),
			"Total time get exists in seconds",
			defaultIndexLabels,
		),
		getMissingTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "get_missing_total"),
			"Total get missing",
			defaultIndexLabels,
		),
		getMissingTimeSeconds: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "get_missing_time_seconds"),
			"Total time of get missing in seconds",
			defaultIndexLabels,
		),
		
		// 索引搜索指标
		searchQueryTime: NewMetrics(
			prometheus.BuildFQName(namespace, "index_stats", "search_query_time_seconds_total"),
			"Total search query time in seconds",
			defaultIndexLabels,
		),
		searchQueryTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "index_stats", "search_query_total"),
			"Total number of queries",
			defaultIndexLabels,
		),
		searchFetchTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "search_fetch_total"),
			"Total number of fetches",
			defaultIndexLabels,
		),
		searchFetchTimeSeconds: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "search_fetch_time_seconds"),
			"Total search fetch time in seconds",
			defaultIndexLabels,
		),
		searchScrollTotal: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "search_scroll_total"),
			"Total number of scrolls",
			defaultIndexLabels,
		),
		searchScrollTimeSeconds: NewMetrics(
			prometheus.BuildFQName(namespace, "indices", "search_scroll_time_seconds"),
			"Total scroll time in seconds",
			defaultIndexLabels,
		),
	}
	
	return indices
}

// fetchAndDecodeIndexStats 获取并解析索引统计信息
func (i *Indices) fetchAndDecodeIndexStats() (indexStatsResponse, error) {
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
		return indexStatsResponse{}, fmt.Errorf("failed to parse ES URL: %s", err)
	}
	
	// 构建URL路径
	u.Path = path.Join(u.Path, "/_stats")
	
	logrus.Debugf("Fetching index stats from %s", u.String())
	
	res, err := i.client.Get(u.String())
	if err != nil {
		return indexStatsResponse{}, fmt.Errorf("failed to get index stats from %s: %s", u.String(), err)
	}
	
	defer func() {
		err = res.Body.Close()
		if err != nil {
			logrus.Warnf("failed to close http.Client: %s", err)
		}
	}()
	
	if res.StatusCode != http.StatusOK {
		return indexStatsResponse{}, fmt.Errorf("HTTP Request failed with code %d", res.StatusCode)
	}
	
	var isr indexStatsResponse
	if err := json.NewDecoder(res.Body).Decode(&isr); err != nil {
		i.jsonParseFailures.Inc()
		return indexStatsResponse{}, err
	}
	
	return isr, nil
}

// fetchClusterInfo 获取集群信息
func (i *Indices) fetchClusterInfo() (string, error) {
	u, err := url.Parse(i.esURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse ES URL: %s", err)
	}
	
	// 构建URL路径
	u.Path = path.Join(u.Path, "/")
	
	logrus.Debugf("Fetching cluster info from %s", u.String())
	
	res, err := i.client.Get(u.String())
	if err != nil {
		return "", fmt.Errorf("failed to get cluster info from %s: %s", u.String(), err)
	}
	
	defer func() {
		err = res.Body.Close()
		if err != nil {
			logrus.Warnf("failed to close http.Client: %s", err)
		}
	}()
	
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP Request failed with code %d", res.StatusCode)
	}
	
	var response struct {
		ClusterName string `json:"cluster_name"`
	}
	
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		i.jsonParseFailures.Inc()
		return "", err
	}
	
	return response.ClusterName, nil
}

// Collect 实现指标收集
func (i *Indices) Collect(ch chan<- prometheus.Metric) {
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
	
	// 获取集群名称
	i.mu.Lock()
	defer i.mu.Unlock()
	
	clusterName, err := i.fetchClusterInfo()
	if err != nil {
		logrus.Warnf("Failed to fetch cluster info: %s", err)
	} else {
		i.clusterName = clusterName
	}
	
	// 获取索引统计信息
	indexStats, err := i.fetchAndDecodeIndexStats()
	if err != nil {
		logrus.Warnf("Failed to fetch and decode index stats: %s", err)
		return
	}
	
	// 收集每个索引的指标
	for indexName, indexData := range indexStats.Indices {
		// 文档指标
		i.docsPrimary.collect(ch, float64(indexData.Primaries.Docs.Count), []string{indexName, i.clusterName})
		i.docsTotal.collect(ch, float64(indexData.Total.Docs.Count), []string{indexName, i.clusterName})
		i.deletedDocsPrimary.collect(ch, float64(indexData.Primaries.Docs.Deleted), []string{indexName, i.clusterName})
		i.deletedDocsTotal.collect(ch, float64(indexData.Total.Docs.Deleted), []string{indexName, i.clusterName})
		
		// 存储指标
		i.storeSizeBytesPrimary.collect(ch, float64(indexData.Primaries.Store.SizeInBytes), []string{indexName, i.clusterName})
		i.storeSizeBytesTotal.collect(ch, float64(indexData.Total.Store.SizeInBytes), []string{indexName, i.clusterName})
		
		// 段指标
		i.segmentCountPrimary.collect(ch, float64(indexData.Primaries.Segments.Count), []string{indexName, i.clusterName})
		i.segmentCountTotal.collect(ch, float64(indexData.Total.Segments.Count), []string{indexName, i.clusterName})
		i.segmentMemoryBytesPrimary.collect(ch, float64(indexData.Primaries.Segments.MemoryInBytes), []string{indexName, i.clusterName})
		i.segmentMemoryBytesTotal.collect(ch, float64(indexData.Total.Segments.MemoryInBytes), []string{indexName, i.clusterName})
		
		// 段详细指标
		i.segmentTermsMemoryPrimary.collect(ch, float64(indexData.Primaries.Segments.TermsMemoryInBytes), []string{indexName, i.clusterName})
		i.segmentTermsMemoryTotal.collect(ch, float64(indexData.Total.Segments.TermsMemoryInBytes), []string{indexName, i.clusterName})
		i.segmentStoredFieldsMemoryPrimary.collect(ch, float64(indexData.Primaries.Segments.StoredFieldsMemoryInBytes), []string{indexName, i.clusterName})
		i.segmentStoredFieldsMemoryTotal.collect(ch, float64(indexData.Total.Segments.StoredFieldsMemoryInBytes), []string{indexName, i.clusterName})
		i.segmentTermVectorsMemoryPrimary.collect(ch, float64(indexData.Primaries.Segments.TermVectorsMemoryInBytes), []string{indexName, i.clusterName})
		i.segmentTermVectorsMemoryTotal.collect(ch, float64(indexData.Total.Segments.TermVectorsMemoryInBytes), []string{indexName, i.clusterName})
		i.segmentNormsMemoryPrimary.collect(ch, float64(indexData.Primaries.Segments.NormsMemoryInBytes), []string{indexName, i.clusterName})
		i.segmentNormsMemoryTotal.collect(ch, float64(indexData.Total.Segments.NormsMemoryInBytes), []string{indexName, i.clusterName})
		i.segmentPointsMemoryPrimary.collect(ch, float64(indexData.Primaries.Segments.PointsMemoryInBytes), []string{indexName, i.clusterName})
		i.segmentPointsMemoryTotal.collect(ch, float64(indexData.Total.Segments.PointsMemoryInBytes), []string{indexName, i.clusterName})
		i.segmentDocValuesMemoryPrimary.collect(ch, float64(indexData.Primaries.Segments.DocValuesMemoryInBytes), []string{indexName, i.clusterName})
		i.segmentDocValuesMemoryTotal.collect(ch, float64(indexData.Total.Segments.DocValuesMemoryInBytes), []string{indexName, i.clusterName})
		i.segmentIndexWriterMemoryPrimary.collect(ch, float64(indexData.Primaries.Segments.IndexWriterMemoryInBytes), []string{indexName, i.clusterName})
		i.segmentIndexWriterMemoryTotal.collect(ch, float64(indexData.Total.Segments.IndexWriterMemoryInBytes), []string{indexName, i.clusterName})
		i.segmentVersionMapMemoryPrimary.collect(ch, float64(indexData.Primaries.Segments.VersionMapMemoryInBytes), []string{indexName, i.clusterName})
		i.segmentVersionMapMemoryTotal.collect(ch, float64(indexData.Total.Segments.VersionMapMemoryInBytes), []string{indexName, i.clusterName})
		i.segmentFixedBitSetMemoryPrimary.collect(ch, float64(indexData.Primaries.Segments.FixedBitSetMemoryInBytes), []string{indexName, i.clusterName})
		i.segmentFixedBitSetMemoryTotal.collect(ch, float64(indexData.Total.Segments.FixedBitSetMemoryInBytes), []string{indexName, i.clusterName})
		
		// 完成指标
		i.completionSizeBytesPrimary.collect(ch, float64(indexData.Primaries.Completion.SizeInBytes), []string{indexName, i.clusterName})
		i.completionSizeBytesTotal.collect(ch, float64(indexData.Total.Completion.SizeInBytes), []string{indexName, i.clusterName})
		
		// 缓存指标
		i.fielddataMemoySizeBytes.collect(ch, float64(indexData.Total.Fielddata.MemorySizeInBytes), []string{indexName, i.clusterName})
		i.fielddataEvictions.collect(ch, float64(indexData.Total.Fielddata.Evictions), []string{indexName, i.clusterName})
		i.queryCacheMemoySizeBytes.collect(ch, float64(indexData.Total.QueryCache.MemorySizeInBytes), []string{indexName, i.clusterName})
		i.queryCacheEvictions.collect(ch, float64(indexData.Total.QueryCache.Evictions), []string{indexName, i.clusterName})
		i.queryCacheHitCount.collect(ch, float64(indexData.Total.QueryCache.HitCount), []string{indexName, i.clusterName})
		i.queryCacheMissCount.collect(ch, float64(indexData.Total.QueryCache.MissCount), []string{indexName, i.clusterName})
		i.queryCacheTotalCount.collect(ch, float64(indexData.Total.QueryCache.TotalCount), []string{indexName, i.clusterName})
		
		// 索引操作指标
		i.indexingIndexTotal.collect(ch, float64(indexData.Total.Indexing.IndexTotal), []string{indexName, i.clusterName})
		i.indexingIndexTimeSeconds.collect(ch, float64(indexData.Total.Indexing.IndexTimeInMillis)/1000.0, []string{indexName, i.clusterName})
		i.indexingIndexCurrent.collect(ch, float64(indexData.Total.Indexing.IndexCurrent), []string{indexName, i.clusterName})
		i.indexingDeleteTotal.collect(ch, float64(indexData.Total.Indexing.DeleteTotal), []string{indexName, i.clusterName})
		i.indexingDeleteTimeSeconds.collect(ch, float64(indexData.Total.Indexing.DeleteTimeInMillis)/1000.0, []string{indexName, i.clusterName})
		i.indexingDeleteCurrent.collect(ch, float64(indexData.Total.Indexing.DeleteCurrent), []string{indexName, i.clusterName})
		
		// 获取指标
		i.getTotal.collect(ch, float64(indexData.Total.Get.Total), []string{indexName, i.clusterName})
		i.getTimeSeconds.collect(ch, float64(indexData.Total.Get.TimeInMillis)/1000.0, []string{indexName, i.clusterName})
		i.getExistsTotal.collect(ch, float64(indexData.Total.Get.ExistsTotal), []string{indexName, i.clusterName})
		i.getExistsTimeSeconds.collect(ch, float64(indexData.Total.Get.ExistsTimeInMillis)/1000.0, []string{indexName, i.clusterName})
		i.getMissingTotal.collect(ch, float64(indexData.Total.Get.MissingTotal), []string{indexName, i.clusterName})
		i.getMissingTimeSeconds.collect(ch, float64(indexData.Total.Get.MissingTimeInMillis)/1000.0, []string{indexName, i.clusterName})
		
		// 搜索指标
		i.searchQueryTime.collect(ch, float64(indexData.Total.Search.QueryTimeInMillis)/1000.0, []string{indexName, i.clusterName})
		i.searchQueryTotal.collect(ch, float64(indexData.Total.Search.QueryTotal), []string{indexName, i.clusterName})
		i.searchFetchTotal.collect(ch, float64(indexData.Total.Search.FetchTotal), []string{indexName, i.clusterName})
		i.searchFetchTimeSeconds.collect(ch, float64(indexData.Total.Search.FetchTimeInMillis)/1000.0, []string{indexName, i.clusterName})
		i.searchScrollTotal.collect(ch, float64(indexData.Total.Search.ScrollTotal), []string{indexName, i.clusterName})
		i.searchScrollTimeSeconds.collect(ch, float64(indexData.Total.Search.ScrollTimeInMillis)/1000.0, []string{indexName, i.clusterName})
	}
} 

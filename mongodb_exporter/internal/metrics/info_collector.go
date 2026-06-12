package metrics

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/bson"
)

// MongoDBInfocollector 负责采集 MongoDB serverStatus 相关指标
type MongoDBInfocollector struct {
	pool         *MongoDBClientPool
	instanceName string
	instanceURI  string
	metrics      *mongodbInfoMetrics
}

type mongodbInfoMetrics struct {
	upMetric            *baseMetrics // mongodb_up
	versionMetric       *baseMetrics // mongodb_version
	uptimeSecondsMetric *baseMetrics // mongodb_uptime_seconds
	processMetric       *baseMetrics // mongodb_process
	storageEngineMetric *baseMetrics // mongodb_storage_engine

	// 内存相关
	memoryResidentBytesMetric   *baseMetrics // mongodb_memory_resident_bytes
	memoryVirtualBytesMetric    *baseMetrics // mongodb_memory_virtual_bytes
	memoryMappedBytesMetric     *baseMetrics // mongodb_memory_mapped_bytes
	memoryPageFaultsTotalMetric *baseMetrics // mongodb_memory_page_faults_total

	// 连接相关
	connectionsCurrentMetric      *baseMetrics // mongodb_connections_current
	connectionsAvailableMetric    *baseMetrics // mongodb_connections_available
	connectionsTotalCreatedMetric *baseMetrics // mongodb_connections_total_created

	// 操作计数器
	opcountersInsertMetric  *baseMetrics // mongodb_opcounters_insert_total
	opcountersQueryMetric   *baseMetrics // mongodb_opcounters_query_total
	opcountersUpdateMetric  *baseMetrics // mongodb_opcounters_update_total
	opcountersDeleteMetric  *baseMetrics // mongodb_opcounters_delete_total
	opcountersGetmoreMetric *baseMetrics // mongodb_opcounters_getmore_total
	opcountersCommandMetric *baseMetrics // mongodb_opcounters_command_total

	// 全局锁
	globalLockCurrentQueueTotalMetric  *baseMetrics // mongodb_globalLock_currentQueue_total
	globalLockActiveClientsTotalMetric *baseMetrics // mongodb_globalLock_activeClients_total
	globalLockTimeLockedReadMetric     *baseMetrics // mongodb_globalLock_timeLockedMicros_read_total
	globalLockTimeLockedWriteMetric    *baseMetrics // mongodb_globalLock_timeLockedMicros_write_total
}

func newMongoDBInfoMetrics() *mongodbInfoMetrics {
	return &mongodbInfoMetrics{
		// Global
		upMetric: NewMetrics(
			"mongodb_up",
			"Whether MongoDB is up (1) or down (0).",
			[]string{"instance", "uri"},
		),
		versionMetric: NewMetrics(
			"mongodb_version",
			"The version of the MongoDB instance.",
			[]string{"instance", "uri", "version"},
		),
		uptimeSecondsMetric: NewMetrics(
			"mongodb_uptime_seconds",
			"The uptime of the MongoDB instance in seconds.",
			[]string{"instance", "uri"},
		),
		processMetric: NewMetrics(
			"mongodb_process",
			"The process type (mongod/mongos/other).",
			[]string{"instance", "uri", "process"},
		),
		storageEngineMetric: NewMetrics(
			"mongodb_storage_engine",
			"The storage engine used by MongoDB (e.g., wiredTiger, mmapv1).",
			[]string{"instance", "uri", "storage_engine"},
		),

		// Memory
		memoryResidentBytesMetric: NewMetrics(
			"mongodb_memory_resident_bytes",
			"The amount of physical memory used by MongoDB.",
			[]string{"instance", "uri"},
		),
		memoryVirtualBytesMetric: NewMetrics(
			"mongodb_memory_virtual_bytes",
			"The amount of virtual memory used by MongoDB.",
			[]string{"instance", "uri"},
		),
		memoryMappedBytesMetric: NewMetrics(
			"mongodb_memory_mapped_bytes",
			"The size of mapped files.",
			[]string{"instance", "uri"},
		),
		memoryPageFaultsTotalMetric: NewMetrics(
			"mongodb_memory_page_faults_total",
			"The total number of page faults.",
			[]string{"instance", "uri"},
		),

		// Connections
		connectionsCurrentMetric: NewMetrics(
			"mongodb_connections_current",
			"The current number of active connections.",
			[]string{"instance", "uri"},
		),
		connectionsAvailableMetric: NewMetrics(
			"mongodb_connections_available",
			"The number of available connections.",
			[]string{"instance", "uri"},
		),
		connectionsTotalCreatedMetric: NewMetrics(
			"mongodb_connections_total_created",
			"The total number of connections created.",
			[]string{"instance", "uri"},
		),

		// OpCounters
		opcountersInsertMetric: NewMetrics(
			"mongodb_opcounters_insert_total",
			"The total number of insert operations.",
			[]string{"instance", "uri"},
		),
		opcountersQueryMetric: NewMetrics(
			"mongodb_opcounters_query_total",
			"The total number of query operations.",
			[]string{"instance", "uri"},
		),
		opcountersUpdateMetric: NewMetrics(
			"mongodb_opcounters_update_total",
			"The total number of update operations.",
			[]string{"instance", "uri"},
		),
		opcountersDeleteMetric: NewMetrics(
			"mongodb_opcounters_delete_total",
			"The total number of delete operations.",
			[]string{"instance", "uri"},
		),
		opcountersGetmoreMetric: NewMetrics(
			"mongodb_opcounters_getmore_total",
			"The total number of getmore operations.",
			[]string{"instance", "uri"},
		),
		opcountersCommandMetric: NewMetrics(
			"mongodb_opcounters_command_total",
			"The total number of command operations.",
			[]string{"instance", "uri"},
		),

		// Global Lock
		globalLockCurrentQueueTotalMetric: NewMetrics(
			"mongodb_globalLock_currentQueue_total",
			"The current number of operations waiting for a lock.",
			[]string{"instance", "uri"},
		),
		globalLockActiveClientsTotalMetric: NewMetrics(
			"mongodb_globalLock_activeClients_total",
			"The number of active clients (with operations in progress or queued).",
			[]string{"instance", "uri"},
		),
		globalLockTimeLockedReadMetric: NewMetrics(
			"mongodb_globalLock_timeLockedMicros_read_total",
			"The cumulative time read locks have been held (microseconds).",
			[]string{"instance", "uri"},
		),
		globalLockTimeLockedWriteMetric: NewMetrics(
			"mongodb_globalLock_timeLockedMicros_write_total",
			"The cumulative time write locks have been held (microseconds).",
			[]string{"instance", "uri"},
		),
	}
}

// NewMongoDBInfocollector 创建一个新的 MongoDB Info collector
func NewMongoDBInfocollector(pool *MongoDBClientPool, instanceName, instanceURI string) *MongoDBInfocollector {
	return &MongoDBInfocollector{
		pool:         pool,
		instanceName: instanceName,
		instanceURI:  instanceURI,
		metrics:      newMongoDBInfoMetrics(),
	}
}

// Describe implements Prometheus collector interface

// collect implements Prometheus collector interface
func (c *MongoDBInfocollector) collect(ch chan<- prometheus.Metric) {
	client, _ := c.pool.GetClient()
	ctx := context.Background()
	adminDB := client.Database("admin")

	cmd := bson.D{{Key: "serverStatus", Value: 1}}
	var result bson.M
	err := adminDB.RunCommand(ctx, cmd).Decode(&result)
	var up float64 = 1
	if err != nil {
		up = 0
	}
	labels := []string{c.instanceName, c.instanceURI}
	c.metrics.upMetric.collect(
		ch,
		up,
		labels,
	)

	if up == 0 {
		return
	}

	// 版本号
	if version, ok := result["version"].(string); ok {
		c.metrics.versionMetric.collect(
			ch,
			1,
			append(labels, version),
		)
	}

	// 启动时间
	if uptime, ok := result["uptime"].(int64); ok {
		c.metrics.uptimeSecondsMetric.collect(
			ch,
			float64(uptime),
			labels,
		)
	}

	// Process 类型
	if process, ok := result["process"].(string); ok {
		c.metrics.processMetric.collect(
			ch,
			1,
			append(labels, process),
		)
	}

	// 存储引擎
	if storageEngine, ok := result["storageEngine"].(bson.M); ok {
		if name, ok := storageEngine["name"].(string); ok {
			c.metrics.storageEngineMetric.collect(
				ch,
				1,
				append(labels, name),
			)
		}
	}

	// 内存
	if mem, ok := result["mem"].(bson.M); ok {
		if resident, ok := mem["resident"].(int32); ok {
			c.metrics.memoryResidentBytesMetric.collect(
				ch,
				float64(resident)*1024*1024,
				labels,
			) // MB -> Bytes
		}
		if virtual, ok := mem["virtual"].(int32); ok {
			c.metrics.memoryVirtualBytesMetric.collect(
				ch,
				float64(virtual)*1024*1024,
				labels,
			)
		}
		if mapped, ok := mem["mapped"].(int32); ok {
			c.metrics.memoryMappedBytesMetric.collect(
				ch,
				float64(mapped)*1024*1024,
				labels,
			)
		}
		if extraInfo, ok := result["extra_info"].(bson.M); ok {
			if pageFaults, ok := extraInfo["page_faults"].(int32); ok {
				c.metrics.memoryPageFaultsTotalMetric.collect(
					ch,
					float64(pageFaults),
					labels,
				)
			}
		}
	}

	// 连接数
	if conn, ok := result["connections"].(bson.M); ok {
		if current, ok := conn["current"].(int32); ok {
			c.metrics.connectionsCurrentMetric.collect(
				ch,
				float64(current),
				labels,
			)
		}
		if available, ok := conn["available"].(int32); ok {
			c.metrics.connectionsAvailableMetric.collect(
				ch,
				float64(available),
				labels,
			)
		}
		if totalCreated, ok := conn["totalCreated"].(int64); ok {
			c.metrics.connectionsTotalCreatedMetric.collect(
				ch,
				float64(totalCreated),
				labels,
			)
		}
	}

	// 操作计数器
	if opcounters, ok := result["opcounters"].(bson.M); ok {
		c.collectOpCounter(
			ch,
			opcounters,
			"insert",
			c.metrics.opcountersInsertMetric,
		)
		c.collectOpCounter(
			ch,
			opcounters,
			"query",
			c.metrics.opcountersQueryMetric,
		)
		c.collectOpCounter(
			ch,
			opcounters,
			"update",
			c.metrics.opcountersUpdateMetric,
		)
		c.collectOpCounter(
			ch,
			opcounters,
			"delete",
			c.metrics.opcountersDeleteMetric,
		)
		c.collectOpCounter(
			ch,
			opcounters,
			"getmore",
			c.metrics.opcountersGetmoreMetric,
		)
		c.collectOpCounter(
			ch,
			opcounters,
			"command",
			c.metrics.opcountersCommandMetric,
		)
	}

	// 锁信息
	if globalLock, ok := result["globalLock"].(bson.M); ok {
		if queues, ok := globalLock["currentQueue"].(bson.M); ok {
			if total, ok := queues["total"].(int32); ok {
				c.metrics.globalLockCurrentQueueTotalMetric.collect(
					ch,
					float64(total),
					labels,
				)
			}
		}
		if activeClients, ok := globalLock["activeClients"].(bson.M); ok {
			if total, ok := activeClients["total"].(int32); ok {
				c.metrics.globalLockActiveClientsTotalMetric.collect(
					ch,
					float64(total),
					labels,
				)
			}
		}
		if timeLockedMicros, ok := globalLock["timeLockedMicros"].(bson.M); ok {
			if read, ok := timeLockedMicros["read"].(int64); ok {
				c.metrics.globalLockTimeLockedReadMetric.collect(
					ch,
					float64(read),
					labels,
				)
			}
			if write, ok := timeLockedMicros["write"].(int64); ok {
				c.metrics.globalLockTimeLockedWriteMetric.collect(
					ch,
					float64(write),
					labels,
				)
			}
		}
	}
}

// collectOpCounter 是一个辅助函数，用于采集 opcounter 指标
func (c *MongoDBInfocollector) collectOpCounter(ch chan<- prometheus.Metric, opcounters bson.M, op string, metric *baseMetrics) {
	if val, ok := opcounters[op].(int64); ok {
		metric.collect(
			ch,
			float64(val),
			[]string{c.instanceName, c.instanceURI},
		)
	}
}

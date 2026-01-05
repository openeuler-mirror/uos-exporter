package metrics

import (
	"context"
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDBShardServerCollector 负责采集 MongoDB 分片集群中各 Shard Server 的状态和元数据
type MongoDBShardServerCollector struct {
	pool         *MongoDBClientPool
	instanceName string
	instanceURI  string
	metrics      *shardServerMetrics
}

type shardServerMetrics struct {
	shardOnlineMetric        *baseMetrics // 是否在线
	shardVersionMetric       *baseMetrics // 版本号
	shardHostMetric          *baseMetrics // 主机地址
	shardTypeMetric          *baseMetrics // 类型（shard, config, mongos）
	shardLastHeartbeatMetric *baseMetrics // 最后一次心跳时间戳
	shardDrainingMetric      *baseMetrics // 是否正在迁移 chunk
	shardTotalSizeMetric     *baseMetrics // 分片总大小（MB）
	shardMaxSizeMetric       *baseMetrics // 分片最大大小（MB）
	shardCountMetric         *baseMetrics // 分片上的集合数量
	shardUptimeMetric        *baseMetrics // 运行时间（秒）
}

func newShardServerMetrics() *shardServerMetrics {
	return &shardServerMetrics{
		shardOnlineMetric: NewMetrics(
			"mongodb_sharding_shard_online",
			"Gauge indicating if the shard is online (1) or not (0).",
			[]string{"instance", "uri", "shard"},
		),
		shardVersionMetric: NewMetrics(
			"mongodb_sharding_shard_version",
			"The version of the shard server.",
			[]string{"instance", "uri", "shard"},
		),
		shardHostMetric: NewMetrics(
			"mongodb_sharding_shard_host",
			"The host address of the shard server.",
			[]string{"instance", "uri", "shard"},
		),
		shardTypeMetric: NewMetrics(
			"mongodb_sharding_shard_type",
			"The type of the shard (shard, config, mongos).",
			[]string{"instance", "uri", "shard"},
		),
		shardLastHeartbeatMetric: NewMetrics(
			"mongodb_sharding_shard_last_heartbeat_timestamp",
			"The timestamp of the last heartbeat from the shard server.",
			[]string{"instance", "uri", "shard"},
		),
		shardDrainingMetric: NewMetrics(
			"mongodb_sharding_shard_draining",
			"Gauge indicating if a shard is draining chunks (1) or not (0).",
			[]string{"instance", "uri", "shard"},
		),
		shardTotalSizeMetric: NewMetrics(
			"mongodb_sharding_shard_total_size_bytes",
			"The total size of data stored on this shard in bytes.",
			[]string{"instance", "uri", "shard"},
		),
		shardMaxSizeMetric: NewMetrics(
			"mongodb_sharding_shard_max_size_bytes",
			"The maximum size configured for this shard.",
			[]string{"instance", "uri", "shard"},
		),
		shardCountMetric: NewMetrics(
			"mongodb_sharding_shard_collections_count",
			"The number of collections on this shard.",
			[]string{"instance", "uri", "shard"},
		),
		shardUptimeMetric: NewMetrics(
			"mongodb_sharding_shard_uptime_seconds",
			"The uptime of the shard server in seconds.",
			[]string{"instance", "uri", "shard"},
		),
	}
}

// NewMongoDBShardServerCollector 创建一个新的 Shard Server Collector
func NewMongoDBShardServerCollector(pool *MongoDBClientPool, instanceName, instanceURI string) *MongoDBShardServerCollector {
	return &MongoDBShardServerCollector{
		pool:         pool,
		instanceName: instanceName,
		instanceURI:  instanceURI,
		metrics:      newShardServerMetrics(),
	}
}

// Describe implements Prometheus Collector interface

// Collect implements Prometheus Collector interface
func (c *MongoDBShardServerCollector) collect(ch chan<- prometheus.Metric) {
	client, _ := c.pool.GetClient()
	ctx := context.Background()
	configDB := client.Database("config")

	// 查询 config.shards 获取所有分片服务器
	shardColl := configDB.Collection("shards")
	cursor, err := shardColl.Find(ctx, bson.D{})
	if err != nil {
		fmt.Printf("Failed to query config.shards: %v\n", err)
		return
	}
	defer cursor.Close(ctx)

	labelsBase := []string{c.instanceName, c.instanceURI}

	for cursor.Next(ctx) {
		var shardDoc bson.M
		err := cursor.Decode(&shardDoc)
		if err != nil {
			fmt.Printf("Failed to decode shard document: %v\n", err)
			continue
		}

		// 提取字段
		name, ok1 := shardDoc["_id"].(string)
		host, ok2 := shardDoc["host"].(string)
		draining, _ := shardDoc["draining"].(bool)
		totalSize, _ := shardDoc["totalSize"].(int64)
		maxSize, _ := shardDoc["maxSize"].(int64)

		if !ok1 || !ok2 {
			continue
		}

		labels := append(labelsBase, name)

		// 上报分片在线状态（假设存在即在线）
		c.metrics.shardOnlineMetric.collect(
			ch,
			1,
			labels,
		)

		// 上报主机地址
		c.metrics.shardHostMetric.collect(
			ch,
			1,
			append(labels, host),
		)

		// 分片类型（默认是 shard）
		typ := "shard"
		if strings.HasPrefix(name, "config") {
			typ = "config"
		} else if strings.HasPrefix(name, "mongos") {
			typ = "mongos"
		}
		c.metrics.shardTypeMetric.collect(
			ch,
			1,
			append(labels, typ),
		)

		// 总大小（字节）
		c.metrics.shardTotalSizeMetric.collect(
			ch,
			float64(totalSize*1024*1024),
			labels,
		) // MB -> Bytes

		// 最大大小（字节）
		if maxSize > 0 {
			c.metrics.shardMaxSizeMetric.collect(
				ch,
				float64(maxSize*1024*1024),
				labels,
			)
		}

		// draining 状态
		var drainVal float64 = 0
		if draining {
			drainVal = 1
		}
		c.metrics.shardDrainingMetric.collect(
			ch,
			drainVal,
			labels,
		)

		// 如果是主节点或从节点，可以连接 shard 获取更详细状态
		if typ == "shard" || typ == "config" {
			// 解析 host 字段获取实际地址
			parts := strings.Split(host, "/")
			if len(parts) >= 2 {
				hostAddr := parts[len(parts)-1]
				shardClient, err := connectToShard(hostAddr)
				if err == nil {
					defer shardClient.Disconnect(context.Background())

					adminDB := shardClient.Database("admin")
					serverStatus := bson.M{}
					err = adminDB.RunCommand(context.Background(), bson.D{{Key: "serverStatus", Value: 1}}).Decode(&serverStatus)
					if err == nil {
						version, _ := serverStatus["version"].(string)
						uptime, _ := serverStatus["uptime"].(int64)

						// 上报版本号（作为 info metric）
						c.metrics.shardVersionMetric.collect(
							ch,
							1,
							append(labels, version),
						)

						// 上报运行时间
						c.metrics.shardUptimeMetric.collect(
							ch,
							float64(uptime),
							labels,
						)
					}
				}
			}
		}
	}
}

// connectToShard 建立与分片服务器的连接
func connectToShard(uri string) (*mongo.Client, error) {
	client, err := mongo.Connect(
		context.Background(),
		options.Client().ApplyURI("mongodb://"+uri),
	)
	if err != nil {
		return nil, err
	}

	err = client.Ping(context.Background(), nil)
	if err != nil {
		return nil, err
	}

	return client, nil
}
// Part 2 commit for mongodb_exporter/internal/metrics/shard_server_collector.go

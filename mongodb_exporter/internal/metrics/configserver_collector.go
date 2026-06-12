package metrics

import (
	"context"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/bson"
)

// MongoDBConfigServerCollector 负责采集 MongoDB Config Server 中的元数据信息
type MongoDBConfigServerCollector struct {
	pool         *MongoDBClientPool
	instanceName string
	instanceURI  string
	metrics      *configServerMetrics
}

type configServerMetrics struct {
	configCollectionsCountMetric   *baseMetrics // 启用分片的集合数量
	configChunksTotalMetric        *baseMetrics // chunk 总数
	configShardsCountMetric        *baseMetrics // 分片总数
	configDatabasesCountMetric     *baseMetrics // 分片数据库数量
	configTagsCountMetric          *baseMetrics // 标签（tag）数量
	configChunksByCollectionMetric *baseMetrics // 每个集合的 chunk 数量
	configCollectionsByShardMetric *baseMetrics // 每个分片上的集合数量
	configChunksByShardMetric      *baseMetrics // 每个分片上的 chunk 数量
	configTagsByShardMetric        *baseMetrics // 每个分片上的 tag 数量
}

func newConfigServerMetrics() *configServerMetrics {
	return &configServerMetrics{
		configCollectionsCountMetric: NewMetrics(
			"mongodb_sharding_config_collections_count",
			"The number of collections that have been sharded.",
			[]string{"instance", "uri"},
		),
		configChunksTotalMetric: NewMetrics(
			"mongodb_sharding_config_chunks_total",
			"The total number of chunks stored in the config server.",
			[]string{"instance", "uri"},
		),
		configShardsCountMetric: NewMetrics(
			"mongodb_sharding_config_shards_count",
			"The number of shards registered in the cluster.",
			[]string{"instance", "uri"},
		),
		configDatabasesCountMetric: NewMetrics(
			"mongodb_sharding_config_databases_count",
			"The number of databases enabled for sharding.",
			[]string{"instance", "uri"},
		),
		configTagsCountMetric: NewMetrics(
			"mongodb_sharding_config_tags_count",
			"The number of tags defined in the sharding configuration.",
			[]string{"instance", "uri"},
		),
		configChunksByCollectionMetric: NewMetrics(
			"mongodb_sharding_config_chunks_by_collection",
			"The number of chunks per collection.",
			[]string{"instance", "uri", "db", "collection"},
		),
		configCollectionsByShardMetric: NewMetrics(
			"mongodb_sharding_config_collections_by_shard",
			"The number of sharded collections on each shard.",
			[]string{"instance", "uri", "shard"},
		),
		configChunksByShardMetric: NewMetrics(
			"mongodb_sharding_config_chunks_by_shard",
			"The number of chunks per shard.",
			[]string{"instance", "uri", "shard"},
		),
		configTagsByShardMetric: NewMetrics(
			"mongodb_sharding_config_tags_by_shard",
			"The number of tags associated with a shard.",
			[]string{"instance", "uri", "shard"},
		),
	}
}

// NewMongoDBConfigServerCollector 创建一个新的 Config Server Collector
func NewMongoDBConfigServerCollector(pool *MongoDBClientPool, instanceName, instanceURI string) *MongoDBConfigServerCollector {
	return &MongoDBConfigServerCollector{
		pool:         pool,
		instanceName: instanceName,
		instanceURI:  instanceURI,
		metrics:      newConfigServerMetrics(),
	}
}

// Describe implements Prometheus Collector interface

// Collect implements Prometheus Collector interface
func (c *MongoDBConfigServerCollector) collect(ch chan<- prometheus.Metric) {
	client, _ := c.pool.GetClient()
	ctx := context.Background()
	configDB := client.Database("config")

	labelsBase := []string{c.instanceName, c.instanceURI}

	// 1. config.collections：启用分片的集合
	collColl := configDB.Collection("collections")
	collCursor, err := collColl.Find(ctx, bson.D{})
	if err == nil {
		defer collCursor.Close(ctx)
		var collectionCount int64 = 0
		collectionShardMap := make(map[string]map[string]bool)

		for collCursor.Next(ctx) {
			var doc bson.M
			err := collCursor.Decode(&doc)
			if err != nil {
				continue
			}

			ns, ok := doc["_id"].(string)
			if !ok {
				continue
			}

			parts := strings.SplitN(ns, ".", 2)
			if len(parts) < 2 {
				continue
			}
			// dbName := parts[0]
			// collName := parts[1]

			collectionCount++

			if collectionShardMap[ns] == nil {
				collectionShardMap[ns] = make(map[string]bool)
			}
		}

		// 上报启用分片的集合总数
		c.metrics.configCollectionsCountMetric.collect(
			ch,
			float64(collectionCount),
			labelsBase,
		)
	}

	// 2. config.chunks：chunk 分布
	chunkColl := configDB.Collection("chunks")
	chunkCursor, err := chunkColl.Find(ctx, bson.D{})
	if err == nil {
		defer chunkCursor.Close(ctx)
		var chunkCount int64 = 0
		shardChunkCounts := make(map[string]int64)

		for chunkCursor.Next(ctx) {
			var doc bson.M
			err := chunkCursor.Decode(&doc)
			if err != nil {
				continue
			}

			ns, _ := doc["ns"].(string)
			shard, _ := doc["shard"].(string)

			chunkCount++
			shardChunkCounts[shard]++

			// 按集合统计 chunk 数量
			parts := strings.SplitN(ns, ".", 2)
			if len(parts) >= 2 {
				dbName := parts[0]
				collName := parts[1]
				c.metrics.configChunksByCollectionMetric.collect(
					ch,
					1,
					append(labelsBase, dbName, collName),
				)
			}
		}

		// 上报总 chunk 数量
		c.metrics.configChunksTotalMetric.collect(
			ch,
			float64(chunkCount),
			labelsBase,
		)

		// 按分片上报 chunk 数量
		for shard, count := range shardChunkCounts {
			c.metrics.configChunksByShardMetric.collect(
				ch,
				float64(count),
				append(labelsBase, shard),
			)
		}
	}

	// 3. config.shards：分片数量
	shardColl := configDB.Collection("shards")
	shardCursor, err := shardColl.Find(ctx, bson.D{})
	if err == nil {
		defer shardCursor.Close(ctx)
		var shardCount int64 = 0
		shardTagCounts := make(map[string]int64)

		for shardCursor.Next(ctx) {
			var doc bson.M
			err := shardCursor.Decode(&doc)
			if err != nil {
				continue
			}

			name, ok := doc["_id"].(string)
			if !ok {
				continue
			}

			shardCount++
			c.metrics.configCollectionsByShardMetric.collect(
				ch,
				float64(0),
				append(labelsBase, name),
			) // 可通过其他方式补充具体值
			c.metrics.configTagsByShardMetric.collect(
				ch,
				float64(shardTagCounts[name]),
				append(labelsBase, name),
			)
		}

		// 上报分片总数
		c.metrics.configShardsCountMetric.collect(
			ch,
			float64(shardCount),
			labelsBase,
		)
	}

	// 4. config.databases：启用了分片的数据库
	dbColl := configDB.Collection("databases")
	dbCursor, err := dbColl.Find(ctx, bson.D{{Key: "partitioned", Value: true}})
	if err == nil {
		defer dbCursor.Close(ctx)
		var dbCount int64 = 0

		for dbCursor.Next(ctx) {
			var doc bson.M
			err := dbCursor.Decode(&doc)
			if err != nil {
				continue
			}

			_, ok := doc["_id"].(string)
			if !ok {
				continue
			}

			dbCount++
		}

		// 上报启用了分片的数据库数量
		c.metrics.configDatabasesCountMetric.collect(
			ch,
			float64(dbCount),
			labelsBase,
		)
	}

	// 5. config.tags：按分片统计标签数量（可选）
	tagColl := configDB.Collection("tags")
	tagCursor, err := tagColl.Find(ctx, bson.D{})
	if err == nil {
		defer tagColl.Database().Client().Disconnect(context.Background())

		for tagCursor.Next(ctx) {
			var doc bson.M
			err := tagCursor.Decode(&doc)
			if err != nil {
				continue
			}

			shard, ok := doc["shard"].(string)
			if !ok {
				continue
			}

			// 统计每个分片的 tag 数量
			c.metrics.configTagsByShardMetric.collect(
				ch,
				1,
				append(labelsBase, shard),
			)
		}
	}
}

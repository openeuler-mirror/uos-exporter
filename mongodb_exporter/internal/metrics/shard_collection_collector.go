package metrics

import (
	"context"
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/bson"
)

// MongoDBShardCollectionCollector 负责采集 MongoDB 中每个集合在各分片上的分布信息
type MongoDBShardCollectionCollector struct {
	pool         *MongoDBClientPool
	instanceName string
	instanceURI  string
	metrics      *shardCollectionMetrics
}

type shardCollectionMetrics struct {
	collectionShardedMetric            *baseMetrics // 是否启用分片
	collectionChunksByShardMetric      *baseMetrics // 每个集合在分片上的 chunk 数量
	collectionSizeByShardMetric        *baseMetrics // 4
	collectionTotalShardsMetric        *baseMetrics // 启用分片的集合数量
	collectionJumboChunksByShardMetric *baseMetrics // 每个集合在分片上的 jumbo chunk 数量
	shardCollectionsCountMetric        *baseMetrics // 分片上的集合数量
}

func newShardCollectionMetrics() *shardCollectionMetrics {
	return &shardCollectionMetrics{
		collectionShardedMetric: NewMetrics(
			"mongodb_sharding_collection_sharded",
			"Gauge indicating if a collection is sharded (1) or not (0).",
			[]string{"instance", "uri", "db", "collection"},
		),
		collectionChunksByShardMetric: NewMetrics(
			"mongodb_sharding_collection_chunks_by_shard",
			"The number of chunks per collection on each shard.",
			[]string{"instance", "uri", "db", "collection", "shard"},
		),
		collectionSizeByShardMetric: NewMetrics(
			"mongodb_sharding_collection_size_bytes",
			"The total size of data per collection on each shard.",
			[]string{"instance", "uri", "db", "collection", "shard"},
		),
		collectionTotalShardsMetric: NewMetrics(
			"mongodb_sharding_collection_total_shards",
			"The number of shards the collection is distributed across.",
			[]string{"instance", "uri", "db", "collection"},
		),
		collectionJumboChunksByShardMetric: NewMetrics(
			"mongodb_sharding_collection_jumbo_chunks_by_shard",
			"The number of jumbo chunks per collection on each shard.",
			[]string{"instance", "uri", "db", "collection", "shard"},
		),
		shardCollectionsCountMetric: NewMetrics(
			"mongodb_sharding_shard_collections_count",
			"The number of collections hosted by this shard.",
			[]string{"instance", "uri", "shard"},
		),
	}
}

// NewMongoDBShardCollectionCollector 创建一个新的 Shard Collection Collector
func NewMongoDBShardCollectionCollector(pool *MongoDBClientPool, instanceName, instanceURI string) *MongoDBShardCollectionCollector {
	return &MongoDBShardCollectionCollector{
		pool:         pool,
		instanceName: instanceName,
		instanceURI:  instanceURI,
		metrics:      newShardCollectionMetrics(),
	}
}

// Describe implements Prometheus Collector interface

// Collect implements Prometheus Collector interface
func (c *MongoDBShardCollectionCollector) collect(ch chan<- prometheus.Metric) {
	client, _ := c.pool.GetClient()
	ctx := context.Background()
	configDB := client.Database("config")

	// 获取所有启用分片的集合
	collColl := configDB.Collection("collections")
	cursor, err := collColl.Find(ctx, bson.D{})
	if err != nil {
		fmt.Printf("Failed to query config.collections: %v\n", err)
		return
	}
	defer cursor.Close(ctx)

	labelsBase := []string{c.instanceName, c.instanceURI}
	shardCollectionCount := make(map[string]int64) // 统计每个分片上的集合数

	for cursor.Next(ctx) {
		var collDoc bson.M
		err := cursor.Decode(&collDoc)
		if err != nil {
			continue
		}

		ns, ok := collDoc["_id"].(string)
		if !ok {
			continue
		}

		parts := strings.SplitN(ns, ".", 2)
		if len(parts) < 2 {
			continue
		}
		dbName := parts[0]
		collName := parts[1]

		// 上报集合是否启用分片
		c.metrics.collectionShardedMetric.collect(
			ch,
			1,
			append(labelsBase, dbName, collName),
		)

		// 查询 chunks 分布
		chunkColl := configDB.Collection("chunks")
		chunkCursor, _ := chunkColl.Find(ctx, bson.D{{Key: "ns", Value: ns}})
		if chunkCursor == nil {
			continue
		}
		defer chunkCursor.Close(ctx)

		chunkCounts := make(map[string]int64)
		jumboChunkCounts := make(map[string]int64)
		shardSet := make(map[string]bool)

		for chunkCursor.Next(ctx) {
			var chunkDoc bson.M
			err := chunkCursor.Decode(&chunkDoc)
			if err != nil {
				continue
			}

			shard, ok1 := chunkDoc["shard"].(string)
			isJumbo, ok2 := chunkDoc["jumbo"].(bool)

			if !ok1 {
				continue
			}

			chunkCounts[shard]++
			if ok2 && isJumbo {
				jumboChunkCounts[shard]++
			}
			shardSet[shard] = true
		}

		// 上报每个分片上的 chunk 数量和 jumbo chunk 数量
		for shard, count := range chunkCounts {
			labels := append(labelsBase, dbName, collName, shard)

			c.metrics.collectionChunksByShardMetric.collect(
				ch,
				float64(count),
				labels,
			)
			c.metrics.collectionJumboChunksByShardMetric.collect(
				ch,
				float64(jumboChunkCounts[shard]),
				labels,
			)

			// 记录该分片上的集合数
			shardCollectionCount[shard]++
		}

		// 上报集合分布在多少个分片上
		c.metrics.collectionTotalShardsMetric.collect(
			ch,
			float64(len(shardSet)),
			append(labelsBase, dbName, collName),
		)
	}

	// 上报每个分片上的集合总数
	for shard, count := range shardCollectionCount {
		c.metrics.shardCollectionsCountMetric.collect(
			ch,
			float64(count),
			append(labelsBase, shard),
		)
	}
}
// Part 2 commit for mongodb_exporter/internal/metrics/shard_collection_collector.go

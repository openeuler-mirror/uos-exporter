package metrics

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/bson"
)

// MongoDBAutoDiscoveryCollector 负责自动发现 MongoDB 中的数据库、集合、分片等元数据
type MongoDBAutoDiscoveryCollector struct {
	pool         *MongoDBClientPool
	instanceName string
	instanceURI  string
	metrics      *autoDiscoveryMetrics
}

type autoDiscoveryMetrics struct {
	databasesCountMetric        *baseMetrics // 数据库数量
	collectionsCountMetric      *baseMetrics // 集合数量
	shardsCountMetric           *baseMetrics // 分片总数
	jumboChunksCountMetric      *baseMetrics // Jumbo Chunk 总数
	collectionsShardedMetric    *baseMetrics // 启用分片的集合数
	zonesCountMetric            *baseMetrics // Zone 总数
	tagsCountMetric             *baseMetrics // Tag 总数
	collectionsByDBMetric       *baseMetrics // 每个 DB 下集合数量
	shardCollectionsCountMetric *baseMetrics // 每个分片上的集合数量
}

func newAutoDiscoveryMetrics() *autoDiscoveryMetrics {
	return &autoDiscoveryMetrics{
		databasesCountMetric: NewMetrics(
			"mongodb_auto_discovery_dbs_count",
			"The total number of databases discovered.",
			[]string{"instance", "uri"},
		),
		collectionsCountMetric: NewMetrics(
			"mongodb_auto_discovery_collections_count",
			"The total number of collections discovered.",
			[]string{"instance", "uri"},
		),
		shardsCountMetric: NewMetrics(
			"mongodb_sharding_auto_discovery_shards_count",
			"The total number of shards discovered.",
			[]string{"instance", "uri"},
		),
		jumboChunksCountMetric: NewMetrics(
			"mongodb_sharding_auto_discovery_jumbo_chunks_count",
			"The total number of jumbo chunks discovered.",
			[]string{"instance", "uri"},
		),
		collectionsShardedMetric: NewMetrics(
			"mongodb_sharding_auto_discovery_sharded_collections_count",
			"The number of sharded collections discovered.",
			[]string{"instance", "uri"},
		),
		zonesCountMetric: NewMetrics(
			"mongodb_sharding_auto_discovery_zones_count",
			"The number of zones defined in the cluster.",
			[]string{"instance", "uri"},
		),
		tagsCountMetric: NewMetrics(
			"mongodb_sharding_auto_discovery_tags_count",
			"The number of tags defined in the cluster.",
			[]string{"instance", "uri"},
		),
		collectionsByDBMetric: NewMetrics(
			"mongodb_auto_discovery_collections_by_db",
			"The number of collections per database.",
			[]string{"instance", "uri", "db"},
		),
		shardCollectionsCountMetric: NewMetrics(
			"mongodb_sharding_auto_discovery_shard_collections_count",
			"The number of collections on each shard.",
			[]string{"instance", "uri", "shard"},
		),
	}
}

// NewMongoDBAutoDiscoveryCollector 创建一个新的 Auto Discovery Collector
func NewMongoDBAutoDiscoveryCollector(pool *MongoDBClientPool, instanceName, instanceURI string) *MongoDBAutoDiscoveryCollector {
	return &MongoDBAutoDiscoveryCollector{
		pool:         pool,
		instanceName: instanceName,
		instanceURI:  instanceURI,
		metrics:      newAutoDiscoveryMetrics(),
	}
}

// Describe implements Prometheus Collector interface

// Collect implements Prometheus Collector interface
func (c *MongoDBAutoDiscoveryCollector) collect(ch chan<- prometheus.Metric) {
	client, _ := c.pool.GetClient()

	ctx := context.Background()
	// adminDB := client.Database("admin")
	configDB := client.Database("config")

	labelsBase := []string{c.instanceName, c.instanceURI}

	// -----------------------------
	// 1. 自动发现数据库数量
	// -----------------------------
	// dbNames, err := adminDB.ListDatabaseNames(ctx, bson.D{})
	dbNames, err := client.ListDatabaseNames(ctx, bson.D{})

	if err != nil {
		fmt.Printf("Failed to list databases: %v\n", err)
	} else {
		c.metrics.databasesCountMetric.collect(
			ch,
			float64(len(dbNames)),
			labelsBase,
		)

		// 按数据库统计集合数量
		for _, dbName := range dbNames {
			if dbName == "local" || dbName == "config" || dbName == "admin" {
				continue
			}

			db := client.Database(dbName)
			colls, err := db.ListCollectionNames(ctx, bson.D{})
			if err != nil {
				continue
			}

			c.metrics.collectionsByDBMetric.collect(
				ch,
				float64(len(colls)), append(labelsBase, dbName))
		}
	}

	// -----------------------------
	// 2. 自动发现集合数量
	// -----------------------------
	totalCollections := 0
	for _, dbName := range dbNames {
		if dbName == "local" || dbName == "config" || dbName == "admin" {
			continue
		}
		db := client.Database(dbName)
		colls, err := db.ListCollectionNames(ctx, bson.D{})
		if err != nil {
			continue
		}
		totalCollections += len(colls)
	}
	c.metrics.collectionsCountMetric.collect(
		ch,
		float64(totalCollections),
		labelsBase,
	)

	// -----------------------------
	// 3. 自动发现分片数量（从 config.shards）
	// -----------------------------
	shardColl := configDB.Collection("shards")
	shardCursor, err := shardColl.Find(ctx, bson.D{})
	if err == nil {
		defer shardCursor.Close(ctx)
		var shardCount int64 = 0
		shardMap := make(map[string]bool)

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

			shardMap[name] = true
		}

		shardCount = int64(len(shardMap))
		c.metrics.shardsCountMetric.collect(
			ch,
			float64(shardCount),
			labelsBase,
		)

		// 每个分片上的集合数量（模拟）
		for shard := range shardMap {
			count := simulateShardCollectionCount()
			c.metrics.shardCollectionsCountMetric.collect(
				ch,
				float64(count),
				append(labelsBase, shard),
			)
		}
	}

	// -----------------------------
	// 4. 自动发现启用分片的集合（来自 config.collections）
	// -----------------------------
	collColl := configDB.Collection("collections")
	collCursor, err := collColl.Find(ctx, bson.D{})
	if err == nil {
		defer collCursor.Close(ctx)
		var shardedCount int64 = 0
		for collCursor.Next(ctx) {
			shardedCount++
		}
		c.metrics.collectionsShardedMetric.collect(
			ch,
			float64(shardedCount),
			labelsBase,
		)
	}

	// -----------------------------
	// 5. 自动发现 Zones（来自 config.tags）
	// -----------------------------
	tagColl := configDB.Collection("tags")
	tagCursor, err := tagColl.Find(ctx, bson.D{})
	if err == nil {
		defer tagCursor.Close(ctx)
		zoneSet := make(map[string]bool)

		for tagCursor.Next(ctx) {
			var doc bson.M
			err := tagCursor.Decode(&doc)
			if err != nil {
				continue
			}

			zone, ok := doc["tag"].(string)
			if ok {
				zoneSet[zone] = true
			}
		}

		c.metrics.zonesCountMetric.collect(
			ch,
			float64(len(zoneSet)),
			labelsBase,
		)
		c.metrics.tagsCountMetric.collect(
			ch,
			float64(len(zoneSet)),
			labelsBase,
		) // tags 数量近似等于 zone 数量
	}

	// -----------------------------
	// 6. 自动发现 Jumbo Chunks（来自 config.chunks）
	// -----------------------------
	chunkColl := configDB.Collection("chunks")
	chunkCursor, err := chunkColl.Find(ctx, bson.D{{Key: "jumbo", Value: true}})
	if err == nil {
		defer chunkCursor.Close(ctx)
		var jumboCount int64 = 0
		for chunkCursor.Next(ctx) {
			jumboCount++
		}
		c.metrics.jumboChunksCountMetric.collect(
			ch,
			float64(jumboCount),
			labelsBase,
		)
	}
}

// 模拟每个分片上的集合数量（实际应通过连接分片查询）
func simulateShardCollectionCount() int64 {
	// 实际应通过访问分片服务器获取
	return 10
}

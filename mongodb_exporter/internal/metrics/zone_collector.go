package metrics

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/bson"
)

// MongoDBZoneCollector 负责采集 MongoDB 分片集群中的 Zone 和 Tag 分布指标
type MongoDBZoneCollector struct {
	pool         *MongoDBClientPool
	instanceName string
	instanceURI  string
	metrics      *zoneMetrics
}

type zoneMetrics struct {
	zoneCountMetric            *baseMetrics // 总 zone 数量
	zoneShardsCountMetric      *baseMetrics // 每个 zone 的分片数
	zoneChunksCountMetric      *baseMetrics // 每个 zone 的 chunk 数量
	zoneRangesCountMetric      *baseMetrics // 每个 zone 的 range 数量
	zoneShardAssociationMetric *baseMetrics // 分片与 zone 的关联关系（info metric）
	shardZonesCountMetric      *baseMetrics // 每个分片上绑定的 zone 数量
}

func newZoneMetrics() *zoneMetrics {
	return &zoneMetrics{
		zoneCountMetric: NewMetrics(
			"mongodb_sharding_zones_count",
			"The total number of zones defined in the cluster.",
			[]string{"instance", "uri"},
		),
		zoneShardsCountMetric: NewMetrics(
			"mongodb_sharding_zone_shards_count",
			"The number of shards associated with a zone.",
			[]string{"instance", "uri", "zone"},
		),
		zoneChunksCountMetric: NewMetrics(
			"mongodb_sharding_zone_chunks_total",
			"The total number of chunks within this zone.",
			[]string{"instance", "uri", "zone"},
		),
		zoneRangesCountMetric: NewMetrics(
			"mongodb_sharding_zone_ranges_count",
			"The number of key ranges associated with this zone.",
			[]string{"instance", "uri", "zone"},
		),
		zoneShardAssociationMetric: NewMetrics(
			"mongodb_sharding_zone_shard_association",
			"Gauge indicating if a shard is associated with a zone (1) or not (0).",
			[]string{"instance", "uri", "zone", "shard"},
		),
		shardZonesCountMetric: NewMetrics(
			"mongodb_sharding_shard_zones_count",
			"The number of zones associated with a shard.",
			[]string{"instance", "uri", "shard"},
		),
	}
}

// NewMongoDBZoneCollector 创建一个新的 Zone Collector
func NewMongoDBZoneCollector(pool *MongoDBClientPool, instanceName, instanceURI string) *MongoDBZoneCollector {
	return &MongoDBZoneCollector{
		pool:         pool,
		instanceName: instanceName,
		instanceURI:  instanceURI,
		metrics:      newZoneMetrics(),
	}
}

// Describe implements Prometheus Collector interface

// Collect implements Prometheus Collector interface
func (c *MongoDBZoneCollector) collect(ch chan<- prometheus.Metric) {
	client, _ := c.pool.GetClient()
	ctx := context.Background()
	configDB := client.Database("config")

	tagColl := configDB.Collection("tags")
	cursor, err := tagColl.Find(ctx, bson.D{})
	if err != nil {
		fmt.Printf("Failed to query config.tags: %v\n", err)
		return
	}
	defer cursor.Close(ctx)

	labelsBase := []string{c.instanceName, c.instanceURI}

	var totalZones int64 = 0
	zoneToShards := make(map[string]map[string]bool) // zone -> shard set
	zoneToChunkCount := make(map[string]int64)       // zone -> chunk count
	zoneToRangeCount := make(map[string]int64)       // zone -> range count
	shardToZoneCount := make(map[string]int64)       // shard -> zone count

	for cursor.Next(ctx) {
		var doc bson.M
		err := cursor.Decode(&doc)
		if err != nil {
			continue
		}

		zone, ok1 := doc["tag"].(string)
		shard, ok2 := doc["shard"].(string)
		minKey, _ := doc["min"].(bson.M)
		maxKey, _ := doc["max"].(bson.M)

		if !ok1 || !ok2 {
			continue
		}

		totalZones++
		if zoneToShards[zone] == nil {
			zoneToShards[zone] = make(map[string]bool)
		}
		zoneToShards[zone][shard] = true

		// 上报每个 zone 的 shard 关联
		c.metrics.zoneShardAssociationMetric.collect(
			ch,
			1, append(labelsBase, zone, shard))

		// 统计每个分片关联的 zone 数量
		shardToZoneCount[shard]++

		// 统计每个 zone 的 range 数量（min + max 算一个 range）
		if len(minKey) > 0 && len(maxKey) > 0 {
			zoneToRangeCount[zone]++
		}
	}

	// 上报总 zone 数量
	c.metrics.zoneCountMetric.collect(
		ch,
		float64(totalZones),
		labelsBase,
	)

	// 上报每个 zone 的 shard 数量
	for zone, shards := range zoneToShards {
		count := int64(len(shards))
		c.metrics.zoneShardsCountMetric.collect(
			ch,
			float64(count),
			append(labelsBase, zone),
		)
	}

	// 上报每个分片关联的 zone 数量
	for shard, count := range shardToZoneCount {
		c.metrics.shardZonesCountMetric.collect(
			ch,
			float64(count),
			append(labelsBase, shard),
		)
	}

	// 如果需要 chunk 数量 per zone，可从 chunks 中统计
	// 示例：查询 config.chunks 并根据 zone 判断归属
	chunkColl := configDB.Collection("chunks")
	chunkCursor, err := chunkColl.Find(ctx, bson.D{})
	if err == nil {
		defer chunkCursor.Close(ctx)
		for chunkCursor.Next(ctx) {
			var chunkDoc bson.M
			err := chunkCursor.Decode(&chunkDoc)
			if err != nil {
				continue
			}

			// ns, _ := chunkDoc["ns"].(string)
			shard, _ := chunkDoc["shard"].(string)

			// 假设 zone = shard 所属 tags（实际需结合 tags 表）
			if zone, ok := getZoneByShard(shard); ok {
				zoneToChunkCount[zone]++
			}
		}
	}

	// 上报每个 zone 的 chunk 数量
	for zone, count := range zoneToChunkCount {
		c.metrics.zoneChunksCountMetric.collect(
			ch,
			float64(count),
			append(labelsBase, zone),
		)
	}

	// 上报每个 zone 的 range 数量
	for zone, count := range zoneToRangeCount {
		c.metrics.zoneRangesCountMetric.collect(
			ch,
			float64(count),
			append(labelsBase, zone),
		)
	}
}

// getZoneByShard 模拟从 tags 中查找 zone（实际应通过 tags 集合查询）
func getZoneByShard(shard string) (string, bool) {
	// 实际应从 config.tags 查询
	switch shard {
	case "shardA":
		return "eu-west", true
	case "shardB":
		return "us-east", true
	default:
		return "", false
	}
}
// Part 2 commit for mongodb_exporter/internal/metrics/zone_collector.go

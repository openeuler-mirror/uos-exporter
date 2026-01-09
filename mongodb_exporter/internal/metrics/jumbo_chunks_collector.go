package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/bson"
)

// MongoDBJumboChunkscollector 负责采集 MongoDB 分片集群中 Jumbo Chunks 的数量和分布
type MongoDBJumboChunkscollector struct {
	pool         *MongoDBClientPool
	instanceName string
	instanceURI  string
	metrics      *jumboChunksMetrics
}

type jumboChunksMetrics struct {
	jumboChunksCountMetric        *baseMetrics // 总数
	jumboChunksBycollectionMetric *baseMetrics // 按集合统计
	jumboChunksByShardMetric      *baseMetrics // 按分片统计
	jumboChunksLastCheckTimestamp *baseMetrics // 最后一次检查时间戳
}

func newJumboChunksMetrics() *jumboChunksMetrics {
	return &jumboChunksMetrics{
		jumboChunksCountMetric: NewMetrics(
			"mongodb_sharding_jumbo_chunks_count",
			"The total number of jumbo chunks in the cluster.",
			[]string{"instance", "uri"},
		),
		jumboChunksBycollectionMetric: NewMetrics(
			"mongodb_sharding_jumbo_chunks_by_collection",
			"The number of jumbo chunks per collection.",
			[]string{"instance", "uri", "collection"},
		),
		jumboChunksByShardMetric: NewMetrics(
			"mongodb_sharding_jumbo_chunks_by_shard",
			"The number of jumbo chunks per shard.",
			[]string{"instance", "uri", "shard"},
		),
		jumboChunksLastCheckTimestamp: NewMetrics(
			"mongodb_sharding_jumbo_chunks_last_check_timestamp",
			"The timestamp of the last check for jumbo chunks.",
			[]string{"instance", "uri"},
		),
	}
}

// NewMongoDBJumboChunkscollector 创建一个新的 Jumbo Chunks collector
func NewMongoDBJumboChunkscollector(pool *MongoDBClientPool, instanceName, instanceURI string) *MongoDBJumboChunkscollector {
	return &MongoDBJumboChunkscollector{
		pool:         pool,
		instanceName: instanceName,
		instanceURI:  instanceURI,
		metrics:      newJumboChunksMetrics(),
	}
}

// Describe implements Prometheus collector interface

// collect implements Prometheus collector interface
func (c *MongoDBJumboChunkscollector) collect(ch chan<- prometheus.Metric) {
	client, _ := c.pool.GetClient()
	ctx := context.Background()
	configDB := client.Database("config")

	chunkColl := configDB.Collection("chunks")

	// 查询所有 jumbo: true 的 chunk
	filter := bson.D{{Key: "jumbo", Value: true}}
	cursor, err := chunkColl.Find(ctx, filter)
	if err != nil {
		fmt.Printf("Failed to query jumbo chunks: %v\n", err)
		return
	}
	defer cursor.Close(ctx)

	labels := []string{c.instanceName, c.instanceURI}

	var count int64 = 0
	collectionCounts := make(map[string]int64)
	shardCounts := make(map[string]int64)

	for cursor.Next(ctx) {
		var chunk bson.M
		err := cursor.Decode(&chunk)
		if err != nil {
			fmt.Printf("Failed to decode chunk document: %v\n", err)
			continue
		}

		ns, ok1 := chunk["ns"].(string)
		shard, ok2 := chunk["shard"].(string)

		if !ok1 || !ok2 {
			continue
		}

		count++
		collectionCounts[ns]++
		shardCounts[shard]++
	}

	// 上报总数
	c.metrics.jumboChunksCountMetric.collect(
		ch,
		float64(count),
		labels,
	)

	// 按 collection 统计
	for coll, cnt := range collectionCounts {
		c.metrics.jumboChunksBycollectionMetric.collect(
			ch,
			float64(cnt),
			append(labels, coll),
		)
	}

	// 按 shard 统计
	for shard, cnt := range shardCounts {
		c.metrics.jumboChunksByShardMetric.collect(
			ch,
			float64(cnt),
			append(labels, shard),
		)
	}

	// 上报最后检查时间戳
	c.metrics.jumboChunksLastCheckTimestamp.collect(
		ch,
		float64(time.Now().Unix()),
		labels,
	)
}

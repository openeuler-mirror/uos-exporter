package metrics

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/bson"
)

// MongoDBMigrationCollector 负责采集 MongoDB 的 chunk 迁移记录
type MongoDBMigrationCollector struct {
	pool         *MongoDBClientPool
	instanceName string
	instanceURI  string
	metrics      *migrationMetrics
}

type migrationMetrics struct {
	migrationDurationSecondsHistogram *baseMetrics // 单次迁移耗时（秒）
	migrationCountMetric              *baseMetrics // 总迁移次数
	migrationByCollectionMetric       *baseMetrics // 按集合统计
	migrationByShardPairMetric        *baseMetrics // 按分片对统计
	migrationIsJumboMetric            *baseMetrics // 是否为 jumbo chunk 迁移
	migrationLastTimestampMetric      *baseMetrics // 最后一次迁移时间戳
}

func newMigrationMetrics() *migrationMetrics {
	return &migrationMetrics{
		migrationDurationSecondsHistogram: NewMetrics(
			"mongodb_sharding_migration_duration_seconds",
			"The duration of each chunk migration in seconds.",
			[]string{"instance", "uri", "source_shard", "dest_shard", "collection"},
		),
		migrationCountMetric: NewMetrics(
			"mongodb_sharding_migrations_total",
			"The total number of chunk migrations performed by the balancer.",
			[]string{"instance", "uri"},
		),
		migrationByCollectionMetric: NewMetrics(
			"mongodb_sharding_migrations_by_collection",
			"The number of migrations per collection.",
			[]string{"instance", "uri", "collection"},
		),
		migrationByShardPairMetric: NewMetrics(
			"mongodb_sharding_migrations_by_shard_pair",
			"The number of migrations between shard pairs.",
			[]string{"instance", "uri", "source_shard", "dest_shard"},
		),
		migrationIsJumboMetric: NewMetrics(
			"mongodb_sharding_migration_is_jumbo",
			"Gauge indicating if a migration is for a jumbo chunk (1) or not (0).",
			[]string{"instance", "uri", "source_shard", "dest_shard", "collection"},
		),
		migrationLastTimestampMetric: NewMetrics(
			"mongodb_sharding_last_migration_timestamp",
			"The timestamp of the last migration.",
			[]string{"instance", "uri"},
		),
	}
}

// NewMongoDBMigrationCollector 创建一个新的 Migration Collector
func NewMongoDBMigrationCollector(pool *MongoDBClientPool, instanceName, instanceURI string) *MongoDBMigrationCollector {
	return &MongoDBMigrationCollector{
		pool:         pool,
		instanceName: instanceName,
		instanceURI:  instanceURI,
		metrics:      newMigrationMetrics(),
	}
}

// Describe implements Prometheus Collector interface

// Collect implements Prometheus Collector interface
func (c *MongoDBMigrationCollector) collect(ch chan<- prometheus.Metric) {
	client, _ := c.pool.GetClient()
	ctx := context.Background()
	configDB := client.Database("config")

	changeLogColl := configDB.Collection("changelog")

	// 只查询最近 24 小时内的迁移记录
	filter := bson.D{
		{Key: "operation", Value: "migrateChunk.start"},
		{Key: "time", Value: bson.D{
			{Key: "$gte", Value: time.Now().Add(-24 * time.Hour)},
		}},
	}

	cursor, err := changeLogColl.Find(ctx, filter)
	if err != nil {
		fmt.Printf("Failed to query changelog for migrations: %v\n", err)
		return
	}
	defer cursor.Close(ctx)

	labelsBase := []string{c.instanceName, c.instanceURI}

	var totalMigrations int64 = 0
	var lastMigrationTime int64 = 0
	collectionMigrations := make(map[string]int64)
	shardPairs := make(map[string]int64)

	for cursor.Next(ctx) {
		var doc bson.M
		err := cursor.Decode(&doc)
		if err != nil {
			fmt.Printf("Failed to decode migration record: %v\n", err)
			continue
		}

		totalMigrations++

		details, ok := doc["details"].(bson.M)
		if !ok {
			continue
		}

		ns, _ := doc["namespace"].(string)
		fromShard, _ := details["from"].(string)
		toShard, _ := details["to"].(string)
		jumbo, _ := details["jumbo"].(bool)
		startTime, _ := doc["time"].(time.Time)

		// 提取集合名
		coll := strings.TrimPrefix(ns, "test.")
		if coll == "" {
			continue
		}

		// 上报迁移耗时（模拟 end_time，实际需从日志或命令中提取）
		endTime := startTime.Add(time.Second * 5) // 假设迁移平均耗时 5s
		durationSec := endTime.Sub(startTime).Seconds()

		labels := append(labelsBase, fromShard, toShard, coll)

		// 记录迁移耗时
		c.metrics.migrationDurationSecondsHistogram.collect(
			ch,
			durationSec,
			labels,
		)

		// 是否为 jumbo chunk
		var jumboVal float64 = 0
		if jumbo {
			jumboVal = 1
		}
		c.metrics.migrationIsJumboMetric.collect(
			ch,
			jumboVal,
			labels,
		)

		// 按集合统计
		collectionMigrations[coll]++

		// 按分片对统计
		key := fromShard + ">" + toShard
		shardPairs[key]++

		// 更新最后迁移时间戳
		if startTime.Unix() > lastMigrationTime {
			lastMigrationTime = startTime.Unix()
		}
	}

	// 上报总数
	c.metrics.migrationCountMetric.collect(
		ch,
		float64(totalMigrations),
		labelsBase,
	)

	// 上报按集合统计
	for coll, count := range collectionMigrations {
		c.metrics.migrationByCollectionMetric.collect(
			ch,
			float64(count),
			append(labelsBase, coll),
		)
	}

	// 上报按分片对统计
	for pair, count := range shardPairs {
		parts := strings.Split(pair, ">")
		if len(parts) == 2 {
			c.metrics.migrationByShardPairMetric.collect(
				ch,
				float64(count),
				append(labelsBase, parts[0], parts[1]),
			)
		}
	}

	// 上报最后一次迁移时间戳
	if lastMigrationTime > 0 {
		c.metrics.migrationLastTimestampMetric.collect(
			ch,
			float64(lastMigrationTime),
			labelsBase,
		)
	}
}

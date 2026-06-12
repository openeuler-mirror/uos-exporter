package metrics

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDBBalancercollector 负责采集 MongoDB 分片集群中的 Balancer 状态和迁移记录
type MongoDBBalancercollector struct {
	pool         *MongoDBClientPool
	instanceName string
	instanceURI  string
	metrics      *balancerMetrics
}

type balancerMetrics struct {
	balancerActiveMetric                      *baseMetrics // Balancer 是否活跃
	balancerLastRunTimestamp                  *baseMetrics // 最后一次运行时间戳
	balancerMigrationsTotalMetric             *baseMetrics // 总迁移数
	balancerMigrationDurationSecondsHistogram *baseMetrics // 单次迁移耗时（秒）
	balancerJumboMigrationsTotalMetric        *baseMetrics // Jumbo Chunk 迁移总数
	balancerMigrationsBycollectionMetric      *baseMetrics // 按集合统计迁移数
	balancerMigrationsByShardPairMetric       *baseMetrics // 按源分片 + 目标分片统计迁移数
}

func newBalancerMetrics() *balancerMetrics {
	return &balancerMetrics{
		balancerActiveMetric: NewMetrics(
			"mongodb_sharding_balancer_active",
			"Gauge indicating if the balancer is active (1) or not (0).",
			[]string{"instance", "uri"},
		),
		balancerLastRunTimestamp: NewMetrics(
			"mongodb_sharding_balancer_last_run_timestamp",
			"The timestamp of the last balancer run.",
			[]string{"instance", "uri"},
		),
		balancerMigrationsTotalMetric: NewMetrics(
			"mongodb_sharding_balancer_migrations_total",
			"The total number of migrations performed by the balancer.",
			[]string{"instance", "uri"},
		),
		balancerMigrationDurationSecondsHistogram: NewMetrics(
			"mongodb_sharding_balancer_migration_duration_seconds",
			"The duration of each chunk migration in seconds.",
			[]string{"instance", "uri", "source_shard", "dest_shard", "collection"},
		),
		balancerJumboMigrationsTotalMetric: NewMetrics(
			"mongodb_sharding_balancer_jumbo_migrations_total",
			"The total number of jumbo chunk migrations performed by the balancer.",
			[]string{"instance", "uri"},
		),
		balancerMigrationsBycollectionMetric: NewMetrics(
			"mongodb_sharding_balancer_migrations_by_collection",
			"The number of migrations per collection.",
			[]string{"instance", "uri", "collection"},
		),
		balancerMigrationsByShardPairMetric: NewMetrics(
			"mongodb_sharding_balancer_migrations_by_shard_pair",
			"The number of migrations between shard pairs.",
			[]string{"instance", "uri", "source_shard", "dest_shard"},
		),
	}
}

// NewMongoDBBalancercollector 创建一个新的 Balancer collector
func NewMongoDBBalancercollector(pool *MongoDBClientPool, instanceName, instanceURI string) *MongoDBBalancercollector {
	return &MongoDBBalancercollector{
		pool:         pool,
		instanceName: instanceName,
		instanceURI:  instanceURI,
		metrics:      newBalancerMetrics(),
	}
}

// Describe implements Prometheus collector interface

// collect implements Prometheus collector interface
func (c *MongoDBBalancercollector) collect(ch chan<- prometheus.Metric) {
	client, _ := c.pool.GetClient()
	ctx := context.Background()
	configDB := client.Database("config")

	// --- Balancer 锁状态 ---
	lockColl := configDB.Collection("locks")
	var lockDoc bson.M
	err := lockColl.FindOne(ctx, bson.D{{Key: "_id", Value: "balancer"}}, options.FindOne()).Decode(&lockDoc)
	// err := lockColl.FindOne(ctx, bson.D{{Key: "_id", Value: "balancer"}}, mongo.FindOneOptions{}).Decode(&lockDoc)
	if err != nil {
		fmt.Printf("Failed to query locks for balancer: %v\n", err)
	} else {
		state, ok := lockDoc["state"].(int32)
		if ok && state == 2 { // state=2 表示锁定中（即 balancer 正在运行）
			c.metrics.balancerActiveMetric.collect(
				ch,
				1,
				[]string{c.instanceName, c.instanceURI},
			)
		} else {
			c.metrics.balancerActiveMetric.collect(
				ch,
				0,
				[]string{c.instanceName, c.instanceURI},
			)
		}
	}

	// --- changelog 中的迁移记录 ---
	changeLogColl := configDB.Collection("changelog")

	// 只查询最近 N 条包含 "migrateChunk." 的操作
	filter := bson.D{
		{Key: "operation", Value: "migrateChunk.start"},
		{Key: "time", Value: bson.D{
			{Key: "$gte", Value: time.Now().Add(-24 * time.Hour)}, // 最近 24 小时
		}},
	}

	cursor, err := changeLogColl.Find(ctx, filter)
	if err != nil {
		fmt.Printf("Failed to query changelog for balancer: %v\n", err)
		return
	}
	defer cursor.Close(ctx)

	labelsBase := []string{c.instanceName, c.instanceURI}

	var totalMigrations int64 = 0
	var totalJumboMigrations int64 = 0
	collectionMigrations := make(map[string]int64)
	shardPairs := make(map[string]int64)

	for cursor.Next(ctx) {
		var doc bson.M
		err := cursor.Decode(&doc)
		if err != nil {
			continue
		}

		// 提取字段
		ns, _ := doc["namespace"].(string)
		details, ok := doc["details"].(bson.M)
		if !ok {
			continue
		}

		fromShard, _ := details["from"].(string)
		toShard, _ := details["to"].(string)
		coll := strings.TrimPrefix(ns, "test.")
		jumbo, _ := details["jumbo"].(bool)

		// 上报迁移记录
		totalMigrations++
		collectionMigrations[coll]++

		key := fromShard + ">" + toShard
		shardPairs[key]++

		if jumbo {
			totalJumboMigrations++
		}

		// 迁移耗时
		startTime, _ := doc["time"].(time.Time)
		endTime := startTime // 假设没有 end 字段，实际可从日志或命令中提取
		durationSec := endTime.Sub(startTime).Seconds()
		c.metrics.balancerMigrationDurationSecondsHistogram.collect(
			ch,
			durationSec,
			append(labelsBase, fromShard, toShard, coll),
		)
	}

	// 上报总数
	c.metrics.balancerMigrationsTotalMetric.collect(
		ch,
		float64(totalMigrations),
		labelsBase,
	)
	c.metrics.balancerJumboMigrationsTotalMetric.collect(
		ch,
		float64(totalJumboMigrations),
		labelsBase,
	)

	// 按集合上报迁移数
	for coll, count := range collectionMigrations {
		c.metrics.balancerMigrationsBycollectionMetric.collect(
			ch,
			float64(count),
			append(labelsBase, coll),
		)
	}

	// 按分片对上报迁移数
	for pair, count := range shardPairs {
		parts := strings.Split(pair, ">")
		if len(parts) == 2 {
			c.metrics.balancerMigrationsByShardPairMetric.collect(
				ch,
				float64(count),
				append(labelsBase, parts[0], parts[1]),
			)
		}
	}

	// 最后一次运行时间戳（模拟为当前时间）
	c.metrics.balancerLastRunTimestamp.collect(
		ch,
		float64(time.Now().Unix()),
		labelsBase,
	)
}

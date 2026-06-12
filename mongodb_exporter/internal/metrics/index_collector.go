package metrics

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/bson"
)

// MongoDBIndexStatscollector 负责采集每个集合下索引的指标
type MongoDBIndexStatscollector struct {
	pool         *MongoDBClientPool
	instanceName string
	instanceURI  string
	metrics      *indexStatsMetrics
}

type indexStatsMetrics struct {
	indexSizeBytesMetric  *baseMetrics
	indexIsUniqueMetric   *baseMetrics
	indexIsSparseMetric   *baseMetrics
	indexIsTtlMetric      *baseMetrics
	indexIsPartialMetric  *baseMetrics
	indexItemsCountMetric *baseMetrics
}

func newIndexStatsMetrics() *indexStatsMetrics {
	return &indexStatsMetrics{
		indexSizeBytesMetric: NewMetrics(
			"mongodb_index_size_bytes",
			"The size of the index in bytes.",
			[]string{"instance", "uri", "db", "collection", "index_name"},
		),
		indexIsUniqueMetric: NewMetrics(
			"mongodb_index_unique",
			"Whether the index is unique (1) or not (0).",
			[]string{"instance", "uri", "db", "collection", "index_name"},
		),
		indexIsSparseMetric: NewMetrics(
			"mongodb_index_sparse",
			"Whether the index is sparse (1) or not (0).",
			[]string{"instance", "uri", "db", "collection", "index_name"},
		),
		indexIsTtlMetric: NewMetrics(
			"mongodb_index_ttl",
			"Whether the index is a TTL index (1) or not (0).",
			[]string{"instance", "uri", "db", "collection", "index_name"},
		),
		indexIsPartialMetric: NewMetrics(
			"mongodb_index_partial",
			"Whether the index is partial (1) or not (0).",
			[]string{"instance", "uri", "db", "collection", "index_name"},
		),
		indexItemsCountMetric: NewMetrics(
			"mongodb_index_items_count",
			"The number of items in the index.",
			[]string{"instance", "uri", "db", "collection", "index_name"},
		),
	}
}

// NewMongoDBIndexStatscollector 创建一个新的 Index Stats collector
func NewMongoDBIndexStatscollector(pool *MongoDBClientPool, instanceName, instanceURI string) *MongoDBIndexStatscollector {
	return &MongoDBIndexStatscollector{
		pool:         pool,
		instanceName: instanceName,
		instanceURI:  instanceURI,
		metrics:      newIndexStatsMetrics(),
	}
}

// Describe implements Prometheus collector interface

// collect implements Prometheus collector interface
func (c *MongoDBIndexStatscollector) collect(ch chan<- prometheus.Metric) {
	client, _ := c.pool.GetClient()
	ctx := context.Background()

	// 获取数据库列表
	// adminDB := client.Database("admin")
	// dbs, err := adminDB.ListDatabaseNames(ctx, bson.D{})
	dbs, err := client.ListDatabaseNames(ctx, bson.D{})
	if err != nil {
		fmt.Printf("Failed to list databases: %v\n", err)
		return
	}

	for _, dbName := range dbs {
		db := client.Database(dbName)

		// 获取集合列表
		names, err := db.ListCollectionNames(ctx, bson.D{})
		if err != nil {
			fmt.Printf("Failed to list collections in DB %s: %v\n", dbName, err)
			continue
		}

		for _, collName := range names {
			// collection := db.Collection(collName)

			// 执行 coll.stats({indexDetails: true})
			cmd := bson.D{
				{Key: "collStats", Value: collName},
				{Key: "indexDetails", Value: true},
			}
			var result bson.M
			err = db.RunCommand(ctx, cmd).Decode(&result)
			if err != nil {
				fmt.Printf("Failed to run collStats on %s.%s: %v\n", dbName, collName, err)
				continue
			}

			// 获取 indexDetails 数据
			indexDetails, ok := result["indexDetails"].(bson.M)
			if !ok {
				continue
			}

			labelsBase := []string{c.instanceName, c.instanceURI, dbName, collName}

			// 遍历每个索引
			for idxName, idxData := range indexDetails {
				idxMap, ok := idxData.(bson.M)
				if !ok {
					continue
				}

				labels := append(labelsBase, idxName)

				// 索引大小
				if size, ok := idxMap["size"].(int32); ok {
					c.metrics.indexSizeBytesMetric.collect(
						ch,
						float64(size),
						labels,
					)
				} else if size, ok := idxMap["size"].(float64); ok {
					c.metrics.indexSizeBytesMetric.collect(
						ch,
						size,
						labels,
					)
				}

				// 是否是唯一索引
				if spec, ok := idxMap["spec"].(bson.M); ok {
					if unique, ok := spec["unique"].(bool); ok {
						var val float64 = 0
						if unique {
							val = 1
						}
						c.metrics.indexIsUniqueMetric.collect(
							ch,
							val,
							labels,
						)
					}

					// 是否是稀疏索引
					if sparse, ok := spec["sparse"].(bool); ok {
						var val float64 = 0
						if sparse {
							val = 1
						}
						c.metrics.indexIsSparseMetric.collect(
							ch,
							val,
							labels,
						)
					}

					// 是否是 TTL 索引
					if expireAfterSeconds, ok := spec["expireAfterSeconds"].(int32); ok && expireAfterSeconds > 0 {
						c.metrics.indexIsTtlMetric.collect(
							ch,
							1,
							labels,
						)
					} else {
						c.metrics.indexIsTtlMetric.collect(
							ch,
							0,
							labels,
						)
					}

					// 是否是 Partial 索引
					if partialFilterExpression, ok := spec["partialFilterExpression"].(bson.M); ok && len(partialFilterExpression) > 0 {
						c.metrics.indexIsPartialMetric.collect(
							ch,
							1,
							labels,
						)
					} else {
						c.metrics.indexIsPartialMetric.collect(
							ch,
							0,
							labels,
						)
					}
				}

				// 索引中条目数量
				if count, ok := idxMap["items"].(int64); ok {
					c.metrics.indexItemsCountMetric.collect(
						ch,
						float64(count),
						labels,
					)
				} else if count, ok := idxMap["items"].(float64); ok {
					c.metrics.indexItemsCountMetric.collect(
						ch,
						count,
						labels,
					)
				}
			}
		}
	}
}

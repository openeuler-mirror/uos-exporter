package metrics

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/bson"
)

// MongoDBcollectionStatscollector 负责采集每个集合的 stats 指标
type MongoDBcollectionStatscollector struct {
	pool         *MongoDBClientPool
	instanceName string
	instanceURI  string
	metrics      *collectionStatsMetrics
}

type collectionStatsMetrics struct {
	collectionSizeBytesMetric           *baseMetrics
	collectionStorageSizeBytesMetric    *baseMetrics
	collectionTotalIndexSizeBytesMetric *baseMetrics
	collectionDocumentCountMetric       *baseMetrics
	collectionAvgObjSizeBytesMetric     *baseMetrics
	collectionIndexCountMetric          *baseMetrics
	collectionExtentCountMetric         *baseMetrics
	collectionMaxDocSizeMetric          *baseMetrics
	collectionNindexesMetric            *baseMetrics
	collectionTotalIndexSizeMetric      *baseMetrics
	collectionCappedMetric              *baseMetrics
}

func newcollectionStatsMetrics() *collectionStatsMetrics {
	return &collectionStatsMetrics{
		collectionSizeBytesMetric: NewMetrics(
			"mongodb_collection_size_bytes",
			"The total size of the data in this collection (excluding indexes).",
			[]string{"instance", "uri", "db", "collection"},
		),
		collectionStorageSizeBytesMetric: NewMetrics(
			"mongodb_collection_storage_size_bytes",
			"The total amount of storage allocated to this collection.",
			[]string{"instance", "uri", "db", "collection"},
		),
		collectionTotalIndexSizeBytesMetric: NewMetrics(
			"mongodb_collection_total_index_size_bytes",
			"The total size of all indexes on this collection.",
			[]string{"instance", "uri", "db", "collection"},
		),
		collectionDocumentCountMetric: NewMetrics(
			"mongodb_collection_doc_count",
			"The number of documents in the collection.",
			[]string{"instance", "uri", "db", "collection"},
		),
		collectionAvgObjSizeBytesMetric: NewMetrics(
			"mongodb_collection_avg_obj_size_bytes",
			"The average document size in bytes.",
			[]string{"instance", "uri", "db", "collection"},
		),
		collectionIndexCountMetric: NewMetrics(
			"mongodb_collection_index_count",
			"The number of indexes on the collection.",
			[]string{"instance", "uri", "db", "collection"},
		),
		collectionExtentCountMetric: NewMetrics(
			"mongodb_collection_extent_count",
			"The number of extents used by the collection.",
			[]string{"instance", "uri", "db", "collection"},
		),
		collectionMaxDocSizeMetric: NewMetrics(
			"mongodb_collection_max_doc_size",
			"The maximum document size allowed for capped collections.",
			[]string{"instance", "uri", "db", "collection"},
		),
		collectionNindexesMetric: NewMetrics(
			"mongodb_collection_nindexes",
			"The number of indexes on the collection.",
			[]string{"instance", "uri", "db", "collection"},
		),
		collectionTotalIndexSizeMetric: NewMetrics(
			"mongodb_collection_total_index_size",
			"The total size of all indexes in bytes.",
			[]string{"instance", "uri", "db", "collection"},
		),
		collectionCappedMetric: NewMetrics(
			"mongodb_collection_capped",
			"Whether the collection is a capped collection (1) or not (0).",
			[]string{"instance", "uri", "db", "collection"},
		),
	}
}

// NewMongoDBcollectionStatscollector 创建一个新的 collection Stats collector
func NewMongoDBcollectionStatscollector(pool *MongoDBClientPool, instanceName, instanceURI string) *MongoDBcollectionStatscollector {
	return &MongoDBcollectionStatscollector{
		pool:         pool,
		instanceName: instanceName,
		instanceURI:  instanceURI,
		metrics:      newcollectionStatsMetrics(),
	}
}

// Describe implements Prometheus collector interface

// collect implements Prometheus collector interface
func (c *MongoDBcollectionStatscollector) collect(ch chan<- prometheus.Metric) {
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

	collected := make(map[string]bool)

	for _, dbName := range dbs {
		db := client.Database(dbName)

		// 获取集合列表
		names_list, err := db.ListCollectionNames(ctx, bson.D{})
		if err != nil {
			fmt.Printf("Failed to list collections in DB %s: %v\n", dbName, err)
			continue
		}

		// names 去重
		names := removeDuplicates(names_list)

		for _, collName := range names {
			// collection := db.Collection(collName)

			key := fmt.Sprintf("%s.%s", dbName, collName)
			if collected[key] {
				continue
			}
			collected[key] = true

			// 执行 coll.stats()
			cmd := bson.D{{Key: "collStats", Value: collName}}
			var result bson.M
			err = db.RunCommand(ctx, cmd).Decode(&result)
			if err != nil {
				fmt.Printf("Failed to run collStats on %s.%s: %v\n", dbName, collName, err)
				continue
			}

			labels := []string{c.instanceName, c.instanceURI, dbName, collName}

			// 数据大小
			if size, ok := result["size"].(float64); ok {
				c.metrics.collectionSizeBytesMetric.collect(
					ch,
					size,
					labels,
				)
			}

			// 存储大小
			if storageSize, ok := result["storageSize"].(float64); ok {
				c.metrics.collectionStorageSizeBytesMetric.collect(
					ch,
					storageSize,
					labels,
				)
			}

			// 总索引大小
			if totalIndexSize, ok := result["totalIndexSize"].(float64); ok {
				c.metrics.collectionTotalIndexSizeBytesMetric.collect(
					ch,
					totalIndexSize,
					labels,
				)
			}

			// 文档数量
			if count, ok := result["count"].(int64); ok {
				c.metrics.collectionDocumentCountMetric.collect(
					ch,
					float64(count),
					labels,
				)
			} else if count, ok := result["count"].(float64); ok {
				c.metrics.collectionDocumentCountMetric.collect(
					ch,
					count,
					labels,
				)
			}

			// 平均文档大小
			if avgObjSize, ok := result["avgObjSize"].(float64); ok {
				c.metrics.collectionAvgObjSizeBytesMetric.collect(
					ch,
					avgObjSize,
					labels,
				)
			}

			// Index Count
			if indexSizes, ok := result["indexSizes"].(bson.M); ok {
				indexCount := len(indexSizes)
				c.metrics.collectionIndexCountMetric.collect(
					ch,
					float64(indexCount),
					labels,
				)
			}

			// Extent Count
			if extentCount, ok := result["extentCount"].(int32); ok {
				c.metrics.collectionExtentCountMetric.collect(
					ch,
					float64(extentCount),
					labels,
				)
			}

			// Max Document Size（仅限 capped）
			if maxDocSize, ok := result["max"].(int64); ok && maxDocSize > 0 {
				c.metrics.collectionMaxDocSizeMetric.collect(
					ch,
					float64(maxDocSize),
					labels,
				)
			}

			// nindexes
			if nindexes, ok := result["nindexes"].(int32); ok {
				c.metrics.collectionNindexesMetric.collect(
					ch,
					float64(nindexes),
					labels,
				)
			}

			// capped 标志
			if capped, ok := result["capped"].(bool); ok {
				var val float64 = 0
				if capped {
					val = 1
				}
				c.metrics.collectionCappedMetric.collect(
					ch,
					val,
					labels,
				)
			}
		}
	}
}

func removeDuplicates(names []string) []string {
	encountered := map[string]bool{}
	result := []string{}

	for v := range names {
		if encountered[names[v]] == true {
			// 已经存在于map中的值被跳过
			continue
		}
		encountered[names[v]] = true      // 首次遇到的值加入到map
		result = append(result, names[v]) // 添加到结果切片
	}
	return result
}

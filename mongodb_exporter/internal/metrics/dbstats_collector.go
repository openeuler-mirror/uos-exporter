package metrics

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/bson"
)

// MongoDBDBStatscollector 负责采集每个数据库的 stats 指标
type MongoDBDBStatscollector struct {
	pool         *MongoDBClientPool
	instanceName string
	instanceURI  string
	metrics      *dbStatsMetrics
}

type dbStatsMetrics struct {
	dbDataSizeBytesMetric             *baseMetrics
	dbStorageSizeBytesMetric          *baseMetrics
	dbIndexSizeBytesMetric            *baseMetrics
	dbcollectionsCountMetric          *baseMetrics
	dbObjectsCountMetric              *baseMetrics
	dbAvgObjSizeBytesMetric           *baseMetrics
	dbFreeStorageSizeBytesMetric      *baseMetrics
	dbViewsCountMetric                *baseMetrics
	dbTableNamespacesCountMetric      *baseMetrics
	dbExtentsFreeListCachedSizeMetric *baseMetrics
}

func newDBStatsMetrics() *dbStatsMetrics {
	return &dbStatsMetrics{
		dbDataSizeBytesMetric: NewMetrics(
			"mongodb_db_data_size_bytes",
			"The total size of the data in the database (uncompressed, not including indexes).",
			[]string{"instance", "uri", "db"},
		),
		dbStorageSizeBytesMetric: NewMetrics(
			"mongodb_db_storage_size_bytes",
			"The total amount of storage allocated to collections for this database.",
			[]string{"instance", "uri", "db"},
		),
		dbIndexSizeBytesMetric: NewMetrics(
			"mongodb_db_index_size_bytes",
			"The total size of all indexes in the database.",
			[]string{"instance", "uri", "db"},
		),
		dbcollectionsCountMetric: NewMetrics(
			"mongodb_db_collections_count",
			"The number of collections in the database.",
			[]string{"instance", "uri", "db"},
		),
		dbObjectsCountMetric: NewMetrics(
			"mongodb_db_objects_count",
			"The total number of documents in the database.",
			[]string{"instance", "uri", "db"},
		),
		dbAvgObjSizeBytesMetric: NewMetrics(
			"mongodb_db_avg_obj_size_bytes",
			"The average size of a document in the database.",
			[]string{"instance", "uri", "db"},
		),
		dbFreeStorageSizeBytesMetric: NewMetrics(
			"mongodb_db_free_storage_size_bytes",
			"The total amount of storage available that has already been allocated for future writes.",
			[]string{"instance", "uri", "db"},
		),
		dbViewsCountMetric: NewMetrics(
			"mongodb_db_views_count",
			"The number of views in the database.",
			[]string{"instance", "uri", "db"},
		),
		dbTableNamespacesCountMetric: NewMetrics(
			"mongodb_db_table_namespaces_count",
			"The number of namespaces used by tables and indexes.",
			[]string{"instance", "uri", "db"},
		),
		dbExtentsFreeListCachedSizeMetric: NewMetrics(
			"mongodb_db_extents_free_list_cached_size_bytes",
			"The total size of free space cached in extents.",
			[]string{"instance", "uri", "db"},
		),
	}
}

// NewMongoDBDBStatscollector 创建一个新的 DB Stats collector
func NewMongoDBDBStatscollector(pool *MongoDBClientPool, instanceName, instanceURI string) *MongoDBDBStatscollector {
	return &MongoDBDBStatscollector{
		pool:         pool,
		instanceName: instanceName,
		instanceURI:  instanceURI,
		metrics:      newDBStatsMetrics(),
	}
}

// Describe implements Prometheus collector interface

// collect implements Prometheus collector interface
func (c *MongoDBDBStatscollector) collect(ch chan<- prometheus.Metric) {
	client, _ := c.pool.GetClient()
	ctx := context.Background()
	// adminDB := client.Database("admin")

	// 获取数据库列表
	// dbs, err := adminDB.ListDatabaseNames(ctx, bson.D{})
	// 获取数据库列表
	dbs, err := client.ListDatabaseNames(ctx, bson.D{})
	if err != nil {
		fmt.Printf("Failed to list databases: %v\n", err)
		return
	}

	for _, dbName := range dbs {
		db := client.Database(dbName)

		// 执行 db.runCommand({dbStats: 1})
		cmd := bson.D{{Key: "dbStats", Value: 1}}
		var result bson.M
		err = db.RunCommand(ctx, cmd).Decode(&result)
		if err != nil {
			fmt.Printf("Failed to run dbStats on %s: %v\n", dbName, err)
			continue
		}

		labels := []string{c.instanceName, c.instanceURI, dbName}

		// 数据大小
		if dataSize, ok := result["dataSize"].(float64); ok {
			c.metrics.dbDataSizeBytesMetric.collect(
				ch,
				dataSize,
				labels,
			)
		}

		// 存储大小
		if storageSize, ok := result["storageSize"].(float64); ok {
			c.metrics.dbStorageSizeBytesMetric.collect(
				ch,
				storageSize,
				labels,
			)
		}

		// 索引大小
		if indexSize, ok := result["indexSize"].(float64); ok {
			c.metrics.dbIndexSizeBytesMetric.collect(
				ch,
				indexSize,
				labels,
			)
		}

		// 集合数量
		if collections, ok := result["collections"].(int32); ok {
			c.metrics.dbcollectionsCountMetric.collect(
				ch,
				float64(collections),
				labels,
			)
		}

		// 文档总数
		if objects, ok := result["objects"].(int64); ok {
			c.metrics.dbObjectsCountMetric.collect(
				ch,
				float64(objects),
				labels,
			)
		} else if objects, ok := result["objects"].(float64); ok {
			c.metrics.dbObjectsCountMetric.collect(
				ch,
				objects,
				labels,
			)
		}

		// 平均文档大小
		if avgObjSize, ok := result["avgObjSize"].(float64); ok {
			c.metrics.dbAvgObjSizeBytesMetric.collect(
				ch,
				avgObjSize,
				labels,
			)
		}

		// Free Storage Size
		if fsTotalSize, ok := result["freeStorageSize"].(float64); ok {
			c.metrics.dbFreeStorageSizeBytesMetric.collect(
				ch,
				fsTotalSize,
				labels,
			)
		}

		// Views 数量
		if views, ok := result["views"].(int32); ok {
			c.metrics.dbViewsCountMetric.collect(
				ch,
				float64(views),
				labels,
			)
		}

		// Table Namespaces Count
		if tableNs, ok := result["table_namespaces"].(int32); ok {
			c.metrics.dbTableNamespacesCountMetric.collect(
				ch,
				float64(tableNs),
				labels,
			)
		}

		// Extents Free List Cached Size
		if extentFreeList, ok := result["extentFreeList"].(bson.M); ok {
			if totalSize, ok := extentFreeList["totalSize"].(float64); ok {
				c.metrics.dbExtentsFreeListCachedSizeMetric.collect(
					ch,
					totalSize,
					labels,
				)
			}
		}
	}
}

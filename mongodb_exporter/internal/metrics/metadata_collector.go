package metrics

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/bson"
)

// MongoDBMetadataCollector 负责采集 MongoDB 的元数据信息（如数据库、集合、索引等）
type MongoDBMetadataCollector struct {
	pool         *MongoDBClientPool
	instanceName string
	instanceURI  string
	metrics      *metadataMetrics
}

type metadataMetrics struct {
	dbCreatedTimestampMetric *baseMetrics // 数据库创建时间戳
	dbAdminInfoMetric        *baseMetrics // 是否是 admin / local / config DB

	collectionCreatedTimestampMetric *baseMetrics // 集合创建时间戳
	collectionIsShardedMetric        *baseMetrics // 是否启用分片
	collectionCappedMetric           *baseMetrics // 是否是 capped collection
	collectionMaxSizeMetric          *baseMetrics // capped collection 最大大小
	collectionShardKeyMetric         *baseMetrics // 分片键字段
	collectionShardKeyHashedMetric   *baseMetrics // 分片键是否是 hashed
	collectionStorageEngineMetric    *baseMetrics // 存储引擎类型

	indexCreatedTimestampMetric *baseMetrics // 索引创建时间戳
	indexIsUniqueMetric         *baseMetrics // 是否唯一
	indexIsSparseMetric         *baseMetrics // 是否稀疏
	indexIsTTLMetric            *baseMetrics // 是否 TTL 索引
	indexIsPartialMetric        *baseMetrics // 是否 partial 索引
	indexKeyPatternMetric       *baseMetrics // 索引字段组合
	indexDirectionMetric        *baseMetrics // 索引方向（升序/降序）

	shardingTagRangeMetric     *baseMetrics // 标签范围 min/max
	shardingZoneForShardMetric *baseMetrics // 分片绑定的 zone
}

func newMetadataMetrics() *metadataMetrics {
	return &metadataMetrics{
		dbCreatedTimestampMetric: NewMetrics(
			"mongodb_metadata_db_created_timestamp",
			"The timestamp when the database was created.",
			[]string{"instance", "uri", "db"},
		),
		dbAdminInfoMetric: NewMetrics(
			"mongodb_metadata_db_is_admin",
			"Gauge indicating if this is an admin, config or local database (1) or not (0).",
			[]string{"instance", "uri", "db"},
		),

		collectionCreatedTimestampMetric: NewMetrics(
			"mongodb_metadata_collection_created_timestamp",
			"The timestamp when the collection was created.",
			[]string{"instance", "uri", "db", "collection"},
		),
		collectionIsShardedMetric: NewMetrics(
			"mongodb_metadata_collection_sharded",
			"Gauge indicating if a collection is sharded (1) or not (0).",
			[]string{"instance", "uri", "db", "collection"},
		),
		collectionCappedMetric: NewMetrics(
			"mongodb_metadata_collection_capped",
			"Gauge indicating if the collection is capped (1) or not (0).",
			[]string{"instance", "uri", "db", "collection"},
		),
		collectionMaxSizeMetric: NewMetrics(
			"mongodb_metadata_collection_max_size_bytes",
			"The maximum size of a capped collection in bytes.",
			[]string{"instance", "uri", "db", "collection"},
		),
		collectionShardKeyMetric: NewMetrics(
			"mongodb_metadata_collection_shard_key",
			"Gauge indicating the shard key used for this collection.",
			[]string{"instance", "uri", "db", "collection", "shard_key"},
		),
		collectionShardKeyHashedMetric: NewMetrics(
			"mongodb_metadata_collection_shard_key_hashed",
			"Gauge indicating if the shard key is hashed (1) or not (0).",
			[]string{"instance", "uri", "db", "collection"},
		),
		collectionStorageEngineMetric: NewMetrics(
			"mongodb_metadata_collection_storage_engine",
			"The storage engine used by the collection.",
			[]string{"instance", "uri", "db", "collection"},
		),

		indexCreatedTimestampMetric: NewMetrics(
			"mongodb_metadata_index_created_timestamp",
			"The timestamp when the index was created.",
			[]string{"instance", "uri", "db", "collection", "index_name"},
		),
		indexIsUniqueMetric: NewMetrics(
			"mongodb_metadata_index_unique",
			"Gauge indicating if the index is unique (1) or not (0).",
			[]string{"instance", "uri", "db", "collection", "index_name"},
		),
		indexIsSparseMetric: NewMetrics(
			"mongodb_metadata_index_sparse",
			"Gauge indicating if the index is sparse (1) or not (0).",
			[]string{"instance", "uri", "db", "collection", "index_name"},
		),
		indexIsTTLMetric: NewMetrics(
			"mongodb_metadata_index_ttl",
			"Gauge indicating if the index is a TTL index (1) or not (0).",
			[]string{"instance", "uri", "db", "collection", "index_name"},
		),
		indexIsPartialMetric: NewMetrics(
			"mongodb_metadata_index_partial",
			"Gauge indicating if the index is a partial index (1) or not (0).",
			[]string{"instance", "uri", "db", "collection", "index_name"},
		),
		indexKeyPatternMetric: NewMetrics(
			"mongodb_metadata_index_key_pattern",
			"Gauge indicating the key pattern used by the index.",
			[]string{"instance", "uri", "db", "collection", "index_name", "key_pattern"},
		),
		indexDirectionMetric: NewMetrics(
			"mongodb_metadata_index_direction",
			"Gauge indicating the direction of the index (1=asc, -1=desc).",
			[]string{"instance", "uri", "db", "collection", "index_name"},
		),

		shardingTagRangeMetric: NewMetrics(
			"mongodb_metadata_sharding_tag_range",
			"Gauge indicating the min and max key range for a tag.",
			[]string{"instance", "uri", "zone", "shard", "min", "max"},
		),
		shardingZoneForShardMetric: NewMetrics(
			"mongodb_metadata_sharding_zone_for_shard",
			"Gauge indicating which zone a shard belongs to.",
			[]string{"instance", "uri", "shard", "zone"},
		),
	}
}

// NewMongoDBMetadataCollector 创建一个新的 Metadata Collector
func NewMongoDBMetadataCollector(pool *MongoDBClientPool, instanceName, instanceURI string) *MongoDBMetadataCollector {
	return &MongoDBMetadataCollector{
		pool:         pool,
		instanceName: instanceName,
		instanceURI:  instanceURI,
		metrics:      newMetadataMetrics(),
	}
}

// Describe implements Prometheus Collector interface

// Collect implements Prometheus Collector interface
func (c *MongoDBMetadataCollector) collect(ch chan<- prometheus.Metric) {
	client, _ := c.pool.GetClient()
	ctx := context.Background()
	// adminDB := client.Database("admin")
	configDB := client.Database("config")

	labelsBase := []string{c.instanceName, c.instanceURI}

	// -----------------------------
	// 1. 自动发现数据库元数据
	// -----------------------------
	// dbNames, err := adminDB.ListDatabaseNames(ctx, bson.D{})
	dbNames, err := client.ListDatabaseNames(ctx, bson.D{})
	if err == nil {
		for _, dbName := range dbNames {
			labels := append(labelsBase, dbName)

			// 上报是否是特殊数据库（admin/config/local）
			isAdmin := 0.0
			if dbName == "admin" || dbName == "config" || dbName == "local" {
				isAdmin = 1
			}
			c.metrics.dbAdminInfoMetric.collect(
				ch,
				isAdmin,
				labels,
			)

			// 获取数据库创建时间戳（模拟）
			c.metrics.dbCreatedTimestampMetric.collect(
				ch,
				float64(time.Now().Unix()),
				labels,
			)
		}
	}

	// -----------------------------
	// 2. 自动发现集合元数据
	// -----------------------------
	collCursor, err := configDB.Collection("collections").Find(ctx, bson.D{})
	if err == nil {
		defer collCursor.Close(ctx)
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
			dbName := parts[0]
			collName := parts[1]

			labels := append(append(labelsBase, dbName), collName)

			// 是否启用分片
			c.metrics.collectionIsShardedMetric.collect(
				ch,
				1,
				labels,
			)

			// 分片键
			if key, ok := doc["key"].(bson.M); ok {
				shardKey := extractShardKey(key)
				hashed := isHashedShardKey(key)

				c.metrics.collectionShardKeyMetric.collect(
					ch,
					1,
					append(labels, shardKey),
				)
				c.metrics.collectionShardKeyHashedMetric.collect(
					ch,
					float64(hashed),
					labels,
				)
			}

			// Capped collection
			if options, ok := doc["options"].(bson.M); ok {
				if val, ok := options["capped"].(bool); ok && val {
					if maxSize, ok := options["size"].(int32); ok {
						c.metrics.collectionCappedMetric.collect(
							ch,
							1,
							labels,
						)
						c.metrics.collectionMaxSizeMetric.collect(
							ch,
							float64(maxSize),
							labels,
						)
					}
				}
			}

			// 存储引擎
			if engine, ok := doc["storageEngine"].(bson.M); ok {
				if name, ok := engine["name"].(string); ok {
					c.metrics.collectionStorageEngineMetric.collect(
						ch,
						1,
						append(labels, name),
					)
				}
			}
		}
	}

	// -----------------------------
	// 3. 自动发现索引元数据
	// -----------------------------
	// 示例：查询某个集合的索引信息（可扩展为遍历所有集合）
	testDB := client.Database("test")
	testColl := testDB.Collection("users")

	indexes, err := testColl.Indexes().List(ctx)
	if err == nil {
		for indexes.Next(ctx) {
			var idx bson.M
			err := indexes.Decode(&idx)
			if err != nil {
				continue
			}

			name, _ := idx["name"].(string)
			key, _ := idx["key"].(bson.M)
			unique, _ := idx["unique"].(bool)
			sparse, _ := idx["sparse"].(bool)
			expireAfter, _ := idx["expireAfterSeconds"].(int32)
			partialFilter, _ := idx["partialFilterExpression"].(bson.M)

			labels := append(append(labelsBase, "test", "users"), name)

			// 创建时间（模拟）
			c.metrics.indexCreatedTimestampMetric.collect(
				ch,
				float64(time.Now().Unix()),
				labels,
			)

			// 是否唯一
			c.metrics.indexIsUniqueMetric.collect(
				ch,
				boolToFloat(unique),
				labels,
			)

			// 是否稀疏
			c.metrics.indexIsSparseMetric.collect(
				ch,
				boolToFloat(sparse),
				labels,
			)

			// 是否 TTL
			if expireAfter > 0 {
				c.metrics.indexIsTTLMetric.collect(
					ch,
					1,
					labels,
				)
			} else {
				c.metrics.indexIsTTLMetric.collect(
					ch,
					0,
					labels,
				)
			}

			// 是否 partial
			if len(partialFilter) > 0 {
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

			// 索引字段组合
			for field, dir := range key {
				dirVal := 1.0
				if d, ok := dir.(int32); ok && d == -1 {
					dirVal = -1
				}
				c.metrics.indexKeyPatternMetric.collect(
					ch,
					1, append(labels, fmt.Sprintf("%s:%v", field, dir)))
				c.metrics.indexDirectionMetric.collect(
					ch,
					dirVal,
					labels,
				)
			}
		}
	}

	// -----------------------------
	// 4. 自动发现 Zone 和 Tag 元数据
	// -----------------------------
	tagCursor, err := configDB.Collection("tags").Find(ctx, bson.D{})
	if err == nil {
		defer tagCursor.Close(ctx)
		for tagCursor.Next(ctx) {
			var tagDoc bson.M
			err := tagCursor.Decode(&tagDoc)
			if err != nil {
				continue
			}

			shard, _ := tagDoc["shard"].(string)
			min, _ := tagDoc["min"].(bson.M)
			max, _ := tagDoc["max"].(bson.M)

			// 上报 zone <-> shard <-> min/max 信息
			c.metrics.shardingTagRangeMetric.collect(
				ch,
				1,
				append(labelsBase, "", shard, bsonToString(min), bsonToString(max)),
			)
		}
	}

	// -----------------------------
	// 5. 自动发现 Shard <-> Zone 映射
	// -----------------------------
	shardCursor, err := configDB.Collection("shards").Find(ctx, bson.D{})
	if err == nil {
		defer shardCursor.Close(ctx)
		for shardCursor.Next(ctx) {
			var shardDoc bson.M
			err := shardCursor.Decode(&shardDoc)
			if err != nil {
				continue
			}

			name, _ := shardDoc["_id"].(string)
			zone, _ := getZoneByShard(name)

			if name != "" && zone != "" {
				c.metrics.shardingZoneForShardMetric.collect(
					ch,
					1,
					append(labelsBase, name, zone),
				)
			}
		}
	}
}

// 辅助函数：提取 shard key 字段
func extractShardKey(key bson.M) string {
	for k := range key {
		return k
	}
	return ""
}

// 判断是否是 Hashed 分片键
func isHashedShardKey(key bson.M) int {
	for _, v := range key {
		if sub, ok := v.(bson.M); ok {
			if _, exists := sub["$hashed"]; exists {
				return 1
			}
		}
	}
	return 0
}

// 辅助函数：将 bson.M 转为字符串
func bsonToString(b bson.M) string {
	str := ""
	for k, v := range b {
		str += fmt.Sprintf("%s=%v,", k, v)
	}
	if len(str) > 0 {
		str = str[:len(str)-1]
	}
	return str
}

// 辅助函数：bool -> float64
func boolToFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

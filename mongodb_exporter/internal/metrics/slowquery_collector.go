package metrics

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/bson"
)

// MongoDBSlowQuerycollector 负责采集 MongoDB 的慢查询日志
type MongoDBSlowQuerycollector struct {
	pool         *MongoDBClientPool
	instanceName string
	instanceURI  string
	metrics      *slowQueryMetrics
}

type slowQueryMetrics struct {
	slowQueriesTotalMetric                 *baseMetrics
	slowQueryDurationMicrosecondsHistogram *baseMetrics
	slowQueryDocsExaminedMetric            *baseMetrics
	slowQueryKeysExaminedMetric            *baseMetrics
	slowQueryNReturnedMetric               *baseMetrics
	slowQueryScanAndOrderMetric            *baseMetrics
}

func newSlowQueryMetrics() *slowQueryMetrics {
	return &slowQueryMetrics{
		slowQueriesTotalMetric: NewMetrics(
			"mongodb_slow_queries_total",
			"The total number of slow queries recorded in the system.profile collection.",
			[]string{"instance", "uri", "db", "collection", "op_type"},
		),
		slowQueryDurationMicrosecondsHistogram: NewMetrics(
			"mongodb_slow_query_duration_usec",
			"The duration of each slow query in microseconds.",
			[]string{"instance", "uri", "db", "collection", "op_type"},
		),
		slowQueryDocsExaminedMetric: NewMetrics(
			"mongodb_slow_query_docs_examined",
			"The number of documents examined for this query.",
			[]string{"instance", "uri", "db", "collection", "op_type"},
		),
		slowQueryKeysExaminedMetric: NewMetrics(
			"mongodb_slow_query_keys_examined",
			"The number of keys examined during this query.",
			[]string{"instance", "uri", "db", "collection", "op_type"},
		),
		slowQueryNReturnedMetric: NewMetrics(
			"mongodb_slow_query_n_returned",
			"The number of documents returned by this query.",
			[]string{"instance", "uri", "db", "collection", "op_type"},
		),
		slowQueryScanAndOrderMetric: NewMetrics(
			"mongodb_slow_query_scan_and_order",
			"Whether the query had to scan in order (1) or used an index (0).",
			[]string{"instance", "uri", "db", "collection", "op_type"},
		),
	}
}

// NewMongoDBSlowQuerycollector 创建一个新的 Slow Query collector
func NewMongoDBSlowQuerycollector(pool *MongoDBClientPool, instanceName, instanceURI string) *MongoDBSlowQuerycollector {
	return &MongoDBSlowQuerycollector{
		pool:         pool,
		instanceName: instanceName,
		instanceURI:  instanceURI,
		metrics:      newSlowQueryMetrics(),
	}
}

// Describe implements Prometheus collector interface

// collect implements Prometheus collector interface
func (c *MongoDBSlowQuerycollector) collect(ch chan<- prometheus.Metric) {
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

	// labelsBase := []string{c.instanceName, c.instanceURI}

	for _, dbName := range dbs {
		db := client.Database(dbName)

		// 查询 system.profile 集合
		coll := db.Collection("system.profile")

		// 只获取最近的 N 条慢查询（例如 100 条）
		cursor, err := coll.Find(ctx, bson.D{})
		if err != nil {
			fmt.Printf("Failed to fetch from system.profile in DB %s: %v\n", dbName, err)
			continue
		}
		defer cursor.Close(ctx)

		var count int64 = 0

		for cursor.Next(ctx) {
			var record bson.M
			err := cursor.Decode(&record)
			if err != nil {
				fmt.Printf("Failed to decode profile record: %v\n", err)
				continue
			}

			opType, okOp := record["op"].(string)
			ns, okNs := record["ns"].(string)
			millis, okMillis := record["millis"].(int32)
			_, okTs := record["ts"].(time.Time)

			if !okOp || !okNs || !okMillis || !okTs {
				continue
			}

			// 解析 ns 字段（格式为 <db>.<collection>）
			nsParts := strings.SplitN(ns, ".", 2)
			var collection string
			if len(nsParts) == 2 {
				dbName = nsParts[0]
				collection = nsParts[1]
			} else {
				collection = ""
			}

			labels := []string{c.instanceName, c.instanceURI, dbName, collection, opType}
			count++

			// 记录慢查询数量
			c.metrics.slowQueriesTotalMetric.collect(
				ch,
				float64(count),
				labels,
			)

			// 记录耗时（转换为微秒）
			durationUsec := float64(millis) * 1000
			c.metrics.slowQueryDurationMicrosecondsHistogram.collect(
				ch,
				durationUsec,
				labels,
			)

			// docsExamined
			if queryStats, ok := record["queryExecStats"].(bson.M); ok {
				if docsExamined, ok := queryStats["docsExamined"].(int32); ok {
					c.metrics.slowQueryDocsExaminedMetric.collect(
						ch,
						float64(docsExamined),
						labels,
					)
				}
				if keysExamined, ok := queryStats["keysExamined"].(int32); ok {
					c.metrics.slowQueryKeysExaminedMetric.collect(
						ch,
						float64(keysExamined),
						labels,
					)
				}
				if nReturned, ok := queryStats["nreturned"].(int32); ok {
					c.metrics.slowQueryNReturnedMetric.collect(
						ch,
						float64(nReturned),
						labels,
					)
				}
				if isSorted, ok := queryStats["scanAndOrder"].(bool); ok {
					var val float64 = 0
					if isSorted {
						val = 1
					}
					c.metrics.slowQueryScanAndOrderMetric.collect(
						ch,
						val,
						labels,
					)
				}
			} else if command, ok := record["command"].(bson.M); ok {
				// 如果是命令（如 createIndexes, count 等）
				for k := range command {
					if k == "createIndexes" {
						labels[4] = "create_index"
						break
					}
				}
			}
		}

	}
}

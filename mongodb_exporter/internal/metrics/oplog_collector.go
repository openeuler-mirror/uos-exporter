package metrics

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDBOplogCollector 负责采集 MongoDB 的 Oplog 时间范围和同步延迟信息
type MongoDBOplogCollector struct {
	pool         *MongoDBClientPool
	instanceName string
	instanceURI  string
	metrics      *oplogMetrics
}

type oplogMetrics struct {
	oplogFirstTimestampMetric  *baseMetrics // Oplog 最早的时间戳
	oplogLastTimestampMetric   *baseMetrics // Oplog 最晚的时间戳
	oplogTimeDiffSecondsMetric *baseMetrics // Oplog 总时长（秒）
	oplogSyncLagSecondsMetric  *baseMetrics // 主从同步延迟（秒）
	oplogIsPrimaryMetric       *baseMetrics // 是否是 PRIMARY
	oplogMemberOptimeTimestamp *baseMetrics // 成员最后应用的 op 时间戳
}

func newOplogMetrics() *oplogMetrics {
	return &oplogMetrics{
		oplogFirstTimestampMetric: NewMetrics(
			"mongodb_oplog_first_timestamp",
			"The timestamp of the first operation in the oplog.",
			[]string{"instance", "uri"},
		),
		oplogLastTimestampMetric: NewMetrics(
			"mongodb_oplog_last_timestamp",
			"The timestamp of the last operation in the oplog.",
			[]string{"instance", "uri"},
		),
		oplogTimeDiffSecondsMetric: NewMetrics(
			"mongodb_oplog_time_diff_seconds",
			"The total time range covered by the oplog (last - first).",
			[]string{"instance", "uri"},
		),
		oplogSyncLagSecondsMetric: NewMetrics(
			"mongodb_replset_sync_lag_seconds",
			"The replication lag between primary and secondary in seconds.",
			[]string{"instance", "uri", "member"},
		),
		oplogIsPrimaryMetric: NewMetrics(
			"mongodb_oplog_is_primary",
			"Gauge indicating if this instance is a primary (1) or not (0).",
			[]string{"instance", "uri"},
		),
		oplogMemberOptimeTimestamp: NewMetrics(
			"mongodb_replset_member_optime_timestamp",
			"The timestamp of the last applied operation on this member.",
			[]string{"instance", "uri", "member"},
		),
	}
}

// NewMongoDBOplogCollector 创建一个新的 Oplog Collector
func NewMongoDBOplogCollector(pool *MongoDBClientPool, instanceName, instanceURI string) *MongoDBOplogCollector {
	return &MongoDBOplogCollector{
		pool:         pool,
		instanceName: instanceName,
		instanceURI:  instanceURI,
		metrics:      newOplogMetrics(),
	}
}

// Describe implements Prometheus Collector interface

// Collect implements Prometheus Collector interface
func (c *MongoDBOplogCollector) collect(ch chan<- prometheus.Metric) {
	client, _ := c.pool.GetClient()

	ctx := context.Background()
	adminDB := client.Database("admin")
	localDB := client.Database("local")

	// 判断是否是主节点
	var isPrimary bool = false

	isMasterResult := bson.M{}
	err := adminDB.RunCommand(ctx, bson.D{{Key: "isMaster", Value: 1}}).Decode(&isMasterResult)
	if err == nil {
		if val, ok := isMasterResult["ismaster"].(bool); ok && val {
			isPrimary = true
		}
	}

	labelsBase := []string{c.instanceName, c.instanceURI}
	if isPrimary {
		c.metrics.oplogIsPrimaryMetric.collect(
			ch,
			float64(1),
			labelsBase,
		)

		// 只在主节点上采集 oplog 时间范围
		coll := localDB.Collection("oplog.rs")

		// 获取最早的和最新的记录
		var firstDoc, lastDoc bson.M

		firstErr := coll.FindOne(ctx, bson.D{}, options.FindOne().SetSort(bson.D{{Key: "$natural", Value: 1}})).Decode(&firstDoc)
		lastErr := coll.FindOne(ctx, bson.D{}, options.FindOne().SetSort(bson.D{{Key: "$natural", Value: -1}})).Decode(&lastDoc)

		if firstErr == nil && lastErr == nil {
			firstTs, ok1 := extractTimestampFromOplog(firstDoc)
			lastTs, ok2 := extractTimestampFromOplog(lastDoc)

			if ok1 && ok2 {
				c.metrics.oplogFirstTimestampMetric.collect(
					ch,
					float64(firstTs.Unix()),
					labelsBase,
				)
				c.metrics.oplogLastTimestampMetric.collect(
					ch,
					float64(lastTs.Unix()),
					labelsBase,
				)

				diff := lastTs.Sub(firstTs).Seconds()
				c.metrics.oplogTimeDiffSecondsMetric.collect(
					ch,
					diff,
					labelsBase,
				)
			}
		}
	} else {
		c.metrics.oplogIsPrimaryMetric.collect(
			ch,
			float64(0),
			labelsBase,
		) // 默认值
	}

	// 在所有节点采集复制延迟（仅从节点与主节点比较）
	var status bson.M
	err = adminDB.RunCommand(ctx, bson.D{{Key: "replSetGetStatus", Value: 1}}).Decode(&status)
	if err != nil {
		fmt.Printf("Failed to get replSetGetStatus: %v\n", err)
		return
	}

	members, ok := status["members"].(bson.A)
	if !ok {
		return
	}

	primaryName := ""
	primaryOptime := time.Time{}

	for _, m := range members {
		member, ok := m.(bson.M)
		if !ok {
			continue
		}

		name, _ := member["name"].(string)
		stateStr, _ := member["stateStr"].(string)
		optimeDate, _ := member["optimeDate"].(time.Time)

		labels := append(labelsBase, name)

		// 上报每个成员的 optime 时间戳
		c.metrics.oplogMemberOptimeTimestamp.collect(
			ch,
			float64(optimeDate.Unix()),
			labels,
		)

		// 记录主节点信息
		if strings.ToLower(stateStr) == "primary" {
			primaryName = name
			primaryOptime = optimeDate
		}
	}

	// 在从节点上报与主节点的同步延迟
	if !isPrimary {
		goto done
	}

	// 主节点采集各从节点的 lag
	for _, m := range members {
		member, ok := m.(bson.M)
		if !ok {
			continue
		}

		name, _ := member["name"].(string)
		optimeDate, _ := member["optimeDate"].(time.Time)

		if name == primaryName {
			continue
		}

		lag := primaryOptime.Sub(optimeDate).Seconds()
		labels := append(labelsBase, name)
		c.metrics.oplogSyncLagSecondsMetric.collect(
			ch,
			lag,
			labels,
		)
	}

done:
	return
}

// extractTimestampFromOplog 提取文档中的 ts 字段时间戳
func extractTimestampFromOplog(doc bson.M) (time.Time, bool) {
	tsField, ok := doc["ts"].(primitive.Timestamp)
	if !ok {
		return time.Time{}, false
	}
	ts := time.Unix(int64(tsField.T), 0)
	return ts, true
}

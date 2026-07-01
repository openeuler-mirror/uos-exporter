package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/bson"
)

// MongoDBReplSetcollector 负责采集 MongoDB 复制集相关指标
type MongoDBReplSetcollector struct {
	pool         *MongoDBClientPool
	instanceName string
	instanceURI  string
	metrics      *replSetMetrics
}

type replSetMetrics struct {
	replSetHealthMetric         *baseMetrics // 成员是否健康
	replSetStateMetric          *baseMetrics // 成员状态（PRIMARY, SECONDARY, ARBITER 等）
	replSetUptimeMetric         *baseMetrics // 成员运行时间（秒）
	replSetOptimeDateMetric     *baseMetrics // 最后一次操作时间戳
	replSetLastHeartbeatMetric  *baseMetrics // 上次心跳时间差（秒）
	replSetSyncLagSecondsMetric *baseMetrics // 同步延迟（秒）
	replSetIsPrimaryMetric      *baseMetrics // 是否是 PRIMARY
	replSetIsSecondaryMetric    *baseMetrics // 是否是 SECONDARY
	replSetIsArbiterMetric      *baseMetrics // 是否是 ARBITER
	replSetMembersCountMetric   *baseMetrics // 当前复制集成员数量
}

func newReplSetMetrics() *replSetMetrics {
	return &replSetMetrics{
		replSetHealthMetric: NewMetrics(
			"mongodb_replset_member_health",
			"Gauge indicating if a member is healthy (1) or not (0).",
			[]string{"instance", "uri", "replica_set", "member"},
		),
		replSetStateMetric: NewMetrics(
			"mongodb_replset_member_state",
			"The current state of the member (e.g. PRIMARY, SECONDARY).",
			[]string{"instance", "uri", "replica_set", "member", "state"},
		),
		replSetUptimeMetric: NewMetrics(
			"mongodb_replset_member_uptime_seconds",
			"The uptime of the member in seconds.",
			[]string{"instance", "uri", "replica_set", "member"},
		),
		replSetOptimeDateMetric: NewMetrics(
			"mongodb_replset_member_optime_date",
			"The timestamp of the last operation applied by this member.",
			[]string{"instance", "uri", "replica_set", "member"},
		),
		replSetLastHeartbeatMetric: NewMetrics(
			"mongodb_replset_member_last_heartbeat_seconds_ago",
			"The number of seconds since the last heartbeat from this member.",
			[]string{"instance", "uri", "replica_set", "member"},
		),
		replSetSyncLagSecondsMetric: NewMetrics(
			"mongodb_replset_sync_lag_seconds",
			"The replication lag behind the primary in seconds.",
			[]string{"instance", "uri", "replica_set", "member"},
		),
		replSetIsPrimaryMetric: NewMetrics(
			"mongodb_replset_is_primary",
			"Gauge indicating if this instance is the primary (1) or not (0).",
			[]string{"instance", "uri", "replica_set"},
		),
		replSetIsSecondaryMetric: NewMetrics(
			"mongodb_replset_is_secondary",
			"Gauge indicating if this instance is a secondary (1) or not (0).",
			[]string{"instance", "uri", "replica_set"},
		),
		replSetIsArbiterMetric: NewMetrics(
			"mongodb_replset_is_arbiter",
			"Gauge indicating if this instance is an arbiter (1) or not (0).",
			[]string{"instance", "uri", "replica_set", "member"},
		),
		replSetMembersCountMetric: NewMetrics(
			"mongodb_replset_members_count",
			"The total number of members in the replica set.",
			[]string{"instance", "uri", "replica_set"},
		),
	}
}

// NewMongoDBReplSetcollector 创建一个新的 ReplSet collector
func NewMongoDBReplSetcollector(pool *MongoDBClientPool, instanceName, instanceURI string) *MongoDBReplSetcollector {
	return &MongoDBReplSetcollector{
		pool:         pool,
		instanceName: instanceName,
		instanceURI:  instanceURI,
		metrics:      newReplSetMetrics(),
	}
}

// Describe implements Prometheus collector interface

// collect implements Prometheus collector interface
func (c *MongoDBReplSetcollector) collect(ch chan<- prometheus.Metric) {
	client, _ := c.pool.GetClient()
	ctx := context.Background()
	adminDB := client.Database("admin")

	// 执行 isMaster 获取当前是否属于复制集
	var isMasterResult bson.M
	err := adminDB.RunCommand(ctx, bson.D{{Key: "isMaster", Value: 1}}).Decode(&isMasterResult)
	if err != nil {
		fmt.Printf("Failed to run isMaster command: %v\n", err)
		return
	}

	setName, ok := isMasterResult["setName"].(string)
	if !ok {
		// 不在复制集中，跳过
		return
	}

	labelsBase := []string{c.instanceName, c.instanceURI, setName}

	// 执行 replSetGetStatus 获取详细状态
	var statusResult bson.M
	err = adminDB.RunCommand(ctx, bson.D{{Key: "replSetGetStatus", Value: 1}}).Decode(&statusResult)
	if err != nil {
		fmt.Printf("Failed to run replSetGetStatus: %v\n", err)
		return
	}

	// 获取所有成员
	membersArray, ok := statusResult["members"].(bson.A)
	if !ok {
		fmt.Println("Could not find 'members' array in replSetGetStatus")
		return
	}

	// 获取自己是否为主节点或从节点
	myState, _ := statusResult["myState"].(int32)
	isPrimary := myState == 1
	isSecondary := myState == 2

	// 记录是否为主/从
	c.metrics.replSetIsPrimaryMetric.collect(
		ch,
		boolToFloat64(isPrimary),
		labelsBase,
	)
	c.metrics.replSetIsSecondaryMetric.collect(
		ch,
		boolToFloat64(isSecondary),
		labelsBase,
	)

	// 成员总数
	c.metrics.replSetMembersCountMetric.collect(
		ch,
		float64(len(membersArray)),
		labelsBase,
	)

	// 遍历每个成员
	for _, memberRaw := range membersArray {
		member, ok := memberRaw.(bson.M)
		if !ok {
			continue
		}

		name, ok := member["name"].(string)
		if !ok {
			continue
		}

		stateStr := parseMemberState(member["state"].(int))
		health, ok := member["health"].(float64)
		if !ok {
			health = 0
		}

		uptime, _ := member["uptime"].(int64)
		optime, _ := member["optimeDate"].(time.Time)
		lastHeartbeat, _ := member["lastHeartbeat"].(time.Time)

		labels := []string{c.instanceName, c.instanceURI, setName, name}

		// 成员健康状态
		c.metrics.replSetHealthMetric.collect(
			ch,
			health,
			labels,
		)

		// 成员状态
		c.metrics.replSetStateMetric.collect(
			ch,
			1, append(labels, stateStr))

		// 成员运行时间
		c.metrics.replSetUptimeMetric.collect(
			ch,
			float64(uptime),
			labels,
		)

		// optime 时间戳
		c.metrics.replSetOptimeDateMetric.collect(
			ch,
			float64(optime.Unix()),
			labels,
		)

		// 上次心跳时间差
		if !lastHeartbeat.IsZero() {
			secondsAgo := time.Since(lastHeartbeat).Seconds()
			c.metrics.replSetLastHeartbeatMetric.collect(
				ch,
				secondsAgo,
				labels,
			)
		}

		// 同步延迟（仅从节点）
		if self, ok := statusResult["self"].(bool); ok && !self {
			primaryOptime, ok1 := statusResult["primaryOptimeDate"].(time.Time)
			primaryName, ok2 := statusResult["primaryName"].(string)

			if ok1 && ok2 && primaryOptime.Unix() > 0 {
				syncLag := optime.Sub(primaryOptime).Seconds()
				if syncLag < 0 {
					syncLag = -syncLag
				}
				c.metrics.replSetSyncLagSecondsMetric.collect(
					ch,
					syncLag,
					labels,
				)
			}

			if name == primaryName {
				c.metrics.replSetIsPrimaryMetric.collect(
					ch,
					1,
					labelsBase,
				)
			} else {
				c.metrics.replSetIsPrimaryMetric.collect(
					ch,
					0,
					labelsBase,
				)
			}
		}
	}
}

// parseMemberState 返回 MongoDB 成员状态字符串表示
func parseMemberState(state interface{}) string {
	switch val := state.(type) {
	case int:
		return mapReplSetState(val)
	case int32:
		return mapReplSetState(int(val))
	default:
		return "unknown"
	}
}

func mapReplSetState(state int) string {
	switch state {
	case 0:
		return "startUp"
	case 1:
		return "primary"
	case 2:
		return "secondary"
	case 3:
		return "recovering"
	case 5:
		return "started"
	case 6:
		return "primary_readonly"
	case 7:
		return "arbiter"
	case 8:
		return "down"
	case 9:
		return "rollback"
	case 10:
		return "shunned"
	default:
		return fmt.Sprintf("state_%d", state)
	}
}

// boolToFloat64 辅助函数
func boolToFloat64(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

package metrics

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-redis/redis/v8"

	"github.com/prometheus/client_golang/prometheus"
)

// RedisClient 表示单个客户端连接信息
type RedisClient struct {
	ID             string // 客户端唯一标识符（如 "127.0.0.1:6379"）
	Name           string // 客户端名称（可选）
	Age            int64  // 已连接时长（秒）
	Idle           int64  // 最后一次交互以来的空闲时间（秒）
	Db             string // 当前选择的数据库编号
	Sub            int    // 订阅的频道数
	PubsubChannels int    // Pub/Sub 频道订阅数量
	PubsubPatterns int    // Pub/Sub 模式订阅数量
	Multi          int    // 是否处于事务中（1 = 是）
	QueuedCmds     int    // 事务队列中的命令数
	Status         string // 状态（active / idle / pubsub 等）
}

// RedisClientList 是所有客户端的集合
type RedisClientList []RedisClient

type clientMetrics struct {
	connectedClientsTotalMetric  *baseMetrics
	clientConnectedSecondsMetric *baseMetrics
	clientIdleSecondsMetric      *baseMetrics
	clientStatusGauge            *baseMetrics
	clientPubsubChannelsMetric   *baseMetrics
	clientPubsubPatternsMetric   *baseMetrics
	clientMultiCommandsMetric    *baseMetrics
}

func newClientMetrics() *clientMetrics {
	return &clientMetrics{
		connectedClientsTotalMetric: NewMetrics(
			"redis_client_connected_total",
			"The total number of connected clients.",
			nil,
		),
		clientConnectedSecondsMetric: NewMetrics(
			"redis_client_connected_seconds",
			"The duration the client has been connected to Redis.",
			[]string{"client_id", "ip"},
		),
		clientIdleSecondsMetric: NewMetrics(
			"redis_client_idle_seconds",
			"The time since last interaction with the client.",
			[]string{"client_id", "ip"},
		),
		clientStatusGauge: NewMetrics(
			"redis_client_status",
			"Gauge indicating client status (1 for active, 0 for idle).",
			[]string{"client_id", "ip", "status"},
		),
		clientPubsubChannelsMetric: NewMetrics(
			"redis_client_pubsub_channels",
			"The number of Pub/Sub channels the client is subscribed to.",
			[]string{"client_id", "ip"},
		),
		clientPubsubPatternsMetric: NewMetrics(
			"redis_client_pubsub_patterns",
			"The number of Pub/Sub patterns the client is subscribed to.",
			[]string{"client_id", "ip"},
		),
		clientMultiCommandsMetric: NewMetrics(
			"redis_client_multi_commands",
			"The number of commands in the transaction queue if client is in MULTI state.",
			[]string{"client_id", "ip"},
		),
	}
}

type clientCollector struct {
	client  *redis.Client
	metrics *clientMetrics
}

func NewClientCollector(client *redis.Client) *clientCollector {
	return &clientCollector{
		client:  client,
		metrics: newClientMetrics(),
	}
}

func (c *clientCollector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.Background()
	cmd := c.client.ClientList(ctx)
	result, err := cmd.Result()
	if err != nil {
		fmt.Println("Error fetching CLIENT LIST:", err)
		return
	}

	clients, err := parseClientList(result)
	if err != nil {
		fmt.Println("Error parsing CLIENT LIST:", err)
		return
	}

	c.metrics.connectedClientsTotalMetric.collect(
		ch,
		float64(len(clients)), nil)

	for _, client := range clients {
		labels := []string{client.ID, client.ID} // ip 可从 client.ID 提取
		ip := client.ID

		// redis_client_connected_seconds
		c.metrics.clientConnectedSecondsMetric.collect(
			ch,
			float64(client.Age),
			labels,
		)

		// redis_client_idle_seconds
		c.metrics.clientIdleSecondsMetric.collect(
			ch,
			float64(client.Idle),
			labels,
		)

		// redis_client_status
		var status float64 = 0
		if client.Idle == 0 {
			status = 1
		}
		c.metrics.clientStatusGauge.collect(
			ch,
			status, append(labels, client.Status))

		// redis_client_pubsub_channels
		c.metrics.clientPubsubChannelsMetric.collect(
			ch,
			float64(client.PubsubChannels),
			labels,
		)

		// redis_client_pubsub_patterns
		c.metrics.clientPubsubPatternsMetric.collect(
			ch,
			float64(client.PubsubPatterns),
			labels,
		)

		// redis_client_multi_commands
		c.metrics.clientMultiCommandsMetric.collect(
			ch,
			float64(client.Multi),
			labels,
		)

		_ = ip // 如果你未使用 ip，可以删除这行
	}
}

// 解析 CLIENT LIST 输出
func parseClientList(raw string) (RedisClientList, error) {
	lines := strings.Split(raw, "\n")
	var clients RedisClientList

	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.FieldsFunc(line, func(r rune) bool {
			return r == ' '
		})

		var client RedisClient

		for _, field := range fields {
			kv := strings.Split(field, "=")
			if len(kv) < 2 {
				continue
			}
			key := kv[0]
			val := kv[1]

			switch key {
			case "id":
				client.ID = val
			case "addr":
				client.ID = val // addr 格式：ip:port
			case "age":
				client.Age, _ = strconv.ParseInt(val, 10, 64)
			case "idle":
				client.Idle, _ = strconv.ParseInt(val, 10, 64)
			case "db":
				client.Db = val
			case "sub":
				i, _ := strconv.Atoi(val)
				client.Sub = i
			case "psub":
				i, _ := strconv.Atoi(val)
				client.PubsubPatterns = i
			case "pubsub_patterns":
				i, _ := strconv.Atoi(val)
				client.PubsubPatterns = i
			case "multi":
				i, _ := strconv.Atoi(val)
				client.Multi = i
			case "cmd":
				// cmd=command
				client.Status = val
			}
		}

		// 判断状态是否为活跃
		if client.Idle > 5 {
			client.Status = "idle"
		} else {
			client.Status = "active"
		}

		clients = append(clients, client)
	}

	return clients, nil
}

package metrics

import (
	"context"
	"time"

	redis "github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// RedisSlowLog 表示单条 Slow Log 记录
type RedisSlowLog struct {
	ID         int64
	Timestamp  time.Time     // Unix 时间戳（秒）
	Duration   time.Duration // 执行耗时（微秒）
	Command    string
	ClientIP   string
	ClientName string
}

type slowlogMetrics struct {
	entriesTotalMetric   *baseMetrics
	durationUseHistogram *baseMetrics
}

type slowlogCollector struct {
	client  *redis.Client
	metrics *slowlogMetrics
}

func newSlowlogMetrics() *slowlogMetrics {
	return &slowlogMetrics{
		entriesTotalMetric: NewMetrics(
			"redis_slowlog_total",
			"The total number of slow log entries.",
			nil,
		),
		durationUseHistogram: NewMetrics(
			"redis_slowlog_duration_usec",
			"The duration of each slow log entry in microseconds.",
			[]string{"client_ip", "command"},
		),
	}
}

func newSlowlogCollector(client *redis.Client) *slowlogCollector {
	return &slowlogCollector{
		client:  client,
		metrics: newSlowlogMetrics(),
	}
}

func (c *slowlogCollector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.Background()
	cmd := c.client.SlowLogGet(ctx, 10) // 获取最多 10 条日志
	result, err := cmd.Result()
	if err != nil {
		logrus.Println("Error fetching slowlog:", err)
		return
	}

	var logs []RedisSlowLog
	for _, item := range result {
		log := parseSlowLogItem(item)
		logs = append(logs, log)
	}

	c.metrics.entriesTotalMetric.collect(
		ch,
		float64(len(logs)),
		nil,
	)

	for _, log := range logs {
		labels := []string{log.ClientIP, log.Command}
		c.metrics.durationUseHistogram.collect(
			ch,
			float64(log.Duration),
			labels,
		)
	}
}

func parseSlowLogItem(item redis.SlowLog) RedisSlowLog {
	return RedisSlowLog{
		ID:         item.ID,
		Timestamp:  item.Time,
		Duration:   item.Duration,
		Command:    "",
		ClientIP:   item.ClientAddr,
		ClientName: item.ClientName,
	}
}

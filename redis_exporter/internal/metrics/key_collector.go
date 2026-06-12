package metrics

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus"
)

// RedisKeyStats 表示某个 pattern 下的 key 数量和内存占用
type RedisKeyStats struct {
	Pattern string
	Count   int64
}

// RedisKeyPatternList 是多个 pattern 的 key 统计集合
type RedisKeyPatternList []RedisKeyStats

type keyMetrics struct {
	keyCountByPatternMetric *baseMetrics
	keyTotalMetric          *baseMetrics
}

func newKeyMetrics() *keyMetrics {
	return &keyMetrics{
		keyCountByPatternMetric: NewMetrics(
			"redis_key_count_by_pattern",
			"The number of keys matching a specific pattern.",
			[]string{"pattern"},
		),
		keyTotalMetric: NewMetrics(
			"redis_key_total",
			"The total number of keys in the current database.",
			nil,
		),
	}
}

type keyCollector struct {
	client      *redis.Client
	metrics     *keyMetrics
	patterns    []string // 用户配置的 key pattern 列表
	db          int      // 数据库编号（未在这里使用，由客户端选择）
	scanCount   int64    // 每次 SCAN 扫描的数量
	scanEnabled bool     // 是否启用 scan 模式
	keysEnabled bool     // 是否启用 KEYS 模式（慎用）
}

func NewKeyCollector(client *redis.Client, patterns []string, db int, scanCount int64, scanEnabled, keysEnabled bool) *keyCollector {
	return &keyCollector{
		client:      client,
		metrics:     newKeyMetrics(),
		patterns:    patterns,
		db:          db,
		scanCount:   scanCount,
		scanEnabled: scanEnabled,
		keysEnabled: keysEnabled,
	}
}

func (c *keyCollector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.Background()
	var totalKeys int64

	if c.scanEnabled {
		totalKeys = c.countKeysUsingScan(ctx, "*")
	} else if c.keysEnabled {
		totalKeys = c.countKeysUsingKeys(ctx)
	}

	c.metrics.keyTotalMetric.collect(
		ch,
		float64(totalKeys),
		nil,
	)

	for _, pattern := range c.patterns {
		count := c.countKeysByPattern(ctx, pattern)
		c.metrics.keyCountByPatternMetric.collect(
			ch,
			float64(count),
			[]string{pattern},
		)
	}
}

// 使用 SCAN 命令统计总 key 数量（推荐）
func (c *keyCollector) countKeysUsingScan(ctx context.Context, pattern string) int64 {
	var n int64
	var cursor uint64 = 0

	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, pattern, c.scanCount).Result()
		if err != nil {
			fmt.Printf("Error scanning pattern %q: %v\n", pattern, err)
			break
		}
		n += int64(len(keys))
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return n
}

// 使用 KEYS * 统计 key 总数（慎用）
func (c *keyCollector) countKeysUsingKeys(ctx context.Context) int64 {
	keys, err := c.client.Keys(ctx, "*").Result()
	if err != nil {
		fmt.Println("Error fetching keys:", err)
		return 0
	}
	return int64(len(keys))
}

// 使用 SCAN 统计指定 pattern 的 key 数量
func (c *keyCollector) countKeysByPattern(ctx context.Context, pattern string) int64 {
	var n int64
	var cursor uint64 = 0

	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, pattern, c.scanCount).Result()
		if err != nil {
			fmt.Printf("Error scanning pattern %q: %v\n", pattern, err)
			break
		}
		n += int64(len(keys))
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return n
}

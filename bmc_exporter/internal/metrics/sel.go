package metrics

import (
	"bmc_exporter/internal/ipmi"
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type SELCollector struct {
	mu     sync.Mutex
	client *ipmi.Client

	metrics struct {
		entriesTotal    prometheus.Gauge
		freeSpace       prometheus.Gauge
		countByState    *prometheus.GaugeVec
		countByName     *prometheus.GaugeVec
		latestTimestamp *prometheus.GaugeVec
	}

	lastUpdate time.Time
	cacheTTL   time.Duration
	cachedData struct {
		spaceInfo map[string]float64
		entries   []map[string]string
	}
}

func NewSELCollector(client *ipmi.Client, cacheTTL time.Duration) *SELCollector {
	c := &SELCollector{
		client:   client,
		cacheTTL: cacheTTL,
		metrics: struct {
			entriesTotal    prometheus.Gauge
			freeSpace       prometheus.Gauge
			countByState    *prometheus.GaugeVec
			countByName     *prometheus.GaugeVec
			latestTimestamp *prometheus.GaugeVec
		}{
			entriesTotal: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "sel_logs_count",
				Help: "Current number of log entries in the SEL",
			}),
			freeSpace: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "sel_free_space_bytes",
				Help: "Current free space remaining for new SEL entries",
			}),
			countByState: prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "sel_events_count_by_state",
					Help: "Number of log entries by state",
				},
				[]string{"state"},
			),
			countByName: prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "sel_events_count_by_name",
					Help: "Number of custom log entries by name",
				},
				[]string{"name"},
			),
			latestTimestamp: prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "sel_events_latest_timestamp",
					Help: "Latest timestamp of custom log entries by name",
				},
				[]string{"name"},
			),
		},
	}
	go c.backgroundCollector()
	return c
}

func (c *SELCollector) backgroundCollector() {
	ticker := time.NewTicker(c.cacheTTL)
	defer ticker.Stop()

	for range ticker.C {
		func() {
			c.mu.Lock()
			defer c.mu.Unlock()

			if time.Since(c.lastUpdate) > c.cacheTTL {
				log.Printf("开始定期采集 SEL 数据")
				if err := c.collect(context.Background()); err != nil {
					log.Printf("采集失败: %v", err)
					return
				}
				c.lastUpdate = time.Now()
				log.Printf("数据采集完成，下次采集时间: %v", time.Now().Add(c.cacheTTL))
			}
		}()
	}
}

func (c *SELCollector) Collect(ch chan<- prometheus.Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.updateMetrics(c.cachedData.spaceInfo, c.cachedData.entries)

	ch <- c.metrics.entriesTotal
	ch <- c.metrics.freeSpace
	c.metrics.countByState.Collect(ch)
	c.metrics.countByName.Collect(ch)
	c.metrics.latestTimestamp.Collect(ch)
}

// 实现Prometheus收集器接口
func (c *SELCollector) Describe(ch chan<- *prometheus.Desc) {
	c.metrics.entriesTotal.Describe(ch)
	c.metrics.freeSpace.Describe(ch)
	c.metrics.countByState.Describe(ch)
	c.metrics.countByName.Describe(ch)
	c.metrics.latestTimestamp.Describe(ch)
}

func (c *SELCollector) collect(ctx context.Context) error {

	// 获取SEL信息
	infoOutput, err := c.client.Execute(ctx, "sel info")
	if err != nil {
		return fmt.Errorf("获取SEL信息失败: %w", err)
	}

	// 获取SEL条目列表
	listOutput, err := c.client.GetSELList(ctx)
	if err != nil {
		return fmt.Errorf("获取SEL列表失败: %w", err)
	}

	// 解析数据
	spaceInfo := parseSELInfo(infoOutput)
	entries := parseSELList(listOutput)

	log.Printf("开始采集 SEL 数据")
	defer func() {
		log.Printf("SEL 指标更新完成: count=%d", len(entries))
	}()

	// 更新指标
	c.updateMetrics(spaceInfo, entries)
	return nil
}
func (c *SELCollector) updateMetrics(spaceInfo map[string]float64, entries []map[string]string) {

	if total, ok := spaceInfo["entries"]; ok {
		c.metrics.entriesTotal.Set(total)
	}
	if free, ok := spaceInfo["free_space"]; ok {
		c.metrics.freeSpace.Set(free)
	}
	if overflow, ok := spaceInfo["overflow"]; ok {
		c.metrics.countByState.WithLabelValues("overflow").Set(overflow)
	}

	// 使用临时映射避免并发问题
	tmpCounts := make(map[string]float64)
	tmpTimestamps := make(map[string]float64)

	for _, entry := range entries {
		eventType := classifyEvent(entry["event"])
		tmpCounts[eventType]++

		if ts := parseTimestamp(entry["timestamp"]); ts > 0 {
			if ts > tmpTimestamps[eventType] {
				tmpTimestamps[eventType] = ts
			}
		}
	}

	// 原子更新指标
	for event, count := range tmpCounts {
		c.metrics.countByName.WithLabelValues(event).Set(count)
	}
	for event, ts := range tmpTimestamps {
		c.metrics.latestTimestamp.WithLabelValues(event).Set(ts)
	}
}

func classifyEvent(event string) string {
	// 事件分类逻辑
	if strings.Contains(event, "Temperature") {
		return "temperature"
	}
	if strings.Contains(event, "Fan") {
		return "fan"
	}
	if strings.Contains(event, "Power") {
		return "power"
	}
	if strings.Contains(event, "Memory") {
		return "memory"
	}
	return "other"
}

func parseSELInfo(raw string) map[string]float64 {
	info := make(map[string]float64)

	// 增强版正则表达式（支持单位和特殊字段）
	re := regexp.MustCompile(`(?mi)^([\w\s]+)\s*:\s+([^\n]+)`)

	for _, match := range re.FindAllStringSubmatch(raw, -1) {
		key := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(match[1]), " ", "_"))
		valueStr := strings.TrimSpace(match[2])

		// 特殊字段处理
		switch key {
		case "free_space":
			// 提取数字部分（如 "9180 bytes" → 9180）
			if val, err := strconv.ParseFloat(strings.Fields(valueStr)[0], 64); err == nil {
				info[key] = val
			}
		case "percent_used":
			// 处理百分比（如 "84%" → 84.0）
			if val, err := strconv.ParseFloat(strings.TrimSuffix(valueStr, "%"), 64); err == nil {
				info[key] = val
			}
		case "overflow":
			// 布尔值转数字（false→0, true→1）
			info[key] = 0
			if valueStr == "true" {
				info[key] = 1
			}
		case "entries", "alloc_unit_size", "max_record_size":
			// 直接解析数字
			if val, err := strconv.ParseFloat(valueStr, 64); err == nil {
				info[key] = val
			}
		case "supported_cmds":
			// 统计支持的命令数量（可选）
			cmds := strings.Split(strings.Trim(valueStr, "' "), "' '")
			info["supported_commands_count"] = float64(len(cmds))
		default:
			// 其他数值型字段通用处理
			if val, err := strconv.ParseFloat(valueStr, 64); err == nil {
				info[key] = val
			}
		}
	}
	return info
}
func parseSELList(raw string) []map[string]string {
	var entries []map[string]string

	// 修改后的正则表达式（支持新版输出格式）
	re := regexp.MustCompile(`(?mi)^\s*([^\|]+)\s+\|\s+([^\|]+)\s+\|\s+([^\|]+)\s+\|\s+([^\|]+)\s+\|\s+([^\|]+)\s+\|\s+([^\|]+)`)

	for _, match := range re.FindAllStringSubmatch(raw, -1) {
		if len(match) < 7 {
			continue
		}

		entry := map[string]string{
			"id":     strings.TrimSpace(match[1]),
			"date":   strings.TrimSpace(match[2]),
			"time":   strings.TrimSpace(match[3]),
			"sensor": strings.TrimSpace(match[4]),
			"event":  strings.TrimSpace(match[5]),
			"state":  strings.TrimSpace(match[6]),
		}

		// 合并日期时间字段
		entry["timestamp"] = fmt.Sprintf("%s | %s", entry["date"], entry["time"])

		entries = append(entries, entry)
	}
	return entries
}

func parseTimestamp(timestamp string) float64 {
	// 时间解析逻辑示例：02/13/2024 | 15:30:45
	layout := "01/02/2006 | 15:04:05"
	t, err := time.Parse(layout, timestamp)
	if err != nil {
		return 0
	}
	return float64(t.Unix())
}

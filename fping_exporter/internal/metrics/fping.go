package metrics

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"

	"fping_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

var (
	fpingCollectorInstance *FpingCollector
	globalConfig           *exporter.Config
)

// SetConfig 设置全局配置
func SetConfig(config *exporter.Config) {
	globalConfig = config
	if fpingCollectorInstance != nil {
		fpingCollectorInstance.updateTargets(config.FpingTargets)
	}
}

// InitFpingCollector 初始化 FpingCollector，如果配置可用则使用配置中的 targets
func InitFpingCollector() {
	targets := []string{
		"1.1.1.1",
		"127.0.0.1",
		"github.com",
	}

	// 如果有全局配置，使用配置中的 targets
	if globalConfig != nil && len(globalConfig.FpingTargets) > 0 {
		targets = globalConfig.FpingTargets
	}

	fpingCollectorInstance = NewFpingCollector(targets)
	exporter.Register(fpingCollectorInstance)
}

func init() {
	// 先创建一个空的 collector，稍后会通过 SetConfig 更新
	InitFpingCollector()
}

// FpingCollector 定义用于收集 fping 指标的结构体
type FpingCollector struct {
	metrics map[string]*baseMetrics
	targets []string
}

// NewFpingCollector 创建一个新的 FpingCollector
func NewFpingCollector(targets []string) *FpingCollector {
	collector := &FpingCollector{
		targets: targets,
		metrics: make(map[string]*baseMetrics),
	}

	// 初始化所有指标
	collector.metrics["sent_count"] = NewMetrics(
		"fping_sent_count",
		"Number of sent probes",
		[]string{"target"},
	)
	collector.metrics["lost_count"] = NewMetrics(
		"fping_lost_count",
		"Number of lost probes",
		[]string{"target"},
	)
	collector.metrics["rtt_count"] = NewMetrics(
		"fping_rtt_count",
		"Number of measured latencies (successful probes)",
		[]string{"target"},
	)
	collector.metrics["rtt_sum"] = NewMetrics(
		"fping_rtt_sum",
		"Sum of measured latencies",
		[]string{"target"},
	)
	collector.metrics["rtt_avg"] = NewMetrics(
		"fping_rtt",
		"Average of measured latencies",
		[]string{"target"},
	)

	return collector
}

// updateTargets 更新 targets 列表
func (c *FpingCollector) updateTargets(targets []string) {
	c.targets = targets
	logrus.Infof("Updated fping targets: %v", targets)
}

func (c *FpingCollector) Collect(ch chan<- prometheus.Metric) {
	// 调用 fping 并解析输出
	results, err := c.runFping()
	if err != nil {
		logrus.Errorf(
			"Error running fping: %v",
			err,
		)
		return
	}

	// 更新 Prometheus 指标
	for target, stats := range results {
		labels := []string{target}

		c.collectMetric(
			ch,
			"sent_count",
			float64(stats.Sent),
			labels,
		)
		c.collectMetric(
			ch, "lost_count",
			float64(stats.Lost),
			labels,
		)
		c.collectMetric(
			ch, "rtt_count",
			float64(stats.RTTCount),
			labels,
		)
		c.collectMetric(
			ch, "rtt_sum",
			stats.RTTSum,
			labels,
		)
		avgRTT := 0.0
		if stats.RTTCount > 0 {
			avgRTT = stats.RTTSum / float64(stats.RTTCount)
		}
		c.collectMetric(
			ch,
			"rtt_avg",
			avgRTT,
			labels,
		)
	}
}

// 辅助函数用于收集指标
func (c *FpingCollector) collectMetric(ch chan<- prometheus.Metric, metricName string, value float64, labels []string) {
	if metric, ok := c.metrics[metricName]; ok {
		metric.collect(
			ch,
			value,
			labels,
		)
	} else {
		logrus.Errorf(
			"Unknown metric: %s",
			metricName,
		)
	}
}

// runFping 执行 fping 并解析输出
func (c *FpingCollector) runFping() (map[string]*FpingStats, error) {
	// 验证 targets 合法性，防止命令注入
	for _, target := range c.targets {
		if !isValidTarget(target) {
			return nil, fmt.Errorf(
				"invalid target: %s",
				target,
			)
		}
	}

	cmd := exec.Command( // #nosec G204 - cmd is trusted
		"fping",
		append([]string{"-C", "3", "-q"}, c.targets...)...)
	out, err := cmd.CombinedOutput()

	// fping 在有主机无法到达时会返回非零退出状态，这是正常的
	// 只有当完全无法执行命令时才认为是错误
	if err != nil {
		// 检查是否是执行错误（如命令不存在）而不是退出状态错误
		if _, ok := err.(*exec.ExitError); !ok {
			return nil, fmt.Errorf(
				"fping execution failed: %v, output: %s",
				err,
				string(out),
			)
		}
		// 如果是退出状态错误但有输出，继续处理
		logrus.Debugf("fping exit with non-zero status (expected when some hosts are unreachable): %v", err)
	}

	results := make(map[string]*FpingStats)
	reader := bytes.NewReader(out)
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 || parts[0] == "" {
			continue
		}

		target := parts[0]
		stats := &FpingStats{
			Sent: len(parts) - 1, // 总发送数
		}
		for _, part := range parts[1:] {
			if part == "-" {
				stats.Lost++
			} else {
				rtt, err := strconv.ParseFloat(part, 64)
				if err == nil {
					stats.RTTCount++
					stats.RTTSum += rtt
				}
			}
		}

		results[target] = stats
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf(
			"scanner error: %v",
			err,
		)
	}

	return results, nil
}

// isValidTarget 检查目标字符串是否为有效的 IP 地址或主机名
// 参数：
// - target: 目标字符串
// 返回值：
// - bool: 如果目标字符串有效则返回 true，否则返回 false
func isValidTarget(target string) bool {
	// 检查空字符串
	if len(target) == 0 {
		return false
	}

	// 去除前后空白字符
	target = strings.TrimSpace(target)

	// 检查是否为有效的 IP 地址（IPv4 或 IPv6）
	if net.ParseIP(target) != nil {
		return true
	}

	// 检查是否为有效的主机名
	// 简单的主机名检查，可以根据需求进一步增强
	if isHostnameValid(target) {
		return true
	}

	return false
}

// isHostnameValid 检查字符串是否为有效的主机名
// 这里仅做简单的主机名检查，可以根据需求进一步增强
func isHostnameValid(hostname string) bool {
	if len(hostname) > 255 {
		return false
	}

	// 不允许以连字符开头或结尾
	if strings.HasPrefix(hostname, "-") ||
		strings.HasSuffix(hostname, "-") {
		return false
	}

	// 分割主机名部分
	parts := strings.Split(hostname, ".")
	for _, part := range parts {
		if len(part) == 0 || len(part) > 63 {
			return false
		}
		for _, char := range part {
			if !((char >= 'a' && char <= 'z') ||
				(char >= 'A' && char <= 'Z') ||
				(char >= '0' && char <= '9') ||
				char == '-') {
				return false
			}
		}
	}

	return true
}

// FpingStats 定义 fping 的统计结果
type FpingStats struct {
	Sent     int     // 发送的探测数
	Lost     int     // 丢失的探测数
	RTTCount int     // 成功探测数
	RTTSum   float64 // 延迟总和
}
// Final commit for fping_exporter/internal/metrics/fping.go

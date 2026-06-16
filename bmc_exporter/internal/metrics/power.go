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

// region 电源健康状态常量
const (
	HealthOK    = 0
	HealthWarn  = 1
	HealthError = 2
)

// endregion

type PowerCollector struct {
	mu               sync.RWMutex
	client           *ipmi.Client
	refreshInterval  time.Duration
	voltageThreshold struct {
		min float64
		max float64
	}
	metrics struct {
		// 基础指标
		currentPower  prometheus.Gauge
		avgPower      prometheus.Gauge
		maxPower      prometheus.Gauge
		energyCounter prometheus.Counter

		// 新增健康指标
		healthStatus  *prometheus.GaugeVec
		inputVoltage  prometheus.Gauge
		outputCurrent prometheus.Gauge

		// 历史趋势
		powerHistogram prometheus.Histogram
	}
}

func (c *PowerCollector) Collect(ch chan<- prometheus.Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.collect(context.Background()); err != nil {
		log.Printf("采集失败: %v", err)
		c.metrics.healthStatus.WithLabelValues("collector").Set(HealthError)
	} else {
		c.metrics.healthStatus.WithLabelValues("collector").Set(HealthOK)
	}

	// 收集所有指标
	c.metrics.currentPower.Collect(ch)
	c.metrics.avgPower.Collect(ch)
	c.metrics.maxPower.Collect(ch)
	c.metrics.energyCounter.Collect(ch)
	c.metrics.healthStatus.Collect(ch)
	c.metrics.inputVoltage.Collect(ch)
	c.metrics.powerHistogram.Collect(ch)
}

// region 数据采集逻辑扩展
func (c *PowerCollector) collect(ctx context.Context) error {

	output, err := c.client.GetPowerMetrics(ctx)
	if err != nil {
		return fmt.Errorf("failed to collect power metrics: %w", err)
	}

	// 解析基础指标
	if err := c.parseBasicMetrics(output); err != nil {
		return err
	}

	// 解析健康状态
	if err := c.parseHealthStatus(output); err != nil {
		return err
	}

	return nil
}

func (c *PowerCollector) parseBasicMetrics(output string) error {
	re := regexp.MustCompile(`(?im)(\w+\s*[\w\s]+):\s*([\d.]+)\s*(\w+)`)
	matches := re.FindAllStringSubmatch(output, -1)

	for _, match := range matches {
		if len(match) != 4 {
			continue
		}

		name := strings.TrimSpace(match[1])
		value, err := strconv.ParseFloat(match[2], 64)
		if err != nil {
			continue
		}

		switch {
		case strings.Contains(name, "Instantaneous"):
			c.metrics.currentPower.Set(value)
		case strings.Contains(name, "Average"):
			c.metrics.avgPower.Set(value)
		case strings.Contains(name, "Maximum"):
			c.metrics.maxPower.Set(value)
		case strings.Contains(name, "Input Voltage"):
			c.metrics.inputVoltage.Set(value)
			c.checkVoltageSafety(value)
		}
	}

	return nil
}

// 新增初始化方法

// TODO: implement functions

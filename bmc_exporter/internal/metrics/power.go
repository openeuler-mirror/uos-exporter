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
func NewPowerCollector(client *ipmi.Client) *PowerCollector {
	pc := &PowerCollector{
		client: client,
	}

	// 初始化默认配置
	pc.refreshInterval = 30 * time.Second
	pc.voltageThreshold.min = 200
	pc.voltageThreshold.max = 240

	// 初始化指标
	pc.metrics.currentPower = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "power_watts",
		Help: "Current power consumption (watts)",
	})

	pc.metrics.avgPower = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "power_avg_watts",
		Help: "Average power consumption (watts)",
	})

	pc.metrics.maxPower = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "power_max_watts",
		Help: "Max power consumption (watts)",
	})

	pc.metrics.energyCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "energy_kwh_total",
		Help: "Total energy consumption (kWh)",
	})

	pc.metrics.healthStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "power_health_status",
			Help: "Power health status (0=OK, 1=Warning, 2=Critical)",
		},
		[]string{"component"},
	)

	pc.metrics.inputVoltage = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "power_input_voltage",
			Help: "Input voltage (volts)",
		},
	)

	pc.metrics.powerHistogram = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "power_watts_distribution",
			Help:    "Histogram of power consumption distribution",
			Buckets: []float64{100, 200, 300, 400, 500},
		},
	)

	return pc
}

func (c *PowerCollector) parseHealthStatus(output string) error {
	// 解析电源模块状态
	reModule := regexp.MustCompile(`Power Supply\s+\|\s+(\w+)`)
	moduleMatches := reModule.FindAllStringSubmatch(output, -1)
	for _, match := range moduleMatches {
		status := parseHealthState(match[1])
		c.metrics.healthStatus.WithLabelValues("module_" + match[1]).Set(float64(status))
	}

	// 解析全局健康状态
	reGlobal := regexp.MustCompile(`Overall Health\s+\:\s+(\w+)`)
	if matches := reGlobal.FindStringSubmatch(output); len(matches) > 1 {
		status := parseHealthState(matches[1])
		c.metrics.healthStatus.WithLabelValues("global").Set(float64(status))
	}
	return nil
}

func parseHealthState(state string) int {
	switch strings.ToLower(state) {
	case "ok", "present":
		return HealthOK
	case "warning", "non-critical":
		return HealthWarn
	case "critical", "absent", "failure":
		return HealthError
	default:
		return HealthError
	}
}

func (c *PowerCollector) checkVoltageSafety(currentVoltage float64) {
	status := HealthOK
	switch {
	case currentVoltage < c.voltageThreshold.min:
		status = HealthWarn
		log.Printf("Low voltage warning: %.1fV < minimum threshold%.1fV",
			currentVoltage, c.voltageThreshold.min)
	case currentVoltage > c.voltageThreshold.max:
		status = HealthError
		log.Printf("High voltage critical: %.1fV > maximum threshold%.1fV",
			currentVoltage, c.voltageThreshold.max)
	}
	c.metrics.healthStatus.WithLabelValues("voltage").Set(float64(status))
}

// 实现Prometheus收集器接口
func (c *PowerCollector) Describe(ch chan<- *prometheus.Desc) {
	c.metrics.currentPower.Describe(ch)
	c.metrics.avgPower.Describe(ch)
	c.metrics.maxPower.Describe(ch)
	c.metrics.energyCounter.Describe(ch)
	c.metrics.healthStatus.Describe(ch)
	c.metrics.inputVoltage.Describe(ch)
	c.metrics.powerHistogram.Describe(ch)
}

func (c *PowerCollector) CollectMetrics(ch chan<- prometheus.Metric) {
	if err := c.collect(context.Background()); err != nil {
		log.Printf("Power metrics collection failed: %v", err)
		return
	}

	c.metrics.currentPower.Collect(ch)
	c.metrics.avgPower.Collect(ch)
	c.metrics.maxPower.Collect(ch)
	c.metrics.energyCounter.Collect(ch)
}

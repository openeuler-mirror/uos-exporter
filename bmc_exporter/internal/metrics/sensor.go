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

type SensorCollector struct {
	mu       sync.Mutex
	client   *ipmi.Client
	sensor   SensorMetrics
	cacheTTL time.Duration

	lastUpdate time.Time
	cachedData struct {
		fanMetrics  map[string]float64
		tempMetrics map[string]float64
		psuMetrics  map[string]float64
	}
}

type SensorMetrics struct {

	// 风扇指标
	fanSpeed      *prometheus.GaugeVec
	fanSpeedRatio *prometheus.GaugeVec
	fanSpeedState *prometheus.GaugeVec

	cpuTemp       *prometheus.GaugeVec
	psuVoltage    *prometheus.GaugeVec
	psuCurrent    *prometheus.GaugeVec
	psuPower      *prometheus.GaugeVec
	componentTemp *prometheus.GaugeVec
}

func NewSensorMetric() *SensorMetrics {
	return &SensorMetrics{
		fanSpeed: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "fan_speed_rpm",
				Help: "Fan speed in revolutions per minute (RPM).",
			},
			[]string{"fan", "location"}, // location=front/rear
		),
		fanSpeedRatio: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "fan_speed_ratio",
				Help: "Fan speed ratio.",
			},
			[]string{"fan", "location"},
		),
		fanSpeedState: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "fan_speed_state",
				Help: "Fan status (0=Normal, 1=Warning, 2=Critical).",
			},
			[]string{"fan", "location"},
		),
		cpuTemp: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "cpu_temperature_celsius",
				Help: "CPU temperature in degrees Celsius.",
			},
			[]string{"cpu", "sensor_type"},
		),
		psuVoltage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "psu_voltage_volts",
				Help: "PSU voltage in volts.",
			},
			[]string{"psu", "type"}, // type=vin/vout
		),
		psuCurrent: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "psu_current_amperes",
				Help: "PSU current in amperes",
			},
			[]string{"psu", "type"},
		),
		psuPower: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "psu_power_watts",
				Help: "PSU power in watts",
			},
			[]string{"psu", "type"}, // type=pin/pout
		),
		componentTemp: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "component_temperature_celsius",
				Help: "Component temperature in degrees Celsius",
			},
			[]string{"component", "type"}, // 组件类型/传感器类型
		),
	}

}

func NewSensorCollector(client *ipmi.Client, cacheTTL time.Duration) *SensorCollector {
	c := &SensorCollector{
		client:   client,
		sensor:   *NewSensorMetric(),
		cacheTTL: cacheTTL,
	}
	go c.backgroundCollector()
	return c
}

func (c *SensorCollector) backgroundCollector() {
	ticker := time.NewTicker(c.cacheTTL)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		if time.Since(c.lastUpdate) > c.cacheTTL {
			// 在锁保护下执行Scrape
			if err := c.Scrape(); err != nil {
				log.Printf("采集失败: %v", err)
			}
			c.lastUpdate = time.Now() // 更新时间戳应在锁保护内
		}
		c.mu.Unlock()
	}
}

func (c *SensorCollector) Describe(ch chan<- *prometheus.Desc) {
	c.sensor.fanSpeed.Describe(ch)
	c.sensor.fanSpeedRatio.Describe(ch)
	c.sensor.fanSpeedState.Describe(ch)
	c.sensor.cpuTemp.Describe(ch)
	c.sensor.psuVoltage.Describe(ch)
	c.sensor.psuCurrent.Describe(ch)
	c.sensor.psuPower.Describe(ch)
	c.sensor.componentTemp.Describe(ch)
}

func (c *SensorCollector) Collect(ch chan<- prometheus.Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 从缓存更新指标值
	c.updateMetricsFromCache()

	c.sensor.fanSpeed.Collect(ch)
	c.sensor.fanSpeedRatio.Collect(ch)
	c.sensor.fanSpeedState.Collect(ch)
	c.sensor.cpuTemp.Collect(ch)
	c.sensor.psuVoltage.Collect(ch)
	c.sensor.psuCurrent.Collect(ch)
	c.sensor.psuPower.Collect(ch)
	c.sensor.componentTemp.Collect(ch)
}

var (
	sensorRegex = regexp.MustCompile(`^([\w_]+)\s+\|.*\|\s+(\w+)\s+\|.*\|\s+([\d.]+)\s+(.*)$`)
	cpuTempRe   = regexp.MustCompile(`CPU(\d+)_(Temp|DTS|VR_Temp|DIMM_T)`)
	psuRe       = regexp.MustCompile(`PSU(\d+)_([A-Z]+)`)
	fanRe       = regexp.MustCompile(`FAN(\d+)_Speed_([FR])`)
	componentRe = regexp.MustCompile(`([A-Z]+)_(Temp|MAX_TEMP|Thermal)`)
	locationRe  = regexp.MustCompile(`(Inlet|Outlet|Front|Rear|PSU|HDD)`)
)

func (c *SensorCollector) updateMetricsFromCache() {
	// 风扇指标
	for key, val := range c.cachedData.fanMetrics {
		parts := strings.Split(key, "|")
		c.sensor.fanSpeed.WithLabelValues(parts[0], parts[1]).Set(val)
	}

	// CPU温度
	for key, val := range c.cachedData.tempMetrics {
		parts := strings.Split(key, "|")
		c.sensor.cpuTemp.WithLabelValues(parts[0], parts[1]).Set(val)
	}

	// PSU指标
	for key, val := range c.cachedData.psuMetrics {
		parts := strings.Split(key, "|")
		c.sensor.psuPower.WithLabelValues(parts[0], parts[1]).Set(val)
	}
}

// 在SensorCollector中添加处理方法
func (c *SensorCollector) processSensor(line string,
	tmpFan map[string]float64,
	tmpTemp map[string]float64,
	tmpPsu map[string]float64) {

	matches := sensorRegex.FindStringSubmatch(line)
	if len(matches) < 5 {
		return
	}

	name, status, valueStr, unit := matches[1], matches[2], matches[3], matches[4]
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return
	}

	// 统一处理逻辑
	switch {
	// 处理风扇指标
	case fanRe.MatchString(name):
		if fanMatch := fanRe.FindStringSubmatch(name); len(fanMatch) > 2 {
			fanID := fanMatch[1]
			location := "front"
			if fanMatch[2] == "R" {
				location = "rear"
			}
			key := fmt.Sprintf("%s|%s", fanID, location)
			tmpFan[key] = value

			// 状态转换逻辑复用
			stateValue := 0.0
			switch status {
			case "ok":
				stateValue = 0
			case "ns":
				stateValue = 2
			default:
				stateValue = 1
			}
			c.sensor.fanSpeedState.WithLabelValues(fanID, location).Set(stateValue)
		}

	// 处理PSU指标
	case psuRe.MatchString(name):
		if psuMatch := psuRe.FindStringSubmatch(name); len(psuMatch) > 2 {
			psuID := psuMatch[1]
			metricType := strings.ToLower(psuMatch[2])
			key := fmt.Sprintf("%s|%s", psuID, metricType)

			switch {
			case strings.Contains(unit, "Volts"):
				tmpPsu[key] = value
			case strings.Contains(unit, "Amps"):
				tmpPsu[key] = value
			case strings.Contains(unit, "Watts"):
				tmpPsu[key] = value
			}
		}

	// 处理温度指标
	default:
		if cpuMatch := cpuTempRe.FindStringSubmatch(name); len(cpuMatch) > 2 {
			cpuID := cpuMatch[1]
			sensorType := strings.ToLower(cpuMatch[2])
			key := fmt.Sprintf("%s|%s", cpuID, sensorType)
			tmpTemp[key] = value
		} else if compMatch := componentRe.FindStringSubmatch(name); len(compMatch) > 1 {
			component := strings.ToLower(compMatch[1])
			location := "unknown"

			// 解析位置信息
			if locMatch := locationRe.FindStringSubmatch(name); len(locMatch) > 1 {
				location = strings.ToLower(locMatch[1])
			}

			// 特殊处理已知组件
			switch component {
			case "pch":
				component = "platform_controller_hub"
			case "hdd":
				component = "hard_drive"
			case "nvme":
				component = "nvme_ssd"
			}

			c.sensor.componentTemp.WithLabelValues(component, location).Set(value)
			return
		}
	}
}

// 添加数据采集入口方法
func (c *SensorCollector) Scrape() error {
	output, err := c.client.GetSensorData(context.Background())
	if err != nil {
		return err
	}

	tmpFan := make(map[string]float64)
	tmpTemp := make(map[string]float64)
	tmpPsu := make(map[string]float64)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		c.processSensor(line, tmpFan, tmpTemp, tmpPsu)
	}

	// 移除锁操作（由调用方保证线程安全）
	c.cachedData.fanMetrics = tmpFan
	c.cachedData.tempMetrics = tmpTemp
	c.cachedData.psuMetrics = tmpPsu
	c.lastUpdate = time.Now()
	return nil
}

package metrics

import (
	"bmc_exporter/internal/ipmi"
	"context"
	"log"
	"regexp"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type ChassisCollector struct {
	client  *ipmi.Client
	metrics struct {
		powerStatus      prometheus.Gauge // 电源状态(0=off,1=on)
		powerOverload    prometheus.Gauge // 电源过载状态
		coolingFault     prometheus.Gauge // 冷却状态
		driverFault      prometheus.Gauge // 驱动器状态
		chassisIntrusion prometheus.Gauge // 机箱入侵状态
		mainPowerFault   prometheus.Gauge // 新增主电源故障
		powerCtrlFault   prometheus.Gauge // 新增电源控制故障
	}
	mu sync.Mutex
}

func NewChassisCollector(client *ipmi.Client) *ChassisCollector {
	return &ChassisCollector{
		client: client,
		metrics: struct {
			powerStatus      prometheus.Gauge
			powerOverload    prometheus.Gauge
			coolingFault     prometheus.Gauge
			driverFault      prometheus.Gauge
			chassisIntrusion prometheus.Gauge
			mainPowerFault   prometheus.Gauge
			powerCtrlFault   prometheus.Gauge
		}{
			powerStatus: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "chassis_power_status",
				Help: "Current power status (1=On, 0=Off)",
			}),
			powerOverload: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "chassis_power_overload",
				Help: "Power overload status (1=Overload, 0=Normal)",
			}),
			coolingFault: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "chassis_cooling_fault",
				Help: "Cooling system fault status (1=Fault, 0=Normal)",
			}),
			driverFault: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "chassis_drive_fault",
				Help: "Drive bay fault status (1=Fault, 0=Normal)",
			}),
			chassisIntrusion: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "chassis_intrusion_status",
				Help: "Chassis intrusion detection (1=Breached, 0=Normal)",
			}),
			mainPowerFault: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "chassis_main_power_fault",
				Help: "Main power fault status (1=Fault, 0=Normal)",
			}),
			powerCtrlFault: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "chassis_power_ctrl_fault",
				Help: "Power control fault status (1=Fault, 0=Normal)",
			}),
		},
	}
}

type chassisStatus struct {
	power          float64
	powerOverload  float64
	cooling        float64
	drives         float64
	intrusion      float64
	mainPowerFault float64
	powerCtrlFault float64
}

func parseChassisStatus(raw string) chassisStatus {
	status := chassisStatus{}
	re := regexp.MustCompile(`(?mi)^([\w/\s-]+)\s*:\s*(\w+)`) // 扩展正则表达式匹配字符

	for _, match := range re.FindAllStringSubmatch(raw, -1) {
		key := strings.TrimSpace(match[1])
		value := strings.ToLower(strings.TrimSpace(match[2]))

		switch key {
		case "System Power":
			status.power = parseBoolState(value, "on")
		case "Power Overload":
			status.powerOverload = parseBoolState(value, "true")
		case "Cooling/Fan Fault":
			status.cooling = parseBoolState(value, "true") // 故障时值为true，取反处理
		case "Drive Fault":
			status.drives = parseBoolState(value, "true") // 故障时值为true，取反处理
		case "Chassis Intrusion":
			status.intrusion = parseBoolState(value, "active")
		case "Main Power Fault":
			status.mainPowerFault = parseBoolState(value, "true") // 故障时值为true，取反处理
		case "Power Control Fault":
			status.powerCtrlFault = parseBoolState(value, "true") // 故障时值为true，取反处理
		}
	}
	return status
}

func parseBoolState(value string, trueValue string) float64 {
	if strings.ToLower(value) == trueValue {
		return 1 // 故障状态
	}
	return 0 // 正常状态
}

func (c *ChassisCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.metrics.powerStatus.Desc()
	ch <- c.metrics.powerOverload.Desc()
	ch <- c.metrics.coolingFault.Desc()
	ch <- c.metrics.driverFault.Desc()
	ch <- c.metrics.chassisIntrusion.Desc()
	ch <- c.metrics.mainPowerFault.Desc()
	ch <- c.metrics.powerCtrlFault.Desc()
}

func (c *ChassisCollector) collect(ctx context.Context) error {
	output, err := c.client.Execute(ctx, "chassis status")
	if err != nil {
		return err
	}

	status := parseChassisStatus(output)
	c.updateMetrics(status)
	return nil
}

func (c *ChassisCollector) updateMetrics(status chassisStatus) {
	c.metrics.powerStatus.Set(status.power)
	c.metrics.powerOverload.Set(status.powerOverload)
	c.metrics.coolingFault.Set(status.cooling)
	c.metrics.driverFault.Set(status.drives)
	c.metrics.chassisIntrusion.Set(status.intrusion)
	c.metrics.mainPowerFault.Set(status.mainPowerFault)
	c.metrics.powerCtrlFault.Set(status.powerCtrlFault)
}

func (c *ChassisCollector) Collect(ch chan<- prometheus.Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.collect(context.Background()); err != nil {
		log.Printf("机箱状态采集失败: %v", err)
	}

	// 收集所有指标
	c.metrics.powerStatus.Collect(ch)
	c.metrics.powerOverload.Collect(ch)
	c.metrics.coolingFault.Collect(ch)
	c.metrics.driverFault.Collect(ch)
	c.metrics.chassisIntrusion.Collect(ch)
	c.metrics.mainPowerFault.Collect(ch)
	c.metrics.powerCtrlFault.Collect(ch)
}

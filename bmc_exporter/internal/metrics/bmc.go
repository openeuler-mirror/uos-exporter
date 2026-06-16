package metrics

import (
	"bmc_exporter/config"
	"bmc_exporter/internal/exporter"
	"bmc_exporter/internal/ipmi"
	"log"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	bmc_config, err := config.LoadBMCConfig("/etc/uos-exporter/bmc-exporter.yaml")
	if err != nil {
		log.Fatalf("加载BMC配置失败: %v", err)
	}

	bmc := bmc_config.GetBMC()

	ipmiClient := ipmi.NewClient(
		bmc.Host,
		bmc.User,
		bmc.Password,
		bmc.Timeout,
		bmc.Retries,
	)
	exporter.Register(NewBMCCollector(ipmiClient, bmc_config))
}

type BMCCollector struct {
	info    *BMCInfocollector
	chassis *ChassisCollector
	power   *PowerCollector
	sel     *SELCollector
	sensor  *SensorCollector
}

func NewBMCCollector(client *ipmi.Client, cfg *config.Config) *BMCCollector {
	return &BMCCollector{
		info:    NewBMCInfocollector(client),
		chassis: NewChassisCollector(client),
		power:   NewPowerCollector(client),
		sel:     NewSELCollector(client, cfg.BMC.CacheTTL),
		sensor:  NewSensorCollector(client, cfg.BMC.CacheTTL),
	}
}

func (c *BMCCollector) Describe(ch chan<- *prometheus.Desc) {
	c.info.Describe(ch)
	c.chassis.Describe(ch)
	c.power.Describe(ch)
	c.sel.Describe(ch)
	c.sensor.Describe(ch)
}

func (c *BMCCollector) Collect(ch chan<- prometheus.Metric) {
	c.info.Collect(ch)
	c.chassis.Collect(ch)
	c.power.Collect(ch)
	c.sel.Collect(ch)
	c.sensor.Collect(ch)
}

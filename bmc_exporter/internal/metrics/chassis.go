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


// TODO: implement functions

package metrics

import (
	"bmc_exporter/internal/ipmi"
	"context"
	"fmt"
	"log"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type BMCStatusMetric struct {
	Info          *prometheus.GaugeVec // BMC基础信息
	ServiceStatus prometheus.Gauge     // 服务健康状态
	NetworkHealth *prometheus.GaugeVec // 网络连接状态
	LastHeartbeat prometheus.Gauge     // 最后心跳时间
	IPMIErrors    prometheus.Counter   // IPMI错误计数
}


// TODO: implement functions

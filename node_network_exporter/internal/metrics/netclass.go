package metrics

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"regexp"
	"sync"
	"node_network_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs/sysfs"
)

var (
	netclassIgnoredDevices = "^$"
	netclassInvalidSpeed   = false
	netclassNetlink        = false
	sysPath                = "/sys"
)

func init() {
	exporter.Register(NewNetClassCollector())
}

type netClassCollector struct {
	*baseMetrics
	fs                    *sysfs.FS
	subsystem             string
	ignoredDevicesPattern *regexp.Regexp
	metricDescs           map[string]*prometheus.Desc
	metricDescsMu         sync.Mutex
	logger                *slog.Logger
}

// NewNetClassCollector returns a new Collector exposing network class stats.
func NewNetClassCollector() *netClassCollector {
	fs, err := sysfs.NewFS(sysPath)
	var fsPtr *sysfs.FS
	if err != nil {
		// If sysfs is not available, set fs to nil
		slog.Default().Debug("failed to open sysfs", "error", err)
		fsPtr = nil
	} else {
		fsPtr = &fs
	}
	
	pattern := regexp.MustCompile(netclassIgnoredDevices)
	return &netClassCollector{
		baseMetrics:           NewMetrics("node_network_class_total", "Network class statistics", []string{"device"}),
		fs:                    fsPtr,
		subsystem:             "network",
		ignoredDevicesPattern: pattern,
		metricDescs:           map[string]*prometheus.Desc{},
		logger:                slog.Default(),
	}
}

func (c *netClassCollector) Collect(ch chan<- prometheus.Metric) {
	if c.fs == nil {
		return
	}
	
	if netclassNetlink {
		err := c.netClassRTNLUpdate(ch)
		if err != nil {
			c.logger.Error("failed to collect netclass via netlink", "error", err)
		}
		return
	}
	
	err := c.netClassSysfsUpdate(ch)
	if err != nil {
		c.logger.Error("failed to collect netclass via sysfs", "error", err)
	}
}

func (c *netClassCollector) netClassSysfsUpdate(ch chan<- prometheus.Metric) error {
	netClass, err := c.getNetClassInfo()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission) {
			c.logger.Debug("Could not read netclass file", "err", err)
			return nil
		}
		return fmt.Errorf("could not get net class info: %w", err)
	}

	for _, ifaceInfo := range netClass {
		// up指标
		upDesc := prometheus.NewDesc(
			prometheus.BuildFQName("node", c.subsystem, "up"),
			"Value is 1 if operstate is 'up', 0 otherwise.",
			[]string{"device"},
			nil,
		)
		upValue := 0.0
		if ifaceInfo.OperState == "up" {
			upValue = 1.0
		}
		ch <- prometheus.MustNewConstMetric(upDesc, prometheus.GaugeValue, upValue, ifaceInfo.Name)

		// info指标
		infoDesc := prometheus.NewDesc(
			prometheus.BuildFQName("node", c.subsystem, "info"),
			"Non-numeric data from /sys/class/net/<iface>, value is always 1.",
			[]string{"device", "address", "broadcast", "duplex", "operstate", "adminstate", "ifalias"},
			nil,
		)
		infoValue := 1.0
		ch <- prometheus.MustNewConstMetric(infoDesc, prometheus.GaugeValue, infoValue, 
			ifaceInfo.Name, ifaceInfo.Address, ifaceInfo.Broadcast, ifaceInfo.Duplex, 
			ifaceInfo.OperState, getAdminState(ifaceInfo.Flags), ifaceInfo.IfAlias)

		// 其他数值指标
		c.pushMetric(ch, "address_assign_type", ifaceInfo.AddrAssignType, prometheus.GaugeValue, ifaceInfo.Name)
		c.pushMetric(ch, "carrier", ifaceInfo.Carrier, prometheus.GaugeValue, ifaceInfo.Name)
		c.pushMetric(ch, "carrier_changes_total", ifaceInfo.CarrierChanges, prometheus.CounterValue, ifaceInfo.Name)
		c.pushMetric(ch, "carrier_up_changes_total", ifaceInfo.CarrierUpCount, prometheus.CounterValue, ifaceInfo.Name)
		c.pushMetric(ch, "carrier_down_changes_total", ifaceInfo.CarrierDownCount, prometheus.CounterValue, ifaceInfo.Name)
		c.pushMetric(ch, "device_id", ifaceInfo.DevID, prometheus.GaugeValue, ifaceInfo.Name)
		c.pushMetric(ch, "dormant", ifaceInfo.Dormant, prometheus.GaugeValue, ifaceInfo.Name)
		c.pushMetric(ch, "flags", ifaceInfo.Flags, prometheus.GaugeValue, ifaceInfo.Name)
		c.pushMetric(ch, "iface_id", ifaceInfo.IfIndex, prometheus.GaugeValue, ifaceInfo.Name)
		c.pushMetric(ch, "iface_link", ifaceInfo.IfLink, prometheus.GaugeValue, ifaceInfo.Name)
		c.pushMetric(ch, "iface_link_mode", ifaceInfo.LinkMode, prometheus.GaugeValue, ifaceInfo.Name)
		c.pushMetric(ch, "mtu_bytes", ifaceInfo.MTU, prometheus.GaugeValue, ifaceInfo.Name)
		c.pushMetric(ch, "name_assign_type", ifaceInfo.NameAssignType, prometheus.GaugeValue, ifaceInfo.Name)
		c.pushMetric(ch, "net_dev_group", ifaceInfo.NetDevGroup, prometheus.GaugeValue, ifaceInfo.Name)

		// speed_bytes指标
		if ifaceInfo.Speed != nil {
			// Some devices return -1 if the speed is unknown.
			if *ifaceInfo.Speed >= 0 || !netclassInvalidSpeed {
				speedBytes := int64(*ifaceInfo.Speed * 1000 * 1000 / 8)
				c.pushMetric(ch, "speed_bytes", speedBytes, prometheus.GaugeValue, ifaceInfo.Name)
			}
		}

		c.pushMetric(ch, "transmit_queue_length", ifaceInfo.TxQueueLen, prometheus.GaugeValue, ifaceInfo.Name)
		c.pushMetric(ch, "protocol_type", ifaceInfo.Type, prometheus.GaugeValue, ifaceInfo.Name)
	}

	return nil
}

func (c *netClassCollector) netClassRTNLUpdate(ch chan<- prometheus.Metric) error {
	// TODO: implement netlink version if needed
	return c.netClassSysfsUpdate(ch)
}

func (c *netClassCollector) getFieldDesc(name string) *prometheus.Desc {
	c.metricDescsMu.Lock()
	defer c.metricDescsMu.Unlock()

	fieldDesc, exists := c.metricDescs[name]
	if !exists {
		fieldDesc = prometheus.NewDesc(
			prometheus.BuildFQName("node", c.subsystem, name),
			fmt.Sprintf("Network device property: %s", name),
			[]string{"device"},
			nil,
		)
		c.metricDescs[name] = fieldDesc
	}

	return fieldDesc
}

func (c *netClassCollector) pushMetric(ch chan<- prometheus.Metric, name string, value interface{}, valueType prometheus.ValueType, device string) {
	if value == nil {
		return
	}

	var floatValue float64
	switch v := value.(type) {
	case *int64:
		if v == nil {
			return
		}
		floatValue = float64(*v)
	case int64:
		floatValue = float64(v)
	case *int:
		if v == nil {
			return
		}
		floatValue = float64(*v)
	case int:
		floatValue = float64(v)
	case *uint64:
		if v == nil {
			return
		}
		floatValue = float64(*v)
	case uint64:
		floatValue = float64(v)
	default:
		return
	}

	desc := c.getFieldDesc(name)
	ch <- prometheus.MustNewConstMetric(desc, valueType, floatValue, device)
}

func (c *netClassCollector) getNetClassInfo() (sysfs.NetClass, error) {
	netClass := sysfs.NetClass{}
	netDevices, err := c.fs.NetClassDevices()
	if err != nil {
		return netClass, err
	}

	for _, device := range netDevices {
		if c.ignoredDevicesPattern.MatchString(device) {
			continue
		}
		interfaceClass, err := c.fs.NetClassByIface(device)
		if err != nil {
			c.logger.Debug("failed to get netclass info for device", "device", device, "error", err)
			continue
		}
		netClass[device] = *interfaceClass
	}

	return netClass, nil
}

func getAdminState(flags *int64) string {
	if flags == nil {
		return "unknown"
	}

	if *flags&int64(net.FlagUp) == 1 {
		return "up"
	}

	return "down"
} 
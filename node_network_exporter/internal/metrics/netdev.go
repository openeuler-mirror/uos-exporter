package metrics

import (
	"node_network_exporter/internal/exporter"
	"github.com/prometheus/client_golang/prometheus"
	"fmt"
	"net"
	"strconv"
	"sync"
	"regexp"
	"log/slog"
	"github.com/jsimonetti/rtnetlink/v2"
	"github.com/prometheus/procfs"
	"github.com/prometheus/procfs/sysfs"
)

var (
	netDevNetlink      = true
	netdevLabelIfAlias = false
	netdevDeviceInclude = ""
	netdevDeviceExclude = ""
	netdevAddressInfo   = false
	netdevDetailedMetrics = false
	procPath = "/proc"
)

func init() {
	exporter.Register(NewNetDevCollector())
}

type NetDevCollector struct {
	*baseMetrics
	subsystem        string
	deviceFilter     deviceFilter
	metricDescsMutex sync.Mutex
	metricDescs      map[string]*prometheus.Desc
	logger           *slog.Logger
}

type netDevStats map[string]map[string]uint64

type deviceFilter struct {
	ignorePattern *regexp.Regexp
	acceptPattern *regexp.Regexp
}

func newDeviceFilter(ignoredPattern, acceptPattern string) (f deviceFilter) {
	if ignoredPattern != "" {
		f.ignorePattern = regexp.MustCompile(ignoredPattern)
	}

	if acceptPattern != "" {
		f.acceptPattern = regexp.MustCompile(acceptPattern)
	}

	return
}

func (f *deviceFilter) ignored(name string) bool {
	return (f.ignorePattern != nil && f.ignorePattern.MatchString(name)) ||
		(f.acceptPattern != nil && !f.acceptPattern.MatchString(name))
}

func NewNetDevCollector() *NetDevCollector {
	return &NetDevCollector{
		baseMetrics:  NewMetrics("node_network_interface_total", "Network device statistics", []string{"device", "type"}),
		subsystem:    "network",
		deviceFilter: newDeviceFilter(netdevDeviceExclude, netdevDeviceInclude),
		metricDescs:  map[string]*prometheus.Desc{},
		logger:       slog.Default(),
	}
}

func (c *NetDevCollector) metricDesc(key string, labels []string) *prometheus.Desc {
	c.metricDescsMutex.Lock()
	defer c.metricDescsMutex.Unlock()

	if _, ok := c.metricDescs[key]; !ok {
		c.metricDescs[key] = prometheus.NewDesc(
			fmt.Sprintf("node_%s_%s_total", c.subsystem, key),
			fmt.Sprintf("Network device statistic %s.", key),
			labels,
			nil,
		)
	}

	return c.metricDescs[key]
}

func (c *NetDevCollector) Collect(ch chan<- prometheus.Metric) {
	netDev, err := c.getNetDevStats()
	if err != nil {
		c.logger.Error("couldn't get netstats", "error", err)
		return
	}

	netDevLabels, err := c.getNetDevLabels()
	if err != nil {
		c.logger.Error("couldn't get netdev labels", "error", err)
		return
	}

	for dev, devStats := range netDev {
		if !netdevDetailedMetrics {
			c.legacy(devStats)
		}

		labels := []string{"device"}
		labelValues := []string{dev}
		if devLabels, exists := netDevLabels[dev]; exists {
			for labelName, labelValue := range devLabels {
				labels = append(labels, labelName)
				labelValues = append(labelValues, labelValue)
			}
		}

		for key, value := range devStats {
			desc := c.metricDesc(key, labels)
			ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, float64(value), labelValues...)
		}
	}

	if netdevAddressInfo {
		c.collectAddressInfo(ch)
	}
}

func (c *NetDevCollector) getNetDevStats() (netDevStats, error) {
	if netDevNetlink {
		return c.netlinkStats()
	}
	return c.procNetDevStats()
}

func (c *NetDevCollector) netlinkStats() (netDevStats, error) {
	conn, err := rtnetlink.Dial(nil)
	if err != nil {
		return nil, err
	}

	defer conn.Close()
	links, err := conn.Link.List()
	if err != nil {
		return nil, err
	}

	return c.parseNetlinkStats(links), nil
}

func (c *NetDevCollector) parseNetlinkStats(links []rtnetlink.LinkMessage) netDevStats {
	metrics := netDevStats{}

	for _, msg := range links {
		if msg.Attributes == nil {
			c.logger.Debug("No netlink attributes, skipping")
			continue
		}
		name := msg.Attributes.Name
		stats := msg.Attributes.Stats64
		if stats32 := msg.Attributes.Stats; stats == nil && stats32 != nil {
			stats = &rtnetlink.LinkStats64{
				RXPackets:          uint64(stats32.RXPackets),
				TXPackets:          uint64(stats32.TXPackets),
				RXBytes:            uint64(stats32.RXBytes),
				TXBytes:            uint64(stats32.TXBytes),
				RXErrors:           uint64(stats32.RXErrors),
				TXErrors:           uint64(stats32.TXErrors),
				RXDropped:          uint64(stats32.RXDropped),
				TXDropped:          uint64(stats32.TXDropped),
				Multicast:          uint64(stats32.Multicast),
				Collisions:         uint64(stats32.Collisions),
				RXLengthErrors:     uint64(stats32.RXLengthErrors),
				RXOverErrors:       uint64(stats32.RXOverErrors),
				RXCRCErrors:        uint64(stats32.RXCRCErrors),
				RXFrameErrors:      uint64(stats32.RXFrameErrors),
				RXFIFOErrors:       uint64(stats32.RXFIFOErrors),
				RXMissedErrors:     uint64(stats32.RXMissedErrors),
				TXAbortedErrors:    uint64(stats32.TXAbortedErrors),
				TXCarrierErrors:    uint64(stats32.TXCarrierErrors),
				TXFIFOErrors:       uint64(stats32.TXFIFOErrors),
				TXHeartbeatErrors:  uint64(stats32.TXHeartbeatErrors),
				TXWindowErrors:     uint64(stats32.TXWindowErrors),
				RXCompressed:       uint64(stats32.RXCompressed),
				TXCompressed:       uint64(stats32.TXCompressed),
				RXNoHandler:        uint64(stats32.RXNoHandler),
				RXOtherhostDropped: 0,
			}
		}

		if c.deviceFilter.ignored(name) {
			c.logger.Debug("Ignoring device", "device", name)
			continue
		}

		if stats == nil {
			c.logger.Debug("No netlink stats, skipping")
			continue
		}

		metrics[name] = map[string]uint64{
			"receive_packets":  stats.RXPackets,
			"transmit_packets": stats.TXPackets,
			"receive_bytes":    stats.RXBytes,
			"transmit_bytes":   stats.TXBytes,
			"receive_errors":   stats.RXErrors,
			"transmit_errors":  stats.TXErrors,
			"receive_dropped":  stats.RXDropped,
			"transmit_dropped": stats.TXDropped,
			"multicast":        stats.Multicast,
			"collisions":       stats.Collisions,

			"receive_length_errors": stats.RXLengthErrors,
			"receive_over_errors":   stats.RXOverErrors,
			"receive_crc_errors":    stats.RXCRCErrors,
			"receive_frame_errors":  stats.RXFrameErrors,
			"receive_fifo_errors":   stats.RXFIFOErrors,
			"receive_missed_errors": stats.RXMissedErrors,

			"transmit_aborted_errors":   stats.TXAbortedErrors,
			"transmit_carrier_errors":   stats.TXCarrierErrors,
			"transmit_fifo_errors":      stats.TXFIFOErrors,
			"transmit_heartbeat_errors": stats.TXHeartbeatErrors,
			"transmit_window_errors":    stats.TXWindowErrors,

			"receive_compressed":  stats.RXCompressed,
			"transmit_compressed": stats.TXCompressed,
			"receive_nohandler":   stats.RXNoHandler,
		}
	}

	return metrics
}

func (c *NetDevCollector) procNetDevStats() (netDevStats, error) {
	metrics := netDevStats{}

	fs, err := procfs.NewFS(procPath)
	if err != nil {
		return metrics, fmt.Errorf("failed to open procfs: %w", err)
	}

	netDev, err := fs.NetDev()
	if err != nil {
		return metrics, fmt.Errorf("failed to parse /proc/net/dev: %w", err)
	}

	for _, stats := range netDev {
		name := stats.Name

		if c.deviceFilter.ignored(name) {
			c.logger.Debug("Ignoring device", "device", name)
			continue
		}

		metrics[name] = map[string]uint64{
			"receive_bytes":       stats.RxBytes,
			"receive_packets":     stats.RxPackets,
			"receive_errors":      stats.RxErrors,
			"receive_dropped":     stats.RxDropped,
			"receive_fifo":        stats.RxFIFO,
			"receive_frame":       stats.RxFrame,
			"receive_compressed":  stats.RxCompressed,
			"receive_multicast":   stats.RxMulticast,
			"transmit_bytes":      stats.TxBytes,
			"transmit_packets":    stats.TxPackets,
			"transmit_errors":     stats.TxErrors,
			"transmit_dropped":    stats.TxDropped,
			"transmit_fifo":       stats.TxFIFO,
			"transmit_colls":      stats.TxCollisions,
			"transmit_carrier":    stats.TxCarrier,
			"transmit_compressed": stats.TxCompressed,
		}
	}

	return metrics, nil
}

func (c *NetDevCollector) getNetDevLabels() (map[string]map[string]string, error) {
	if !netdevLabelIfAlias {
		return nil, nil
	}

	fs, err := sysfs.NewFS(sysPath)
	if err != nil {
		return nil, err
	}

	interfaces, err := fs.NetClass()
	if err != nil {
		return nil, err
	}

	labels := make(map[string]map[string]string)
	for iface, params := range interfaces {
		labels[iface] = map[string]string{"ifalias": params.IfAlias}
	}

	return labels, nil
}

func (c *NetDevCollector) collectAddressInfo(ch chan<- prometheus.Metric) {
	interfaces, err := net.Interfaces()
	if err != nil {
		c.logger.Error("could not get network interfaces", "error", err)
		return
	}

	desc := prometheus.NewDesc("node_network_address_info", "node network address by device",
		[]string{"device", "address", "netmask", "scope"}, nil)

	for _, addr := range c.getAddrsInfo(interfaces) {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, 1,
			addr.device, addr.addr, addr.netmask, addr.scope)
	}
}

type addrInfo struct {
	device  string
	addr    string
	scope   string
	netmask string
}

func (c *NetDevCollector) scope(ip net.IP) string {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return "link-local"
	}

	if ip.IsInterfaceLocalMulticast() {
		return "interface-local"
	}

	if ip.IsGlobalUnicast() {
		return "global"
	}

	return ""
}

func (c *NetDevCollector) getAddrsInfo(interfaces []net.Interface) []addrInfo {
	var res []addrInfo

	for _, ifs := range interfaces {
		addrs, _ := ifs.Addrs()
		for _, addr := range addrs {
			ip, ipNet, err := net.ParseCIDR(addr.String())
			if err != nil {
				continue
			}
			size, _ := ipNet.Mask.Size()

			res = append(res, addrInfo{
				device:  ifs.Name,
				addr:    ip.String(),
				scope:   c.scope(ip),
				netmask: strconv.Itoa(size),
			})
		}
	}

	return res
}

func (c *NetDevCollector) legacy(metrics map[string]uint64) {
	if metric, ok := c.pop(metrics, "receive_errors"); ok {
		metrics["receive_errs"] = metric
	}
	if metric, ok := c.pop(metrics, "receive_dropped"); ok {
		metrics["receive_drop"] = metric + c.popz(metrics, "receive_missed_errors")
	}
	if metric, ok := c.pop(metrics, "receive_fifo_errors"); ok {
		metrics["receive_fifo"] = metric
	}
	if metric, ok := c.pop(metrics, "receive_frame_errors"); ok {
		metrics["receive_frame"] = metric + c.popz(metrics, "receive_length_errors") + c.popz(metrics, "receive_over_errors") + c.popz(metrics, "receive_crc_errors")
	}
	if metric, ok := c.pop(metrics, "multicast"); ok {
		metrics["receive_multicast"] = metric
	}
	if metric, ok := c.pop(metrics, "transmit_errors"); ok {
		metrics["transmit_errs"] = metric
	}
	if metric, ok := c.pop(metrics, "transmit_dropped"); ok {
		metrics["transmit_drop"] = metric
	}
	if metric, ok := c.pop(metrics, "transmit_fifo_errors"); ok {
		metrics["transmit_fifo"] = metric
	}
	if metric, ok := c.pop(metrics, "collisions"); ok {
		metrics["transmit_colls"] = metric
	}
	if metric, ok := c.pop(metrics, "transmit_carrier_errors"); ok {
		metrics["transmit_carrier"] = metric + c.popz(metrics, "transmit_aborted_errors") + c.popz(metrics, "transmit_heartbeat_errors") + c.popz(metrics, "transmit_window_errors")
	}
}

func (c *NetDevCollector) pop(m map[string]uint64, key string) (uint64, bool) {
	value, ok := m[key]
	delete(m, key)
	return value, ok
}

func (c *NetDevCollector) popz(m map[string]uint64, key string) uint64 {
	if value, ok := m[key]; ok {
		delete(m, key)
		return value
	}
	return 0
} 
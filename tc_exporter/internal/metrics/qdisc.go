package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"tc_exporter/internal/exporter"
	"tc_exporter/internal/tc"
)

func init() {
	exporter.Register(
		NewQdisc())
}

type Qdisc struct {
	qdiscBytesTotal
	qdiscpacketsTotal
	qdiscDropsTotal
	qdiscOverlimitsTotal
	qdiscBps
	qdiscPps
	qdiscQlenTotal
	qdiscBacklogTotal
	qdiscRequeuesTotal
}

func NewQdisc() *Qdisc {
	return &Qdisc{
		qdiscBytesTotal:      *newQdiscBytesTotal(),
		qdiscpacketsTotal:    *newQdiscpacketsTotal(),
		qdiscDropsTotal:      *NewQdiscDropTotal(),
		qdiscOverlimitsTotal: *NewQdiscOverlimitsTotal(),
		qdiscBps:             *NewQdiscBps(),
		qdiscPps:             *NewQdiscPps(),
		qdiscQlenTotal:       *NewQdiscQlenTotal(),
		qdiscBacklogTotal:    *NewQdiscBacklogTotal(),
		qdiscRequeuesTotal:   *NewQdiscRequeuesTotal(),
	}
}

func (qd *Qdisc) Collect(ch chan<- prometheus.Metric) {
	logrus.Info("Start collecting qdisc metrics")
	logrus.Info("get net namespace list")
	nsList, err := tc.GetNetNameSpaceList()
	if err != nil {
		logrus.Warnf("Get net namespace list failed: %v", err)
		return
	}
	if len(nsList) == 0 {
		logrus.Info("No net namespace found")
		return
	}
	for _, ns := range nsList {
		devices, err := tc.GetInterfaceInNetNS(ns)
		if err != nil {
			logrus.Warnf("Get interface in netns %s failed: %v", ns, err)
			continue
		}
		for _, device := range devices {
			qdiscs, err := tc.GetQdiscs(device.Index, ns)
			if err != nil {
				logrus.Warnf("Get qdiscs in netns %s failed: %v", ns, err)
				continue
			}
			for _, qdisc := range qdiscs {
				//handleMaj, handleMin := tc.HandleStr(qdisc.Handle)
				//parentMaj, parentMin := tc.HandleStr(qdisc.Parent)
				var bytes, packets, drops, overlimits, qlen, backlog float64
				if qdisc.Stats2 != nil {
					bytes = float64(qdisc.Stats2.Bytes)
					packets = float64(qdisc.Stats2.Packets)
					drops = float64(qdisc.Stats2.Drops)
					overlimits = float64(qdisc.Stats2.Overlimits)
					qlen = float64(qdisc.Stats2.Qlen)
					backlog = float64(qdisc.Stats2.Backlog)
					qd.qdiscRequeuesTotal.Collect(ch,
						float64(qdisc.Stats2.Requeues),
						[]string{ns,
							device.Attributes.Name})
				} else {
					logrus.Debug("stats2 struct is empty for this qdisc", "qdisc",
						qdisc)
				}
				if qdisc.Stats != nil {
					bytes = float64(qdisc.Stats.Bytes)
					packets = float64(qdisc.Stats.Packets)
					drops = float64(qdisc.Stats.Drops)
					overlimits = float64(qdisc.Stats.Overlimits)
					qlen = float64(qdisc.Stats.Qlen)
					backlog = float64(qdisc.Stats.Backlog)
				}
				if qdisc.Stats2 == nil && qdisc.Stats == nil {
					continue
				}
				qd.qdiscBytesTotal.Collect(ch,
					bytes,
					[]string{ns,
						device.Attributes.Name})
				qd.qdiscpacketsTotal.Collect(ch,
					packets,
					[]string{ns,
						device.Attributes.Name})
				qd.qdiscDropsTotal.Collect(ch,
					drops,
					[]string{ns,
						device.Attributes.Name})
				qd.qdiscOverlimitsTotal.Collect(ch,
					overlimits,
					[]string{ns,
						device.Attributes.Name})
				qd.qdiscQlenTotal.Collect(ch,
					qlen,
					[]string{ns,
						device.Attributes.Name})
				qd.qdiscBacklogTotal.Collect(ch,
					backlog,
					[]string{ns,
						device.Attributes.Name})
				if qdisc.Stats == nil {
					continue
				}
				qd.qdiscBps.Collect(ch,
					float64(qdisc.Stats.Bps),
					[]string{ns,
						device.Attributes.Name})
				qd.qdiscPps.Collect(ch,
					float64(qdisc.Stats.Pps),
					[]string{ns,
						device.Attributes.Name})
			}
		}
	}
}

type qdiscBytesTotal struct {
	*baseMetrics
}

func newQdiscBytesTotal() *qdiscBytesTotal {
	return &qdiscBytesTotal{
		NewMetrics(
			"qdisc_bytes_total",
			"QdiscPie byte counter",
			[]string{"namespace",
				"device"})}
}

func (qd *qdiscBytesTotal) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscpacketsTotal struct {
	*baseMetrics
}

func newQdiscpacketsTotal() *qdiscpacketsTotal {
	return &qdiscpacketsTotal{
		NewMetrics(
			"qdisc_packets_total",
			"QdiscPie packet counter",
			[]string{"namespace",
				"device"})}
}

func (qd *qdiscpacketsTotal) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscDropsTotal struct {
	*baseMetrics
}

func NewQdiscDropTotal() *qdiscDropsTotal {
	return &qdiscDropsTotal{
		NewMetrics(
			"qdisc_drops_total",
			"QdiscPie queue drops",
			[]string{"namespace",
				"device"})}
}

func (qd *qdiscDropsTotal) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscOverlimitsTotal struct {
	*baseMetrics
}

func NewQdiscOverlimitsTotal() *qdiscOverlimitsTotal {
	return &qdiscOverlimitsTotal{
		NewMetrics(
			"qdisc_overlimits_total",
			"QdiscPie queue overlimits",
			[]string{"namespace",
				"device"})}
}

func (qd *qdiscOverlimitsTotal) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscBps struct {
	*baseMetrics
}

func NewQdiscBps() *qdiscBps {
	return &qdiscBps{
		NewMetrics(
			"qdisc_bps",
			"QdiscPie byte rate",
			[]string{"namespace",
				"device"})}
}

func (qd *qdiscBps) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscPps struct {
	*baseMetrics
}

func NewQdiscPps() *qdiscPps {
	return &qdiscPps{
		NewMetrics(
			"qdisc_pps",
			"QdiscPie packet rate",
			[]string{"namespace",
				"device"})}
}

func (qd *qdiscPps) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscQlenTotal struct {
	*baseMetrics
}

func NewQdiscQlenTotal() *qdiscQlenTotal {
	return &qdiscQlenTotal{
		NewMetrics(
			"qdisc_qlen_total",
			"QdiscPie queue length",
			[]string{"namespace",
				"device"})}
}

func (qd *qdiscQlenTotal) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscBacklogTotal struct {
	*baseMetrics
}

func NewQdiscBacklogTotal() *qdiscBacklogTotal {
	return &qdiscBacklogTotal{
		NewMetrics(
			"qdisc_backlog_total",
			"QdiscPie backlog",
			[]string{"namespace",
				"device"})}
}
func (qd *qdiscBacklogTotal) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscRequeuesTotal struct {
	*baseMetrics
}

func NewQdiscRequeuesTotal() *qdiscRequeuesTotal {
	return &qdiscRequeuesTotal{
		NewMetrics(
			"qdisc_requeues_total",
			"QdiscPie requeues",
			[]string{"namespace",
				"device"})}
}
func (qd *qdiscRequeuesTotal) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

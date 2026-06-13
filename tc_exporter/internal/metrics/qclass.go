package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"tc_exporter/internal/exporter"
	"tc_exporter/internal/tc"
)

func init() {
	exporter.Register(
		NewClass())
}

type Class struct {
	ClassBytesTotal
	ClassPacketsTotal
	ClassDropsTotal
	ClassOverlimitsTotal
	ClassBps
	ClassPps
	ClassQlenTotal
	ClassBacklogTotal
	ClassRequequeTotal
}

func NewClass() *Class {
	return &Class{
		ClassBytesTotal:      *newClassBytes(),
		ClassPacketsTotal:    *newClassPackets(),
		ClassDropsTotal:      *newClassDrops(),
		ClassOverlimitsTotal: *newClassOverlimits(),
		ClassBps:             *newClassBps(),
		ClassPps:             *newClassPps(),
		ClassQlenTotal:       *newClassQlen(),
		ClassBacklogTotal:    *newClassBacklog(),
		ClassRequequeTotal:   *newClassRequeue(),
		//ClassQlenTotal: *newClassQlen(),
	}
}

func (cls *Class) Collect(ch chan<- prometheus.Metric) {
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
			classes, err := tc.GetClasses(device.Index, ns)
			if err != nil {
				logrus.Warnf("Get classes in netns %s failed: %v", ns, err)
				continue
			}
			for _, class := range classes {
				//handleMaj, handleMin := tc.HandleStr(class.Handle)
				//parentMaj, parentMin := tc.HandleStr(class.Parent)
				var bytes, packets, drops, overlimits, qlen, backlog float64
				if class.Stats2 != nil {
					bytes = float64(class.Stats2.Bytes)
					packets = float64(class.Stats2.Packets)
					drops = float64(class.Stats2.Drops)
					overlimits = float64(class.Stats2.Overlimits)
					qlen = float64(class.Stats2.Qlen)
					backlog = float64(class.Stats2.Backlog)
					cls.ClassRequequeTotal.Collect(ch,
						float64(class.Stats2.Requeues),
						[]string{ns,
							device.Attributes.Name})
				} else {
					logrus.Debug("stats2 struct is empty for this class", "class",
						class)
				}
				if class.Stats != nil {
					bytes = float64(class.Stats.Bytes)
					packets = float64(class.Stats.Packets)
					drops = float64(class.Stats.Drops)
					overlimits = float64(class.Stats.Overlimits)
					qlen = float64(class.Stats.Qlen)
					backlog = float64(class.Stats.Backlog)
				}

				if class.Stats2 != nil || class.Stats != nil {
					cls.ClassBytesTotal.Collect(ch,
						bytes,
						[]string{ns,
							device.Attributes.Name})
					cls.ClassPacketsTotal.Collect(ch,
						packets,
						[]string{ns,
							device.Attributes.Name})
					cls.ClassBacklogTotal.Collect(ch,
						backlog,
						[]string{ns,
							device.Attributes.Name})
					cls.ClassDropsTotal.Collect(ch,
						drops,
						[]string{ns,
							device.Attributes.Name})
					cls.ClassOverlimitsTotal.Collect(ch,
						overlimits,
						[]string{ns,
							device.Attributes.Name})
					cls.ClassQlenTotal.Collect(ch,
						qlen,
						[]string{ns,
							device.Attributes.Name})
				}
				if class.Stats == nil {
					continue
				}
				cls.ClassBps.Collect(ch,
					float64(
						class.Stats.Bps),
					[]string{ns,
						device.Attributes.Name})
				cls.ClassPps.Collect(ch,
					float64(class.Stats.Pps),
					[]string{ns,
						device.Attributes.Name})

			}
		}
	}
}

type ClassBytesTotal struct {
	*baseMetrics
}

func newClassBytes() *ClassBytesTotal {
	return &ClassBytesTotal{
		NewMetrics(
			"class_bytes_total",
			"class byte counter",
			[]string{"namespace",
				"device"})}
}

func (qd *ClassBytesTotal) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type ClassPacketsTotal struct {
	*baseMetrics
}

func newClassPackets() *ClassPacketsTotal {
	return &ClassPacketsTotal{
		NewMetrics(
			"class_packets_total",
			"class packet counter",
			[]string{"namespace",
				"device"})}
}

func (qd *ClassPacketsTotal) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type ClassDropsTotal struct {
	*baseMetrics
}

func newClassDrops() *ClassDropsTotal {
	return &ClassDropsTotal{
		NewMetrics(
			"class_drops_total",
			"class drop counter",
			[]string{"namespace",
				"device"})}
}

func (qd *ClassDropsTotal) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type ClassOverlimitsTotal struct {
	*baseMetrics
}

func newClassOverlimits() *ClassOverlimitsTotal {
	return &ClassOverlimitsTotal{
		NewMetrics(
			"class_overlimits_total",
			"class overlimits counter",
			[]string{"namespace",
				"device"})}
}

func (qd *ClassOverlimitsTotal) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type ClassBps struct {
	*baseMetrics
}

func newClassBps() *ClassBps {
	return &ClassBps{
		NewMetrics(
			"class_bps",
			"Class byte rate",
			[]string{"namespace",
				"device"})}
}

func (qd *ClassBps) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type ClassPps struct {
	*baseMetrics
}

func newClassPps() *ClassPps {
	return &ClassPps{
		NewMetrics(
			"class_pps",
			"Class packet rate",
			[]string{"namespace",
				"device"})}
}

func (qd *ClassPps) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type ClassQlenTotal struct {
	*baseMetrics
}

func newClassQlen() *ClassQlenTotal {
	return &ClassQlenTotal{
		NewMetrics(
			"class_qlen_total",
			"Class queue length",
			[]string{"namespace",
				"device"})}
}

func (qd *ClassQlenTotal) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type ClassBacklogTotal struct {
	*baseMetrics
}

func newClassBacklog() *ClassBacklogTotal {
	return &ClassBacklogTotal{
		NewMetrics(
			"class_backlog_total",
			"Class backlog",
			[]string{"namespace",
				"device"})}
}

func (qd *ClassBacklogTotal) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type ClassRequequeTotal struct {
	*baseMetrics
}

func newClassRequeue() *ClassRequequeTotal {
	return &ClassRequequeTotal{
		NewMetrics(
			"class_requeue_total",
			"Class requeue counter",
			[]string{"namespace",
				"device"})}
}

func (qd *ClassRequequeTotal) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

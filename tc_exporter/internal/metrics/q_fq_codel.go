package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"tc_exporter/internal/exporter"
	"tc_exporter/internal/tc"
)

func init() {
	exporter.Register(
		NewQdiscFqCodel())
}

type QdiscFqCodel struct {
	qdiscFqCodelCeMark
	qdiscFqCodelDropOverlimit
	qdiscFqCodelDropOverMemory
	qdiscFqCodelEcnMark
	qdiscFqCodelMaxPacket
	qdiscFqCodelMemoryUsage
	qdiscFqCodelNewFlowsCount
	qdiscFqCodelNewFlowsLen
	qdiscFqCodelOldFlowsLen
}

func NewQdiscFqCodel() *QdiscFqCodel {
	return &QdiscFqCodel{
		qdiscFqCodelCeMark: *newQdiscFqCodelCeMark(),
	}
}

func (qd *QdiscFqCodel) Collect(ch chan<- prometheus.Metric) {
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
				if qdisc.Kind != "fq_codel" {
					continue
				}
				if qdisc.XStats == nil {
					continue
				}
				if qdisc.XStats.FqCodel == nil {
					continue
				}
				qd.qdiscFqCodelCeMark.Collect(ch,
					float64(qdisc.XStats.FqCodel.Qd.CeMark),
					[]string{ns,
						device.Attributes.Name,
						"fq_codel"})
				qd.qdiscFqCodelDropOverlimit.Collect(ch,
					float64(qdisc.XStats.FqCodel.Qd.DropOverlimit),
					[]string{ns,
						device.Attributes.Name,
						"fq_codel"})
				qd.qdiscFqCodelDropOverMemory.Collect(ch,
					float64(qdisc.XStats.FqCodel.Qd.DropOvermemory),
					[]string{ns,
						device.Attributes.Name,
						"fq_codel"})

				qd.qdiscFqCodelEcnMark.Collect(ch,
					float64(qdisc.XStats.FqCodel.Qd.EcnMark),
					[]string{ns,
						device.Attributes.Name,
						"fq_codel"})
				qd.qdiscFqCodelMaxPacket.Collect(ch,
					float64(qdisc.XStats.FqCodel.Qd.MaxPacket),
					[]string{ns,
						device.Attributes.Name,
						"fq_codel"})
				qd.qdiscFqCodelMemoryUsage.Collect(ch,
					float64(qdisc.XStats.FqCodel.Qd.MemoryUsage),
					[]string{ns,
						device.Attributes.Name,
						"fq_codel"})
				qd.qdiscFqCodelNewFlowsCount.Collect(ch,
					float64(qdisc.XStats.FqCodel.Qd.NewFlowCount),
					[]string{ns,
						device.Attributes.Name,
						"fq_codel"})
				qd.qdiscFqCodelNewFlowsLen.Collect(ch,
					float64(qdisc.XStats.FqCodel.Qd.NewFlowsLen),
					[]string{ns,
						device.Attributes.Name,
						"fq_codel"})
			}
		}
	}
}

type qdiscFqCodelCeMark struct {
	*baseMetrics
}

func newQdiscFqCodelCeMark() *qdiscFqCodelCeMark {
	logrus.Debug("create qdiscFqCodelCeMark")
	return &qdiscFqCodelCeMark{
		NewMetrics(
			"qdisc_fq_codel_ce_mark",
			"fq_codel ce mark xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscFqCodelCeMark) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscFqCodelDropOverlimit struct {
	*baseMetrics
}

func newQdiscFqCodelDropOverlimit() *qdiscFqCodelDropOverlimit {
	logrus.Debug("create qdiscFqCodelDropOverlimit")
	return &qdiscFqCodelDropOverlimit{
		NewMetrics(
			"qdisc_fq_codel_drop_overlimit",
			"fq_codel drop overlimit xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}
func (qd *qdiscFqCodelDropOverlimit) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscFqCodelDropOverMemory struct {
	*baseMetrics
}

func newQdiscFqCodelDropOverMemory() *qdiscFqCodelDropOverMemory {
	logrus.Debug("create qdiscFqCodelDropOverMemory")
	return &qdiscFqCodelDropOverMemory{
		NewMetrics(
			"qdisc_fq_codel_drop_overmemory",
			"fq_codel drop overmemory xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}
func (qd *qdiscFqCodelDropOverMemory) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscFqCodelEcnMark struct {
	*baseMetrics
}

func newQdiscFqCodelEcnMark() *qdiscFqCodelEcnMark {
	logrus.Debug("create qdiscFqCodelEcnMark")
	return &qdiscFqCodelEcnMark{
		NewMetrics(
			"qdisc_fq_codel_ecn_mark",
			"fq_codel ecn mark xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscFqCodelEcnMark) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscFqCodelMaxPacket struct {
	*baseMetrics
}

func newQdiscFqCodelMaxPacket() *qdiscFqCodelMaxPacket {
	logrus.Debug("create qdiscFqCodelMaxPacket")
	return &qdiscFqCodelMaxPacket{
		NewMetrics(
			"qdisc_fq_codel_max_packet",
			"fq_codel max packet xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscFqCodelMaxPacket) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscFqCodelMemoryUsage struct {
	*baseMetrics
}

func newQdiscFqCodelMemoryUsage() *qdiscFqCodelMemoryUsage {
	logrus.Debug("create qdiscFqCodelMemoryUsage")
	return &qdiscFqCodelMemoryUsage{
		NewMetrics(
			"qdisc_fq_codel_memory_usage",
			"fq_codel memory usage xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscFqCodelMemoryUsage) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscFqCodelNewFlowsCount struct {
	*baseMetrics
}

func newQdiscFqCodelNewFlowsCount() *qdiscFqCodelNewFlowsCount {
	logrus.Debug("create qdiscFqCodelNewFlowsCount")
	return &qdiscFqCodelNewFlowsCount{
		NewMetrics(
			"qdisc_fq_codel_new_flows_count",
			"fq_codel new flows count xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscFqCodelNewFlowsCount) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscFqCodelNewFlowsLen struct {
	*baseMetrics
}

func newQdiscFqCodelNewFlowsLen() *qdiscFqCodelNewFlowsLen {
	logrus.Debug("create qdiscFqCodelNewFlowsLen")
	return &qdiscFqCodelNewFlowsLen{
		NewMetrics(
			"qdisc_fq_codel_new_flows_len",
			"fq_codel new flows len xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscFqCodelNewFlowsLen) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscFqCodelOldFlowsLen struct {
	*baseMetrics
}

func newQdiscFqCodelOldFlowsLen() *qdiscFqCodelOldFlowsLen {
	logrus.Debug("create qdiscFqCodelOldFlowsLen")
	return &qdiscFqCodelOldFlowsLen{
		NewMetrics(
			"qdisc_fq_codel_old_flows_len",
			"fq_codel old flows len xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscFqCodelOldFlowsLen) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

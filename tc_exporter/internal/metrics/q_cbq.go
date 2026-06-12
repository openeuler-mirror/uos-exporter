package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"tc_exporter/internal/exporter"
	"tc_exporter/internal/tc"
)

func init() {
	exporter.Register(
		NewQdiscCbq())
}

type QdiscCbq struct {
	qdiscCbqAvgIdle
	qdiscCbqBorrows
	qdiscCbqOveractions
	qdiscCbqUnderTime
}

func NewQdiscCbq() *QdiscCbq {
	return &QdiscCbq{
		qdiscCbqAvgIdle: *newQdiscCbqAvgIdle(),
	}
}

func (qd *QdiscCbq) Collect(ch chan<- prometheus.Metric) {
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
				if qdisc.Kind != "cbq" {
					continue
				}
				if qdisc.XStats == nil {
					continue
				}
				if qdisc.XStats.Cbq == nil {
					continue
				}
				qd.qdiscCbqAvgIdle.Collect(ch,
					float64(qdisc.XStats.Cbq.AvgIdle),
					[]string{ns,
						device.Attributes.Name,
						"cbq"})
				qd.qdiscCbqBorrows.Collect(ch,
					float64(qdisc.XStats.Cbq.Borrows),
					[]string{ns,
						device.Attributes.Name,
						"cbq"})
				qd.qdiscCbqOveractions.Collect(ch,
					float64(qdisc.XStats.Cbq.Overactions),
					[]string{ns,
						device.Attributes.Name,
						"cbq"})
				qd.qdiscCbqUnderTime.Collect(ch,
					float64(qdisc.XStats.Cbq.Undertime),
					[]string{ns,
						device.Attributes.Name,
						"cbq"})

			}
		}
	}
}

type qdiscCbqAvgIdle struct {
	*baseMetrics
}

func newQdiscCbqAvgIdle() *qdiscCbqAvgIdle {
	logrus.Debug("create qdiscFqCodelCeMark")
	return &qdiscCbqAvgIdle{
		NewMetrics(
			"qdisc_cbq_bavg_idle",
			"CBQ avg idle xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscCbqAvgIdle) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscCbqBorrows struct {
	*baseMetrics
}

func newQdiscCbqBorrows() *qdiscCbqBorrows {
	logrus.Debug("create qdiscCbqBorrows")
	return &qdiscCbqBorrows{
		NewMetrics(
			"qdisc_cbq_borrows",
			"CBQ borrows xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscCbqBorrows) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscCbqOveractions struct {
	*baseMetrics
}

func newQdiscCbqOveractions() *qdiscCbqOveractions {
	logrus.Debug("create qdiscCbqOveractions")
	return &qdiscCbqOveractions{
		NewMetrics(
			"qdisc_cbq_overactions",
			"CBQ overactions xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscCbqOveractions) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscCbqUnderTime struct {
	*baseMetrics
}

func newQdiscCbqUndeTime() *qdiscCbqUnderTime {
	logrus.Debug("create qdiscCbqUnderTime")
	return &qdiscCbqUnderTime{
		NewMetrics(
			"qdisc_cbq_undertime",
			"CBQ undetime xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscCbqUnderTime) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

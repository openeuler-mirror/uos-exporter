package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"tc_exporter/internal/exporter"
	"tc_exporter/internal/tc"
)

func init() {
	exporter.Register(
		NewQdiscChoke())
}

type QdiscChoke struct {
	qdiscChokeEarly
	qdiscChokeMarked
	qdiscChokeMatched
	qdiscChokeOther
	qdiscChokePdrop
}

func NewQdiscChoke() *QdiscChoke {
	return &QdiscChoke{
		qdiscChokeEarly:   *newQdiscChokeEarly(),
		qdiscChokeMarked:  *newQdiscChokeMarked(),
		qdiscChokeMatched: *newQdiscChokeMatched(),
		qdiscChokeOther:   *newQdiscChokeOther(),
		qdiscChokePdrop:   *newQdiscChokePdrop(),
	}
}

func (qd *QdiscChoke) Collect(ch chan<- prometheus.Metric) {
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
				if qdisc.Kind != "choke" {
					continue
				}
				if qdisc.XStats == nil {
					continue
				}
				if qdisc.XStats.Choke == nil {
					continue
				}
				qd.qdiscChokeEarly.Collect(ch,
					float64(qdisc.XStats.Choke.Early),
					[]string{ns,
						device.Attributes.Name,
						"choke"})
				qd.qdiscChokeMarked.Collect(ch,
					float64(qdisc.XStats.Choke.Marked),
					[]string{ns,
						device.Attributes.Name,
						"choke"})
				qd.qdiscChokeMatched.Collect(ch,
					float64(qdisc.XStats.Choke.Matched),
					[]string{ns,
						device.Attributes.Name,
						"choke"})
				qd.qdiscChokeOther.Collect(ch,
					float64(qdisc.XStats.Choke.Other),
					[]string{ns,
						device.Attributes.Name,
						"choke"})
				qd.qdiscChokePdrop.Collect(ch,
					float64(qdisc.XStats.Choke.PDrop),
					[]string{ns,
						device.Attributes.Name,
						"choke"})
			}
		}
	}
}

type qdiscChokeEarly struct {
	*baseMetrics
}

func newQdiscChokeEarly() *qdiscChokeEarly {
	logrus.Debug("create qdiscFqCodelCeMark")
	return &qdiscChokeEarly{
		NewMetrics(
			"qdisc_choke_early",
			"Choke early xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscChokeEarly) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscChokeMarked struct {
	*baseMetrics
}

func newQdiscChokeMarked() *qdiscChokeMarked {
	logrus.Debug("create qdiscFqCodelCeMark")
	return &qdiscChokeMarked{
		NewMetrics(
			"qdisc_choke_marked",
			"Choke marked xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}
func (qd *qdiscChokeMarked) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscChokeMatched struct {
	*baseMetrics
}

func newQdiscChokeMatched() *qdiscChokeMatched {
	logrus.Debug("create qdiscFqCodelCeMark")
	return &qdiscChokeMatched{
		NewMetrics(
			"qdisc_choke_matched",
			"Choke matched xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscChokeMatched) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscChokeOther struct {
	*baseMetrics
}

func newQdiscChokeOther() *qdiscChokeOther {
	logrus.Debug("create qdiscFqCodelCeMark")
	return &qdiscChokeOther{
		NewMetrics(
			"qdisc_choke_other",
			"Choke other xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}
func (qd *qdiscChokeOther) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscChokePdrop struct {
	*baseMetrics
}

func newQdiscChokePdrop() *qdiscChokePdrop {
	logrus.Debug("create qdiscFqCodelCeMark")
	return &qdiscChokePdrop{
		NewMetrics(
			"qdisc_choke_pdrop",
			"Choke pdrop xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscChokePdrop) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

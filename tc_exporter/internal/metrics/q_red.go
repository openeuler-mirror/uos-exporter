package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"tc_exporter/internal/exporter"
	"tc_exporter/internal/tc"
)

func init() {
	exporter.Register(
		NewQdiscRed())
}

type QdiscRed struct {
	qdiscRedEarly
	qdiscRedMarked
	qdiscRedOther
	qdiscRedPdrop
}

func NewQdiscRed() *QdiscRed {
	return &QdiscRed{
		qdiscRedEarly:  *newQdiscRedEarly(),
		qdiscRedMarked: *newQdiscRedMarked(),
		qdiscRedOther:  *newQdiscRedOther(),
		qdiscRedPdrop:  *newQdiscRedPdrop(),
	}
}

func (qd *QdiscRed) Collect(ch chan<- prometheus.Metric) {
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
				if qdisc.Kind != "red" {
					continue
				}
				if qdisc.XStats == nil {
					continue
				}
				if qdisc.XStats.Red == nil {
					continue
				}
				qd.qdiscRedEarly.Collect(ch,
					float64(qdisc.XStats.Red.Early),
					[]string{ns,
						device.Attributes.Name,
						"red"})

				qd.qdiscRedMarked.Collect(ch,
					float64(qdisc.XStats.Red.Marked),
					[]string{ns,
						device.Attributes.Name,
						"red"})

				qd.qdiscRedOther.Collect(ch,
					float64(qdisc.XStats.Red.Other),
					[]string{ns,
						device.Attributes.Name,
						"red"})

				qd.qdiscRedPdrop.Collect(ch,
					float64(qdisc.XStats.Red.PDrop),
					[]string{ns,
						device.Attributes.Name,
						"red"})

			}
		}
	}
}

type qdiscRedEarly struct {
	*baseMetrics
}

func newQdiscRedEarly() *qdiscRedEarly {
	logrus.Debug("create qdiscPieAvgDqRate")
	return &qdiscRedEarly{
		NewMetrics(
			"qdisc_red_early",
			"RED early xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscRedEarly) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscRedMarked struct {
	*baseMetrics
}

func newQdiscRedMarked() *qdiscRedMarked {
	logrus.Debug("create qdiscPieAvgDqRate")
	return &qdiscRedMarked{
		NewMetrics(
			"qdisc_red_marked",
			"RED marked xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscRedMarked) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscRedOther struct {
	*baseMetrics
}

func newQdiscRedOther() *qdiscRedOther {
	logrus.Debug("create qdiscPieAvgDqRate")
	return &qdiscRedOther{
		NewMetrics(
			"qdisc_red_other",
			"RED other xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscRedOther) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscRedPdrop struct {
	*baseMetrics
}

func newQdiscRedPdrop() *qdiscRedPdrop {
	logrus.Debug("create qdiscPieAvgDqRate")
	return &qdiscRedPdrop{
		NewMetrics(
			"qdisc_red_pdrop",
			"RED pdrop xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscRedPdrop) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

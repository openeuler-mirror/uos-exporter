package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"tc_exporter/internal/exporter"
	"tc_exporter/internal/tc"
)

func init() {
	exporter.Register(
		NewQdiscHfsc())
}

type QdiscHfsc struct {
	qdiscHfscLevel
	qdiscHfscPeriod
	qdiscHfscRtWork
	qdiscHfscWork
}

func NewQdiscHfsc() *QdiscHfsc {
	return &QdiscHfsc{
		qdiscHfscLevel: *newQdiscHfscLevel(),
	}
}

func (qd *QdiscHfsc) Collect(ch chan<- prometheus.Metric) {
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
				if qdisc.Kind != "hfsc" {
					continue
				}
				if qdisc.XStats == nil {
					continue
				}
				if qdisc.XStats.Hfsc == nil {
					continue
				}
				qd.qdiscHfscLevel.Collect(ch,
					float64(qdisc.XStats.Hfsc.Level),
					[]string{ns,
						device.Attributes.Name,
						"hfsc"})

				qd.qdiscHfscPeriod.Collect(ch,
					float64(qdisc.XStats.Hfsc.Period),
					[]string{ns,
						device.Attributes.Name,
						"hfsc"})

				qd.qdiscHfscRtWork.Collect(ch,
					float64(qdisc.XStats.Hfsc.RtWork),
					[]string{ns,
						device.Attributes.Name,
						"hfsc"})

				qd.qdiscHfscWork.Collect(ch,
					float64(qdisc.XStats.Hfsc.Work),
					[]string{ns,
						device.Attributes.Name,
						"hfsc"})

			}
		}
	}
}

type qdiscHfscLevel struct {
	*baseMetrics
}

func newQdiscHfscLevel() *qdiscHfscLevel {
	logrus.Debug("create QdiscHfscLevel")
	return &qdiscHfscLevel{
		NewMetrics(
			"qdisc_hfsc_level",
			"hfsc level xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscHfscLevel) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscHfscPeriod struct {
	*baseMetrics
}

func newQdiscHfscPeriod() *qdiscHfscPeriod {
	logrus.Debug("create QdiscHfscPeriod")
	return &qdiscHfscPeriod{
		NewMetrics(
			"qdisc_hfsc_period",
			"hfsc period xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscHfscPeriod) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscHfscRtWork struct {
	*baseMetrics
}

func newQdiscHfscRtWork() *qdiscHfscRtWork {
	logrus.Debug("create QdiscHfscRtWork")
	return &qdiscHfscRtWork{
		NewMetrics(
			"qdisc_hfsc_rt_work",
			"hfsc rt work xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscHfscRtWork) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscHfscWork struct {
	*baseMetrics
}

func newQdiscHfscWork() *qdiscHfscWork {
	logrus.Debug("create QdiscHfscWork")
	return &qdiscHfscWork{
		NewMetrics(
			"qdisc_hfsc_work",
			"hfsc work xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscHfscWork) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

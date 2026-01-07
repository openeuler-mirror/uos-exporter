package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"tc_exporter/internal/exporter"
	"tc_exporter/internal/tc"
)

func init() {
	exporter.Register(
		NewQdiscSfq())
}

type QdiscSfq struct {
	qdiscSfqAllot
}

func NewQdiscSfq() *QdiscSfq {
	return &QdiscSfq{
		qdiscSfqAllot: *newQdiscSfqAllot(),
	}
}

func (qd *QdiscSfq) Collect(ch chan<- prometheus.Metric) {
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
				if qdisc.Kind != "sfq" {
					continue
				}
				if qdisc.XStats == nil {
					continue
				}
				if qdisc.XStats.Sfq == nil {
					continue
				}
				qd.qdiscSfqAllot.Collect(ch,
					float64(qdisc.XStats.Sfq.Allot),
					[]string{ns,
						device.Attributes.Name,
						"sfq"})

			}
		}
	}
}

type qdiscSfqAllot struct {
	*baseMetrics
}

func newQdiscSfqAllot() *qdiscSfqAllot {
	logrus.Debug("create qdiscPieAvgDqRate")
	return &qdiscSfqAllot{
		NewMetrics(
			"qdisc_sfq_allot",
			"SFQ allot xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscSfqAllot) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

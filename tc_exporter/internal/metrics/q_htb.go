package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"tc_exporter/internal/exporter"
	"tc_exporter/internal/tc"
)

func init() {
	exporter.Register(
		NewQdiscHtb())
}

type QdiscHtb struct {
	qdiscHtbBorrows
	qdiscHtbCtokens
	qdiscHtbGiants
	qdiscHtbLends
}

func NewQdiscHtb() *QdiscHtb {
	return &QdiscHtb{
		qdiscHtbBorrows: *newQdiscHtbBorrows(),
		qdiscHtbCtokens: *newQdiscHtbCtokens(),
		qdiscHtbGiants:  *newQdiscHtbGiants(),
		qdiscHtbLends:   *newQdiscHtbLends(),
	}
}

func (qd *QdiscHtb) Collect(ch chan<- prometheus.Metric) {
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
				if qdisc.Kind != "htb" {
					continue
				}
				if qdisc.XStats == nil {
					continue
				}
				if qdisc.XStats.Htb == nil {
					continue
				}
				qd.qdiscHtbBorrows.Collect(ch,
					float64(qdisc.XStats.Htb.Borrows),
					[]string{ns,
						device.Attributes.Name,
						"htb"})
				qd.qdiscHtbCtokens.Collect(ch,
					float64(qdisc.XStats.Htb.CTokens),
					[]string{ns,
						device.Attributes.Name,
						"htb"})
				qd.qdiscHtbGiants.Collect(ch,
					float64(qdisc.XStats.Htb.Giants),
					[]string{ns,
						device.Attributes.Name,
						"htb"})
				qd.qdiscHtbLends.Collect(ch,
					float64(qdisc.XStats.Htb.Lends),
					[]string{ns,
						device.Attributes.Name,
						"htb"})

			}
		}
	}
}

type qdiscHtbBorrows struct {
	*baseMetrics
}

func newQdiscHtbBorrows() *qdiscHtbBorrows {
	logrus.Debug("create qdiscPieAvgDqRate")
	return &qdiscHtbBorrows{
		NewMetrics(
			"qdisc_htb_borrows",
			"HTB borrows xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscHtbBorrows) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscHtbCtokens struct {
	*baseMetrics
}

func newQdiscHtbCtokens() *qdiscHtbCtokens {
	logrus.Debug("create qdiscHtbCtokens")
	return &qdiscHtbCtokens{
		NewMetrics(
			"qdisc_htb_ctokens",
			"HTB ctokens xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscHtbCtokens) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscHtbGiants struct {
	*baseMetrics
}

func newQdiscHtbGiants() *qdiscHtbGiants {
	logrus.Debug("create qdiscHtbGiants")
	return &qdiscHtbGiants{
		NewMetrics(
			"qdisc_htb_giants",
			"HTB giants xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscHtbGiants) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscHtbLends struct {
	*baseMetrics
}

func newQdiscHtbLends() *qdiscHtbLends {
	logrus.Debug("create qdiscHtbLends")
	return &qdiscHtbLends{
		NewMetrics(
			"qdisc_htb_lends",
			"HTB lends xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscHtbLends) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

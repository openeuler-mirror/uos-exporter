package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"tc_exporter/internal/exporter"
	"tc_exporter/internal/tc"
)

func init() {
	exporter.Register(
		NewQdiscCodel())
}

type QdiscCodel struct {
	qdiscCodelCeMark
	qdiscCodelCount
	qdiscCodelDropNext
	qdiscCodelDropOverlimit
	qdiscCodelDropping
	qdiscCodelEcnMark
	qdiscCodelLdelay
	qdiscCodelLastCount
	qdiscCodelMaxPacket
}

func NewQdiscCodel() *QdiscCodel {
	return &QdiscCodel{
		qdiscCodelCeMark:        *newQdiscCodelCeMark(),
		qdiscCodelCount:         *newQdiscCodelCount(),
		qdiscCodelDropNext:      *newQdiscCodelDropNext(),
		qdiscCodelDropOverlimit: *newQdiscCodelDropOverlimit(),
		qdiscCodelDropping:      *newQdiscCodelDropping(),
		qdiscCodelEcnMark:       *newQdiscCodelEcnMark(),
		qdiscCodelLdelay:        *newQdiscCodelLdelay(),
		qdiscCodelMaxPacket:     *newQdiscCodelMaxPacket(),
	}
}

func (qd *QdiscCodel) Collect(ch chan<- prometheus.Metric) {
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
				if qdisc.Kind != "codel" {
					continue
				}
				if qdisc.XStats == nil {
					continue
				}
				if qdisc.XStats.Codel == nil {
					continue
				}
				qd.qdiscCodelCeMark.Collect(ch,
					float64(qdisc.XStats.Codel.CeMark),
					[]string{ns,
						device.Attributes.Name,
						"codel"})
				qd.qdiscCodelCount.Collect(ch,
					float64(qdisc.XStats.Codel.Count),
					[]string{ns,
						device.Attributes.Name,
						"codel"})
				qd.qdiscCodelDropNext.Collect(ch,
					float64(qdisc.XStats.Codel.DropNext),
					[]string{ns,
						device.Attributes.Name,
						"codel"})
				qd.qdiscCodelDropOverlimit.Collect(ch,
					float64(qdisc.XStats.Codel.DropOverlimit),
					[]string{ns,
						device.Attributes.Name,
						"codel"})
				qd.qdiscCodelDropping.Collect(ch,
					float64(qdisc.XStats.Codel.Dropping),
					[]string{ns,
						device.Attributes.Name,
						"codel"})
				qd.qdiscCodelEcnMark.Collect(ch,
					float64(qdisc.XStats.Codel.EcnMark),
					[]string{ns,
						device.Attributes.Name,
						"codel"})
				qd.qdiscCodelLdelay.Collect(ch,
					float64(qdisc.XStats.Codel.LDelay),
					[]string{ns,
						device.Attributes.Name,
						"codel"})
				qd.qdiscCodelMaxPacket.Collect(ch,
					float64(qdisc.XStats.Codel.MaxPacket),
					[]string{ns,
						device.Attributes.Name,
						"codel"})

			}
		}
	}
}

type qdiscCodelCeMark struct {
	*baseMetrics
}

func newQdiscCodelCeMark() *qdiscCodelCeMark {
	logrus.Debug("create qdiscFqCodelCeMark")
	return &qdiscCodelCeMark{
		NewMetrics(
			"qdisc_codel_ce_mark",
			"Codel CE mark xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscCodelCeMark) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscCodelCount struct {
	*baseMetrics
}

func newQdiscCodelCount() *qdiscCodelCount {
	logrus.Debug("create qdiscCodelCount")
	return &qdiscCodelCount{
		NewMetrics(
			"qdisc_codel_count",
			"Codel count xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}
func (qd *qdiscCodelCount) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscCodelDropNext struct {
	*baseMetrics
}

func newQdiscCodelDropNext() *qdiscCodelDropNext {
	logrus.Debug("create qdiscCodelDropNext")
	return &qdiscCodelDropNext{
		NewMetrics(
			"qdisc_codel_drop_next",
			"Codel drop next xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscCodelDropNext) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscCodelDropOverlimit struct {
	*baseMetrics
}

func newQdiscCodelDropOverlimit() *qdiscCodelDropOverlimit {
	logrus.Debug("create qdiscCodelDropOverlimit")
	return &qdiscCodelDropOverlimit{
		NewMetrics(
			"qdisc_codel_drop_overlimit",
			"Codel drop overlimit xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}
func (qd *qdiscCodelDropOverlimit) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscCodelDropping struct {
	*baseMetrics
}

func newQdiscCodelDropping() *qdiscCodelDropping {
	logrus.Debug("create qdiscCodelDropping")
	return &qdiscCodelDropping{
		NewMetrics(
			"qdisc_codel_dropping",
			"Codel dropping xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}
func (qd *qdiscCodelDropping) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscCodelEcnMark struct {
	*baseMetrics
}

func newQdiscCodelEcnMark() *qdiscCodelEcnMark {
	logrus.Debug("create qdiscCodelEcnMark")
	return &qdiscCodelEcnMark{
		NewMetrics(
			"qdisc_codel_ecn_mark",
			"Codel ecn mark xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscCodelEcnMark) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscCodelLdelay struct {
	*baseMetrics
}

func newQdiscCodelLdelay() *qdiscCodelLdelay {
	logrus.Debug("create qdiscCodelLdelay")
	return &qdiscCodelLdelay{
		NewMetrics(
			"qdisc_codel_ldelay",
			"Codel ldelay xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscCodelLdelay) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscCodelLastCount struct {
	*baseMetrics
}

func newQdiscCodelLastCount() *qdiscCodelLastCount {
	logrus.Debug("create qdiscCodelLastCount")
	return &qdiscCodelLastCount{
		NewMetrics(
			"qdisc_codel_last_count",
			"Codel last count xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscCodelLastCount) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscCodelMaxPacket struct {
	*baseMetrics
}

func newQdiscCodelMaxPacket() *qdiscCodelMaxPacket {
	logrus.Debug("create qdiscCodelMaxPacket")
	return &qdiscCodelMaxPacket{
		NewMetrics(
			"qdisc_codel_max_packet",
			"Codel max packet xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscCodelMaxPacket) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"tc_exporter/internal/exporter"
	"tc_exporter/internal/tc"
)

func init() {
	exporter.Register(
		NewQdiscSfb())
}

type QdiscSfb struct {
	qdiscSfbAvgProbe
	qdiscSfbBucketDrop
	qdiscSfbChildDrop
	qdiscSfbEarlyDrop
	qdiscSfbMarked
	qdiscSfbMaxProb
	qdiscSfbMaxQlen
	qdiscSfbPenaltyDrop
	qdiscSfbQueueDrop
}

func NewQdiscSfb() *QdiscSfb {
	return &QdiscSfb{
		qdiscSfbAvgProbe: *newQdiscSfbAvgProbe(),
	}
}

func (qd *QdiscSfb) Collect(ch chan<- prometheus.Metric) {
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
				if qdisc.Kind != "sfb" {
					continue
				}
				if qdisc.XStats == nil {
					continue
				}
				if qdisc.XStats.Sfb == nil {
					continue
				}
				qd.qdiscSfbAvgProbe.Collect(ch,
					float64(qdisc.XStats.Sfb.AvgProb),
					[]string{ns,
						device.Attributes.Name,
						"sfb"})

				qd.qdiscSfbBucketDrop.Collect(ch,
					float64(qdisc.XStats.Sfb.BucketDrop),
					[]string{ns,
						device.Attributes.Name,
						"sfb"})

				qd.qdiscSfbChildDrop.Collect(ch,
					float64(qdisc.XStats.Sfb.ChildDrop),
					[]string{ns,
						device.Attributes.Name,
						"sfb"})

				qd.qdiscSfbEarlyDrop.Collect(ch,
					float64(qdisc.XStats.Sfb.EarlyDrop),
					[]string{ns,
						device.Attributes.Name,
						"sfb"})

				qd.qdiscSfbMarked.Collect(ch,
					float64(qdisc.XStats.Sfb.Marked),
					[]string{ns,
						device.Attributes.Name,
						"sfb"})

				qd.qdiscSfbMaxProb.Collect(ch,
					float64(qdisc.XStats.Sfb.MaxProb),
					[]string{ns,
						device.Attributes.Name,
						"sfb"})

				qd.qdiscSfbMaxQlen.Collect(ch,
					float64(qdisc.XStats.Sfb.MaxQlen),
					[]string{ns,
						device.Attributes.Name,
						"sfb"})

				qd.qdiscSfbPenaltyDrop.Collect(ch,
					float64(qdisc.XStats.Sfb.PenaltyDrop),
					[]string{ns,
						device.Attributes.Name,
						"sfb"})

				qd.qdiscSfbQueueDrop.Collect(ch,
					float64(qdisc.XStats.Sfb.QueueDrop),
					[]string{ns,
						device.Attributes.Name,
						"sfb"})

			}
		}
	}
}

type qdiscSfbAvgProbe struct {
	*baseMetrics
}

func newQdiscSfbAvgProbe() *qdiscSfbAvgProbe {
	logrus.Debug("create qdiscPieAvgDqRate")
	return &qdiscSfbAvgProbe{
		NewMetrics(
			"qdisc_sfb_avg_probe",
			"SFB avg probe xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscSfbAvgProbe) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscSfbBucketDrop struct {
	*baseMetrics
}

func newQdiscSfbBucketDrop() *qdiscSfbBucketDrop {
	logrus.Debug("create qdiscPieAvgDqRate")
	return &qdiscSfbBucketDrop{
		NewMetrics(
			"qdisc_sfb_bucket_drop",
			"SFB bucket drop xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscSfbBucketDrop) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscSfbChildDrop struct {
	*baseMetrics
}

func newQdiscSfbChildDrop() *qdiscSfbChildDrop {
	logrus.Debug("create qdiscPieAvgDqRate")
	return &qdiscSfbChildDrop{
		NewMetrics(
			"qdisc_sfb_child_drop",
			"SFB child drop xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscSfbChildDrop) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscSfbEarlyDrop struct {
	*baseMetrics
}

func newQdiscSfbEarlyDrop() *qdiscSfbEarlyDrop {
	logrus.Debug("create qdiscPieAvgDqRate")
	return &qdiscSfbEarlyDrop{
		NewMetrics(
			"qdisc_sfb_early_drop",
			"SFB early drop xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscSfbEarlyDrop) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscSfbMarked struct {
	*baseMetrics
}

func newQdiscSfbMarked() *qdiscSfbMarked {
	logrus.Debug("create qdiscPieAvgDqRate")
	return &qdiscSfbMarked{
		NewMetrics(
			"qdisc_sfb_marked",
			"SFB marked xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscSfbMarked) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscSfbMaxProb struct {
	*baseMetrics
}

func newQdiscSfbMaxProb() *qdiscSfbMaxProb {
	logrus.Debug("create qdiscPieAvgDqRate")
	return &qdiscSfbMaxProb{
		NewMetrics(
			"qdisc_sfb_max_prob",
			"SFB max prob xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscSfbMaxProb) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscSfbMaxQlen struct {
	*baseMetrics
}

func newQdiscSfbMaxQlen() *qdiscSfbMaxQlen {
	logrus.Debug("create qdiscPieAvgDqRate")
	return &qdiscSfbMaxQlen{
		NewMetrics(
			"qdisc_sfb_max_qlen",
			"SFB max qlen xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscSfbMaxQlen) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscSfbPenaltyDrop struct {
	*baseMetrics
}

func newQdiscSfbPenaltyDrop() *qdiscSfbPenaltyDrop {
	logrus.Debug("create qdiscPieAvgDqRate")
	return &qdiscSfbPenaltyDrop{
		NewMetrics(
			"qdisc_sfb_penalty_drop",
			"SFB penalty drop xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscSfbPenaltyDrop) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type qdiscSfbQueueDrop struct {
	*baseMetrics
}

func newQdiscSfbQueueDrop() *qdiscSfbQueueDrop {
	logrus.Debug("create qdiscPieAvgDqRate")
	return &qdiscSfbQueueDrop{
		NewMetrics(
			"qdisc_sfb_queue_drop",
			"SFB queue drop xstat",
			[]string{"namespace",
				"device",
				"kind"})}
}

func (qd *qdiscSfbQueueDrop) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

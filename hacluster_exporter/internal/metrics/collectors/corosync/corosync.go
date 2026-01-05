package corosync

import (
	"hacluster_exporter/internal/metrics/collectors/core"
	"hacluster_exporter/pkg/utils"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

const subsystem = "corosync"

type corosyncMetrics struct {
	quorateDesc     *prometheus.Desc
	ringsDesc       *prometheus.Desc
	ringErrorsDesc  *prometheus.Desc
	quorumVotesDesc *prometheus.Desc
	memberVotesDesc *prometheus.Desc
}

func NewCollector(cfgToolPath string, quorumToolPath string, timestamps bool) (*CorosyncCollector, error) {
	err := core.CheckExecutables(cfgToolPath, quorumToolPath)
	if err != nil {
		return nil, errors.Wrapf(err, "could not initialize '%s' collector", subsystem)
	}

	c := &CorosyncCollector{
		DefaultCollector: core.NewDefaultCollector(subsystem, timestamps),
		cfgToolPath:      cfgToolPath,
		quorumToolPath:   quorumToolPath,
		parser:           NewParser(),
		metrics: corosyncMetrics{
			quorateDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "quorate"),
				"Whether or not the cluster is quorate",
				nil,
				nil,
			),
			ringsDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "rings"),
				"The status of each Corosync ring; 1 means healthy, 0 means faulty.",
				[]string{"ring_id", "node_id", "number", "address"},
				nil,
			),
			ringErrorsDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "ring_errors"),
				"The total number of faulty corosync rings",
				nil,
				nil,
			),
			memberVotesDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "member_votes"),
				"How many votes each member node has contributed with to the current quorum",
				[]string{"node_id", "node", "local"},
				nil,
			),
			quorumVotesDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "quorum_votes"),
				"Cluster quorum votes; one line per type",
				[]string{"type"},
				nil,
			),
		},
	}

	return c, err
}

type CorosyncCollector struct {
	core.DefaultCollector
	cfgToolPath    string
	quorumToolPath string
	parser         Parser
	metrics        corosyncMetrics
}

func (c *CorosyncCollector) CollectWithError(ch chan<- prometheus.Metric) error {
	logrus.Debug("Collecting corosync metrics...")

	// We suppress the exec errors because if any interface is faulty the tools will exit with code 1, but we still want to parse the output.
	cfgToolOutput, _ := utils.RunCommand(c.cfgToolPath, "-s")
	quorumToolOutput, _ := utils.RunCommand(c.quorumToolPath, "-p")

	status, err := c.parser.Parse(cfgToolOutput, quorumToolOutput)
	if err != nil {
		return errors.Wrap(err, "corosync parser error")
	}

	c.collectRings(status, ch)
	c.collectRingErrors(status, ch)
	c.collectQuorate(status, ch)
	c.collectQuorumVotes(status, ch)
	c.collectMemberVotes(status, ch)

	return nil
}

func (c *CorosyncCollector) Collect(ch chan<- prometheus.Metric) {
	// level.Debug(c.Logger).Log("msg", "Collecting corosync metrics...")

	err := c.CollectWithError(ch)
	if err != nil {
		logrus.Warn("collector scrape failed",
			"subsystem", c.GetSubsystem(),
			"error", err,
		)
	}
}

func (c *CorosyncCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.metrics.quorateDesc
	ch <- c.metrics.ringsDesc
	ch <- c.metrics.ringErrorsDesc
	ch <- c.metrics.memberVotesDesc
	ch <- c.metrics.quorumVotesDesc
}

func (c *CorosyncCollector) collectQuorumVotes(status *Status, ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(
		c.metrics.quorumVotesDesc,
		prometheus.GaugeValue,
		float64(status.QuorumVotes.ExpectedVotes),
		"expected_votes",
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics.quorumVotesDesc,
		prometheus.GaugeValue,
		float64(status.QuorumVotes.HighestExpected),
		"highest_expected",
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics.quorumVotesDesc,
		prometheus.GaugeValue,
		float64(status.QuorumVotes.TotalVotes),
		"total_votes",
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics.quorumVotesDesc,
		prometheus.GaugeValue,
		float64(status.QuorumVotes.Quorum),
		"quorum",
	)
}

func (c *CorosyncCollector) collectQuorate(status *Status, ch chan<- prometheus.Metric) {
	var quorate float64
	if status.Quorate {
		quorate = 1
	}
	ch <- prometheus.MustNewConstMetric(
		c.metrics.quorateDesc,
		prometheus.GaugeValue,
		quorate,
	)
}

func (c *CorosyncCollector) collectRingErrors(status *Status, ch chan<- prometheus.Metric) {
	var numErrors float64
	for _, ring := range status.Rings {
		if ring.Faulty {
			numErrors += 1
		}
	}
	ch <- prometheus.MustNewConstMetric(
		c.metrics.ringErrorsDesc,
		prometheus.GaugeValue,
		numErrors,
	)
}

func (c *CorosyncCollector) collectRings(status *Status, ch chan<- prometheus.Metric) {
	for _, ring := range status.Rings {
		var healthy float64 = 1
		if ring.Faulty {
			healthy = 0
		}
		ch <- prometheus.MustNewConstMetric(
			c.metrics.ringsDesc,
			prometheus.GaugeValue,
			healthy,
			status.RingId, status.NodeId, ring.Number, ring.Address,
		)
	}
}

func (c *CorosyncCollector) collectMemberVotes(status *Status, ch chan<- prometheus.Metric) {
	for _, member := range status.Members {
		local := "false"
		if member.Local {
			local = "true"
		}
		ch <- prometheus.MustNewConstMetric(
			c.metrics.memberVotesDesc,
			prometheus.GaugeValue,
			float64(member.Votes),
			member.Id, member.Name, local,
		)
	}
}

package corosync

import (
	"strings"
	"testing"

	"hacluster_exporter/internal/metrics/collectors/core"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

type mockParser struct {
	status *Status
	err    error
}

func (m *mockParser) Parse(cfgOutput, quorumOutput []byte) (*Status, error) {
	return m.status, m.err
}

func newTestMetrics() corosyncMetrics {
	return corosyncMetrics{
		quorateDesc: prometheus.NewDesc(
			prometheus.BuildFQName(core.NAMESPACE, "corosync", "quorate"),
			"Whether or not the cluster is quorate",
			nil,
			nil,
		),
		ringsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(core.NAMESPACE, "corosync", "rings"),
			"The status of each Corosync ring; 1 means healthy, 0 means faulty.",
			[]string{"ring_id", "node_id", "number", "address"},
			nil,
		),
		ringErrorsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(core.NAMESPACE, "corosync", "ring_errors"),
			"The total number of faulty corosync rings",
			nil,
			nil,
		),
		memberVotesDesc: prometheus.NewDesc(
			prometheus.BuildFQName(core.NAMESPACE, "corosync", "member_votes"),
			"How many votes each member node has contributed with to the current quorum",
			[]string{"node_id", "node", "local"},
			nil,
		),
		quorumVotesDesc: prometheus.NewDesc(
			prometheus.BuildFQName(core.NAMESPACE, "corosync", "quorum_votes"),
			"Cluster quorum votes; one line per type",
			[]string{"type"},
			nil,
		),
	}
}

func TestCorosyncCollector_Collect(t *testing.T) {
	status := &Status{
		NodeId: "node1",
		RingId: "ring1",
		Rings: []Ring{
			{Number: "0", Address: "10.0.0.1", Faulty: false},
			{Number: "1", Address: "10.0.0.2", Faulty: true},
		},
		QuorumVotes: QuorumVotes{
			ExpectedVotes:   10,
			HighestExpected: 8,
			TotalVotes:      7,
			Quorum:          5,
		},
		Quorate: true,
		Members: []Member{
			{Id: "node1", Name: "host1", Qdevice: "A", Votes: 3, Local: true},
			{Id: "node2", Name: "host2", Qdevice: "NR", Votes: 4, Local: false},
		},
	}

	mockParser := &mockParser{status: status, err: nil}
	c := &CorosyncCollector{
		DefaultCollector: core.NewDefaultCollector("corosync", false),
		cfgToolPath:      "/fake/path",
		quorumToolPath:   "/fake/path",
		parser:           mockParser,
		metrics:          newTestMetrics(),
	}

	expected := `
# HELP hacluster_corosync_member_votes How many votes each member node has contributed with to the current quorum
# TYPE hacluster_corosync_member_votes gauge
hacluster_corosync_member_votes{local="true",node="host1",node_id="node1"} 3
hacluster_corosync_member_votes{local="false",node="host2",node_id="node2"} 4
# HELP hacluster_corosync_quorate Whether or not the cluster is quorate
# TYPE hacluster_corosync_quorate gauge
hacluster_corosync_quorate 1
# HELP hacluster_corosync_quorum_votes Cluster quorum votes; one line per type
# TYPE hacluster_corosync_quorum_votes gauge
hacluster_corosync_quorum_votes{type="expected_votes"} 10
hacluster_corosync_quorum_votes{type="highest_expected"} 8
hacluster_corosync_quorum_votes{type="total_votes"} 7
hacluster_corosync_quorum_votes{type="quorum"} 5
# HELP hacluster_corosync_ring_errors The total number of faulty corosync rings
# TYPE hacluster_corosync_ring_errors gauge
hacluster_corosync_ring_errors 1
# HELP hacluster_corosync_rings The status of each Corosync ring; 1 means healthy, 0 means faulty.
# TYPE hacluster_corosync_rings gauge
hacluster_corosync_rings{address="10.0.0.1",node_id="node1",number="0",ring_id="ring1"} 1
hacluster_corosync_rings{address="10.0.0.2",node_id="node1",number="1",ring_id="ring1"} 0
`

	err := testutil.CollectAndCompare(c, strings.NewReader(expected))
	assert.NoError(t, err)
}

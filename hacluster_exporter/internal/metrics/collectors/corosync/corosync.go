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


// TODO: implement functions

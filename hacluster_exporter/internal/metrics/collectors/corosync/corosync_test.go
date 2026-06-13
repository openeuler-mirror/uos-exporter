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


// TODO: implement functions

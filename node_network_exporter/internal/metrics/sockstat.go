package metrics

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"node_network_exporter/internal/exporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
)

const (
	sockStatSubsystem = "sockstat"
)

// Used for calculating the total memory bytes on TCP and UDP.
var pageSize = os.Getpagesize()


// TODO: implement functions

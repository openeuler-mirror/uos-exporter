package metrics

import (
	"log/slog"

	"node_network_exporter/internal/exporter"
	"github.com/jsimonetti/rtnetlink/v2/rtnl"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
	"golang.org/x/sys/unix"
)

var (
	arpDeviceInclude = ""
	arpDeviceExclude = ""
	arpNetlink       = true
)


// TODO: implement functions

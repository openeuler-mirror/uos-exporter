package metrics

import (
	"node_network_exporter/internal/exporter"
	"github.com/prometheus/client_golang/prometheus"
	"fmt"
	"net"
	"strconv"
	"sync"
	"regexp"
	"log/slog"
	"github.com/jsimonetti/rtnetlink/v2"
	"github.com/prometheus/procfs"
	"github.com/prometheus/procfs/sysfs"
)

var (
	netDevNetlink      = true
	netdevLabelIfAlias = false
	netdevDeviceInclude = ""
	netdevDeviceExclude = ""
	netdevAddressInfo   = false
	netdevDetailedMetrics = false
	procPath = "/proc"
)


// TODO: implement functions

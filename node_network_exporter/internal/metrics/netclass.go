package metrics

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"regexp"
	"sync"
	"node_network_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs/sysfs"
)

var (
	netclassIgnoredDevices = "^$"
	netclassInvalidSpeed   = false
	netclassNetlink        = false
	sysPath                = "/sys"
)


// TODO: implement functions

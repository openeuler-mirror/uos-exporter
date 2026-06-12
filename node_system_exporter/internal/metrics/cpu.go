package metrics

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"node_system_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
	"github.com/prometheus/procfs/sysfs"
)

const (
	cpuCollectorSubsystem = "cpu"
	jumpBackSeconds       = 3.0
)


// TODO: implement

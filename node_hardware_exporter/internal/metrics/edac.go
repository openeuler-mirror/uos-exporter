package metrics

import (
	"fmt"
	"node_hardware_exporter/internal/exporter"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	edacMemControllerRE = regexp.MustCompile(`.*devices/system/edac/mc/mc([0-9]*)`)
	edacMemCsrowRE      = regexp.MustCompile(`.*devices/system/edac/mc/mc[0-9]*/csrow([0-9]*)`)
)


// TODO: implement functions

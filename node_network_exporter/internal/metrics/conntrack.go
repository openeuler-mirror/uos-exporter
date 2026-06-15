package metrics

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"node_network_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
)


// TODO: implement functions

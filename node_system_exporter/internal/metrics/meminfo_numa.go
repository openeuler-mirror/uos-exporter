package metrics

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"node_system_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
)


// TODO: implement functions

package metrics

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"node_storage_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	exporter.Register(NewZFSCollector())
}

var errZFSNotAvailable = errors.New("ZFS / ZFS statistics are not available")

type ZFSCollector struct {
	logger *slog.Logger
	descs  map[string]*prometheus.Desc
}

func NewZFSCollector() *ZFSCollector {
	logger := slog.Default()

	const subsystem = "zfs"

	return &ZFSCollector{
		logger: logger,
		descs: map[string]*prometheus.Desc{
			"arc_hits_total": prometheus.NewDesc(
				prometheus.BuildFQName("node", subsystem, "arc_hits_total"),
				"Total number of ZFS ARC hits.",
				nil, nil,
			),
			"arc_misses_total": prometheus.NewDesc(
				prometheus.BuildFQName("node", subsystem, "arc_misses_total"),
				"Total number of ZFS ARC misses.",
				nil, nil,
			),
			"arc_size_bytes": prometheus.NewDesc(
				prometheus.BuildFQName("node", subsystem, "arc_size_bytes"),
				"Size of ZFS ARC in bytes.",
				nil, nil,
			),
			"allocated_bytes": prometheus.NewDesc(
				prometheus.BuildFQName("node", subsystem, "allocated_bytes"),
				"Allocated bytes for a ZFS dataset.",
				[]string{"dataset", "type"}, nil,
			),
			"free_bytes": prometheus.NewDesc(
				prometheus.BuildFQName("node", subsystem, "free_bytes"),
				"Free bytes for a ZFS dataset.",
				[]string{"dataset", "type"}, nil,
			),
			"size_bytes": prometheus.NewDesc(
				prometheus.BuildFQName("node", subsystem, "size_bytes"),
				"Size of a ZFS dataset.",
				[]string{"dataset", "type"}, nil,
			),
		},
	}
}

func (c *ZFSCollector) Collect(ch chan<- prometheus.Metric) {
	// Try to read ZFS statistics
	if err := c.updateZFSStats(ch); err != nil {
		if err == errZFSNotAvailable {
			c.logger.Debug("ZFS not available")
		} else {
			c.logger.Debug("Error reading ZFS stats", "error", err)
		}
	}
}

func (c *ZFSCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, desc := range c.descs {
		ch <- desc
	}
}

func (c *ZFSCollector) updateZFSStats(ch chan<- prometheus.Metric) error {
	// Check if ZFS is available
	file, err := c.openProcFile("spl/kstat/zfs/arcstats")
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}

		name := fields[0]
		valueStr := fields[2]
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			continue
		}

		switch name {
		case "hits":
			ch <- prometheus.MustNewConstMetric(
				c.descs["arc_hits_total"], prometheus.CounterValue, value,
			)
		case "misses":
			ch <- prometheus.MustNewConstMetric(
				c.descs["arc_misses_total"], prometheus.CounterValue, value,
			)
		case "size":
			ch <- prometheus.MustNewConstMetric(
				c.descs["arc_size_bytes"], prometheus.GaugeValue, value,
			)
		}
	}

	return scanner.Err()
}

func (c *ZFSCollector) openProcFile(path string) (*os.File, error) {
	fullPath := filepath.Join("/proc", path)
	cleanPath := filepath.Clean(fullPath)
	statDir := "/proc"
	if !strings.HasPrefix(cleanPath, statDir) {
		return nil, fmt.Errorf("stat file must be located within %s", statDir)
	}
	file, err := os.Open(fullPath)
	if err != nil {
		c.logger.Debug("Cannot open file for reading", "path", fullPath)
		return nil, errZFSNotAvailable
	}
	return file, nil
}

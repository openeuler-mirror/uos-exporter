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

func init() {
	exporter.Register(NewConntrackCollector())
}

type conntrackCollector struct {
	current       *prometheus.Desc
	limit         *prometheus.Desc
	found         *prometheus.Desc
	invalid       *prometheus.Desc
	ignore        *prometheus.Desc
	insert        *prometheus.Desc
	insertFailed  *prometheus.Desc
	drop          *prometheus.Desc
	earlyDrop     *prometheus.Desc
	searchRestart *prometheus.Desc
	logger        *slog.Logger
}

type conntrackStatistics struct {
	found         uint64 // Number of searched entries which were successful
	invalid       uint64 // Number of packets seen which can not be tracked
	ignore        uint64 // Number of packets seen which are already connected to a conntrack entry
	insert        uint64 // Number of entries inserted into the list
	insertFailed  uint64 // Number of entries for which list insertion was attempted but failed (happens if the same entry is already present)
	drop          uint64 // Number of packets dropped due to conntrack failure. Either new conntrack entry allocation failed, or protocol helper dropped the packet
	earlyDrop     uint64 // Number of dropped conntrack entries to make room for new ones, if maximum table size was reached
	searchRestart uint64 // Number of conntrack table lookups which had to be restarted due to hashtable resizes
}

// NewConntrackCollector returns a new Collector exposing conntrack stats.
func NewConntrackCollector() *conntrackCollector {
	return &conntrackCollector{
		current: prometheus.NewDesc(
			prometheus.BuildFQName("node", "", "nf_conntrack_entries"),
			"Number of currently allocated flow entries for connection tracking.",
			nil, nil,
		),
		limit: prometheus.NewDesc(
			prometheus.BuildFQName("node", "", "nf_conntrack_entries_limit"),
			"Maximum size of connection tracking table.",
			nil, nil,
		),
		found: prometheus.NewDesc(
			prometheus.BuildFQName("node", "", "nf_conntrack_stat_found"),
			"Number of searched entries which were successful.",
			nil, nil,
		),
		invalid: prometheus.NewDesc(
			prometheus.BuildFQName("node", "", "nf_conntrack_stat_invalid"),
			"Number of packets seen which can not be tracked.",
			nil, nil,
		),
		ignore: prometheus.NewDesc(
			prometheus.BuildFQName("node", "", "nf_conntrack_stat_ignore"),
			"Number of packets seen which are already connected to a conntrack entry.",
			nil, nil,
		),
		insert: prometheus.NewDesc(
			prometheus.BuildFQName("node", "", "nf_conntrack_stat_insert"),
			"Number of entries inserted into the list.",
			nil, nil,
		),
		insertFailed: prometheus.NewDesc(
			prometheus.BuildFQName("node", "", "nf_conntrack_stat_insert_failed"),
			"Number of entries for which list insertion was attempted but failed.",
			nil, nil,
		),
		drop: prometheus.NewDesc(
			prometheus.BuildFQName("node", "", "nf_conntrack_stat_drop"),
			"Number of packets dropped due to conntrack failure.",
			nil, nil,
		),
		earlyDrop: prometheus.NewDesc(
			prometheus.BuildFQName("node", "", "nf_conntrack_stat_early_drop"),
			"Number of dropped conntrack entries to make room for new ones, if maximum table size was reached.",
			nil, nil,
		),
		searchRestart: prometheus.NewDesc(
			prometheus.BuildFQName("node", "", "nf_conntrack_stat_search_restart"),
			"Number of conntrack table lookups which had to be restarted due to hashtable resizes.",
			nil, nil,
		),
		logger: slog.Default(),
	}
}

func (c *conntrackCollector) Collect(ch chan<- prometheus.Metric) {
	value, err := c.readUintFromFile("/proc/sys/net/netfilter/nf_conntrack_count")
	if err != nil {
		if c.handleErr(err) != nil {
			return
		}
	} else {
		ch <- prometheus.MustNewConstMetric(
			c.current, prometheus.GaugeValue, float64(value))
	}

	value, err = c.readUintFromFile("/proc/sys/net/netfilter/nf_conntrack_max")
	if err != nil {
		if c.handleErr(err) != nil {
			return
		}
	} else {
		ch <- prometheus.MustNewConstMetric(
			c.limit, prometheus.GaugeValue, float64(value))
	}

	conntrackStats, err := c.getConntrackStatistics()
	if err != nil {
		if c.handleErr(err) != nil {
			return
		}
	} else {
		ch <- prometheus.MustNewConstMetric(
			c.found, prometheus.GaugeValue, float64(conntrackStats.found))
		ch <- prometheus.MustNewConstMetric(
			c.invalid, prometheus.GaugeValue, float64(conntrackStats.invalid))
		ch <- prometheus.MustNewConstMetric(
			c.ignore, prometheus.GaugeValue, float64(conntrackStats.ignore))
		ch <- prometheus.MustNewConstMetric(
			c.insert, prometheus.GaugeValue, float64(conntrackStats.insert))
		ch <- prometheus.MustNewConstMetric(
			c.insertFailed, prometheus.GaugeValue, float64(conntrackStats.insertFailed))
		ch <- prometheus.MustNewConstMetric(
			c.drop, prometheus.GaugeValue, float64(conntrackStats.drop))
		ch <- prometheus.MustNewConstMetric(
			c.earlyDrop, prometheus.GaugeValue, float64(conntrackStats.earlyDrop))
		ch <- prometheus.MustNewConstMetric(
			c.searchRestart, prometheus.GaugeValue, float64(conntrackStats.searchRestart))
	}
}

func (c *conntrackCollector) handleErr(err error) error {
	if errors.Is(err, os.ErrNotExist) {
		c.logger.Debug("conntrack probably not loaded")
		return nil
	}
	c.logger.Error("failed to retrieve conntrack stats", "error", err)
	return err
}

func (c *conntrackCollector) readUintFromFile(path string) (uint64, error) {
	cleanPath := filepath.Clean(path)
	statDir := "/proc/sys"
	if !strings.HasPrefix(cleanPath, statDir) {
		return 0, fmt.Errorf("stat file must be located within %s", statDir)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	value, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, err
	}
	return value, nil
}

func (c *conntrackCollector) getConntrackStatistics() (*conntrackStatistics, error) {
	stats := conntrackStatistics{}

	fs, err := procfs.NewFS("/proc")
	if err != nil {
		return nil, fmt.Errorf("failed to open procfs: %w", err)
	}

	connStats, err := fs.ConntrackStat()
	if err != nil {
		return nil, err
	}

	for _, connStat := range connStats {
		stats.found += connStat.Found
		stats.invalid += connStat.Invalid
		stats.ignore += connStat.Ignore
		stats.insert += connStat.Insert
		stats.insertFailed += connStat.InsertFailed
		stats.drop += connStat.Drop
		stats.earlyDrop += connStat.EarlyDrop
		stats.searchRestart += connStat.SearchRestart
	}

	return &stats, nil
}

func (c *conntrackCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.current
	ch <- c.limit
	ch <- c.found
	ch <- c.invalid
	ch <- c.ignore
	ch <- c.insert
	ch <- c.insertFailed
	ch <- c.drop
	ch <- c.earlyDrop
	ch <- c.searchRestart
}

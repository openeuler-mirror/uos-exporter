package metrics

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"node_network_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	netStatsSubsystem = "netstat"
)

var (
	netStatFields = "^(.*_(InErrors|InErrs)|Ip_Forwarding|Ip(6|Ext)_(InOctets|OutOctets)|Icmp6?_(InMsgs|OutMsgs)|TcpExt_(Listen.*|Syncookies.*|TCPSynRetrans|TCPTimeouts|TCPOFOQueue|TCPRcvQDrop)|Tcp_(ActiveOpens|InSegs|OutSegs|OutRsts|PassiveOpens|RetransSegs|CurrEstab)|Udp6?_(InDatagrams|OutDatagrams|NoPorts|RcvbufErrors|SndbufErrors))$"
)

func init() {
	exporter.Register(NewNetStatCollector())
}

type netStatCollector struct {
	fieldPattern *regexp.Regexp
	logger       *slog.Logger
}

// NewNetStatCollector takes and returns
// a new Collector exposing network stats.
func NewNetStatCollector() *netStatCollector {
	pattern := regexp.MustCompile(netStatFields)
	return &netStatCollector{
		fieldPattern: pattern,
		logger:       slog.Default(),
	}
}

func (c *netStatCollector) Collect(ch chan<- prometheus.Metric) {
	netStats, err := c.getNetStats("/proc/net/netstat")
	if err != nil {
		c.logger.Error("couldn't get netstats", "error", err)
		return
	}

	snmpStats, err := c.getNetStats("/proc/net/snmp")
	if err != nil {
		c.logger.Error("couldn't get SNMP stats", "error", err)
		return
	}

	snmp6Stats, err := c.getSNMP6Stats("/proc/net/snmp6")
	if err != nil {
		c.logger.Error("couldn't get SNMP6 stats", "error", err)
		return
	}

	// Merge the results of snmpStats into netStats (collisions are possible, but
	// we know that the keys are always unique for the given use case).
	for k, v := range snmpStats {
		netStats[k] = v
	}
	for k, v := range snmp6Stats {
		netStats[k] = v
	}

	for protocol, protocolStats := range netStats {
		for name, value := range protocolStats {
			key := protocol + "_" + name
			v, err := strconv.ParseFloat(value, 64)
			if err != nil {
				c.logger.Error("invalid value in netstats", "value", value, "error", err)
				continue
			}
			if !c.fieldPattern.MatchString(key) {
				continue
			}
			ch <- prometheus.MustNewConstMetric(
				prometheus.NewDesc(
					prometheus.BuildFQName("node", netStatsSubsystem, key),
					fmt.Sprintf("Statistic %s.", protocol+name),
					nil, nil,
				),
				prometheus.UntypedValue, v,
			)
		}
	}
}

func (c *netStatCollector) getNetStats(fileName string) (map[string]map[string]string, error) {
	cleanPath := filepath.Clean(fileName)
	statDir := "/proc/net"
	if !strings.HasPrefix(cleanPath, statDir) {
		return nil, fmt.Errorf("stat file must be located within %s", statDir)
	}
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return c.parseNetStats(file, fileName)
}

func (c *netStatCollector) parseNetStats(r io.Reader, fileName string) (map[string]map[string]string, error) {
	var (
		netStats = map[string]map[string]string{}
		scanner  = bufio.NewScanner(r)
	)

	for scanner.Scan() {
		nameParts := strings.Split(scanner.Text(), " ")
		scanner.Scan()
		valueParts := strings.Split(scanner.Text(), " ")
		// Remove trailing :.
		protocol := nameParts[0][:len(nameParts[0])-1]
		netStats[protocol] = map[string]string{}
		if len(nameParts) != len(valueParts) {
			return nil, fmt.Errorf("mismatch field count mismatch in %s: %s",
				fileName, protocol)
		}
		for i := 1; i < len(nameParts); i++ {
			netStats[protocol][nameParts[i]] = valueParts[i]
		}
	}

	return netStats, scanner.Err()
}

func (c *netStatCollector) getSNMP6Stats(fileName string) (map[string]map[string]string, error) {
	cleanPath := filepath.Clean(fileName)
	statDir := "/proc/net"
	if !strings.HasPrefix(cleanPath, statDir) {
		return nil, fmt.Errorf("stat file must be located within %s", statDir)
	}
	file, err := os.Open(fileName)
	if err != nil {
		// On systems with IPv6 disabled, this file won't exist.
		// Do nothing.
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, err
	}
	defer file.Close()

	return c.parseSNMP6Stats(file)
}

func (c *netStatCollector) parseSNMP6Stats(r io.Reader) (map[string]map[string]string, error) {
	var (
		netStats = map[string]map[string]string{}
		scanner  = bufio.NewScanner(r)
	)

	for scanner.Scan() {
		stat := strings.Fields(scanner.Text())
		if len(stat) < 2 {
			continue
		}
		// Expect to have "6" in metric name, skip line otherwise
		if sixIndex := strings.Index(stat[0], "6"); sixIndex != -1 {
			protocol := stat[0][:sixIndex+1]
			name := stat[0][sixIndex+1:]
			if _, present := netStats[protocol]; !present {
				netStats[protocol] = map[string]string{}
			}
			netStats[protocol][name] = stat[1]
		}
	}

	return netStats, scanner.Err()
}

func (c *netStatCollector) Describe(ch chan<- *prometheus.Desc) {
	// netstat collector creates dynamic descriptors
}

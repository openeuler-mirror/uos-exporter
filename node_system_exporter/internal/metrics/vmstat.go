package metrics

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strconv"
	"strings"

	"node_system_exporter/internal/exporter"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	exporter.Register(NewVMStatCollector())
}

type VMStatCollector struct {
	*baseMetrics
	fieldPattern *regexp.Regexp
	logger       *slog.Logger
}

func NewVMStatCollector() *VMStatCollector {
	// Match important vmstat fields like oom_kill, page faults, swapping, etc.
	pattern := regexp.MustCompile(`^(oom_kill|pgpg|pswp|pg.*fault).*`)
	
	return &VMStatCollector{
		fieldPattern: pattern,
		logger:       slog.Default(),
	}
}

func (c *VMStatCollector) Collect(ch chan<- prometheus.Metric) {
	file, err := os.Open("/proc/vmstat")
	if err != nil {
		c.logger.Error("Error opening /proc/vmstat", "error", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) != 2 {
			continue
		}
		
		value, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			c.logger.Debug("Error parsing vmstat value", "field", parts[0], "value", parts[1], "error", err)
			continue
		}
		
		if !c.fieldPattern.MatchString(parts[0]) {
			continue
		}

		desc := prometheus.NewDesc(
			prometheus.BuildFQName("node", "vmstat", parts[0]),
			fmt.Sprintf("/proc/vmstat information field %s.", parts[0]),
			nil, nil,
		)
		
		ch <- prometheus.MustNewConstMetric(
			desc,
			prometheus.UntypedValue,
			value,
		)
	}
	
	if err := scanner.Err(); err != nil {
		c.logger.Error("Error reading /proc/vmstat", "error", err)
	}
}

func (c *VMStatCollector) Describe(ch chan<- *prometheus.Desc) {
	// VMStat collector creates dynamic descriptors
} 
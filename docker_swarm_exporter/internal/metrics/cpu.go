//go:build example

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"os"
	"strconv"
	"strings"
	"docker_swarm_exporter/internal/exporter"
)

func init() {
	exporter.Register(
		NewCpu("cpu_usage",
			"cpu usage",
			[]string{"name"}))
}

type Cpu struct {
	*baseMetrics
}

func NewCpu(fqname, help string, labels []string) *Cpu {
	return &Cpu{NewMetrics(fqname, help, labels)}
}

func (c *Cpu) Collect(ch chan<- prometheus.Metric) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		logrus.Warnf("Read /proc/stat failed: %v", err)
		return
	}

	// 解析 CPU 使用率
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "cpu") {
			fields := strings.Fields(line)
			if len(fields) < 8 {
				continue
			}

			// 解析 CPU 时间
			user, _ := strconv.ParseUint(fields[1], 10, 64)
			nice, _ := strconv.ParseUint(fields[2], 10, 64)
			system, _ := strconv.ParseUint(fields[3], 10, 64)
			idle, _ := strconv.ParseUint(fields[4], 10, 64)

			total := user + nice + system + idle
			idlePercent := 100.0 * float64(idle) / float64(total)
			c.baseMetrics.collect(ch, 100.0-idlePercent, []string{fields[0]})
		}
	}
}

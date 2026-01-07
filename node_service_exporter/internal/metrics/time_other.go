//go:build !linux && !notime
// +build !linux,!notime

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// updateClocksources is a no-op implementation for non-Linux systems
func (c *TimeCollector) updateClocksources(ch chan<- prometheus.Metric) error {
	// No clocksource metrics available on non-Linux systems
	return nil
} 
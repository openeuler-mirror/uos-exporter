//go:build linux && !notimex
// +build linux,!notimex

package metrics

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"node_service_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sys/unix"
)

func init() {
	exporter.Register(NewTimexCollectorWrapper())
}

const (
	// The system clock is not synchronized to a reliable
	// server (TIME_ERROR).
	timeError = 5
	// The timex.Status time resolution bit (STA_NANO),
	// 0 = microsecond, 1 = nanoseconds.
	staNano = 0x2000

	// 1 second in
	nanoSeconds  = 1000000000
	microSeconds = 1000000

	// See NOTES in adjtimex(2).
	ppm16frac = 1000000.0 * 65536.0
)

// TimexCollectorWrapper wraps the old collector to work with new framework
type TimexCollectorWrapper struct {
	collector *TimexCollector
}

func NewTimexCollectorWrapper() *TimexCollectorWrapper {
	collector, err := NewTimexCollector(nil)
	if err != nil {
		return nil
	}
	return &TimexCollectorWrapper{
		collector: collector,
	}
}

func (t *TimexCollectorWrapper) Collect(ch chan<- prometheus.Metric) {
	if t.collector != nil {
		if err := t.collector.Collect(ch); err != nil {
			fmt.Printf("Error collecting metrics: %v\n", err)
		}
	}
}

// TimexCollector collects timex-related metrics
type TimexCollector struct {
	offset,
	freq,
	maxerror,
	esterror,
	status,
	constant,
	tick,
	ppsfreq,
	jitter,
	shift,
	stabil,
	jitcnt,
	calcnt,
	errcnt,
	stbcnt,
	tai,
	syncStatus typedDesc
	logger *slog.Logger
}

// NewTimexCollector creates a new timex collector
func NewTimexCollector(logger *slog.Logger) (*TimexCollector, error) {
	if logger == nil {
		logger = slog.Default()
	}

	const subsystem = "timex"

	return &TimexCollector{
		offset: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, subsystem, "offset_seconds"),
				"Time offset in between local system and reference clock.",
				nil,
				nil,
			),
			valueType: prometheus.GaugeValue,
		},
		freq: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, subsystem, "frequency_adjustment_ratio"),
				"Local clock frequency adjustment.",
				nil,
				nil,
			),
			valueType: prometheus.GaugeValue,
		},
		maxerror: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, subsystem, "maxerror_seconds"),
				"Maximum error in seconds.",
				nil,
				nil,
			),
			valueType: prometheus.GaugeValue,
		},
		esterror: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, subsystem, "estimated_error_seconds"),
				"Estimated error in seconds.",
				nil,
				nil,
			),
			valueType: prometheus.GaugeValue,
		},
		status: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, subsystem, "status"),
				"Value of the status array bits.",
				nil,
				nil,
			),
			valueType: prometheus.GaugeValue,
		},
		constant: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, subsystem, "loop_time_constant"),
				"Phase-locked loop time constant.",
				nil,
				nil,
			),
			valueType: prometheus.GaugeValue,
		},
		tick: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, subsystem, "tick_seconds"),
				"Seconds between clock ticks.",
				nil,
				nil,
			),
			valueType: prometheus.GaugeValue,
		},
		ppsfreq: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, subsystem, "pps_frequency_hertz"),
				"Pulse per second frequency.",
				nil,
				nil,
			),
			valueType: prometheus.GaugeValue,
		},
		jitter: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, subsystem, "pps_jitter_seconds"),
				"Pulse per second jitter.",
				nil,
				nil,
			),
			valueType: prometheus.GaugeValue,
		},
		shift: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, subsystem, "pps_shift_seconds"),
				"Pulse per second interval duration.",
				nil,
				nil,
			),
			valueType: prometheus.GaugeValue,
		},
		stabil: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, subsystem, "pps_stability_hertz"),
				"Pulse per second stability, average of recent frequency changes.",
				nil,
				nil,
			),
			valueType: prometheus.GaugeValue,
		},
		jitcnt: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, subsystem, "pps_jitter_total"),
				"Pulse per second count of jitter limit exceeded events.",
				nil,
				nil,
			),
			valueType: prometheus.CounterValue,
		},
		calcnt: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, subsystem, "pps_calibration_total"),
				"Pulse per second count of calibration intervals.",
				nil,
				nil,
			),
			valueType: prometheus.CounterValue,
		},
		errcnt: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, subsystem, "pps_error_total"),
				"Pulse per second count of calibration errors.",
				nil,
				nil,
			),
			valueType: prometheus.CounterValue,
		},
		stbcnt: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, subsystem, "pps_stability_exceeded_total"),
				"Pulse per second count of stability limit exceeded events.",
				nil,
				nil,
			),
			valueType: prometheus.CounterValue,
		},
		tai: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, subsystem, "tai_offset_seconds"),
				"International Atomic Time (TAI) offset.",
				nil,
				nil,
			),
			valueType: prometheus.GaugeValue,
		},
		syncStatus: typedDesc{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, subsystem, "sync_status"),
				"Is clock synchronized to a reliable server (1 = yes, 0 = no).",
				nil,
				nil,
			),
			valueType: prometheus.GaugeValue,
		},
		logger: logger,
	}, nil
}

// Collect implements the Collector interface
func (c *TimexCollector) Collect(ch chan<- prometheus.Metric) error {
	if c == nil {
		return fmt.Errorf("TimexCollector is nil")
	}

	var syncStatus float64
	var divisor float64
	var timex = new(unix.Timex)

	status, err := unix.Adjtimex(timex)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			c.logger.Debug("Not collecting timex metrics, permission denied", "err", err)
			return nil
		}
		return fmt.Errorf("failed to retrieve adjtimex stats: %w", err)
	}

	if status == timeError {
		syncStatus = 0
	} else {
		syncStatus = 1
	}
	if (timex.Status & staNano) != 0 {
		divisor = nanoSeconds
	} else {
		divisor = microSeconds
	}

	ch <- c.syncStatus.mustNewConstMetric(syncStatus)
	ch <- c.offset.mustNewConstMetric(float64(timex.Offset) / divisor)
	ch <- c.freq.mustNewConstMetric(1 + float64(timex.Freq)/ppm16frac)
	ch <- c.maxerror.mustNewConstMetric(float64(timex.Maxerror) / microSeconds)
	ch <- c.esterror.mustNewConstMetric(float64(timex.Esterror) / microSeconds)
	ch <- c.status.mustNewConstMetric(float64(timex.Status))
	ch <- c.constant.mustNewConstMetric(float64(timex.Constant))
	ch <- c.tick.mustNewConstMetric(float64(timex.Tick) / microSeconds)
	ch <- c.ppsfreq.mustNewConstMetric(float64(timex.Ppsfreq) / ppm16frac)
	ch <- c.jitter.mustNewConstMetric(float64(timex.Jitter) / divisor)
	ch <- c.shift.mustNewConstMetric(float64(timex.Shift))
	ch <- c.stabil.mustNewConstMetric(float64(timex.Stabil) / ppm16frac)
	ch <- c.jitcnt.mustNewConstMetric(float64(timex.Jitcnt))
	ch <- c.calcnt.mustNewConstMetric(float64(timex.Calcnt))
	ch <- c.errcnt.mustNewConstMetric(float64(timex.Errcnt))
	ch <- c.stbcnt.mustNewConstMetric(float64(timex.Stbcnt))
	ch <- c.tai.mustNewConstMetric(float64(timex.Tai))

	return nil
}
// Part 2 commit for node_service_exporter/internal/metrics/timex.go

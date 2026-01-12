package metrics

import (
	"log/slog"

	"node_process_exporter/internal/exporter"
	"github.com/prometheus-community/go-runit/runit"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	exporter.Register(NewRunitCollector())
}

type typedDesc struct {
	desc      *prometheus.Desc
	valueType prometheus.ValueType
}

func (d *typedDesc) mustNewConstMetric(value float64, labels ...string) prometheus.Metric {
	return prometheus.MustNewConstMetric(d.desc, d.valueType, value, labels...)
}

type runitCollector struct {
	*baseMetrics
	state          typedDesc
	stateDesired   typedDesc
	stateNormal    typedDesc
	stateTimestamp typedDesc
	runitServiceDir string
	logger         *slog.Logger
}

func NewRunitCollector() *runitCollector {
	var (
		constLabels = prometheus.Labels{"supervisor": "runit"}
		labelNames  = []string{"service"}
	)

	logger := slog.Default()
	logger.Warn("This collector is deprecated and will be removed in the next major version release.")

	// 默认runit服务目录
	runitServiceDir := "/etc/service"

	return &runitCollector{
		baseMetrics: NewMetrics("node_runit_collect_errors_total", "Number of errors that occurred during runit collection", []string{}),
		state: typedDesc{prometheus.NewDesc(
			"node_service_state",
			"State of runit service.",
			labelNames, constLabels,
		), prometheus.GaugeValue},
		stateDesired: typedDesc{prometheus.NewDesc(
			"node_service_desired_state",
			"Desired state of runit service.",
			labelNames, constLabels,
		), prometheus.GaugeValue},
		stateNormal: typedDesc{prometheus.NewDesc(
			"node_service_normal_state",
			"Normal state of runit service.",
			labelNames, constLabels,
		), prometheus.GaugeValue},
		stateTimestamp: typedDesc{prometheus.NewDesc(
			"node_service_state_last_change_timestamp_seconds",
			"Unix timestamp of the last runit service state change.",
			labelNames, constLabels,
		), prometheus.GaugeValue},
		runitServiceDir: runitServiceDir,
		logger:         logger,
	}
}

func (c *runitCollector) Collect(ch chan<- prometheus.Metric) {
	if err := c.Update(ch); err != nil {
		c.logger.Error("Error updating runit metrics", "error", err)
		ch <- prometheus.MustNewConstMetric(c.baseMetrics.desc, prometheus.CounterValue, 1)
	}
}

func (c *runitCollector) Update(ch chan<- prometheus.Metric) error {
	services, err := runit.GetServices(c.runitServiceDir)
	if err != nil {
		return err
	}

	for _, service := range services {
		status, err := service.Status()
		if err != nil {
			c.logger.Debug("Couldn't get status", "service", service.Name, "err", err)
			continue
		}

		c.logger.Debug("duration", "service", service.Name, "status", status.State, "pid", status.Pid, "duration_seconds", status.Duration)
		ch <- c.state.mustNewConstMetric(float64(status.State), service.Name)
		ch <- c.stateDesired.mustNewConstMetric(float64(status.Want), service.Name)
		ch <- c.stateTimestamp.mustNewConstMetric(float64(status.Timestamp.Unix()), service.Name)
		if status.NormallyUp {
			ch <- c.stateNormal.mustNewConstMetric(1, service.Name)
		} else {
			ch <- c.stateNormal.mustNewConstMetric(0, service.Name)
		}
	}
	return nil
} 
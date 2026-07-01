package metrics

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"node_process_exporter/internal/exporter"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// minSystemdVersionSystemState is the minimum SystemD version for availability of
	// the 'SystemState' manager property and the timer property 'LastTriggerUSec'
	// https://github.com/prometheus/node_exporter/issues/291
	minSystemdVersionSystemState = 212
)

var (
	systemdVersionRE = regexp.MustCompile(`[0-9]{3,}(\.[0-9]+)?`)
)

func init() {
	exporter.Register(NewSystemdCollector())
}

type systemdCollector struct {
	*baseMetrics
	unitDesc                      *prometheus.Desc
	unitStartTimeDesc             *prometheus.Desc
	unitTasksCurrentDesc          *prometheus.Desc
	unitTasksMaxDesc              *prometheus.Desc
	systemRunningDesc             *prometheus.Desc
	summaryDesc                   *prometheus.Desc
	nRestartsDesc                 *prometheus.Desc
	timerLastTriggerDesc          *prometheus.Desc
	socketAcceptedConnectionsDesc *prometheus.Desc
	socketCurrentConnectionsDesc  *prometheus.Desc
	socketRefusedConnectionsDesc  *prometheus.Desc
	systemdVersionDesc            *prometheus.Desc
	virtualizationDesc            *prometheus.Desc
	// Use regexps for more flexibility than device_filter.go allows
	systemdUnitIncludePattern *regexp.Regexp
	systemdUnitExcludePattern *regexp.Regexp
	logger                    *slog.Logger
}

var unitStatesName = []string{"active", "activating", "deactivating", "inactive", "failed"}

func NewSystemdCollector() *systemdCollector {
	const subsystem = "systemd"

	logger := slog.Default()

	unitDesc := prometheus.NewDesc(
		"node_systemd_unit_state",
		"Systemd unit", []string{"name", "state", "type"}, nil,
	)
	unitStartTimeDesc := prometheus.NewDesc(
		"node_systemd_unit_start_time_seconds",
		"Start time of the unit since unix epoch in seconds.", []string{"name"}, nil,
	)
	unitTasksCurrentDesc := prometheus.NewDesc(
		"node_systemd_unit_tasks_current",
		"Current number of tasks per Systemd unit", []string{"name"}, nil,
	)
	unitTasksMaxDesc := prometheus.NewDesc(
		"node_systemd_unit_tasks_max",
		"Maximum number of tasks per Systemd unit", []string{"name"}, nil,
	)
	systemRunningDesc := prometheus.NewDesc(
		"node_systemd_system_running",
		"Whether the system is operational (see 'systemctl is-system-running')",
		nil, nil,
	)
	summaryDesc := prometheus.NewDesc(
		"node_systemd_units",
		"Summary of systemd unit states", []string{"state"}, nil)
	nRestartsDesc := prometheus.NewDesc(
		"node_systemd_service_restart_total",
		"Service unit count of Restart triggers", []string{"name"}, nil)
	timerLastTriggerDesc := prometheus.NewDesc(
		"node_systemd_timer_last_trigger_seconds",
		"Seconds since epoch of last trigger.", []string{"name"}, nil)
	socketAcceptedConnectionsDesc := prometheus.NewDesc(
		"node_systemd_socket_accepted_connections_total",
		"Total number of accepted socket connections", []string{"name"}, nil)
	socketCurrentConnectionsDesc := prometheus.NewDesc(
		"node_systemd_socket_current_connections",
		"Current number of socket connections", []string{"name"}, nil)
	socketRefusedConnectionsDesc := prometheus.NewDesc(
		"node_systemd_socket_refused_connections_total",
		"Total number of refused socket connections", []string{"name"}, nil)
	systemdVersionDesc := prometheus.NewDesc(
		"node_systemd_version",
		"Detected systemd version", []string{"version"}, nil)
	virtualizationDesc := prometheus.NewDesc(
		"node_systemd_virtualization_info",
		"Detected virtualization technology", []string{"virtualization_type"}, nil)

	// 默认配置：包含所有单元，排除某些类型
	systemdUnitIncludePattern := regexp.MustCompile("^(?:.+)$")
	systemdUnitExcludePattern := regexp.MustCompile("^(?:.+\\.(automount|device|mount|scope|slice))$")

	return &systemdCollector{
		baseMetrics:                   NewMetrics("node_systemd_collect_errors_total", "Number of errors that occurred during systemd collection", []string{}),
		unitDesc:                      unitDesc,
		unitStartTimeDesc:             unitStartTimeDesc,
		unitTasksCurrentDesc:          unitTasksCurrentDesc,
		unitTasksMaxDesc:              unitTasksMaxDesc,
		systemRunningDesc:             systemRunningDesc,
		summaryDesc:                   summaryDesc,
		nRestartsDesc:                 nRestartsDesc,
		timerLastTriggerDesc:          timerLastTriggerDesc,
		socketAcceptedConnectionsDesc: socketAcceptedConnectionsDesc,
		socketCurrentConnectionsDesc:  socketCurrentConnectionsDesc,
		socketRefusedConnectionsDesc:  socketRefusedConnectionsDesc,
		systemdVersionDesc:            systemdVersionDesc,
		virtualizationDesc:            virtualizationDesc,
		systemdUnitIncludePattern:     systemdUnitIncludePattern,
		systemdUnitExcludePattern:     systemdUnitExcludePattern,
		logger:                        logger,
	}
}

func (c *systemdCollector) Collect(ch chan<- prometheus.Metric) {
	if err := c.Update(ch); err != nil {
		c.logger.Error("Error updating systemd metrics", "error", err)
		ch <- prometheus.MustNewConstMetric(c.baseMetrics.desc, prometheus.CounterValue, 1)
	}
}

// Update gathers metrics from systemd.  Dbus collection is done in parallel
// to reduce wait time for responses.
func (c *systemdCollector) Update(ch chan<- prometheus.Metric) error {
	begin := time.Now()
	conn, err := c.newSystemdDbusConn()
	if err != nil {
		return fmt.Errorf("couldn't get dbus connection: %w", err)
	}
	defer conn.Close()

	systemdVersion, systemdVersionFull := c.getSystemdVersion(conn)
	if systemdVersion < minSystemdVersionSystemState {
		c.logger.Debug("Detected systemd version is lower than minimum, some systemd state and timer metrics will not be available", "current", systemdVersion, "minimum", minSystemdVersionSystemState)
	}
	ch <- prometheus.MustNewConstMetric(
		c.systemdVersionDesc,
		prometheus.GaugeValue,
		systemdVersion,
		systemdVersionFull,
	)

	virt := c.getSystemdVirtualization(conn)
	if virt != "" {
		ch <- prometheus.MustNewConstMetric(
			c.virtualizationDesc,
			prometheus.GaugeValue,
			1,
			virt,
		)
	}

	allUnits, err := c.getAllUnits(conn)
	if err != nil {
		return fmt.Errorf("couldn't get units: %w", err)
	}
	c.logger.Debug("systemd getAllUnits took", "duration_seconds", time.Since(begin).Seconds())

	begin = time.Now()
	summary := c.summarizeUnits(allUnits)
	c.collectSummaryMetrics(ch, summary)
	c.logger.Debug("systemd summarizeUnits took", "duration_seconds", time.Since(begin).Seconds())

	begin = time.Now()
	units := c.filterUnits(allUnits, c.systemdUnitIncludePattern, c.systemdUnitExcludePattern, c.logger)
	c.logger.Debug("systemd filterUnits took", "duration_seconds", time.Since(begin).Seconds())

	var wg sync.WaitGroup
	defer wg.Wait()

	wg.Add(1)
	go func() {
		defer wg.Done()
		begin = time.Now()
		c.collectUnitStatusMetrics(conn, ch, units)
		c.logger.Debug("systemd collectUnitStatusMetrics took", "duration_seconds", time.Since(begin).Seconds())
	}()

	if systemdVersion >= minSystemdVersionSystemState {
		wg.Add(1)
		go func() {
			defer wg.Done()
			begin = time.Now()
			c.collectSockets(conn, ch, units)
			c.logger.Debug("systemd collectSockets took", "duration_seconds", time.Since(begin).Seconds())
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			begin = time.Now()
			_ = c.collectSystemState(conn, ch)
			c.logger.Debug("systemd collectSystemState took", "duration_seconds", time.Since(begin).Seconds())
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			begin = time.Now()
			c.collectUnitStartTimeMetrics(conn, ch, units)
			c.logger.Debug("systemd collectUnitStartTimeMetrics took", "duration_seconds", time.Since(begin).Seconds())
		}()

		// 启用任务指标
		wg.Add(1)
		go func() {
			defer wg.Done()
			begin = time.Now()
			c.collectUnitTasksMetrics(conn, ch, units)
			c.logger.Debug("systemd collectUnitTasksMetrics took", "duration_seconds", time.Since(begin).Seconds())
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			begin = time.Now()
			c.collectTimers(conn, ch, units)
			c.logger.Debug("systemd collectTimers took", "duration_seconds", time.Since(begin).Seconds())
		}()
	}

	return nil
}

func (c *systemdCollector) collectUnitStatusMetrics(conn *dbus.Conn, ch chan<- prometheus.Metric, units []unit) {
	for _, unit := range units {
		for _, stateName := range unitStatesName {
			isActive := 0.0
			if stateName == unit.ActiveState {
				isActive = 1.0
			}
			ch <- prometheus.MustNewConstMetric(
				c.unitDesc, prometheus.GaugeValue, isActive,
				unit.Name, stateName, unit.unitType(),
			)
		}
		if strings.HasSuffix(unit.Name, ".service") {
			// 启用重启指标
			if prop, err := conn.GetUnitPropertyContext(context.TODO(), unit.Name, "NRestarts"); err != nil {
				c.logger.Debug("couldn't get unit NRestarts", "unit", unit.Name, "err", err)
			} else {
				ch <- prometheus.MustNewConstMetric(
					c.nRestartsDesc, prometheus.CounterValue,
					float64(prop.Value.Value().(uint32)), unit.Name)
			}
		}
	}
}

func (c *systemdCollector) collectSockets(conn *dbus.Conn, ch chan<- prometheus.Metric, units []unit) {
	for _, unit := range units {
		if !strings.HasSuffix(unit.Name, ".socket") {
			continue
		}

		acceptedConnectionCount, err := conn.GetUnitPropertyContext(context.TODO(), unit.Name, "NAccepted")
		if err != nil {
			c.logger.Debug("couldn't get unit NAccepted", "unit", unit.Name, "err", err)
			continue
		}
		ch <- prometheus.MustNewConstMetric(
			c.socketAcceptedConnectionsDesc, prometheus.CounterValue,
			float64(acceptedConnectionCount.Value.Value().(uint32)), unit.Name)

		currentConnectionCount, err := conn.GetUnitPropertyContext(context.TODO(), unit.Name, "NConnections")
		if err != nil {
			c.logger.Debug("couldn't get unit NConnections", "unit", unit.Name, "err", err)
			continue
		}
		ch <- prometheus.MustNewConstMetric(
			c.socketCurrentConnectionsDesc, prometheus.GaugeValue,
			float64(currentConnectionCount.Value.Value().(uint32)), unit.Name)

		// NRefused wasn't added until systemd 239.
		refusedConnectionCount, err := conn.GetUnitPropertyContext(context.TODO(), unit.Name, "NRefused")
		if err != nil {
			c.logger.Debug("couldn't get unit NRefused", "unit", unit.Name, "err", err)
		} else {
			ch <- prometheus.MustNewConstMetric(
				c.socketRefusedConnectionsDesc, prometheus.CounterValue,
				float64(refusedConnectionCount.Value.Value().(uint32)), unit.Name)
		}
	}
}

func (c *systemdCollector) collectUnitStartTimeMetrics(conn *dbus.Conn, ch chan<- prometheus.Metric, units []unit) {
	for _, unit := range units {
		var startTimeUsec uint64

		if unit.ActiveState != "active" {
			startTimeUsec = 0
		} else {
			timestampValue, err := conn.GetUnitPropertyContext(context.TODO(), unit.Name, "ActiveEnterTimestamp")
			if err != nil {
				c.logger.Debug("couldn't get unit ActiveEnterTimestamp", "unit", unit.Name, "err", err)
				continue
			}
			startTimeUsec = timestampValue.Value.Value().(uint64)
		}

		ch <- prometheus.MustNewConstMetric(
			c.unitStartTimeDesc, prometheus.GaugeValue,
			float64(startTimeUsec)/1e6, unit.Name)
	}
}

func (c *systemdCollector) collectUnitTasksMetrics(conn *dbus.Conn, ch chan<- prometheus.Metric, units []unit) {
	for _, unit := range units {
		if !strings.HasSuffix(unit.Name, ".service") {
			continue
		}

		tasksCurrentCount, err := conn.GetUnitPropertyContext(context.TODO(), unit.Name, "TasksCurrent")
		if err != nil {
			c.logger.Debug("couldn't get unit TasksCurrent", "unit", unit.Name, "err", err)
		} else {
			ch <- prometheus.MustNewConstMetric(
				c.unitTasksCurrentDesc, prometheus.GaugeValue,
				float64(tasksCurrentCount.Value.Value().(uint64)), unit.Name)
		}

		tasksMaxCount, err := conn.GetUnitPropertyContext(context.TODO(), unit.Name, "TasksMax")
		if err != nil {
			c.logger.Debug("couldn't get unit TasksMax", "unit", unit.Name, "err", err)
		} else {
			ch <- prometheus.MustNewConstMetric(
				c.unitTasksMaxDesc, prometheus.GaugeValue,
				float64(tasksMaxCount.Value.Value().(uint64)), unit.Name)
		}
	}
}

func (c *systemdCollector) collectTimers(conn *dbus.Conn, ch chan<- prometheus.Metric, units []unit) {
	for _, unit := range units {
		if !strings.HasSuffix(unit.Name, ".timer") {
			continue
		}

		lastTriggerValue, err := conn.GetUnitPropertyContext(context.TODO(), unit.Name, "LastTriggerUSec")
		if err != nil {
			c.logger.Debug("couldn't get unit LastTriggerUSec", "unit", unit.Name, "err", err)
			continue
		}

		ch <- prometheus.MustNewConstMetric(
			c.timerLastTriggerDesc, prometheus.GaugeValue,
			float64(lastTriggerValue.Value.Value().(uint64))/1e6, unit.Name)
	}
}

func (c *systemdCollector) collectSummaryMetrics(ch chan<- prometheus.Metric, summary map[string]float64) {
	for stateName, count := range summary {
		ch <- prometheus.MustNewConstMetric(
			c.summaryDesc, prometheus.GaugeValue, count, stateName)
	}
}

func (c *systemdCollector) collectSystemState(conn *dbus.Conn, ch chan<- prometheus.Metric) error {
	systemState, err := conn.GetManagerProperty("SystemState")
	if err != nil {
		return fmt.Errorf("couldn't get system state: %w", err)
	}
	isSystemRunning := 0.0
	if systemState == "running" {
		isSystemRunning = 1.0
	}
	ch <- prometheus.MustNewConstMetric(c.systemRunningDesc, prometheus.GaugeValue, isSystemRunning)
	return nil
}

func (c *systemdCollector) newSystemdDbusConn() (*dbus.Conn, error) {
	return dbus.NewWithContext(context.TODO())
}

type unit struct {
	dbus.UnitStatus
}

func (u unit) unitType() string {
	return strings.SplitN(u.Name, ".", 2)[1]
}

func (c *systemdCollector) getAllUnits(conn *dbus.Conn) ([]unit, error) {
	allUnits, err := conn.ListUnitsContext(context.TODO())
	if err != nil {
		return nil, err
	}

	result := make([]unit, 0, len(allUnits))
	for _, status := range allUnits {
		result = append(result, unit{status})
	}

	return result, nil
}

func (c *systemdCollector) summarizeUnits(units []unit) map[string]float64 {
	// 总结单元状态
	summary := make(map[string]float64)

	for _, unit := range units {
		summary[unit.ActiveState]++
	}

	return summary
}

func (c *systemdCollector) filterUnits(units []unit, includePattern, excludePattern *regexp.Regexp, logger *slog.Logger) []unit {
	filtered := make([]unit, 0, len(units))
	for _, unit := range units {
		if includePattern.MatchString(unit.Name) && !excludePattern.MatchString(unit.Name) {
			logger.Debug("Adding unit", "unit", unit.Name)
			filtered = append(filtered, unit)
		} else {
			logger.Debug("Ignoring unit", "unit", unit.Name)
		}
	}

	return filtered
}

func (c *systemdCollector) getSystemdVersion(conn *dbus.Conn) (float64, string) {
	version, err := conn.GetManagerProperty("Version")
	if err != nil {
		c.logger.Debug("unable to get systemd version property", "err", err)
		return math.NaN(), ""
	}

	version = strings.TrimPrefix(version, "systemd ")
	version = strings.Split(version, " ")[0]

	parsed := systemdVersionRE.FindString(version)
	if parsed == "" {
		c.logger.Debug("unable to parse systemd version", "version", version)
		return math.NaN(), version
	}

	v, err := strconv.ParseFloat(parsed, 64)
	if err != nil {
		c.logger.Debug("unable to parse systemd version", "version", parsed, "err", err)
		return math.NaN(), version
	}

	return v, version
}

func (c *systemdCollector) getSystemdVirtualization(conn *dbus.Conn) string {
	virtualization, err := conn.GetManagerProperty("Virtualization")
	if err != nil {
		c.logger.Debug("unable to get systemd virtualization property", "err", err)
		return ""
	}
	return virtualization
}

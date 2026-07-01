package metrics

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/prometheus/client_golang/prometheus"
	"systemd_exporter/internal/exporter"
)

const namespace = "systemd"
const watchdogSubsystem = "watchdog"

var (
	unitInclude               = ".*"
	unitExclude               = ".*\\.(device)"
	systemdPrivate            = false
	systemdUser               = false
	enableRestartsMetrics     = true
	enableIPAccountingMetrics = true
)

var unitStatesName = []string{"active", "activating", "deactivating", "inactive", "failed"}

var (
	errGetPropertyMsg           = "couldn't get unit's %s property: %w"
	errConvertUint64PropertyMsg = "couldn't convert unit's %s property %v to uint64"
	errConvertUint32PropertyMsg = "couldn't convert unit's %s property %v to uint32"
	errConvertStringPropertyMsg = "couldn't convert unit's %s property %v to string"
	errUnitMetricsMsg           = "couldn't get unit's metrics: %s"
	infoUnitNoHandler           = "no unit type handler for %s"
)

// 注册所有的指标收集器
func init() {
	exporter.Register(NewSystemdMetric("systemd_boot_time_seconds", "Systemd boot stage timestamps", []string{"stage"}))
	exporter.Register(NewSystemdMetric("systemd_boot_monotonic_seconds", "Systemd boot stage monotonic timestamps", []string{"stage"}))
	exporter.Register(NewSystemdMetric("systemd_unit_state", "Systemd unit", []string{"name", "type", "state"}))
	exporter.Register(NewSystemdMetric("systemd_unit_info", "Mostly-static metadata for all unit types", []string{"name", "type", "mount_type", "service_type"}))
	exporter.Register(NewSystemdMetric("systemd_unit_start_time_seconds", "Start time of the unit since unix epoch in seconds.", []string{"name", "type"}))
	exporter.Register(NewSystemdMetric("systemd_unit_tasks_current", "Current number of tasks per Systemd unit", []string{"name"}))
	exporter.Register(NewSystemdMetric("systemd_unit_tasks_max", "Maximum number of tasks per Systemd unit", []string{"name", "type"}))
	exporter.Register(NewSystemdMetric("systemd_unit_active_enter_time_seconds", "Last time the unit transitioned into the active state", []string{"name", "type"}))
	exporter.Register(NewSystemdMetric("systemd_unit_active_exit_time_seconds", "Last time the unit transitioned out of the active state", []string{"name", "type"}))
	exporter.Register(NewSystemdMetric("systemd_unit_inactive_enter_time_seconds", "Last time the unit transitioned into the inactive state", []string{"name", "type"}))
	exporter.Register(NewSystemdMetric("systemd_unit_inactive_exit_time_seconds", "Last time the unit transitioned out of the inactive state", []string{"name", "type"}))
	exporter.Register(NewSystemdMetric("systemd_service_restart_total", "Service unit count of Restart triggers", []string{"name"}))
	exporter.Register(NewSystemdMetric("systemd_timer_last_trigger_seconds", "Seconds since epoch of last trigger.", []string{"name"}))
	exporter.Register(NewSystemdMetric("systemd_socket_accepted_connections_total", "Total number of accepted socket connections", []string{"name"}))
	exporter.Register(NewSystemdMetric("systemd_socket_current_connections", "Current number of socket connections", []string{"name"}))
	exporter.Register(NewSystemdMetric("systemd_socket_refused_connections_total", "Total number of refused socket connections", []string{"name"}))
	exporter.Register(NewSystemdMetric("systemd_unit_cpu_seconds_total", "Unit CPU time in seconds", []string{"name", "type", "mode"}))
	exporter.Register(NewSystemdMetric("systemd_service_ip_ingress_bytes", "Service unit ingress IP accounting in bytes.", []string{"name"}))
	exporter.Register(NewSystemdMetric("systemd_service_ip_egress_bytes", "Service unit egress IP accounting in bytes.", []string{"name"}))
	exporter.Register(NewSystemdMetric("systemd_service_ip_ingress_packets_total", "Service unit ingress IP accounting in packets.", []string{"name"}))
	exporter.Register(NewSystemdMetric("systemd_service_ip_egress_packets_total", "Service unit egress IP accounting in packets.", []string{"name"}))
	exporter.Register(NewSystemdMetric("systemd_watchdog_enabled", "systemd watchdog enabled", []string{}))
	exporter.Register(NewSystemdMetric("systemd_watchdog_last_ping_monotonic_seconds", "systemd watchdog last ping monotonic seconds", []string{"device"}))
	exporter.Register(NewSystemdMetric("systemd_watchdog_last_ping_time_seconds", "systemd watchdog last ping time seconds", []string{"device"}))
	exporter.Register(NewSystemdMetric("systemd_watchdog_runtime_seconds", "systemd watchdog runtime seconds", []string{"device"}))
}

// SystemdMetric 是一个单独的 systemd 指标收集器
type SystemdMetric struct {
	*baseMetrics
	name string
}

// NewSystemdMetric 创建一个新的 systemd 指标收集器
func NewSystemdMetric(name, help string, labels []string) *SystemdMetric {
	return &SystemdMetric{
		baseMetrics: NewMetrics(name, help, labels),
		name:        name,
	}
}

// Collect 实现 prometheus.Collector 接口
func (sm *SystemdMetric) Collect(ch chan<- prometheus.Metric) {
	ctx := context.TODO()
	
	conn, err := newDbus(ctx)
	if err != nil {
		fmt.Printf("couldn't get dbus connection: %v\n", err)
		return
	}
	defer conn.Close()

	// 根据指标名称选择要收集的数据
	switch sm.name {
	case "systemd_boot_time_seconds", "systemd_boot_monotonic_seconds":
		collectBootStageTimestamps(ctx, conn, ch, sm.baseMetrics, sm.name)
	case "systemd_watchdog_enabled", "systemd_watchdog_last_ping_monotonic_seconds", 
		 "systemd_watchdog_last_ping_time_seconds", "systemd_watchdog_runtime_seconds":
		collectWatchdogMetrics(ctx, conn, ch, sm.baseMetrics, sm.name)
	default:
		// 其他指标需要遍历所有单元
		allUnits, err := conn.ListUnitsContext(ctx)
		if err != nil {
			fmt.Printf("could not get list of systemd units from dbus: %v\n", err)
			return
		}

		unitIncludePattern := regexp.MustCompile(fmt.Sprintf("^(?:%s)$", unitInclude))
		unitExcludePattern := regexp.MustCompile(fmt.Sprintf("^(?:%s)$", unitExclude))
		
		units := filterUnits(allUnits, unitIncludePattern, unitExcludePattern)

		for _, unit := range units {
			collectUnitMetric(ctx, conn, ch, unit, sm.baseMetrics, sm.name)
		}
	}
}

func newDbus(ctx context.Context) (*dbus.Conn, error) {
	if systemdPrivate {
		return dbus.NewSystemdConnectionContext(ctx)
	}
	if systemdUser {
		return dbus.NewUserConnectionContext(ctx)
	}
	return dbus.NewWithContext(ctx)
}

func filterUnits(units []dbus.UnitStatus, includePattern, excludePattern *regexp.Regexp) []dbus.UnitStatus {
	filtered := make([]dbus.UnitStatus, 0, len(units))
	for _, unit := range units {
		if includePattern.MatchString(unit.Name) &&
			!excludePattern.MatchString(unit.Name) &&
			unit.LoadState == "loaded" {
			filtered = append(filtered, unit)
		}
	}

	return filtered
}

func collectBootStageTimestamps(ctx context.Context, conn *dbus.Conn, ch chan<- prometheus.Metric, bm *baseMetrics, metricName string) {
	stages := []string{"Finish", "Firmware", "Loader", "Kernel", "InitRD",
		"InitRDGeneratorsStart", "InitRDGeneratorsFinish",
		"InitRDSecurityStart", "InitRDSecurityFinish",
		"InitRDUnitsLoadStart", "InitRDUnitsLoadFinish",
		"GeneratorsStart", "GeneratorsFinish",
		"SecurityStart", "SecurityFinish", "Userspace",
		"UnitsLoadStart", "UnitsLoadFinish"}

	for _, stage := range stages {
		stageMonotonicValue, err := conn.GetManagerProperty(fmt.Sprintf("%sTimestampMonotonic", stage))
		if err != nil {
			continue
		}

		stageTimestampValue, err := conn.GetManagerProperty(fmt.Sprintf("%sTimestamp", stage))
		if err != nil {
			continue
		}

		stageMonotonic := strings.TrimPrefix(strings.TrimSuffix(stageMonotonicValue, `"`), `"`)
		stageTimestamp := strings.TrimPrefix(strings.TrimSuffix(stageTimestampValue, `"`), `"`)

		vMonotonic, err := strconv.ParseFloat(strings.TrimLeft(stageMonotonic, "@t "), 64)
		if err != nil {
			continue
		}

		vTimestamp, err := strconv.ParseFloat(strings.TrimLeft(stageTimestamp, "@t "), 64)
		if err != nil {
			continue
		}

		// 只收集对应指标的数据
		if metricName == "systemd_boot_monotonic_seconds" {
			bm.collect(ch, float64(vMonotonic)/1e6, []string{stage})
		} else if metricName == "systemd_boot_time_seconds" {
			bm.collect(ch, float64(vTimestamp)/1e6, []string{stage})
		}
	}
}

func parseUnitType(unit dbus.UnitStatus) string {
	t := strings.Split(unit.Name, ".")
	return t[len(t)-1]
}

func collectUnitMetric(ctx context.Context, conn *dbus.Conn, ch chan<- prometheus.Metric, unit dbus.UnitStatus, bm *baseMetrics, metricName string) {
	// 根据指标名称选择合适的收集函数
	switch metricName {
	case "systemd_unit_state":
		collectUnitState(ctx, conn, ch, unit, bm)
	case "systemd_unit_info":
		// 只对特定类型的单元收集信息
		switch {
		case strings.HasSuffix(unit.Name, ".service"):
			collectServiceMetainfo(ctx, conn, ch, unit, bm)
		case strings.HasSuffix(unit.Name, ".mount"):
			collectMountMetainfo(ctx, conn, ch, unit, bm)
		}
	case "systemd_unit_start_time_seconds":
		collectServiceStartTimeMetrics(ctx, conn, ch, unit, bm)
	case "systemd_unit_tasks_current":
		if strings.HasSuffix(unit.Name, ".service") {
			collectServiceTasksCurrentMetrics(ctx, conn, ch, unit, bm)
		}
	case "systemd_unit_tasks_max":
		if strings.HasSuffix(unit.Name, ".service") {
			collectServiceTasksMaxMetrics(ctx, conn, ch, unit, bm)
		}
	case "systemd_unit_active_enter_time_seconds":
		collectUnitTimeMetric(ctx, conn, ch, unit, "ActiveEnterTimestamp", bm)
	case "systemd_unit_active_exit_time_seconds":
		collectUnitTimeMetric(ctx, conn, ch, unit, "ActiveExitTimestamp", bm)
	case "systemd_unit_inactive_enter_time_seconds":
		collectUnitTimeMetric(ctx, conn, ch, unit, "InactiveEnterTimestamp", bm)
	case "systemd_unit_inactive_exit_time_seconds":
		collectUnitTimeMetric(ctx, conn, ch, unit, "InactiveExitTimestamp", bm)
	case "systemd_service_restart_total":
		if strings.HasSuffix(unit.Name, ".service") && enableRestartsMetrics {
			collectServiceRestartCount(ctx, conn, ch, unit, bm)
		}
	case "systemd_timer_last_trigger_seconds":
		if strings.HasSuffix(unit.Name, ".timer") {
			collectTimerTriggerTime(ctx, conn, ch, unit, bm)
		}
	case "systemd_socket_accepted_connections_total", "systemd_socket_current_connections", "systemd_socket_refused_connections_total":
		if strings.HasSuffix(unit.Name, ".socket") {
			collectSocketConnMetricsForMetric(ctx, conn, ch, unit, bm, metricName)
		}
	case "systemd_service_ip_ingress_bytes", "systemd_service_ip_egress_bytes", 
		 "systemd_service_ip_ingress_packets_total", "systemd_service_ip_egress_packets_total":
		if strings.HasSuffix(unit.Name, ".service") && enableIPAccountingMetrics {
			collectIPAccountingMetricForMetric(ctx, conn, ch, unit, bm, metricName)
		}
	}
}

func collectUnitState(ctx context.Context, conn *dbus.Conn, ch chan<- prometheus.Metric, unit dbus.UnitStatus, bm *baseMetrics) {
	for _, stateName := range unitStatesName {
		isActive := 0.0
		if stateName == unit.ActiveState {
			isActive = 1.0
		}
		bm.collect(ch, isActive, []string{unit.Name, parseUnitType(unit), stateName})
	}
}

func collectUnitTimeMetric(ctx context.Context, conn *dbus.Conn, ch chan<- prometheus.Metric, unit dbus.UnitStatus, propertyName string, bm *baseMetrics) {
	timestampValue, err := conn.GetUnitPropertyContext(ctx, unit.Name, propertyName)
	if err != nil {
		return
	}
	startTimeUsec, ok := timestampValue.Value.Value().(uint64)
	if !ok {
		return
	}

	bm.collect(ch, float64(startTimeUsec)/1e6, []string{unit.Name, parseUnitType(unit)})
}

func collectMountMetainfo(ctx context.Context, conn *dbus.Conn, ch chan<- prometheus.Metric, unit dbus.UnitStatus, bm *baseMetrics) {
	serviceTypeProperty, err := conn.GetUnitTypePropertyContext(ctx, unit.Name, "Mount", "Type")
	if err != nil {
		return
	}

	serviceType, ok := serviceTypeProperty.Value.Value().(string)
	if !ok {
		return
	}

	bm.collect(ch, 1.0, []string{unit.Name, parseUnitType(unit), serviceType, ""})
}

func collectServiceMetainfo(ctx context.Context, conn *dbus.Conn, ch chan<- prometheus.Metric, unit dbus.UnitStatus, bm *baseMetrics) {
	serviceTypeProperty, err := conn.GetUnitTypePropertyContext(ctx, unit.Name, "Service", "Type")
	if err != nil {
		return
	}
	serviceType, ok := serviceTypeProperty.Value.Value().(string)
	if !ok {
		return
	}

	bm.collect(ch, 1.0, []string{unit.Name, parseUnitType(unit), "", serviceType})
}

func collectServiceRestartCount(ctx context.Context, conn *dbus.Conn, ch chan<- prometheus.Metric, unit dbus.UnitStatus, bm *baseMetrics) {
	restartsCount, err := conn.GetUnitTypePropertyContext(ctx, unit.Name, "Service", "NRestarts")
	if err != nil {
		return
	}
	val, ok := restartsCount.Value.Value().(uint32)
	if !ok {
		return
	}
	bm.collect(ch, float64(val), []string{unit.Name})
}

func collectServiceStartTimeMetrics(ctx context.Context, conn *dbus.Conn, ch chan<- prometheus.Metric, unit dbus.UnitStatus, bm *baseMetrics) {
	var startTimeUsec uint64

	switch unit.ActiveState {
	case "active":
		timestampValue, err := conn.GetUnitPropertyContext(ctx, unit.Name, "ActiveEnterTimestamp")
		if err != nil {
			return
		}
		startTime, ok := timestampValue.Value.Value().(uint64)
		if !ok {
			return
		}
		startTimeUsec = startTime
	default:
		startTimeUsec = 0
	}

	bm.collect(ch, float64(startTimeUsec)/1e6, []string{unit.Name, parseUnitType(unit)})
}

func collectSocketConnMetricsForMetric(ctx context.Context, conn *dbus.Conn, ch chan<- prometheus.Metric, unit dbus.UnitStatus, bm *baseMetrics, metricName string) {
	switch metricName {
	case "systemd_socket_accepted_connections_total":
		acceptedConnectionCount, err := conn.GetUnitTypePropertyContext(ctx, unit.Name, "Socket", "NAccepted")
		if err != nil {
			return
		}
		bm.collect(ch, float64(acceptedConnectionCount.Value.Value().(uint32)), []string{unit.Name})
	
	case "systemd_socket_current_connections":
		currentConnectionCount, err := conn.GetUnitTypePropertyContext(ctx, unit.Name, "Socket", "NConnections")
		if err != nil {
			return
		}
		bm.collect(ch, float64(currentConnectionCount.Value.Value().(uint32)), []string{unit.Name})
	
	case "systemd_socket_refused_connections_total":
		refusedConnectionCount, err := conn.GetUnitTypePropertyContext(ctx, unit.Name, "Socket", "NRefused")
		if err != nil {
			return
		}
		bm.collect(ch, float64(refusedConnectionCount.Value.Value().(uint32)), []string{unit.Name})
	}
}

func collectIPAccountingMetricForMetric(ctx context.Context, conn *dbus.Conn, ch chan<- prometheus.Metric, unit dbus.UnitStatus, bm *baseMetrics, metricName string) {
	var propertyName string
	switch metricName {
	case "systemd_service_ip_ingress_bytes":
		propertyName = "IPIngressBytes"
	case "systemd_service_ip_egress_bytes":
		propertyName = "IPEgressBytes"
	case "systemd_service_ip_ingress_packets_total":
		propertyName = "IPIngressPackets"
	case "systemd_service_ip_egress_packets_total":
		propertyName = "IPEgressPackets"
	default:
		return
	}

	property, err := conn.GetUnitTypePropertyContext(ctx, unit.Name, "Service", propertyName)
	if err != nil {
		return
	}

	counter, ok := property.Value.Value().(uint64)
	if !ok {
		return
	}

	bm.collect(ch, float64(counter), []string{unit.Name})
}

func collectServiceTasksCurrentMetrics(ctx context.Context, conn *dbus.Conn, ch chan<- prometheus.Metric, unit dbus.UnitStatus, bm *baseMetrics) {
	tasksCurrentCount, err := conn.GetUnitTypePropertyContext(ctx, unit.Name, "Service", "TasksCurrent")
	if err != nil {
		return
	}

	currentCount, ok := tasksCurrentCount.Value.Value().(uint64)
	if !ok {
		return
	}

	if currentCount != math.MaxUint64 {
		bm.collect(ch, float64(currentCount), []string{unit.Name})
	}
}

func collectServiceTasksMaxMetrics(ctx context.Context, conn *dbus.Conn, ch chan<- prometheus.Metric, unit dbus.UnitStatus, bm *baseMetrics) {
	tasksMaxCount, err := conn.GetUnitTypePropertyContext(ctx, unit.Name, "Service", "TasksMax")
	if err != nil {
		return
	}

	maxCount, ok := tasksMaxCount.Value.Value().(uint64)
	if !ok {
		return
	}
	
	if maxCount != math.MaxUint64 {
		bm.collect(ch, float64(maxCount), []string{unit.Name, parseUnitType(unit)})
	}
}

func collectTimerTriggerTime(ctx context.Context, conn *dbus.Conn, ch chan<- prometheus.Metric, unit dbus.UnitStatus, bm *baseMetrics) {
	lastTriggerValue, err := conn.GetUnitTypePropertyContext(ctx, unit.Name, "Timer", "LastTriggerUSec")
	if err != nil {
		return
	}
	val, ok := lastTriggerValue.Value.Value().(uint64)
	if !ok {
		return
	}
	bm.collect(ch, float64(val)/1e6, []string{unit.Name})
}

func collectWatchdogMetrics(ctx context.Context, conn *dbus.Conn, ch chan<- prometheus.Metric, bm *baseMetrics, metricName string) {
	watchdogDevice, err := conn.GetManagerProperty("WatchdogDevice")
	if err != nil {
		return
	}

	watchdogDeviceString := strings.TrimPrefix(strings.TrimSuffix(watchdogDevice, `"`), `"`)
	
	// 无论是否有看门狗，都报告启用状态
	if metricName == "systemd_watchdog_enabled" {
		if len(watchdogDeviceString) == 0 {
			bm.collect(ch, 0, []string{})
		} else {
			bm.collect(ch, 1, []string{})
		}
		return
	}

	// 如果没有看门狗，不需要收集其他指标
	if len(watchdogDeviceString) == 0 {
		return
	}

	// 对于其他看门狗指标，只有当看门狗存在时才收集
	switch metricName {
	case "systemd_watchdog_last_ping_monotonic_seconds":
		watchdogLastPingMonotonicProperty, err := conn.GetManagerProperty("WatchdogLastPingTimestampMonotonic")
		if err != nil {
			return
		}
		watchdogLastPingMonotonic, err := strconv.ParseFloat(strings.TrimLeft(watchdogLastPingMonotonicProperty, "@t "), 64)
		if err != nil {
			return
		}
		bm.collect(ch, float64(watchdogLastPingMonotonic)/1e6, []string{watchdogDeviceString})
	
	case "systemd_watchdog_last_ping_time_seconds":
		watchdogLastPingTimeProperty, err := conn.GetManagerProperty("WatchdogLastPingTimestamp")
		if err != nil {
			return
		}
		watchdogLastPingTimestamp, err := strconv.ParseFloat(strings.TrimLeft(watchdogLastPingTimeProperty, "@t "), 64)
		if err != nil {
			return
		}
		bm.collect(ch, float64(watchdogLastPingTimestamp)/1e6, []string{watchdogDeviceString})
	
	case "systemd_watchdog_runtime_seconds":
		runtimeWatchdogUSecProperty, err := conn.GetManagerProperty("RuntimeWatchdogUSec")
		if err != nil {
			return
		}
		runtimeWatchdogUSec, err := strconv.ParseFloat(strings.TrimLeft(runtimeWatchdogUSecProperty, "@t "), 64)
		if err != nil {
			return
		}
		bm.collect(ch, float64(runtimeWatchdogUSec)/1e6, []string{watchdogDeviceString})
	}
} 


package metrics

import (
    "encoding/json"
    "log/slog"
    "time"

    "github.com/alecthomas/kingpin/v2"
    "github.com/prometheus/client_golang/prometheus"
)

const (
    defaultComdTimeout = 30 * time.Second
    ipv4RouteCommand      = "show ip route vrf all summary json"
    ipv6RouteCommand      = "show ipv6 route vrf all summary json"
)

type RouteCollectorConfig struct {
    CommandTimeout time.Duration
    Logger         *slog.Logger
    EnableDetailed bool
}

type RouteCommandExecutor interface {
    Execute(command string, timeout time.Duration) ([]byte, error)
}

type DefaultCommandExecutor struct{}

type RouteProcessor interface {
    Process(ch chan<- prometheus.Metric, data []byte, afi string) error
}

type routeCollector struct {
    logger       *slog.Logger
    descriptions map[string]*prometheus.Desc
    executor     RouteCommandExecutor
    processor    RouteProcessor
    config       *RouteCollectorConfig
}

func NewRouteCollectorConfig(logger *slog.Logger) *RouteCollectorConfig {
    return &RouteCollectorConfig{
        CommandTimeout: defaultComdTimeout,
        Logger:         logger,
        EnableDetailed: *detailedRoutes,
    }
}

func init() {
    registerCollector(routeSubsystem, enabledByDefault, NewRouteCollector)
}

func NewRouteCollector(logger *slog.Logger) (Collector, error) {
    config := NewRouteCollectorConfig(logger)
    return NewRouteCollectorWithConfig(config)
}

func NewRouteCollectorWithConfig(config *RouteCollectorConfig) (Collector, error) {
    return &routeCollector{
        logger:       config.Logger,
        descriptions: getRouteDesc(),
        executor:     &DefaultCommandExecutor{},
        processor:    NewRouteProcessor(config.Logger, getRouteDesc(), config.EnableDetailed),
        config:       config,
    }, nil
}

func (e *DefaultCommandExecutor) Execute(command string, timeout time.Duration) ([]byte, error) {
    return executeZebraCommand(command)
}

func (c *routeCollector) Update(ch chan<- prometheus.Metric) error {
    if err := c.processRouteFamily(ch, ipv4RouteCommand, "ipv4"); err != nil {
        return err
    }
    return c.processRouteFamily(ch, ipv6RouteCommand, "ipv6")
}

func (c *routeCollector) processRouteFamily(ch chan<- prometheus.Metric, cmd, afi string) error {
    data, err := c.executor.Execute(cmd, c.config.CommandTimeout)
    if err != nil {
        return err
    }
    if err := c.processor.Process(ch, data, afi); err != nil {
        return cmdOutputProcessError(cmd, string(data), err)
    }
    return nil
}

type RouteProcessorImpl struct {
    logger       *slog.Logger
    descriptions map[string]*prometheus.Desc
    enableDetail bool
}

func NewRouteProcessor(logger *slog.Logger, desc map[string]*prometheus.Desc, enableDetail bool) RouteProcessor {
    return &RouteProcessorImpl{
        logger:       logger,
        descriptions: desc,
        enableDetail: enableDetail,
    }
}

func (p *RouteProcessorImpl) Process(ch chan<- prometheus.Metric, data []byte, afi string) error {
    var summaries map[string]routeSummary
    if err := json.Unmarshal(data, &summaries); err != nil {
        return err
    }

    for vrf, summary := range summaries {
        processor := NewVRFProcessor(p.logger, p.descriptions, p.enableDetail)
        if err := processor.Process(ch, summary, afi, vrf); err != nil {
            return err
        }
    }
    return nil
}

type VRFProcessor struct {
    logger       *slog.Logger
    descriptions map[string]*prometheus.Desc
    enableDetail bool
}

func NewVRFProcessor(logger *slog.Logger, desc map[string]*prometheus.Desc, enableDetail bool) *VRFProcessor {
    return &VRFProcessor{
        logger:       logger,
        descriptions: desc,
        enableDetail: enableDetail,
    }
}

func (p *VRFProcessor) Process(ch chan<- prometheus.Metric, summary routeSummary, afi, vrf string) error {
    newGauge(ch, p.descriptions["total"], float64(summary.RoutesTotal), afi, vrf)
    newGauge(ch, p.descriptions["totalFib"], float64(summary.RoutesTotalFib), afi, vrf)

    if p.enableDetail {
        for _, route := range summary.Routes {
            routeProcessor := NewRouteTypeProcessor(p.descriptions)
            routeProcessor.Process(ch, route, afi, vrf)
        }
    }
    return nil
}

type RouteTypeProcessor struct {
    descriptions map[string]*prometheus.Desc
}

func NewRouteTypeProcessor(desc map[string]*prometheus.Desc) *RouteTypeProcessor {
    return &RouteTypeProcessor{
        descriptions: desc,
    }
}

func (p *RouteTypeProcessor) Process(ch chan<- prometheus.Metric, r route, afi, vrf string) {
    labels := []string{afi, r.Type, vrf}
    newGauge(ch, p.descriptions["fibCount"], float64(r.Fib), labels...)
    newGauge(ch, p.descriptions["fibOffloadedCount"], float64(r.FibOffLoaded), labels...)
    newGauge(ch, p.descriptions["fibTrappedCount"], float64(r.FibTrapped), labels...)
    newGauge(ch, p.descriptions["ribCount"], float64(r.Rib), labels...)
}

func getRouteDesc() map[string]*prometheus.Desc {
    labels := []string{"afi", "route_type", "vrf"}
    totalLabels := []string{"afi", "vrf"}

    return map[string]*prometheus.Desc{
        "total":             colPromDesc(routeSubsystem, "total", "Total number of routes", totalLabels),
        "totalFib":          colPromDesc(routeSubsystem, "total_fib", "Total number of routes in FIB", totalLabels),
        "fibCount":          colPromDesc(routeSubsystem, "fib_count", "Number of routes of route type in FIB", labels),
        "fibOffloadedCount": colPromDesc(routeSubsystem, "fib_offloaded_count", "Number of offloaded routes of route type in FIB", labels),
        "fibTrappedCount":   colPromDesc(routeSubsystem, "fib_trapped_count", "Number of trapped routes of route type in FIB", labels),
        "ribCount":          colPromDesc(routeSubsystem, "rib_count", "Number of routes of route type in RIB", labels),
    }
}

type routeSummary struct {
    Routes         []route `json:"routes"`
    RoutesTotal    uint32  `json:"routesTotal"`
    RoutesTotalFib uint32  `json:"routesTotalFib"`
}

type route struct {
    Fib          uint32 `json:"fib"`
    Rib          uint32 `json:"rib"`
    FibOffLoaded uint32 `json:"fibOffLoaded"`
    FibTrapped   uint32 `json:"fibTrapped"`
    Type         string `json:"type"`
}

var (
    routeSubsystem = "route"
    detailedRoutes = kingpin.Flag("collector.route.detailed-routes", "Enable detailed route count of each route type (default: disabled).").Default("False").Bool()
)

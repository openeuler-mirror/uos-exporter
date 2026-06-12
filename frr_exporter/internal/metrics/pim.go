
package metrics

import (
    "encoding/json"
    "fmt"
    "log/slog"
    "strings"
    "time"

    "github.com/prometheus/client_golang/prometheus"
)

const (
    pimSubsystem = "pim"
    defaultCommandTimeout = 30 * time.Second
)

type DefaultCommandRunner struct{}

type DefaultTimeParser struct{}

type CommandRunner interface {
    Execute(command string, timeout time.Duration) ([]byte, error)
}

type TimeParser interface {
    ParseHMS(timeStr string) (uint64, error)
}

type PIMConfig struct {
    CommandTimeout time.Duration
    Logger         *slog.Logger
}

type pimCollector struct {
    logger        *slog.Logger
    descriptions  map[string]*prometheus.Desc
    commandRunner CommandRunner
    timeParser    TimeParser
	config       *PIMConfig
}

type PIMNeighborProcessor struct {
    logger       *slog.Logger
    descriptions map[string]*prometheus.Desc
}

type NeighborCounter struct {
    count float64
}

type pimNeighbor struct {
    Interface string `json:"interface"`
    Neighbor  string `json:"neighbor"`
    UpTime    string `json:"uptime"`
}

func NewPIMConfig(logger *slog.Logger) *PIMConfig {
    return &PIMConfig{
        CommandTimeout: defaultCommandTimeout,
        Logger:         logger,
    }
}

func init() {
    registerCollector(pimSubsystem, disabledByDefault, NewPIMCollector)
}

func NewPIMCollector(logger *slog.Logger) (Collector, error) {
    config := NewPIMConfig(logger)
    return NewPIMCollectorWithConfig(config)
}

func NewPIMCollectorWithConfig(config *PIMConfig) (Collector, error) {
    return &pimCollector{
        logger:        config.Logger,
        descriptions:  getPIMDesc(),
        commandRunner: &DefaultCommandRunner{},
        timeParser:    &DefaultTimeParser{},
		config:       config,
    }, nil
}

func (r *DefaultCommandRunner) Execute(command string, timeout time.Duration) ([]byte, error) {
    return executePIMCommand(command)
}

func (p *DefaultTimeParser) ParseHMS(timeStr string) (uint64, error) {
    return parseHMS(timeStr)
}

func getPIMDesc() map[string]*prometheus.Desc {
    labels := []string{"vrf"}
    neighborLabels := append(labels, "iface", "neighbor")

    return map[string]*prometheus.Desc{
        "neighborCount": colPromDesc(pimSubsystem, 
            "neighbor_count_total", 
            "Number of neighbors detected", 
            labels),
        "upTime": colPromDesc(pimSubsystem, 
            "neighbor_uptime_seconds", 
            "How long has the peer been up", 
            neighborLabels),
    }
}

func (c *pimCollector) Update(ch chan<- prometheus.Metric) error {
    cmd := "show ip pim vrf all neighbor json"
    jsonPIMNeighbors, err := c.commandRunner.Execute(cmd, c.config.CommandTimeout)
    if err != nil {
        return err
    }

    processor := NewPIMNeighborProcessor(c.logger, c.descriptions)
    if err := processor.Process(ch, jsonPIMNeighbors); err != nil {
        return cmdOutputProcessError(cmd, string(jsonPIMNeighbors), err)
    }
    return nil
}

func NewPIMNeighborProcessor(logger *slog.Logger, desc map[string]*prometheus.Desc) *PIMNeighborProcessor {
    return &PIMNeighborProcessor{
        logger:       logger,
        descriptions: desc,
    }
}

func (p *PIMNeighborProcessor) Process(ch chan<- prometheus.Metric, data []byte) error {
    var jsonMap map[string]json.RawMessage
    if err := json.Unmarshal(data, &jsonMap); err != nil {
        return fmt.Errorf("failed to unmarshal PIM neighbors data: %w", err)
    }

    for vrfName, vrfData := range jsonMap {
        if err := p.processVRF(ch, vrfName, vrfData); err != nil {
            return err
        }
    }
    return nil
}

func (p *PIMNeighborProcessor) processVRF(ch chan<- prometheus.Metric, vrfName string, vrfData json.RawMessage) error {
    var vrfInstance map[string]json.RawMessage
    if err := json.Unmarshal(vrfData, &vrfInstance); err != nil {
        return fmt.Errorf("failed to unmarshal VRF instance: %w", err)
    }

    neighborCounter := NewNeighborCounter()
    for ifaceName, ifaceData := range vrfInstance {
        if err := p.processInterface(ch, vrfName, ifaceName, ifaceData, neighborCounter); err != nil {
            return err
        }
    }

    newGauge(ch, p.descriptions["neighborCount"], neighborCounter.Count(), vrfName)
    return nil
}

func (p *PIMNeighborProcessor) processInterface(
    ch chan<- prometheus.Metric,
    vrfName string,
    ifaceName string,
    ifaceData json.RawMessage,
    counter *NeighborCounter,
) error {
    var neighbors map[string]pimNeighbor
    if err := json.Unmarshal(ifaceData, &neighbors); err != nil {
        return fmt.Errorf("failed to unmarshal neighbor data: %w", err)
    }

    for neighborIP, neighborData := range neighbors {
        counter.Increment()
        if err := p.processNeighbor(ch, vrfName, ifaceName, neighborIP, neighborData); err != nil {
            return err
        }
    }
    return nil
}

func (p *PIMNeighborProcessor) processNeighbor(
    ch chan<- prometheus.Metric,
    vrfName string,
    ifaceName string,
    neighborIP string,
    neighborData pimNeighbor,
) error {
    uptimeSec, err := parseHMS(neighborData.UpTime)
    if err != nil {
        // p.logger.Error("failed to parse neighbor uptime", 
        //     "uptime", neighborData.UpTime, 
        //     "error", err)
        return nil
    }

    neighborLabels := []string{
        strings.ToLower(vrfName),
        strings.ToLower(ifaceName),
        neighborIP,
    }
    newGauge(ch, p.descriptions["upTime"], float64(uptimeSec), neighborLabels...)
    return nil
}

func NewNeighborCounter() *NeighborCounter {
    return &NeighborCounter{}
}

func (c *NeighborCounter) Increment() {
    c.count++
}

func (c *NeighborCounter) Count() float64 {
    return c.count
}

func parseHMS(st string) (uint64, error) {
    var h, m, s uint64
    n, err := fmt.Sscanf(st, "%d:%d:%d", &h, &m, &s)
    if err != nil || n != 3 {
        return 0, fmt.Errorf("invalid time format: %w", err)
    }
    return h*3600 + m*60 + s, nil
}

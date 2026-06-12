
package metrics

import (
    "encoding/json"
    "log/slog"
    "strconv"
    "strings"
    "time"

    "github.com/prometheus/client_golang/prometheus"
)

const (
    defaultVrrpCommandTimeout = 30 * time.Second
    vrrpStatusInitialize = "Initialize"
    vrrpStatusBackup     = "Backup"
    vrrpStatusMaster     = "Master"
)

type VRRPConfig struct {
    CommandTimeout time.Duration
    Logger         *slog.Logger
}

type CommandExecutor interface {
    Execute(command string, timeout time.Duration) ([]byte, error)
}

type DefaultVrrpCommandExecutor struct{}

type VRRPProcessor interface {
    Process(ch chan<- prometheus.Metric, data []byte, desc map[string]*prometheus.Desc) error
}

type vrrpCollector struct {
    logger       *slog.Logger
    descriptions map[string]*prometheus.Desc
    executor     CommandExecutor
    processor    VRRPProcessor
    config       *VRRPConfig
}

func NewVRRPConfig(logger *slog.Logger) *VRRPConfig {
    return &VRRPConfig{
        CommandTimeout: defaultVrrpCommandTimeout,
        Logger:         logger,
    }
}

func init() {
    registerCollector(vrrpSubsystem, disabledByDefault, NewVRRPCollector)
}

func NewVRRPCollector(logger *slog.Logger) (Collector, error) {
    config := NewVRRPConfig(logger)
    return NewVRRPCollectorWithConfig(config)
}

func NewVRRPCollectorWithConfig(config *VRRPConfig) (Collector, error) {
    return &vrrpCollector{
        logger:       config.Logger,
        descriptions: getVRRPDesc(),
        executor:     &DefaultVrrpCommandExecutor{},
        processor:    NewVRRPProcessor(config.Logger),
        config:       config,
    }, nil
}

func (e *DefaultVrrpCommandExecutor) Execute(command string, timeout time.Duration) ([]byte, error) {
    return executeVRRPCommand(command)
}

func (c *vrrpCollector) Update(ch chan<- prometheus.Metric) error {
    cmd := "show vrrp json"
    data, err := c.executor.Execute(cmd, c.config.CommandTimeout)
    if err != nil {
        return err
    }
    if err := c.processor.Process(ch, data, c.descriptions); err != nil {
        return cmdOutputProcessError(cmd, string(data), err)
    }
    return nil
}

type VRRPProcessorImpl struct {
    logger *slog.Logger
}

func NewVRRPProcessor(logger *slog.Logger) VRRPProcessor {
    return &VRRPProcessorImpl{
        logger: logger,
    }
}

func (p *VRRPProcessorImpl) Process(ch chan<- prometheus.Metric, data []byte, desc map[string]*prometheus.Desc) error {
    var vrInfos []VrrpVrInfo
    if err := json.Unmarshal(data, &vrInfos); err != nil {
        return err
    }

    for _, vrInfo := range vrInfos {
        instanceProcessor := NewVRRPInstanceProcessor(desc)
        instanceProcessor.Process(ch, "v4", vrInfo.Vrid, vrInfo.Interface, vrInfo.V4Info)
        instanceProcessor.Process(ch, "v6", vrInfo.Vrid, vrInfo.Interface, vrInfo.V6Info)
    }
    return nil
}

type VRRPInstanceProcessor struct {
    desc map[string]*prometheus.Desc
}

func NewVRRPInstanceProcessor(desc map[string]*prometheus.Desc) *VRRPInstanceProcessor {
    return &VRRPInstanceProcessor{
        desc: desc,
    }
}

func (p *VRRPInstanceProcessor) Process(ch chan<- prometheus.Metric, proto string, vrid uint32, iface string, instance VrrpInstanceInfo) {
    labelGenerator := NewVRRPLabelGenerator(proto, vrid, iface, instance.Subinterface)
    p.processStateMetrics(ch, instance.Status, labelGenerator)
    p.processStatistics(ch, instance.Statistics, labelGenerator)
}

func (p *VRRPInstanceProcessor) processStateMetrics(ch chan<- prometheus.Metric, status string, labelGenerator *VRRPLabelGenerator) {
    for _, state := range vrrpStates {
        value := 0.0
        if strings.EqualFold(status, state) {
            value = 1.0
        }
        newGauge(ch, p.desc["vrrpState"], value, labelGenerator.GetStateLabels(state)...)
    }
}

func (p *VRRPInstanceProcessor) processStatistics(ch chan<- prometheus.Metric, stats VrrpInstanceStats, labelGenerator *VRRPLabelGenerator) {
    labels := labelGenerator.GetBaseLabels()
    if stats.AdverTx != nil {
        newCounter(ch, p.desc["adverTx"], float64(*stats.AdverTx), labels...)
    }
    if stats.AdverRx != nil {
        newCounter(ch, p.desc["adverRx"], float64(*stats.AdverRx), labels...)
    }
    if stats.GarpTx != nil {
        newCounter(ch, p.desc["garpTx"], float64(*stats.GarpTx), labels...)
    }
    if stats.NeighborAdverTx != nil {
        newCounter(ch, p.desc["neighborAdverTx"], float64(*stats.NeighborAdverTx), labels...)
    }
    if stats.Transitions != nil {
        newCounter(ch, p.desc["transitions"], float64(*stats.Transitions), labels...)
    }
}

type VRRPLabelGenerator struct {
    proto        string
    vrid         uint32
    iface        string
    subinterface string
}

func NewVRRPLabelGenerator(proto string, vrid uint32, iface, subinterface string) *VRRPLabelGenerator {
    return &VRRPLabelGenerator{
        proto:        proto,
        vrid:         vrid,
        iface:        iface,
        subinterface: subinterface,
    }
}

func (g *VRRPLabelGenerator) GetBaseLabels() []string {
    return []string{
        g.proto,
        strconv.FormatUint(uint64(g.vrid), 10),
        g.iface,
        g.subinterface,
    }
}

func (g *VRRPLabelGenerator) GetStateLabels(state string) []string {
    return append(g.GetBaseLabels(), state)
}

var (
    vrrpSubsystem = "vrrp"
    vrrpStates    = []string{vrrpStatusInitialize, vrrpStatusMaster, vrrpStatusBackup}
)

func getVRRPDesc() map[string]*prometheus.Desc {
    labels := []string{"proto", "vrid", "interface", "subinterface"}
    stateLabels := append(labels, "state")

    return map[string]*prometheus.Desc{
        "vrrpState":       colPromDesc(vrrpSubsystem, "state", "Status of the VRRP state machine.", stateLabels),
        "adverTx":         colPromDesc(vrrpSubsystem, "advertisements_sent_total", "Advertisements sent total.", labels),
        "adverRx":         colPromDesc(vrrpSubsystem, "advertisements_received_total", "Advertisements received total.", labels),
        "garpTx":          colPromDesc(vrrpSubsystem, "gratuitous_arp_sent_total", "Gratuitous ARP sent total.", labels),
        "neighborAdverTx": colPromDesc(vrrpSubsystem, "neighbor_advertisements_sent_total", "Neighbor Advertisements sent total.", labels),
        "transitions":     colPromDesc(vrrpSubsystem, "state_transitions_total", "Number of transitions of the VRRP state machine in total.", labels),
    }
}

type VrrpVrInfo struct {
    Vrid      uint32
    Interface string
    V6Info    VrrpInstanceInfo `json:"v6"`
    V4Info    VrrpInstanceInfo `json:"v4"`
}

type VrrpInstanceInfo struct {
    Subinterface string `json:"interface"`
    Status       string
    Statistics   VrrpInstanceStats `json:"stats"`
}

type VrrpInstanceStats struct {
    AdverTx         *uint32
    AdverRx         *uint32
    GarpTx          *uint32
    NeighborAdverTx *uint32
    Transitions     *uint32
}

package metrics

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	ospfSubsystem    = "ospf"
	frrOSPFInstances = kingpin.Flag("collector.ospf.instances", "Comma-separated list of instance IDs if using multiple OSPF instances").Default("").String()
)

func init() {
	registerCollector(ospfSubsystem, enabledByDefault, NewOSPFCollector)
}

type OSPFCollector struct {
	logger                *slog.Logger
	interfaceDescriptors  map[string]*prometheus.Desc
	routerDescriptors     map[string]*prometheus.Desc
	areaDescriptors       map[string]*prometheus.Desc
	instanceIdentifiers   []int
	commandExecutor       OSPFCommandExecutor
	processor             OSPFDataProcessor
	metricEmitter         MetricEmitter
}

type OSPFCommandExecutor interface {
	ExecuteSingleInstanceCommand(cmd string) ([]byte, error)
	ExecuteMultiInstanceCommand(cmd string, instanceID int) ([]byte, error)
}

type OSPFDataProcessor interface {
	ProcessInterfaceData(data []byte, instanceID int) ([]OSPFInterfaceMetric, error)
	ProcessRouterData(data []byte, instanceID int) ([]OSPCRouterMetric, error)
}

type MetricEmitter interface {
	EmitGauge(ch chan<- prometheus.Metric, desc *prometheus.Desc, value float64, labels ...string)
	EmitCounter(ch chan<- prometheus.Metric, desc *prometheus.Desc, value float64, labels ...string)
}

type DefaultOSPFCommandExecutor struct{}

type DefaultOSPFDataProcessor struct{}

type DefaultMetricEmitter struct{}

func NewOSPFCollector(logger *slog.Logger) (Collector, error) {
	instanceIDs, err := parseInstanceIDs(*frrOSPFInstances)
	if err != nil {
		return nil, err
	}

	if *vtyshEnable && len(instanceIDs) > 0 {
		return nil, fmt.Errorf("cannot use --frr.vtysh with --collector.ospf.instances")
	}

	return &OSPFCollector{
		logger:               logger,
		interfaceDescriptors: createOSPFInterfaceDescriptors(),
		routerDescriptors:    createOSPCRouterDescriptors(),
		areaDescriptors:      createOSPFAreaDescriptors(),
		instanceIdentifiers:  instanceIDs,
		commandExecutor:      &DefaultOSPFCommandExecutor{},
		processor:           &DefaultOSPFDataProcessor{},
		metricEmitter:       &DefaultMetricEmitter{},
	}, nil
}

func parseInstanceIDs(instances string) ([]int, error) {
	if len(instances) == 0 {
		return nil, nil
	}

	var instanceIDs []int
	for _, id := range strings.Split(instances, ",") {
		parsedID, err := strconv.Atoi(id)
		if err != nil {
			return nil, fmt.Errorf("unable to parse instance ID %s: %w", id, err)
		}
		instanceIDs = append(instanceIDs, parsedID)
	}
	return instanceIDs, nil
}

func createOSPFInterfaceDescriptors() map[string]*prometheus.Desc {
	baseLabels := []string{"vrf", "iface", "area"}
	if len(*frrOSPFInstances) > 0 {
		baseLabels = append(baseLabels, "instance")
	}

	return map[string]*prometheus.Desc{
		"neighbors":            colPromDesc(ospfSubsystem, "neighbors", "Number of neighbors detected", baseLabels),
		"neighbor_adjacencies": colPromDesc(ospfSubsystem, "neighbor_adjacencies", "Number of neighbor adjacencies formed", baseLabels),
	}
}

func createOSPCRouterDescriptors() map[string]*prometheus.Desc {
	baseLabels := []string{"vrf"}
	if len(*frrOSPFInstances) > 0 {
		baseLabels = append(baseLabels, "instance")
	}

	return map[string]*prometheus.Desc{
		"lsa_external_counter": colPromDesc(ospfSubsystem, "lsa_external_counter", "Number of external LSAs", baseLabels),
		"lsa_as_opaque_counter": colPromDesc(ospfSubsystem, "lsa_as_opaque_counter", "Number of AS Opaque LSAs", baseLabels),
	}
}

func createOSPFAreaDescriptors() map[string]*prometheus.Desc {
	baseLabels := []string{"vrf", "area"}
	if len(*frrOSPFInstances) > 0 {
		baseLabels = append(baseLabels, "instance")
	}

	return map[string]*prometheus.Desc{
		"area_lsa_number":         colPromDesc(ospfSubsystem, "area_lsa_number", "Number of LSAs in the area", baseLabels),
		"area_lsa_network_number": colPromDesc(ospfSubsystem, "area_lsa_network_number", "Number of network LSAs in the area", baseLabels),
		"area_lsa_summary_number": colPromDesc(ospfSubsystem, "area_lsa_summary_number", "Number of summary LSAs in the area", baseLabels),
		"area_lsa_asbr_number":    colPromDesc(ospfSubsystem, "area_lsa_asbr_number", "Number of ASBR LSAs in the area", baseLabels),
		"area_lsa_nssa_number":    colPromDesc(ospfSubsystem, "area_lsa_nssa_number", "Number of NSSA LSAs in the area", baseLabels),
	}
}

func (c *OSPFCollector) Update(ch chan<- prometheus.Metric) error {
	if err := c.collectRouterMetrics(ch); err != nil {
		return err
	}

	if err := c.collectInterfaceMetrics(ch); err != nil {
		return err
	}

	return nil
}

func (c *OSPFCollector) collectRouterMetrics(ch chan<- prometheus.Metric) error {
	command := "show ip ospf vrf all json"
	processor := c.processRouterData

	if len(c.instanceIdentifiers) > 0 {
		for _, instanceID := range c.instanceIdentifiers {
			data, err := c.commandExecutor.ExecuteMultiInstanceCommand(command, instanceID)
			if err != nil {
				return err
			}

			metrics, err := processor(data, instanceID)
			if err != nil {
				return cmdOutputProcessError(command, string(data), err)
			}

			c.emitRouterMetrics(ch, metrics)
		}
		return nil
	}

	data, err := c.commandExecutor.ExecuteSingleInstanceCommand(command)
	if err != nil {
		return err
	}

	metrics, err := processor(data, 0)
	if err != nil {
		return cmdOutputProcessError(command, string(data), err)
	}

	c.emitRouterMetrics(ch, metrics)
	return nil
}

func (c *OSPFCollector) collectInterfaceMetrics(ch chan<- prometheus.Metric) error {
	command := "show ip ospf vrf all interface json"
	processor := c.processInterfaceData

	if len(c.instanceIdentifiers) > 0 {
		for _, instanceID := range c.instanceIdentifiers {
			data, err := c.commandExecutor.ExecuteMultiInstanceCommand(command, instanceID)
			if err != nil {
				return err
			}

			metrics, err := processor(data, instanceID)
			if err != nil {
				return cmdOutputProcessError(command, string(data), err)
			}

			c.emitInterfaceMetrics(ch, metrics)
		}
		return nil
	}

	data, err := c.commandExecutor.ExecuteSingleInstanceCommand(command)
	if err != nil {
		return err
	}

	metrics, err := processor(data, 0)
	if err != nil {
		return cmdOutputProcessError(command, string(data), err)
	}

	c.emitInterfaceMetrics(ch, metrics)
	return nil
}

func (c *OSPFCollector) processRouterData(data []byte, instanceID int) ([]OSPCRouterMetric, error) {
	var instances map[string]OSPFInstance
	if err := json.Unmarshal(data, &instances); err != nil {
		return nil, fmt.Errorf("cannot unmarshal ospf json: %w", err)
	}

	var metrics []OSPCRouterMetric
	for vrfName, instanceData := range instances {
		metrics = append(metrics, OSPCRouterMetric{
			VRF:          vrfName,
			InstanceID:   instanceID,
			ExternalLSAs: instanceData.LsaExternalCounter,
			ASOpaqueLSAs: instanceData.LsaAsopaqueCounter,
			Areas:        instanceData.Areas,
		})
	}

	return metrics, nil
}

func (c *OSPFCollector) processInterfaceData(data []byte, instanceID int) ([]OSPFInterfaceMetric, error) {
	var rawData map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawData); err != nil {
		return nil, fmt.Errorf("cannot unmarshal ospf interface json: %w", err)
	}

	var metrics []OSPFInterfaceMetric
	for vrfName, vrfData := range rawData {
		if vrfName == "ospfInstance" {
			continue
		}

		vrfMetrics, err := c.extractVRFInterfaceMetrics(vrfName, vrfData, instanceID)
		if err != nil {
			return nil, err
		}

		metrics = append(metrics, vrfMetrics...)
	}

	return metrics, nil
}

func (c *OSPFCollector) extractVRFInterfaceMetrics(vrfName string, vrfData json.RawMessage, instanceID int) ([]OSPFInterfaceMetric, error) {
	var vrfInstance map[string]json.RawMessage
	if err := json.Unmarshal(vrfData, &vrfInstance); err != nil {
		return nil, fmt.Errorf("cannot unmarshal VRF instance json: %w", err)
	}

	var metrics []OSPFInterfaceMetric
	for key, value := range vrfInstance {
		switch key {
		case "vrfName", "vrfId":
			continue
		case "interfaces":
			interfaceMetrics, err := c.extractInterfacesFromContainer(vrfName, value, instanceID)
			if err != nil {
				return nil, err
			}
			metrics = append(metrics, interfaceMetrics...)
		default:
			interfaceMetric, err := c.extractSingleInterfaceMetric(vrfName, key, value, instanceID)
			if err != nil {
				return nil, err
			}
			if interfaceMetric != nil {
				metrics = append(metrics, *interfaceMetric)
			}
		}
	}

	return metrics, nil
}

func (c *OSPFCollector) extractInterfacesFromContainer(vrfName string, data json.RawMessage, instanceID int) ([]OSPFInterfaceMetric, error) {
	var interfaces map[string]json.RawMessage
	if err := json.Unmarshal(data, &interfaces); err != nil {
		return nil, fmt.Errorf("cannot unmarshal interface container json: %w", err)
	}

	var metrics []OSPFInterfaceMetric
	for ifaceName, ifaceData := range interfaces {
		metric, err := c.createInterfaceMetric(vrfName, ifaceName, ifaceData, instanceID)
		if err != nil {
			return nil, err
		}
		if metric != nil {
			metrics = append(metrics, *metric)
		}
	}

	return metrics, nil
}

func (c *OSPFCollector) extractSingleInterfaceMetric(vrfName, ifaceName string, data json.RawMessage, instanceID int) (*OSPFInterfaceMetric, error) {
	var iface OSPFInterface
	if err := json.Unmarshal(data, &iface); err != nil {
		return nil, fmt.Errorf("cannot unmarshal interface json: %w", err)
	}

	if iface.TimerPassiveIface {
		return nil, nil
	}

	return &OSPFInterfaceMetric{
		VRF:             vrfName,
		Interface:       ifaceName,
		Area:            iface.Area,
		InstanceID:      instanceID,
		NeighborCount:   iface.NbrCount,
		AdjacencyCount:  iface.NbrAdjacentCount,
	}, nil
}

func (c *OSPFCollector) createInterfaceMetric(vrfName, ifaceName string, data json.RawMessage, instanceID int) (*OSPFInterfaceMetric, error) {
	var iface OSPFInterface
	if err := json.Unmarshal(data, &iface); err != nil {
		return nil, fmt.Errorf("cannot unmarshal interface json: %w", err)
	}

	if iface.TimerPassiveIface {
		return nil, nil
	}

	return &OSPFInterfaceMetric{
		VRF:             vrfName,
		Interface:       ifaceName,
		Area:            iface.Area,
		InstanceID:      instanceID,
		NeighborCount:   iface.NbrCount,
		AdjacencyCount:  iface.NbrAdjacentCount,
	}, nil
}

func (c *OSPFCollector) emitRouterMetrics(ch chan<- prometheus.Metric, metrics []OSPCRouterMetric) {
	for _, metric := range metrics {
		c.emitRouterMetric(ch, metric)
		c.emitAreaMetrics(ch, metric)
	}
}

func (c *OSPFCollector) emitRouterMetric(ch chan<- prometheus.Metric, metric OSPCRouterMetric) {
	labels := []string{strings.ToLower(metric.VRF)}
	if metric.InstanceID != 0 {
		labels = append(labels, strconv.Itoa(metric.InstanceID))
	}

	c.metricEmitter.EmitGauge(ch, c.routerDescriptors["lsa_external_counter"], float64(metric.ExternalLSAs), labels...)
	c.metricEmitter.EmitGauge(ch, c.routerDescriptors["lsa_as_opaque_counter"], float64(metric.ASOpaqueLSAs), labels...)
}

func (c *OSPFCollector) emitAreaMetrics(ch chan<- prometheus.Metric, metric OSPCRouterMetric) {
	for areaName, area := range metric.Areas {
		labels := []string{strings.ToLower(metric.VRF), areaName}
		if metric.InstanceID != 0 {
			labels = append(labels, strconv.Itoa(metric.InstanceID))
		}

		c.metricEmitter.EmitGauge(ch, c.areaDescriptors["area_lsa_number"], float64(area.LsaNumber), labels...)
		c.metricEmitter.EmitGauge(ch, c.areaDescriptors["area_lsa_network_number"], float64(area.LsaNetworkNumber), labels...)
		c.metricEmitter.EmitGauge(ch, c.areaDescriptors["area_lsa_summary_number"], float64(area.LsaSummaryNumber), labels...)
		c.metricEmitter.EmitGauge(ch, c.areaDescriptors["area_lsa_asbr_number"], float64(area.LsaAsbrNumber), labels...)
		c.metricEmitter.EmitGauge(ch, c.areaDescriptors["area_lsa_nssa_number"], float64(area.LsaNssaNumber), labels...)
	}
}

func (c *OSPFCollector) emitInterfaceMetrics(ch chan<- prometheus.Metric, metrics []OSPFInterfaceMetric) {
	for _, metric := range metrics {
		labels := []string{
			strings.ToLower(metric.VRF),
			metric.Interface,
			metric.Area,
		}
		if metric.InstanceID != 0 {
			labels = append(labels, strconv.Itoa(metric.InstanceID))
		}

		c.metricEmitter.EmitGauge(ch, c.interfaceDescriptors["neighbors"], float64(metric.NeighborCount), labels...)
		c.metricEmitter.EmitGauge(ch, c.interfaceDescriptors["neighbor_adjacencies"], float64(metric.AdjacencyCount), labels...)
	}
}

type OSPFInterfaceMetric struct {
	VRF            string
	Interface      string
	Area           string
	InstanceID     int
	NeighborCount  uint32
	AdjacencyCount uint32
}

type OSPCRouterMetric struct {
	VRF          string
	InstanceID   int
	ExternalLSAs uint32
	ASOpaqueLSAs uint32
	Areas        map[string]OSPFArea
}

type OSPFInterface struct {
	NbrCount          uint32
	NbrAdjacentCount  uint32
	Area              string
	TimerPassiveIface bool
}

type OSPFInstance struct {
	LsaExternalCounter uint32
	LsaAsopaqueCounter uint32
	Areas              map[string]OSPFArea
}

type OSPFArea struct {
	LsaNumber        uint32
	LsaNetworkNumber uint32
	LsaSummaryNumber uint32
	LsaAsbrNumber    uint32
	LsaNssaNumber    uint32
}

func (e *DefaultOSPFCommandExecutor) ExecuteSingleInstanceCommand(cmd string) ([]byte, error) {
	return executeOSPFCommand(cmd)
}

func (e *DefaultOSPFCommandExecutor) ExecuteMultiInstanceCommand(cmd string, instanceID int) ([]byte, error) {
	return executeOSPFMultiInstanceCommand(cmd, instanceID)
}

func (p *DefaultOSPFDataProcessor) ProcessInterfaceData(data []byte, instanceID int) ([]OSPFInterfaceMetric, error) {
	// Implementation would mirror the existing processInterfaceData method
	return nil, nil
}

func (p *DefaultOSPFDataProcessor) ProcessRouterData(data []byte, instanceID int) ([]OSPCRouterMetric, error) {
	// Implementation would mirror the existing processRouterData method
	return nil, nil
}

func (e *DefaultMetricEmitter) EmitGauge(ch chan<- prometheus.Metric, desc *prometheus.Desc, value float64, labels ...string) {
	newGauge(ch, desc, value, labels...)
}

func (e *DefaultMetricEmitter) EmitCounter(ch chan<- prometheus.Metric, desc *prometheus.Desc, value float64, labels ...string) {
	// Not currently used but implemented for completeness
	newCounter(ch, desc, value, labels...)
}
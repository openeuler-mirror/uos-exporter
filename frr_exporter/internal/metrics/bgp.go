package metrics

import (
	"encoding/json"
	"fmt"
	
	"log/slog"
	"strconv"
	"strings"
	"sync"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	bgpSubsystem = "bgp"

	bgpPeerTypes                = kingpin.Flag("collector.bgp.peer-types", "Enable the frr_bgp_peer_types_up metric (default: disabled).").Default("False").Bool()
	frrBGPDescKey               = kingpin.Flag("collector.bgp.peer-types.keys", "Select the keys from the JSON formatted BGP peer description of which the values will be used with the frr_bgp_peer_types_up metric. Supports multiple values (default: type).").Default("type").Strings()
	bgpPeerDescs                = kingpin.Flag("collector.bgp.peer-descriptions", "Add the value of the desc key from the JSON formatted BGP peer description as a label to peer metrics. (default: disabled).").Default("False").Bool()
	bgpPeerGroups               = kingpin.Flag("collector.bgp.peer-groups", "Adds the peer's peer group name as a label. (default: disabled).").Default("False").Bool()
	bgpPeerHostnames            = kingpin.Flag("collector.bgp.peer-hostnames", "Adds the peer's hostname as a label. (default: disabled).").Default("False").Bool()
	bgpPeerDescsText            = kingpin.Flag("collector.bgp.peer-descriptions.plain-text", "Use the full text field of the BGP peer description instead of the value of the JSON formatted desc key (default: disabled).").Default("False").Bool()
	bgpAdvertisedPrefixes       = kingpin.Flag("collector.bgp.advertised-prefixes", "Enables the frr_exporter_bgp_prefixes_advertised_count_total metric which exports the number of advertised prefixes to a BGP peer. This is an option for older versions of FRR that don't have PfxSent field (default: disabled).").Default("False").Bool()
	bgpAcceptedFilteredPrefixes = kingpin.Flag("collector.bgp.accepted-filtered-prefixes", "Enable retrieval of accepted and filtered BGP prefix counts (default: disabled).").Default("False").Bool()
)

func init() {
	registerCollector(bgpSubsystem, enabledByDefault, NewBGPCollector)
	registerCollector(bgpSubsystem+"6", disabledByDefault, NewBGP6Collector)
	registerCollector(bgpSubsystem+"l2vpn", disabledByDefault, NewBGPL2VPNCollector)
}

var (
    executeBGPCommandFunc   = executeBGPCommand   // 包装函数变量
    executeZebraCommandFunc = executeZebraCommand // 包装函数变量
)

// type CommandBgpExecutor interface {
//     ExecuteBGPCommand(cmd string) ([]byte, error)
//     ExecuteZebraCommand(cmd string) ([]byte, error)
// }

// type defaultExecutor struct{}

// func (e *defaultExecutor) ExecuteBGPCommand(cmd string) ([]byte, error) {
//     return executeBGPCommand(cmd) // 调用原始函数
// }

// func (e *defaultExecutor) ExecuteZebraCommand(cmd string) ([]byte, error) {
//     return executeZebraCommand(cmd) // 调用原始函数
// }

type bgpCollector struct {
	logger       *slog.Logger
	descriptions map[string]*prometheus.Desc
	afi          string
	executor      func(string) ([]byte, error)
}

func NewBGPCollector(logger *slog.Logger) (Collector, error) {
	return &bgpCollector{
		logger:       logger,
		descriptions: createBGPDescriptions(),
		afi:          "ipv4",
	}, nil
}

type bgpL2VPNCollector struct {
	logger       *slog.Logger
	descriptions map[string]*prometheus.Desc
}

func NewBGP6Collector(logger *slog.Logger) (Collector, error) {
	return &bgpCollector{
		logger:       logger,
		descriptions: createBGPDescriptions(),
		afi:          "ipv6",
	}, nil
}

func NewBGPL2VPNCollector(logger *slog.Logger) (Collector, error) {
	return &bgpL2VPNCollector{
		logger:       logger,
		descriptions: createBGPL2VPNDescriptions(),
	}, nil
}

func createBGPDescriptions() map[string]*prometheus.Desc {
	bgpLabels := []string{"vrf", "afi", "safi", "local_as"}
	bgpPeerTypeLabels := []string{"type", "afi", "safi"}
	bgpPeerLabels := append(bgpLabels, "peer", "peer_as")

	if *bgpPeerDescs {
		bgpPeerLabels = append(bgpPeerLabels, "peer_desc")
	}

	if *bgpPeerHostnames {
		bgpPeerLabels = append(bgpPeerLabels, "peer_hostname")
	}

	if *bgpPeerGroups {
		bgpPeerLabels = append(bgpPeerLabels, "peer_group")
	}

	return map[string]*prometheus.Desc{
		"ribCount":              colPromDesc(bgpSubsystem, "rib_count_total", "Number of routes in the RIB.", bgpLabels),
		"ribMemory":             colPromDesc(bgpSubsystem, "rib_memory_bytes", "Memory consumbed by the RIB.", bgpLabels),
		"peerCount":             colPromDesc(bgpSubsystem, "peers_count_total", "Number peers configured.", bgpLabels),
		"peerMemory":            colPromDesc(bgpSubsystem, "peers_memory_bytes", "Memory consumed by peers.", bgpLabels),
		"peerGroupCount":        colPromDesc(bgpSubsystem, "peer_groups_count_total", "Number of peer groups configured.", bgpLabels),
		"peerGroupMemory":       colPromDesc(bgpSubsystem, "peer_groups_memory_bytes", "Memory consumed by peer groups.", bgpLabels),
		"msgRcvd":               colPromDesc(bgpSubsystem, "peer_message_received_total", "Number of received messages.", bgpPeerLabels),
		"msgSent":               colPromDesc(bgpSubsystem, "peer_message_sent_total", "Number of sent messages.", bgpPeerLabels),
		"prefixReceivedCount":   colPromDesc(bgpSubsystem, "peer_prefixes_received_count_total", "Number of prefixes received.", bgpPeerLabels),
		"prefixAdvertisedCount": colPromDesc(bgpSubsystem, "peer_prefixes_advertised_count_total", "Number of prefixes advertised.", bgpPeerLabels),
		"prefixAcceptedCount":   colPromDesc(bgpSubsystem, "peer_prefixes_accepted_count_total", "Number of prefixes accepted.", bgpPeerLabels),
		"prefixFilteredCount":   colPromDesc(bgpSubsystem, "peer_prefixes_filtered_count_total", "Number of prefixes filtered.", bgpPeerLabels),
		"state":                 colPromDesc(bgpSubsystem, "peer_state", "State of the peer (2 = Administratively Down, 1 = Established, 0 = Down).", bgpPeerLabels),
		"UptimeSec":             colPromDesc(bgpSubsystem, "peer_uptime_seconds", "How long has the peer been up.", bgpPeerLabels),
		"peerTypesUp":           colPromDesc(bgpSubsystem, "peer_types_up", "Total Number of Peer Types that are Up.", bgpPeerTypeLabels),
	}
}

func createBGPL2VPNDescriptions() map[string]*prometheus.Desc {
	bgpDesc := createBGPDescriptions()
	labels := []string{"vni", "type", "vxlanIf", "tenantVrf"}
	metricPrefix := "bgp_l2vpn_evpn"

	bgpDesc["numMacs"] = colPromDesc(metricPrefix, "mac_count_total", "Number of known MAC addresses", labels)
	bgpDesc["numArpNd"] = colPromDesc(metricPrefix, "arp_nd_count_total", "Number of ARP / ND entries", labels)
	bgpDesc["numRemoteVteps"] = colPromDesc(metricPrefix, "remote_vtep_count_total", "Number of known remote VTEPs. A value of -1 indicates a non-integer output from FRR, such as n/a.", labels)

	return bgpDesc
}

func (c *bgpCollector) Update(ch chan<- prometheus.Metric) error {
	return gatherBGPData(ch, c.afi, c.logger, c.descriptions)
}

func (c *bgpL2VPNCollector) Update(ch chan<- prometheus.Metric) error {
	if err := gatherBGPData(ch, "l2vpn", c.logger, c.descriptions); err != nil {
		return err
	}
	return processL2VPNData(ch, c.descriptions, c.logger)
}

func processL2VPNData(ch chan<- prometheus.Metric, desc map[string]*prometheus.Desc, logger *slog.Logger) error {
	cmd := "show evpn vni json"
	jsonData, err := executeZebraCommand(cmd)
	if err != nil {
		return err
	}
	if len(jsonData) == 0 {
		return nil
	}
	
	return handleL2VPNSummary(ch, jsonData, desc)
}

func handleL2VPNSummary(ch chan<- prometheus.Metric, jsonData []byte, desc map[string]*prometheus.Desc) error {
	var vxlanStats map[string]vxLanStat
	if err := json.Unmarshal(jsonData, &vxlanStats); err != nil {
		return err
	}

	for _, stat := range vxlanStats {
		processVxlanStat(ch, stat, desc)
	}
	return nil
}

func processVxlanStat(ch chan<- prometheus.Metric, stat vxLanStat, desc map[string]*prometheus.Desc) {
	labels := []string{
		strconv.FormatUint(uint64(stat.Vni), 10),
		stat.VxlanType,
		stat.VxlanIf,
		stat.TenantVrf,
	}
	
	newGauge(ch, desc["numMacs"], float64(stat.NumMacs), labels...)
	newGauge(ch, desc["numArpNd"], float64(stat.NumArpNd), labels...)
	
	remoteVteps := convertRemoteVteps(stat.NumRemoteVteps)
	newGauge(ch, desc["numRemoteVteps"], remoteVteps, labels...)
}

func convertRemoteVteps(v interface{}) float64 {
	if val, ok := v.(float64); ok {
		return val
	}
	return -1
}

func gatherBGPData(ch chan<- prometheus.Metric, afi string, logger *slog.Logger, desc map[string]*prometheus.Desc) error {
	safi := determineSAFI(afi)
	cmd := buildBGPCommand(afi, safi)
	
	jsonData, err := executeBGPCommandFunc(cmd)
	if err != nil {
		return err
	}
	
	return processBGPData(ch, jsonData, afi, safi, logger, desc)
}

func determineSAFI(afi string) string {
	if afi == "l2vpn" {
		return "evpn"
	}
	return ""
}

func buildBGPCommand(afi, safi string) string {
	return fmt.Sprintf("show bgp vrf all %s %s summary json", afi, safi)
}

func processBGPData(ch chan<- prometheus.Metric, jsonData []byte, afi, safi string, logger *slog.Logger, desc map[string]*prometheus.Desc) error {
	data, err := parseBGPData(jsonData, afi, safi)
	if err != nil {
		return cmdOutputProcessError(buildBGPCommand(afi, safi), string(jsonData), err)
	}

	peerDesc, err := retrievePeerDescriptions()
	if err != nil {
		return err
	}

	return processVRFs(ch, data, afi, safi, peerDesc, logger, desc)
}

func parseBGPData(jsonData []byte, afi, safi string) (map[string]map[string]bgpProcess, error) {
	if afi == "l2vpn" && safi == "evpn" {
		return handleL2VPNData(jsonData)
	}
	return parseStandardBGPData(jsonData)
}

func handleL2VPNData(jsonData []byte) (map[string]map[string]bgpProcess, error) {
	var tempData map[string]bgpProcess
	if err := json.Unmarshal(jsonData, &tempData); err != nil {
		return nil, err
	}

	result := make(map[string]map[string]bgpProcess)
	for vrf, data := range tempData {
		result[vrf] = map[string]bgpProcess{"xxxxevpn": data}
	}
	return result, nil
}

func parseStandardBGPData(jsonData []byte) (map[string]map[string]bgpProcess, error) {
	var data map[string]map[string]bgpProcess
	err := json.Unmarshal(jsonData, &data)
	return data, err
}

func retrievePeerDescriptions() (map[string]bgpVRF, error) {
	if !(*bgpPeerTypes || *bgpPeerDescs || *bgpPeerGroups) {
		return nil, nil
	}
	return fetchBGPPeerDescriptions()
}

func fetchBGPPeerDescriptions() (map[string]bgpVRF, error) {
	output, err := executeBGPCommandFunc("show bgp vrf all neighbors json")
	if err != nil {
		return nil, err
	}
	return parsePeerDescriptions(output)
}

func parsePeerDescriptions(output []byte) (map[string]bgpVRF, error) {
	var vrfMap map[string]bgpVRF
	if err := json.Unmarshal(output, &vrfMap); err != nil {
		return nil, err
	}
	return vrfMap, nil
}

func processVRFs(ch chan<- prometheus.Metric, data map[string]map[string]bgpProcess, afi, safi string, peerDesc map[string]bgpVRF, logger *slog.Logger, desc map[string]*prometheus.Desc) error {
	var wg sync.WaitGroup
	peerTypes := make(map[string]map[string]float64)

	for vrfName, vrfData := range data {
		for safiName, safiData := range vrfData {
			err := processSAFI(ch, safiData, vrfName, afi, safiName, peerDesc, logger, desc, &wg, peerTypes)
			if err != nil {
				return err
			}
		}
	}

	wg.Wait()
	emitPeerTypeMetrics(ch, peerTypes, afi, desc)
	return nil
}

func processSAFI(ch chan<- prometheus.Metric, safiData bgpProcess, vrfName, afi, safiName string, peerDesc map[string]bgpVRF, logger *slog.Logger, desc map[string]*prometheus.Desc, wg *sync.WaitGroup, peerTypes map[string]map[string]float64) error {
	if safiData.PeerCount == 0 {
		return nil
	}

	localAs := strconv.FormatUint(uint64(safiData.AS), 10)
	procLabels := []string{strings.ToLower(vrfName), strings.ToLower(afi), strings.ToLower(safiName[4:]), localAs}
	emitProcessMetrics(ch, safiData, desc, procLabels)

	return processPeers(ch, safiData.Peers, vrfName, afi, safiName, localAs, peerDesc, logger, desc, wg, peerTypes)
}

func emitProcessMetrics(ch chan<- prometheus.Metric, data bgpProcess, desc map[string]*prometheus.Desc, labels []string) {
	newGauge(ch, desc["ribCount"], float64(data.RIBCount), labels...)
	newGauge(ch, desc["ribMemory"], float64(data.RIBMemory), labels...)
	newGauge(ch, desc["peerCount"], float64(data.PeerCount), labels...)
	newGauge(ch, desc["peerMemory"], float64(data.PeerMemory), labels...)
	newGauge(ch, desc["peerGroupCount"], float64(data.PeerGroupCount), labels...)
	newGauge(ch, desc["peerGroupMemory"], float64(data.PeerGroupMemory), labels...)
}

func processPeers(ch chan<- prometheus.Metric, peers map[string]*bgpPeerSession, vrfName, afi, safiName, localAs string, peerDesc map[string]bgpVRF, logger *slog.Logger, desc map[string]*prometheus.Desc, wg *sync.WaitGroup, peerTypes map[string]map[string]float64) error {
	for peerIP, peerData := range peers {
		peerLabels := buildPeerLabels(vrfName, afi, safiName, localAs, peerIP, peerData, peerDesc, logger)
		
		processPeerSession(ch, peerData, desc, peerLabels, wg, vrfName, peerIP, afi, safiName, peerDesc, logger)
		
		if *bgpPeerTypes {
			updatePeerTypes(peerTypes, safiName, peerData, peerDesc, vrfName, peerIP)
		}
	}
	return nil
}

func buildPeerLabels(vrfName, afi, safiName, localAs, peerIP string, peerData *bgpPeerSession, peerDesc map[string]bgpVRF, logger *slog.Logger) []string {
	labels := []string{
		strings.ToLower(vrfName),
		strings.ToLower(afi),
		strings.ToLower(safiName[4:]),
		localAs,
		peerIP,
		strconv.FormatUint(uint64(peerData.RemoteAs), 10),
	}

	if *bgpPeerDescs {
		labels = appendPeerDescription(labels, peerDesc, vrfName, peerIP, logger)
	}

	if *bgpPeerHostnames {
		labels = append(labels, peerData.Hostname)
	}

	if *bgpPeerGroups {
		labels = appendPeerGroup(labels, peerDesc, vrfName, peerIP)
	}

	return labels
}

func appendPeerDescription(labels []string, peerDesc map[string]bgpVRF, vrfName, peerIP string, logger *slog.Logger) []string {
	if peerDesc == nil {
		return labels
	}

	desc := peerDesc[vrfName].BGPNeighbors[peerIP].Desc
	if *bgpPeerDescsText {
		return append(labels, desc)
	}

	var jsonDesc struct{ Desc string }
	if err := json.Unmarshal([]byte(desc), &jsonDesc); err != nil {
		logger.Error("cannot unmarshal bgp description", "description", desc, "err", err)
		return labels
	}
	return append(labels, jsonDesc.Desc)
}

func appendPeerGroup(labels []string, peerDesc map[string]bgpVRF, vrfName, peerIP string) []string {
	if peerDesc == nil {
		return labels
	}
	return append(labels, peerDesc[vrfName].BGPNeighbors[peerIP].PeerGroup)
}

func processPeerSession(ch chan<- prometheus.Metric, peerData *bgpPeerSession, desc map[string]*prometheus.Desc, labels []string, wg *sync.WaitGroup, vrfName, peerIP, afi, safiName string, peerDesc map[string]bgpVRF, logger *slog.Logger) {
	handleAdvertisedPrefixes(ch, wg, peerData, vrfName, peerIP, afi, safiName, logger, desc, labels)
	
	newCounter(ch, desc["msgRcvd"], float64(peerData.MsgRcvd), labels...)
	newCounter(ch, desc["msgSent"], float64(peerData.MsgSent), labels...)
	newGauge(ch, desc["UptimeSec"], float64(peerData.PeerUptimeMsec)*0.001, labels...)
	
	prefixReceived := calculatePrefixReceived(peerData)
	newGauge(ch, desc["prefixReceivedCount"], prefixReceived, labels...)
	
	if *bgpAcceptedFilteredPrefixes {
		wg.Add(1)
		go fetchPeerAcceptedFilteredRoutes(ch, wg, afi, safiName, vrfName, peerIP, prefixReceived, logger, desc, labels...)
	}
	
	peerState := determinePeerState(peerData.State)
	newGauge(ch, desc["state"], peerState, labels...)
}

func handleAdvertisedPrefixes(ch chan<- prometheus.Metric, wg *sync.WaitGroup, peerData *bgpPeerSession, vrfName, peerIP, afi, safiName string, logger *slog.Logger, desc map[string]*prometheus.Desc, labels []string) {
	if peerData.PfxSnt != nil {
		newGauge(ch, desc["prefixAdvertisedCount"], float64(*peerData.PfxSnt), labels...)
	} else if *bgpAdvertisedPrefixes {
		wg.Add(1)
		go fetchPeerAdvertisedPrefixes(ch, wg, afi, safiName, vrfName, peerIP, logger, desc, labels...)
	}
}

func calculatePrefixReceived(peerData *bgpPeerSession) float64 {
	if peerData.PrefixReceivedCount != 0 {
		return float64(peerData.PrefixReceivedCount)
	}
	return float64(peerData.PfxRcd)
}

func determinePeerState(state string) float64 {
	switch strings.ToLower(state) {
	case "established":
		return 1
	case "idle (admin)":
		return 2
	default:
		return 0
	}
}

func updatePeerTypes(peerTypes map[string]map[string]float64, safiName string, peerData *bgpPeerSession, peerDesc map[string]bgpVRF, vrfName, peerIP string) {
	if peerData.State != "Established" || peerDesc == nil {
		return
	}

	var descTypes map[string]string
	if err := json.Unmarshal([]byte(peerDesc[vrfName].BGPNeighbors[peerIP].Desc), &descTypes); err != nil {
		return
	}

	safiKey := strings.ToLower(safiName[4:])
	if _, exists := peerTypes[safiKey]; !exists {
		peerTypes[safiKey] = make(map[string]float64)
	}

	for _, key := range *frrBGPDescKey {
		if value := strings.TrimSpace(descTypes[key]); value != "" {
			peerTypes[safiKey][value] += 1
		}
	}
}

func emitPeerTypeMetrics(ch chan<- prometheus.Metric, peerTypes map[string]map[string]float64, afi string, desc map[string]*prometheus.Desc) {
	for safi, types := range peerTypes {
		for peerType, count := range types {
			labels := []string{peerType, strings.ToLower(afi), safi}
			newGauge(ch, desc["peerTypesUp"], count, labels...)
		}
	}
}

func fetchPeerAdvertisedPrefixes(ch chan<- prometheus.Metric, wg *sync.WaitGroup, afi, safi, vrfName, neighbor string, logger *slog.Logger, desc map[string]*prometheus.Desc, labels ...string) {
	defer wg.Done()

	cmd := buildAdvertisedRoutesCommand(afi, safi, vrfName, neighbor)
	output, err := executeBGPCommandFunc(cmd)
	if err != nil {
		logger.Error("get neighbor advertised prefixes failed", "afi", afi, "safi", safi, "vrf", vrfName, "neighbor", neighbor, "err", err)
		return
	}

	advertisedPrefixes, err := parseAdvertisedRoutes(output)
	if err != nil {
		logger.Error("get neighbor advertised prefixes failed", "afi", afi, "safi", safi, "vrf", vrfName, "neighbor", neighbor, "err", err)
		return
	}

	newGauge(ch, desc["prefixAdvertisedCount"], float64(advertisedPrefixes.TotalPrefixCounter), labels...)
}

func buildAdvertisedRoutesCommand(afi, safi, vrfName, neighbor string) string {
	if strings.ToLower(vrfName) == "default" {
		return fmt.Sprintf("show bgp %s %s neighbors %s advertised-routes json", afi, safi, neighbor)
	}
	return fmt.Sprintf("show bgp vrf %s %s %s neighbors %s advertised-routes json", vrfName, afi, safi, neighbor)
}

func parseAdvertisedRoutes(output []byte) (bgpAdvertisedRoutes, error) {
	var routes bgpAdvertisedRoutes
	err := json.Unmarshal(output, &routes)
	return routes, err
}

func fetchPeerAcceptedFilteredRoutes(ch chan<- prometheus.Metric, wg *sync.WaitGroup, afi, safi, vrfName, neighbor string, prefixesReceived float64, logger *slog.Logger, desc map[string]*prometheus.Desc, labels ...string) {
	defer wg.Done()

	cmd := buildRoutesCommand(afi, safi, vrfName, neighbor)
	output, err := executeBGPCommandFunc(cmd)
	if err != nil {
		logger.Error("get neighbor routes failed", "afi", afi, "safi", safi, "vrf", vrfName, "neighbor", neighbor, "err", err)
		return
	}

	routes, err := parseRoutes(output)
	if err != nil {
		logger.Error("get neighbor routes failed", "afi", afi, "safi", safi, "vrf", vrfName, "neighbor", neighbor, "err", err)
		return
	}

	prefixesAccepted := float64(len(routes.Routes))
	newGauge(ch, desc["prefixAcceptedCount"], prefixesAccepted, labels...)
	newGauge(ch, desc["prefixFilteredCount"], prefixesReceived-prefixesAccepted, labels...)
}

func buildRoutesCommand(afi, safi, vrfName, neighbor string) string {
	if strings.ToLower(vrfName) == "default" {
		return fmt.Sprintf("show bgp %s %s neighbors %s routes json", strings.ToLower(afi), strings.ToLower(safi), neighbor)
	}
	return fmt.Sprintf("show bgp vrf %s %s %s neighbors %s routes json", vrfName, strings.ToLower(afi), strings.ToLower(safi), neighbor)
}

func parseRoutes(output []byte) (bgpRoutes, error) {
	var routes bgpRoutes
	err := json.Unmarshal(output, &routes)
	return routes, err
}

type bgpProcess struct {
	RouterID        string
	AS              uint32
	RIBCount        uint32
	RIBMemory       uint32
	PeerCount       uint32
	PeerMemory      uint32
	PeerGroupCount  uint32
	PeerGroupMemory uint32
	Peers           map[string]*bgpPeerSession
}

type bgpPeerSession struct {
	State               string
	RemoteAs            uint32
	MsgRcvd             uint32
	MsgSent             uint32
	PeerUptimeMsec      uint64
	PrefixReceivedCount uint32
	PfxRcd              uint32
	PfxSnt              *uint32
	Hostname            string
}

type bgpAdvertisedRoutes struct {
	TotalPrefixCounter uint32 `json:"totalPrefixCounter"`
}

type bgpRoutes struct {
	Routes map[string][]json.RawMessage `json:"routes"`
}

type vxLanStat struct {
	Vni            uint32
	VxlanType      string `json:"type"`
	VxlanIf        string
	NumMacs        uint32
	NumArpNd       uint32
	NumRemoteVteps interface{}
	TenantVrf      string
}

type bgpVRF struct {
	ID           int
	Name         string
	BGPNeighbors map[string]bgpNeighbor
}

func (vrf *bgpVRF) UnmarshalJSON(data []byte) error {
	var raw map[string]*json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	vrf.BGPNeighbors = make(map[string]bgpNeighbor)

	for key, value := range raw {
		switch key {
		case "vrfId":
			if err := json.Unmarshal(*value, &vrf.ID); err != nil {
				return err
			}
		case "vrfName":
			if err := json.Unmarshal(*value, &vrf.Name); err != nil {
				return err
			}
		default:
			var neighbor bgpNeighbor
			if err := json.Unmarshal(*value, &neighbor); err != nil {
				return err
			}
			vrf.BGPNeighbors[key] = neighbor
		}
	}
	return nil
}

type bgpNeighbor struct {
	Desc      string `json:"nbrDesc"`
	PeerGroup string `json:"peerGroup"`
	AsNumber  int `json:"asNumber,omitempty"`
}
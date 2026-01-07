package metrics

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const bfdSubsystem = "bfd"

func init() {
	registerCollector(bfdSubsystem, enabledByDefault, NewBFDCollector)
}

type BFDPeerConnection struct {
	LocalAddress  string
	RemoteAddress string
	Status        string
	UptimeSeconds uint64
}

type BFDPeerDiagnostics struct {
	LocalDiagnostic  string
	RemoteDiagnostic string
}

type BFDPeerTimers struct {
	ReceiveInterval        uint32
	TransmitInterval       uint32
	EchoInterval           uint32
	RemoteReceiveInterval  uint32
	RemoteTransmitInterval uint32
	RemoteEchoInterval     uint32
}

type BFDPeer struct {
	Multihop   bool
	PeerConfig struct {
		Vrf      string
		LocalID  uint32
		RemoteID uint32
	}
	Connection   BFDPeerConnection
	Diagnostics  BFDPeerDiagnostics
	TimerConfig  BFDPeerTimers
}

type BFDPeerCollection struct {
	Peers []BFDPeer
}

func NewBFDPeerCollection() *BFDPeerCollection {
	return &BFDPeerCollection{
		Peers: make([]BFDPeer, 0),
	}
}

func (c *BFDPeerCollection) AddPeer(peer BFDPeer) {
	c.Peers = append(c.Peers, peer)
}

func (c *BFDPeerCollection) Count() int {
	return len(c.Peers)
}

type BFDCommandExecutor interface {
	ExecuteBFDCommand(cmd string) ([]byte, error)
}

type DefaultBFDExecutor struct{}

func (e *DefaultBFDExecutor) ExecuteBFDCommand(cmd string) ([]byte, error) {
	return executeBFDCommand(cmd)
}

type BFDCollectorState struct {
	LastCollectionTime time.Time
	CollectionCount    uint64
}

type bfdCollector struct {
	logger       *slog.Logger
	descriptions map[string]*prometheus.Desc
	executor     BFDCommandExecutor
	state        BFDCollectorState
}

func NewBFDCollector(logger *slog.Logger) (Collector, error) {
	return &bfdCollector{
		logger:       logger,
		descriptions: getBFDDesc(),
		executor:     &DefaultBFDExecutor{},
		state:        BFDCollectorState{},
	}, nil
}

func getBFDDesc() map[string]*prometheus.Desc {
	countLabels := []string{}
	peerLabels := []string{"local", "peer"}
	
	return map[string]*prometheus.Desc{
		"bfdPeerCount":  colPromDesc(bfdSubsystem, "peer_count", "Number of peers detected.", countLabels),
		"bfdPeerUptime": colPromDesc(bfdSubsystem, "peer_uptime", "Uptime of bfd peer in seconds", peerLabels),
		"bfdPeerState":  colPromDesc(bfdSubsystem, "peer_state", "State of the bfd peer (1 = Up, 0 = Down).", peerLabels),
	}
}

func (c *bfdCollector) Update(ch chan<- prometheus.Metric) error {
	startTime := time.Now()
	defer func() {
		c.state.LastCollectionTime = time.Now()
		c.state.CollectionCount++
		c.logger.Debug("BFD collection completed", 
			"duration", time.Since(startTime), 
			"total_collections", c.state.CollectionCount)
	}()

	c.logger.Info("Starting BFD metrics collection")

	rawData, err := c.fetchBFDRawData()
	if err != nil {
		return fmt.Errorf("failed to fetch BFD data: %w", err)
	}

	if err := c.processBFDPeers(ch, rawData); err != nil {
		return fmt.Errorf("failed to process BFD peers: %w", err)
	}
	
	c.logger.Info("BFD metrics collection completed successfully", 
		"peers_processed", len(rawData))
	return nil
}

func (c *bfdCollector) fetchBFDRawData() ([]bfdPeer, error) {
	const cmd = "show bfd peers json"
	
	c.logger.Debug("Executing BFD command", "command", cmd)
	
	start := time.Now()
	jsonData, err := c.executor.ExecuteBFDCommand(cmd)
	if err != nil {
		c.logger.Error("BFD command execution failed", "command", cmd, "error", err)
		return nil, fmt.Errorf("executeBFDCommand failed: %w", err)
	}
	
	c.logger.Debug("BFD command executed", 
		"command", cmd, 
		"duration", time.Since(start),
		"response_size", len(jsonData))
	
	c.logger.Debug("Raw BFD response", "data", string(jsonData))
	
	return c.parseBFDResponse(jsonData)
}

func (c *bfdCollector) parseBFDResponse(jsonData []byte) ([]bfdPeer, error) {
	if len(jsonData) == 0 {
		c.logger.Warn("Empty response received from BFD command")
		return nil, nil
	}
	
	var peers []bfdPeer
	if err := json.Unmarshal(jsonData, &peers); err != nil {
		c.logger.Error("Failed to parse BFD response", "error", err)
		return nil, fmt.Errorf("json unmarshal failed: %w", err)
	}
	
	c.logger.Info("BFD response parsed", "peer_count", len(peers))
	return peers, nil
}

func (c *bfdCollector) processBFDPeers(ch chan<- prometheus.Metric, peers []bfdPeer) error {
	if peers == nil {
		c.logger.Info("No BFD peers found")
		return nil
	}
	
	collection := NewBFDPeerCollection()
	
	for _, p := range peers {
		peer := c.convertToStructuredPeer(p)
		collection.AddPeer(peer)
		c.processSinglePeer(ch, peer)
	}

	newGauge(ch, c.descriptions["bfdPeerCount"], float64(collection.Count()))
	return nil
}

func (c *bfdCollector) convertToStructuredPeer(p bfdPeer) BFDPeer {
	peer := BFDPeer{
		Multihop: p.Multihop,
	}
	
	peer.PeerConfig.Vrf = p.Vrf
	peer.PeerConfig.LocalID = p.ID
	peer.PeerConfig.RemoteID = p.RemoteID
	
	peer.Connection.LocalAddress = p.Local
	peer.Connection.RemoteAddress = p.Peer
	peer.Connection.Status = p.Status
	peer.Connection.UptimeSeconds = p.Uptime
	
	peer.Diagnostics.LocalDiagnostic = p.Diagnostic
	peer.Diagnostics.RemoteDiagnostic = p.RemoteDiagnostic
	
	peer.TimerConfig.ReceiveInterval = p.ReceiveInterval
	peer.TimerConfig.TransmitInterval = p.TransmitInterval
	peer.TimerConfig.EchoInterval = p.EchoInterval
	peer.TimerConfig.RemoteReceiveInterval = p.RemoteReceiveInterval
	peer.TimerConfig.RemoteTransmitInterval = p.RemoteTransmitInterval
	peer.TimerConfig.RemoteEchoInterval = p.RemoteEchoInterval
	
	return peer
}

func (c *bfdCollector) processSinglePeer(ch chan<- prometheus.Metric, peer BFDPeer) {
	labels := []string{peer.Connection.LocalAddress, peer.Connection.RemoteAddress}

	c.logger.Debug("Adding peer uptime metric",
		"local", peer.Connection.LocalAddress,
		"peer", peer.Connection.RemoteAddress,
		"uptime", peer.Connection.UptimeSeconds)
	newGauge(ch, c.descriptions["bfdPeerUptime"], float64(peer.Connection.UptimeSeconds), labels...)
	
	// 添加对等体状态指标
	var stateValue float64
	if peer.Connection.Status == "up" {
		stateValue = 1
	}
	
	c.logger.Debug("Adding peer state metric",
		"local", peer.Connection.LocalAddress,
		"peer", peer.Connection.RemoteAddress,
		"status", peer.Connection.Status,
		"state_value", stateValue)
	newGauge(ch, c.descriptions["bfdPeerState"], stateValue, labels...)
}

// bfdPeer 保持与原始JSON结构的兼容性
type bfdPeer struct {
	Multihop               bool   `json:"multihop"`
	Peer                   string `json:"peer"`
	Local                  string `json:"local"`
	Vrf                    string `json:"vrf"`
	ID                     uint32 `json:"id"`
	RemoteID               uint32 `json:"remote-id"`
	Status                 string `json:"status"`
	Uptime                 uint64 `json:"uptime"`
	Diagnostic             string `json:"diagnostic"`
	RemoteDiagnostic       string `json:"remote-diagnostic"`
	ReceiveInterval        uint32 `json:"receive-interval"`
	TransmitInterval       uint32 `json:"transmit-interval"`
	EchoInterval           uint32 `json:"echo-interval"`
	RemoteReceiveInterval  uint32 `json:"remote-receive-interval"`
	RemoteTransmitInterval uint32 `json:"remote-transmit-interval"`
	RemoteEchoInterval     uint32 `json:"remote-echo-interval"`
}
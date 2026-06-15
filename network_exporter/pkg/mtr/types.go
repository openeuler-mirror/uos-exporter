package mtr

import (
	"time"
	"network_exporter/pkg/common"
)

const defaultMaxHops = 30
const defaultTimeout = 5 * time.Second
const defaultPackerSize = 56
const defaultCount = 10

// MtrResult Calculated results
type MtrResult struct {
	DestAddr      string                         `json:"dest_address"`
	Hops          []common.IcmpHop               `json:"hops"`
	HopSummaryMap map[string]*common.IcmpSummary `json:"hop_summary_map"`
}

// MtrReturn MTR Response
type MtrReturn struct {
	success   bool
	ttl       int
	host      string
	succSum   int
	lastTime  time.Duration
	allTime   []time.Duration
	sumTime   time.Duration
	bestTime  time.Duration
	avgTime   time.Duration
	worstTime time.Duration
}

// MtrOptions MTR Options
type MtrOptions struct {

// TODO: implement functions

package ping

import "time"

const defaultTimeout = 5 * time.Second
const defaultPackerSize = 56
const defaultCount = 10
const defaultTTL = 128

// PingResult Calculated results
type PingResult struct {
	Success              bool          `json:"success"`
	DestAddr             string        `json:"dest_address"`
	DestIp               string        `json:"dest_ip"`
	DropRate             float64       `json:"drop_rate"`
	SumTime              time.Duration `json:"sum"`
	BestTime             time.Duration `json:"best"`
	AvgTime              time.Duration `json:"avg"`
	WorstTime            time.Duration `json:"worst"`
	SquaredDeviationTime time.Duration `json:"sd"`
	UncorrectedSDTime    time.Duration `json:"usd"`
	CorrectedSDTime      time.Duration `json:"csd"`
	RangeTime            time.Duration `json:"range"`
	SntSummary           int           `json:"snt_summary"`
	SntFailSummary       int           `json:"snt_fail_summary"`
	SntTimeSummary       time.Duration `json:"snt_time_summary"`
}

// PingReturn ICMP Response
type PingReturn struct {
	success   bool
	succSum   int
	allTime   []time.Duration
	sumTime   time.Duration
	bestTime  time.Duration
	avgTime   time.Duration
	worstTime time.Duration
}

// PingOptions ICMP Options
type PingOptions struct {

// TODO: implement functions

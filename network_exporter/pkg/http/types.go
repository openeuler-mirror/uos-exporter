package http

import (
	"sync"
	"time"
)

// HTTPReturn Calculated results
type HTTPReturn struct {
	Success               bool          `json:"success"`
	DestAddr              string        `json:"dest_address"`
	Status                int           `json:"status,omitempty"`
	ContentLength         int64         `json:"content_length,omitempty"`
	DNSLookup             time.Duration `json:"dnsLookup,omitempty"`
	TCPConnection         time.Duration `json:"tcpConnection,omitempty"`
	TLSHandshake          time.Duration `json:"tlsHandshake,omitempty"`
	TLSVersion            string        `json:"tlsVersion,omitempty"`
	TLSEarliestCertExpiry time.Time     `json:"tlsEarliestCertExpiry,omitempty"`
	TLSLastChainExpiry    time.Time     `json:"tlsLastChainExpiry,omitempty"`
	ServerProcessing      time.Duration `json:"serverProcessing,omitempty"`
	ContentTransfer       time.Duration `json:"contentTransfer,omitempty"`
	Total                 time.Duration `json:"total,omitempty"`
}

// HTTPTimelineStats http timeline stats
type HTTPTimelineStats struct {
	DNSLookup        time.Duration `json:"dnsLookup,omitempty"`
	TCPConnection    time.Duration `json:"tcpConnection,omitempty"`
	TLSHandshake     time.Duration `json:"tlsHandshake,omitempty"`
	ServerProcessing time.Duration `json:"serverProcessing,omitempty"`
	ContentTransfer  time.Duration `json:"contentTransfer,omitempty"`
	Total            time.Duration `json:"total,omitempty"`
}

// HTTPTrace http trace
type HTTPTrace struct {

// TODO: implement functions

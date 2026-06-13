package corosync

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// Parser defines the interface for parsing corosync tool outputs
type Parser interface {
	Parse(cfgToolOutput []byte, quorumToolOutput []byte) (*Status, error)
}

// Status contains all parsed corosync cluster status information
type Status struct {
	NodeId      string
	RingId      string
	Rings       []Ring
	QuorumVotes QuorumVotes
	Quorate     bool
	Members     []Member
}

// QuorumVotes contains vote-related quorum information
type QuorumVotes struct {
	ExpectedVotes   uint64
	HighestExpected uint64
	TotalVotes      uint64
	Quorum          uint64
}

// Ring represents a corosync communication ring
type Ring struct {
	Number  string
	Address string
	Faulty  bool
}

// Member represents a cluster member node
type Member struct {
	Id      string
	Name    string
	Qdevice string
	Votes   uint64
	Local   bool
}

// NewParser creates a new default parser instance

// TODO: implement functions

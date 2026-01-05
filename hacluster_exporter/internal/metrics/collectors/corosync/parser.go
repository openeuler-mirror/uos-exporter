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
func NewParser() Parser {
	return &defaultParser{
		patterns: map[string]*regexp.Regexp{
			"nodeId":         regexp.MustCompile(`(?m)Node ID:\s+(\w+)`),
			"ringId":         regexp.MustCompile(`(?m)Ring ID:\s+\b(.+)\b`),
			"quorate":        regexp.MustCompile(`(?m)Quorate:\s+(Yes|No)`),
			"quorumVotes":    regexp.MustCompile(`(?m)Expected votes:\s+(\d+)\s+Highest expected:\s+(\d+)\s+Total votes:\s+(\d+)\s+Quorum:\s+(\d+)`),
			"membersSection": regexp.MustCompile(`(?m)Membership information\n-+\s+Nodeid\s+Votes\s+Qdevice\s+Name\n+((?:.*\n?)+)`),
			"memberLine":     regexp.MustCompile(`(?m)(?P<node_id>\w+)\s+(?P<votes>\S+)\s+(?P<qdevice>(\w,?)+)?\s+(?P<name>[^\s]+)(?:\s(?P<local>\(local\)))?\n?`),
			"ringInfo":       regexp.MustCompile(`(?m)(?P<prefix>RING|Link) ID (?P<number>\d+)\s+(?P<id>id|addr)\s+= (?P<address>.+)\s+status\s+= (?P<status>.+)`),
		},
	}
}

type defaultParser struct {
	patterns map[string]*regexp.Regexp
}

func (p *defaultParser) Parse(cfgToolOutput []byte, quorumToolOutput []byte) (*Status, error) {
	status := &Status{}
	var err error

	if status.NodeId, err = p.parseNodeId(quorumToolOutput); err != nil {
		return nil, errors.Wrap(err, "node id parsing failed")
	}

	if status.RingId, err = p.parseRingId(quorumToolOutput); err != nil {
		return nil, errors.Wrap(err, "ring id parsing failed")
	}

	if status.Quorate, err = p.parseQuorate(quorumToolOutput); err != nil {
		return nil, errors.Wrap(err, "quorate status parsing failed")
	}

	if status.QuorumVotes, err = p.parseQuoromVotes(quorumToolOutput); err != nil {
		return nil, errors.Wrap(err, "quorum votes parsing failed")
	}

	if status.Members, err = p.parseMembers(quorumToolOutput); err != nil {
		return nil, errors.Wrap(err, "member list parsing failed")
	}

	status.Rings = p.parseRings(cfgToolOutput)

	return status, nil
}

// Node ID parsing
func (p *defaultParser) parseNodeId(output []byte) (string, error) {
	matches := p.patterns["nodeId"].FindSubmatch(output)
	if matches == nil {
		return "", errors.New("node ID pattern not found")
	}
	return string(matches[1]), nil
}

// Ring ID parsing (handles different corosync version formats)
func (p *defaultParser) parseRingId(output []byte) (string, error) {
	matches := p.patterns["ringId"].FindSubmatch(output)
	if matches == nil {
		return "", errors.New("ring ID pattern not found")
	}
	return string(matches[1]), nil
}

// Quorate status parsing
func (p *defaultParser) parseQuorate(output []byte) (bool, error) {
	matches := p.patterns["quorate"].FindSubmatch(output)
	if matches == nil {
		return false, errors.New("quorate status pattern not found")
	}
	return string(matches[1]) == "Yes", nil
}

// Ring information parsing
func (p *defaultParser) parseRings(output []byte) []Ring {
	matches := p.patterns["ringInfo"].FindAllSubmatch(output, -1)
	rings := make([]Ring, len(matches))

	for i, match := range matches {
		namedMatches := p.extractNamedGroups(p.patterns["ringInfo"], match)
		rings[i] = Ring{
			Number:  namedMatches["number"],
			Address: namedMatches["address"],
			Faulty:  strings.Contains(namedMatches["status"], "FAULTY"),
		}
	}
	return rings
}

// Quorum votes parsing
func (p *defaultParser) parseQuoromVotes(output []byte) (QuorumVotes, error) {
	var qv QuorumVotes
	matches := p.patterns["quorumVotes"].FindSubmatch(output)
	if matches == nil {
		return qv, errors.New("quorum votes pattern not found")
	}

	var err error
	values := []*uint64{
		&qv.ExpectedVotes,
		&qv.HighestExpected,
		&qv.TotalVotes,
		&qv.Quorum,
	}

	for i := 1; i <= 4; i++ {
		if len(matches) <= i {
			return qv, errors.New("insufficient quorum votes matches")
		}
		*values[i-1], err = strconv.ParseUint(string(matches[i]), 10, 64)
		if err != nil {
			return qv, errors.Wrapf(err, "failed to parse vote value: %s", matches[i])
		}
	}

	return qv, nil
}

// Member list parsing
func (p *defaultParser) parseMembers(output []byte) ([]Member, error) {
	var members []Member

	sectionMatch := p.patterns["membersSection"].FindSubmatch(output)
	if sectionMatch == nil {
		return nil, errors.New("membership section not found")
	}

	linesMatches := p.patterns["memberLine"].FindAllSubmatch(sectionMatch[1], -1)
	for _, match := range linesMatches {
		member, err := p.parseMemberLine(match)
		if err != nil {
			return nil, errors.Wrap(err, "member line parsing failed")
		}
		members = append(members, member)
	}

	return members, nil
}

func (p *defaultParser) parseMemberLine(match [][]byte) (Member, error) {
	var m Member
	var err error

	namedMatches := p.extractNamedGroups(p.patterns["memberLine"], match)

	m.Id = namedMatches["node_id"]
	m.Name = namedMatches["name"]
	m.Qdevice = namedMatches["qdevice"]
	m.Local = namedMatches["local"] != ""

	m.Votes, err = strconv.ParseUint(namedMatches["votes"], 10, 64)
	if err != nil {
		return m, errors.Wrapf(err, "invalid votes value: %s", namedMatches["votes"])
	}

	return m, nil
}

// Helper to extract named capture groups from regex matches
func (p *defaultParser) extractNamedGroups(re *regexp.Regexp, match [][]byte) map[string]string {
	groups := make(map[string]string)
	for i, name := range re.SubexpNames() {
		if i != 0 && name != "" {
			groups[name] = string(match[i])
		}
	}
	return groups
}

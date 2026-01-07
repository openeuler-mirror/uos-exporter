package metrics

import (
	"time"
)

// Client queries the BIND API, parses the response and returns stats in a
// generic format.
type Client interface {
	Stats(...StatisticGroup) (Statistics, error)
}

const (
	// QryRTT is the common prefix of query round-trip histogram counters.
	QryRTT = "QryRTT"
)

// StatisticGroup describes a sub-group of BIND statistics.
type StatisticGroup string

// Available statistic groups.
const (
	ServerStats StatisticGroup = "server"
	ViewStats   StatisticGroup = "view"
	TaskStats   StatisticGroup = "tasks"
)

// Statistics is a generic representation of BIND statistics.
type Statistics struct {
	Server      Server
	Views       []View
	ZoneViews   []ZoneView
	TaskManager TaskManager
}

// Server represents BIND server statistics.
type Server struct {
	BootTime         time.Time
	ConfigTime       time.Time
	IncomingQueries  []Counter
	IncomingRequests []Counter
	NameServerStats  []Counter
	ZoneStatistics   []Counter
	ServerRcodes     []Counter
}

// View represents statistics for a single BIND view.
type View struct {
	Name            string
	Cache           []Gauge
	ResolverStats   []Counter
	ResolverQueries []Counter
}

// View represents statistics for a single BIND zone view.
type ZoneView struct {
	Name     string
	ZoneData []ZoneCounter
}

// Counter represents a single counter value.
type Counter struct {
	Name    string `xml:"name,attr"`
	Counter uint64 `xml:",chardata"`
}

// Counter represents a single zone counter value.
type ZoneCounter struct {
	Name   string
	Serial string
}

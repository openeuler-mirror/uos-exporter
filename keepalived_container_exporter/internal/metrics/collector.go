package metrics

import (
	"keepalived_container_exporter/pkg/utils"
	"strconv"

	"bytes"
	"errors"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/cenkalti/backoff/v4"
	"github.com/sirupsen/logrus"
)

type Collector interface {
	Refresh() error
	ScriptVrrps() ([]VRRPScript, error)
	DataVrrps() (map[string]*VRRPData, error)
	StatsVrrps() (map[string]*VRRPStats, error)
	JSONVrrps() ([]VRRP, error)
	HasVRRPScriptStateSupport() bool
}

// KeepalivedCollector implements prometheus.Collector interface and stores required info to collect data.
type KeepalivedCollector struct {
	sync.Mutex
	useJSON    bool
	scriptPath string
	metrics    map[string]*prometheus.Desc
	collector  Collector
}

// VRRPStats represents Keepalived stats about VRRP.
type VRRPStats struct {
	AdvertRcvd        int `json:"advert_rcvd"`
	AdvertSent        int `json:"advert_sent"`
	BecomeMaster      int `json:"become_master"`
	ReleaseMaster     int `json:"release_master"`
	PacketLenErr      int `json:"packet_len_err"`
	AdvertIntervalErr int `json:"advert_interval_err"`
	IPTTLErr          int `json:"ip_ttl_err"`
	InvalidTypeRcvd   int `json:"invalid_type_rcvd"`
	AddrListErr       int `json:"addr_list_err"`
	InvalidAuthType   int `json:"invalid_authtype"`
	AuthTypeMismatch  int `json:"authtype_mismatch"`
	AuthFailure       int `json:"auth_failure"`
	PRIZeroRcvd       int `json:"pri_zero_rcvd"`
	PRIZeroSent       int `json:"pri_zero_sent"`
}

// VRRPData represents Keepalived data about VRRP.
type VRRPData struct {
	IName        string   `json:"iname"`
	State        int      `json:"state"`
	WantState    int      `json:"wantstate"`
	Intf         string   `json:"ifp_ifname"`
	GArpDelay    int      `json:"garp_delay"`
	VRID         int      `json:"vrid"`
	VIPs         []string `json:"vips"`
	ExcludedVIPs []string `json:"evips"`
}

// VRRPScript represents Keepalived script about VRRP.
type VRRPScript struct {
	Name   string
	Status string
	State  string
}

// VRRP ties together VRRPData and VRRPStats.
type VRRP struct {
	Data  VRRPData  `json:"data"`
	Stats VRRPStats `json:"stats"`
}

// KeepalivedStats ties together VRRP and VRRPScript.
type KeepalivedStats struct {
	VRRPs   []VRRP
	Scripts []VRRPScript
}

// NewKeepalivedCollector is creating new instance of KeepalivedCollector.

// TODO: implement functions

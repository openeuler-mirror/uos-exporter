package drbd

import (
	"encoding/json"
	"hacluster_exporter/internal/metrics/collectors/core"
	"hacluster_exporter/pkg/utils"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

const subsystem = "drbd"

// drbdStatus is for parsing relevant data we want to convert to metrics
type drbdStatus struct {
	Name    string `json:"name"`
	Role    string `json:"role"`
	Devices []struct {
		Volume    int    `json:"volume"`
		Written   int    `json:"written"`
		Read      int    `json:"read"`
		AlWrites  int    `json:"al-writes"`
		BmWrites  int    `json:"bm-writes"`
		UpPending int    `json:"upper-pending"`
		LoPending int    `json:"lower-pending"`
		Quorum    bool   `json:"quorum"`
		DiskState string `json:"disk-state"`
	} `json:"devices"`
	Connections []struct {
		PeerNodeID  int    `json:"peer-node-id"`
		PeerRole    string `json:"peer-role"`
		PeerDevices []struct {
			Volume        int     `json:"volume"`
			Received      int     `json:"received"`
			Sent          int     `json:"sent"`
			Pending       int     `json:"pending"`
			Unacked       int     `json:"unacked"`
			PeerDiskState string  `json:"peer-disk-state"`
			PercentInSync float64 `json:"percent-in-sync"`
		} `json:"peer_devices"`
	} `json:"connections"`
}

type drbdMetrics struct {
	resourcesDesc          *prometheus.Desc
	writtenDesc            *prometheus.Desc
	readDesc               *prometheus.Desc
	alWritesDesc           *prometheus.Desc
	bmWritesDesc           *prometheus.Desc
	upperPendingDesc       *prometheus.Desc
	lowerPendingDesc       *prometheus.Desc
	quorumDesc             *prometheus.Desc
	connectionsDesc        *prometheus.Desc
	connectionsSyncDesc    *prometheus.Desc
	connectionsRecvDesc    *prometheus.Desc
	connectionsSentDesc    *prometheus.Desc
	connectionsPendingDesc *prometheus.Desc
	connectionsUnackedDesc *prometheus.Desc
	splitBrainDesc         *prometheus.Desc
}


// TODO: implement functions

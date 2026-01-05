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

func NewCollector(drbdSetupPath string, drbdSplitBrainPath string, timestamps bool) (*DrbdCollector, error) {
	err := core.CheckExecutables(drbdSetupPath)
	if err != nil {
		return nil, errors.Wrapf(err, "could not initialize '%s' collector", subsystem)
	}

	c := &DrbdCollector{
		core.NewDefaultCollector(subsystem, timestamps),
		drbdSetupPath,
		drbdSplitBrainPath,
		drbdMetrics{
			resourcesDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "resources"),
				"The DRBD resources; 1 line per name, per volume",
				[]string{"resource", "role", "volume", "disk_state"},
				nil,
			),
			writtenDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "written"),
				"KiB written to DRBD; 1 line per res, per volume",
				[]string{"resource", "volume"},
				nil,
			),
			readDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "read"),
				"KiB read from DRBD; 1 line per res, per volume",
				[]string{"resource", "volume"},
				nil,
			),
			alWritesDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "al_writes"),
				"Writes to activity log; 1 line per res, per volume",
				[]string{"resource", "volume"},
				nil,
			),
			bmWritesDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "bm_writes"),
				"Writes to bitmap; 1 line per res, per volume",
				[]string{"resource", "volume"},
				nil,
			),
			upperPendingDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "upper_pending"),
				"Upper pending; 1 line per res, per volume",
				[]string{"resource", "volume"},
				nil,
			),
			lowerPendingDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "lower_pending"),
				"Lower pending; 1 line per res, per volume",
				[]string{"resource", "volume"},
				nil,
			),
			quorumDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "quorum"),
				"Quorum status per resource and per volume",
				[]string{"resource", "volume"},
				nil,
			),
			connectionsDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "connections"),
				"The DRBD resource connections; 1 line per per resource, per peer_node_id",
				[]string{"resource", "peer_node_id", "peer_role", "volume", "peer_disk_state"},
				nil,
			),
			connectionsSyncDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "connections_sync"),
				"The in sync percentage value for DRBD resource connections",
				[]string{"resource", "peer_node_id", "volume"},
				nil,
			),
			connectionsRecvDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "connections_received"),
				"KiB received per connection",
				[]string{"resource", "peer_node_id", "volume"},
				nil,
			),
			connectionsSentDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "connections_sent"),
				"KiB sent per connection",
				[]string{"resource", "peer_node_id", "volume"},
				nil,
			),
			connectionsPendingDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "connections_pending"),
				"Pending bytes per connection",
				[]string{"resource", "peer_node_id", "volume"},
				nil,
			),
			connectionsUnackedDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "connections_unacked"),
				"Unacked bytes per connection",
				[]string{"resource", "peer_node_id", "volume"},
				nil,
			),
			splitBrainDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "split_brain"),
				"Split brain status per resource",
				[]string{"resource"},
				nil,
			),
		},
	}

	return c, nil
}

type DrbdCollector struct {
	core.DefaultCollector
	drbdsetupPath      string
	drbdSplitBrainPath string
	metrics            drbdMetrics
}

func (c *DrbdCollector) CollectWithError(ch chan<- prometheus.Metric) error {
	logrus.Debug("Starting DRBD metrics collection...")

	if err := c.collectSplitBrain(ch); err != nil {
		logrus.WithError(err).Error("failed to collect split brain information")
		return errors.Wrap(err, "split brain collection failed")
	}

	output, err := utils.RunCommand(c.drbdsetupPath, "status", "--json")
	if err != nil {
		logrus.WithError(err).Error("drbdsetup command failed")
		return errors.Wrap(err, "drbdsetup command failed")
	}

	status, err := parseDrbdStatus(output)
	if err != nil {
		logrus.WithError(err).Error("failed to parse DRBD status")
		return errors.Wrap(err, "status parsing failed")
	}

	c.collectResources(status, ch)
	c.collectConnections(status, ch)

	return nil

}

func (c *DrbdCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.metrics.resourcesDesc
	ch <- c.metrics.writtenDesc
	ch <- c.metrics.readDesc
	ch <- c.metrics.alWritesDesc
	ch <- c.metrics.bmWritesDesc
	ch <- c.metrics.upperPendingDesc
	ch <- c.metrics.lowerPendingDesc

	ch <- c.metrics.quorumDesc

	ch <- c.metrics.connectionsDesc
	ch <- c.metrics.connectionsSyncDesc
	ch <- c.metrics.connectionsRecvDesc
	ch <- c.metrics.connectionsSentDesc
	ch <- c.metrics.connectionsPendingDesc
	ch <- c.metrics.connectionsUnackedDesc
	ch <- c.metrics.splitBrainDesc
}

func (c *DrbdCollector) Collect(ch chan<- prometheus.Metric) {
	logrus.Debug("Collecting DRBD metrics...")

	err := c.CollectWithError(ch)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"subsystem": c.GetSubsystem(),
			"error":     err,
		}).Warn("collector scrape failed")
	}
}

func (c *DrbdCollector) collectResources(status []drbdStatus, ch chan<- prometheus.Metric) {
	for _, resource := range status {
		for _, device := range resource.Devices {
			volume := strconv.Itoa(device.Volume)
			diskState := strings.ToLower(device.DiskState)

			ch <- prometheus.MustNewConstMetric(
				c.metrics.resourcesDesc,
				prometheus.GaugeValue,
				1,
				resource.Name,
				resource.Role,
				volume,
				diskState,
			)

			ch <- prometheus.MustNewConstMetric(
				c.metrics.writtenDesc,
				prometheus.GaugeValue,
				float64(device.Written),
				resource.Name,
				volume,
			)

			ch <- prometheus.MustNewConstMetric(
				c.metrics.readDesc,
				prometheus.GaugeValue,
				float64(device.Read),
				resource.Name,
				volume,
			)

			ch <- prometheus.MustNewConstMetric(
				c.metrics.alWritesDesc,
				prometheus.GaugeValue,
				float64(device.AlWrites),
				resource.Name,
				volume,
			)

			ch <- prometheus.MustNewConstMetric(
				c.metrics.bmWritesDesc,
				prometheus.GaugeValue,
				float64(device.BmWrites),
				resource.Name,
				volume,
			)

			ch <- prometheus.MustNewConstMetric(
				c.metrics.upperPendingDesc,
				prometheus.GaugeValue,
				float64(device.UpPending),
				resource.Name,
				volume,
			)

			ch <- prometheus.MustNewConstMetric(
				c.metrics.lowerPendingDesc,
				prometheus.GaugeValue,
				float64(device.LoPending),
				resource.Name,
				volume,
			)

			var quorumValue float64
			if device.Quorum {
				quorumValue = 1
			}
			ch <- prometheus.MustNewConstMetric(
				c.metrics.quorumDesc,
				prometheus.GaugeValue,
				quorumValue,
				resource.Name,
				volume,
			)
		}
	}
}

func (c *DrbdCollector) collectConnections(status []drbdStatus, ch chan<- prometheus.Metric) {
	for _, resource := range status {
		if len(resource.Connections) == 0 {
			logrus.WithFields(logrus.Fields{
				"resource": resource.Name,
			}).Debug("no connections found for resource")
			continue
		}

		for _, conn := range resource.Connections {
			peerNodeID := strconv.Itoa(conn.PeerNodeID)

			for _, peerDev := range conn.PeerDevices {
				volume := strconv.Itoa(peerDev.Volume)
				peerDiskState := strings.ToLower(peerDev.PeerDiskState)

				ch <- prometheus.MustNewConstMetric(
					c.metrics.connectionsDesc,
					prometheus.GaugeValue,
					1,
					resource.Name,
					peerNodeID,
					conn.PeerRole,
					volume,
					peerDiskState,
				)

				ch <- prometheus.MustNewConstMetric(
					c.metrics.connectionsSyncDesc,
					prometheus.GaugeValue,
					peerDev.PercentInSync,
					resource.Name,
					peerNodeID,
					volume,
				)

				ch <- prometheus.MustNewConstMetric(
					c.metrics.connectionsRecvDesc,
					prometheus.GaugeValue,
					float64(peerDev.Received),
					resource.Name,
					peerNodeID,
					volume,
				)

				ch <- prometheus.MustNewConstMetric(
					c.metrics.connectionsSentDesc,
					prometheus.GaugeValue,
					float64(peerDev.Sent),
					resource.Name,
					peerNodeID,
					volume,
				)

				ch <- prometheus.MustNewConstMetric(
					c.metrics.connectionsPendingDesc,
					prometheus.GaugeValue,
					float64(peerDev.Pending),
					resource.Name,
					peerNodeID,
					volume,
				)

				ch <- prometheus.MustNewConstMetric(
					c.metrics.connectionsUnackedDesc,
					prometheus.GaugeValue,
					float64(peerDev.Unacked),
					resource.Name,
					peerNodeID,
					volume,
				)
			}
		}
	}
}

func (c *DrbdCollector) collectSplitBrain(ch chan<- prometheus.Metric) error {
	logrus.Debug("Checking for DRBD split brain files...")

	files, _ := filepath.Glob(c.drbdSplitBrainPath + "/drbd-split-brain-detected-*")
	re := regexp.MustCompile(`drbd-split-brain-detected-(?P<resource>[\w-]+)-(?P<volume>[\w-]+)`)

	for _, f := range files {
		matches := re.FindStringSubmatch(f)
		if matches == nil {
			logrus.WithFields(logrus.Fields{
				"file": f,
			}).Warn("split brain file name did not match expected pattern")
			continue
		}

		ch <- prometheus.MustNewConstMetric(
			c.metrics.splitBrainDesc,
			prometheus.GaugeValue,
			1,
			matches[1],
		)
	}
	return nil
}

func parseDrbdStatus(statusRaw []byte) ([]drbdStatus, error) {
	var drbdDevs []drbdStatus
	err := json.Unmarshal(statusRaw, &drbdDevs)
	if err != nil {
		return drbdDevs, err
	}
	return drbdDevs, nil
}

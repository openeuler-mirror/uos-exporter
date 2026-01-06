package metrics

import (
	"errors"
	"node_hardware_exporter/internal/exporter"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
	"github.com/prometheus/procfs/sysfs"
)

func init() {
	exporter.Register(NewMdadmCollector())
}

type MdadmCollector struct {
	*baseMetrics
	activeDesc              *prometheus.Desc
	inActiveDesc            *prometheus.Desc
	recoveringDesc          *prometheus.Desc
	resyncDesc              *prometheus.Desc
	checkDesc               *prometheus.Desc
	disksDesc               *prometheus.Desc
	disksTotalDesc          *prometheus.Desc
	blocksTotalDesc         *prometheus.Desc
	blocksSyncedDesc        *prometheus.Desc
	mdraidDisks             *prometheus.Desc
	mdraidDegradedDisksDesc *prometheus.Desc
}

func NewMdadmCollector() *MdadmCollector {
	return &MdadmCollector{
		baseMetrics: NewMetrics("node_mdadm_collector", "MDADM RAID collector metrics", []string{}),
		activeDesc: prometheus.NewDesc(
			"node_md_state",
			"Indicates the state of md-device.",
			[]string{"device"},
			prometheus.Labels{"state": "active"},
		),
		inActiveDesc: prometheus.NewDesc(
			"node_md_state",
			"Indicates the state of md-device.",
			[]string{"device"},
			prometheus.Labels{"state": "inactive"},
		),
		recoveringDesc: prometheus.NewDesc(
			"node_md_state",
			"Indicates the state of md-device.",
			[]string{"device"},
			prometheus.Labels{"state": "recovering"},
		),
		resyncDesc: prometheus.NewDesc(
			"node_md_state",
			"Indicates the state of md-device.",
			[]string{"device"},
			prometheus.Labels{"state": "resync"},
		),
		checkDesc: prometheus.NewDesc(
			"node_md_state",
			"Indicates the state of md-device.",
			[]string{"device"},
			prometheus.Labels{"state": "check"},
		),
		disksDesc: prometheus.NewDesc(
			"node_md_disks",
			"Number of active/failed/spare disks of device.",
			[]string{"device", "state"},
			nil,
		),
		disksTotalDesc: prometheus.NewDesc(
			"node_md_disks_required",
			"Total number of disks of device.",
			[]string{"device"},
			nil,
		),
		blocksTotalDesc: prometheus.NewDesc(
			"node_md_blocks",
			"Total number of blocks on device.",
			[]string{"device"},
			nil,
		),
		blocksSyncedDesc: prometheus.NewDesc(
			"node_md_blocks_synced",
			"Number of blocks synced on device.",
			[]string{"device"},
			nil,
		),
		mdraidDisks: prometheus.NewDesc(
			"node_md_raid_disks",
			"Number of raid disks on device.",
			[]string{"device"},
			nil,
		),
		mdraidDegradedDisksDesc: prometheus.NewDesc(
			"node_md_degraded",
			"Number of degraded disks on device.",
			[]string{"device"},
			nil,
		),
	}
}

func (c *MdadmCollector) Collect(ch chan<- prometheus.Metric) {
	procFS, err := procfs.NewFS("/proc")
	if err != nil {
		return
	}

	mdStats, err := procFS.MDStat()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	for _, mdStat := range mdStats {
		stateVals := make(map[string]float64)
		stateVals[mdStat.ActivityState] = 1

		ch <- prometheus.MustNewConstMetric(
			c.disksTotalDesc,
			prometheus.GaugeValue,
			float64(mdStat.DisksTotal),
			mdStat.Name,
		)

		ch <- prometheus.MustNewConstMetric(
			c.disksDesc,
			prometheus.GaugeValue,
			float64(mdStat.DisksActive),
			mdStat.Name,
			"active",
		)
		ch <- prometheus.MustNewConstMetric(
			c.disksDesc,
			prometheus.GaugeValue,
			float64(mdStat.DisksFailed),
			mdStat.Name,
			"failed",
		)
		ch <- prometheus.MustNewConstMetric(
			c.disksDesc,
			prometheus.GaugeValue,
			float64(mdStat.DisksSpare),
			mdStat.Name,
			"spare",
		)
		ch <- prometheus.MustNewConstMetric(
			c.activeDesc,
			prometheus.GaugeValue,
			stateVals["active"],
			mdStat.Name,
		)

		ch <- prometheus.MustNewConstMetric(
			c.inActiveDesc,
			prometheus.GaugeValue,
			stateVals["inactive"],
			mdStat.Name,
		)

		ch <- prometheus.MustNewConstMetric(
			c.recoveringDesc,
			prometheus.GaugeValue,
			stateVals["recovering"],
			mdStat.Name,
		)

		ch <- prometheus.MustNewConstMetric(
			c.resyncDesc,
			prometheus.GaugeValue,
			stateVals["resyncing"],
			mdStat.Name,
		)

		ch <- prometheus.MustNewConstMetric(
			c.checkDesc,
			prometheus.GaugeValue,
			stateVals["checking"],
			mdStat.Name,
		)

		ch <- prometheus.MustNewConstMetric(
			c.blocksTotalDesc,
			prometheus.GaugeValue,
			float64(mdStat.BlocksTotal),
			mdStat.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			c.blocksSyncedDesc,
			prometheus.GaugeValue,
			float64(mdStat.BlocksSynced),
			mdStat.Name,
		)
	}

	sysFS, err := sysfs.NewFS("/sys")
	if err != nil {
		return
	}
	
	mdraids, err := sysFS.Mdraids()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	for _, mdraid := range mdraids {
		ch <- prometheus.MustNewConstMetric(
			c.mdraidDisks,
			prometheus.GaugeValue,
			float64(mdraid.Disks),
			mdraid.Device,
		)
		ch <- prometheus.MustNewConstMetric(
			c.mdraidDegradedDisksDesc,
			prometheus.GaugeValue,
			float64(mdraid.DegradedDisks),
			mdraid.Device,
		)
	}
}
// Part 2 commit for node_hardware_exporter/internal/metrics/mdadm.go

package metrics

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"
)

type VgExporter struct {
	lv_count            *prometheus.Desc
	max_lv              *prometheus.Desc
	max_pv              *prometheus.Desc
	pv_count            *prometheus.Desc
	snap_count          *prometheus.Desc
	vg_clustered        *prometheus.Desc
	vg_exported         *prometheus.Desc
	vg_extendable       *prometheus.Desc
	vg_extent_count     *prometheus.Desc
	vg_extent_size      *prometheus.Desc
	vg_free             *prometheus.Desc
	vg_free_count       *prometheus.Desc
	vg_mda_copies       *prometheus.Desc
	vg_mda_count        *prometheus.Desc
	vg_mda_free         *prometheus.Desc
	vg_mda_size         *prometheus.Desc
	vg_mda_used_count   *prometheus.Desc
	vg_missing_pv_count *prometheus.Desc
	vg_partial          *prometheus.Desc
	vg_seqno            *prometheus.Desc
	vg_shared           *prometheus.Desc
	vg_size             *prometheus.Desc
}

var vgnamespace = "vg"

func NewVgExporter() *VgExporter {
	lv_count := prometheus.NewDesc(
		prometheus.BuildFQName(vgnamespace, "", "vg_lv_count"),
		"Number of LVs", []string{"vg_uuid"}, nil,
	)

	max_lv := prometheus.NewDesc(
		prometheus.BuildFQName(vgnamespace, "", "vg_max_lv"),
		"Maximum number of LVs allowed in VG or 0 if unlimited", []string{"vg_uuid"}, nil,
	)

	max_pv := prometheus.NewDesc(
		prometheus.BuildFQName(vgnamespace, "", "vg_max_pv"),
		"Maximum number of PVs allowed in VG or 0 if unlimited", []string{"vg_uuid"}, nil,
	)

	pv_count := prometheus.NewDesc(
		prometheus.BuildFQName(vgnamespace, "", "vg_pv_count"),
		"Number of PVs in VG", []string{"vg_uuid"}, nil,
	)

	snap_count := prometheus.NewDesc(
		prometheus.BuildFQName(vgnamespace, "", "vg_snap_count"),
		"Number of snapshots", []string{"vg_uuid"}, nil,
	)
	vg_clustered := prometheus.NewDesc(
		prometheus.BuildFQName(vgnamespace, "", "vg_clustered"),
		"Set if VG is clustered", []string{"vg_uuid"}, nil,
	)
	vg_exported := prometheus.NewDesc(
		prometheus.BuildFQName(vgnamespace, "", "vg_exported"),
		"Set if VG is exported", []string{"vg_uuid"}, nil,
	)
	vg_extendable := prometheus.NewDesc(
		prometheus.BuildFQName(vgnamespace, "", "vg_extendable"),
		"Set if VG is extendable", []string{"vg_uuid"}, nil,
	)
	vg_extent_count := prometheus.NewDesc(
		prometheus.BuildFQName(vgnamespace, "", "vg_extent_count"),
		"Total number of Physical Extents", []string{"vg_uuid"}, nil,
	)

	vg_extent_size := prometheus.NewDesc(
		prometheus.BuildFQName(vgnamespace, "", "vg_extent_size_bytes"),
		"Size of Physical Extents", []string{"vg_uuid"}, nil,
	)

	vg_free := prometheus.NewDesc(
		prometheus.BuildFQName(vgnamespace, "", "vg_free_bytes"),
		"Total amount of free space in bytes", []string{"vg_uuid"}, nil,
	)

	vg_free_count := prometheus.NewDesc(
		prometheus.BuildFQName(vgnamespace, "", "vg_free_count"),
		"Total number of unallocated Physical Extents", []string{"vg_uuid"}, nil,
	)

	vg_mda_copies := prometheus.NewDesc(
		prometheus.BuildFQName(vgnamespace, "", "vg_mda_copies"),
		"Target number of in use metadata areas in the VG", []string{"vg_uuid"}, nil,
	)

	vg_mda_count := prometheus.NewDesc(
		prometheus.BuildFQName(vgnamespace, "", "vg_mda_count"),
		"Number of metadata areas", []string{"vg_uuid"}, nil,
	)

	vg_mda_free := prometheus.NewDesc(
		prometheus.BuildFQName(vgnamespace, "", "vg_mda_free_bytes"),
		"Free metadata area space for this VG", []string{"vg_uuid"}, nil,
	)

	vg_mda_size := prometheus.NewDesc(
		prometheus.BuildFQName(vgnamespace, "", "vg_mda_size_bytes"),
		"Size of smallest metadata area for this VG", []string{"vg_uuid"}, nil,
	)

	vg_mda_used_count := prometheus.NewDesc(
		prometheus.BuildFQName(vgnamespace, "", "vg_mda_used_count"),
		"Number of metadata areas in use on this VG", []string{"vg_uuid"}, nil,
	)
	vg_missing_pv_count := prometheus.NewDesc(
		prometheus.BuildFQName(vgnamespace, "", "vg_missing_pv_count"),
		"Number of PVs in VG which are missing", []string{"vg_uuid"}, nil,
	)
	vg_partial := prometheus.NewDesc(
		prometheus.BuildFQName(vgnamespace, "", "vg_partial"),
		"Set if VG is partial", []string{"vg_uuid"}, nil,
	)
	vg_seqno := prometheus.NewDesc(
		prometheus.BuildFQName(vgnamespace, "", "vg_seqno"),
		"Revision number of internal metadata", []string{"vg_uuid"}, nil,
	)
	vg_shared := prometheus.NewDesc(
		prometheus.BuildFQName(vgnamespace, "", "vg_shared"),
		"Set if VG is shared", []string{"vg_uuid"}, nil,
	)
	vg_size := prometheus.NewDesc(
		prometheus.BuildFQName(vgnamespace, "", "vg_size_bytes"),
		"Total size of VG in bytes", []string{"vg_uuid"}, nil,
	)

	return &VgExporter{
		lv_count:            lv_count,
		max_lv:              max_lv,
		max_pv:              max_pv,
		pv_count:            pv_count,
		snap_count:          snap_count,
		vg_clustered:        vg_clustered,
		vg_exported:         vg_exported,
		vg_extendable:       vg_extendable,
		vg_extent_count:     vg_extent_count,
		vg_extent_size:      vg_extent_size,
		vg_free:             vg_free,
		vg_free_count:       vg_free_count,
		vg_mda_copies:       vg_mda_copies,
		vg_mda_count:        vg_mda_count,
		vg_mda_free:         vg_mda_free,
		vg_mda_size:         vg_mda_size,
		vg_mda_used_count:   vg_mda_used_count,
		vg_missing_pv_count: vg_missing_pv_count,
		vg_partial:          vg_partial,
		vg_seqno:            vg_seqno,
		vg_shared:           vg_shared,
		vg_size:             vg_size,
	}
}

func (e *VgExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.lv_count
	ch <- e.max_lv
	ch <- e.max_pv
	ch <- e.pv_count
	ch <- e.snap_count
	ch <- e.vg_clustered
	ch <- e.vg_exported
	ch <- e.vg_extendable
	ch <- e.vg_extent_count
	ch <- e.vg_extent_size
	ch <- e.vg_free
	ch <- e.vg_free_count
	ch <- e.vg_mda_copies
	ch <- e.vg_mda_count
	ch <- e.vg_mda_free
	ch <- e.vg_mda_size
	ch <- e.vg_mda_used_count
	ch <- e.vg_missing_pv_count
	ch <- e.vg_partial
	ch <- e.vg_seqno
	ch <- e.vg_shared
	ch <- e.vg_size
}

func (e *VgExporter) Collect(ch chan<- prometheus.Metric) {
	// log.Println("run here Collect")

	e.VgCollect(ch)
}

func (e *VgExporter) VgCollect(ch chan<- prometheus.Metric) {
	report, err := GetLvmReport()
	if err != nil {
		log.Println("Error get JSON:", err)
	}
	vgs, err := GetVgInfo(report)
	if err != nil {
		log.Println("Error get PvInfo:", err)
	}
	for _, vg := range vgs {
		exportLvCount(ch, e, &vg)
		exportMaxLv(ch, e, &vg)
		exportMaxPv(ch, e, &vg)
		exportPvCount(ch, e, &vg)
		exportSnapCount(ch, e, &vg)
		exportVgClustered(ch, e, &vg)
		exportVgExported(ch, e, &vg)
		exportVgExtendable(ch, e, &vg)
		exportVgExtentCount(ch, e, &vg)
		exportVgExtentSize(ch, e, &vg)
		exportVgFree(ch, e, &vg)
		exportVgFreeCount(ch, e, &vg)
		exportVgMdaCopies(ch, e, &vg)
		exportVgMdaCount(ch, e, &vg)
		exportVgMdaFree(ch, e, &vg)
		exportVgMdaSize(ch, e, &vg)
		exportVgMdaUsedCount(ch, e, &vg)
		exportVgMissingPvCount(ch, e, &vg)
		exportVgPartial(ch, e, &vg)
		exportVgSeqNo(ch, e, &vg)
		exportVgShared(ch, e, &vg)
		exportVgSize(ch, e, &vg)
	}

}

func exportLvCount(ch chan<- prometheus.Metric, e *VgExporter, vg *VgInfo) {
	lv_count := parseString(vg.Lv_count)
	ch <- prometheus.MustNewConstMetric(
		e.lv_count, prometheus.GaugeValue, lv_count, vg.Vg_uuid)
}

func exportMaxLv(ch chan<- prometheus.Metric, e *VgExporter, vg *VgInfo) {
	max_lv := parseString(vg.Max_lv)
	ch <- prometheus.MustNewConstMetric(
		e.max_lv, prometheus.GaugeValue, max_lv, vg.Vg_uuid)
}

func exportMaxPv(ch chan<- prometheus.Metric, e *VgExporter, vg *VgInfo) {
	max_pv := parseString(vg.Max_pv)
	ch <- prometheus.MustNewConstMetric(
		e.max_pv, prometheus.GaugeValue, max_pv, vg.Vg_uuid)
}

func exportPvCount(ch chan<- prometheus.Metric, e *VgExporter, vg *VgInfo) {
	pv_count := parseString(vg.Pv_count)
	ch <- prometheus.MustNewConstMetric(
		e.pv_count, prometheus.GaugeValue, pv_count, vg.Vg_uuid)
}

func exportSnapCount(ch chan<- prometheus.Metric, e *VgExporter, vg *VgInfo) {
	snap_count := parseString(vg.Snap_count)
	ch <- prometheus.MustNewConstMetric(
		e.snap_count, prometheus.GaugeValue, snap_count, vg.Vg_uuid)
}

func exportVgClustered(ch chan<- prometheus.Metric, e *VgExporter, vg *VgInfo) {
	vg_clustered := 0.0
	switch vg.Vg_clustered {
	case "":
		vg_clustered = 0.0
	case "true":
		vg_clustered = 1.0
	default:
		vg_clustered = 0.0
	}
	ch <- prometheus.MustNewConstMetric(
		e.vg_clustered, prometheus.GaugeValue, vg_clustered, vg.Vg_uuid)
}

func exportVgExported(ch chan<- prometheus.Metric, e *VgExporter, vg *VgInfo) {
	vg_exported := 0.0
	switch vg.Vg_exported {
	case "":
		vg_exported = 0.0
	case "true":
		vg_exported = 1.0
	default:
		vg_exported = 0.0
	}
	ch <- prometheus.MustNewConstMetric(
		e.vg_exported, prometheus.GaugeValue, vg_exported, vg.Vg_uuid)
}

func exportVgExtendable(ch chan<- prometheus.Metric, e *VgExporter, vg *VgInfo) {
	vg_extendable := 1.0
	switch vg.Vg_extendable {
	case "extendable":
		vg_extendable = 1.0
	case "":
		vg_extendable = 0.0
	default:
		vg_extendable = 1.0
	}
	ch <- prometheus.MustNewConstMetric(
		e.vg_extendable, prometheus.GaugeValue, vg_extendable, vg.Vg_uuid)
}

func exportVgExtentCount(ch chan<- prometheus.Metric, e *VgExporter, vg *VgInfo) {
	vg_extent_count := parseString(vg.Vg_extent_count)
	ch <- prometheus.MustNewConstMetric(
		e.vg_extent_count, prometheus.GaugeValue, vg_extent_count, vg.Vg_uuid)
}

func exportVgExtentSize(ch chan<- prometheus.Metric, e *VgExporter, vg *VgInfo) {
	vg_extent_size := parseSize(vg.Vg_extent_size)
	ch <- prometheus.MustNewConstMetric(
		e.vg_extent_size, prometheus.GaugeValue, vg_extent_size, vg.Vg_uuid)
}

func exportVgFree(ch chan<- prometheus.Metric, e *VgExporter, vg *VgInfo) {
	vg_free := parseSize(vg.Vg_free)
	ch <- prometheus.MustNewConstMetric(
		e.vg_free, prometheus.GaugeValue, vg_free, vg.Vg_uuid)
}

func exportVgFreeCount(ch chan<- prometheus.Metric, e *VgExporter, vg *VgInfo) {
	vg_free_count := parseString(vg.Vg_free_count)
	ch <- prometheus.MustNewConstMetric(
		e.vg_free_count, prometheus.GaugeValue, vg_free_count, vg.Vg_uuid)
}

func exportVgMdaCopies(ch chan<- prometheus.Metric, e *VgExporter, vg *VgInfo) {
	vg_mda_copies := 1.0
	switch vg.Vg_mda_copies {
	case "unmanaged":
		vg_mda_copies = 0.0
	case "managed":
		vg_mda_copies = 1.0
	default:
		vg_mda_copies = 0.0
	}
	ch <- prometheus.MustNewConstMetric(
		e.vg_mda_copies, prometheus.GaugeValue, vg_mda_copies, vg.Vg_uuid)
}

func exportVgMdaCount(ch chan<- prometheus.Metric, e *VgExporter, vg *VgInfo) {
	vg_mda_count := parseString(vg.Vg_mda_count)
	ch <- prometheus.MustNewConstMetric(
		e.vg_mda_count, prometheus.GaugeValue, vg_mda_count, vg.Vg_uuid)
}

func exportVgMdaFree(ch chan<- prometheus.Metric, e *VgExporter, vg *VgInfo) {
	vg_mda_free := parseSize(vg.Vg_mda_free)
	ch <- prometheus.MustNewConstMetric(
		e.vg_mda_free, prometheus.GaugeValue, vg_mda_free, vg.Vg_uuid)
}

func exportVgMdaSize(ch chan<- prometheus.Metric, e *VgExporter, vg *VgInfo) {
	vg_mda_size := parseSize(vg.Vg_mda_size)
	ch <- prometheus.MustNewConstMetric(
		e.vg_mda_size, prometheus.GaugeValue, vg_mda_size, vg.Vg_uuid)
}

func exportVgMdaUsedCount(ch chan<- prometheus.Metric, e *VgExporter, vg *VgInfo) {
	vg_mda_used_count := parseString(vg.Vg_mda_used_count)
	ch <- prometheus.MustNewConstMetric(
		e.vg_mda_used_count, prometheus.GaugeValue, vg_mda_used_count, vg.Vg_uuid)
}

func exportVgMissingPvCount(ch chan<- prometheus.Metric, e *VgExporter, vg *VgInfo) {
	vg_missing_pv_count := parseString(vg.Vg_missing_pv_count)
	ch <- prometheus.MustNewConstMetric(
		e.vg_missing_pv_count, prometheus.GaugeValue, vg_missing_pv_count, vg.Vg_uuid)
}

func exportVgPartial(ch chan<- prometheus.Metric, e *VgExporter, vg *VgInfo) {
	vg_partial := 0.0
	switch vg.Vg_partial {
	case "":
		vg_partial = 0.0
	case "true":
		vg_partial = 1.0
	default:
		vg_partial = 0.0
	}
	ch <- prometheus.MustNewConstMetric(
		e.vg_partial, prometheus.GaugeValue, vg_partial, vg.Vg_uuid)
}

func exportVgSeqNo(ch chan<- prometheus.Metric, e *VgExporter, vg *VgInfo) {
	vg_seqno := parseString(vg.Vg_seqno)
	ch <- prometheus.MustNewConstMetric(
		e.vg_seqno, prometheus.GaugeValue, vg_seqno, vg.Vg_uuid)
}

func exportVgShared(ch chan<- prometheus.Metric, e *VgExporter, vg *VgInfo) {
	vg_shared := 0.0
	switch vg.Vg_shared {
	case "":
		vg_shared = 0.0
	case "true":
		vg_shared = 1.0
	default:
		vg_shared = 0.0
	}
	ch <- prometheus.MustNewConstMetric(
		e.vg_shared, prometheus.GaugeValue, vg_shared, vg.Vg_uuid)
}

func exportVgSize(ch chan<- prometheus.Metric, e *VgExporter, vg *VgInfo) {
	vg_size := parseSize(vg.Vg_size)
	ch <- prometheus.MustNewConstMetric(
		e.vg_size, prometheus.GaugeValue, vg_size, vg.Vg_uuid)
}

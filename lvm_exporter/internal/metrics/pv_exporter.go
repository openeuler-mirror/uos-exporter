package metrics

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	pvnamespace = "pv"
)

type PvExporter struct {
	dev_size          *prometheus.Desc
	pe_start          *prometheus.Desc
	pv_allocatable    *prometheus.Desc
	pv_ba_size        *prometheus.Desc
	pv_ba_start       *prometheus.Desc
	pv_duplicate      *prometheus.Desc
	pv_exported       *prometheus.Desc
	pv_ext_vsn        *prometheus.Desc
	pv_free           *prometheus.Desc
	pv_in_use         *prometheus.Desc
	pv_major          *prometheus.Desc
	pv_minor          *prometheus.Desc
	pv_mda_count      *prometheus.Desc
	pv_mda_free       *prometheus.Desc
	pv_mda_size       *prometheus.Desc
	pv_mda_used_count *prometheus.Desc
	pv_missing        *prometheus.Desc
	pv_pe_alloc_count *prometheus.Desc
	pv_pe_count       *prometheus.Desc
	pv_size           *prometheus.Desc
	pv_used           *prometheus.Desc
}

func NewPvExporter() *PvExporter {
	// return &LvmExporter{NewMetrics(fqname, help, labels)}
	dev_size := prometheus.NewDesc(
		prometheus.BuildFQName(pvnamespace, "", "pv_dev_size_bytes"),
		"Size of underlying device", []string{"pv_uuid"}, nil,
	)

	pe_start := prometheus.NewDesc(
		prometheus.BuildFQName(pvnamespace, "", "pv_pe_start"),
		"Offset to the start of data on the underlying device", []string{"pv_uuid"}, nil,
	)

	pv_allocatable := prometheus.NewDesc(
		prometheus.BuildFQName(pvnamespace, "", "pv_allocatable"),
		"Set if this device can be used for allocation", []string{"pv_uuid"}, nil,
	)

	pv_ba_size := prometheus.NewDesc(
		prometheus.BuildFQName(pvnamespace, "", "pv_ba_size_bytes"),
		"Size of PV Bootloader Area", []string{"pv_uuid"}, nil,
	)

	pv_ba_start := prometheus.NewDesc(
		prometheus.BuildFQName(pvnamespace, "", "pv_ba_start"),
		"Offset to the start of PV Bootloader Area on the underlying device", []string{"pv_uuid"}, nil,
	)

	pv_duplicate := prometheus.NewDesc(
		prometheus.BuildFQName(pvnamespace, "", "pv_duplicate"),
		"Set if PV is an unchosen duplicate", []string{"pv_uuid"}, nil,
	)

	pv_exported := prometheus.NewDesc(
		prometheus.BuildFQName(pvnamespace, "", "pv_exported"),
		"Set if this device is exported", []string{"pv_uuid"}, nil,
	)

	pv_ext_vsn := prometheus.NewDesc(
		prometheus.BuildFQName(pvnamespace, "", "pv_ext_vsn"),
		"PV header extension version", []string{"pv_uuid"}, nil,
	)

	pv_free := prometheus.NewDesc(
		prometheus.BuildFQName(pvnamespace, "", "pv_free"),
		"Total amount of unallocated space", []string{"pv_uuid"}, nil,
	)

	pv_in_use := prometheus.NewDesc(
		prometheus.BuildFQName(pvnamespace, "", "pv_in_use"),
		"Set if PV is used", []string{"pv_uuid"}, nil,
	)

	pv_major := prometheus.NewDesc(
		prometheus.BuildFQName(pvnamespace, "", "pv_major"),
		"Device major number", []string{"pv_uuid"}, nil,
	)
	pv_minor := prometheus.NewDesc(
		prometheus.BuildFQName(pvnamespace, "", "pv_minor"),
		"Device minor number", []string{"pv_uuid"}, nil,
	)
	pv_mda_count := prometheus.NewDesc(
		prometheus.BuildFQName(pvnamespace, "", "pv_mda_count"),
		"Number of metadata areas", []string{"pv_uuid"}, nil,
	)
	pv_mda_free := prometheus.NewDesc(
		prometheus.BuildFQName(pvnamespace, "", "pv_mda_free"),
		"Free metadata area space", []string{"pv_uuid"}, nil,
	)
	pv_mda_size := prometheus.NewDesc(
		prometheus.BuildFQName(pvnamespace, "", "pv_mda_size_bytes"),
		"Size of smallest metadata area", []string{"pv_uuid"}, nil,
	)
	pv_mda_used_count := prometheus.NewDesc(
		prometheus.BuildFQName(pvnamespace, "", "pv_mda_used_count"),
		"Number of metadata areas in use", []string{"pv_uuid"}, nil,
	)

	pv_missing := prometheus.NewDesc(
		prometheus.BuildFQName(pvnamespace, "", "pv_missing"),
		"Set if this device is missing in system", []string{"pv_uuid"}, nil,
	)
	pv_pe_alloc_count := prometheus.NewDesc(
		prometheus.BuildFQName(pvnamespace, "", "pv_pe_alloc_count"),
		"Total number of allocated Physical Extents", []string{"pv_uuid"}, nil,
	)
	pv_pe_count := prometheus.NewDesc(
		prometheus.BuildFQName(pvnamespace, "", "pv_pe_count"),
		"Total number of Physical Extents", []string{"pv_uuid"}, nil,
	)
	pv_size := prometheus.NewDesc(
		prometheus.BuildFQName(pvnamespace, "", "pv_size_bytes"),
		"Size of PV", []string{"pv_uuid"}, nil,
	)
	pv_used := prometheus.NewDesc(
		prometheus.BuildFQName(pvnamespace, "", "pv_used"),
		"Total amount of allocated space", []string{"pv_uuid"}, nil,
	)

	return &PvExporter{
		dev_size:          dev_size,
		pe_start:          pe_start,
		pv_allocatable:    pv_allocatable,
		pv_ba_size:        pv_ba_size,
		pv_ba_start:       pv_ba_start,
		pv_duplicate:      pv_duplicate,
		pv_exported:       pv_exported,
		pv_ext_vsn:        pv_ext_vsn,
		pv_free:           pv_free,
		pv_in_use:         pv_in_use,
		pv_major:          pv_major,
		pv_minor:          pv_minor,
		pv_mda_count:      pv_mda_count,
		pv_mda_free:       pv_mda_free,
		pv_mda_size:       pv_mda_size,
		pv_mda_used_count: pv_mda_used_count,
		pv_missing:        pv_missing,
		pv_pe_alloc_count: pv_pe_alloc_count,
		pv_pe_count:       pv_pe_count,
		pv_size:           pv_size,
		pv_used:           pv_used,
	}
}

func (e *PvExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.dev_size
	ch <- e.pe_start
	ch <- e.pv_allocatable
	ch <- e.pv_ba_size
	ch <- e.pv_ba_start
	ch <- e.pv_duplicate
	ch <- e.pv_exported
	ch <- e.pv_ext_vsn
	ch <- e.pv_free
	ch <- e.pv_in_use
	ch <- e.pv_major
	ch <- e.pv_minor
	ch <- e.pv_mda_count
	ch <- e.pv_mda_free
	ch <- e.pv_mda_size
	ch <- e.pv_mda_used_count
	ch <- e.pv_missing
	ch <- e.pv_pe_alloc_count
	ch <- e.pv_pe_count
	ch <- e.pv_size
	ch <- e.pv_used
}

func (e *PvExporter) Collect(ch chan<- prometheus.Metric) {
	// log.Println("run here Collect")

	e.PvCollect(ch)
}

func (e *PvExporter) PvCollect(ch chan<- prometheus.Metric) {
	report, err := GetLvmReport()
	if err != nil {
		log.Println("Error get JSON:", err)
	}
	pvs, err := GetPvInfo(report)
	if err != nil {
		log.Println("Error get PvInfo:", err)
	}
	for _, pv := range pvs {
		exportDevSize(ch, e, &pv)
		exportPeStart(ch, e, &pv)
		exportPvAllocatable(ch, e, &pv)
		exportPvBaSize(ch, e, &pv)
		exportPvBaStart(ch, e, &pv)
		exportPvDuplicate(ch, e, &pv)
		exportPvExported(ch, e, &pv)
		exportPvExtVsn(ch, e, &pv)
		exportPvFree(ch, e, &pv)
		exportPvInUse(ch, e, &pv)
		exportPvMajor(ch, e, &pv)
		exportPvMinor(ch, e, &pv)
		exportPvMdaCount(ch, e, &pv)
		exportPvMdaFree(ch, e, &pv)
		exportPvMdaSize(ch, e, &pv)
		exportPvMdaUsedCount(ch, e, &pv)
		exportPvMissing(ch, e, &pv)
		exportPvPeAllocCount(ch, e, &pv)
		exportPvPeCount(ch, e, &pv)
		exportPvUsed(ch, e, &pv)
		exportPvSize(ch, e, &pv)
	}

}

func exportDevSize(ch chan<- prometheus.Metric, e *PvExporter, pv *PvInfo) {
	dev_size := parseSize(pv.Dev_size)
	ch <- prometheus.MustNewConstMetric(
		e.dev_size, prometheus.GaugeValue, dev_size, pv.Pv_uuid)
}

func exportPeStart(ch chan<- prometheus.Metric, e *PvExporter, pv *PvInfo) {
	pe_start := parseTime(pv.Pe_start)
	ch <- prometheus.MustNewConstMetric(
		e.pe_start, prometheus.GaugeValue, pe_start, pv.Pv_uuid)
}

func exportPvAllocatable(ch chan<- prometheus.Metric, e *PvExporter, pv *PvInfo) {
	pv_allocatable := 0.0
	switch pv.Pv_allocatable {
	case "allocatable":
		pv_allocatable = 1.0
	case "not allocatable":
		pv_allocatable = 0.0
	default:
		pv_allocatable = 1.0
	}
	ch <- prometheus.MustNewConstMetric(
		e.pv_allocatable, prometheus.GaugeValue, pv_allocatable, pv.Pv_uuid)
}

func exportPvBaSize(ch chan<- prometheus.Metric, e *PvExporter, pv *PvInfo) {
	pv_ba_size := parseString(pv.Pv_ba_size)
	ch <- prometheus.MustNewConstMetric(
		e.pv_ba_size, prometheus.GaugeValue, pv_ba_size, pv.Pv_uuid)
}

func exportPvBaStart(ch chan<- prometheus.Metric, e *PvExporter, pv *PvInfo) {
	pv_ba_start := parseString(pv.Pv_ba_start)
	ch <- prometheus.MustNewConstMetric(
		e.pv_ba_start, prometheus.GaugeValue, pv_ba_start, pv.Pv_uuid)
}

func exportPvDuplicate(ch chan<- prometheus.Metric, e *PvExporter, pv *PvInfo) {
	pv_duplicate := 0.0
	switch pv.Pv_duplicate {
	case "":
		pv_duplicate = 0.0
	case "no":
		pv_duplicate = 1.0
	default:
		pv_duplicate = 0.0
	}
	ch <- prometheus.MustNewConstMetric(
		e.pv_duplicate, prometheus.GaugeValue, pv_duplicate, pv.Pv_uuid)
}

func exportPvExported(ch chan<- prometheus.Metric, e *PvExporter, pv *PvInfo) {
	pv_exported := 0.0
	switch pv.Pv_exported {
	case "":
		pv_exported = 0.0
	case "no":
		pv_exported = 1.0
	default:
		pv_exported = 0.0
	}
	ch <- prometheus.MustNewConstMetric(
		e.pv_exported, prometheus.GaugeValue, pv_exported, pv.Pv_uuid)
}

func exportPvExtVsn(ch chan<- prometheus.Metric, e *PvExporter, pv *PvInfo) {
	pv_ext_vsn := parseString(pv.Pv_ext_vsn)
	ch <- prometheus.MustNewConstMetric(
		e.pv_ext_vsn, prometheus.GaugeValue, pv_ext_vsn, pv.Pv_uuid)
}

func exportPvFree(ch chan<- prometheus.Metric, e *PvExporter, pv *PvInfo) {
	pv_free := parseSize(pv.Pv_free)
	ch <- prometheus.MustNewConstMetric(
		e.pv_free, prometheus.GaugeValue, pv_free, pv.Pv_uuid)
}

func exportPvInUse(ch chan<- prometheus.Metric, e *PvExporter, pv *PvInfo) {
	pv_in_use := 1.0
	switch pv.Pv_in_use {
	case "used":
		pv_in_use = 1.0
	case "no":
		pv_in_use = 0.0
	default:
		pv_in_use = 0.0
	}
	ch <- prometheus.MustNewConstMetric(
		e.pv_in_use, prometheus.GaugeValue, pv_in_use, pv.Pv_uuid)
}

func exportPvMajor(ch chan<- prometheus.Metric, e *PvExporter, pv *PvInfo) {
	pv_major := parseString(pv.Pv_major)
	ch <- prometheus.MustNewConstMetric(
		e.pv_major, prometheus.GaugeValue, pv_major, pv.Pv_uuid)
}

func exportPvMinor(ch chan<- prometheus.Metric, e *PvExporter, pv *PvInfo) {
	pv_minor := parseString(pv.Pv_minor)
	ch <- prometheus.MustNewConstMetric(
		e.pv_minor, prometheus.GaugeValue, pv_minor, pv.Pv_uuid)
}

func exportPvMdaCount(ch chan<- prometheus.Metric, e *PvExporter, pv *PvInfo) {
	pv_mda_count := parseString(pv.Pv_mda_count)
	ch <- prometheus.MustNewConstMetric(
		e.pv_mda_count, prometheus.GaugeValue, pv_mda_count, pv.Pv_uuid)
}

func exportPvMdaFree(ch chan<- prometheus.Metric, e *PvExporter, pv *PvInfo) {
	pv_mda_free := parseSize(pv.Pv_mda_free)
	ch <- prometheus.MustNewConstMetric(
		e.pv_mda_free, prometheus.GaugeValue, pv_mda_free, pv.Pv_uuid)
}

func exportPvMdaSize(ch chan<- prometheus.Metric, e *PvExporter, pv *PvInfo) {
	pv_mda_size := parseSize(pv.Pv_mda_size)
	ch <- prometheus.MustNewConstMetric(
		e.pv_mda_size, prometheus.GaugeValue, pv_mda_size, pv.Pv_uuid)
}

func exportPvMdaUsedCount(ch chan<- prometheus.Metric, e *PvExporter, pv *PvInfo) {
	pv_mda_used_count := parseString(pv.Pv_mda_used_count)
	ch <- prometheus.MustNewConstMetric(
		e.pv_mda_used_count, prometheus.GaugeValue, pv_mda_used_count, pv.Pv_uuid)
}

func exportPvMissing(ch chan<- prometheus.Metric, e *PvExporter, pv *PvInfo) {
	pv_missing := 0.0
	switch pv.Pv_missing {
	case "yes":
		pv_missing = 1.0
	case "":
		pv_missing = 0.0
	default:
		pv_missing = 0.0
	}
	ch <- prometheus.MustNewConstMetric(
		e.pv_missing, prometheus.GaugeValue, pv_missing, pv.Pv_uuid)
}

func exportPvPeAllocCount(ch chan<- prometheus.Metric, e *PvExporter, pv *PvInfo) {
	pv_pe_alloc_count := parseString(pv.Pv_pe_alloc_count)
	ch <- prometheus.MustNewConstMetric(
		e.pv_pe_alloc_count, prometheus.GaugeValue, pv_pe_alloc_count, pv.Pv_uuid)
}

func exportPvPeCount(ch chan<- prometheus.Metric, e *PvExporter, pv *PvInfo) {
	pv_pe_count := parseString(pv.Pv_pe_count)
	ch <- prometheus.MustNewConstMetric(
		e.pv_pe_count, prometheus.GaugeValue, pv_pe_count, pv.Pv_uuid)
}

func exportPvUsed(ch chan<- prometheus.Metric, e *PvExporter, pv *PvInfo) {
	pv_used := parseSize(pv.Pv_used)
	ch <- prometheus.MustNewConstMetric(
		e.pv_used, prometheus.GaugeValue, pv_used, pv.Pv_uuid)
}

func exportPvSize(ch chan<- prometheus.Metric, e *PvExporter, pv *PvInfo) {
	pv_size := parseSize(pv.Pv_size)
	ch <- prometheus.MustNewConstMetric(
		e.pv_size, prometheus.GaugeValue, pv_size, pv.Pv_uuid)
}

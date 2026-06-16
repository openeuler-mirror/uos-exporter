package metrics

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"
)

type PvSegExporter struct {
	pvseg_start *prometheus.Desc
	pvseg_size  *prometheus.Desc
}

var pvsegnamespace = "pvseg"

func NewPvSegExporter() *PvSegExporter {
	pvseg_size := prometheus.NewDesc(
		prometheus.BuildFQName(pvsegnamespace, "", "pvseg_size_bytes"),
		"Size of PV Segment", []string{"pv_uuid", "lv_uuid"}, nil,
	)

	pvseg_start := prometheus.NewDesc(
		prometheus.BuildFQName(pvsegnamespace, "", "pvseg_start"),
		"Offset to the start of PV Segment on the underlying device", []string{"pv_uuid", "lv_uuid"}, nil,
	)

	return &PvSegExporter{
		pvseg_start: pvseg_start,
		pvseg_size:  pvseg_size,
	}
}

func (e *PvSegExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.pvseg_start
	ch <- e.pvseg_size
}

func (e *PvSegExporter) Collect(ch chan<- prometheus.Metric) {
	// log.Println("run here Collect")

	e.PvSegCollect(ch)
}

func (e *PvSegExporter) PvSegCollect(ch chan<- prometheus.Metric) {
	report, err := GetLvmReport()
	if err != nil {
		log.Println("Error get JSON:", err)
	}
	pvsegs, err := GetPvSegInfo(report)
	if err != nil {
		log.Println("Error get PvInfo:", err)
	}
	for _, pvseg := range pvsegs {
		exportPvSegSize(ch, e, &pvseg)
		exportPvSegStart(ch, e, &pvseg)
	}

}

func exportPvSegSize(ch chan<- prometheus.Metric, e *PvSegExporter, pvseg *PvSegInfo) {
	pvseg_size := parseSize(pvseg.Pvseg_size)
	ch <- prometheus.MustNewConstMetric(
		e.pvseg_size, prometheus.GaugeValue, pvseg_size, pvseg.Pv_uuid, pvseg.Lv_uuid)
}

func exportPvSegStart(ch chan<- prometheus.Metric, e *PvSegExporter, pvseg *PvSegInfo) {
	pvseg_start := parseString(pvseg.Pvseg_start)
	ch <- prometheus.MustNewConstMetric(
		e.pvseg_start, prometheus.GaugeValue, pvseg_start, pvseg.Pv_uuid, pvseg.Lv_uuid)
}

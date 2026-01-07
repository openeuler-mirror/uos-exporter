package metrics

import (
	"lvm_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	exporter.Register(
		NewLvmExporter())
}

type LvmExporter struct {
	pvexporter    *PvExporter
	vgexporter    *VgExporter
	lvexporter    *LvExporter
	pvsegexporter *PvSegExporter
}

func NewLvmExporter() *LvmExporter {
	return &LvmExporter{
		pvexporter:    NewPvExporter(),
		vgexporter:    NewVgExporter(),
		lvexporter:    NewLvExporter(),
		pvsegexporter: NewPvSegExporter(),
	}
}

func (e *LvmExporter) Describe(ch chan<- *prometheus.Desc) {
	e.pvexporter.Describe(ch)
	e.vgexporter.Describe(ch)
	e.lvexporter.Describe(ch)
	e.pvsegexporter.Describe(ch)
}

func (e *LvmExporter) Collect(ch chan<- prometheus.Metric) {
	e.pvexporter.Collect(ch)
	e.vgexporter.Collect(ch)
	e.lvexporter.Collect(ch)
	e.pvsegexporter.Collect(ch)
}

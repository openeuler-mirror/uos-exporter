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


// TODO: implement functions
